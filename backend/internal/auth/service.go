package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
	Type  string `json:"type,omitempty"`
}

// Service provides authentication business logic.
type Service interface {
	Register(ctx context.Context, req *RegisterRequest) (*TokenResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*TokenResponse, error)
	ChangePassword(ctx context.Context, userID int64, req *ChangePasswordRequest) error
	VerifyToken(tokenString string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	InitiatePasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword, confirm string) error
	GetUser(ctx context.Context, userID int64) (*User, error)
}

// AuthService implements the Service interface.
type AuthService struct {
	repo          Repository
	jwtSecret     string
	tokenTTL      time.Duration
	refreshTTL    time.Duration
	resetMailer   ResetEmailSender
}


// NewAuthService creates a new authentication service.
func NewAuthService(repo Repository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtSecret:  jwtSecret,
		tokenTTL:   15 * time.Minute,   // Short-lived access token
		refreshTTL: 7 * 24 * time.Hour, // 7-day refresh token
	}
}

// WithPasswordResetMailer configures outbound reset email delivery.
func (s *AuthService) WithPasswordResetMailer(mailer ResetEmailSender) *AuthService {
	s.resetMailer = mailer
	return s
}

// Register creates a new user and returns a token.
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*TokenResponse, error) {
	email, err := ValidateEmail(req.Email)
	if err != nil {
		return nil, err
	}
	req.Email = email
	req.Name = SanitizeUserInput(req.Name)
	if req.Password != req.PasswordConfirm {
		return nil, errors.New("passwords do not match")
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}
	if !req.AcceptTerms {
		return nil, errors.New("you must accept the terms of service")
	}

	// Check if user already exists
	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &User{
		Name:          req.Name,
		Email:         req.Email,
		Phone:         req.Phone,
		PasswordHash:  passwordHash,
		Role:          "student",
		Status:        "active",
		EmailVerified: false,
	}

	userID, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = userID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.tokenTTL.Seconds()),
		TokenType:    "Bearer",
		User: &UserInfo{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// Login authenticates a user and returns a token.
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*TokenResponse, error) {
	email, err := ValidateEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}
	req.Email = email
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	if err := VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Check user status
	if user.Status != "active" {
		return nil, fmt.Errorf("user account is %s", user.Status)
	}
	if err := s.repo.UpdateLastLogin(ctx, user.ID, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to update login state: %w", err)
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.tokenTTL.Seconds()),
		TokenType:    "Bearer",
		User: &UserInfo{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// ChangePassword updates a user's password.
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, req *ChangePasswordRequest) error {
	if req.NewPassword != req.NewPasswordConfirm {
		return errors.New("new passwords do not match")
	}
	if err := ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if err := VerifyPassword(user.PasswordHash, req.CurrentPassword); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newPasswordHash, err := HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	return s.repo.UpdateUserPassword(ctx, userID, newPasswordHash)
}

func (s *AuthService) InitiatePasswordReset(ctx context.Context, email string) error {
	normalized, err := ValidateEmail(email)
	if err != nil {
		return nil
	}
	user, err := s.repo.GetUserByEmail(ctx, normalized)
	if err != nil {
		return nil
	}
	resetToken, err := GenerateRandomToken(24)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}
	if err := s.repo.CreatePasswordResetToken(ctx, &PasswordResetToken{
		UserID:    user.ID,
		Token:     resetToken,
		Used:      false,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		CreatedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}
	if s.resetMailer == nil {
		return fmt.Errorf("password reset email is not configured")
	}
	if err := s.resetMailer.SendPasswordReset(ctx, user, resetToken); err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword, confirm string) error {
	if newPassword != confirm {
		return errors.New("new passwords do not match")
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	resetToken, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}
	if resetToken.Used || time.Now().After(resetToken.ExpiresAt) {
		return errors.New("reset token has expired")
	}
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	if err := s.repo.UpdateUserPassword(ctx, resetToken.UserID, newHash); err != nil {
		return err
	}
	return s.repo.MarkPasswordResetTokenUsed(ctx, token)
}

// VerifyToken validates a JWT token and returns its claims.
func (s *AuthService) VerifyToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id in token: %w", err)
	}

	role := claims.Role
	if role == "" {
		role = "student"
	}
	claimType := claims.Type
	if claimType == "" && len(claims.Audience) > 0 {
		claimType = claims.Audience[0]
	}
	return &TokenClaims{
		UserID: userID,
		Email:  claims.Email,
		Role:   role,
		Type:   claimType,
	}, nil
}

// RefreshToken generates a new access token from a refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	claims, err := s.VerifyToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	if claims.Type != "refresh" {
		return nil, errors.New("token is not a refresh token")
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(s.tokenTTL.Seconds()),
		TokenType:   "Bearer",
		User: &UserInfo{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (s *AuthService) GetUser(ctx context.Context, userID int64) (*User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// generateAccessToken creates a short-lived JWT access token.
func (s *AuthService) generateAccessToken(user *User) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"access", user.Role},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "studyapp-backend",
		},
		Email: user.Email,
		Role:  user.Role,
		Type:  "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateRefreshToken creates a long-lived JWT refresh token.
func (s *AuthService) generateRefreshToken(user *User) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"refresh", user.Role},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "studyapp-backend",
		},
		Email: user.Email,
		Role:  user.Role,
		Type:  "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
