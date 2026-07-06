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
	maxWords := rules.MaxWords
	if maxWords <= 0 {
		maxWords = 150
	}
	return fmt.Sprintf(`STRICT OUTPUT RULES (always follow):
%s
- Each explanation: %d words maximum.
- If you use a hard word, immediately give its simple meaning in parentheses.
- Always include one small real-life example.
- Explain WHY the correct answer is right — student-friendly, no confusing jargon.
- Avoid long theory; keep answers direct and easy to remember.`, languageRulesLines(rules.Language), maxWords)
}

func languageRulesLines(lang string) string {
	switch NormalizeLanguage(lang) {
	case "english":
		return `- Language: English only — clear, school-student level.
- Questions, options, and explanations MUST all be in English.`
	case "simple_hindi", "hindi":
		return `- Language: simple Hindi (Hinglish OK) — short sentences, school-student level.
- Questions, options, and explanations MUST all be in Hindi.`
	default:
		return PageLanguageRules()
	}
}

// PageGroundingRules prevents hallucinated topics when reading scan images.
func PageGroundingRules() string {
	return `PAGE GROUNDING (mandatory — never break):
- Read ONLY the attached page image(s). Every question, option, and answer MUST come from text or facts visible on that page.
- Do NOT use general knowledge or other chapters (e.g. never ask about photosynthesis unless the page mentions it).
- Question count is FLEXIBLE: create only as many questions as the page content truly supports — do NOT pad with unrelated topics.
- Prefer fewer accurate questions over many wrong ones.
- Options must use words/concepts from the page, not random distractors from other subjects.`
}

func buildVisionSystemPrompt(rules ExplanationRules) string {
	extraLang := ""
	if NormalizeLanguage(rules.Language) == LanguageAuto || rules.Language == "match_page" {
		extraLang = PageLanguageRules() + "\n"
	}
	return "You are an educational quiz generator for Indian school students (NCERT / state board).\n" +
		PageGroundingRules() + "\n" +
		extraLang +
		SystemRulesBlock(rules) + "\n" +
		"Return ONLY valid JSON. No markdown, no text outside JSON."
}
