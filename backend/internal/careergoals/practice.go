package careergoals

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"studyapp/backend/internal/careergoals/adaptive"
)

// GetTodayPractice returns today's daily practice set for the active goal,
// generating it on first request of the day (idempotent per user/goal/date).
//
// GET /goals/my/practice/today
func (h *Handler) GetTodayPractice(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	studentGoalID, subjects, err := h.activeGoal(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	setID, err := h.ensureTodaySet(r.Context(), userID, studentGoalID, subjects)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	var (
		status     string
		score      *float64
		topicFocus []byte
		setDate    time.Time
	)
	err = h.db.QueryRow(r.Context(), `
		SELECT status, score, COALESCE(topic_focus, '[]'::jsonb), set_date
		FROM daily_practice_sets WHERE id = $1
	`, setID).Scan(&status, &score, &topicFocus, &setDate)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	questions, err := h.loadSetQuestions(r.Context(), setID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.json(w, http.StatusOK, TodayPractice{
		SetID:      setID,
		Date:       setDate.UTC().Format(dateLayout),
		Status:     status,
		Score:      score,
		TopicFocus: decodeStringArray(topicFocus),
		Questions:  questions,
	})
}

// SubmitPractice scores a daily set and adaptively updates skill gaps.
//
// POST /goals/my/practice/{setId}/submit
func (h *Handler) SubmitPractice(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	setID, err := strconv.ParseInt(chi.URLParam(r, "setId"), 10, 64)
	if err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}

	var req SubmitPracticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var (
		ownerID       int64
		studentGoalID int64
		status        string
		score         *float64
	)
	err = tx.QueryRow(r.Context(), `
		SELECT user_id, student_goal_id, status, score FROM daily_practice_sets WHERE id = $1 FOR UPDATE
	`, setID).Scan(&ownerID, &studentGoalID, &status, &score)
	if errors.Is(err, pgx.ErrNoRows) {
		h.handleServiceError(w, r, ErrSetNotFound)
		return
	}
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	if ownerID != userID {
		h.handleServiceError(w, r, ErrSetForbidden)
		return
	}

	// Idempotent: already submitted -> return stored score.
	if status == "completed" {
		if err := tx.Commit(r.Context()); err != nil {
			h.handleServiceError(w, r, err)
			return
		}
		h.json(w, http.StatusOK, SubmitPracticeResult{
			SetID:        setID,
			Score:        derefFloat(score),
			SkillUpdates: []SkillDelta{},
		})
		return
	}

	// Answer key for the set: correct option + topic/subject per question.
	rows, err := tx.Query(r.Context(), `
		SELECT psq.question_id,
		       (SELECT o.id FROM question_options o WHERE o.question_id = psq.question_id AND o.is_correct = true LIMIT 1),
		       COALESCE(c.title, ''), COALESCE(b.subject, '')
		FROM practice_set_questions psq
		JOIN questions q ON q.id = psq.question_id
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE psq.set_type = 'daily' AND psq.set_id = $1
		ORDER BY psq.display_order ASC
	`, setID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	type qmeta struct {
		correct *int64
		topic   string
		subject string
	}
	meta := make(map[int64]qmeta)
	var order []int64
	for rows.Next() {
		var qid int64
		var m qmeta
		if err := rows.Scan(&qid, &m.correct, &m.topic, &m.subject); err != nil {
			rows.Close()
			h.handleServiceError(w, r, err)
			return
		}
		meta[qid] = m
		order = append(order, qid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	selected := make(map[int64]*int64, len(req.Answers))
	for _, a := range req.Answers {
		selected[a.QuestionID] = a.SelectedOptionID
	}

	// Per-topic tallies keyed by (subject, topic).
	type tally struct {
		subject string
		topic   string
		correct int
		total   int
	}
	topicTally := make(map[string]*tally)

	correct, wrong, skipped := 0, 0, 0
	for _, qid := range order {
		m := meta[qid]
		key := skillGapKey(m.subject, m.topic)
		t := topicTally[key]
		if t == nil {
			t = &tally{subject: m.subject, topic: m.topic}
			topicTally[key] = t
		}
		t.total++

		sel, provided := selected[qid]
		switch {
		case !provided || sel == nil:
			skipped++
		case m.correct != nil && *sel == *m.correct:
			correct++
			t.correct++
		default:
			wrong++
		}
	}

	total := len(order)
	overall := scorePct(correct, total)

	if _, err := tx.Exec(r.Context(), `
		UPDATE daily_practice_sets SET status = 'completed', score = $2 WHERE id = $1
	`, setID, overall); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	// Build per-topic scores and update skill gaps atomically in the same tx.
	topicScores := make(map[string]float64, len(topicTally))
	skillUpdates := make([]SkillDelta, 0, len(topicTally))
	for key, t := range topicTally {
		if t.topic == "" {
			continue
		}
		pct := scorePct(t.correct, t.total)
		topicScores[key] = pct
		skillUpdates = append(skillUpdates, SkillDelta{
			Topic:     t.topic,
			Subject:   t.subject,
			Direction: directionForScore(pct),
			Score:     pct,
		})
	}

	if err := UpdateSkillGaps(r.Context(), tx, strconv.FormatInt(userID, 10), topicScores); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	// Best-effort: keep the weekly goal set / weekly score history up to date.
	scorer := adaptive.NewPerformanceScorer(h.db, h.logger)
	if err := scorer.UpdateWeeklyGoal(r.Context(), strconv.FormatInt(userID, 10), strconv.FormatInt(studentGoalID, 10)); err != nil {
		h.logger.Warn("weekly goal update failed", "error", err, "userID", userID, "studentGoalID", studentGoalID)
	}

	h.json(w, http.StatusOK, SubmitPracticeResult{
		SetID:          setID,
		Score:          overall,
		CorrectCount:   correct,
		WrongCount:     wrong,
		SkippedCount:   skipped,
		TotalQuestions: total,
		SkillUpdates:   skillUpdates,
	})
}

// GetPracticeHistory returns the last 30 days of daily sets for the trend chart.
//
// GET /goals/my/practice/history
func (h *Handler) GetPracticeHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	studentGoalID, _, err := h.activeGoal(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT id, set_date, status, score
		FROM daily_practice_sets
		WHERE student_goal_id = $1 AND set_date >= (CURRENT_DATE - INTERVAL '30 days')
		ORDER BY set_date ASC
	`, studentGoalID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	defer rows.Close()

	history := PracticeHistory{Items: []PracticeHistoryItem{}}
	for rows.Next() {
		var (
			item PracticeHistoryItem
			date time.Time
		)
		if err := rows.Scan(&item.SetID, &date, &item.Status, &item.Score); err != nil {
			h.handleServiceError(w, r, err)
			return
		}
		item.Date = date.UTC().Format(dateLayout)
		history.Items = append(history.Items, item)
	}
	if err := rows.Err(); err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, history)
}

// --- helpers ---

// activeGoal returns the user's active student_goal id and its subject areas.
func (h *Handler) activeGoal(ctx context.Context, userID int64) (int64, []string, error) {
	var (
		studentGoalID int64
		subjects      []byte
	)
	err := h.db.QueryRow(ctx, `
		SELECT sg.id, COALESCE(cg.subject_areas, '[]'::jsonb)
		FROM student_goals sg
		JOIN career_goals cg ON cg.id = sg.career_goal_id
		WHERE sg.user_id = $1 AND sg.status = 'active'
		ORDER BY sg.created_at DESC
		LIMIT 1
	`, userID).Scan(&studentGoalID, &subjects)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, ErrNoActiveGoal
	}
	if err != nil {
		return 0, nil, err
	}
	return studentGoalID, decodeStringArray(subjects), nil
}

// ensureTodaySet returns today's set id, generating it if it does not exist.
func (h *Handler) ensureTodaySet(ctx context.Context, userID, studentGoalID int64, subjects []string) (int64, error) {
	today := time.Now().UTC().Format(dateLayout)

	var setID int64
	err := h.db.QueryRow(ctx, `
		SELECT id FROM daily_practice_sets
		WHERE student_goal_id = $1 AND set_date = $2::date
	`, studentGoalID, today).Scan(&setID)
	if err == nil {
		return setID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	return h.generateDailySet(ctx, userID, studentGoalID, subjects, today)
}

// generateDailySet picks a weak-biased question mix (via the adaptive engine)
// and persists the set.
func (h *Handler) generateDailySet(ctx context.Context, userID, studentGoalID int64, subjects []string, today string) (int64, error) {
	selector := adaptive.NewQuestionSelector(h.db, h.logger)

	recentlySeenIDs, err := selector.GetRecentlySeenIDs(ctx, strconv.FormatInt(userID, 10), 7)
	if err != nil {
		return 0, err
	}

	picked, err := selector.SelectForDaily(ctx, adaptive.SelectionConfig{
		UserID:         strconv.FormatInt(userID, 10),
		GoalID:         strconv.FormatInt(studentGoalID, 10),
		TotalQuestions: adaptive.DefaultDailyQuestions,
		WeakTopicRatio: adaptive.DefaultWeakTopicRatio,
		ExcludeIDs:     recentlySeenIDs,
		Subjects:       subjects,
	})
	if err != nil {
		return 0, err
	}

	// Topic focus = selected questions whose topic is one of the user's weak topics.
	weakTopics, err := h.weakTopics(ctx, userID)
	if err != nil {
		return 0, err
	}
	weakSet := make(map[string]struct{}, len(weakTopics))
	for _, t := range weakTopics {
		weakSet[t] = struct{}{}
	}
	focusSet := map[string]struct{}{}
	for _, q := range picked {
		if q.Topic == "" {
			continue
		}
		if _, ok := weakSet[q.Topic]; ok {
			focusSet[q.Topic] = struct{}{}
		}
	}
	topicFocus := make([]string, 0, len(focusSet))
	for t := range focusSet {
		topicFocus = append(topicFocus, t)
	}
	focusJSON, _ := json.Marshal(topicFocus)

	tx, err := h.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var setID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO daily_practice_sets (user_id, student_goal_id, set_date, status, topic_focus)
		VALUES ($1, $2, $3::date, 'pending', $4)
		ON CONFLICT (user_id, student_goal_id, set_date) DO NOTHING
		RETURNING id
	`, userID, studentGoalID, today, focusJSON).Scan(&setID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Lost a race: another request created today's set. Return that one.
		_ = tx.Rollback(ctx)
		var existing int64
		if selErr := h.db.QueryRow(ctx, `
			SELECT id FROM daily_practice_sets
			WHERE student_goal_id = $1 AND set_date = $2::date
		`, studentGoalID, today).Scan(&existing); selErr != nil {
			return 0, selErr
		}
		return existing, nil
	}
	if err != nil {
		return 0, err
	}

	for i, q := range picked {
		qid, convErr := strconv.ParseInt(q.ID, 10, 64)
		if convErr != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO practice_set_questions (set_type, set_id, question_id, display_order)
			VALUES ('daily', $1, $2, $3)
		`, setID, qid, i+1); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return setID, nil
}

// weakTopics returns the user's weakest topic names (highest weakness first).
func (h *Handler) weakTopics(ctx context.Context, userID int64) ([]string, error) {
	rows, err := h.db.Query(ctx, `
		SELECT topic_name FROM skill_gaps
		WHERE user_id = $1 AND weakness_score > 0
		ORDER BY weakness_score DESC
		LIMIT 20
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

// loadSetQuestions loads the set's questions with options, ordered for display.
func (h *Handler) loadSetQuestions(ctx context.Context, setID int64) ([]PracticeQuestion, error) {
	rows, err := h.db.Query(ctx, `
		SELECT q.id, q.question_text, q.question_type,
		       COALESCE(c.title, ''), COALESCE(b.subject, '')
		FROM practice_set_questions psq
		JOIN questions q ON q.id = psq.question_id
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE psq.set_type = 'daily' AND psq.set_id = $1
		ORDER BY psq.display_order ASC
	`, setID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questions := []PracticeQuestion{}
	index := map[int64]int{}
	var ids []int64
	for rows.Next() {
		var q PracticeQuestion
		if err := rows.Scan(&q.ID, &q.Text, &q.Type, &q.Topic, &q.Subject); err != nil {
			return nil, err
		}
		q.Options = []PracticeOption{}
		index[q.ID] = len(questions)
		questions = append(questions, q)
		ids = append(ids, q.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return questions, nil
	}

	optRows, err := h.db.Query(ctx, `
		SELECT question_id, id, option_label, option_text
		FROM question_options
		WHERE question_id = ANY($1)
		ORDER BY question_id, option_label ASC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer optRows.Close()
	for optRows.Next() {
		var qid int64
		var opt PracticeOption
		if err := optRows.Scan(&qid, &opt.ID, &opt.Label, &opt.Text); err != nil {
			return nil, err
		}
		if i, ok := index[qid]; ok {
			questions[i].Options = append(questions[i].Options, opt)
		}
	}
	return questions, optRows.Err()
}

func scorePct(correct, total int) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(correct)/float64(total)*1000) / 10
}

func directionForScore(pct float64) string {
	switch {
	case pct > 80:
		return "improved"
	case pct < 60:
		return "needs_work"
	default:
		return "unchanged"
	}
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
