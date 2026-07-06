package middleware

import (
	"net/http"
	"strings"
)

type securityConfig struct {
	production bool
}

// SecurityHeaders returns middleware that sets standard security response headers.
func SecurityHeaders(production bool) func(http.Handler) http.Handler {
	cfg := securityConfig{production: production}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			if cfg.production {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000")
			}
			next.ServeHTTP(&stripServerHeaders{ResponseWriter: w}, r)
		})
	}
}

type stripServerHeaders struct {
	http.ResponseWriter
}

func (w *stripServerHeaders) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Server")
	w.ResponseWriter.Header().Del("X-Powered-By")
	for k, vals := range w.ResponseWriter.Header() {
		if strings.EqualFold(k, "Server") || strings.EqualFold(k, "X-Powered-By") {
			w.ResponseWriter.Header().Del(k)
			continue
		}
		_ = vals
	}
	w.ResponseWriter.WriteHeader(code)
}
