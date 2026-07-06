-- Sprint 10-B: audit log metadata for security events

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS ip_address TEXT,
    ADD COLUMN IF NOT EXISTS user_agent TEXT,
    ADD COLUMN IF NOT EXISTS success BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{}'::jsonb;
