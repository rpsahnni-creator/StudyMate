package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPlanNotFound        = errors.New("plan not found")
	ErrPaymentNotFound     = errors.New("payment not found")
	ErrEventAlreadyHandled = errors.New("payment event already processed")
)

type Repository interface {
	ListActivePlans(ctx context.Context) ([]Plan, error)
	GetPlanBySlug(ctx context.Context, slug string) (Plan, error)
	GetPlanByID(ctx context.Context, planID int64) (Plan, error)
	GetUserProfile(ctx context.Context, userID int64) (UserProfile, error)
	CreatePendingPayment(ctx context.Context, userID, planID int64, provider, orderID string, amountPaise int64, currency string) (int64, error)
	GetPaymentByProviderOrderID(ctx context.Context, provider, orderID string) (PaymentRecord, error)
	MarkPaymentCompleted(ctx context.Context, paymentID int64, providerPaymentID string, paidAt time.Time) error
	MarkPaymentFailed(ctx context.Context, paymentID int64) error
	UpsertActiveSubscription(ctx context.Context, userID, planID int64, provider, providerRef string) error
	CancelSubscription(ctx context.Context, userID int64) error
	GetEntitlements(ctx context.Context, userID int64) (Entitlements, error)
	EventExists(ctx context.Context, providerEventID string) (bool, error)
	InsertPaymentEvent(ctx context.Context, providerEventID, provider, eventType string, payload []byte, paymentID *int64) (int64, error)
	MarkPaymentEventProcessed(ctx context.Context, eventID int64) error
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) ListActivePlans(ctx context.Context) ([]Plan, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, price_paise, currency, billing_interval, features_json, is_popular, scan_limit, active, created_at
		FROM plans
		WHERE active = true AND slug IS NOT NULL
		ORDER BY price_paise ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []Plan
	for rows.Next() {
		var p Plan
		var features []byte
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.PricePaise, &p.Currency, &p.Interval, &features, &p.IsPopular, &p.ScanLimit, &p.Active, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Features = features
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func (r *postgresRepository) GetPlanBySlug(ctx context.Context, slug string) (Plan, error) {
	var p Plan
	var features []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, price_paise, currency, billing_interval, features_json, is_popular, scan_limit, active, created_at
		FROM plans WHERE slug = $1 AND active = true
	`, slug).Scan(&p.ID, &p.Slug, &p.Name, &p.PricePaise, &p.Currency, &p.Interval, &features, &p.IsPopular, &p.ScanLimit, &p.Active, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	p.Features = features
	return p, nil
}

func (r *postgresRepository) GetPlanByID(ctx context.Context, planID int64) (Plan, error) {
	var p Plan
	var features []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, price_paise, currency, billing_interval, features_json, is_popular, scan_limit, active, created_at
		FROM plans WHERE id = $1
	`, planID).Scan(&p.ID, &p.Slug, &p.Name, &p.PricePaise, &p.Currency, &p.Interval, &features, &p.IsPopular, &p.ScanLimit, &p.Active, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return Plan{}, ErrPlanNotFound
	}
	if err != nil {
		return Plan{}, err
	}
	p.Features = features
	return p, nil
}

func (r *postgresRepository) GetUserProfile(ctx context.Context, userID int64) (UserProfile, error) {
	var u UserProfile
	err := r.pool.QueryRow(ctx, `SELECT id, name, email FROM users WHERE id = $1`, userID).
		Scan(&u.ID, &u.Name, &u.Email)
	if err == pgx.ErrNoRows {
		return UserProfile{}, fmt.Errorf("user not found")
	}
	return u, err
}

func (r *postgresRepository) CreatePendingPayment(ctx context.Context, userID, planID int64, provider, orderID string, amountPaise int64, currency string) (int64, error) {
	var id int64
	amountINR := float64(amountPaise) / 100.0
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payments (user_id, plan_id, amount, amount_paise, currency, status, provider, provider_order_id, transaction_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, now())
		RETURNING id
	`, userID, planID, amountINR, amountPaise, currency, PaymentStatusPending, provider, orderID).Scan(&id)
	return id, err
}

func (r *postgresRepository) GetPaymentByProviderOrderID(ctx context.Context, provider, orderID string) (PaymentRecord, error) {
	var p PaymentRecord
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, plan_id, provider, provider_order_id, amount_paise, currency, status
		FROM payments
		WHERE provider = $1 AND provider_order_id = $2
	`, provider, orderID).Scan(&p.ID, &p.UserID, &p.PlanID, &p.Provider, &p.ProviderOrderID, &p.AmountPaise, &p.Currency, &p.Status)
	if err == pgx.ErrNoRows {
		return PaymentRecord{}, ErrPaymentNotFound
	}
	return p, err
}

