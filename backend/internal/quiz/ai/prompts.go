package ai

import (
	"fmt"
	"strings"
)

func buildSystemPrompt(rules ExplanationRules) string {
	return "You are an educational quiz generator for Indian school students (NCERT / state board).\n" +
		SystemRulesBlock(rules) + "\n" +
		"Return ONLY valid JSON. No markdown, no text outside JSON."
}

func buildUserPrompt(req GenerateRequest) string {
	rules := req.explanationRules()
	mix := req.effectiveMix()
	total := mix.Total()
	if total <= 0 {
		total = req.questionCount()
	}

	mode := strings.TrimSpace(req.ScanMode)
	if mode == "" {
		mode = ScanModeChapter
	}

	switch mode {
	case ScanModeExistingQuestions:
		return buildExistingQuestionsPrompt(req, rules, mix)
	default:
		return buildChapterPrompt(req, rules, mix, total)
	}
}

func buildChapterPrompt(req GenerateRequest, rules ExplanationRules, mix QuestionMix, total int) string {
	summaryBlock := ""
	if req.WantSummary {
		summaryBlock = `
Also include a top-level "chapter_summary" field: 3-5 short Hindi sentences summarizing the chapter/page (simple language).`
	}
	return fmt.Sprintf(`Generate a quiz from this educational text.
Board: %s, Subject: %s, Chapter: %s
Difficulty: %s

Create exactly:
- %d MCQ (type "mcq", 4 options each)
- %d fill-in-the-blank (type "fill_blank", question text must contain "_____", 4 word/phrase options)
- %d true/false (type "true_false", exactly 2 options: True and False)

Total questions: %d
%s

Return JSON object:
{
  "chapter_summary": "optional short Hindi summary",
  "questions": [
    {
      "text": "question text",
      "type": "mcq|fill_blank|true_false",
      "options": [{"label": "A", "text": "..."}],
      "correct_index": 0,
      "explanation": "simple Hindi explanation with example and hard-word meanings",
      "difficulty": "medium",
      "topic": "topic name"
    }
  ]
}

TEXT:
%s`, req.Board, req.Subject, req.Chapter, req.difficulty(), mix.MCQ, mix.FillBlank, mix.TrueFalse, total, summaryBlock, req.Text)
}

func buildExistingQuestionsPrompt(req GenerateRequest, rules ExplanationRules, mix QuestionMix) string {
	count := mix.MCQ
	if count <= 0 {
		count = 20
	}
	return fmt.Sprintf(`This page contains PRINTED questions from a textbook. Detect and extract them — do NOT invent new chapter content.
Clean formatting, fix OCR typos, keep original meaning.
Generate an answer key with simple Hindi explanations (follow all language rules).

Extract up to %d MCQ questions (type "mcq", 4 options when possible; if only 2 options exist use 2).
If the page has fill-blank or true/false printed questions, preserve their type.

Return JSON object:
{
  "questions": [
    {
      "text": "cleaned question text",
      "type": "mcq|fill_blank|true_false",
      "options": [{"label": "A", "text": "..."}],
      "correct_index": 0,
      "explanation": "simple Hindi explanation",
      "difficulty": "medium",
      "topic": "extracted"
    }
  ]
}

TEXT:
%s`, count, req.Text)
}
