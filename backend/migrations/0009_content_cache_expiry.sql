-- Content cache expiry + metadata for production cache layer
ALTER TABLE content_cache
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS question_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS board TEXT,
    ADD COLUMN IF NOT EXISTS subject TEXT,
    ADD COLUMN IF NOT EXISTS chapter TEXT;

UPDATE content_cache
SET expires_at = created_at + INTERVAL '7 days'
WHERE expires_at IS NULL;

ALTER TABLE scan_jobs
    ADD COLUMN IF NOT EXISTS quiz_id BIGINT REFERENCES quizzes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_content_cache_expires_at ON content_cache (expires_at);
