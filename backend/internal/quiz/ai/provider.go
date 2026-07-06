package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	ProviderStub   = "stub"
	ProviderOpenAI = "openai"
	ProviderGemini = "gemini"

	defaultQuestionCount = 10
	defaultMaxRetries    = 3
	defaultTimeoutSec    = 30

	modelOpenAIDefault = "gpt-4o-mini"
	modelGeminiDefault = "gemini-2.0-flash"
)

// Generator produces quiz questions from educational text.
type Generator interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error)
	ProviderName() string
	ModelName() string
}

// GenerateRequest holds ephemeral input. Text is never logged.
type GenerateRequest struct {
	Text          string
	Board         string
	Subject       string
	Chapter       string
	QuestionCount int
	Difficulty    string // easy | medium | hard
	Language      string // english | hindi | bilingual
	ScanMode      string // chapter | existing_questions
	PageType      string
	Mix           QuestionMix
	WantSummary   bool
	Rules         ExplanationRules
}

func (r GenerateRequest) questionCount() int {
	if total := r.effectiveMix().Total(); total > 0 {
		return total
	}
	if r.QuestionCount <= 0 {
		return defaultQuestionCount
	}
	return r.QuestionCount
}

func (r GenerateRequest) effectiveMix() QuestionMix {
	if r.Mix.Total() > 0 {
		return r.Mix
	}
	if r.ScanMode == ScanModeExistingQuestions {
		return DefaultExistingMix()
	}
	return DefaultChapterMix()
}

func (r GenerateRequest) explanationRules() ExplanationRules {
	if r.Rules.Language != "" || r.Rules.MaxWords > 0 {
		return r.Rules
	}
	return DefaultExplanationRules()
}

func (r GenerateRequest) difficulty() string {
	if strings.TrimSpace(r.Difficulty) == "" {
		return "medium"
	}
	return r.Difficulty
}

func (r GenerateRequest) language() string {
	if strings.TrimSpace(r.Language) == "" {
		return "english"
	}
	return r.Language
}

// GenerateResult is the provider output plus usage metadata.
type GenerateResult struct {
	Questions      []GeneratedQuestion
	ChapterSummary string
	TokensUsed     int
	ModelUsed      string
	DurationMs     int64
}

// GeneratedQuestion is a single MCQ.
type GeneratedQuestion struct {
	Text         string
	Type         string
	Options      []GeneratedOption
	CorrectIndex int
	Explanation  string
	Difficulty   string
	Topic        string
}

// GeneratedOption is one MCQ choice.
type GeneratedOption struct {
	Label string
	Text  string
}

// AIConfig configures provider selection and tuning from environment.
type AIConfig struct {
	Provider      string
	OpenAIKey     string
	GeminiKey     string
	ModelOverride string
	MaxRetries    int
	TimeoutSec    int
	QuestionCount int
}

// LoadConfig reads AI settings from environment variables.
func LoadConfig() AIConfig {
	maxRetries := defaultMaxRetries
	timeoutSec := defaultTimeoutSec
	questionCount := defaultQuestionCount

	if v := strings.TrimSpace(os.Getenv("AI_TIMEOUT_SEC")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			timeoutSec = parsed
		}
	}
	if v := strings.TrimSpace(os.Getenv("AI_QUESTION_COUNT")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			questionCount = parsed
		}
	}

	return AIConfig{
		Provider:      strings.ToLower(strings.TrimSpace(os.Getenv("AI_PROVIDER"))),
		OpenAIKey:     strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		GeminiKey:     strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		ModelOverride: strings.TrimSpace(os.Getenv("AI_MODEL_OVERRIDE")),
		MaxRetries:    maxRetries,
		TimeoutSec:    timeoutSec,
		QuestionCount: questionCount,
	}
}

