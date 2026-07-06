-- Extend ai_generation_logs so the quiz AI module can record per-generation
-- provider, timing, question count, status, and errors for the admin dashboard.
ALTER TABLE ai_generation_logs
    ADD COLUMN IF NOT EXISTS scan_job_id     BIGINT,
    ADD COLUMN IF NOT EXISTS provider        TEXT,
    ADD COLUMN IF NOT EXISTS question_count  INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS duration_ms     BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS status          TEXT NOT NULL DEFAULT 'success',
    ADD COLUMN IF NOT EXISTS error_message   TEXT;

CREATE INDEX IF NOT EXISTS idx_ai_generation_logs_scan_job_id ON ai_generation_logs(scan_job_id);
CREATE INDEX IF NOT EXISTS idx_ai_generation_logs_provider ON ai_generation_logs(provider);
