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
- Read ONLY the attached page image(s). Every question, option, answer, and explanation MUST come word-for-word or fact-for-fact from text visible on that page. Treat the page as your ONLY source of truth.
- ABSOLUTELY FORBIDDEN: using your own knowledge, memory, the internet, other chapters, or any topic not printed on THIS page. If a fact is not visible on the page, you may not use it.
- Do NOT rephrase the chapter into a different subject. Never ask about photosynthesis, gravity, water cycle, electric circuits, fractions, etc. unless that exact topic is printed on this page.
- UNREADABLE PAGE: if the image is blurry, dark, cropped, rotated, blank, or you cannot clearly read enough text to write real questions, do NOT invent anything. Instead return {"questions": [], "unreadable": true} so the student is asked to scan again.
- Never fill the quiz with made-up or generic questions to reach a count. Fewer accurate, page-sourced questions is always better than any invented one.
- Question count is FLEXIBLE: create only as many questions as the page content truly supports — do NOT pad with unrelated topics.
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
