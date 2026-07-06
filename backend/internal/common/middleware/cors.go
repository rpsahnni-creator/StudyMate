package middleware

import (
	"net/http"
	"os"
	"strings"
)

// DefaultDevOrigins are allowed when ALLOWED_ORIGINS is unset in development.
var DefaultDevOrigins = []string{
	"http://localhost:3000",
	"http://localhost:8081",
}

// LoadAllowedOrigins returns CORS allowed origins from env or environment defaults.
func LoadAllowedOrigins(environment string) []string {
	if raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")); raw != "" {
		return splitOrigins(raw)
	}
	if strings.EqualFold(environment, "development") {
		return append([]string(nil), DefaultDevOrigins...)
	}
	return nil
}

func splitOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// CORSMiddleware restricts cross-origin access to an explicit allowlist.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				} else if len(allowed) > 0 {
					if r.Method == http.MethodOptions {
						w.WriteHeader(http.StatusForbidden)
						return
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Trace-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace-ID, X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After")

			if r.Method == http.MethodOptions {
				if origin != "" && len(allowed) > 0 {
					if _, ok := allowed[origin]; !ok {
						w.WriteHeader(http.StatusForbidden)
						return
					}
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS allows default development origins. Prefer CORSMiddleware in production wiring.
func CORS(next http.Handler) http.Handler {
	return CORSMiddleware(DefaultDevOrigins)(next)
}
