package ai

import (
	"context"
	"testing"
	"time"
)

func TestStubGeneratorReturnsChapterMix(t *testing.T) {
	g := NewStubGenerator()
	res, err := g.Generate(context.Background(), GenerateRequest{Text: "photosynthesis basics", ScanMode: ScanModeChapter})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := DevChapterMix().Total()
	if len(res.Questions) != want {
		t.Fatalf("expected %d questions, got %d", want, len(res.Questions))
	}
	types := map[string]int{}
	for i, q := range res.Questions {
		types[q.Type]++
		optCount := len(q.Options)
		if q.Type == QuestionTypeTrueFalse && optCount != 2 {
			t.Fatalf("question %d expected 2 options, got %d", i, optCount)
		}
		if q.Type != QuestionTypeTrueFalse && optCount != 4 {
			t.Fatalf("question %d expected 4 options, got %d", i, optCount)
		}
	}
	if types[QuestionTypeMCQ] == 0 || types[QuestionTypeFillBlank] == 0 || types[QuestionTypeTrueFalse] == 0 {
		t.Fatalf("expected all types, got %v", types)
	}
}

func TestStubGeneratorDeterministic(t *testing.T) {
	g := NewStubGenerator()
	a, _ := g.Generate(context.Background(), GenerateRequest{Text: "same text", ScanMode: ScanModeChapter})
	b, _ := g.Generate(context.Background(), GenerateRequest{Text: "same text", ScanMode: ScanModeChapter})
	if a.Questions[0].CorrectIndex != b.Questions[0].CorrectIndex {
		t.Fatal("expected deterministic output for identical text")
	}
}

func TestParseAndValidateRejectsWrongCount(t *testing.T) {
	raw := []rawQuestion{{
		Text:    "q",
		Type:    QuestionTypeMCQ,
		Options: []rawOption{{Label: "A"}, {Label: "B"}, {Label: "C"}, {Label: "D"}},
	}}
	if _, err := parseAndValidate(raw, 10); err == nil {
		t.Fatal("expected error for wrong question count")
	}
}

func TestParseAndValidateAcceptsTrueFalse(t *testing.T) {
	raw := []rawQuestion{{
		Text:         "Gravity pulls objects down.",
		Type:         QuestionTypeTrueFalse,
		Options:      []rawOption{{Label: "A", Text: "True"}, {Label: "B", Text: "False"}},
		CorrectIndex: 0,
	}}
	got, err := parseAndValidate(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Type != QuestionTypeTrueFalse {
		t.Fatalf("expected true_false, got %s", got[0].Type)
	}
}

func TestParseAndValidateRejectsBadMCQOptions(t *testing.T) {
	raw := []rawQuestion{{
		Text:    "q",
		Type:    QuestionTypeMCQ,
		Options: []rawOption{{Label: "A"}, {Label: "B"}},
	}}
	if _, err := parseAndValidate(raw, 1); err == nil {
		t.Fatal("expected error for wrong option count")
	}
}

func TestExtractJSONArrayStripsFences(t *testing.T) {
	in := "```json\n[{\"a\":1}]\n```"
	out := extractJSONArray(in)
	if out != `[{"a":1}]` {
		t.Fatalf("unexpected extraction: %q", out)
	}
}

func TestEstimateCostKnownModel(t *testing.T) {
	cost := EstimateCost(modelOpenAIDefault, 1_000_000)
	if cost < 0.32 || cost > 0.34 {
		t.Fatalf("unexpected cost estimate: %f", cost)
	}
}

func TestRateLimiterBlocksAfterMax(t *testing.T) {
	l := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow(42) {
			t.Fatalf("call %d should be allowed", i)
		}
	}
	if l.Allow(42) {
		t.Fatal("4th call should be blocked")
	}
	if !l.Allow(99) {
		t.Fatal("different user should be allowed")
	}
}

func TestNewGeneratorFallsBackToStubWithoutKey(t *testing.T) {
	g, err := NewGenerator(AIConfig{Provider: ProviderOpenAI})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.ProviderName() != ProviderStub {
		t.Fatalf("expected stub fallback, got %s", g.ProviderName())
	}
}

func TestDetectPageTypeMixed(t *testing.T) {
	got := DetectPageType("Topic: Plants. Q1. Choose the correct answer?")
	if got != PageTypeMixed {
		t.Fatalf("expected mixed, got %s", got)
	}
}
