package billing

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Service coordinates billing use-cases.
type Service struct {
	repo     Repository
	cfg      Config
	limiter  *Limiter
	notifier PaymentNotifier
}

// PaymentNotifier sends billing-related notifications.
type PaymentNotifier interface {
	PaymentCaptured(ctx context.Context, userID int64, planName, expiresAt string) error
	PaymentFailed(ctx context.Context, userID int64) error
}

func NewService(repo Repository, pool *pgxpool.Pool, cache *redis.Client, cfg Config) *Service {
	_ = pool
	return &Service{
		repo:    repo,
		cfg:     cfg,
		limiter: NewLimiter(repo, cache),
	}
}

// SetPaymentNotifier wires live notification delivery for payment events.
func (s *Service) SetPaymentNotifier(n PaymentNotifier) {
	s.notifier = n
}

func (s *Service) Limiter() *Limiter {
	return s.limiter
}

func (s *Service) HealthCheck() string {
	return "billing module ready"
}

// CheckScanLimit delegates to the Redis-backed limiter.
func (s *Service) CheckScanLimit(ctx context.Context, userID int64) error {
	if s.limiter == nil {
		return nil
	}
	return s.limiter.CheckScanLimit(ctx, userID)
}

// RecordScan increments the user's daily scan usage.
func (s *Service) RecordScan(ctx context.Context, userID int64) error {
	if s.limiter == nil {
		return nil
	}
	return s.limiter.RecordScan(ctx, userID)
}
