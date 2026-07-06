package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

// analyticsTTL is how long a user's analytics payload is cached in Valkey.
const analyticsTTL = 15 * time.Minute

func analyticsCacheKey(userID int64) string {
	return fmt.Sprintf("analytics:%d", userID)
}

// invalidateAnalytics drops a user's cached analytics after their data changes.
func (s *Service) invalidateAnalytics(ctx context.Context, userID int64) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Del(ctx, analyticsCacheKey(userID)).Err(); err != nil {
		s.logger.Warn("failed to invalidate analytics cache", "error", err, "userID", userID)
	}
}

// GetAnalytics returns the user's aggregated analytics, served from Valkey when
// a fresh (15-min TTL) entry exists.
func (s *Service) GetAnalytics(ctx context.Context, userID int64) (*Analytics, error) {
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, analyticsCacheKey(userID)).Bytes(); err == nil {
			var out Analytics
			if json.Unmarshal(cached, &out) == nil {
				return &out, nil
			}
		}
	}

	analytics, err := s.computeAnalytics(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if payload, err := json.Marshal(analytics); err == nil {
			if err := s.cache.Set(ctx, analyticsCacheKey(userID), payload, analyticsTTL).Err(); err != nil {
				s.logger.Warn("failed to cache analytics", "error", err, "userID", userID)
			}
		}
	}
	return analytics, nil
}

// attemptRow is one completed attempt with the fields analytics aggregates over.
type attemptRow struct {
	score       float64
	submittedAt time.Time
	subject     string
	correct     int
	wrong       int
	skipped     int
}

