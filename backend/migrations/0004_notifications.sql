-- Migration: Create notification tables
-- Version: 0004_notifications
-- PostgreSQL-compatible (indexes created separately; user_id uses deterministic UUID, not FK to users.id BIGINT)

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- FCM Device Tokens Table
CREATE TABLE IF NOT EXISTS fcm_device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token TEXT NOT NULL UNIQUE,
    platform VARCHAR(10) NOT NULL,
    app_version VARCHAR(10),
    os_version VARCHAR(10),
    push_enabled BOOLEAN DEFAULT true,
    last_seen TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    failure_count INT DEFAULT 0,
    replaced_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fcm_device_tokens_user_active ON fcm_device_tokens (user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_fcm_device_tokens_last_seen ON fcm_device_tokens (last_seen);

-- Notification Jobs Table
CREATE TABLE IF NOT EXISTS notification_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    priority VARCHAR(10) DEFAULT 'normal',
    category VARCHAR(50) NOT NULL,
    template_key VARCHAR(100) NOT NULL,
    template_data JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'pending',
    idempotency_key VARCHAR(255) UNIQUE,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 5,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_jobs_status_priority ON notification_jobs (status, priority);
CREATE INDEX IF NOT EXISTS idx_notification_jobs_user_channel ON notification_jobs (user_id, channel);
CREATE INDEX IF NOT EXISTS idx_notification_jobs_next_retry
    ON notification_jobs (next_retry_at)
    WHERE status IN ('pending', 'failed');

-- Notification Preferences Table
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    push_enabled BOOLEAN DEFAULT true,
    email_enabled BOOLEAN DEFAULT true,
    sms_enabled BOOLEAN DEFAULT false,
    preferences JSONB DEFAULT '{}',
    max_push_per_day INT DEFAULT 10,
    max_email_per_week INT DEFAULT 5,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    quiet_hours_tz VARCHAR(50) DEFAULT 'Asia/Kolkata',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Email Events Table
CREATE TABLE IF NOT EXISTS email_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    email_address TEXT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    provider_event_id TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_events_user_event ON email_events (user_id, event_type);
CREATE INDEX IF NOT EXISTS idx_email_events_email_event ON email_events (email_address, event_type);

-- Notification Templates Table
CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "key" VARCHAR(100) NOT NULL UNIQUE,
    subject TEXT NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,
    body_i18n JSONB DEFAULT '{}',
    variables JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_templates_key ON notification_templates ("key");

-- Notification Delivery Log Table
CREATE TABLE IF NOT EXISTS notification_delivery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_job_id UUID NOT NULL REFERENCES notification_jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    provider_id TEXT,
    status VARCHAR(20) NOT NULL,
    status_timestamp TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_delivery_logs_job ON notification_delivery_logs (notification_job_id);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_logs_user ON notification_delivery_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_logs_status ON notification_delivery_logs (status);

-- Extend users table for notification/email status
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_status VARCHAR(50) DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS language_preference VARCHAR(10) DEFAULT 'en';
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_bounce_count INT DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_email_status ON users (email_status);
CREATE INDEX IF NOT EXISTS idx_users_language_preference ON users (language_preference);
