// Package adaptive implements weakness-biased question selection for the career
// goals practice engine. It replaces the naive random selection that previously
// lived in careergoals.practice.go.
//
// Schema mapping (the sprint brief used an idealized schema; the reconciled
// production schema differs):
//
//   - A question's "topic"   == chapters.title   (via questions.chapter_id)
//   - A question's "subject" == books.subject     (via chapters.book_id)
//   - Weakness lives in skill_gaps(user_id, subject_name, topic_name, weakness_score)
//
// All IDs are exposed as strings on the public API (per the brief) and converted
// to BIGINT internally for queries.
package adaptive

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DefaultDailyQuestions is the target size of a daily practice set.
	DefaultDailyQuestions = 20
	// DefaultWeeklyQuestions is the target size of a weekly review set.
	DefaultWeeklyQuestions = 40
	// DefaultWeakTopicRatio is the share of a set drawn from weak topics.
	DefaultWeakTopicRatio = 0.6
	// weakTopicLimit caps how many of the weakest topics we bias towards.
	weakTopicLimit = 5
	// weeklyReviewCap caps how many "previously wrong" review questions a weekly
	// set leads with.
	weeklyReviewCap = 10
)

// Question is a selectable practice question. ID is a string per the brief's API.
type Question struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Type    string `json:"type"`
	Topic   string `json:"topic"`
	Subject string `json:"subject"`
}

// SelectionConfig controls a single selection run.
type SelectionConfig struct {
	UserID         string
	GoalID         string   // student_goals.id
	TotalQuestions int      // default 20 (daily) / 40 (weekly)
	WeakTopicRatio float64  // default 0.6
	ExcludeIDs     []string // recently-seen question ids to avoid
	// Subjects optionally scopes the "other" pool to the goal's subjects. When
	// empty it is resolved from GoalID.
	Subjects []string
}

// QuestionSelector selects questions biased towards a user's weak topics.
type QuestionSelector struct {
	db     *pgxpool.Pool
	logger *slog.Logger
	// rng is injectable for deterministic tests; nil uses the global source.
	rng *rand.Rand
}

// NewQuestionSelector wires the selector.
func NewQuestionSelector(db *pgxpool.Pool, logger *slog.Logger) *QuestionSelector {
	if logger == nil {
		logger = slog.Default()
	}
	return &QuestionSelector{db: db, logger: logger}
}

// SelectForDaily selects a weakness-biased daily set (default 20 questions).
//
// Algorithm:
//  1. Fetch the user's weakest topics (skill_gaps, weakness_score DESC).
//  2. Allocate weak_count = int(total * ratio) across those topics.
//  3. Pull weak-topic questions (random per topic), excluding recently seen.
//  4. Fill the remainder with random questions from the goal's subjects.
//  5. Shuffle and return (may be fewer than requested if the pool is small).
func (s *QuestionSelector) SelectForDaily(ctx context.Context, cfg SelectionConfig) ([]Question, error) {
	total := cfg.TotalQuestions
	if total <= 0 {
		total = DefaultDailyQuestions
	}
	ratio := cfg.WeakTopicRatio
	if ratio <= 0 {
		ratio = DefaultWeakTopicRatio
	}

	subjects, err := s.resolveSubjects(ctx, cfg)
	if err != nil {
		return nil, err
	}
	exclude := parseIDs(cfg.ExcludeIDs)

	weakTopics, err := s.weakTopics(ctx, cfg.UserID)
	if err != nil {
		return nil, err
	}

	weakCount := allocateWeak(total, ratio)
	selected := make([]Question, 0, total)
	seen := make(map[int64]struct{}, total)
	excludeSet := toSet(exclude)

	// Step 3 — weak-topic questions, distributed across the weak topics.
	alloc := perTopicAllocation(weakCount, len(weakTopics))
	for i, topic := range weakTopics {
		if alloc[i] <= 0 {
			continue
		}
		qs, err := s.selectByTopic(ctx, topic, mergeSets(excludeSet, seen), alloc[i])
		if err != nil {
			return nil, err
		}
		for _, q := range qs {
			id, _ := strconv.ParseInt(q.ID, 10, 64)
			seen[id] = struct{}{}
			selected = append(selected, q)
		}
	}

	// Step 4 — fill the remainder with random questions from the goal subjects.
	// If weak topics were short on questions, this naturally backfills them.
	otherCount := total - len(selected)
	if otherCount > 0 {
		qs, err := s.selectOther(ctx, subjects, mergeSets(excludeSet, seen), otherCount)
		if err != nil {
			return nil, err
		}
		selected = append(selected, qs...)
	}

	s.shuffle(selected)
	if len(selected) > total {
		selected = selected[:total]
	}

	s.logSelection(ctx, "daily", cfg.UserID, weakTopics, weakCount, len(selected))
	return selected, nil
}

