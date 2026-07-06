package auth

import (
	"context"
	"fmt"
	"strings"

	"studyapp/backend/internal/notifications"
)

// RegistrationEmailSender delivers signup OTP and welcome emails.
type RegistrationEmailSender interface {
	SendRegistrationOTP(ctx context.Context, email, otp string) error
	SendRegistrationWelcome(ctx context.Context, user *User, classLevel string) error
}

// RegistrationMailer sends registration OTP and confirmation emails.
type RegistrationMailer struct {
	client TransactionalEmailClient
}

// NewRegistrationMailer creates a registration email mailer.
func NewRegistrationMailer(client TransactionalEmailClient) *RegistrationMailer {
	return &RegistrationMailer{client: client}
}

// SendRegistrationOTP emails a one-time verification code.
func (m *RegistrationMailer) SendRegistrationOTP(ctx context.Context, email, otp string) error {
	if m.client == nil {
		return fmt.Errorf("email client not configured")
	}

	tmpl, err := notifications.Render(notifications.TmplEmailOTP, map[string]string{
		"otp":           otp,
		"expiryMinutes": "10",
	})
	if err != nil {
		return fmt.Errorf("render registration otp template: %w", err)
	}

	return m.client.SendTransactional(ctx, notifications.EmailRequest{
		To:      email,
		Subject: tmpl.EmailSubj,
		HTML:    tmpl.EmailHTML,
		Text:    tmpl.EmailText,
		Tags:    []string{"registration_otp"},
	})
}

// SendRegistrationWelcome emails account confirmation after signup.
func (m *RegistrationMailer) SendRegistrationWelcome(ctx context.Context, user *User, classLevel string) error {
	if m.client == nil {
		return fmt.Errorf("email client not configured")
	}

	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = "there"
	}

	tmpl, err := notifications.Render(notifications.TmplWelcome, map[string]string{
		"name":  name,
		"email": user.Email,
		"class": classLevel,
	})
	if err != nil {
		return fmt.Errorf("render welcome template: %w", err)
	}

	return m.client.SendTransactional(ctx, notifications.EmailRequest{
		To:      user.Email,
		Subject: tmpl.EmailSubj,
		HTML:    tmpl.EmailHTML,
		Text:    tmpl.EmailText,
		Tags:    []string{"registration_welcome"},
	})
}
