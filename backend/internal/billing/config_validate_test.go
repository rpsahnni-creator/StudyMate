package billing

import (
	"strings"
	"testing"
)

func TestConfigValidate_ProductionRequiresRazorpaySecrets(t *testing.T) {
	cfg := Config{
		Environment:           "production",
		RazorpayKeyID:         "rzp_live_x",
		RazorpayKeySecret:     "",
		RazorpayWebhookSecret: "",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected production billing validation error")
	}
	msg := err.Error()
	for _, field := range []string{"RAZORPAY_KEY_SECRET", "RAZORPAY_WEBHOOK_SECRET"} {
		if !strings.Contains(msg, field) {
			t.Fatalf("expected %q in error, got: %s", field, msg)
		}
	}
}

func TestConfigValidate_DevelopmentAllowsMissingKeys(t *testing.T) {
	cfg := Config{Environment: "development"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development should not fail billing validation, got: %v", err)
	}
}

func TestCheckoutReady(t *testing.T) {
	if (Config{RazorpayKeyID: "id", RazorpayKeySecret: "secret"}).CheckoutReady() != true {
		t.Fatal("expected checkout ready")
	}
	if (Config{RazorpayKeyID: "id"}).CheckoutReady() {
		t.Fatal("expected checkout not ready without secret")
	}
	if (Config{
		RazorpayKeyID:     "rzp_test_StudyAppDevDummy01",
		RazorpayKeySecret: "rzp_test_dev_secret_dummy_key_12345",
	}).CheckoutReady() {
		t.Fatal("expected placeholder keys to disable live checkout")
	}
}
