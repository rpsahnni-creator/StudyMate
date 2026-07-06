package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiEndpointTmpl = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

// GeminiGenerator calls the Google Gemini generateContent API in JSON mode.
// If it fails and an OpenAI key is available, it falls back to OpenAI.
type GeminiGenerator struct {
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
	fallback   Generator
}

func NewGeminiGenerator(cfg AIConfig) *GeminiGenerator {
	model := modelGeminiDefault
	if cfg.ModelOverride != "" {
		model = cfg.ModelOverride
	}
	timeout := time.Duration(cfg.TimeoutSec) * time.Second

	var fallback Generator
	if cfg.OpenAIKey != "" {
		fallback = NewOpenAIGenerator(cfg)
	}

	return &GeminiGenerator{
		apiKey:     cfg.GeminiKey,
		model:      model,
		timeout:    timeout,
		httpClient: &http.Client{Timeout: timeout},
		fallback:   fallback,
	}
}

func (g *GeminiGenerator) ProviderName() string { return ProviderGemini }

func (g *GeminiGenerator) ModelName() string { return g.model }

type geminiRequest struct {
	Contents         []geminiContent   `json:"contents"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	GenerationConfig geminiGenConfig   `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenConfig struct {
	ResponseMIMEType string  `json:"response_mime_type"`
	Temperature      float64 `json:"temperature"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		TotalTokenCount int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *GeminiGenerator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	result, err := g.generate(ctx, req)
	if err != nil && g.fallback != nil {
		return g.fallback.Generate(ctx, req)
	}
	return result, err
}

func (g *GeminiGenerator) generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	start := time.Now()
	wantCount := req.questionCount()

	userPrompt := buildUserPrompt(req)
	var (
		questions      []GeneratedQuestion
		chapterSummary string
		tokensUsed     int
		lastErr        error
	)

	for attempt := 0; attempt < 2; attempt++ {
		prompt := userPrompt
		if attempt > 0 {
			prompt = "Fix the JSON format. Return ONLY a valid JSON array with exactly " +
				fmt.Sprintf("%d", wantCount) + " questions.\n\n" + userPrompt
		}
		content, tokens, err := g.call(ctx, prompt)
		if err != nil {
			lastErr = err
			break
		}
		tokensUsed += tokens

		parsed, summary, jsonErr := parseProviderResponse(content, wantCount)
		if jsonErr != nil {
			lastErr = jsonErr
			continue
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

func (g *GeminiGenerator) call(ctx context.Context, userPrompt string) (string, int, error) {
	payload := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: userPrompt}}},
		},
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: buildSystemPrompt(DefaultExplanationRules())}},
		},
		GenerationConfig: geminiGenConfig{
			ResponseMIMEType: "application/json",
			Temperature:      0.4,
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
		return "", 0, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, fmt.Errorf("decode gemini response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return "", 0, fmt.Errorf("gemini api error: %s", parsed.Error.Message)
		}
		return "", 0, fmt.Errorf("gemini api status %d", resp.StatusCode)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", 0, fmt.Errorf("gemini returned no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, parsed.UsageMetadata.TotalTokenCount, nil
}
