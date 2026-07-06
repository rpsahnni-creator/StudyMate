package ocr

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

const visionTimeout = 10 * time.Second

// GoogleVisionProvider calls Google Cloud Vision DOCUMENT_TEXT_DETECTION.
type GoogleVisionProvider struct {
	apiKey        string
	minConfidence float64
	httpClient    *http.Client
}

func NewGoogleVisionProvider(apiKey string, minConfidence float64) *GoogleVisionProvider {
	return &GoogleVisionProvider{
		apiKey:        apiKey,
		minConfidence: minConfidence,
		httpClient:    &http.Client{Timeout: visionTimeout},
	}
}

func (g *GoogleVisionProvider) Name() string { return ProviderGoogleVision }

func (g *GoogleVisionProvider) ExtractText(ctx context.Context, imageURL string) (string, error) {
	result, err := g.extractResult(ctx, imageURL)
	if err != nil {
		return "", err
	}
	return result.RawText, nil
}

func (g *GoogleVisionProvider) extractResult(ctx context.Context, imageURL string) (Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, visionTimeout)
	defer cancel()

	imageBytes, err := downloadBytes(runCtx, g.httpClient, imageURL)
	if err != nil {
		return Result{}, err
	}

	text, confidence, err := g.annotate(runCtx, imageBytes)
	if err != nil {
		return Result{}, err
	}
	if confidence < g.minConfidence {
		return Result{}, fmt.Errorf("vision confidence %.2f below minimum %.2f", confidence, g.minConfidence)
	}
	return buildResult(text, confidence, "eng"), nil
}

func downloadBytes(ctx context.Context, client *http.Client, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download image: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type visionRequest struct {
	Requests []visionImageRequest `json:"requests"`
}

type visionImageRequest struct {
	Image    visionImage     `json:"image"`
	Features []visionFeature `json:"features"`
}

type visionImage struct {
	Content string `json:"content"`
}

type visionFeature struct {
	Type string `json:"type"`
}

type visionResponse struct {
	Responses []struct {
		FullTextAnnotation struct {
			Text string `json:"text"`
		} `json:"fullTextAnnotation"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"responses"`
}

func (g *GoogleVisionProvider) annotate(ctx context.Context, imageBytes []byte) (string, float64, error) {
	payload := visionRequest{
		Requests: []visionImageRequest{{
			Image:    visionImage{Content: base64.StdEncoding.EncodeToString(imageBytes)},
			Features: []visionFeature{{Type: "DOCUMENT_TEXT_DETECTION"}},
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}

	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", g.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("vision api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("vision api status %d", resp.StatusCode)
	}

	var parsed visionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", 0, err
	}
	if len(parsed.Responses) == 0 {
		return "", 0, fmt.Errorf("vision api returned no responses")
	}
	if msg := parsed.Responses[0].Error.Message; msg != "" {
		return "", 0, fmt.Errorf("vision api error: %s", msg)
	}

	text := strings.TrimSpace(parsed.Responses[0].FullTextAnnotation.Text)
	if text == "" {
		return "", 0, fmt.Errorf("vision api returned empty text")
	}
	return text, estimateConfidence(text, g.minConfidence), nil
}
