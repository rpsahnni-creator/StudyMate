package ai

import (
	"strings"
	"unicode"
)

var groundingStopWords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "that": {}, "this": {}, "with": {}, "from": {},
	"what": {}, "which": {}, "when": {}, "where": {}, "your": {}, "have": {}, "been": {},
	"does": {}, "into": {}, "about": {}, "their": {}, "there": {}, "would": {}, "should": {},
	"question": {}, "answer": {}, "option": {}, "true": {}, "false": {}, "page": {},
}

// knownOffTopicTopics are common hallucinated school topics when the page is about something else.
var knownOffTopicMarkers = []string{
	"photosynthesis", "chloroplast", "chlorophyll", "mitochondria",
	"gravity", "water cycle", "evaporation", "condensation", "precipitation",
	"electric circuit", "numerator", "denominator",
}

// FilterPageGroundedQuestions drops questions that do not relate to the scanned chapter/page.
func FilterPageGroundedQuestions(questions []GeneratedQuestion, chapterTitle, summary string) ([]GeneratedQuestion, []string) {
	if len(questions) == 0 {
		return questions, nil
	}

	anchor := buildAnchorTerms(chapterTitle, summary, questions)
	if len(anchor) == 0 {
		return questions, nil
	}

	grammarChapter := isGrammarLikeChapter(chapterTitle, summary)

	filtered := make([]GeneratedQuestion, 0, len(questions))
	rejected := make([]string, 0)

	for _, q := range questions {
		blob := questionBlob(q)
		if !questionRelatesToAnchor(blob, anchor) {
			rejected = append(rejected, truncate(q.Text, 120))
			continue
		}
		if grammarChapter && containsKnownOffTopic(blob) {
			rejected = append(rejected, truncate(q.Text, 120))
			continue
		}
		filtered = append(filtered, q)
	}

	return filtered, rejected
}

func buildAnchorTerms(chapterTitle, summary string, questions []GeneratedQuestion) map[string]struct{} {
	anchor := make(map[string]struct{})
	for _, t := range significantTokens(chapterTitle) {
		anchor[t] = struct{}{}
	}
	for _, t := range significantTokens(summary) {
		anchor[t] = struct{}{}
	}
	// Terms repeated across generated topics likely came from the page.
	topicCounts := map[string]int{}
	for _, q := range questions {
		for _, t := range significantTokens(q.Topic) {
			topicCounts[t]++
		}
	}
	for term, count := range topicCounts {
		if count >= 2 {
			anchor[term] = struct{}{}
		}
	}
	return anchor
}

func questionRelatesToAnchor(blob string, anchor map[string]struct{}) bool {
	if len(anchor) == 0 {
		return true
	}
	tokens := significantTokens(blob)
	if len(tokens) == 0 {
		return false
	}
	matches := 0
	for _, t := range tokens {
		if _, ok := anchor[t]; ok {
			matches++
		}
	}
	return matches > 0
}

func questionBlob(q GeneratedQuestion) string {
	var b strings.Builder
	b.WriteString(q.Text)
	b.WriteByte(' ')
	b.WriteString(q.Topic)
	b.WriteByte(' ')
	b.WriteString(q.Explanation)
	for _, o := range q.Options {
		b.WriteByte(' ')
		b.WriteString(o.Text)
	}
	return strings.ToLower(b.String())
}

func significantTokens(text string) []string {
	text = strings.ToLower(text)
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 3 {
			continue
		}
		if _, stop := groundingStopWords[f]; stop {
			continue
		}
		out = append(out, f)
	}
	return out
}

func isGrammarLikeChapter(chapterTitle, summary string) bool {
	blob := strings.ToLower(chapterTitle + " " + summary)
	markers := []string{
		"interjection", "noun", "verb", "adjective", "grammar", "pronoun",
		"adverb", "preposition", "conjunction", "tense", "sentence",
	}
	for _, m := range markers {
		if strings.Contains(blob, m) {
			return true
		}
	}
	return false
}

func containsKnownOffTopic(blob string) bool {
	lower := strings.ToLower(blob)
	for _, marker := range knownOffTopicMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
