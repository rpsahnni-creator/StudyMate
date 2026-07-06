package billing

import (
	"context"
	"strings"
	"testing"
)

type devCheckoutRepo struct {
	fakeEntitlementsRepo
	payment PaymentRecord
}

func (r *devCheckoutRepo) GetPaymentByProviderOrderID(context.Context, string, string) (PaymentRecord, error) {
	return r.payment, nil
}

func TestCompleteDevCheckout_RejectsOutsideDevelopment(t *testing.T) {
	svc := &Service{
		cfg: Config{Environment: "production"},
		repo: &devCheckoutRepo{
			payment: PaymentRecord{UserID: 1, ProviderOrderID: "dev_order_1_plan"},
		},
	}
	err := svc.CompleteDevCheckout(context.Background(), 1, "dev_order_1_plan")
	if err == nil || !strings.Contains(err.Error(), "development") {
		t.Fatalf("expected development guard, got: %v", err)
	}
}

func TestCompleteDevCheckout_ActivatesPendingOrder(t *testing.T) {
	repo := &devCheckoutRepo{
		payment: PaymentRecord{
			ID:              10,
			UserID:          42,
			PlanID:          2,
			Provider:        ProviderRazorpay,
			ProviderOrderID: "dev_order_42_plan_basic_monthly",
			Status:          PaymentStatusPending,
		},
	}
	svc := &Service{
		cfg:  Config{Environment: "development"},
		repo: repo,
	}
	if err := svc.CompleteDevCheckout(context.Background(), 42, "dev_order_42_plan_basic_monthly"); err != nil {
		t.Fatalf("CompleteDevCheckout failed: %v", err)
	}
}

func TestCompleteDevCheckout_RejectsWrongUser(t *testing.T) {
	svc := &Service{
		cfg: Config{Environment: "development"},
		repo: &devCheckoutRepo{
			payment: PaymentRecord{UserID: 99, ProviderOrderID: "dev_order_42_plan_basic_monthly"},
		},
	}
	err := svc.CompleteDevCheckout(context.Background(), 42, "dev_order_42_plan_basic_monthly")
	if err == nil || !strings.Contains(err.Error(), "authenticated user") {
		t.Fatalf("expected ownership error, got: %v", err)
	}
}
