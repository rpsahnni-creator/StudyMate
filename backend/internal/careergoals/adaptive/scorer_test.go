package adaptive

import (
	"math"
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return parsed
}

func TestComputeTopicScoresGroupsByTopic(t *testing.T) {
	scorer := &PerformanceScorer{}
	answers := []PracticeAnswer{
		{QuestionID: "1", Topic: "Kinematics", Correct: true},
		{QuestionID: "2", Topic: "Kinematics", Correct: false},
		{QuestionID: "3", Topic: "Kinematics", Correct: true},
		{QuestionID: "4", Topic: "Thermodynamics", Correct: true},
		{QuestionID: "5", Topic: "Thermodynamics", Correct: true},
		{QuestionID: "6", Topic: "", Correct: false}, // ignored (no topic)
	}

	scores := scorer.ComputeTopicScores(answers)

	if len(scores) != 2 {
		t.Fatalf("expected 2 topics, got %d (%v)", len(scores), scores)
	}
	if math.Abs(scores["Kinematics"]-(2.0/3.0)) > 1e-9 {
		t.Errorf("Kinematics = %v, want 0.6667", scores["Kinematics"])
	}
	if scores["Thermodynamics"] != 1.0 {
		t.Errorf("Thermodynamics = %v, want 1.0", scores["Thermodynamics"])
	}
	if _, ok := scores[""]; ok {
		t.Error("empty topic should be excluded")
	}
}

func TestComputeTopicScoresEmpty(t *testing.T) {
	scorer := &PerformanceScorer{}
	if got := scorer.ComputeTopicScores(nil); len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestISOWeekLabel(t *testing.T) {
	// 2025-01-05 (Sunday) is in ISO week 2025-W01.
	start, _ := weekBounds(mustDate(t, "2025-01-08"))
	if got := isoWeekLabel(start); got != "2025-W01" {
		t.Errorf("isoWeekLabel = %s, want 2025-W01", got)
	}
}

func TestRound1(t *testing.T) {
	cases := map[float64]float64{
		67.54: 67.5,
		67.55: 67.6,
		72.0:  72.0,
	}
	for in, want := range cases {
		if got := round1(in); math.Abs(got-want) > 1e-9 {
			t.Errorf("round1(%v) = %v, want %v", in, got, want)
		}
	}
}
