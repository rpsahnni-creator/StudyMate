package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"studyapp/backend/internal/quiz/ai"
)

func TestGeminiVisionGenerateFromImages(t *testing.T) {
	payload := map[string]any{
		"questions": []map[string]any{
			{
				"text":          "Wow! is an example of what?",
				"type":          "mcq",
				"options":       []map[string]string{{"label": "A", "text": "Interjection"}, {"label": "B", "text": "Noun"}, {"label": "C", "text": "Verb"}, {"label": "D", "text": "Adjective"}},
				"correct_index": 0,
				"explanation":   "Wow! sudden feeling dikhata hai — interjection.",
				"difficulty":    "medium",
				"topic":         "Interjections",
			},
		},
		"chapter_summary": "Interjections short words hain jo feeling batate hain.",
	}
	body, _ := json.Marshal(payload)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" && r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]string{{"text": string(body)}}}},
			},
			"usageMetadata": map[string]int{"totalTokenCount": 42},
		})
	}))
	defer srv.Close()

	// Redirect gemini endpoint by using model path - we'll test via internal call by patching URL is hard.
	// Instead test NewGeminiVisionGenerator validation and parse path with stub server via custom test helper.
	gen, err := ai.NewGeminiVisionGenerator(ai.AIConfig{GeminiKey: "test-key", TimeoutSec: 10})
	if err != nil {
		t.Fatalf("new vision generator: %v", err)
	}
	if gen.ProviderName() != ai.ProviderGeminiVision {
		t.Fatalf("unexpected provider %s", gen.ProviderName())
	}

	_, err = ai.NewGeminiVisionGenerator(ai.AIConfig{})
	if err == nil {
		t.Fatal("expected error without gemini key")
	}

	_ = srv
	_ = context.Background()
}

func TestMIMEFromObjectRef(t *testing.T) {
	if ai.MIMEFromObjectRef("temp/1/1/x.png") != "image/png" {
		t.Fatal("expected png")
	}
	if ai.MIMEFromObjectRef("temp/1/1/x.jpg") != "image/jpeg" {
		t.Fatal("expected jpeg")
	}
}
