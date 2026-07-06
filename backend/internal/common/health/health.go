package health

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"studyapp/backend/internal/scan/storage"
)

const defaultVersion = "1.0.0"

// CheckResult is the status of a single dependency probe.
type CheckResult struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Response is the JSON payload for GET /health.
type Response struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
}

// Prober runs dependency health checks.
type Prober struct {
	Pool            *pgxpool.Pool
	Cache           *redis.Client
	Storage         storage.Client
	OCRProviderName string
	AIProviderName  string
	Version         string
}

// Handler returns GET /health with component-level status.
func (p *Prober) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp := p.Check(ctx)
		code := http.StatusOK
		if resp.Status == "down" {
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// ReadyHandler returns GET /ready for load balancer readiness probes.
func (p *Prober) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		dbOK := p.checkDatabase(ctx).Status == "ok"
		cacheOK := p.checkCache(ctx).Status == "ok"

		if dbOK && cacheOK {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	})
}

// Check evaluates all dependencies and returns aggregate status.
func (p *Prober) Check(ctx context.Context) Response {
	checks := map[string]CheckResult{
		"database":    p.checkDatabase(ctx),
		"cache":       p.checkCache(ctx),
		"storage":     p.checkStorage(ctx),
		"ai_provider": p.checkAIProvider(),
	}

	overall := "ok"
	criticalDown := checks["database"].Status == "down" || checks["cache"].Status == "down"
	if criticalDown {
		overall = "down"
	} else {
		for name, c := range checks {
			if name == "database" || name == "cache" {
				continue
			}
			if c.Status != "ok" {
				overall = "degraded"
				break
			}
		}
	}

	version := p.Version
	if version == "" {
		version = os.Getenv("APP_VERSION")
	}
	if version == "" {
		version = defaultVersion
	}

	return Response{
		Status:    overall,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   version,
		Checks:    checks,
	}
}

func (p *Prober) checkDatabase(ctx context.Context) CheckResult {
	if p.Pool == nil {
		return CheckResult{Status: "down", Message: "not configured"}
	}
	start := time.Now()
	err := p.Pool.Ping(ctx)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return CheckResult{Status: "down", LatencyMs: latency, Message: "unreachable"}
	}
	return CheckResult{Status: "ok", LatencyMs: latency}
}

func (p *Prober) checkCache(ctx context.Context) CheckResult {
	if p.Cache == nil {
		return CheckResult{Status: "down", Message: "not configured"}
	}
	start := time.Now()
	err := p.Cache.Ping(ctx).Err()
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return CheckResult{Status: "down", LatencyMs: latency, Message: "unreachable"}
	}
	return CheckResult{Status: "ok", LatencyMs: latency}
}

func (p *Prober) checkStorage(ctx context.Context) CheckResult {
	if p.Storage == nil {
		return CheckResult{Status: "degraded", Message: "not configured"}
	}
	start := time.Now()
	key := "health/probe-" + time.Now().UTC().Format("20060102150405")
	err := p.Storage.PutObject(ctx, key, []byte("ok"))
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return CheckResult{Status: "degraded", LatencyMs: latency, Message: "write failed"}
	}
	_ = p.Storage.DeleteObject(ctx, key)
	return CheckResult{Status: "ok", LatencyMs: latency}
}

func (p *Prober) checkAIProvider() CheckResult {
	if p.AIProviderName == "" && p.OCRProviderName == "" {
		return CheckResult{Status: "degraded", Message: "provider not configured"}
	}
	msg := p.AIProviderName
	if p.OCRProviderName != "" {
		if p.OCRProviderName == "gemini_vision" {
			msg = "scan=gemini_vision (page image)"
		} else if p.OCRProviderName == "stub" {
			msg = "scan=stub (dev fake OCR — NOT for real scans)"
		} else {
			msg = "scan=" + p.OCRProviderName + " ai=" + p.AIProviderName
		}
	}
	return CheckResult{Status: "ok", Message: msg}
}
