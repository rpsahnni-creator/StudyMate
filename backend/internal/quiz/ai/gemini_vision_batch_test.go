package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// mcqQuestion builds a valid 4-option MCQ payload map.
func mcqQuestion(text, topic string) map[string]any {
	return map[string]any{
		"text": text,
		"type": "mcq",
		"options": []map[string]string{
			{"label": "A", "text": "opt-a-" + topic},
			{"label": "B", "text": "opt-b-" + topic},
			{"label": "C", "text": "opt-c-" + topic},
			{"label": "D", "text": "opt-d-" + topic},
		},
		"correct_index": 0,
		"explanation":   "explanation for " + text,
		"difficulty":    "medium",
		"topic":         topic,
	}
}

func geminiCandidate(inner map[string]any) map[string]any {
	body, _ := json.Marshal(inner)
	return map[string]any{
		"candidates": []map[string]any{
			{"content": map[string]any{"parts": []map[string]string{{"text": string(body)}}}},
		},
		"usageMetadata": map[string]int{"totalTokenCount": 10},
	}
}

func newVisionGeneratorForTest(base string) *GeminiVisionGenerator {
	g, _ := NewGeminiVisionGenerator(AIConfig{GeminiKey: "test-key", TimeoutSec: 10})
	g.endpointBase = base
	return g
}

func imagesN(n int) []VisionImage {
	imgs := make([]VisionImage, 0, n)
	for i := 0; i < n; i++ {
		imgs = append(imgs, VisionImage{Bytes: []byte{0x1, 0x2, 0x3}, MIME: "image/jpeg"})
	}
	return imgs
}

// TestVisionBatchingAndMerge verifies that pages are split into batches of 3,
// results are merged, and duplicate questions (same text across batches) are
// deduplicated.
func TestVisionBatchingAndMerge(t *testing.T) {
	t.Setenv("AI_VISION_QUESTIONS_PER_PAGE", "6")
	t.Setenv("AI_VISION_MAX_QUESTIONS", "60")
	t.Setenv("AI_VISION_PAGES_PER_BATCH", "3")

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		payload := map[string]any{
			"questions": []map[string]any{
				mcqQuestion("Shared duplicate question?", "topicShared"),
				mcqQuestion(fmt.Sprintf("Unique A call %d?", n), fmt.Sprintf("topicA%d", n)),
				mcqQuestion(fmt.Sprintf("Unique B call %d?", n), fmt.Sprintf("topicB%d", n)),
			},
			"chapter_summary": "",
		}
		_ = json.NewEncoder(w).Encode(geminiCandidate(payload))
	}))
	defer srv.Close()

	gen := newVisionGeneratorForTest(srv.URL)

	// 7 pages with batch size 3 => 3 calls (3 + 3 + 1).
	req := VisionRequest{
		GenerateRequest: GenerateRequest{
			Board:          "ncert",
			ScanMode:       ScanModeChapter,
			FlexibleVision: true,
			Language:       "english",
			Rules:          ExplanationRulesForLanguage("english"),
		},
		Images: imagesN(7),
	}

	result, err := gen.GenerateFromImages(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateFromImages failed: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 batched Gemini calls for 7 pages, got %d", got)
	}

	// Each call: 1 shared + 2 unique = 3 raw. Across 3 calls: shared deduped to 1,
	// unique = 6 => total 7 questions.
	if len(result.Questions) != 7 {
		t.Fatalf("expected 7 merged/deduped questions, got %d", len(result.Questions))
	}

	sharedCount := 0
	for _, q := range result.Questions {
		if normalizeQuestionKey(q.Text) == normalizeQuestionKey("Shared duplicate question?") {
			sharedCount++
		}
	}
	if sharedCount != 1 {
		t.Fatalf("expected shared question deduped to 1, got %d", sharedCount)
	}
}

// TestVisionScalingCap verifies the overall cap is enforced across batches.
func TestVisionScalingCap(t *testing.T) {
	t.Setenv("AI_VISION_QUESTIONS_PER_PAGE", "6")
	t.Setenv("AI_VISION_MAX_QUESTIONS", "5") // tiny cap to prove truncation
	t.Setenv("AI_VISION_PAGES_PER_BATCH", "3")

	var seq int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := atomic.AddInt32(&seq, 1) * 100
		qs := make([]map[string]any, 0, 6)
		for i := 0; i < 6; i++ {
			qs = append(qs, mcqQuestion(fmt.Sprintf("Q%d?", int(base)+i), fmt.Sprintf("t%d", int(base)+i)))
		}
		_ = json.NewEncoder(w).Encode(geminiCandidate(map[string]any{"questions": qs, "chapter_summary": ""}))
	}))
	defer srv.Close()

	gen := newVisionGeneratorForTest(srv.URL)
	req := VisionRequest{
		GenerateRequest: GenerateRequest{
			Board:          "ncert",
			ScanMode:       ScanModeChapter,
			FlexibleVision: true,
			Language:       "english",
			Rules:          ExplanationRulesForLanguage("english"),
		},
		Images: imagesN(6),
	}

	result, err := gen.GenerateFromImages(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateFromImages failed: %v", err)
	}
	if len(result.Questions) != 5 {
		t.Fatalf("expected cap of 5 questions, got %d", len(result.Questions))
	}
}

// TestVisionUnreadableAllBatches verifies that if every batch is flagged
// unreadable, ErrPageUnreadable is returned (ask for rescan).
func TestVisionUnreadableAllBatches(t *testing.T) {
	t.Setenv("AI_VISION_PAGES_PER_BATCH", "3")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(geminiCandidate(map[string]any{
			"questions":  []map[string]any{},
			"unreadable": true,
		}))
	}))
	defer srv.Close()

	gen := newVisionGeneratorForTest(srv.URL)
	req := VisionRequest{
		GenerateRequest: GenerateRequest{
			Board:          "ncert",
			ScanMode:       ScanModeChapter,
			FlexibleVision: true,
			Language:       "english",
			Rules:          ExplanationRulesForLanguage("english"),
		},
		Images: imagesN(4),
	}

	_, err := gen.GenerateFromImages(context.Background(), req)
	if err != ErrPageUnreadable {
		t.Fatalf("expected ErrPageUnreadable, got %v", err)
	}
}
