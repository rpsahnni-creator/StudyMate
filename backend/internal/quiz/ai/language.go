package ai

import (
	"strings"
	"unicode"
)

const LanguageAuto = "auto"

// NormalizeLanguage maps user/provider language hints to a canonical value.
func NormalizeLanguage(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch l {
	case "", LanguageAuto, "match_page":
		return LanguageAuto
	case "english", "en", "eng":
		return "english"
	case "hindi", "hi", "hin", "simple_hindi":
		return "hindi"
	default:
		return l
	}
}

// DetectContentLanguage guesses english vs hindi from OCR or generated text.
func DetectContentLanguage(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "english"
	}
	latin, other := 0, 0
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		if r <= 127 {
			latin++
		} else {
			other++
		}
	}
	if other > latin/2 {
		return "hindi"
	}
	return "english"
}

// DetectQuestionsLanguage infers quiz language from generated question bodies.
func DetectQuestionsLanguage(questions []GeneratedQuestion, fallback string) string {
	if len(questions) == 0 {
		if fb := NormalizeLanguage(fallback); fb != LanguageAuto {
			return fb
		}
		return "english"
	}
	var b strings.Builder
	for _, q := range questions {
		b.WriteString(q.Text)
		b.WriteByte(' ')
		b.WriteString(q.Explanation)
		for _, o := range q.Options {
			b.WriteByte(' ')
			b.WriteString(o.Text)
		}
	}
	return DetectContentLanguage(b.String())
}

// ExplanationRulesForLanguage builds prompt constraints for the target language.
func ExplanationRulesForLanguage(lang string) ExplanationRules {
	lang = NormalizeLanguage(lang)
	base := ExplanationRules{
		MaxWords:        150,
		RequireExample:  true,
		DefineHardWords: true,
		AvoidJargon:     true,
	}
	switch lang {
	case "english":
		base.Language = "english"
	case "hindi":
		base.Language = "simple_hindi"
	default:
		base.Language = "match_page"
	}
	return base
}

// ExplanationDBCode maps a language hint to question_explanations.language column values.
func ExplanationDBCode(lang string) string {
	switch NormalizeLanguage(lang) {
	case "hindi":
		return "hi"
	default:
		return "en"
	}
}

// PageLanguageRules instructs models to mirror the scanned page language.
func PageLanguageRules() string {
	return `PAGE LANGUAGE (mandatory — never break):
- Detect the primary language printed on the page or source text.
- Write EVERY question, option, explanation, and chapter_summary in THAT SAME language only.
- English textbook page → English questions and English explanations only.
- Hindi textbook page → Hindi questions and Hindi explanations only.
- Never translate to another language and never mix languages in one quiz.`
}
