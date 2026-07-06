-- Minimal version of the ai_generation_logs table (full version comes with
-- the quiz/AI module later). Exists here already so the admin monitoring
-- dashboard has real cost data to show as soon as the AI module starts
-- writing to it — no migration needed later to "unlock" the dashboard.

CREATE TABLE ai_generation_logs (
    id              BIGSERIAL PRIMARY KEY,
    flag_key        TEXT REFERENCES feature_flags(flag_key),
    content_hash    TEXT,
    model_name      TEXT,
    cache_hit       BOOLEAN NOT NULL DEFAULT false,
    cost_estimate   NUMERIC(10, 4) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_generation_logs_flag_key ON ai_generation_logs(flag_key);
CREATE INDEX idx_ai_generation_logs_created_at ON ai_generation_logs(created_at);
