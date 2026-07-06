package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for users.
type Repository interface {
	CreateUser(ctx context.Context, user *User) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	UpdateUserPassword(ctx context.Context, userID int64, newPasswordHash string) error
	UpdateUserStatus(ctx context.Context, userID int64, status string) error
	UpdateUserEmailVerified(ctx context.Context, userID int64, verified bool) error
	UpdateLastLogin(ctx context.Context, userID int64, ts time.Time) error
	CreatePasswordResetToken(ctx context.Context, token *PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, token string) error
	DeleteEmailOTPsForEmail(ctx context.Context, email string) error
	CreateEmailOTP(ctx context.Context, otp *EmailVerificationOTP) error
	GetLatestEmailOTP(ctx context.Context, email string) (*EmailVerificationOTP, error)
	IncrementEmailOTPAttempts(ctx context.Context, id int64) error
	MarkEmailOTPVerified(ctx context.Context, id int64, verificationToken string, tokenExpiresAt time.Time) error
	GetVerifiedEmailOTPByToken(ctx context.Context, email, verificationToken string) (*EmailVerificationOTP, error)
	DeleteEmailOTPsForEmailAfterUse(ctx context.Context, email string) error
	CreateUserProfile(ctx context.Context, userID int64, classLevel string) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

// CreateUser inserts a new user into the database.
func (r *PostgresRepository) CreateUser(ctx context.Context, user *User) (int64, error) {
	query := `
		INSERT INTO users (name, email, phone, password_hash, role, status, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	role := user.Role
	if role == "" {
		role = "student"
	}
	status := user.Status
	if status == "" {
		status = "active"
	}

	var userID int64
	err := r.pool.QueryRow(ctx, query,
		user.Name,
		user.Email,
		user.Phone,
		user.PasswordHash,
		role,
		status,
		user.EmailVerified,
		time.Now(),
		time.Now(),
	).Scan(&userID)

	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// GetUserByEmail retrieves a user by email address.
func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, name, email, phone, password_hash, role, status, email_verified, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID.
func (r *PostgresRepository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, name, email, phone, password_hash, role, status, email_verified, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// UpdateUserPassword updates a user's password hash.
func (r *PostgresRepository) UpdateUserPassword(ctx context.Context, userID int64, newPasswordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, newPasswordHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserStatus updates a user's status.
func (r *PostgresRepository) UpdateUserStatus(ctx context.Context, userID int64, status string) error {
	query := `
		UPDATE users
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, status, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserEmailVerified marks a user's email as verified.
func (r *PostgresRepository) UpdateUserEmailVerified(ctx context.Context, userID int64, verified bool) error {
	query := `
		UPDATE users
		SET email_verified = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, verified, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update email_verified: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *PostgresRepository) UpdateLastLogin(ctx context.Context, userID int64, ts time.Time) error {
	query := `
		UPDATE users
		SET last_login_at = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, ts, ts, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *PostgresRepository) CreatePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token, used, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query, token.UserID, token.Token, token.Used, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token, used, expires_at, created_at
		FROM password_reset_tokens
		WHERE token = $1
	`

	var resetToken PasswordResetToken
	err := r.pool.QueryRow(ctx, query, token).Scan(&resetToken.ID, &resetToken.UserID, &resetToken.Token, &resetToken.Used, &resetToken.ExpiresAt, &resetToken.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}
	return &resetToken, nil
}

func (r *PostgresRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	query := `
		UPDATE password_reset_tokens
		SET used = true
		WHERE token = $1
	`

	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to mark password reset token used: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteEmailOTPsForEmail(ctx context.Context, email string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_verification_otps WHERE email = $1`, email)
	if err != nil {
		return fmt.Errorf("failed to delete email otps: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateEmailOTP(ctx context.Context, otp *EmailVerificationOTP) error {
	query := `
		INSERT INTO email_verification_otps (email, otp_hash, attempts, verified, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return r.pool.QueryRow(ctx, query,
		otp.Email,
		otp.OTPHash,
		otp.Attempts,
		otp.Verified,
		otp.ExpiresAt,
		otp.CreatedAt,
	).Scan(&otp.ID)
}

func (r *PostgresRepository) GetLatestEmailOTP(ctx context.Context, email string) (*EmailVerificationOTP, error) {
	query := `
		SELECT id, email, otp_hash, attempts, verified, verification_token, token_expires_at, expires_at, created_at
		FROM email_verification_otps
		WHERE email = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	otp := &EmailVerificationOTP{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&otp.ID,
		&otp.Email,
		&otp.OTPHash,
		&otp.Attempts,
		&otp.Verified,
		&otp.VerificationToken,
		&otp.TokenExpiresAt,
		&otp.ExpiresAt,
		&otp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest email otp: %w", err)
	}
	return otp, nil
}

func (r *PostgresRepository) IncrementEmailOTPAttempts(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE email_verification_otps SET attempts = attempts + 1 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to increment otp attempts: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkEmailOTPVerified(ctx context.Context, id int64, verificationToken string, tokenExpiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE email_verification_otps
		SET verified = true, verification_token = $1, token_expires_at = $2
		WHERE id = $3
	`, verificationToken, tokenExpiresAt, id)
	if err != nil {
		return fmt.Errorf("failed to mark email otp verified: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetVerifiedEmailOTPByToken(ctx context.Context, email, verificationToken string) (*EmailVerificationOTP, error) {
	query := `
		SELECT id, email, otp_hash, attempts, verified, verification_token, token_expires_at, expires_at, created_at
		FROM email_verification_otps
		WHERE email = $1 AND verification_token = $2 AND verified = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	otp := &EmailVerificationOTP{}
	err := r.pool.QueryRow(ctx, query, email, verificationToken).Scan(
		&otp.ID,
		&otp.Email,
		&otp.OTPHash,
		&otp.Attempts,
		&otp.Verified,
		&otp.VerificationToken,
		&otp.TokenExpiresAt,
		&otp.ExpiresAt,
		&otp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get verified email otp: %w", err)
	}
	return otp, nil
}

func (r *PostgresRepository) DeleteEmailOTPsForEmailAfterUse(ctx context.Context, email string) error {
	return r.DeleteEmailOTPsForEmail(ctx, email)
}

func (r *PostgresRepository) CreateUserProfile(ctx context.Context, userID int64, classLevel string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_profiles (user_id, class_level)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET class_level = EXCLUDED.class_level
	`, userID, classLevel)
	if err != nil {
		return fmt.Errorf("failed to create user profile: %w", err)
	}
	return nil
}