// SelectForWeekly selects a larger, more structured review set (default 40).
// It leads with a capped set of the previous week's wrong answers, then applies
// the same weakness-biased selection, covering the goal's subjects.
func (s *QuestionSelector) SelectForWeekly(ctx context.Context, cfg SelectionConfig) ([]Question, error) {
	if cfg.TotalQuestions <= 0 {
		cfg.TotalQuestions = DefaultWeeklyQuestions
	}
	total := cfg.TotalQuestions

	exclude := parseIDs(cfg.ExcludeIDs)
	excludeSet := toSet(exclude)
	seen := make(map[int64]struct{}, total)

	selected := make([]Question, 0, total)

	// Review questions: previously wrong answers from the last 7 days.
	reviewIDs, err := s.wrongAnswerIDs(ctx, cfg.UserID, 7)
	if err != nil {
		return nil, err
	}
	if len(reviewIDs) > weeklyReviewCap {
		reviewIDs = reviewIDs[:weeklyReviewCap]
	}
	if len(reviewIDs) > 0 {
		reviews, err := s.loadByIDs(ctx, reviewIDs)
		if err != nil {
			return nil, err
		}
		for _, q := range reviews {
			id, _ := strconv.ParseInt(q.ID, 10, 64)
			seen[id] = struct{}{}
			selected = append(selected, q)
		}
	}

	// Remaining questions come from the standard weakness-biased daily algorithm,
	// scaled to the weekly total (minus the review questions already chosen).
	remaining := total - len(selected)
	if remaining > 0 {
		dailyCfg := cfg
		dailyCfg.TotalQuestions = remaining
		dailyCfg.ExcludeIDs = idsToStrings(setToSlice(mergeSets(excludeSet, seen)))
		fill, err := s.SelectForDaily(ctx, dailyCfg)
		if err != nil {
			return nil, err
		}
		for _, q := range fill {
			id, _ := strconv.ParseInt(q.ID, 10, 64)
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			selected = append(selected, q)
		}
	}

	s.shuffle(selected)
	if len(selected) > total {
		selected = selected[:total]
	}
	return selected, nil
}

// GetRecentlySeenIDs returns question ids the user has been served in daily sets
// within the last `days` days, so callers can avoid repeats.
func (s *QuestionSelector) GetRecentlySeenIDs(ctx context.Context, userID string, days int) ([]string, error) {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, err
	}
	if days <= 0 {
		days = 7
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT psq.question_id
		FROM practice_set_questions psq
		JOIN daily_practice_sets dps ON dps.id = psq.set_id AND psq.set_type = 'daily'
		WHERE dps.user_id = $1
		  AND dps.created_at > now() - make_interval(days => $2)
	`, uid, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, strconv.FormatInt(id, 10))
	}
	return out, rows.Err()
}

// --- internal query helpers ---

// weakTopics returns the user's weakest topic names, highest weakness first.
func (s *QuestionSelector) weakTopics(ctx context.Context, userID string) ([]string, error) {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT topic_name FROM skill_gaps
		WHERE user_id = $1 AND weakness_score > 0
		ORDER BY weakness_score DESC
		LIMIT $2
	`, uid, weakTopicLimit)
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
		if t != "" {
			topics = append(topics, t)
		}
	}
	return topics, rows.Err()
}

