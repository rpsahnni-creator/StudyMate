package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"studyapp/backend/internal/common/metrics"
)

var ErrInvalidWebhookSignature = errors.New("invalid webhook signature")

// HandleRazorpayWebhook processes Razorpay payment events with signature verification and idempotency.
func (s *Service) HandleRazorpayWebhook(ctx context.Context, payload []byte, signature, eventID string) error {
	if err := s.verifyRazorpaySignature(payload, signature); err != nil {
		return err
	}

	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	if eventID == "" {
		eventID = razorpayEventID(event)
	}
	if eventID == "" {
		return fmt.Errorf("missing razorpay event id")
	}

	exists, err := s.repo.EventExists(ctx, eventID)
	if err != nil {
		return err
	}
	if exists {
		metrics.RecordPaymentEvent(ProviderRazorpay, "already_processed")
		return ErrEventAlreadyHandled
	}

	eventType, _ := event["event"].(string)
	eventRowID, err := s.repo.InsertPaymentEvent(ctx, eventID, ProviderRazorpay, eventType, payload, nil)
	if err != nil {
		return err
	}

	switch eventType {
	case "payment.captured":
		if err := s.handleRazorpayPaymentCaptured(ctx, event); err != nil {
			return err
		}
	case "payment.failed":
		if err := s.handleRazorpayPaymentFailed(ctx, event); err != nil {
			return err
		}
	case "subscription.cancelled":
		if err := s.handleRazorpaySubscriptionCancelled(ctx, event); err != nil {
			return err
		}
	}

	if err := s.repo.MarkPaymentEventProcessed(ctx, eventRowID); err != nil {
		return err
	}
	metrics.RecordPaymentEvent(ProviderRazorpay, paymentEventStatus(eventType))
	return nil
}

func (s *Service) verifyRazorpaySignature(payload []byte, signature string) error {
	secret := s.cfg.RazorpayWebhookSecret
	if secret == "" {
		if s.cfg.IsDevelopment() {
			return nil
		}
		return fmt.Errorf("RAZORPAY_WEBHOOK_SECRET is required")
	}
	if strings.TrimSpace(signature) == "" {
		return ErrInvalidWebhookSignature
	}
	expected := hmacSHA256(payload, secret)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return ErrInvalidWebhookSignature
	}
	return nil
}

func (s *Service) handleRazorpayPaymentCaptured(ctx context.Context, event map[string]any) error {
	orderID := razorpayOrderID(event)
	paymentID := razorpayPaymentID(event)
	if orderID == "" {
		return fmt.Errorf("payment.captured missing order id")
	}

	payment, err := s.repo.GetPaymentByProviderOrderID(ctx, ProviderRazorpay, orderID)
	if err != nil {
		return err
	}
	if payment.Status == PaymentStatusCompleted {
		return nil
	}

	paidAt := time.Now()
	if err := s.repo.MarkPaymentCompleted(ctx, payment.ID, paymentID, paidAt); err != nil {
		return err
	}

	if err := s.repo.UpsertActiveSubscription(ctx, payment.UserID, payment.PlanID, ProviderRazorpay, paymentID); err != nil {
		return err
	}

	return s.notifyPaymentCaptured(ctx, payment.UserID, payment.PlanID)
}

func (s *Service) handleRazorpayPaymentFailed(ctx context.Context, event map[string]any) error {
	orderID := razorpayOrderID(event)
	if orderID == "" {
		return fmt.Errorf("payment.failed missing order id")
	}
	payment, err := s.repo.GetPaymentByProviderOrderID(ctx, ProviderRazorpay, orderID)
	if err != nil {
		return err
	}
	if err := s.repo.MarkPaymentFailed(ctx, payment.ID); err != nil {
		return err
	}
	if s.notifier != nil {
		_ = s.notifier.PaymentFailed(ctx, payment.UserID)
	}
	return nil
}

func (s *Service) handleRazorpaySubscriptionCancelled(ctx context.Context, event map[string]any) error {
	userID := razorpayUserIDFromNotes(event)
	if userID <= 0 {
		return nil
	}
	return s.repo.CancelSubscription(ctx, userID)
}

