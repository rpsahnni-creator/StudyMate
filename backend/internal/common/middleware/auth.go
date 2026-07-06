package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	apierrors "studyapp/backend/internal/common/errors"
)

// RequireAuth is JWT auth middleware that verifies tokens and extracts user_id.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			apierrors.WriteUnauthorized(w)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, role, err := extractUserIDAndRoleFromToken(tokenString)
		if err != nil {
			apierrors.WriteError(w, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, "invalid or expired token", nil)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "user_role", role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin should run after RequireAuth and check the user's role.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value("user_role").(string)
		if !ok || role != "admin" {
			apierrors.WriteForbidden(w, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractUserIDAndRoleFromToken verifies JWT signature and extracts user_id and role.
func extractUserIDAndRoleFromToken(tokenString string) (int64, string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method (HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return 0, "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return 0, "", fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return 0, "", fmt.Errorf("invalid token claims")
	}

	// Extract user_id from Subject
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid user_id in token: %w", err)
	}

	// Extract role from Audience[1] (Audience[0] is email, Audience[1] is role)
	role := "student" // default role
	if len(claims.Audience) > 1 {
		role = claims.Audience[1]
	}

	return userID, role, nil
}
