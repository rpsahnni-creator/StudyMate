package auth

import "time"

// User represents a registered user in the system.
type User struct {
	ID            int64     `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Email         string    `json:"email" db:"email"`
	Phone         *string   `json:"phone" db:"phone"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Role          string    `json:"role" db:"role"` // "student" | "admin"
	Status        string    `json:"status" db:"status"` // "active" | "inactive" | "suspended"
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// RegisterRequest is the payload for user registration after email OTP verification.
type RegisterRequest struct {
	VerificationToken string `json:"verification_token" validate:"required"`
	Name              string `json:"name" validate:"required,min=2,max=100"`
	Email             string `json:"email" validate:"required,email"`
	Class             string `json:"class" validate:"required"`
	Mobile            string `json:"mobile" validate:"required"`
	Password          string `json:"password" validate:"required,min=8"`
	PasswordConfirm   string `json:"password_confirm" validate:"required,eqfield=Password"`
	AcceptTerms       bool   `json:"accept_terms" validate:"required,eq=true"`
}

// SendRegistrationOTPResponse confirms OTP dispatch (dev_otp only in development stub mode).
type SendRegistrationOTPResponse struct {
	Message string `json:"message"`
	DevOTP  string `json:"dev_otp,omitempty"`
}

// SendRegistrationOTPRequest requests an email OTP before signup.
type SendRegistrationOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// VerifyRegistrationOTPRequest verifies the email OTP.
type VerifyRegistrationOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

// VerifyRegistrationOTPResponse returns a short-lived token for completing registration.
type VerifyRegistrationOTPResponse struct {
	VerificationToken string `json:"verification_token"`
	ExpiresIn         int64  `json:"expires_in"`
}

// EmailVerificationOTP stores a hashed email OTP and optional verification token.
type EmailVerificationOTP struct {
	ID                int64
	Email             string
	OTPHash           string
	Attempts          int
	Verified          bool
	VerificationToken *string
	TokenExpiresAt    *time.Time
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// TokenResponse is returned after successful login/registration.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int64     `json:"expires_in"` // seconds
	TokenType    string    `json:"token_type"` // "Bearer"
	User         *UserInfo `json:"user"`
}

// UserInfo is a safe subset of User for public API responses.
type UserInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// TokenClaims represents the JWT payload.
type TokenClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"type,omitempty"`
	// Standard claims are handled by jwt.RegisteredClaims embedded in token.Claims
}

// RefreshTokenRequest is the payload for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest is the payload for password change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	NewPasswordConfirm string `json:"new_password_confirm" validate:"required,eqfield=NewPassword"`
}

// ForgotPasswordRequest is the payload for password reset initiation.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest is the payload for password reset completion.
type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	NewPasswordConfirm string `json:"new_password_confirm" validate:"required,eqfield=NewPassword"`
}


// PasswordResetToken stores a secure one-time reset token.
type PasswordResetToken struct {
	ID        int64
	UserID    int64
	Token     string
	Used      bool
	ExpiresAt time.Time
	CreatedAt time.Time
}
