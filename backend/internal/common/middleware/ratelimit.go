package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	apierrors "studyapp/backend/internal/common/errors"
)

// RateLimiter provides IP- and user-based request rate limiting via Valkey/Redis.
type RateLimiter struct {
	cache  *redis.Client
	logger *slog.Logger
}

func NewRateLimiter(cache *redis.Client, logger *slog.Logger) *RateLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	return &RateLimiter{cache: cache, logger: logger}
}

// ByIP limits requests per client IP for a route group.
func (rl *RateLimiter) ByIP(limit int, window time.Duration, routeGroup string) func(http.Handler) http.Handler {
	return rl.limit(func(r *http.Request) string {
		return fmt.Sprintf("rl:ip:%s:%s", clientIP(r), routeGroup)
	}, limit, window)
}

// ByUser limits requests per authenticated user for a route group.
func (rl *RateLimiter) ByUser(limit int, window time.Duration, routeGroup string) func(http.Handler) http.Handler {
	return rl.limit(func(r *http.Request) string {
		userID, ok := r.Context().Value("user_id").(int64)
		if !ok {
			return ""
		}
		return fmt.Sprintf("rl:user:%d:%s", userID, routeGroup)
	}, limit, window)
}

func (rl *RateLimiter) limit(keyFn func(*http.Request) string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			count, err := rl.cache.Incr(ctx, key).Result()
			if err != nil {
				rl.logger.Warn("rate limit incr failed", "error", err, "key", key)
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				_ = rl.cache.Expire(ctx, key, window).Err()
			}

			remaining := limit - int(count)
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if int(count) > limit {
				retryAfter := int(window.Seconds())
				if ttl, err := rl.cache.TTL(ctx, key).Result(); err == nil && ttl > 0 {
					retryAfter = int(ttl.Seconds())
					if retryAfter < 1 {
						retryAfter = 1
					}
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				apierrors.WriteError(w, http.StatusTooManyRequests, apierrors.ErrCodeRateLimit, "rate limit exceeded", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the best-effort client IP from the request.
func ClientIP(r *http.Request) string {
	return clientIP(r)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	if host == "" {
		return r.RemoteAddr
	}
	return host
}

// ResetRateLimitKey removes a rate limit key (for tests).
func ResetRateLimitKey(ctx context.Context, cache *redis.Client, key string) error {
	return cache.Del(ctx, key).Err()
}
