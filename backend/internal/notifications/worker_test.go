package notifications

import (
	"context"
	"testing"
)

func TestIsTransactionalTemplate(t *testing.T) {
	if !isTransactionalTemplate(TmplEmailOTP) {
		t.Fatal("expected email_otp to be transactional")
	}
	if !isTransactionalTemplate(TmplWelcome) {
		t.Fatal("expected welcome to be transactional")
	}
	if isTransactionalTemplate(TmplQuizReady) {
		t.Fatal("expected quiz_ready to be non-transactional")
	}
}

func TestNotificationWorker_resolveRecipientEmail(t *testing.T) {
	w := &NotificationWorker{}

	email, err := w.resolveRecipientEmail(context.Background(), QueueJob{
		UserID: 0,
		Data:   map[string]string{"email": "student@example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "student@example.com" {
		t.Fatalf("got %q", email)
	}

	_, err = w.resolveRecipientEmail(context.Background(), QueueJob{UserID: 0})
	if err == nil {
		t.Fatal("expected error for missing recipient")
	}
}