// NewGenerator selects an AI backend from configuration. It falls back to the
// stub generator when a provider is requested without its API key.
func NewGenerator(cfg AIConfig) (Generator, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderStub
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = defaultTimeoutSec
	}

	switch provider {
	case ProviderStub:
		return NewStubGenerator(), nil
	case ProviderOpenAI:
		if cfg.OpenAIKey == "" {
			return NewStubGenerator(), nil
		}
		return NewOpenAIGenerator(cfg), nil
	case ProviderGemini:
		if cfg.GeminiKey == "" {
			return NewStubGenerator(), nil
		}
		return NewGeminiGenerator(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI_PROVIDER %q", provider)
	}
}

// rawQuestion is the strict JSON shape returned by real providers.
type rawQuestion struct {
	Text         string      `json:"text"`
	Type         string      `json:"type"`
	Options      []rawOption `json:"options"`
	CorrectIndex int         `json:"correct_index"`
	Explanation  string      `json:"explanation"`
	Difficulty   string      `json:"difficulty"`
	Topic        string      `json:"topic"`
}

type rawProviderPayload struct {
	ChapterSummary string        `json:"chapter_summary"`
	Questions      []rawQuestion `json:"questions"`
}

type rawOption struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

// parseProviderResponse extracts questions (and optional summary) from provider JSON.
func parseProviderResponse(content string, wantCount int) ([]GeneratedQuestion, string, error) {
	trimmed := extractJSONPayload(content)

	var payload rawProviderPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err == nil && len(payload.Questions) > 0 {
		questions, err := parseAndValidate(payload.Questions, wantCount)
		return questions, payload.ChapterSummary, err
	}

	var arr []rawQuestion
	if err := json.Unmarshal([]byte(extractJSONArray(trimmed)), &arr); err != nil {
		return nil, "", fmt.Errorf("invalid JSON from provider: %w", err)
	}
	questions, err := parseAndValidate(arr, wantCount)
	return questions, "", err
}

// parseAndValidate converts raw JSON questions to domain questions with checks.
func parseAndValidate(questions []rawQuestion, wantCount int) ([]GeneratedQuestion, error) {
	if wantCount > 0 && len(questions) != wantCount {
		return nil, fmt.Errorf("expected %d questions, got %d", wantCount, len(questions))
	}
	out := make([]GeneratedQuestion, 0, len(questions))
	for i, q := range questions {
		if strings.TrimSpace(q.Text) == "" {
			return nil, fmt.Errorf("question %d has empty text", i)
		}
		qType := strings.TrimSpace(q.Type)
		if qType == "" {
			qType = QuestionTypeMCQ
		}
		minOpts, maxOpts := 4, 4
		switch qType {
		case QuestionTypeTrueFalse:
			minOpts, maxOpts = 2, 2
		case QuestionTypeFillBlank:
			minOpts, maxOpts = 2, 4
		case QuestionTypeMCQ:
			minOpts, maxOpts = 4, 4
		default:
			return nil, fmt.Errorf("question %d has unsupported type %q", i, qType)
		}
		if len(q.Options) < minOpts || len(q.Options) > maxOpts {
			return nil, fmt.Errorf("question %d type %s must have %d-%d options, got %d", i, qType, minOpts, maxOpts, len(q.Options))
		}
		if q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Options) {
			return nil, fmt.Errorf("question %d correct_index %d out of range", i, q.CorrectIndex)
		}
		options := make([]GeneratedOption, 0, len(q.Options))
		for _, o := range q.Options {
			options = append(options, GeneratedOption{Label: o.Label, Text: o.Text})
		}
		out = append(out, GeneratedQuestion{
			Text:         q.Text,
			Type:         qType,
			Options:      options,
			CorrectIndex: q.CorrectIndex,
			Explanation:  q.Explanation,
			Difficulty:   q.Difficulty,
			Topic:        q.Topic,
		})
	}
	return out, nil
}

func extractJSONPayload(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	return extractJSONArray(trimmed)
}

// extractJSONArray strips markdown fences and isolates the JSON array body.
func extractJSONArray(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "[")
	end := strings.LastIndex(trimmed, "]")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}
	return trimmed
}
