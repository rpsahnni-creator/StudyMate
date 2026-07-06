package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestHMACSHA256Deterministic(t *testing.T) {
	payload := []byte(`{"event":"payment.captured"}`)
	secret := "whsec_test"
	got := hmacSHA256(payload, secret)
	want := hex.EncodeToString(func() []byte {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		return mac.Sum(nil)
	}())
	if got != want {
		t.Fatalf("unexpected hmac: %s", got)
	}
}

func TestVerifyRazorpaySignatureRejectsInvalid(t *testing.T) {
	svc := &Service{cfg: Config{RazorpayWebhookSecret: "secret", Environment: "production"}}
	err := svc.verifyRazorpaySignature([]byte(`{"ok":true}`), "bad-signature")
	if err != ErrInvalidWebhookSignature {
		t.Fatalf("expected invalid signature error, got %v", err)
	}
}

func TestVerifyRazorpaySignatureAllowsValid(t *testing.T) {
	payload := []byte(`{"event":"payment.captured"}`)
	secret := "whsec_test"
	sig := hmacSHA256(payload, secret)
	svc := &Service{cfg: Config{RazorpayWebhookSecret: secret, Environment: "production"}}
	if err := svc.verifyRazorpaySignature(payload, sig); err != nil {
		t.Fatalf("expected valid signature, got %v", err)
	}
}

func TestScansPerDayFromLimit(t *testing.T) {
	if got := scansPerDayFromLimit(10, "basic"); got != 10 {
		t.Fatalf("expected basic limit 10, got %d", got)
	}
	if got := scansPerDayFromLimit(0, "pro"); got != -1 {
		t.Fatalf("expected pro unlimited, got %d", got)
	}
}

func TestLimiterUnlimitedSkipsRedis(t *testing.T) {
	repo := &fakeEntitlementsRepo{ent: Entitlements{Plan: "pro", ScansPerDay: -1}}
	limiter := NewLimiter(repo, nil)
	if err := limiter.CheckScanLimit(context.Background(), 1); err != nil {
		t.Fatalf("expected unlimited pass, got %v", err)
	}
}

type fakeEntitlementsRepo struct {
	ent Entitlements
}

func (f *fakeEntitlementsRepo) ListActivePlans(context.Context) ([]Plan, error) { return nil, nil }
func (f *fakeEntitlementsRepo) GetPlanBySlug(context.Context, string) (Plan, error) {
	return Plan{}, ErrPlanNotFound
}
func (f *fakeEntitlementsRepo) GetPlanByID(context.Context, int64) (Plan, error) {
	return Plan{Name: "Basic"}, nil
}
func (f *fakeEntitlementsRepo) GetUserProfile(context.Context, int64) (UserProfile, error) {
	return UserProfile{}, nil
}
func (f *fakeEntitlementsRepo) CreatePendingPayment(context.Context, int64, int64, string, string, int64, string) (int64, error) {
	return 0, nil
}
func (f *fakeEntitlementsRepo) GetPaymentByProviderOrderID(context.Context, string, string) (PaymentRecord, error) {
	return PaymentRecord{}, ErrPaymentNotFound
}
func (f *fakeEntitlementsRepo) MarkPaymentCompleted(context.Context, int64, string, time.Time) error {
	return nil
}
func (f *fakeEntitlementsRepo) MarkPaymentFailed(context.Context, int64) error { return nil }
func (f *fakeEntitlementsRepo) UpsertActiveSubscription(context.Context, int64, int64, string, string) error {
	return nil
}
func (f *fakeEntitlementsRepo) CancelSubscription(context.Context, int64) error { return nil }
func (f *fakeEntitlementsRepo) GetEntitlements(context.Context, int64) (Entitlements, error) {
	return f.ent, nil
}
func (f *fakeEntitlementsRepo) EventExists(context.Context, string) (bool, error) { return false, nil }
func (f *fakeEntitlementsRepo) InsertPaymentEvent(context.Context, string, string, string, []byte, *int64) (int64, error) {
	return 0, nil
}
func (f *fakeEntitlementsRepo) MarkPaymentEventProcessed(context.Context, int64) error { return nil }
