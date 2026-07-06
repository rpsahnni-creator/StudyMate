package billing

import (
	"encoding/json"
	"time"
)

const (
	ProviderRazorpay = "razorpay"
	ProviderPayU     = "payu"

	PaymentStatusPending   = "pending"
	PaymentStatusCompleted = "completed"
	PaymentStatusFailed    = "failed"

	SubStatusActive    = "active"
	SubStatusCancelled = "cancelled"
	SubStatusInactive  = "inactive"

	FreeScansPerDay  = 5
	BasicScansPerDay = 10
)

type Plan struct {
	ID           int64           `json:"-"`
	Slug         string          `json:"id"`
	Name         string          `json:"name"`
	PricePaise   int64           `json:"price"`
	Currency     string          `json:"currency"`
	Interval     string          `json:"interval"`
	Features     json.RawMessage `json:"features"`
	IsPopular    bool            `json:"isPopular"`
	ScanLimit    int             `json:"-"`
	Active       bool            `json:"-"`
	CreatedAt    time.Time       `json:"-"`
}

type CheckoutRequest struct {
	PlanID   string `json:"planId"`
	Provider string `json:"provider"`
}

type CheckoutPrefill struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type CheckoutResponse struct {
	OrderID  string           `json:"orderId"`
	Amount   int64            `json:"amount"`
	Currency string           `json:"currency"`
	KeyID    string           `json:"keyId,omitempty"`
	Prefill  *CheckoutPrefill `json:"prefill,omitempty"`
	Provider string           `json:"provider"`
	PayuURL  string           `json:"payuUrl,omitempty"`
	FormData map[string]string `json:"formData,omitempty"`
	DevMode  bool             `json:"devMode,omitempty"`
}

type Entitlements struct {
	HasActiveSub  bool       `json:"has_active_sub"`
	Plan          string     `json:"plan"`
	ScansPerDay   int        `json:"scans_per_day"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	DaysRemaining int        `json:"days_remaining"`
}

type UserProfile struct {
	ID    int64
	Name  string
	Email string
}

type PaymentRecord struct {
	ID               int64
	UserID           int64
	PlanID           int64
	Provider         string
	ProviderOrderID  string
	AmountPaise      int64
	Currency         string
	Status           string
}
