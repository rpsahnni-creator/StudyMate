package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrScanLimitExceeded = errors.New("daily scan limit exceeded")

// Limiter enforces per-day scan quotas using Valkey INCR.
type Limiter struct {
	repo  Repository
	cache *redis.Client
}

func NewLimiter(repo Repository, cache *redis.Client) *Limiter {
	return &Limiter{repo: repo, cache: cache}
}

// CheckScanLimit returns an error if the user exceeded their daily scan quota.
func (l *Limiter) CheckScanLimit(ctx context.Context, userID int64) error {
	ent, err := l.repo.GetEntitlements(ctx, userID)
	if err != nil {
		return err
	}
	if ent.ScansPerDay < 0 {
		return nil
	}
	limit := ent.ScansPerDay
	if limit <= 0 {
		limit = FreeScansPerDay
	}

	if l.cache == nil {
		return nil
	}

	key := scanDailyKey(userID, time.Now())
	count, err := l.cache.Get(ctx, key).Int()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	if count >= limit {
		return ErrScanLimitExceeded
	}
	return nil
}

// RecordScan increments the user's daily scan counter after a job is created.
func (l *Limiter) RecordScan(ctx context.Context, userID int64) error {
	if l.cache == nil {
		return nil
	}
	now := time.Now()
	key := scanDailyKey(userID, now)
	pipe := l.cache.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireAt(ctx, key, endOfDay(now))
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	_ = incr
	return nil
}

func scanDailyKey(userID int64, day time.Time) string {
	return fmt.Sprintf("scans:%d:%s", userID, day.Format("2006-01-02"))
}

func endOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 0, t.Location())
}
