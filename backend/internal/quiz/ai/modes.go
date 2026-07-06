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
)
