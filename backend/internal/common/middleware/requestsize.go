package middleware

import (
	"errors"
	"net/http"

	apierrors "studyapp/backend/internal/common/errors"
)

// MaxBodySize limits the request body size. Handlers that read r.Body should
// call HandleBodyTooLarge when Read returns an error.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes && r.ContentLength != -1 {
				apierrors.WriteError(w, http.StatusRequestEntityTooLarge, apierrors.ErrCodeValidation, "request body too large", nil)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// HandleBodyTooLarge writes a 413 response when err is a MaxBytesError.
func HandleBodyTooLarge(w http.ResponseWriter, err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		apierrors.WriteError(w, http.StatusRequestEntityTooLarge, apierrors.ErrCodeValidation, "request body too large", nil)
		return true
	}
	return false
}
