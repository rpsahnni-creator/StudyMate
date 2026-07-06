package ocr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Prerequisites: apt-get install tesseract-ocr tesseract-ocr-hin

const tesseractTimeout = 30 * time.Second

// TesseractProvider runs local tesseract on a downloaded temp image file.
type TesseractProvider struct {
	minConfidence float64
	httpClient    *http.Client
}

func NewTesseractProvider(minConfidence float64) *TesseractProvider {
	return &TesseractProvider{
		minConfidence: minConfidence,
		httpClient:    &http.Client{Timeout: tesseractTimeout},
	}
}

func (t *TesseractProvider) Name() string { return ProviderTesseract }

func (t *TesseractProvider) ExtractText(ctx context.Context, imageURL string) (string, error) {
	result, err := t.extractResult(ctx, imageURL)
	if err != nil {
		return "", err
	}
	return result.RawText, nil
}

func (t *TesseractProvider) extractResult(ctx context.Context, imageURL string) (Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, tesseractTimeout)
	defer cancel()

	tmpFile, err := downloadToTemp(runCtx, t.httpClient, imageURL)
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(tmpFile)

	cmd := exec.CommandContext(runCtx, "tesseract", tmpFile, "stdout", "-l", "eng+hin")
	output, err := cmd.Output()
	if err != nil {
		return Result{}, fmt.Errorf("tesseract failed: %w", err)
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return Result{}, fmt.Errorf("tesseract returned empty text")
	}
	confidence := estimateConfidence(text, t.minConfidence)
	if confidence < t.minConfidence {
		return Result{}, fmt.Errorf("ocr confidence %.2f below minimum %.2f", confidence, t.minConfidence)
	}
	return buildResult(text, confidence, "eng+hin"), nil
}

func downloadToTemp(ctx context.Context, client *http.Client, imageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download image: status %d", resp.StatusCode)
	}

	ext := filepath.Ext(imageURL)
	if ext == "" {
		ext = ".img"
	}
	tmp, err := os.CreateTemp("", "studyapp-ocr-*"+ext)
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func estimateConfidence(text string, floor float64) float64 {
	wc := wordCount(text)
	if wc >= 20 {
		return 0.95
	}
	if wc >= 5 {
		return 0.85
	}
	if floor > 0 {
		return floor
	}
	return 0.75
}
