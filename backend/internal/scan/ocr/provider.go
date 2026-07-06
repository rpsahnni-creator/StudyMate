package ocr

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderStub         = "stub"
	ProviderTesseract    = "tesseract"
	ProviderGoogleVision = "google_vision"
	ProviderGeminiVision = "gemini_vision"
)

// OCRConfig holds provider selection and tuning from environment.
type OCRConfig struct {
	Provider        string
	GoogleVisionKey string
	MinConfidence   float64
	MaxPagesPerJob  int
}

// LoadOCRConfig reads OCR settings from environment variables.
func LoadOCRConfig() OCRConfig {
	minConf := 0.7
	if v := strings.TrimSpace(os.Getenv("OCR_MIN_CONFIDENCE")); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			minConf = parsed
		}
	}
	maxPages := 10
	if v := strings.TrimSpace(os.Getenv("OCR_MAX_PAGES_PER_JOB")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxPages = parsed
		}
	}
	return OCRConfig{
		Provider:        strings.ToLower(strings.TrimSpace(os.Getenv("OCR_PROVIDER"))),
		GoogleVisionKey: strings.TrimSpace(os.Getenv("GOOGLE_VISION_KEY")),
		MinConfidence:   minConf,
		MaxPagesPerJob:  maxPages,
	}
}

type contextKey int

const jobIDContextKey contextKey = iota
const chapterTitleContextKey contextKey = 1

// ContextWithJobID attaches a scan job ID for providers that need deterministic per-job output.
func ContextWithJobID(ctx context.Context, jobID int64) context.Context {
	return context.WithValue(ctx, jobIDContextKey, jobID)
}

// ContextWithChapterTitle passes the user-entered chapter name into OCR (stub uses it for topic selection).
func ContextWithChapterTitle(ctx context.Context, title string) context.Context {
	title = strings.TrimSpace(title)
	if title == "" {
		return ctx
	}
	return context.WithValue(ctx, chapterTitleContextKey, title)
}

func jobIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(jobIDContextKey).(int64)
	return id, ok
}

func chapterTitleFromContext(ctx context.Context) string {
	title, _ := ctx.Value(chapterTitleContextKey).(string)
	return strings.TrimSpace(title)
}

// Result holds ephemeral OCR output. RawText must NEVER be persisted or logged.
type Result struct {
	RawText     string
	Confidence  float64
	Language    string
	WordCount   int
	ProcessedAt time.Time
}

// Provider extracts text from an image URL. Implementations must not log raw text.
type Provider interface {
	ExtractText(ctx context.Context, imageURL string) (string, error)
	Name() string
}

type resultProvider interface {
	extractResult(ctx context.Context, imageURL string) (Result, error)
}

// NewProvider selects an OCR backend from configuration.
func NewProvider(cfg OCRConfig) (Provider, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderStub
	}
	switch provider {
	case ProviderStub:
		return NewStubProvider(), nil
	case ProviderGeminiVision:
		// OCR is skipped; worker uses Gemini Vision on image bytes directly.
		return NewStubProvider(), nil
	case ProviderTesseract:
		return NewTesseractProvider(cfg.MinConfidence), nil
	case ProviderGoogleVision:
		if cfg.GoogleVisionKey == "" {
			return nil, fmt.Errorf("GOOGLE_VISION_KEY is required when OCR_PROVIDER=google_vision")
		}
		return NewGoogleVisionProvider(cfg.GoogleVisionKey, cfg.MinConfidence), nil
	default:
		return nil, fmt.Errorf("unsupported OCR_PROVIDER %q", provider)
	}
}

// ExtractResult returns structured OCR metadata without exposing raw text in logs.
func ExtractResult(ctx context.Context, p Provider, imageURL string) (Result, error) {
	if rp, ok := p.(resultProvider); ok {
		return rp.extractResult(ctx, imageURL)
	}
	text, err := p.ExtractText(ctx, imageURL)
	if err != nil {
		return Result{}, err
	}
	return buildResult(text, 1.0, "eng"), nil
}

func buildResult(text string, confidence float64, language string) Result {
	return Result{
		RawText:     text,
		Confidence:  confidence,
		Language:    language,
		WordCount:   wordCount(text),
		ProcessedAt: time.Now().UTC(),
	}
}

func wordCount(text string) int {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return 0
	}
	return len(fields)
}
