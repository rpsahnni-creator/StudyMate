package ai

const (
	ScanModeChapter           = "chapter"
	ScanModeExistingQuestions = "existing_questions"

	PageTypeChapterText       = "chapter_text"
	PageTypeExistingQuestions = "existing_questions"
	PageTypeMixed             = "mixed"

	StrategyExtractQuestions   = "extract_questions"
	StrategyGenerateFromChapter = "generate_from_chapter"

	QuestionTypeMCQ       = "mcq"
	QuestionTypeFillBlank = "fill_blank"
	QuestionTypeTrueFalse = "true_false"

	SourceAIGenerated     = "ai_generated"
	SourceScannedExisting = "scanned_existing"

	// CorrectIndexUnknown marks an extracted question whose answer was not printed
	// on the page. The reviewer must fill it in before the exam is published.
	CorrectIndexUnknown = -1

	// Answer-known state on the questions table.
	AnswerStatusSet     = "set"
	AnswerStatusUnknown = "unknown"

	// Quiz publish lifecycle.
	QuizStatusDraft     = "draft"
	QuizStatusPublished = "published"
)
