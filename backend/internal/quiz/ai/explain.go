package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ExplainRequest describes a single question whose answer is now known and needs
// a short student-facing explanation.
type ExplainRequest struct {
	QuestionText string
	QuestionType string
	Options      []GeneratedOption
	CorrectLabel string
	CorrectText  string
	// Language hint: "english", "hindi", or "" to auto-detect from the question.
	Language string
}

// Explainer produces a short explanation for why an answer is correct. It is used
// when publishing a scanned (question-scan) exam after the reviewer fills in the
// answers, mirroring the explanations chapter-scan generates at scan time.
type Explainer interface {
	Explain(ctx context.Context, req ExplainRequest) (string, error)
	ProviderName() string
}

// NewExplainer selects an explainer from AI config. It uses Gemini when a key is
// present, otherwise a deterministic stub so publishing always works offline.
func NewExplainer(cfg AIConfig) Explainer {
	if strings.TrimSpace(cfg.GeminiKey) != "" {
		model := modelGeminiDefault
		if cfg.ModelOverride != "" {
			model = cfg.ModelOverride
		}
		timeoutSec := cfg.TimeoutSec
		if timeoutSec <= 0 {
			timeoutSec = defaultTimeoutSec
		}
		return &geminiExplainer{
			apiKey:     cfg.GeminiKey,
			model:      model,
			httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		}
	}
	return stubExplainer{}
}

// stubExplainer returns a concise, deterministic explanation from the correct
// answer. Used in dev/CI and as a fallback when the AI call fails.
type stubExplainer struct{}

func (stubExplainer) ProviderName() string { return ProviderStub }

func (stubExplainer) Explain(_ context.Context, req ExplainRequest) (string, error) {
	return fallbackExplanation(req), nil
}

func fallbackExplanation(req ExplainRequest) string {
	answer := strings.TrimSpace(req.CorrectText)
	if answer == "" {
		answer = strings.TrimSpace(req.CorrectLabel)
	}
	if answer == "" {
		return "Review the question and the marked correct option."
	}
	if NormalizeLanguage(req.Language) == "hindi" {
		return fmt.Sprintf("Sahi jawab \"%s\" hai.", answer)
	}
	return fmt.Sprintf("The correct answer is \"%s\".", answer)
}

type geminiExplainer struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func (g *geminiExplainer) ProviderName() string { return ProviderGemini }

func (g *geminiExplainer) Explain(ctx context.Context, req ExplainRequest) (string, error) {
	prompt := buildExplainPrompt(req)
	payload := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: geminiGenConfig{
			ResponseMIMEType: "application/json",
			Temperature:      0.3,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fallbackExplanation(req), nil
	}
	url := fmt.Sprintf(geminiEndpointTmpl, g.model, g.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fallbackExplanation(req), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return fallbackExplanation(req), nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallbackExplanation(req), nil
	}
	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fallbackExplanation(req), nil
	}
	if resp.StatusCode != http.StatusOK || len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return fallbackExplanation(req), nil
	}

	var out struct {
		Explanation string `json:"explanation"`
	}
	content := extractJSONPayload(parsed.Candidates[0].Content.Parts[0].Text)
	if err := json.Unmarshal([]byte(content), &out); err != nil || strings.TrimSpace(out.Explanation) == "" {
		return fallbackExplanation(req), nil
	}
	return strings.TrimSpace(out.Explanation), nil
}

func buildExplainPrompt(req ExplainRequest) string {
	var opts strings.Builder
	for _, o := range req.Options {
		label := o.Label
		if label == "" {
			label = "-"
		}
		fmt.Fprintf(&opts, "\n%s) %s", label, o.Text)
	}
	langLine := "Detect the question language and write the explanation in that same language."
	switch NormalizeLanguage(req.Language) {
	case "english":
		langLine = "Write the explanation in English."
	case "hindi":
		langLine = "Write the explanation in simple Hindi."
	}
	return fmt.Sprintf(`A student answered this question. Explain briefly (1-2 sentences) why the marked answer is correct.

Question: %s
Options:%s
Correct answer: %s

RULES:
- Base the explanation only on the question and its correct answer. Do not invent unrelated facts.
- %s

Return JSON: {"explanation": "..."}`, req.QuestionText, opts.String(), strings.TrimSpace(req.CorrectLabel+" "+req.CorrectText), langLine)
}
