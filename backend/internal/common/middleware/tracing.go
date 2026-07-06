package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type traceIDContextKey struct{}

// TraceIDKey is the context key for the request trace identifier.
var TraceIDKey = traceIDContextKey{}

const traceIDHeader = "X-Trace-ID"

type traceResponseWriter struct {
	http.ResponseWriter
	traceID string
}

func (tw *traceResponseWriter) TraceID() string {
	return tw.traceID
}

// TraceIDMiddleware reads or generates a trace ID, stores it in context,
// and sets the X-Trace-ID response header on every response.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := strings.TrimSpace(r.Header.Get(traceIDHeader))
		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		wrapped := &traceResponseWriter{ResponseWriter: w, traceID: traceID}
		w.Header().Set(traceIDHeader, traceID)
		next.ServeHTTP(wrapped, r.WithContext(ctx))
	})
}

// GetTraceID returns the trace ID from context, or an empty string if absent.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}
