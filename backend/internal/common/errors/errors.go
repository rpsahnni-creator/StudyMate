package errors

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse is the standard API error envelope returned to clients.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

const (
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeRateLimit         = "RATE_LIMIT_EXCEEDED"
	ErrCodeFeatureGated      = "FEATURE_NOT_AVAILABLE"
	ErrCodeScanLimitExceeded = "SCAN_LIMIT_EXCEEDED"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodePaymentFailed     = "PAYMENT_FAILED"
)

type traceIDWriter interface {
	TraceID() string
}

func traceIDFromWriter(w http.ResponseWriter) string {
	if tw, ok := w.(traceIDWriter); ok {
		return tw.TraceID()
	}
	return ""
}

// WriteError writes a structured JSON error response.
func WriteError(w http.ResponseWriter, statusCode int, code, message string, details any) {
	traceID := traceIDFromWriter(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
		TraceID: traceID,
	})
}

// WriteValidationError writes a 400 with field-level validation details.
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteError(w, http.StatusBadRequest, ErrCodeValidation, "validation failed", fields)
}

// WriteNotFound writes a 404 for a missing resource.
func WriteNotFound(w http.ResponseWriter, resourceName string) {
	WriteError(w, http.StatusNotFound, ErrCodeNotFound, resourceName+" not found", nil)
}

// WriteUnauthorized writes a 401 response.
func WriteUnauthorized(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "authentication required", nil)
}

// WriteForbidden writes a 403 response.
func WriteForbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "access denied"
	}
	WriteError(w, http.StatusForbidden, ErrCodeForbidden, message, nil)
}

// WriteInternal logs the full error server-side and returns a generic message to the client.
func WriteInternal(w http.ResponseWriter, logger *slog.Logger, err error, traceID string) {
	if traceID == "" {
		traceID = traceIDFromWriter(w)
	}
	if logger != nil && err != nil {
		logger.Error("internal server error", "error", err, "trace_id", traceID)
	}
	WriteError(w, http.StatusInternalServerError, ErrCodeInternalError, "an internal error occurred", nil)
}
