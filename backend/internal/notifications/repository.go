package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserIDToUUID maps an int64 auth user id to the deterministic UUID used by notifications.
func UserIDToUUID(userID int64) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("user:%d", userID)))
}

// Repository handles database operations for notifications
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new notifications repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateFCMDeviceToken creates a new device token
func (r *Repository) CreateFCMDeviceToken(ctx context.Context, token *FCMDeviceToken) error {
	query := `
		INSERT INTO fcm_device_tokens (
			id, user_id, token, platform, app_version, os_version, 
			push_enabled, last_seen, is_active, failure_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (token) DO UPDATE SET
			is_active = $9,
			last_seen = $8,
			updated_at = $12
	`
	_, err := r.db.Exec(ctx, query,
		token.ID, token.UserID, token.Token, token.Platform, token.AppVersion,
		token.OSVersion, token.PushEnabled, token.LastSeen, token.IsActive,
		token.FailureCount, token.CreatedAt, token.UpdatedAt,
	)
	return err
}

// GetActiveDeviceTokens retrieves all active tokens for a user
func (r *Repository) GetActiveDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT token FROM fcm_device_tokens
		WHERE user_id = $1 AND is_active = true AND last_seen > NOW() - INTERVAL '30 days'
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

// MarkTokenInactive marks a token as inactive
func (r *Repository) MarkTokenInactive(ctx context.Context, token string) error {
	query := `
		UPDATE fcm_device_tokens 
		SET is_active = false, updated_at = NOW()
		WHERE token = $1
	`
	_, err := r.db.Exec(ctx, query, token)
	return err
}

// UpdateTokenLastSeen updates the last_seen timestamp
func (r *Repository) UpdateTokenLastSeen(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE fcm_device_tokens 
		SET last_seen = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, tokenID)
	return err
}

// MarkTokenInactiveByID marks a specific token record as inactive.
func (r *Repository) MarkTokenInactiveByID(ctx context.Context, tokenID uuid.UUID) error {
	query := `
		UPDATE fcm_device_tokens
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, tokenID)
	return err
}

// CleanupStaleTokens marks inactive tokens from 30+ days ago
func (r *Repository) CleanupStaleTokens(ctx context.Context) error {
	query := `
		UPDATE fcm_device_tokens
		SET is_active = false, updated_at = NOW()
		WHERE is_active = true AND last_seen < NOW() - INTERVAL '30 days'
	`
	_, err := r.db.Exec(ctx, query)
	return err
}

// HardDeleteOldTokens deletes tokens inactive for 90+ days
func (r *Repository) HardDeleteOldTokens(ctx context.Context) error {
	query := `
		DELETE FROM fcm_device_tokens
		WHERE is_active = false AND last_seen < NOW() - INTERVAL '90 days'
	`
	_, err := r.db.Exec(ctx, query)
	return err
}

// CreateNotificationJob creates a new notification job
func (r *Repository) CreateNotificationJob(ctx context.Context, job *NotificationJob) error {
	query := `
		INSERT INTO notification_jobs (
			id, user_id, channel, priority, category, template_key, template_data,
			status, idempotency_key, retry_count, max_retries, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		job.ID, job.UserID, job.Channel, job.Priority, job.Category,
		job.TemplateKey, job.TemplateData, job.Status, job.IdempotencyKey,
		job.RetryCount, job.MaxRetries, job.CreatedAt, job.UpdatedAt,
	)
	return err
}

// GetPendingJobs retrieves pending jobs by priority
func (r *Repository) GetPendingJobs(ctx context.Context, priority string, limit int) ([]*NotificationJob, error) {
	query := `
		SELECT id, user_id, channel, priority, category, template_key, template_data,
		       status, idempotency_key, retry_count, max_retries, last_error, 
		       next_retry_at, sent_at, delivered_at, created_at, updated_at
		FROM notification_jobs
		WHERE status IN ('pending', 'failed')
		  AND priority = $1
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, priority, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*NotificationJob
	for rows.Next() {
		job := &NotificationJob{}
		err := rows.Scan(
			&job.ID, &job.UserID, &job.Channel, &job.Priority, &job.Category,
			&job.TemplateKey, &job.TemplateData, &job.Status, &job.IdempotencyKey,
			&job.RetryCount, &job.MaxRetries, &job.LastError, &job.NextRetryAt,
			&job.SentAt, &job.DeliveredAt, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// UpdateJobStatus updates notification job status
func (r *Repository) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	query := `
		UPDATE notification_jobs
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, status, jobID)
	return err
}

