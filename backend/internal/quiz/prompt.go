package quiz

import "studyapp/backend/internal/quiz/ai"

// PromptRules mirrors centralized AI explanation rules (see quiz/ai/rules.go).
type PromptRules = ai.ExplanationRules

func DefaultPromptRules() PromptRules {
	return ai.DefaultExplanationRules()
}

// BuildPrompt renders a legacy single-shot prompt using centralized rules.
func BuildPrompt(content string, mode string, rules PromptRules) string {
	if rules.Language == "" {
		rules = ai.DefaultExplanationRules()
	}
	return ai.SystemRulesBlock(rules) + "\nCreate a " + mode + "-style educational quiz from this content. Content: " + content
}
