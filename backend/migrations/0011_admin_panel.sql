-- Sprint 6-A: admin panel support

-- Fix pre-existing gap: auth repository writes users.updated_at but no column exists.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Extend content_flags for moderation workflow.
ALTER TABLE content_flags
    ADD COLUMN IF NOT EXISTS reported_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS resolved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS resolution_reason TEXT;

-- Indexes for admin list/filter queries.
CREATE INDEX IF NOT EXISTS idx_scan_jobs_status ON scan_jobs (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_content_flags_status ON content_flags (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_generation_logs_created_at_provider ON ai_generation_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_actions_admin_user_id ON admin_actions (admin_user_id, created_at DESC);
