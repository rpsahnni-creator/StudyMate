package billing

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// Validate checks billing credentials for the current environment.
func (c Config) Validate() error {
	env := strings.ToLower(strings.TrimSpace(c.Environment))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT")))
	}
	if env == "" {
		env = "development"
	}

	switch env {
	case "development":
		c.warnDevelopmentDefaults()
		return nil
	default:
		return c.validateProduction()
	}
}

func (c Config) validateProduction() error {
	var errs []error

	if strings.TrimSpace(c.RazorpayKeyID) == "" {
		errs = append(errs, fmt.Errorf("RAZORPAY_KEY_ID is required in production"))
	}
	if strings.TrimSpace(c.RazorpayKeySecret) == "" {
		errs = append(errs, fmt.Errorf("RAZORPAY_KEY_SECRET is required in production"))
	}
	if strings.TrimSpace(c.RazorpayWebhookSecret) == "" {
		errs = append(errs, fmt.Errorf("RAZORPAY_WEBHOOK_SECRET is required in production"))
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid billing config: %w", errors.Join(errs...))
}

func (c Config) warnDevelopmentDefaults() {
	if c.RazorpayKeyID == "" || c.RazorpayKeySecret == "" {
		log.Printf("config warning: Razorpay keys not set — use POST /billing/dev/complete for local checkout simulation")
	}
	if c.RazorpayWebhookSecret == "" {
		log.Printf("config warning: RAZORPAY_WEBHOOK_SECRET not set (webhook signature skipped in development)")
	}
}

func isPlaceholderRazorpayCredential(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return true
	}
	for _, marker := range []string{"dummy", "placeholder", "example", "changeme", "your_"} {
		if strings.Contains(v, marker) {
			return true
		}
	}
	return false
}

// CheckoutReady reports whether live Razorpay checkout can be opened in the client.
func (c Config) CheckoutReady() bool {
	id := strings.TrimSpace(c.RazorpayKeyID)
	secret := strings.TrimSpace(c.RazorpayKeySecret)
	if id == "" || secret == "" {
		return false
	}
	if isPlaceholderRazorpayCredential(id) || isPlaceholderRazorpayCredential(secret) {
		return false
	}
	return true
}
