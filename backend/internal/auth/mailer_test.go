package auth

import (
	"context"
	"strings"
	"testing"

	"studyapp/backend/internal/notifications"
)

type recordingEmailClient struct {
	lastReq notifications.EmailRequest
}

func (c *recordingEmailClient) SendTransactional(ctx context.Context, req notifications.EmailRequest) error {
	c.lastReq = req
	return nil
}

func TestPasswordResetMailer_SendsRenderedEmail(t *testing.T) {
	client := &recordingEmailClient{}
	mailer := NewPasswordResetMailer(client, "https://app.studyapp.in")

	user := &User{
		ID:    1,
		Name:  "Asha",
		Email: "asha@example.com",
	}
	if err := mailer.SendPasswordReset(context.Background(), user, "reset-token-abc"); err != nil {
		t.Fatalf("SendPasswordReset failed: %v", err)
	}
	if client.lastReq.To != "asha@example.com" {
		t.Fatalf("unexpected recipient: %q", client.lastReq.To)
	}
	if client.lastReq.Subject == "" {
		t.Fatal("expected non-empty subject")
	}
	if client.lastReq.HTML == "" || client.lastReq.Text == "" {
		t.Fatal("expected html and text body")
	}
	if !strings.Contains(client.lastReq.HTML, "Asha") || !strings.Contains(client.lastReq.HTML, "reset-password") {
		t.Fatalf("unexpected html body: %q", client.lastReq.HTML)
	}
}