// computeAnalytics runs the aggregation from a single scan of the user's
// completed attempts. Streaks and weekly buckets are computed server-side in UTC.
func (s *Service) computeAnalytics(ctx context.Context, userID int64) (*Analytics, error) {
	rows, err := s.db.Query(ctx, `
		SELECT COALESCE(a.score, 0), a.submitted_at,
		       COALESCE(b.subject, 'Unknown'),
		       a.correct_count, a.wrong_count, a.skipped_count
		FROM quiz_attempts a
		JOIN quizzes q ON q.id = a.quiz_id
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE a.user_id = $1 AND a.status = 'completed' AND a.submitted_at IS NOT NULL
		ORDER BY a.submitted_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []attemptRow
	for rows.Next() {
		var (
			r  attemptRow
			ts *time.Time
		)
		if err := rows.Scan(&r.score, &ts, &r.subject, &r.correct, &r.wrong, &r.skipped); err != nil {
			return nil, err
		}
		if ts != nil {
			r.submittedAt = ts.UTC()
		}
		attempts = append(attempts, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	analytics := &Analytics{
		SubjectBreakdown: buildSubjectBreakdown(attempts),
		WeeklyScores:     buildWeeklyScores(attempts, 8),
		RecentActivity:   buildRecentActivity(attempts, now, 14),
	}
	analytics.Summary = buildSummary(attempts, now)
	return analytics, nil
}

// GetTopicAnalytics returns per-topic accuracy, weakest first.
func (s *Service) GetTopicAnalytics(ctx context.Context, userID int64) (*TopicAnalytics, error) {
	rows, err := s.db.Query(ctx, `
		SELECT COALESCE(c.title, 'Unknown') AS topic,
		       COALESCE(b.subject, '') AS subject,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE aa.is_correct) AS correct,
		       MIN(a.quiz_id) AS sample_quiz_id
		FROM quiz_attempt_answers aa
		JOIN quiz_attempts a ON a.id = aa.attempt_id
		JOIN questions ques ON ques.id = aa.question_id
		LEFT JOIN chapters c ON c.id = ques.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE a.user_id = $1 AND a.status = 'completed'
		GROUP BY c.title, b.subject
		ORDER BY (COUNT(*) FILTER (WHERE aa.is_correct))::float / NULLIF(COUNT(*), 0) ASC, total DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &TopicAnalytics{Topics: []TopicAccuracy{}}
	for rows.Next() {
		var (
			t           TopicAccuracy
			total       int
			correct     int
			sampleQuiz  *int64
		)
		if err := rows.Scan(&t.Topic, &t.Subject, &total, &correct, &sampleQuiz); err != nil {
			return nil, err
		}
		t.TotalAnswered = total
		t.CorrectCount = correct
		t.Accuracy = pct(correct, total)
		t.SampleQuizID = sampleQuiz
		result.Topics = append(result.Topics, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// --- pure aggregation helpers ---

func buildSummary(attempts []attemptRow, now time.Time) AnalyticsSummary {
	var summary AnalyticsSummary
	summary.TotalQuizzes = len(attempts)
	if len(attempts) == 0 {
		return summary
	}

	var scoreSum float64
	dates := make(map[string]struct{}, len(attempts))
	for _, a := range attempts {
		scoreSum += a.score
		summary.TotalQuestionsAttempted += a.correct + a.wrong + a.skipped
		summary.CorrectAnswers += a.correct
		dates[a.submittedAt.Format("2006-01-02")] = struct{}{}
	}
	summary.AverageScore = round1(scoreSum / float64(len(attempts)))

	thisWeek := isoWeek(now)
	lastWeek := isoWeek(now.AddDate(0, 0, -7))
	weekly := weeklyBuckets(attempts)
	if b, ok := weekly[thisWeek]; ok {
		summary.ThisWeekScore = round1(b.sum / float64(b.count))
	}
	if b, ok := weekly[lastWeek]; ok {
		summary.LastWeekScore = round1(b.sum / float64(b.count))
	}
	summary.Improvement = round1(summary.ThisWeekScore - summary.LastWeekScore)
	summary.StudyStreakDays = computeStreak(dates, now)
	return summary
}

type bucket struct {
	sum   float64
	count int
}

func weeklyBuckets(attempts []attemptRow) map[string]*bucket {
	weekly := make(map[string]*bucket)
	for _, a := range attempts {
		label := isoWeek(a.submittedAt)
		b := weekly[label]
		if b == nil {
			b = &bucket{}
			weekly[label] = b
		}
		b.sum += a.score
		b.count++
	}
	return weekly
}

func buildWeeklyScores(attempts []attemptRow, lastN int) []WeeklyScore {
	weekly := weeklyBuckets(attempts)
	out := make([]WeeklyScore, 0, len(weekly))
	for label, b := range weekly {
		out = append(out, WeeklyScore{
			Week:      label,
			Score:     round1(b.sum / float64(b.count)),
			QuizCount: b.count,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Week < out[j].Week })
	if lastN > 0 && len(out) > lastN {
		out = out[len(out)-lastN:]
	}
	if out == nil {
		out = []WeeklyScore{}
	}
	return out
}

func buildSubjectBreakdown(attempts []attemptRow) []SubjectBreakdown {
	type subjectAgg struct {
		scores []float64
	}
	agg := make(map[string]*subjectAgg)
	var order []string
	for _, a := range attempts {
		s := agg[a.subject]
		if s == nil {
			s = &subjectAgg{}
			agg[a.subject] = s
			order = append(order, a.subject)
		}
		s.scores = append(s.scores, a.score)
	}

	out := make([]SubjectBreakdown, 0, len(order))
	for _, subject := range order {
		s := agg[subject]
		var sum float64
		for _, v := range s.scores {
			sum += v
		}
		out = append(out, SubjectBreakdown{
			Subject:   subject,
			QuizCount: len(s.scores),
			AvgScore:  round1(sum / float64(len(s.scores))),
			Trend:     trendFor(s.scores),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Subject < out[j].Subject })
	if out == nil {
		out = []SubjectBreakdown{}
	}
	return out
}

// trendFor classifies a chronological score series as improving/declining/stable
// by comparing the recent half's average to the earlier half's.
func trendFor(scores []float64) string {
	if len(scores) < 2 {
		return "stable"
	}
	mid := len(scores) / 2
	older := scores[:mid]
	recent := scores[mid:]
	diff := mean(recent) - mean(older)
	switch {
	case diff > 3:
		return "improving"
	case diff < -3:
		return "declining"
	default:
		return "stable"
	}
}

func buildRecentActivity(attempts []attemptRow, now time.Time, days int) []DailyActivity {
	cutoff := now.AddDate(0, 0, -days)
	daily := make(map[string]*bucket)
	for _, a := range attempts {
		if a.submittedAt.Before(cutoff) {
			continue
		}
		key := a.submittedAt.Format("2006-01-02")
		b := daily[key]
		if b == nil {
			b = &bucket{}
			daily[key] = b
		}
		b.sum += a.score
		b.count++
	}

	out := make([]DailyActivity, 0, len(daily))
	for date, b := range daily {
		out = append(out, DailyActivity{
			Date:      date,
			QuizCount: b.count,
			AvgScore:  round1(b.sum / float64(b.count)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if out == nil {
		out = []DailyActivity{}
	}
	return out
}

// computeStreak counts consecutive days (UTC) with at least one completed attempt,
// ending at today (or yesterday, allowing for a not-yet-practiced today).
func computeStreak(dates map[string]struct{}, now time.Time) int {
	if len(dates) == 0 {
		return 0
	}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if _, ok := dates[day.Format("2006-01-02")]; !ok {
		// Grace: allow the streak to be anchored at yesterday if today is empty.
		day = day.AddDate(0, 0, -1)
		if _, ok := dates[day.Format("2006-01-02")]; !ok {
			return 0
		}
	}
	streak := 0
	for {
		if _, ok := dates[day.Format("2006-01-02")]; !ok {
			break
		}
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

func isoWeek(t time.Time) string {
	year, week := t.UTC().ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

func round1(f float64) float64 {
	return math.Round(f*10) / 10
}

func pct(correct, total int) float64 {
	if total <= 0 {
		return 0
	}
	return round1(float64(correct) / float64(total) * 100)
}
