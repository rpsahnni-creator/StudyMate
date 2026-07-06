package scan_test

import (
	"context"
	"strings"
	"testing"

	"studyapp/backend/internal/quiz/ai"
	"studyapp/backend/internal/scan/ocr"
)

// Ensures dev stub OCR + AI produce matching, multi-type quiz content.
func TestStubPipelineProducesCoherentQuiz(t *testing.T) {
	ocrProvider := ocr.NewStubProvider()
	ctx := ocr.ContextWithJobID(context.Background(), 99)

	text, err := ocrProvider.ExtractText(ctx, "temp/99/1/page.jpg")
	if err != nil {
		t.Fatalf("ocr stub failed: %v", err)
	}
	if !strings.Contains(text, "Topic:") {
		t.Fatalf("expected topic-based OCR text, got %q", text)
	}

	gen := ai.NewStubGenerator()
	result, err := gen.Generate(context.Background(), ai.GenerateRequest{
		Text:       text,
		ScanMode:   ai.ScanModeChapter,
		WantSummary: true,
		Difficulty: "medium",
	})
	if err != nil {
		t.Fatalf("ai stub failed: %v", err)
	}
	want := ai.DevChapterMix().Total()
	if len(result.Questions) != want {
		t.Fatalf("expected %d questions, got %d", want, len(result.Questions))
	}

	types := map[string]int{}
	for _, q := range result.Questions {
		types[q.Type]++
		if strings.Contains(q.Text, "Stub question") {
			t.Fatalf("legacy question text: %q", q.Text)
		}
	}
	if types[ai.QuestionTypeMCQ] == 0 || types[ai.QuestionTypeFillBlank] == 0 || types[ai.QuestionTypeTrueFalse] == 0 {
		t.Fatalf("expected all question types, got %v", types)
	}
	if result.ChapterSummary == "" {
		t.Fatal("expected chapter summary in chapter mode")
	}
}

func TestDetectPageType(t *testing.T) {
	chapter := "Topic: Photosynthesis. Green plants make food using sunlight."
	if got := ai.DetectPageType(chapter); got != ai.PageTypeChapterText {
		t.Fatalf("expected chapter_text, got %s", got)
	}
	questions := "Q1. What is gravity? A) pull B) push\nQ2. Choose the correct answer: True or False?"
	if got := ai.DetectPageType(questions); got != ai.PageTypeExistingQuestions {
		t.Fatalf("expected existing_questions, got %s", got)
	}
	mixed := "Topic: Water cycle. Q1. What is evaporation? Choose the correct option."
	if got := ai.DetectPageType(mixed); got != ai.PageTypeMixed {
		t.Fatalf("expected mixed, got %s", got)
	}
}