func (r *postgresRepository) MarkPaymentCompleted(ctx context.Context, paymentID int64, providerPaymentID string, paidAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE payments
		SET status = $1, provider_payment_id = $2, paid_at = $3, transaction_id = COALESCE(transaction_id, $2)
		WHERE id = $4 AND status = $5
	`, PaymentStatusCompleted, providerPaymentID, paidAt, paymentID, PaymentStatusPending)
	return err
}

func (r *postgresRepository) MarkPaymentFailed(ctx context.Context, paymentID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE payments SET status = $1 WHERE id = $2 AND status = $3
	`, PaymentStatusFailed, paymentID, PaymentStatusPending)
	return err
}

func (r *postgresRepository) UpsertActiveSubscription(ctx context.Context, userID, planID int64, provider, providerRef string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, plan_id, status, provider, provider_ref, starts_at, ends_at, created_at)
		VALUES ($1, $2, $3, $4, $5, now(), now() + interval '1 month', now())
		ON CONFLICT (user_id) DO UPDATE SET
			plan_id = EXCLUDED.plan_id,
			status = EXCLUDED.status,
			provider = EXCLUDED.provider,
			provider_ref = EXCLUDED.provider_ref,
			ends_at = CASE
				WHEN subscriptions.status = 'active' AND subscriptions.ends_at > now()
				THEN subscriptions.ends_at + interval '1 month'
				ELSE now() + interval '1 month'
			END,
			cancelled_at = NULL
	`, userID, planID, SubStatusActive, provider, providerRef)
	return err
}

func (r *postgresRepository) CancelSubscription(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE subscriptions SET status = $1, cancelled_at = now()
		WHERE user_id = $2 AND status = $3
	`, SubStatusCancelled, userID, SubStatusActive)
	return err
}

func (r *postgresRepository) GetEntitlements(ctx context.Context, userID int64) (Entitlements, error) {
	var (
		status    string
		planName  string
		scanLimit int
		endsAt    *time.Time
	)
	err := r.pool.QueryRow(ctx, `
		SELECT s.status, p.name, p.scan_limit, s.ends_at
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.user_id = $1 AND s.status = 'active' AND s.ends_at > now()
		ORDER BY s.ends_at DESC
		LIMIT 1
	`, userID).Scan(&status, &planName, &scanLimit, &endsAt)
	if err == pgx.ErrNoRows {
		return Entitlements{
			HasActiveSub: false,
			Plan:         "free",
			ScansPerDay:  FreeScansPerDay,
		}, nil
	}
	if err != nil {
		return Entitlements{}, err
	}

	ent := Entitlements{
		HasActiveSub: true,
		Plan:         normalizePlanName(planName),
		ExpiresAt:    endsAt,
	}
	ent.ScansPerDay = scansPerDayFromLimit(scanLimit, ent.Plan)
	if endsAt != nil {
		ent.DaysRemaining = int(time.Until(*endsAt).Hours() / 24)
		if ent.DaysRemaining < 0 {
			ent.DaysRemaining = 0
		}
	}
	return ent, nil
}

func (r *postgresRepository) EventExists(ctx context.Context, providerEventID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM payment_events WHERE provider_event_id = $1)
	`, providerEventID).Scan(&exists)
	return exists, err
}

func (r *postgresRepository) InsertPaymentEvent(ctx context.Context, providerEventID, provider, eventType string, payload []byte, paymentID *int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payment_events (payment_id, provider_event_id, provider, event_type, event_payload, processed, received_at, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, false, now(), now())
		RETURNING id
	`, paymentID, providerEventID, provider, eventType, string(payload)).Scan(&id)
	return id, err
}

func (r *postgresRepository) MarkPaymentEventProcessed(ctx context.Context, eventID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE payment_events SET processed = true WHERE id = $1`, eventID)
	return err
}

func normalizePlanName(name string) string {
	switch name {
	case "Basic":
		return "basic"
	case "Pro":
		return "pro"
	default:
		return "free"
	}
}

func scansPerDayFromLimit(scanLimit int, plan string) int {
	if plan == "pro" || scanLimit <= 0 {
		return -1
	}
	if scanLimit > 0 {
		return scanLimit
	}
	return FreeScansPerDay
}

func planResponse(p Plan) map[string]any {
	features := []string{}
	_ = json.Unmarshal(p.Features, &features)
	priceINR := p.PricePaise / 100
	if p.PricePaise%100 != 0 {
		// keep fractional rupees if ever needed
	}
	return map[string]any{
		"id":        p.Slug,
		"name":      p.Name,
		"price":     priceINR,
		"currency":  p.Currency,
		"interval":  p.Interval,
		"features":  features,
		"isPopular": p.IsPopular,
	}
}
