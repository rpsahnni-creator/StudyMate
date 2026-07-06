package quiz

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t.UTC()
}

func TestComputeStreak(t *testing.T) {
	now := day("2025-01-10")

	// Three consecutive days ending today.
	dates := map[string]struct{}{
		"2025-01-10": {}, "2025-01-09": {}, "2025-01-08": {},
		"2025-01-06": {}, // gap on the 7th breaks the streak
	}
	if got := computeStreak(dates, now); got != 3 {
		t.Errorf("streak = %d, want 3", got)
	}

	// No activity at all.
	if got := computeStreak(map[string]struct{}{}, now); got != 0 {
		t.Errorf("empty streak = %d, want 0", got)
	}

	// Grace: today empty but yesterday active still counts.
	grace := map[string]struct{}{"2025-01-09": {}, "2025-01-08": {}}
	if got := computeStreak(grace, now); got != 2 {
		t.Errorf("grace streak = %d, want 2", got)
	}

	// Stale: last activity 3 days ago -> streak 0.
	stale := map[string]struct{}{"2025-01-07": {}}
	if got := computeStreak(stale, now); got != 0 {
		t.Errorf("stale streak = %d, want 0", got)
	}
}

func TestTrendFor(t *testing.T) {
	if got := trendFor([]float64{50, 60, 70, 80}); got != "improving" {
		t.Errorf("trend = %q, want improving", got)
	}
	if got := trendFor([]float64{80, 70, 60, 50}); got != "declining" {
		t.Errorf("trend = %q, want declining", got)
	}
	if got := trendFor([]float64{70, 71, 70, 72}); got != "stable" {
		t.Errorf("trend = %q, want stable", got)
	}
	if got := trendFor([]float64{60}); got != "stable" {
		t.Errorf("single-sample trend = %q, want stable", got)
	}
}

func TestBuildSummary(t *testing.T) {
	now := day("2025-01-10") // ISO week 2025-W02
	attempts := []attemptRow{
		{score: 60, submittedAt: day("2025-01-02"), subject: "Physics", correct: 6, wrong: 4, skipped: 0}, // W01
		{score: 80, submittedAt: day("2025-01-09"), subject: "Physics", correct: 8, wrong: 2, skipped: 0}, // W02
		{score: 70, submittedAt: day("2025-01-10"), subject: "Math", correct: 7, wrong: 1, skipped: 2},    // W02
	}
	s := buildSummary(attempts, now)

	if s.TotalQuizzes != 3 {
		t.Errorf("totalQuizzes = %d, want 3", s.TotalQuizzes)
	}
	if s.CorrectAnswers != 21 {
		t.Errorf("correctAnswers = %d, want 21", s.CorrectAnswers)
	}
	if s.TotalQuestionsAttempted != 30 {
		t.Errorf("totalQuestionsAttempted = %d, want 30", s.TotalQuestionsAttempted)
	}
	if s.AverageScore != 70 {
		t.Errorf("averageScore = %v, want 70", s.AverageScore)
	}
	if s.ThisWeekScore != 75 { // (80+70)/2
		t.Errorf("thisWeekScore = %v, want 75", s.ThisWeekScore)
	}
	if s.LastWeekScore != 60 {
		t.Errorf("lastWeekScore = %v, want 60", s.LastWeekScore)
	}
	if s.Improvement != 15 {
		t.Errorf("improvement = %v, want 15", s.Improvement)
	}
}

func TestBuildWeeklyScoresLastN(t *testing.T) {
	var attempts []attemptRow
	// 10 weeks of one attempt each.
	base := day("2025-01-06") // Monday of W02
	for i := 0; i < 10; i++ {
		attempts = append(attempts, attemptRow{score: float64(50 + i), submittedAt: base.AddDate(0, 0, i*7)})
	}
	weeks := buildWeeklyScores(attempts, 8)
	if len(weeks) != 8 {
		t.Fatalf("weeks = %d, want 8 (last N)", len(weeks))
	}
	// Ensure chronological order.
	for i := 1; i < len(weeks); i++ {
		if weeks[i-1].Week > weeks[i].Week {
			t.Errorf("weeks not sorted: %s > %s", weeks[i-1].Week, weeks[i].Week)
		}
	}
}

func TestBuildSubjectBreakdownEmpty(t *testing.T) {
	if got := buildSubjectBreakdown(nil); got == nil || len(got) != 0 {
		t.Errorf("empty subject breakdown should be non-nil empty slice, got %v", got)
	}
}

func TestPct(t *testing.T) {
	if got := pct(1, 3); got != 33.3 {
		t.Errorf("pct(1,3) = %v, want 33.3", got)
	}
	if got := pct(0, 0); got != 0 {
		t.Errorf("pct(0,0) = %v, want 0", got)
	}
}