// HandlePayUWebhook processes PayU callbacks with hash verification and idempotency.
func (s *Service) HandlePayUWebhook(ctx context.Context, payload []byte, receivedHash string) error {
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	if err := s.verifyPayUHash(event, receivedHash); err != nil {
		return err
	}

	txnID, _ := event["txnid"].(string)
	if txnID == "" {
		return fmt.Errorf("missing payu txnid")
	}

	exists, err := s.repo.EventExists(ctx, txnID)
	if err != nil {
		return err
	}
	if exists {
		metrics.RecordPaymentEvent(ProviderPayU, "already_processed")
		return ErrEventAlreadyHandled
	}

	status, _ := event["status"].(string)
	eventRowID, err := s.repo.InsertPaymentEvent(ctx, txnID, ProviderPayU, "payu."+status, payload, nil)
	if err != nil {
		return err
	}

	payment, err := s.repo.GetPaymentByProviderOrderID(ctx, ProviderPayU, txnID)
	if err != nil {
		return err
	}

	switch strings.ToLower(status) {
	case "success":
		paymentRef, _ := event["mihpayid"].(string)
		if err := s.repo.MarkPaymentCompleted(ctx, payment.ID, paymentRef, time.Now()); err != nil {
			return err
		}
		if err := s.repo.UpsertActiveSubscription(ctx, payment.UserID, payment.PlanID, ProviderPayU, paymentRef); err != nil {
			return err
		}
		if err := s.notifyPaymentCaptured(ctx, payment.UserID, payment.PlanID); err != nil {
			return err
		}
	case "failure", "failed":
		if err := s.repo.MarkPaymentFailed(ctx, payment.ID); err != nil {
			return err
		}
		if s.notifier != nil {
			_ = s.notifier.PaymentFailed(ctx, payment.UserID)
		}
	}

	if err := s.repo.MarkPaymentEventProcessed(ctx, eventRowID); err != nil {
		return err
	}
	metrics.RecordPaymentEvent(ProviderPayU, paymentEventStatus(status))
	return nil
}

func paymentEventStatus(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "payment.captured", "success":
		return "completed"
	case "payment.failed", "failure", "failed":
		return "failed"
	case "subscription.cancelled":
		return "cancelled"
	default:
		if raw == "" {
			return "unknown"
		}
		return raw
	}
}

func (s *Service) verifyPayUHash(event map[string]any, receivedHash string) error {
	if s.cfg.PayUSalt == "" {
		if s.cfg.IsDevelopment() {
			return nil
		}
		return fmt.Errorf("PAYU_SALT is required")
	}
	if receivedHash == "" && !s.cfg.IsDevelopment() {
		return ErrInvalidWebhookSignature
	}
	if receivedHash == "" {
		return nil
	}

	status, _ := event["status"].(string)
	amount, _ := event["amount"].(string)
	txnid, _ := event["txnid"].(string)
	email, _ := event["email"].(string)
	firstname, _ := event["firstname"].(string)
	productinfo, _ := event["productinfo"].(string)
	udf1, _ := event["udf1"].(string)
	udf2, _ := event["udf2"].(string)
	key := s.cfg.PayUKey

	seq := strings.Join([]string{
		s.cfg.PayUSalt, status, "", "", "", "", "", "", "", "", "", "",
		udf2, udf1, email, firstname, productinfo, amount, txnid, key,
	}, "|")
	expected := sha512Hex(seq)
	if !hmac.Equal([]byte(strings.ToLower(receivedHash)), []byte(strings.ToLower(expected))) {
		return ErrInvalidWebhookSignature
	}
	return nil
}

func razorpayEventID(event map[string]any) string {
	if id, ok := event["id"].(string); ok {
		return id
	}
	return ""
}

func razorpayOrderID(event map[string]any) string {
	payload, _ := event["payload"].(map[string]any)
	payment, _ := payload["payment"].(map[string]any)
	entity, _ := payment["entity"].(map[string]any)
	if orderID, ok := entity["order_id"].(string); ok {
		return orderID
	}
	order, _ := payload["order"].(map[string]any)
	orderEntity, _ := order["entity"].(map[string]any)
	if orderID, ok := orderEntity["id"].(string); ok {
		return orderID
	}
	return ""
}

func razorpayPaymentID(event map[string]any) string {
	payload, _ := event["payload"].(map[string]any)
	payment, _ := payload["payment"].(map[string]any)
	entity, _ := payment["entity"].(map[string]any)
	if id, ok := entity["id"].(string); ok {
		return id
	}
	return ""
}

func razorpayUserIDFromNotes(event map[string]any) int64 {
	payload, _ := event["payload"].(map[string]any)
	sub, _ := payload["subscription"].(map[string]any)
	entity, _ := sub["entity"].(map[string]any)
	notes, _ := entity["notes"].(map[string]any)
	if notes == nil {
		return 0
	}
	if raw, ok := notes["user_id"].(float64); ok {
		return int64(raw)
	}
	return 0
}

func hmacSHA256(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func ParseRazorpaySignature(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Razorpay-Signature"))
}

func ParseRazorpayEventID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Razorpay-Event-Id"))
}

func ParsePayUHash(r *http.Request) string {
	return strings.TrimSpace(r.FormValue("hash"))
}

func (s *Service) notifyPaymentCaptured(ctx context.Context, userID, planID int64) error {
	if s.notifier == nil {
		return nil
	}

	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil {
		plan.Name = "Premium"
	}

	expires := time.Now().Add(30 * 24 * time.Hour).Format("2 Jan 2006")
	if ent, err := s.repo.GetEntitlements(ctx, userID); err == nil && ent.ExpiresAt != nil {
		expires = ent.ExpiresAt.Format("2 Jan 2006")
	}

	return s.notifier.PaymentCaptured(ctx, userID, plan.Name, expires)
}
