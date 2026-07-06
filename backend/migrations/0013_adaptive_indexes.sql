-- Sprint 9-A — Adaptive practice engine.
--
-- Adds the weekly score history column used by the adaptive scorer and the
-- supporting indexes for weak-topic biased question selection and recently-seen
-- lookups. Some of these indexes were already created in 0012; they are repeated
-- here with IF NOT EXISTS so this migration is safe to run standalone and matches
-- the sprint brief's index list (adapted to the reconciled column names).
--
-- NOTE: the sprint brief called this file 0008_adaptive_indexes.sql, but 0008 is
-- already taken (0008_scan_temp_cleanup.sql). It is numbered 0013 to preserve
-- migration ordering.

-- student_goals: rolling per-week scores appended by UpdateWeeklyGoal.
ALTER TABLE student_goals
    ADD COLUMN IF NOT EXISTS weekly_score_history JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Weak-topic selection reads skill_gaps ordered by weakness, per user.
CREATE INDEX IF NOT EXISTS idx_skill_gaps_user_weakness
    ON skill_gaps(user_id, weakness_score DESC);

-- Loading a set's questions and the recently-seen join both key off (set_type, set_id).
CREATE INDEX IF NOT EXISTS idx_practice_set_questions_set
    ON practice_set_questions(set_type, set_id);

-- GetRecentlySeenIDs filters daily sets by user + recency.
CREATE INDEX IF NOT EXISTS idx_daily_practice_user_created
    ON daily_practice_sets(user_id, created_at DESC);

-- UpdateWeeklyGoal looks up / upserts the current week's set.
CREATE INDEX IF NOT EXISTS idx_weekly_goal_sets_user_week
    ON weekly_goal_sets(user_id, student_goal_id, week_start);
