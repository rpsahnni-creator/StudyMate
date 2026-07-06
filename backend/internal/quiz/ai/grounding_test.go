package ai

import (
	"strings"
	"testing"
)

func TestFilterPageGroundedQuestions_DropsPhotosynthesisOnInterjections(t *testing.T) {
	questions := []GeneratedQuestion{
		{
			Text:  "Which word is an interjection?",
			Topic: "Interjections",
			Options: []GeneratedOption{
				{Text: "Wow!"}, {Text: "Quickly"}, {Text: "Because"}, {Text: "Under"},
			},
		},
		{
			Text:  "Where does photosynthesis mainly occur in plant cells?",
			Topic: "Photosynthesis",
			Options: []GeneratedOption{
				{Text: "Chloroplasts"}, {Text: "Mitochondria"}, {Text: "Nucleus"}, {Text: "Cell wall"},
			},
		},
	}

	filtered, rejected := FilterPageGroundedQuestions(questions, "Interjections", "Interjections express sudden feelings like wow and alas.")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 grounded question, got %d (rejected=%v)", len(filtered), rejected)
	}
	if !strings.Contains(strings.ToLower(filtered[0].Text), "interjection") {
		t.Fatalf("unexpected kept question: %q", filtered[0].Text)
	}
	if len(rejected) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(rejected))
	}
}
