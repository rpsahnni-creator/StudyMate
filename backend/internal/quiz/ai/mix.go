package ai

import (
	"os"
	"strconv"
	"strings"
)

// QuestionMix defines how many questions of each type to generate.
type QuestionMix struct {
	MCQ       int
	FillBlank int
	TrueFalse int
}

func (m QuestionMix) Total() int {
	return m.MCQ + m.FillBlank + m.TrueFalse
}

// DefaultChapterMix returns the chapter-scan product mix (20 MCQ + 20 fill + 10 T/F).
func DefaultChapterMix() QuestionMix {
	return QuestionMix{MCQ: envCount("AI_MCQ_COUNT", 20), FillBlank: envCount("AI_FILL_COUNT", 20), TrueFalse: envCount("AI_TF_COUNT", 10)}
}

// DefaultExistingMix returns the existing-question scan mix (MCQ only).
func DefaultExistingMix() QuestionMix {
	return QuestionMix{MCQ: envCount("AI_EXISTING_MCQ_COUNT", 20), FillBlank: 0, TrueFalse: 0}
}

// DevChapterMix is a smaller mix for faster local stub runs unless AI_FULL_MIX=1.
func DevChapterMix() QuestionMix {
	if strings.TrimSpace(os.Getenv("AI_FULL_MIX")) == "1" {
		return DefaultChapterMix()
	}
	return QuestionMix{MCQ: 5, FillBlank: 5, TrueFalse: 3}
}

func envCount(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// VisionMaxQuestionsDefault is the soft upper cap for page-grounded vision quizzes.
func VisionMaxQuestionsDefault() int {
	return envCount("AI_VISION_MAX_QUESTIONS", 25)
}

// VisionMinQuestionsDefault is the minimum acceptable vision output.
func VisionMinQuestionsDefault() int {
	n := envCount("AI_VISION_MIN_QUESTIONS", 1)
	if n < 1 {
		return 1
	}
	return n
}
