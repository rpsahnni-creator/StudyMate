package billing

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const devOrderPrefix = "dev_order_"

// CompleteDevCheckout finalizes a pending dev/stub order without Razorpay (development only).
func (s *Service) CompleteDevCheckout(ctx context.Context, userID int64, orderID string) error {
	if !s.cfg.IsDevelopment() {
		return fmt.Errorf("dev checkout completion is only available in development")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("orderId is required")
	}
	if !strings.HasPrefix(orderID, devOrderPrefix) && !s.cfg.CheckoutReady() {
		return fmt.Errorf("order %q is not a dev stub order", orderID)
	}

	payment, err := s.repo.GetPaymentByProviderOrderID(ctx, ProviderRazorpay, orderID)
	if err != nil {
		return err
	}
	if payment.UserID != userID {
		return fmt.Errorf("order does not belong to the authenticated user")
	}
	if payment.Status == PaymentStatusCompleted {
		return nil
	}

	paymentID := fmt.Sprintf("dev_pay_%d", time.Now().UnixNano())
	if err := s.repo.MarkPaymentCompleted(ctx, payment.ID, paymentID, time.Now()); err != nil {
		return fmt.Errorf("mark payment completed: %w", err)
	}
	if err := s.repo.UpsertActiveSubscription(ctx, payment.UserID, payment.PlanID, ProviderRazorpay, paymentID); err != nil {
		return fmt.Errorf("activate subscription: %w", err)
	}
	return s.notifyPaymentCaptured(ctx, payment.UserID, payment.PlanID)
}
