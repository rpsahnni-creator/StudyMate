package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for _, header := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	} {
		if rec.Header().Get(header) == "" {
			t.Fatalf("expected %s header", header)
		}
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("HSTS should not be set outside production")
	}
}

func TestSecurityHeaders_ProductionHSTS(t *testing.T) {
	handler := SecurityHeaders(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("expected HSTS in production")
	}
}

func TestCORSMiddleware_AllowsListedOrigin(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("expected allowed origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_BlocksUnknownOriginOnOptions(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown origin preflight, got %d", rec.Code)
	}
}

func TestLoadAllowedOrigins_DevelopmentDefaults(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	origins := LoadAllowedOrigins("development")
	if len(origins) != 2 {
		t.Fatalf("expected 2 dev origins, got %v", origins)
	}
}

func TestMaxBodySize_RejectsLargeContentLength(t *testing.T) {
	handler := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.ContentLength = 2048
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if got := ClientIP(req); got != "203.0.113.1" {
		t.Fatalf("expected forwarded IP, got %q", got)
	}
}
