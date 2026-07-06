package config

import (
	"strings"
	"testing"
)

func TestValidate_ProductionDefaultJWTSecret(t *testing.T) {
	cfg := Config{
		Port:           "8080",
		DatabaseURL:    "postgres://db.example.com:5432/studyapp",
		ValkeyAddr:     "cache.example.com:6379",
		JWTSecret:      defaultJWTSecret,
		Environment:    "production",
		RazorpayKeyID:  "rzp_test_key",
		AllowedOrigins: "https://app.studyapp.in",
		FrontendURL:    "https://app.studyapp.in",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for default JWT secret in production")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET in error, got: %v", err)
	}
}

func TestValidate_DevelopmentDefaultJWTSecret(t *testing.T) {
	cfg := Config{
		Port:          defaultPort,
		DatabaseURL:   defaultDatabaseURL,
		ValkeyAddr:    defaultValkeyAddr,
		JWTSecret:     defaultJWTSecret,
		Environment:   "development",
		RazorpayKeyID: "",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("development with defaults should not error, got: %v", err)
	}
}

func TestValidate_ProductionMissingFieldsListsAll(t *testing.T) {
	cfg := Config{
		Port:          "",
		DatabaseURL:   "",
		ValkeyAddr:    "",
		JWTSecret:     "",
		Environment:   "production",
		RazorpayKeyID: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing production fields")
	}

	msg := err.Error()
	for _, field := range []string{"PORT", "DATABASE_URL", "VALKEY_ADDR", "JWT_SECRET", "RAZORPAY_KEY_ID", "ALLOWED_ORIGINS", "FRONTEND_URL"} {
		if !strings.Contains(msg, field) {
			t.Errorf("expected error to mention %q, got: %s", field, msg)
		}
	}
}

func TestValidate_ProductionMissingAllowedOrigins(t *testing.T) {
	cfg := Config{
		Port:           "8080",
		DatabaseURL:    "postgres://db.example.com:5432/studyapp",
		ValkeyAddr:     "cache.example.com:6379",
		JWTSecret:      "super-secret-production-key",
		Environment:    "production",
		RazorpayKeyID:  "rzp_test_key",
		AllowedOrigins: "",
		FrontendURL:    "https://app.studyapp.in",
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected ALLOWED_ORIGINS error, got: %v", err)
	}
}

func TestValidate_ProductionRejectsLocalhostOrigin(t *testing.T) {
	cfg := Config{
		Port:           "8080",
		DatabaseURL:    "postgres://db.example.com:5432/studyapp",
		ValkeyAddr:     "cache.example.com:6379",
		JWTSecret:      "super-secret-production-key",
		Environment:    "production",
		RazorpayKeyID:  "rzp_test_key",
		AllowedOrigins: "http://localhost:3000",
		FrontendURL:    "https://app.studyapp.in",
	}

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected localhost origin rejection, got: %v", err)
	}
}

func TestValidate_ProductionLocalhostDatabaseRejected(t *testing.T) {
	cfg := Config{
		Port:           "8080",
		DatabaseURL:    defaultDatabaseURL,
		ValkeyAddr:     "cache.example.com:6379",
		JWTSecret:      "super-secret-production-key",
		Environment:    "production",
		RazorpayKeyID:  "rzp_test_key",
		AllowedOrigins: "https://app.studyapp.in",
		FrontendURL:    "https://app.studyapp.in",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for localhost DATABASE_URL in production")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL in error, got: %v", err)
	}
}
