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
	apiKey       string
	model        string
	timeout      time.Duration
	httpClient   *http.Client
	endpointBase string
}

// geminiVisionEndpointBase is the default Gemini generateContent base URL.
const geminiVisionEndpointBase = "https://generativelanguage.googleapis.com/v1beta/models"

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
		apiKey:       cfg.GeminiKey,
		model:        model,
		timeout:      time.Duration(timeoutSec) * time.Second,
		httpClient:   &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		endpointBase: geminiVisionEndpointBase,
	}, nil
}

func (g *GeminiVisionGenerator) ProviderName() string { return ProviderGeminiVision }

func (g *GeminiVisionGenerator) ModelName() string { return g.model }

func (g *GeminiVisionGenerator) GenerateFromImages(ctx context.Context, req VisionRequest) (*GenerateResult, error) {
	if len(req.Images) == 0 {
		return nil, fmt.Errorf("no images provided for vision generation")
	}
	start := time.Now()

	numPages := len(req.Images)
	perPage := VisionQuestionsPerPage()
	maxCap := req.GenerateRequest.visionMaxQuestions()
	overallTarget := perPage * numPages
	if overallTarget > maxCap {
		overallTarget = maxCap
	}
	if overallTarget < 1 {
		overallTarget = 1
	}
	batchSize := VisionPagesPerBatch()

	rules := req.explanationRules()

	var (
		allQuestions []GeneratedQuestion
		summary      string
		tokensUsed   int
		anyReadable  bool
		lastErr      error
		seen         = make(map[string]struct{})
	)

	for batchStart := 0; batchStart < numPages; batchStart += batchSize {
		end := batchStart + batchSize
		if end > numPages {
			end = numPages
		}
		batchImages := req.Images[batchStart:end]

		remaining := overallTarget - len(allQuestions)
		if remaining <= 0 {
			break
		}
		batchMax := perPage * len(batchImages)
		if batchMax > remaining {
			batchMax = remaining
		}

		qs, sum, tokens, unreadable, err := g.generateBatch(ctx, req.GenerateRequest, rules, batchImages, batchMax)
		tokensUsed += tokens
		if err != nil {
			lastErr = err
			continue
		}
		if unreadable {
			continue
		}
		anyReadable = true
		if summary == "" && sum != "" {
			summary = sum
		}
		for _, q := range qs {
			key := normalizeQuestionKey(q.Text)
			if key != "" {
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
			}
			allQuestions = append(allQuestions, q)
			if len(allQuestions) >= overallTarget {
				break
			}
		}
	}

	if len(allQuestions) == 0 {
		if !anyReadable {
			return nil, ErrPageUnreadable
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("vision produced no page-grounded questions")
	}

	if len(allQuestions) > overallTarget {
		allQuestions = allQuestions[:overallTarget]
	}

	return &GenerateResult{
		Questions:      allQuestions,
		ChapterSummary: summary,
		TokensUsed:     tokensUsed,
		ModelUsed:      g.model,
		DurationMs:     time.Since(start).Milliseconds(),
	}, nil
}

// generateBatch runs one Gemini vision call for a subset of page images and
// returns page-grounded questions. unreadable=true means the model flagged the
// batch as too blurry/blank to use (caller should skip, not fail).
func (g *GeminiVisionGenerator) generateBatch(
	ctx context.Context,
	gen GenerateRequest,
	rules ExplanationRules,
	images []VisionImage,
	maxCount int,
) (questions []GeneratedQuestion, summary string, tokensUsed int, unreadable bool, err error) {
	if maxCount < 1 {
		maxCount = 1
	}
	minCount := VisionMinQuestionsDefault()
	if minCount > maxCount {
		minCount = maxCount
	}

	batchReq := gen
	batchReq.VisionMaxQuestions = maxCount
	userPrompt := buildVisionUserPrompt(batchReq)
	chapter := strings.TrimSpace(gen.Chapter)
	// Question-scan extracts printed questions from the page (answers may be
	// absent). Allow unknown answers and skip chapter-topic grounding, which is
	// tuned for generated chapter quizzes and would wrongly drop real questions.
	extractMode := strings.TrimSpace(gen.ScanMode) == ScanModeExistingQuestions

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		prompt := userPrompt
		if attempt > 0 {
			prompt = fmt.Sprintf(`Fix the JSON. Return ONLY valid JSON with %d-%d page-grounded questions (no unrelated topics).
Every question MUST have valid options: mcq=4 options, fill_blank=2-4 options with "_____" in text, true_false=2 options.

%s`, minCount, maxCount, userPrompt)
		}
		content, tokens, callErr := g.callVision(ctx, prompt, rules, images)
		if callErr != nil {
			lastErr = callErr
			break
		}
		tokensUsed += tokens

		if visionResponseUnreadable(content) {
			return nil, "", tokensUsed, true, nil
		}

		parsed, sum, jsonErr := parseProviderResponse(content, 0, extractMode)
		if jsonErr != nil {
			lastErr = jsonErr
			continue
		}
		if !extractMode {
			grounded, rejected := FilterPageGroundedQuestions(parsed, chapter, sum)
			if len(grounded) > 0 {
				parsed = grounded
			} else if len(rejected) > 0 && attempt == 0 {
				lastErr = fmt.Errorf("batch returned %d off-page question(s); retrying with stricter grounding", len(rejected))
				continue
			}
		}
		if len(parsed) < minCount {
			lastErr = fmt.Errorf("batch returned %d questions (min %d)", len(parsed), minCount)
			if attempt == 0 {
				continue
			}
		}
		if len(parsed) > maxCount {
			parsed = parsed[:maxCount]
		}
		return parsed, sum, tokensUsed, false, nil
	}

	return nil, "", tokensUsed, false, lastErr
}

func normalizeQuestionKey(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(text))), " ")
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

	base := g.endpointBase
	if base == "" {
		base = geminiVisionEndpointBase
	}
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", base, g.model, g.apiKey)
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
