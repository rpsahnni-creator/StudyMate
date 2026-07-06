package admin

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListUsers(ctx context.Context, search string, limit, offset int) ([]AdminUser, int, error)
	SetUserStatus(ctx context.Context, userID int64, status string) error
	GetUserRole(ctx context.Context, userID int64) (string, error)
	ListJobs(ctx context.Context, status string, limit, offset int) ([]AdminJob, int, error)
	RetryJob(ctx context.Context, jobID int64) (bool, error)
	AICosts(ctx context.Context, from, to time.Time) ([]AICostRow, error)
	ListAuditLogs(ctx context.Context, actorUserID int64, limit, offset int) ([]AuditLogEntry, int, error)
	ListContentFlags(ctx context.Context, status string, limit, offset int) ([]ContentFlag, int, error)
	ResolveContentFlag(ctx context.Context, flagID int64, status, reason string, adminID int64) (string, error)
	DeleteContentCache(ctx context.Context, contentHash string) error
	LogAdminAction(ctx context.Context, adminID int64, actionType, targetID, notes string) error
	LogAudit(ctx context.Context, actorUserID int64, action, entityType, entityID string) error
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) ListUsers(ctx context.Context, search string, limit, offset int) ([]AdminUser, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE ($1 = '' OR email ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%')
	`, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.name, u.email, u.role, u.status, u.created_at,
		       p.name AS plan_name, s.status AS sub_status, s.ends_at
		FROM users u
		LEFT JOIN subscriptions s ON s.user_id = u.id AND s.status = 'active' AND s.ends_at > now()
		LEFT JOIN plans p ON p.id = s.plan_id
		WHERE ($1 = '' OR u.email ILIKE '%' || $1 || '%' OR u.name ILIKE '%' || $1 || '%')
		ORDER BY u.created_at DESC
		LIMIT $2 OFFSET $3
	`, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		var planName, subStatus *string
		var expiresAt *time.Time
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Status, &u.CreatedAt,
			&planName, &subStatus, &expiresAt); err != nil {
			return nil, 0, err
		}
		u.IsSuspended = u.Status == "suspended"
		u.Plan = "free"
		if planName != nil {
			u.Plan = normalizePlan(*planName)
		}
		if subStatus != nil {
			u.SubStatus = *subStatus
		}
		u.ExpiresAt = expiresAt
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *postgresRepository) SetUserStatus(ctx context.Context, userID int64, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET status = $1, updated_at = now() WHERE id = $2
	`, status, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *postgresRepository) GetUserRole(ctx context.Context, userID int64) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
	return role, err
}

func (r *postgresRepository) ListJobs(ctx context.Context, status string, limit, offset int) ([]AdminJob, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM scan_jobs WHERE ($1 = '' OR status = $1)
	`, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT j.id, u.email, j.status, j.progress, j.error_message, j.created_at, j.updated_at,
		       (SELECT provider FROM ai_generation_logs
		        WHERE scan_job_id = j.id ORDER BY created_at DESC LIMIT 1) AS provider
		FROM scan_jobs j
		JOIN users u ON u.id = j.user_id
		WHERE ($1 = '' OR j.status = $1)
		ORDER BY j.created_at DESC
		LIMIT $2 OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []AdminJob
	for rows.Next() {
		var j AdminJob
		var errMsg, provider *string
		if err := rows.Scan(&j.ID, &j.UserEmail, &j.Status, &j.Progress, &errMsg,
			&j.CreatedAt, &j.UpdatedAt, &provider); err != nil {
			return nil, 0, err
		}
		if errMsg != nil {
			j.ErrorMessage = *errMsg
		}
		if provider != nil {
			j.Provider = *provider
		}
		j.DurationMs = j.UpdatedAt.Sub(j.CreatedAt).Milliseconds()
		jobs = append(jobs, j)
	}
	return jobs, total, rows.Err()
}

