package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"studyapp/backend/internal/featureflags"
)

// Options controls one-time startup bootstrap tasks.
type Options struct {
	Environment         string
	AdminBootstrapEmail string
	EnableDevModules    bool
}

// LoadOptions reads bootstrap settings from the environment.
func LoadOptions(environment string) Options {
	enableDev := strings.EqualFold(environment, "development") ||
		os.Getenv("DEV_ENABLE_ALL_MODULES") == "true"

	return Options{
		Environment:         environment,
		AdminBootstrapEmail: strings.TrimSpace(os.Getenv("ADMIN_BOOTSTRAP_EMAIL")),
		EnableDevModules:    enableDev,
	}
}

// Run applies idempotent startup bootstrap (admin promotion, dev feature flags).
func Run(ctx context.Context, pool *pgxpool.Pool, flags *featureflags.Service, opts Options, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}

	if email := opts.AdminBootstrapEmail; email != "" {
		promoted, err := PromoteAdminByEmail(ctx, pool, email)
		if err != nil {
			logger.Warn("admin bootstrap failed", "email", email, "error", err)
		} else if promoted {
			logger.Info("admin bootstrap: user promoted to admin (re-login required for JWT role)", "email", email)
		} else {
			logger.Info("admin bootstrap: waiting for user registration", "email", email)
		}
	}

	if opts.EnableDevModules && flags != nil {
		if err := EnableDevFeatureModules(ctx, flags); err != nil {
			logger.Warn("dev feature bootstrap failed", "error", err)
		} else {
			logger.Info("dev bootstrap: enabled scan_quiz_module and career_goals_module")
		}
	}
}

// PromoteAdminByEmail sets users.role = admin for the given email.
// Returns true when a row was updated.
func PromoteAdminByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (bool, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return false, fmt.Errorf("email is required")
	}

	tag, err := pool.Exec(ctx, `
		UPDATE users
		SET role = 'admin', updated_at = now()
		WHERE lower(email) = $1 AND role <> 'admin'
	`, email)
	if err != nil {
		return false, fmt.Errorf("promote admin: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// AdminExists reports whether an admin user already exists.
func AdminExists(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin')`).Scan(&exists)
	return exists, err
}

// EnableDevFeatureModules turns on core modules for local development.
func EnableDevFeatureModules(ctx context.Context, flags *featureflags.Service) error {
	if flags == nil {
		return fmt.Errorf("feature flag service is nil")
	}
	for _, key := range []featureflags.FlagKey{
		featureflags.FlagScanQuiz,
		featureflags.FlagCareerGoals,
	} {
		if err := flags.SetFlag(ctx, key, true, 100, "dev_bootstrap"); err != nil {
			return fmt.Errorf("enable %s: %w", key, err)
		}
	}
	return nil
}

// LookupUserIDByEmail returns the numeric user id for bootstrap scripts.
func LookupUserIDByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("user not found: %s", email)
		}
		return 0, err
	}
	return id, nil
}