// RescheduleJob reschedules a notification job for later
func (r *Repository) RescheduleJob(ctx context.Context, jobID uuid.UUID, delay time.Duration, reason string) error {
	nextRetry := time.Now().Add(delay)
	query := `
		UPDATE notification_jobs
		SET status = $1, next_retry_at = $2, retry_count = retry_count + 1, 
		    last_error = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, StatusPending, nextRetry, reason, jobID)
	return err
}

// FailJob marks a notification job as failed permanently
func (r *Repository) FailJob(ctx context.Context, jobID uuid.UUID, reason string) error {
	query := `
		UPDATE notification_jobs
		SET status = $1, last_error = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, StatusFailed, reason, jobID)
	return err
}

// GetUserPreferences retrieves user notification preferences.
// When no row exists, defaults are returned (not an error).
func (r *Repository) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*NotificationPreferences, error) {
	query := `
		SELECT id, user_id, push_enabled, email_enabled, sms_enabled, preferences,
		       max_push_per_day, max_email_per_week,
		       quiet_hours_start::text, quiet_hours_end::text,
		       quiet_hours_tz, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`
	prefs := &NotificationPreferences{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.PushEnabled, &prefs.EmailEnabled,
		&prefs.SMSEnabled, &prefs.Preferences, &prefs.MaxPushPerDay,
		&prefs.MaxEmailPerWeek, &prefs.QuietHoursStart, &prefs.QuietHoursEnd,
		&prefs.QuietHoursTZ, &prefs.CreatedAt, &prefs.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultNotificationPreferences(userID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	if prefs.Preferences == nil {
		prefs.Preferences = JSONB{}
	}
	return prefs, nil
}

// UpsertUserPreferences creates or updates user notification preferences
func (r *Repository) UpsertUserPreferences(ctx context.Context, prefs *NotificationPreferences) error {
	normalizeNotificationPreferences(prefs)
	query := `
		INSERT INTO notification_preferences (
			id, user_id, push_enabled, email_enabled, sms_enabled, preferences,
			max_push_per_day, max_email_per_week, quiet_hours_start, quiet_hours_end,
			quiet_hours_tz, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (user_id) DO UPDATE SET
			push_enabled = $3, email_enabled = $4, sms_enabled = $5,
			preferences = $6, max_push_per_day = $7, max_email_per_week = $8,
			quiet_hours_start = $9, quiet_hours_end = $10, quiet_hours_tz = $11,
			updated_at = $13
	`
	_, err := r.db.Exec(ctx, query,
		prefs.ID, prefs.UserID, prefs.PushEnabled, prefs.EmailEnabled,
		prefs.SMSEnabled, prefs.Preferences, prefs.MaxPushPerDay,
		prefs.MaxEmailPerWeek, prefs.QuietHoursStart, prefs.QuietHoursEnd,
		prefs.QuietHoursTZ, prefs.CreatedAt, prefs.UpdatedAt,
	)
	return err
}

// CountNotifications counts notifications in time window
func (r *Repository) CountNotifications(ctx context.Context, userID uuid.UUID, channel string, statuses []string, duration time.Duration) (int, error) {
	query := `
		SELECT COUNT(*) FROM notification_jobs
		WHERE user_id = $1 AND channel = $2 
		  AND status = ANY($3)
		  AND created_at > NOW() - $4::INTERVAL
	`
	var count int
	statusArray := "{" + statuses[0]
	for _, s := range statuses[1:] {
		statusArray += "," + s
	}
	statusArray += "}"

	err := r.db.QueryRow(ctx, query, userID, channel, statusArray, fmt.Sprintf("%d seconds", int(duration.Seconds()))).Scan(&count)
	return count, err
}

// JobExistsByIdempotencyKey checks if job exists by idempotency key
func (r *Repository) JobExistsByIdempotencyKey(ctx context.Context, key string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM notification_jobs WHERE idempotency_key = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, key).Scan(&exists)
	return exists, err
}

// LogEmailEvent logs an email delivery event
func (r *Repository) LogEmailEvent(ctx context.Context, event *EmailEvent) error {
	query := `
		INSERT INTO email_events (id, user_id, email_address, event_type, provider_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		event.ID, event.UserID, event.EmailAddress, event.EventType,
		event.ProviderEventID, event.Metadata, event.CreatedAt,
	)
	return err
}

// GetUserEmail returns the email address for an auth user id.
func (r *Repository) GetUserEmail(ctx context.Context, userID int64) (string, error) {
	var email string
	err := r.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("user not found")
	}
	return email, err
}

// GetTemplate retrieves a notification template
func (r *Repository) GetTemplate(ctx context.Context, key string) (*NotificationTemplate, error) {
	query := `
		SELECT id, key, subject, body_html, body_text, body_i18n, variables, created_at, updated_at
		FROM notification_templates
		WHERE key = $1
	`
	tmpl := &NotificationTemplate{}
	err := r.db.QueryRow(ctx, query, key).Scan(
		&tmpl.ID, &tmpl.Key, &tmpl.Subject, &tmpl.BodyHTML, &tmpl.BodyText,
		&tmpl.BodyI18n, &tmpl.Variables, &tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
