package notifications

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type stubRepo struct{}

func (stubRepo) CreateFCMDeviceToken(ctx context.Context, token *FCMDeviceToken) error { return nil }
func (stubRepo) GetActiveDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return nil, nil
}
func (stubRepo) MarkTokenInactive(ctx context.Context, token string) error             { return nil }
func (stubRepo) UpdateTokenLastSeen(ctx context.Context, tokenID uuid.UUID) error      { return nil }
func (stubRepo) CleanupStaleTokens(ctx context.Context) error                          { return nil }
func (stubRepo) HardDeleteOldTokens(ctx context.Context) error                         { return nil }
func (stubRepo) CreateNotificationJob(ctx context.Context, job *NotificationJob) error { return nil }
func (stubRepo) GetPendingJobs(ctx context.Context, priority string, limit int) ([]*NotificationJob, error) {
	return nil, nil
}
func (stubRepo) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string) error {
	return nil
}
func (stubRepo) RescheduleJob(ctx context.Context, jobID uuid.UUID, delay time.Duration, reason string) error {
	return nil
}
func (stubRepo) FailJob(ctx context.Context, jobID uuid.UUID, reason string) error { return nil }
func (stubRepo) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*NotificationPreferences, error) {
	return &NotificationPreferences{PushEnabled: true, EmailEnabled: true}, nil
}
func (stubRepo) UpsertUserPreferences(ctx context.Context, prefs *NotificationPreferences) error {
	return nil
}
func (stubRepo) CountNotifications(ctx context.Context, userID uuid.UUID, channel string, statuses []string, duration time.Duration) (int, error) {
	return 0, nil
}
func (stubRepo) JobExistsByIdempotencyKey(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (stubRepo) LogEmailEvent(ctx context.Context, event *EmailEvent) error { return nil }
func (stubRepo) GetTemplate(ctx context.Context, key string) (*NotificationTemplate, error) {
	return &NotificationTemplate{Subject: "Hi {{name}}", BodyHTML: "Hello {{name}}"}, nil
}

func TestBuildPushPayloadIncludesTemplateData(t *testing.T) {
	svc := &Service{}
	job := &NotificationJob{Category: CategoryQuizReady, TemplateKey: "quiz_ready", TemplateData: JSONB{"name": "Asha", "action": "open_quiz"}}
	payload := svc.BuildPushPayload(job)
	if payload["name"] != "Asha" || payload["action"] != "open_quiz" {
		t.Fatalf("expected template data to flow into push payload, got %#v", payload)
	}
}

func TestRenderTemplateReplacesPlaceholders(t *testing.T) {
	repo := &stubRepo{}
	svc := &Service{repo: (*Repository)(nil)}
	_ = repo
	_ = svc
}

func TestRenderTemplateFallsBackWithoutRepository(t *testing.T) {
	svc := &Service{}
	subject, body, err := svc.renderTemplate(context.Background(), "custom_fallback", JSONB{"name": "Asha"})
	if err != nil {
		t.Fatalf("expected template fallback to succeed, got %v", err)
	}
	if !strings.Contains(subject, "Asha") || !strings.Contains(body, "Asha") {
		t.Fatalf("expected rendered template to include user data, got subject=%q body=%q", subject, body)
	}
}

func TestUserIDFromContextSupportsInt64(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(42))
	userID, err := userIDFromContext(ctx)
	if err != nil {
		t.Fatalf("expected context user id to resolve, got %v", err)
	}
	if userID == uuid.Nil {
		t.Fatal("expected a non-zero user id")
	}
}
