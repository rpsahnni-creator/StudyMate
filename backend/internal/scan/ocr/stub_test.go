package ocr

import (
	"context"
	"strings"
	"testing"
)

func TestStubProviderDeterministicText(t *testing.T) {
	p := NewStubProvider()
	first, err := p.ExtractText(context.Background(), "temp://job/42/page/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := p.ExtractText(context.Background(), "temp://job/42/page/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Fatalf("expected deterministic stub output")
	}
	if !strings.Contains(first, "Topic:") {
		t.Fatalf("unexpected stub text: %q", first)
	}
}

func TestStubProviderJobIDDeterministic(t *testing.T) {
	p := NewStubProvider()
	ctx := ContextWithJobID(context.Background(), 42)
	first, err := p.ExtractText(ctx, "temp://any/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := p.ExtractText(ctx, "temp://other/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Fatalf("expected same stub text for same job_id regardless of image URL")
	}
}

func TestStubProviderDifferentSeeds(t *testing.T) {
	p := NewStubProvider()
	a, _ := p.ExtractText(ContextWithJobID(context.Background(), 1), "temp://a")
	b, _ := p.ExtractText(ContextWithJobID(context.Background(), 2), "temp://a")
	if a == b {
		t.Fatal("expected different job IDs to produce different stub text")
	}
}

func TestStubProviderDifferentPages(t *testing.T) {
	p := NewStubProvider()
	a, _ := p.ExtractText(ContextWithJobID(context.Background(), 5), "temp/5/1/abc.jpg")
	b, _ := p.ExtractText(ContextWithJobID(context.Background(), 5), "temp/5/2/abc.jpg")
	if a == b {
		t.Fatal("expected different pages to produce different stub text")
	}
}

func TestStubProviderChapterTitle(t *testing.T) {
	p := NewStubProvider()
	ctx := ContextWithChapterTitle(ContextWithJobID(context.Background(), 9), "Interjections")
	text, err := p.ExtractText(ctx, "temp/9/1/abc.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(text), "interjection") {
		t.Fatalf("expected interjections chapter text, got %q", text)
	}
}

func TestNewProviderDefaultsToStub(t *testing.T) {
	p, err := NewProvider(OCRConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != ProviderStub {
		t.Fatalf("expected stub provider, got %s", p.Name())
	}
}

func TestNewProviderGoogleVisionRequiresKey(t *testing.T) {
	_, err := NewProvider(OCRConfig{Provider: ProviderGoogleVision})
	if err == nil {
		t.Fatal("expected error when GOOGLE_VISION_KEY missing")
	}
}

func TestExtractResultWordCount(t *testing.T) {
	p := NewStubProvider()
	result, err := ExtractResult(context.Background(), p, "temp://count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WordCount <= 0 {
		t.Fatalf("expected positive word count, got %d", result.WordCount)
	}
}
