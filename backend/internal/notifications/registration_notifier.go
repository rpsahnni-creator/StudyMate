package notifications

import (
	"context"
	"strconv"
	"strings"
)

// RegistrationNotifier enqueues auth-related transactional emails.
type RegistrationNotifier struct {
	worker *NotificationWorker
}

// NewRegistrationNotifier creates a notifier for signup OTP and welcome emails.
func NewRegistrationNotifier(worker *NotificationWorker) *RegistrationNotifier {
	return &RegistrationNotifier{worker: worker}
}

// EnqueueRegistrationOTP queues a pre-signup verification email.
func (n *RegistrationNotifier) EnqueueRegistrationOTP(ctx context.Context, email, otp string, expiryMinutes int) error {
	if n == nil || n.worker == nil {
		return nil
	}
	if expiryMinutes <= 0 {
		expiryMinutes = 10
	}
	return n.worker.Enqueue(ctx, QueueJob{
		UserID:     0,
		TemplateID: TmplEmailOTP,
		Data: map[string]string{
			"email":         strings.TrimSpace(email),
			"otp":           otp,
			"expiryMinutes": strconv.Itoa(expiryMinutes),
		},
		Channels: []string{ChannelEmail},
	})
}

// EnqueueRegistrationWelcome queues account confirmation after signup.
func (n *RegistrationNotifier) EnqueueRegistrationWelcome(ctx context.Context, userID int64, name, email, classLevel string) error {
	if n == nil || n.worker == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "there"
	}
	return n.worker.Enqueue(ctx, QueueJob{
		UserID:     userID,
		TemplateID: TmplWelcome,
		Data: map[string]string{
			"name":  name,
			"email": strings.TrimSpace(email),
			"class": classLevel,
		},
		Channels: []string{ChannelEmail},
	})
}
