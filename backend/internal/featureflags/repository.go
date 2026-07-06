package featureflags

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetAllFlags(ctx context.Context) ([]Flag, error)
	GetUserOverrides(ctx context.Context, userID string) ([]UserOverride, error)
	SetFlag(ctx context.Context, key FlagKey, enabled bool, rollout int, updatedBy string) error
	GetAllFlagsWithStats(ctx context.Context) ([]FlagWithStats, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) GetAllFlags(ctx context.Context) ([]Flag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT flag_key, enabled, rollout_percentage, updated_by, updated_at
		FROM feature_flags`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.Key, &f.Enabled, &f.RolloutPercentage, &f.UpdatedBy, &f.UpdatedAt); err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (r *pgRepository) GetUserOverrides(ctx context.Context, userID string) ([]UserOverride, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, flag_key, enabled
		FROM user_feature_overrides
		WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overrides []UserOverride
	for rows.Next() {
		var o UserOverride
		if err := rows.Scan(&o.UserID, &o.Key, &o.Enabled); err != nil {
			return nil, err
		}
		overrides = append(overrides, o)
	}
	return overrides, rows.Err()
}

func (r *pgRepository) GetAllFlagsWithStats(ctx context.Context) ([]FlagWithStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			f.flag_key, f.enabled, f.rollout_percentage, f.updated_by, f.updated_at,
			COALESCE(o.override_count, 0),
			COALESCE(l.ai_call_count_24h, 0),
			COALESCE(l.ai_cache_hit_rate_24h, 0),
			COALESCE(l.ai_estimated_cost_24h, 0)
		FROM feature_flags f
		LEFT JOIN (
			SELECT flag_key, COUNT(*) AS override_count
			FROM user_feature_overrides
			GROUP BY flag_key
		) o ON o.flag_key = f.flag_key
		LEFT JOIN (
			SELECT
				flag_key,
				COUNT(*) AS ai_call_count_24h,
				AVG(CASE WHEN cache_hit THEN 1.0 ELSE 0.0 END) AS ai_cache_hit_rate_24h,
				SUM(cost_estimate) AS ai_estimated_cost_24h
			FROM ai_generation_logs
			WHERE created_at > now() - interval '24 hours'
			GROUP BY flag_key
		) l ON l.flag_key = f.flag_key
		ORDER BY f.flag_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []FlagWithStats
	for rows.Next() {
		var s FlagWithStats
		if err := rows.Scan(
			&s.Key, &s.Enabled, &s.RolloutPercentage, &s.UpdatedBy, &s.UpdatedAt,
			&s.OverrideCount, &s.AICallCount24h, &s.AICacheHitRate24h, &s.AIEstimatedCost24h,
		); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *pgRepository) SetFlag(ctx context.Context, key FlagKey, enabled bool, rollout int, updatedBy string) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO feature_flags (flag_key, enabled, rollout_percentage, updated_by, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (flag_key)
		DO UPDATE SET enabled = $2, rollout_percentage = $3, updated_by = $4, updated_at = now()`,
		key, enabled, rollout, updatedBy)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SQL executed for key=%q -> rowsAffected=%d", key, tag.RowsAffected())
	return nil
}