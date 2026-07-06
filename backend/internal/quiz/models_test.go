package quiz

import "testing"

func TestScorePercentRoundsToOneDecimal(t *testing.T) {
	cases := []struct {
		correct, total int
		want           float64
	}{
		{7, 10, 70.0},
		{1, 3, 33.3},
		{2, 3, 66.7},
		{0, 0, 0},
		{10, 10, 100.0},
	}
	for _, c := range cases {
		got := scorePercent(c.correct, c.total)
		if got != c.want {
			t.Fatalf("scorePercent(%d,%d)=%.1f want %.1f", c.correct, c.total, got, c.want)
		}
	}
}

func TestTimeLimitForQuestions(t *testing.T) {
	if got := timeLimitForQuestions(10); got != 600 {
		t.Fatalf("expected 600, got %d", got)
	}
	if got := timeLimitForQuestions(0); got != defaultTimeLimitPerQuestion {
		t.Fatalf("expected default for zero, got %d", got)
	}
}

func TestSubmitAttemptRequestValidate(t *testing.T) {
	valid := SubmitAttemptRequest{Answers: []Answer{{QuestionID: 1}}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if valid.SubmittedAt.IsZero() {
		t.Fatal("expected SubmittedAt to be defaulted")
	}

	nilAnswers := SubmitAttemptRequest{}
	if err := nilAnswers.Validate(); err == nil {
		t.Fatal("expected error for nil answers")
	}

	dup := SubmitAttemptRequest{Answers: []Answer{{QuestionID: 1}, {QuestionID: 1}}}
	if err := dup.Validate(); err == nil {
		t.Fatal("expected error for duplicate questionId")
	}

	badID := SubmitAttemptRequest{Answers: []Answer{{QuestionID: 0}}}
	if err := badID.Validate(); err == nil {
		t.Fatal("expected error for invalid questionId")
	}
}

func TestCreateAttemptRequestDefaultsStartedAt(t *testing.T) {
	req := CreateAttemptRequest{}
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.StartedAt.IsZero() {
		t.Fatal("expected StartedAt to be defaulted")
	}
}
