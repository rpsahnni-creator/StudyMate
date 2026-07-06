package careergoals

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// ListGoals returns the active career goals catalog. Public (no flag check) so
// the feature can be discovered before it is enabled for the user.
//
// GET /goals
func (h *Handler) ListGoals(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT id, slug, title, COALESCE(description, ''),
		       COALESCE(exam_name, ''), target_months, COALESCE(subject_areas, '[]'::jsonb)
		FROM career_goals
		WHERE active = true
		ORDER BY id ASC
	`)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	defer rows.Close()

	goals := []CareerGoal{}
	for rows.Next() {
		var g CareerGoal
		var subjects []byte
		if err := rows.Scan(&g.ID, &g.Slug, &g.Name, &g.Description, &g.ExamName, &g.TargetMonths, &subjects); err != nil {
			h.handleServiceError(w, r, err)
			return
		}
		g.SubjectAreas = decodeStringArray(subjects)
		goals = append(goals, g)
	}
	if err := rows.Err(); err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, goals)
}

// SelectGoal sets (or switches to) the user's active career goal.
//
// POST /goals/select  { goalId, targetDate }
func (h *Handler) SelectGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req SelectGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.GoalID <= 0 {
		h.jsonError(w, http.StatusBadRequest, "goalId is required")
		return
	}

	var targetDate *time.Time
	if req.TargetDate != "" {
		parsed, err := time.Parse(dateLayout, req.TargetDate)
		if err != nil {
			h.jsonError(w, http.StatusBadRequest, "targetDate must be YYYY-MM-DD")
			return
		}
		targetDate = &parsed
	}

	// Goal must exist and be active.
	var exists bool
	err := h.db.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM career_goals WHERE id = $1 AND active = true)`, req.GoalID).Scan(&exists)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	if !exists {
		h.handleServiceError(w, r, ErrGoalNotFound)
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// Abandon any currently active goal (allow switching).
	if _, err := tx.Exec(r.Context(), `
		UPDATE student_goals SET status = 'abandoned', active = false
		WHERE user_id = $1 AND status = 'active'
	`, userID); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	var studentGoalID int64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO student_goals (user_id, career_goal_id, target_date, status, active)
		VALUES ($1, $2, $3, 'active', true)
		ON CONFLICT (user_id, career_goal_id)
		DO UPDATE SET status = 'active', active = true, target_date = EXCLUDED.target_date
		RETURNING id
	`, userID, req.GoalID, targetDate).Scan(&studentGoalID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	h.json(w, http.StatusCreated, SelectGoalResponse{StudentGoalID: studentGoalID})
}

// GetMyGoal returns the user's active goal with a progress summary.
//
// GET /goals/my
func (h *Handler) GetMyGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var (
		goal       MyGoal
		targetDate *time.Time
		subjects   []byte
	)
	err := h.db.QueryRow(r.Context(), `
		SELECT sg.id, sg.career_goal_id, cg.title, COALESCE(cg.exam_name, ''),
		       sg.target_date, sg.status, COALESCE(cg.subject_areas, '[]'::jsonb)
		FROM student_goals sg
		JOIN career_goals cg ON cg.id = sg.career_goal_id
		WHERE sg.user_id = $1 AND sg.status = 'active'
		ORDER BY sg.created_at DESC
		LIMIT 1
	`, userID).Scan(&goal.StudentGoalID, &goal.GoalID, &goal.Name, &goal.ExamName, &targetDate, &goal.Status, &subjects)
	if errors.Is(err, pgx.ErrNoRows) {
		h.handleServiceError(w, r, ErrNoActiveGoal)
		return
	}
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}

	goal.SubjectAreas = decodeStringArray(subjects)
	if targetDate != nil {
		s := targetDate.Format(dateLayout)
		goal.TargetDate = &s
		days := int(time.Until(*targetDate).Hours() / 24)
		if days < 0 {
			days = 0
		}
		goal.DaysRemaining = &days
	}

	progress, err := h.loadProgress(r.Context(), userID, goal.StudentGoalID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	goal.Progress = progress

	h.json(w, http.StatusOK, goal)
}

// AbandonGoal deactivates the user's current active goal.
//
// DELETE /goals/my
func (h *Handler) AbandonGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	tag, err := h.db.Exec(r.Context(), `
		UPDATE student_goals SET status = 'abandoned', active = false
		WHERE user_id = $1 AND status = 'active'
	`, userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		h.handleServiceError(w, r, ErrNoActiveGoal)
		return
	}
	h.json(w, http.StatusOK, map[string]string{"status": "abandoned"})
}

// loadProgress computes the practice-based progress summary for a goal.
func (h *Handler) loadProgress(ctx context.Context, userID, studentGoalID int64) (GoalProgress, error) {
	var p GoalProgress

	var avg *float64
	err := h.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'completed'),
			AVG(score) FILTER (WHERE status = 'completed')
		FROM daily_practice_sets
		WHERE student_goal_id = $1
	`, studentGoalID).Scan(&p.TotalPractices, &p.CompletedPractices, &avg)
	if err != nil {
		return p, err
	}
	p.AverageScore = avg

	// Today's set (if any).
	today := time.Now().UTC().Format(dateLayout)
	var (
		todayStatus string
		todayScore  *float64
	)
	err = h.db.QueryRow(ctx, `
		SELECT status, score FROM daily_practice_sets
		WHERE student_goal_id = $1 AND set_date = $2::date
	`, studentGoalID, today).Scan(&todayStatus, &todayScore)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		p.TodayStatus = "none"
	case err != nil:
		return p, err
	default:
		p.TodayStatus = todayStatus
		p.TodayScore = todayScore
	}

	streak, err := h.loadStreak(ctx, studentGoalID)
	if err != nil {
		return p, err
	}
	p.CurrentStreak = streak

	return p, nil
}

// loadStreak counts consecutive days (ending today or yesterday) with a
// completed practice set.
func (h *Handler) loadStreak(ctx context.Context, studentGoalID int64) (int, error) {
	rows, err := h.db.Query(ctx, `
		SELECT DISTINCT set_date FROM daily_practice_sets
		WHERE student_goal_id = $1 AND status = 'completed'
		ORDER BY set_date DESC
	`, studentGoalID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		dates = append(dates, d.UTC().Truncate(24*time.Hour))
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(dates) == 0 {
		return 0, nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	// Streak is only "live" if the most recent completion is today or yesterday.
	gap := int(today.Sub(dates[0]).Hours() / 24)
	if gap > 1 {
		return 0, nil
	}

	streak := 1
	for i := 1; i < len(dates); i++ {
		diff := int(dates[i-1].Sub(dates[i]).Hours() / 24)
		if diff == 1 {
			streak++
		} else {
			break
		}
	}
	return streak, nil
}

// decodeStringArray unmarshals a JSONB string array, tolerating null/empty.
func decodeStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}
