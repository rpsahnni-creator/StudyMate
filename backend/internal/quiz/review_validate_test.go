package quiz

import "testing"

func opt(label, text string) DraftOptionInput {
	return DraftOptionInput{Label: label, Text: text}
}

func TestValidateDraftInput_MCQAndUnknownAnswer(t *testing.T) {
	in := []DraftQuestionInput{
		{
			Text:         "Capital of France?",
			Type:         "mcq",
			CorrectIndex: 1,
			Options:      []DraftOptionInput{opt("A", "Berlin"), opt("B", "Paris"), opt("C", "Rome"), opt("D", "Madrid")},
		},
		{
			Text:         "Blank _____ here",
			Type:         "fill_blank",
			CorrectIndex: -1, // reviewer has not chosen yet
			Options:      []DraftOptionInput{opt("A", "cat"), opt("B", "dog")},
		},
	}
	out, err := validateDraftInput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(out))
	}
	if out[0].AnswerStatus != "set" || out[0].CorrectIndex != 1 {
		t.Fatalf("q0 should be set with index 1, got status=%s index=%d", out[0].AnswerStatus, out[0].CorrectIndex)
	}
	if out[1].AnswerStatus != "unknown" || out[1].CorrectIndex != -1 {
		t.Fatalf("q1 should be unknown, got status=%s index=%d", out[1].AnswerStatus, out[1].CorrectIndex)
	}
}

func TestValidateDraftInput_TrueFalse(t *testing.T) {
	in := []DraftQuestionInput{
		{Text: "Sky is blue", Type: "true_false", CorrectIndex: 0, Options: []DraftOptionInput{opt("A", "True"), opt("B", "False")}},
	}
	out, err := validateDraftInput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out[0].AnswerStatus != "set" || out[0].CorrectIndex != 0 {
		t.Fatalf("true_false should be set index 0, got %s/%d", out[0].AnswerStatus, out[0].CorrectIndex)
	}
}

func TestValidateDraftInput_Rejects(t *testing.T) {
	cases := map[string][]DraftQuestionInput{
		"empty list": {},
		"empty text": {
			{Text: "  ", Type: "mcq", CorrectIndex: 0, Options: []DraftOptionInput{opt("A", "1"), opt("B", "2"), opt("C", "3"), opt("D", "4")}},
		},
		"bad type": {
			{Text: "Q", Type: "essay", CorrectIndex: 0, Options: []DraftOptionInput{opt("A", "1"), opt("B", "2")}},
		},
		"too few options": {
			{Text: "Q", Type: "mcq", CorrectIndex: 0, Options: []DraftOptionInput{opt("A", "1")}},
		},
	}
	for name, in := range cases {
		if _, err := validateDraftInput(in); err == nil {
			t.Fatalf("case %q: expected error, got nil", name)
		}
	}
}

func TestCorrectOption(t *testing.T) {
	none := correctOption([]DraftOption{{Label: "A", Text: "x"}, {Label: "B", Text: "y"}})
	if none != nil {
		t.Fatal("expected nil when no option is correct")
	}
	got := correctOption([]DraftOption{{Label: "A", Text: "x"}, {Label: "B", Text: "y", IsCorrect: true}})
	if got == nil || got.Label != "B" {
		t.Fatalf("expected option B, got %+v", got)
	}
}
