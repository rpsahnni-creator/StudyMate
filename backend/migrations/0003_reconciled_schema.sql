-- Reconciled production schema for the StudyApp roadmap.
-- This migration creates the core tables for identity, content, scan pipeline,
-- quizzes, attempts, career goals, billing, and admin/audit tracking.

-- Identity & access
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    email           TEXT NOT NULL UNIQUE,
    phone           TEXT,
    password_hash   TEXT,
    role            TEXT NOT NULL DEFAULT 'student',
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_profiles (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    class_level     TEXT,
    board           TEXT,
    language        TEXT,
    school_name     TEXT,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id)
);

ALTER TABLE feature_flags ADD COLUMN IF NOT EXISTS id BIGSERIAL;
ALTER TABLE feature_flags ADD CONSTRAINT feature_flags_id_unique UNIQUE (id);

ALTER TABLE user_feature_overrides ADD COLUMN IF NOT EXISTS id BIGSERIAL;
ALTER TABLE user_feature_overrides ADD CONSTRAINT user_feature_overrides_id_unique UNIQUE (id);

-- Content (stable, rarely changes)
CREATE TABLE IF NOT EXISTS books (
    id              BIGSERIAL PRIMARY KEY,
    title           TEXT NOT NULL,
    subject         TEXT NOT NULL,
    grade           TEXT,
    board           TEXT,
    language        TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chapters (
    id              BIGSERIAL PRIMARY KEY,
    book_id         BIGINT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    chapter_no      INT NOT NULL,
    title           TEXT NOT NULL,
    summary         TEXT,
    order_no        INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (book_id, chapter_no)
);

CREATE TABLE IF NOT EXISTS ai_generation_logs (
    id              BIGSERIAL PRIMARY KEY,
    flag_key        TEXT REFERENCES feature_flags(flag_key),
    content_hash    TEXT,
    model_name      TEXT,
    cache_hit       BOOLEAN NOT NULL DEFAULT false,
    cost_estimate   NUMERIC(10, 4) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE ai_generation_logs
    ADD COLUMN IF NOT EXISTS prompt_version TEXT,
    ADD COLUMN IF NOT EXISTS token_usage INT;

-- Scan pipeline
CREATE TABLE IF NOT EXISTS scan_jobs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id         BIGINT REFERENCES books(id) ON DELETE SET NULL,
    chapter_id      BIGINT REFERENCES chapters(id) ON DELETE SET NULL,
    mode            TEXT NOT NULL DEFAULT 'scan',
    status          TEXT NOT NULL DEFAULT 'queued',
    progress        INT NOT NULL DEFAULT 0,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scan_pages (
    id              BIGSERIAL PRIMARY KEY,
    scan_job_id     BIGINT NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    page_no         INT NOT NULL,
    image_url       TEXT,
    page_type       TEXT,
    ocr_confidence  NUMERIC(5, 2),
    content_hash    TEXT,
    processed       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scan_job_id, page_no)
);

-- Questions & quizzes
CREATE TABLE IF NOT EXISTS questions (
    id                  BIGSERIAL PRIMARY KEY,
    chapter_id          BIGINT REFERENCES chapters(id) ON DELETE CASCADE,
    content_hash        TEXT,
    question_type       TEXT NOT NULL,
    question_text       TEXT NOT NULL,
    difficulty          TEXT,
    source_type         TEXT NOT NULL DEFAULT 'ai_generated' CHECK (source_type IN ('ai_generated', 'scanned_existing')),
    status              TEXT NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS question_options (
    id              BIGSERIAL PRIMARY KEY,
    question_id     BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    option_label    TEXT NOT NULL,
    option_text     TEXT NOT NULL,
    is_correct      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS question_explanations (
    id                  BIGSERIAL PRIMARY KEY,
    question_id         BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    style               TEXT,
    explanation_text    TEXT NOT NULL,
    language            TEXT NOT NULL DEFAULT 'en',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS quizzes (
    id                  BIGSERIAL PRIMARY KEY,
    chapter_id          BIGINT REFERENCES chapters(id) ON DELETE CASCADE,
    content_hash        TEXT,
    title               TEXT NOT NULL,
    total_questions     INT NOT NULL DEFAULT 0,
    generation_type     TEXT NOT NULL DEFAULT 'ai',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS content_cache (
    id                  BIGSERIAL PRIMARY KEY,
    book_id             BIGINT REFERENCES books(id) ON DELETE SET NULL,
    chapter_id          BIGINT REFERENCES chapters(id) ON DELETE SET NULL,
    page_no             INT NOT NULL,
    content_hash        TEXT NOT NULL,
    page_type           TEXT,
    generated_quiz_id  BIGINT REFERENCES quizzes(id) ON DELETE SET NULL,
    ai_generation_id    BIGINT REFERENCES ai_generation_logs(id) ON DELETE SET NULL,
    hit_count           INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (content_hash)
);

CREATE TABLE IF NOT EXISTS quiz_questions (
    id              BIGSERIAL PRIMARY KEY,
    quiz_id         BIGINT NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    question_id     BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    order_no        INT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (quiz_id, order_no)
);

-- Attempts (immutable history)
CREATE TABLE IF NOT EXISTS quiz_attempts (
    id                  BIGSERIAL PRIMARY KEY,
    quiz_id             BIGINT NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    submitted_at        TIMESTAMPTZ,
    score               NUMERIC(5, 2),
    correct_count       INT NOT NULL DEFAULT 0,
    wrong_count         INT NOT NULL DEFAULT 0,
    skipped_count       INT NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'in_progress'
);

CREATE TABLE IF NOT EXISTS quiz_attempt_answers (
    id                  BIGSERIAL PRIMARY KEY,
    attempt_id          BIGINT NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id         BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    selected_option_id  BIGINT REFERENCES question_options(id) ON DELETE SET NULL,
    is_correct          BOOLEAN,
    answered_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS quiz_reports (
    id                  BIGSERIAL PRIMARY KEY,
    attempt_id          BIGINT NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    accuracy            NUMERIC(5, 2),
    weak_topics_json    JSONB,
    summary_text        TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (attempt_id)
);

-- Career goals module (Phase 3)
CREATE TABLE IF NOT EXISTS career_goals (
    id              BIGSERIAL PRIMARY KEY,
    slug            TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    description     TEXT,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS student_goals (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    career_goal_id      BIGINT NOT NULL REFERENCES career_goals(id) ON DELETE CASCADE,
    daily_time_minutes  INT NOT NULL DEFAULT 0,
    skill_level         TEXT,
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, career_goal_id)
);

CREATE TABLE IF NOT EXISTS daily_practice_sets (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_goal_id     BIGINT NOT NULL REFERENCES student_goals(id) ON DELETE CASCADE,
    set_date            DATE NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, student_goal_id, set_date)
);

CREATE TABLE IF NOT EXISTS weekly_goal_sets (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_goal_id     BIGINT NOT NULL REFERENCES student_goals(id) ON DELETE CASCADE,
    week_start          DATE NOT NULL,
    week_end            DATE NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, student_goal_id, week_start)
);

CREATE TABLE IF NOT EXISTS practice_set_questions (
    id              BIGSERIAL PRIMARY KEY,
    set_type        TEXT NOT NULL CHECK (set_type IN ('daily', 'weekly')),
    set_id          BIGINT NOT NULL,
    question_id     BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    display_order   INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skill_gaps (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_name        TEXT NOT NULL,
    topic_name          TEXT NOT NULL,
    weakness_score      NUMERIC(5, 2) NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, subject_name, topic_name)
);

-- Billing
CREATE TABLE IF NOT EXISTS plans (
    id                  BIGSERIAL PRIMARY KEY,
    name                TEXT NOT NULL,
    price_monthly       NUMERIC(10, 2) NOT NULL,
    scan_limit          INT NOT NULL DEFAULT 0,
    features_json       JSONB,
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id             BIGINT NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status              TEXT NOT NULL DEFAULT 'inactive',
    provider            TEXT NOT NULL DEFAULT 'razorpay' CHECK (provider IN ('razorpay', 'payu')),
    provider_ref        TEXT,
    starts_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS payments (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id     BIGINT REFERENCES subscriptions(id) ON DELETE SET NULL,
    amount              NUMERIC(10, 2) NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    provider            TEXT NOT NULL DEFAULT 'razorpay' CHECK (provider IN ('razorpay', 'payu')),
    transaction_id      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS payment_events (
    id                  BIGSERIAL PRIMARY KEY,
    payment_id          BIGINT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    event_type          TEXT NOT NULL,
    event_payload       JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Processing logs & admin
CREATE TABLE IF NOT EXISTS job_events (
    id                  BIGSERIAL PRIMARY KEY,
    job_id              TEXT NOT NULL,
    event_type          TEXT NOT NULL,
    message             TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_actions (
    id                  BIGSERIAL PRIMARY KEY,
    admin_user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_type         TEXT NOT NULL,
    target_id           TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id                  BIGSERIAL PRIMARY KEY,
    actor_user_id       BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action              TEXT NOT NULL,
    entity_type         TEXT NOT NULL,
    entity_id           TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS content_flags (
    id                  BIGSERIAL PRIMARY KEY,
    question_id         BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    reason              TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common access patterns
CREATE INDEX IF NOT EXISTS idx_chapters_book_id ON chapters(book_id);
CREATE INDEX IF NOT EXISTS idx_content_cache_content_hash ON content_cache(content_hash);
CREATE INDEX IF NOT EXISTS idx_scan_jobs_user_id ON scan_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_scan_pages_job_id ON scan_pages(scan_job_id);
CREATE INDEX IF NOT EXISTS idx_questions_chapter_id ON questions(chapter_id);
CREATE INDEX IF NOT EXISTS idx_quizzes_chapter_id ON quizzes(chapter_id);
CREATE INDEX IF NOT EXISTS idx_quiz_attempts_user_id ON quiz_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_quiz_attempts_quiz_id ON quiz_attempts(quiz_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_user_id ON audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_content_flags_question_id ON content_flags(question_id);
