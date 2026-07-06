package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ProviderGeminiVision = "gemini_vision"
	// gemini-2.5-flash has active free-tier quota and strong multimodal (page
	// image) reading; gemini-2.0-flash returns HTTP 429 (free_tier limit: 0).
	defaultGeminiVisionModel = "gemini-2.5-flash"
	visionTimeoutDefault     = 90
)

// GeminiVisionGenerator reads textbook page images via Gemini multimodal API.
type GeminiVisionGenerator struct {
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

func NewGeminiVisionGenerator(cfg AIConfig) (*GeminiVisionGenerator, error) {
	if strings.TrimSpace(cfg.GeminiKey) == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required for gemini_vision")
	}
	model := defaultGeminiVisionModel
	if cfg.ModelOverride != "" {
		model = cfg.ModelOverride
	}
	timeoutSec := cfg.TimeoutSec
	if timeoutSec < visionTimeoutDefault {
		timeoutSec = visionTimeoutDefault
	}
	return &GeminiVisionGenerator{
		apiKey:     cfg.GeminiKey,
		model:      model,
		timeout:    time.Duration(timeoutSec) * time.Second,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}, nil
}

func (g *GeminiVisionGenerator) ProviderName() string { return ProviderGeminiVision }

func (g *GeminiVisionGenerator) ModelName() string { return g.model }

func (g *GeminiVisionGenerator) GenerateFromImages(ctx context.Context, req VisionRequest) (*GenerateResult, error) {
	if len(req.Images) == 0 {
		return nil, fmt.Errorf("no images provided for vision generation")
	}
	start := time.Now()
	minCount := VisionMinQuestionsDefault()
	maxCount := req.GenerateRequest.visionMaxQuestions()
	rules := req.explanationRules()

	var (
		questions      []GeneratedQuestion
		chapterSummary string
		tokensUsed     int
		lastErr        error
	)

	userPrompt := buildVisionUserPrompt(req.GenerateRequest)

	for attempt := 0; attempt < 2; attempt++ {
		prompt := userPrompt
		if attempt > 0 {
			prompt = fmt.Sprintf(`Fix the JSON. Return ONLY valid JSON with %d-%d page-grounded questions (no unrelated topics).
Every question MUST have valid options: mcq=4 options, fill_blank=2-4 options with "_____" in text, true_false=2 options.

%s`, minCount, maxCount, userPrompt)
		}
		content, tokens, err := g.callVision(ctx, prompt, rules, req.Images)
		if err != nil {
			lastErr = err
			break
		}
		tokensUsed += tokens

		parsed, summary, jsonErr := parseProviderResponse(content, 0)
		if jsonErr != nil {
			lastErr = jsonErr
			continue
		}
		chapter := strings.TrimSpace(req.GenerateRequest.Chapter)
		grounded, rejected := FilterPageGroundedQuestions(parsed, chapter, summary)
		if len(rejected) > 0 {
			lastErr = fmt.Errorf("vision returned %d off-page question(s); retrying with stricter grounding", len(rejected))
			if len(grounded) >= minCount {
				parsed = grounded
				lastErr = nil
			} else if attempt == 0 {
				continue
			}
		}
		if len(grounded) > 0 {
			parsed = grounded
		}
		if len(parsed) < minCount {
			lastErr = fmt.Errorf("vision returned %d questions (min %d) — page may be unreadable", len(parsed), minCount)
			continue
		}
		if len(parsed) > maxCount {
			parsed = parsed[:maxCount]
		}
		questions = parsed
		chapterSummary = summary
		lastErr = nil
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return &GenerateResult{
		Questions:      questions,
		ChapterSummary: chapterSummary,
		TokensUsed:     tokensUsed,
		ModelUsed:      g.model,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}

func (g *GeminiVisionGenerator) callVision(ctx context.Context, userPrompt string, rules ExplanationRules, images []VisionImage) (string, int, error) {
	parts := make([]geminiPart, 0, len(images)+1)
	for _, img := range images {
		mime := img.MIME
		if mime == "" {
			mime = "image/jpeg"
		}
		parts = append(parts, geminiPart{
			InlineData: &geminiInlineData{
				MimeType: mime,
				Data:     base64.StdEncoding.EncodeToString(img.Bytes),
			},
		})
	}
	parts = append(parts, geminiPart{Text: userPrompt})

	payload := geminiRequest{
		Contents: []geminiContent{{Parts: parts}},
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: buildVisionSystemPrompt(rules)}},
		},
		GenerationConfig: geminiGenConfig{
			ResponseMIMEType: "application/json",
			Temperature:      0.2,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}

	url := fmt.Sprintf(geminiEndpointTmpl, g.model, g.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("gemini vision request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, fmt.Errorf("decode gemini vision response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return "", 0, fmt.Errorf("gemini vision api error: %s", parsed.Error.Message)
		}
		return "", 0, fmt.Errorf("gemini vision api status %d", resp.StatusCode)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", 0, fmt.Errorf("gemini vision returned no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, parsed.UsageMetadata.TotalTokenCount, nil
}

func MIMEFromObjectRef(ref string) string {
	lower := strings.ToLower(ref)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
