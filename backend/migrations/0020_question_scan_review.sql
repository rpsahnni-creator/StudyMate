-- Question Scan review/publish workflow.
-- Adds a publish lifecycle to quizzes and an answer-known flag to questions so
-- scanned exams can be reviewed and have answers filled in before students take them.

-- Quiz publish lifecycle: existing quizzes stay 'published' (backward compatible);
-- question-scan quizzes start as 'draft' until the owner reviews and publishes.
ALTER TABLE quizzes
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published'
    CHECK (status IN ('draft', 'published'));

-- Whether a question's correct answer is known. Chapter-scan questions are always
-- 'set'. Scanned questions without a printed answer key start as 'unknown' and must
-- be filled in during review before the exam can be published.
ALTER TABLE questions
    ADD COLUMN IF NOT EXISTS answer_status TEXT NOT NULL DEFAULT 'set'
    CHECK (answer_status IN ('set', 'unknown'));

CREATE INDEX IF NOT EXISTS idx_quizzes_status ON quizzes(status);
