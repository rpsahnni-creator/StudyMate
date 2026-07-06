-- Sprint 9-B — Performance analytics.
--
-- Indexes supporting the /users/me/analytics aggregations. The reconciled schema
-- stores attempt completion time in quiz_attempts.submitted_at (there is no
-- completed_at column), so the partial index is built on submitted_at.
--
-- NOTE: the sprint brief called this file 0009_analytics_indexes.sql, but 0009 is
-- already taken (0009_content_cache_expiry.sql). It is numbered 0014 to preserve
-- migration ordering.

-- Summary / weekly / subject aggregations scan a user's completed attempts.
CREATE INDEX IF NOT EXISTS idx_quiz_attempts_user_completed
    ON quiz_attempts(user_id, submitted_at DESC)
    WHERE status = 'completed';

-- Topic breakdown joins answers back to their attempt.
CREATE INDEX IF NOT EXISTS idx_quiz_attempt_answers_attempt
    ON quiz_attempt_answers(attempt_id);
