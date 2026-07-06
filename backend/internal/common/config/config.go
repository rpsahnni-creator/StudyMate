package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultPort         = "8080"
	defaultDatabaseURL  = "postgres://localhost:5432/studyapp?sslmode=disable"
	defaultValkeyAddr   = "localhost:6379"
	defaultJWTSecret    = "dev-secret-change-in-production"
	defaultEnvironment  = "development"
	envAllowLocalhostDB = "ALLOW_LOCALHOST_DB"
)

// Config holds all environment-driven settings. Keep this as the single
// place new env vars get added, so nothing is scattered across the codebase.
type Config struct {
	Port           string
	DatabaseURL    string
	ValkeyAddr     string // Valkey is Redis-protocol-compatible; same client works
	JWTSecret      string
	Environment    string // "development" | "staging" | "production"
	RazorpayKeyID  string
	AllowedOrigins string
	FrontendURL    string
}

func Load() (Config, error) {
	loadDotEnv()

	cfg := Config{
		Port:           getEnv("PORT", defaultPort),
		DatabaseURL:    getEnv("DATABASE_URL", defaultDatabaseURL),
		ValkeyAddr:     getEnv("VALKEY_ADDR", defaultValkeyAddr),
		JWTSecret:      getEnv("JWT_SECRET", defaultJWTSecret),
		Environment:    strings.ToLower(getEnv("ENVIRONMENT", defaultEnvironment)),
		RazorpayKeyID:  getEnv("RAZORPAY_KEY_ID", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),
		FrontendURL:    strings.TrimRight(getEnv("FRONTEND_URL", defaultFrontendURL(env())), "/"),
	}
	return cfg, nil
}

func env() string {
	return strings.ToLower(getEnv("ENVIRONMENT", defaultEnvironment))
}

func defaultFrontendURL(environment string) string {
	if environment == "development" {
		return "http://localhost:3000"
	}
	return ""
}

// Validate checks configuration before the server starts.
// In production it fails fast with all invalid fields listed at once.
// In development it logs warnings for default values but does not fail.
func (c Config) Validate() error {
	switch c.Environment {
	case "production":
		return c.validateProduction()
	case "development":
		c.warnDevelopmentDefaults()
		return nil
	default:
		// staging and other environments: apply production rules for safety
		return c.validateProduction()
	}
}

func (c Config) validateProduction() error {
	var errs []error

	if strings.TrimSpace(c.Port) == "" {
		errs = append(errs, fmt.Errorf("PORT is required"))
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		errs = append(errs, fmt.Errorf("DATABASE_URL is required"))
	} else if strings.Contains(strings.ToLower(c.DatabaseURL), "localhost") && !allowLocalhostDB() {
		errs = append(errs, fmt.Errorf("DATABASE_URL must not use localhost in production (set %s=true to override)", envAllowLocalhostDB))
	}
	if strings.TrimSpace(c.ValkeyAddr) == "" {
		errs = append(errs, fmt.Errorf("VALKEY_ADDR is required"))
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		errs = append(errs, fmt.Errorf("JWT_SECRET is required"))
	} else if c.JWTSecret == defaultJWTSecret {
		errs = append(errs, fmt.Errorf("JWT_SECRET must not use the default dev secret in production"))
	}
	if strings.TrimSpace(c.Environment) == "" {
		errs = append(errs, fmt.Errorf("ENVIRONMENT is required"))
	}
	if strings.TrimSpace(c.RazorpayKeyID) == "" {
		errs = append(errs, fmt.Errorf("RAZORPAY_KEY_ID is required in production"))
	}
	if strings.TrimSpace(c.AllowedOrigins) == "" {
		errs = append(errs, fmt.Errorf("ALLOWED_ORIGINS is required in production"))
	} else {
		for _, origin := range splitList(c.AllowedOrigins) {
			lower := strings.ToLower(origin)
			if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
				errs = append(errs, fmt.Errorf("ALLOWED_ORIGINS must not include localhost in production"))
				break
			}
		}
	}
	if strings.TrimSpace(c.FrontendURL) == "" {
		errs = append(errs, fmt.Errorf("FRONTEND_URL is required in production"))
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid production config: %w", errors.Join(errs...))
}

func (c Config) warnDevelopmentDefaults() {
	if os.Getenv("PORT") == "" {
		log.Printf("config warning: PORT not set, using default %q", defaultPort)
	}
	if os.Getenv("DATABASE_URL") == "" {
		log.Printf("config warning: DATABASE_URL not set, using default %q", defaultDatabaseURL)
	}
	if os.Getenv("VALKEY_ADDR") == "" {
		log.Printf("config warning: VALKEY_ADDR not set, using default %q", defaultValkeyAddr)
	}
	if os.Getenv("JWT_SECRET") == "" {
		log.Printf("config warning: JWT_SECRET not set, using insecure default (ok for local dev only)")
	}
	if os.Getenv("RAZORPAY_KEY_ID") == "" {
		log.Printf("config warning: RAZORPAY_KEY_ID not set (billing webhooks disabled)")
	}
}

// loadDotEnv reads backend/.env when present (local dev only; production uses real env vars).
func loadDotEnv() {
	for _, path := range []string{".env", "backend/.env"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// PowerShell Set-Content -Encoding utf8 writes a BOM; godotenv.Load rejects the file.
		data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
		envMap, err := godotenv.Unmarshal(string(data))
		if err != nil {
			continue
		}
		for key, val := range envMap {
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		return
	}
}

func allowLocalhostDB() bool {
	return os.Getenv(envAllowLocalhostDB) == "true"
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
