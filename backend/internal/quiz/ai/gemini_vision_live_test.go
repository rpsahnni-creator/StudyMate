package ai

import (
	"context"
	"os"
	"testing"
)

// TestRealGeminiVisionLive runs an end-to-end vision generation against the real
// Gemini API using a local page image. It only runs when GEMINI_LIVE_TEST=1 to
// avoid hitting the network in normal test runs.
//
// Required env:
//   GEMINI_LIVE_TEST=1
//   GEMINI_API_KEY=<key>
//   LIVE_IMAGE_PATH=<absolute path to a page image>
func TestRealGeminiVisionLive(t *testing.T) {
	if os.Getenv("GEMINI_LIVE_TEST") != "1" {
		t.Skip("set GEMINI_LIVE_TEST=1 to run the live Gemini vision test")
	}
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Fatal("GEMINI_API_KEY is required")
	}
	imgPath := os.Getenv("LIVE_IMAGE_PATH")
	data, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("read image %q: %v", imgPath, err)
	}

	gen, err := NewGeminiVisionGenerator(AIConfig{GeminiKey: key, TimeoutSec: 90})
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}

	req := VisionRequest{
		GenerateRequest: GenerateRequest{
			Board:          "ncert",
			ScanMode:       ScanModeChapter,
			Chapter:        "The sun - A natural source of energy",
			FlexibleVision: true,
			WantSummary:    true,
			Difficulty:     "medium",
			Language:       LanguageAuto,
			Rules:          ExplanationRulesForLanguage(LanguageAuto),
		},
		Images: []VisionImage{{Bytes: data, MIME: MIMEFromObjectRef(imgPath)}},
	}

	result, err := gen.GenerateFromImages(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateFromImages failed: %v", err)
	}

	t.Logf("=== SUMMARY: %s", result.ChapterSummary)
	t.Logf("=== MODEL: %s | tokens=%d | duration=%dms | questions=%d",
		result.ModelUsed, result.TokensUsed, result.DurationMs, len(result.Questions))
	for i, q := range result.Questions {
		t.Logf("--- Q%d [%s | topic=%s]: %s", i+1, q.Type, q.Topic, q.Text)
		for _, o := range q.Options {
			marker := " "
			t.Logf("      (%s) %s", o.Label, o.Text)
			_ = marker
		}
		t.Logf("      correct_index=%d | explanation=%s", q.CorrectIndex, q.Explanation)
	}
	if len(result.Questions) == 0 {
		t.Fatal("expected at least 1 question from the page")
	}
}
