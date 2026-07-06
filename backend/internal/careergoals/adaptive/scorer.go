package adaptive

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PracticeAnswer is one graded answer used to compute per-topic performance.
type PracticeAnswer struct {
	QuestionID string
	Topic      string
	Correct    bool
}

// PerformanceScorer computes topic scores and maintains weekly goal sets.
type PerformanceScorer struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewPerformanceScorer wires the scorer.
func NewPerformanceScorer(db *pgxpool.Pool, logger *slog.Logger) *PerformanceScorer {
	if logger == nil {
		logger = slog.Default()
	}
	return &PerformanceScorer{db: db, logger: logger}
}

// ComputeTopicScores groups answers by topic and returns correct/total per
// topic as a value in [0.0, 1.0]. Answers without a topic are ignored.
func (p *PerformanceScorer) ComputeTopicScores(answers []PracticeAnswer) map[string]float64 {
	type agg struct{ correct, total int }
	byTopic := make(map[string]*agg)
	for _, a := range answers {
		if a.Topic == "" {
			continue
		}
		x := byTopic[a.Topic]
		if x == nil {
			x = &agg{}
			byTopic[a.Topic] = x
		}
		x.total++
		if a.Correct {
			x.correct++
		}
	}
	out := make(map[string]float64, len(byTopic))
	for topic, x := range byTopic {
		if x.total > 0 {
			out[topic] = float64(x.correct) / float64(x.total)
		}
	}
	return out
}

// UpdateWeeklyGoal ensures the current week's weekly_goal_sets row exists (created
// with SelectForWeekly on first call of the week), and once the week has 7 days of
// completed daily practice, appends the week's average score to
// student_goals.weekly_score_history.
//
// goalID is the student_goals.id. The ISO-ish week runs Sunday..Saturday (UTC).
func (p *PerformanceScorer) UpdateWeeklyGoal(ctx context.Context, userID, goalID string) error {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	gid, err := strconv.ParseInt(goalID, 10, 64)
	if err != nil {
		return err
	}

	weekStart, weekEnd := weekBounds(time.Now().UTC())

	created, err := p.ensureWeeklySet(ctx, uid, gid, userID, goalID, weekStart, weekEnd)
	if err != nil {
		return err
	}
	if created {
		p.logger.Debug("created weekly goal set", "userID", userID, "goalID", goalID, "weekStart", weekStart.Format("2006-01-02"))
	}

	// If the week has 7 distinct completed practice days, record the weekly score.
	completedDays, avgScore, err := p.weekProgress(ctx, uid, gid, weekStart, weekEnd)
	if err != nil {
		return err
	}
	if completedDays < 7 {
		return nil
	}
	return p.appendWeeklyScore(ctx, gid, isoWeekLabel(weekStart), avgScore)
}

// ensureWeeklySet creates the current week's set with a SelectForWeekly payload if
// it does not already exist. Returns true when a new set was created.
func (p *PerformanceScorer) ensureWeeklySet(ctx context.Context, uid, gid int64, userID, goalID string, weekStart, weekEnd time.Time) (bool, error) {
	var existing int64
	err := p.db.QueryRow(ctx, `
		SELECT id FROM weekly_goal_sets
		WHERE user_id = $1 AND student_goal_id = $2 AND week_start = $3::date
	`, uid, gid, weekStart.Format("2006-01-02")).Scan(&existing)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}

	selector := NewQuestionSelector(p.db, p.logger)
	questions, err := selector.SelectForWeekly(ctx, SelectionConfig{
		UserID:         userID,
		GoalID:         goalID,
		TotalQuestions: DefaultWeeklyQuestions,
		WeakTopicRatio: DefaultWeakTopicRatio,
	})
	if err != nil {
		return false, err
	}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var setID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO weekly_goal_sets (user_id, student_goal_id, week_start, week_end, status)
		VALUES ($1, $2, $3::date, $4::date, 'pending')
		ON CONFLICT (user_id, student_goal_id, week_start) DO NOTHING
		RETURNING id
	`, uid, gid, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02")).Scan(&setID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Lost a race: another caller created it.
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for i, q := range questions {
		qid, convErr := strconv.ParseInt(q.ID, 10, 64)
		if convErr != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO practice_set_questions (set_type, set_id, question_id, display_order)
			VALUES ('weekly', $1, $2, $3)
		`, setID, qid, i+1); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// weekProgress returns the number of distinct completed daily-practice days in the
// week and the average score across those completed days.
func (p *PerformanceScorer) weekProgress(ctx context.Context, uid, gid int64, weekStart, weekEnd time.Time) (int, float64, error) {
	var days int
	var avg *float64
	err := p.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT set_date), AVG(score)
		FROM daily_practice_sets
		WHERE user_id = $1 AND student_goal_id = $2
		  AND status = 'completed'
		  AND set_date BETWEEN $3::date AND $4::date
	`, uid, gid, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02")).Scan(&days, &avg)
	if err != nil {
		return 0, 0, err
	}
	if avg == nil {
		return days, 0, nil
	}
	return days, *avg, nil
}

// appendWeeklyScore appends {week, score} to weekly_score_history unless that week
// is already recorded (idempotent).
func (p *PerformanceScorer) appendWeeklyScore(ctx context.Context, gid int64, week string, score float64) error {
	entry, err := json.Marshal(map[string]any{"week": week, "score": round1(score)})
	if err != nil {
		return err
	}
	_, err = p.db.Exec(ctx, `
		UPDATE student_goals
		SET weekly_score_history = COALESCE(weekly_score_history, '[]'::jsonb) || $2::jsonb
		WHERE id = $1
		  AND NOT (weekly_score_history @> $3::jsonb)
	`, gid, string(entry), `[{"week":"`+week+`"}]`)
	return err
}

// --- pure helpers ---

// weekBounds returns the Sunday..Saturday bounds (date-only, UTC) for t's week.
func weekBounds(t time.Time) (time.Time, time.Time) {
	t = t.UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	offset := int(day.Weekday()) // Sunday == 0
	start := day.AddDate(0, 0, -offset)
	end := start.AddDate(0, 0, 6)
	return start, end
}

// isoWeekLabel formats a date as "YYYY-Www" using ISO week numbering.
func isoWeekLabel(t time.Time) string {
	year, week := t.ISOWeek()
	return strconv.Itoa(year) + "-W" + pad2(week)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
