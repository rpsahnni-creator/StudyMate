package auth

import "context"

// RegistrationNotifier enqueues signup OTP and welcome emails for async delivery.
type RegistrationNotifier interface {
	EnqueueRegistrationOTP(ctx context.Context, email, otp string, expiryMinutes int) error
	EnqueueRegistrationWelcome(ctx context.Context, userID int64, name, email, classLevel string) error
}
