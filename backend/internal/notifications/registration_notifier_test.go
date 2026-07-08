package notifications

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRegistrationNotifier_EnqueueRegistrationOTP(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	cache := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	worker := NewNotificationWorker(nil, cache, nil, nil, nil, nil)
	notifier := NewRegistrationNotifier(worker)

	if err := notifier.EnqueueRegistrationOTP(context.Background(), "student@example.com", "123456", 10); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	payload, err := cache.LPop(context.Background(), queueKey).Result()
	if err != nil {
		t.Fatalf("lpop failed: %v", err)
	}

	var job QueueJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if job.UserID != 0 {
		t.Fatalf("user_id = %d, want 0", job.UserID)
	}
	if job.TemplateID != TmplEmailOTP {
		t.Fatalf("template = %q", job.TemplateID)
	}
	if job.Data["email"] != "student@example.com" {
		t.Fatalf("email = %q", job.Data["email"])
	}
	if job.Data["otp"] != "123456" {
		t.Fatalf("otp = %q", job.Data["otp"])
	}
}
