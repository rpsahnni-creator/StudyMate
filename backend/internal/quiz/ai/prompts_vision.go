package ai

import (
	"fmt"
	"strings"
)

func buildVisionUserPrompt(req GenerateRequest) string {
	chapter := strings.TrimSpace(req.Chapter)
	if chapter == "" {
		chapter = "(read topic from the page image)"
	}

	mode := strings.TrimSpace(req.ScanMode)
	if mode == "" {
		mode = ScanModeChapter
	}

	maxQ := req.visionMaxQuestions()
	minQ := VisionMinQuestionsDefault()

	if mode == ScanModeExistingQuestions {
		return fmt.Sprintf(`Extract the questions that are already printed on the attached page image(s). Do NOT create new questions.

Chapter/topic hint: %s
Board: %s

RULES:
- Extract ONLY the exact questions printed on the page — never invent, rephrase into a new topic, or add extra questions.
- Return as many as are printed (minimum %d, maximum %d). Fewer is fine if the page has fewer.
- Detect and preserve each question's real type:
  - mcq: the question already lists choices. Give exactly 4 options (A-D) using the printed choices; if only 2-3 are printed, keep the printed ones and add plausible ones from the same page so there are 4.
  - fill_blank: the question has a blank. Put "_____" where the blank is and provide 2-4 short word/phrase options (never an empty options array).
  - true_false: a statement to judge. Use exactly 2 options: "True", "False".
- ANSWER KEY: If the page (or an answer key / answers section / bold/ticked/circled/underlined choice) shows the correct answer, set "correct_index" to that option's 0-based position.
- If the correct answer is NOT printed anywhere on the page, set "correct_index" to -1 (unknown). NEVER guess an answer that is not on the page — a human reviewer will fill it in.
- Only write "explanation" if the answer is known from the page; otherwise leave it "".
- %s

Return JSON:
{
  "questions": [
    {
      "text": "exact question from the page",
      "type": "mcq|fill_blank|true_false",
      "options": [{"label": "A", "text": "..."}],
      "correct_index": 0,
      "explanation": "explanation in the page language, or empty if answer unknown",
      "difficulty": "medium",
      "topic": "from page"
    }
  ]
}`, chapter, req.Board, minQ, maxQ, visionLanguageLine(req))
	}

	summaryBlock := ""
	if req.WantSummary {
		summaryBlock = `
Include "chapter_summary": 2-4 short sentences in the page language summarizing ONLY what is written on the page.`
	}

	langLine := visionLanguageLine(req)

	return fmt.Sprintf(`Create a quiz from the attached NCERT/state-board page image(s).

Chapter/topic (MUST match page — user hint): %s
Board: %s
Difficulty: %s

STRICT RULES:
- Read the image carefully. Every question MUST be about "%s" or the exact topic printed on the page.
- NEVER ask about photosynthesis, gravity, water cycle, electric circuits, fractions, etc. unless that exact topic is on THIS page.
- Use ONLY content visible on the page — every question must be answerable from the page text.
- Question count is FLEXIBLE: generate between %d and %d questions based on how much content the page has.
- Use mcq, fill_blank, and true_false only where the page content supports each type.
- mcq: exactly 4 options (A-D).
- fill_blank: question text MUST contain "_____" and MUST have 2-4 word/phrase options (never an empty options array).
- true_false: exactly 2 options ("True", "False").
- All distractor options must use words/concepts from the same page topic.
- %s
%s

Return JSON:
{
  "page_keywords": ["key terms copied from the page"],
  "chapter_summary": "optional summary of page only, in page language",
  "questions": [
    {
      "text": "question from page content",
      "type": "mcq|fill_blank|true_false",
      "options": [{"label": "A", "text": "..."}],
      "correct_index": 0,
      "explanation": "explanation in the page language with example from the page",
      "difficulty": "medium",
      "topic": "must match page topic e.g. %s"
    }
  ]
}`, chapter, req.Board, req.difficulty(), chapter, minQ, maxQ, langLine, summaryBlock, chapter)
}

func visionLanguageLine(req GenerateRequest) string {
	switch NormalizeLanguage(req.Language) {
	case "english":
		return "Write ALL questions, options, and explanations in English (the page is English)."
	case "hindi":
		return "Write ALL questions, options, and explanations in Hindi (the page is Hindi)."
	default:
		return "Detect the page language and write ALL questions, options, and explanations in that same language only."
	}
}
