-- Sprint 8 — Career Goals module (behind career_goals_module flag).
--
-- The reconciled schema (0003) created the career goals tables with a minimal
-- shape. This migration extends them with the fields the module's API needs
-- (exam metadata, per-student target date + lifecycle status, and per-set
-- scoring) and seeds the initial catalog of career goals.
--
-- NOTE: the sprint brief called this file 0007_seed_career_goals.sql, but 0007
-- is already taken (0007_ai_generation_logs_quiz.sql). It is numbered 0012 to
-- preserve migration ordering.

-- --- Schema extensions -------------------------------------------------------

-- career_goals: discovery metadata surfaced to students choosing a goal.
ALTER TABLE career_goals ADD COLUMN IF NOT EXISTS exam_name     TEXT;
ALTER TABLE career_goals ADD COLUMN IF NOT EXISTS target_months INT NOT NULL DEFAULT 12;
ALTER TABLE career_goals ADD COLUMN IF NOT EXISTS subject_areas JSONB NOT NULL DEFAULT '[]'::jsonb;

-- student_goals: exam date + lifecycle status ('active' | 'abandoned' | 'completed').
ALTER TABLE student_goals ADD COLUMN IF NOT EXISTS target_date DATE;
ALTER TABLE student_goals ADD COLUMN IF NOT EXISTS status      TEXT NOT NULL DEFAULT 'active';

-- Derive status from the legacy `active` boolean for any pre-existing rows.
UPDATE student_goals SET status = CASE WHEN active THEN 'active' ELSE 'abandoned' END;

-- daily_practice_sets: score + the topics the set focused on.
ALTER TABLE daily_practice_sets ADD COLUMN IF NOT EXISTS score       NUMERIC(5, 2);
ALTER TABLE daily_practice_sets ADD COLUMN IF NOT EXISTS topic_focus JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_student_goals_user_status ON student_goals(user_id, status);
CREATE INDEX IF NOT EXISTS idx_daily_practice_sets_user_date ON daily_practice_sets(user_id, set_date);
CREATE INDEX IF NOT EXISTS idx_practice_set_questions_set ON practice_set_questions(set_type, set_id);
CREATE INDEX IF NOT EXISTS idx_skill_gaps_user_weakness ON skill_gaps(user_id, weakness_score DESC);

-- --- Seed data ---------------------------------------------------------------
-- Slugs act as the stable public identifiers; ids remain BIGSERIAL.

INSERT INTO career_goals (slug, title, description, exam_name, target_months, subject_areas, active)
VALUES
    ('goal_jee_main', 'JEE Main Preparation',
     'Prepare for Joint Entrance Examination Main',
     'JEE Main', 12,
     '["Physics","Chemistry","Mathematics"]'::jsonb, true),
    ('goal_neet', 'NEET Preparation',
     'Prepare for National Eligibility cum Entrance Test',
     'NEET', 12,
     '["Physics","Chemistry","Biology"]'::jsonb, true),
    ('goal_class10_board', 'Class 10 Board Preparation',
     'Prepare for Class 10 CBSE Board Examination',
     'CBSE Class 10', 6,
     '["Mathematics","Science","Social Science","English","Hindi"]'::jsonb, true)
ON CONFLICT (slug) DO NOTHING;