func (r *postgresRepository) RetryJob(ctx context.Context, jobID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs
		SET status = 'pending', progress = 0, error_message = NULL, updated_at = now()
		WHERE id = $1 AND status IN ('failed', 'processing')
	`, jobID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *postgresRepository) AICosts(ctx context.Context, from, to time.Time) ([]AICostRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date_trunc('day', created_at)::date AS day,
		       COALESCE(provider, 'unknown') AS provider,
		       COALESCE(model_name, 'unknown') AS model,
		       COUNT(*)::int AS request_count,
		       COALESCE(SUM(token_usage), 0)::bigint AS total_tokens,
		       COALESCE(SUM(cost_estimate), 0)::float8 AS total_cost
		FROM ai_generation_logs
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY 1, 2, 3
		ORDER BY 1 DESC, 2 ASC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AICostRow
	for rows.Next() {
		var row AICostRow
		var day time.Time
		if err := rows.Scan(&day, &row.Provider, &row.Model, &row.RequestCount, &row.TotalTokens, &row.TotalCost); err != nil {
			return nil, err
		}
		row.Day = day.Format("2006-01-02")
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *postgresRepository) ListAuditLogs(ctx context.Context, actorUserID int64, limit, offset int) ([]AuditLogEntry, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs WHERE ($1 = 0 OR actor_user_id = $1)
	`, actorUserID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.actor_user_id, u.email, a.action, a.entity_type, a.entity_id, a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE ($1 = 0 OR a.actor_user_id = $1)
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`, actorUserID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var email, entityID *string
		if err := rows.Scan(&e.ID, &e.ActorID, &email, &e.Action, &e.EntityType, &entityID, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		if email != nil {
			e.ActorEmail = *email
		}
		if entityID != nil {
			e.EntityID = *entityID
		}
		logs = append(logs, e)
	}
	return logs, total, rows.Err()
}

func (r *postgresRepository) ListContentFlags(ctx context.Context, status string, limit, offset int) ([]ContentFlag, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM content_flags WHERE ($1 = '' OR status = $1)
	`, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.question_id, q.content_hash, f.reason, ru.email,
		       f.status, f.resolution_reason, f.resolved_at, f.created_at
		FROM content_flags f
		LEFT JOIN questions q ON q.id = f.question_id
		LEFT JOIN users ru ON ru.id = f.reported_by
		WHERE ($1 = '' OR f.status = $1)
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var flags []ContentFlag
	for rows.Next() {
		var f ContentFlag
		var contentHash, email, resolutionReason *string
		var resolvedAt *time.Time
		if err := rows.Scan(&f.ID, &f.QuestionID, &contentHash, &f.Reason, &email,
			&f.Status, &resolutionReason, &resolvedAt, &f.CreatedAt); err != nil {
			return nil, 0, err
		}
		if contentHash != nil {
			f.ContentHash = *contentHash
		}
		if email != nil {
			f.ReportedByEmail = *email
		}
		if resolutionReason != nil {
			f.ResolutionReason = *resolutionReason
		}
		f.ResolvedAt = resolvedAt
		flags = append(flags, f)
	}
	return flags, total, rows.Err()
}

// ResolveContentFlag updates the flag and returns the associated content hash (for cache invalidation).
func (r *postgresRepository) ResolveContentFlag(ctx context.Context, flagID int64, status, reason string, adminID int64) (string, error) {
	var contentHash *string
	err := r.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE content_flags
			SET status = $2, resolution_reason = $3, resolved_by = $4, resolved_at = now()
			WHERE id = $1
			RETURNING question_id
		)
		SELECT q.content_hash
		FROM updated
		LEFT JOIN questions q ON q.id = updated.question_id
	`, flagID, status, reason, adminID).Scan(&contentHash)
	if err != nil {
		return "", err
	}
	if contentHash != nil {
		return *contentHash, nil
	}
	return "", nil
}

func (r *postgresRepository) DeleteContentCache(ctx context.Context, contentHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM content_cache WHERE content_hash = $1`, contentHash)
	return err
}

func (r *postgresRepository) LogAdminAction(ctx context.Context, adminID int64, actionType, targetID, notes string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO admin_actions (admin_user_id, action_type, target_id, notes, created_at)
		VALUES ($1, $2, $3, $4, now())
	`, adminID, actionType, targetID, notes)
	return err
}

func (r *postgresRepository) LogAudit(ctx context.Context, actorUserID int64, action, entityType, entityID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, action, entity_type, entity_id, created_at)
		VALUES ($1, $2, $3, $4, now())
	`, actorUserID, action, entityType, entityID)
	return err
}

func normalizePlan(planName string) string {
	switch planName {
	case "Pro":
		return "pro"
	case "Basic":
		return "basic"
	default:
		return "free"
	}
}
