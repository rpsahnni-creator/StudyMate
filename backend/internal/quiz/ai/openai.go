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

const openAIEndpoint = "https://api.openai.com/v1/chat/completions"

// OpenAIGenerator calls the OpenAI chat completions API.
type OpenAIGenerator struct {
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

func NewOpenAIGenerator(cfg AIConfig) *OpenAIGenerator {
	model := modelOpenAIDefault
	if cfg.ModelOverride != "" {
		model = cfg.ModelOverride
	}
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	return &OpenAIGenerator{
		apiKey:     cfg.OpenAIKey,
		model:      model,
		timeout:    timeout,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (g *OpenAIGenerator) ProviderName() string { return ProviderOpenAI }

func (g *OpenAIGenerator) ModelName() string { return g.model }

type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	ResponseFormat *openAIRespFmt  `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRespFmt struct {
	Type string `json:"type"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *OpenAIGenerator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	start := time.Now()
	wantCount := req.questionCount()

	messages := []openAIMessage{
		{Role: "system", Content: buildSystemPrompt(DefaultExplanationRules())},
		{Role: "user", Content: buildUserPrompt(req)},
	}

	var (
		questions      []GeneratedQuestion
		chapterSummary string
		tokensUsed     int
		lastErr        error
	)

	// Try up to twice: initial call, then a "fix the JSON" retry.
	for attempt := 0; attempt < 2; attempt++ {
		content, tokens, err := g.call(ctx, messages)
		if err != nil {
			lastErr = err
			break
		}
		tokensUsed += tokens

		parsed, summary, jsonErr := parseProviderResponse(content, wantCount, false)
		if jsonErr != nil {
			lastErr = jsonErr
			messages = append(messages,
				openAIMessage{Role: "assistant", Content: content},
				openAIMessage{Role: "user", Content: fmt.Sprintf("Fix the JSON format. %v. Return ONLY valid JSON with a questions array of exactly %d items.", jsonErr, wantCount)},
			)
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

func (g *OpenAIGenerator) call(ctx context.Context, messages []openAIMessage) (string, int, error) {
	payload := openAIRequest{
		Model:          g.model,
		Messages:       messages,
		ResponseFormat: &openAIRespFmt{Type: "json_object"},
		Temperature:    0.4,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, fmt.Errorf("decode openai response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return "", 0, fmt.Errorf("openai api error: %s", parsed.Error.Message)
		}
		return "", 0, fmt.Errorf("openai api status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", 0, fmt.Errorf("openai returned no choices")
	}
	return parsed.Choices[0].Message.Content, parsed.Usage.TotalTokens, nil
}
