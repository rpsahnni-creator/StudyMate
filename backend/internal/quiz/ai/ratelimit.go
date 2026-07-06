package ai

import (
	"sync"
	"time"
)

const (
	rateLimitWindow   = time.Minute
	rateLimitMaxCalls = 10
)

// RateLimiter enforces a per-user sliding-window cap on AI calls.
type RateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	maxCalls int
	calls    map[int64][]time.Time
}

// NewRateLimiter creates a limiter of maxCalls per window per user.
// Zero values fall back to 10 calls per minute.
func NewRateLimiter(maxCalls int, window time.Duration) *RateLimiter {
	if maxCalls <= 0 {
		maxCalls = rateLimitMaxCalls
	}
	if window <= 0 {
		window = rateLimitWindow
	}
	return &RateLimiter{
		window:   window,
		maxCalls: maxCalls,
		calls:    make(map[int64][]time.Time),
	}
}

// Allow reports whether a call for userID is permitted right now, recording it
// if so.
func (l *RateLimiter) Allow(userID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	recent := l.calls[userID][:0]
	for _, t := range l.calls[userID] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= l.maxCalls {
		l.calls[userID] = recent
		return false
	}

	l.calls[userID] = append(recent, now)
	return true
}
