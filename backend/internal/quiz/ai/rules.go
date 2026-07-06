package ai

import "fmt"

// ExplanationRules are hard constraints applied to every AI surface (generation + review).
type ExplanationRules struct {
	Language        string // simple_hindi | bilingual | english
	MaxWords        int
	RequireExample  bool
	DefineHardWords bool
	AvoidJargon     bool
}

// DefaultExplanationRules returns the product-standard explanation constraints.
func DefaultExplanationRules() ExplanationRules {
	return ExplanationRules{
		Language:        "simple_hindi",
		MaxWords:        150,
		RequireExample:  true,
		DefineHardWords: true,
		AvoidJargon:     true,
	}
}

// SystemRulesBlock renders the centralized prompt rules block for providers.
func SystemRulesBlock(rules ExplanationRules) string {
	lang := rules.Language
	if lang == "" {
		lang = "simple_hindi"
	}
	maxWords := rules.MaxWords
	if maxWords <= 0 {
		maxWords = 150
	}
	return fmt.Sprintf(`STRICT OUTPUT RULES (always follow):
- Language: simple Hindi (Hinglish OK) — short sentences, school-student level.
- Each explanation: %d words maximum.
- If you use a hard word, immediately give its simple meaning in parentheses.
- Always include one small real-life example.
- Explain WHY the correct answer is right — student-friendly, no confusing jargon.
- Avoid long theory; keep answers direct and easy to remember.
- Preferred language mode: %s`, maxWords, lang)
}