// selectByTopic pulls up to `limit` random active questions for a single topic
// (chapter title), excluding the given ids.
func (s *QuestionSelector) selectByTopic(ctx context.Context, topic string, exclude map[int64]struct{}, limit int) ([]Question, error) {
	if limit <= 0 || topic == "" {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT q.id, q.question_text, q.question_type,
		       COALESCE(c.title, ''), COALESCE(b.subject, '')
		FROM questions q
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE q.status = 'active'
		  AND c.title = $1
		  AND NOT (q.id = ANY($2))
		ORDER BY random()
		LIMIT $3
	`, topic, setToSlice(exclude), limit)
	if err != nil {
		return nil, err
	}
	return scanQuestions(rows)
}

// selectOther pulls up to `limit` random active questions from the goal subjects,
// excluding the given ids.
func (s *QuestionSelector) selectOther(ctx context.Context, subjects []string, exclude map[int64]struct{}, limit int) ([]Question, error) {
	if limit <= 0 {
		return nil, nil
	}
	if subjects == nil {
		subjects = []string{}
	}
	rows, err := s.db.Query(ctx, `
		SELECT q.id, q.question_text, q.question_type,
		       COALESCE(c.title, ''), COALESCE(b.subject, '')
		FROM questions q
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE q.status = 'active'
		  AND (cardinality($1::text[]) = 0 OR b.subject = ANY($1))
		  AND NOT (q.id = ANY($2))
		ORDER BY random()
		LIMIT $3
	`, subjects, setToSlice(exclude), limit)
	if err != nil {
		return nil, err
	}
	return scanQuestions(rows)
}

// loadByIDs loads questions by id, preserving no particular order.
func (s *QuestionSelector) loadByIDs(ctx context.Context, ids []int64) ([]Question, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT q.id, q.question_text, q.question_type,
		       COALESCE(c.title, ''), COALESCE(b.subject, '')
		FROM questions q
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE q.status = 'active' AND q.id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	return scanQuestions(rows)
}

// wrongAnswerIDs returns distinct question ids the user answered incorrectly in
// completed attempts within the last `days` days.
func (s *QuestionSelector) wrongAnswerIDs(ctx context.Context, userID string, days int) ([]int64, error) {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT aa.question_id
		FROM quiz_attempt_answers aa
		JOIN quiz_attempts a ON a.id = aa.attempt_id
		WHERE a.user_id = $1
		  AND aa.is_correct = false
		  AND a.submitted_at > now() - make_interval(days => $2)
	`, uid, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// resolveSubjects returns the goal's subjects, using the config override when set.
func (s *QuestionSelector) resolveSubjects(ctx context.Context, cfg SelectionConfig) ([]string, error) {
	if len(cfg.Subjects) > 0 {
		return cfg.Subjects, nil
	}
	if cfg.GoalID == "" {
		return []string{}, nil
	}
	gid, err := strconv.ParseInt(cfg.GoalID, 10, 64)
	if err != nil {
		return []string{}, nil
	}
	var raw []byte
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(cg.subject_areas, '[]'::jsonb)
		FROM student_goals sg
		JOIN career_goals cg ON cg.id = sg.career_goal_id
		WHERE sg.id = $1
	`, gid).Scan(&raw)
	if err != nil {
		// A missing goal is not fatal for selection; fall back to all subjects.
		return []string{}, nil
	}
	var subjects []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &subjects)
	}
	return subjects, nil
}

// logSelection records the selection rationale at debug level, both to the
// structured logger and the job_events audit table.
func (s *QuestionSelector) logSelection(ctx context.Context, kind, userID string, weakTopics []string, weakCount, total int) {
	msg := fmt.Sprintf("adaptive %s selection: user=%s total=%d weakCount=%d weakTopics=[%s]",
		kind, userID, total, weakCount, strings.Join(weakTopics, ", "))
	s.logger.Debug(msg, "kind", kind, "userID", userID, "total", total, "weakCount", weakCount, "weakTopics", weakTopics)
	_, _ = s.db.Exec(ctx, `
		INSERT INTO job_events (job_id, event_type, message, created_at)
		VALUES ($1, $2, $3, now())
	`, "adaptive:"+kind+":"+userID, "adaptive_selection", msg)
}

func (s *QuestionSelector) shuffle(qs []Question) {
	swap := func(i, j int) { qs[i], qs[j] = qs[j], qs[i] }
	if s.rng != nil {
		s.rng.Shuffle(len(qs), swap)
		return
	}
	rand.Shuffle(len(qs), swap)
}

// --- pure helpers (unit-tested) ---

// allocateWeak returns the number of weak-topic questions for a total set size
// and ratio, matching the brief's `int(total * ratio)` truncation.
func allocateWeak(total int, ratio float64) int {
	if total <= 0 {
		return 0
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return int(float64(total) * ratio)
}

// perTopicAllocation splits weakCount as evenly as possible across numTopics,
// giving any remainder to the earliest (weakest) topics.
func perTopicAllocation(weakCount, numTopics int) []int {
	if numTopics <= 0 || weakCount <= 0 {
		return make([]int, max(numTopics, 0))
	}
	out := make([]int, numTopics)
	base := weakCount / numTopics
	rem := weakCount % numTopics
	for i := range out {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out
}

func scanQuestions(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]Question, error) {
	defer rows.Close()
	var out []Question
	for rows.Next() {
		var (
			id int64
			q  Question
		)
		if err := rows.Scan(&id, &q.Text, &q.Type, &q.Topic, &q.Subject); err != nil {
			return nil, err
		}
		q.ID = strconv.FormatInt(id, 10)
		out = append(out, q)
	}
	return out, rows.Err()
}

func parseIDs(ids []string) []int64 {
	out := make([]int64, 0, len(ids))
	for _, s := range ids {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func idsToStrings(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, v := range ids {
		out = append(out, strconv.FormatInt(v, 10))
	}
	return out
}

func toSet(ids []int64) map[int64]struct{} {
	m := make(map[int64]struct{}, len(ids))
	for _, v := range ids {
		m[v] = struct{}{}
	}
	return m
}

func mergeSets(a, b map[int64]struct{}) map[int64]struct{} {
	m := make(map[int64]struct{}, len(a)+len(b))
	for k := range a {
		m[k] = struct{}{}
	}
	for k := range b {
		m[k] = struct{}{}
	}
	return m
}

func setToSlice(m map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
