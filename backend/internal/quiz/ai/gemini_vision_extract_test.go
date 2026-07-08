package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// extractQuestion builds an MCQ payload with an explicit correct_index. Pass -1
// for correctIndex to simulate a page whose answer key is not printed.
func extractQuestion(text string, correctIndex int) map[string]any {
	return map[string]any{
		"text": text,
		"type": "mcq",
		"options": []map[string]string{
			{"label": "A", "text": "opt-a"},
			{"label": "B", "text": "opt-b"},
			{"label": "C", "text": "opt-c"},
			{"label": "D", "text": "opt-d"},
		},
		"correct_index": correctIndex,
		"explanation":   "",
		"difficulty":    "medium",
		"topic":         "extracted",
	}
}

// TestVisionExtractKeepsUnknownAnswers verifies question-scan extraction keeps
// printed questions even when no answer is printed (correct_index -1), storing
// CorrectIndexUnknown rather than guessing or dropping the question.
func TestVisionExtractKeepsUnknownAnswers(t *testing.T) {
	t.Setenv("AI_VISION_QUESTIONS_PER_PAGE", "6")
	t.Setenv("AI_VISION_PAGES_PER_BATCH", "3")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]any{
			"questions": []map[string]any{
				extractQuestion("Printed question with no answer key?", -1),
				extractQuestion("Printed question with a marked answer?", 2),
			},
		}
		_ = json.NewEncoder(w).Encode(geminiCandidate(payload))
	}))
	defer srv.Close()

	gen := newVisionGeneratorForTest(srv.URL)
	req := VisionRequest{
		GenerateRequest: GenerateRequest{
			Board:          "ncert",
			ScanMode:       ScanModeExistingQuestions,
			FlexibleVision: true,
			Language:       "english",
			Rules:          ExplanationRulesForLanguage("english"),
		},
		Images: imagesN(1),
	}

	result, err := gen.GenerateFromImages(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateFromImages failed: %v", err)
	}
	if len(result.Questions) != 2 {
		t.Fatalf("expected 2 extracted questions, got %d", len(result.Questions))
	}
	if result.Questions[0].CorrectIndex != CorrectIndexUnknown {
		t.Fatalf("expected first question answer unknown (%d), got %d", CorrectIndexUnknown, result.Questions[0].CorrectIndex)
	}
	if result.Questions[1].CorrectIndex != 2 {
		t.Fatalf("expected second question correct_index 2, got %d", result.Questions[1].CorrectIndex)
	}
}

// TestParseExtractionUnknownAnswer covers the parser directly for the three
// question types with a missing/invalid correct_index in extraction mode.
func TestParseExtractionUnknownAnswer(t *testing.T) {
	content := `{"questions":[
		{"text":"MCQ no answer?","type":"mcq","options":[{"label":"A","text":"1"},{"label":"B","text":"2"},{"label":"C","text":"3"},{"label":"D","text":"4"}]},
		{"text":"Blank _____ here","type":"fill_blank","options":[{"label":"A","text":"x"},{"label":"B","text":"y"}],"correct_index":-1},
		{"text":"Sky is green","type":"true_false","options":[{"label":"A","text":"True"},{"label":"B","text":"False"}],"correct_index":1}
	]}`

	questions, _, err := parseProviderResponse(content, 0, true)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}
	if questions[0].CorrectIndex != CorrectIndexUnknown {
		t.Fatalf("mcq missing correct_index should be unknown, got %d", questions[0].CorrectIndex)
	}
	if questions[1].CorrectIndex != CorrectIndexUnknown {
		t.Fatalf("fill_blank -1 should be unknown, got %d", questions[1].CorrectIndex)
	}
	if questions[2].CorrectIndex != 1 {
		t.Fatalf("true_false known answer should be 1, got %d", questions[2].CorrectIndex)
	}
}

// TestParseNonExtractionRejectsMissingAnswer verifies chapter mode still requires
// a valid correct_index (a fixed count is expected so the batch fails).
func TestParseNonExtractionRejectsMissingAnswer(t *testing.T) {
	content := `{"questions":[
		{"text":"MCQ no answer?","type":"mcq","options":[{"label":"A","text":"1"},{"label":"B","text":"2"},{"label":"C","text":"3"},{"label":"D","text":"4"}]}
	]}`
	if _, _, err := parseProviderResponse(content, 1, false); err == nil {
		t.Fatal("expected error for missing correct_index in non-extraction mode")
	}
}
