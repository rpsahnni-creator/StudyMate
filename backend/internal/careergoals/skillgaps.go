package careergoals

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// skillKeySep separates subject from topic in the topicScores map keys used by
// UpdateSkillGaps. Chapter/subject names never contain this control character,
// so it is a safe delimiter while still keeping the required map[string]float64
// signature.
const skillKeySep = "\x1f"

// DB is the subset of pgx used by UpdateSkillGaps. Both *pgxpool.Pool and pgx.Tx
// satisfy it, so skill gaps can be updated inside or outside a transaction.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// GetSkillGaps returns the user's skill gaps, weakest topics first.
//
// GET /goals/my/skills
func (h *Handler) GetSkillGaps(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(r)
	if !ok {
		h.jsonError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT subject_name, topic_name, weakness_score, updated_at
		FROM skill_gaps
		WHERE user_id = $1
		ORDER BY weakness_score DESC, topic_name ASC
	`, userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	defer rows.Close()

	gaps := []SkillGap{}
	for rows.Next() {
		var (
			g       SkillGap
			updated time.Time
		)
		if err := rows.Scan(&g.Subject, &g.Topic, &g.WeaknessScore, &updated); err != nil {
			h.handleServiceError(w, r, err)
			return
		}
		s := updated.UTC().Format(time.RFC3339)
		g.LastPracticed = &s
		gaps = append(gaps, g)
	}
	if err := rows.Err(); err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, gaps)
}

// UpdateSkillGaps upserts the user's per-topic weakness scores after a practice
// submission. Keys of topicScores are built with skillGapKey(subject, topic);
// values are the topic's score for the just-submitted set (0-100).
//
// Weakness moves adaptively:
//   - score < 60  -> weakness_score += 10 (capped at 100)  [needs more work]
//   - score > 80  -> weakness_score -= 15 (floored at 0)    [improving]
//   - otherwise    -> unchanged
func UpdateSkillGaps(ctx context.Context, db DB, userID string, topicScores map[string]float64) error {
	uid, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}

	for key, score := range topicScores {
		subject, topic := splitSkillGapKey(key)
		if topic == "" {
			continue
		}

		// Seed value for a brand-new topic row (the CASE below only runs on
		// conflict, i.e. for existing rows).
		initial := 0.0
		if score < 60 {
			initial = 10
		}

		_, err := db.Exec(ctx, `
			INSERT INTO skill_gaps (user_id, subject_name, topic_name, weakness_score, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (user_id, subject_name, topic_name) DO UPDATE SET
				weakness_score = CASE
					WHEN $5 < 60 THEN LEAST(skill_gaps.weakness_score + 10, 100)
					WHEN $5 > 80 THEN GREATEST(skill_gaps.weakness_score - 15, 0)
					ELSE skill_gaps.weakness_score
				END,
				updated_at = now()
		`, uid, subject, topic, initial, score)
		if err != nil {
			return err
		}
	}
	return nil
}

// skillGapKey encodes a (subject, topic) pair into a topicScores map key.
func skillGapKey(subject, topic string) string {
	return subject + skillKeySep + topic
}

// splitSkillGapKey reverses skillGapKey.
func splitSkillGapKey(key string) (subject, topic string) {
	parts := strings.SplitN(key, skillKeySep, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}
