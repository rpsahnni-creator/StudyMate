package scan

import (
	"strings"

	"studyapp/backend/internal/quiz/ai"
)

// DetectPageType classifies OCR text (delegates to shared AI heuristics).
func DetectPageType(text string) string {
	return ai.DetectPageType(text)
}

// ResolveGenerationMode picks effective generation mode for a job.
func ResolveGenerationMode(jobMode, pageType, strategy string) string {
	return ai.ResolveGenerationMode(jobMode, pageType, strategy)
}

func normalizeScanMode(mode string) string {
	m := strings.TrimSpace(mode)
	if m == "" {
		return ai.ScanModeChapter
	}
	return m
}
