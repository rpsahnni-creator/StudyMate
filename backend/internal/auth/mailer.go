package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"studyapp/backend/internal/notifications"
)

// ResetEmailSender delivers password reset emails.
type ResetEmailSender interface {
	SendPasswordReset(ctx context.Context, user *User, token string) error
}

// TransactionalEmailClient is the minimal email interface auth needs.
type TransactionalEmailClient interface {
	SendTransactional(ctx context.Context, req notifications.EmailRequest) error
}

// PasswordResetMailer sends password reset links via the notifications email stack.
type PasswordResetMailer struct {
	client      TransactionalEmailClient
	frontendURL string
}

// NewPasswordResetMailer creates a mailer that builds reset links from FRONTEND_URL.
func NewPasswordResetMailer(client TransactionalEmailClient, frontendURL string) *PasswordResetMailer {
	return &PasswordResetMailer{
		client:      client,
		frontendURL: strings.TrimRight(frontendURL, "/"),
	}
}

// SendPasswordReset renders and sends the reset email.
func (m *PasswordResetMailer) SendPasswordReset(ctx context.Context, user *User, token string) error {
	if m.client == nil {
		return fmt.Errorf("email client not configured")
	}
	if m.frontendURL == "" {
		return fmt.Errorf("FRONTEND_URL is not configured")
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", m.frontendURL, url.QueryEscape(token))

	name := user.Name
	if name == "" {
		name = "there"
	}

	tmpl, err := notifications.Render(notifications.TmplPasswordReset, map[string]string{
		"name":          name,
		"resetUrl":      resetURL,
		"expiryMinutes": "30",
	})
	if err != nil {
		return fmt.Errorf("render password reset template: %w", err)
	}

	return m.client.SendTransactional(ctx, notifications.EmailRequest{
		To:      user.Email,
		Subject: tmpl.EmailSubj,
		HTML:    tmpl.EmailHTML,
		Text:    tmpl.EmailText,
		Tags:    []string{"password_reset"},
	})
}
