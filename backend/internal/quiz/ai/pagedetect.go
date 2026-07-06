package ai

import (
	"strings"
)

// DetectPageType classifies OCR text into chapter_text, existing_questions, or mixed.
func DetectPageType(text string) string {
	lower := strings.ToLower(text)
	if strings.TrimSpace(text) == "" {
		return PageTypeChapterText
	}

	qScore := 0
	for _, marker := range []string{
		"?", "answer:", "answers:", "choose", "select", "fill in the blank", "fill in the blanks",
		"true or false", "true/false", "mcq", "question ", "q.", "q1", "q2", "option a", "option b",
	} {
		if strings.Contains(lower, marker) {
			qScore++
		}
	}

	chapterScore := 0
	for _, marker := range []string{"topic:", "chapter", "introduction", "summary", "learn about", "definition"} {
		if strings.Contains(lower, marker) {
			chapterScore++
		}
	}

	if qScore >= 2 && chapterScore >= 1 {
		return PageTypeMixed
	}
	if qScore >= 2 {
		return PageTypeExistingQuestions
	}
	return PageTypeChapterText
}

// ResolveGenerationMode picks the effective scan mode after page-type detection.
func ResolveGenerationMode(jobMode, pageType, strategy string) string {
	switch pageType {
	case PageTypeExistingQuestions:
		return ScanModeExistingQuestions
	case PageTypeChapterText:
		return ScanModeChapter
	case PageTypeMixed:
		switch strategy {
		case StrategyExtractQuestions:
			return ScanModeExistingQuestions
		case StrategyGenerateFromChapter:
			return ScanModeChapter
		default:
			return ""
		}
	default:
		if jobMode == ScanModeExistingQuestions {
			return ScanModeExistingQuestions
		}
		return ScanModeChapter
	}
}

func ValidScanMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case ScanModeChapter, ScanModeExistingQuestions, "":
		return true
	default:
		return false
	}
}

func ValidGenerationStrategy(s string) bool {
	switch strings.TrimSpace(s) {
	case StrategyExtractQuestions, StrategyGenerateFromChapter:
		return true
	default:
		return false
	}
}
