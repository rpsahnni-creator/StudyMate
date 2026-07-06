-- Feature flag tables: the on/off switch mechanism for modules
-- (scan_quiz_module, career_goals_module, etc.)

CREATE TABLE feature_flags (
    flag_key            TEXT PRIMARY KEY,
    enabled             BOOLEAN NOT NULL DEFAULT false,
    rollout_percentage  INT NOT NULL DEFAULT 0 CHECK (rollout_percentage BETWEEN 0 AND 100),
    updated_by          TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_feature_overrides (
    id          BIGSERIAL PRIMARY KEY,
    user_id     TEXT NOT NULL,
    flag_key    TEXT NOT NULL REFERENCES feature_flags(flag_key),
    enabled     BOOLEAN NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, flag_key)
);

-- Seed initial flags: scan_quiz is on for everyone (core module, Phase 1),
-- career_goals stays off until Phase 3 build is ready.
INSERT INTO feature_flags (flag_key, enabled, rollout_percentage, updated_by) VALUES
    ('scan_quiz_module', true, 100, 'system_seed'),
    ('career_goals_module', false, 0, 'system_seed');
