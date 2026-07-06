package auth

import (
	"context"
	"testing"
	"time"
)

// MockRepository is a mock implementation of Repository for testing.
type MockRepository struct {
	users       map[string]*User
	resetTokens map[string]*PasswordResetToken
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:       make(map[string]*User),
		resetTokens: make(map[string]*PasswordResetToken),
	}
}

func (m *MockRepository) CreateUser(ctx context.Context, user *User) (int64, error) {
	user.ID = int64(len(m.users) + 1)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return user.ID, nil
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user, ok := m.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *MockRepository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MockRepository) UpdateUserPassword(ctx context.Context, userID int64, newPasswordHash string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.PasswordHash = newPasswordHash
			user.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *MockRepository) UpdateUserStatus(ctx context.Context, userID int64, status string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.Status = status
			user.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *MockRepository) UpdateUserEmailVerified(ctx context.Context, userID int64, verified bool) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.EmailVerified = verified
			user.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *MockRepository) UpdateLastLogin(ctx context.Context, userID int64, ts time.Time) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.UpdatedAt = ts
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *MockRepository) CreatePasswordResetToken(ctx context.Context, token *PasswordResetToken) error {
	m.resetTokens[token.Token] = token
	return nil
}

func (m *MockRepository) GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	stored, ok := m.resetTokens[token]
	if !ok {
		return nil, ErrUserNotFound
	}
	return stored, nil
}

func (m *MockRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	stored, ok := m.resetTokens[token]
	if !ok {
		return ErrUserNotFound
	}
	stored.Used = true
	return nil
}

// ErrUserNotFound indicates user was not found
type UserNotFoundError struct{}

func (e *UserNotFoundError) Error() string {
	return "user not found"
}

var ErrUserNotFound = &UserNotFoundError{}

type mockResetMailer struct{}

func (m *mockResetMailer) SendPasswordReset(ctx context.Context, user *User, token string) error {
	return nil
}

func authServiceWithResetMailer(repo Repository) *AuthService {
	return NewAuthService(repo, "test-secret").WithPasswordResetMailer(&mockResetMailer{})
}

func TestRegister_Success(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	resp, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("Expected access token")
	}
}

func TestRegister_PasswordMismatch(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "DifferentPass456!",
		AcceptTerms:     true,
	})
	if err == nil {
		t.Fatal("Expected password mismatch error")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	_, err = service.Register(context.Background(), &RegisterRequest{
		Name:            "Jane Doe",
		Email:           "john@example.com",
		Password:        "AnotherPass456!",
		PasswordConfirm: "AnotherPass456!",
		AcceptTerms:     true,
	})
	if err == nil {
		t.Fatal("Expected duplicate email error")
	}
}

func TestLogin_Success(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	resp, err := service.Login(context.Background(), &LoginRequest{Email: "john@example.com", Password: "StrongPass123!"})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("Expected access token")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err = service.Login(context.Background(), &LoginRequest{Email: "john@example.com", Password: "WrongPass789!"})
	if err == nil {
		t.Fatal("Expected invalid password error")
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.Login(context.Background(), &LoginRequest{Email: "missing@example.com", Password: "StrongPass123!"})
	if err == nil {
		t.Fatal("Expected nonexistent user error")
	}
}

func TestChangePassword_Success(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	resp, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = service.ChangePassword(context.Background(), resp.User.ID, &ChangePasswordRequest{
		CurrentPassword:    "StrongPass123!",
		NewPassword:        "NewStrongPass456!",
		NewPasswordConfirm: "NewStrongPass456!",
	})
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
}

func TestVerifyToken_Success(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	resp, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	claims, err := service.VerifyToken(resp.AccessToken)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if claims.Email == "" {
		t.Fatal("Expected claims email")
	}
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.VerifyToken("invalid.token.string")
	if err == nil {
		t.Fatal("Expected invalid token error")
	}
}

func TestRegister_RejectsWeakPassword(t *testing.T) {
	repo := NewMockRepository()
	service := NewAuthService(repo, "test-secret")

	_, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "Jane Doe",
		Email:           "jane@example.com",
		Password:        "weak",
		PasswordConfirm: "weak",
		AcceptTerms:     true,
	})
	if err == nil {
		t.Fatal("Expected weak password rejection")
	}
}

func TestResetPassword_UsesToken(t *testing.T) {
	repo := NewMockRepository()
	service := authServiceWithResetMailer(repo)

	_, err := service.Register(context.Background(), &RegisterRequest{
		Name:            "Reset User",
		Email:           "reset@example.com",
		Password:        "StrongPass123!",
		PasswordConfirm: "StrongPass123!",
		AcceptTerms:     true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := service.InitiatePasswordReset(context.Background(), "reset@example.com"); err != nil {
		t.Fatalf("InitiatePasswordReset failed: %v", err)
	}

	var token string
	for k := range repo.resetTokens {
		token = k
		break
	}
	if token == "" {
		t.Fatal("Expected reset token in repository")
	}

	err = service.ResetPassword(context.Background(), token, "NewStrongPass456!", "NewStrongPass456!")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}
}
