package billing

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// CreateCheckout starts a payment session for the authenticated user.
func (s *Service) CreateCheckout(ctx context.Context, userID int64, req CheckoutRequest) (CheckoutResponse, error) {
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = ProviderRazorpay
	}

	plan, err := s.repo.GetPlanBySlug(ctx, strings.TrimSpace(req.PlanID))
	if err != nil {
		return CheckoutResponse{}, err
	}

	user, err := s.repo.GetUserProfile(ctx, userID)
	if err != nil {
		return CheckoutResponse{}, err
	}

	switch provider {
	case ProviderPayU:
		return s.createPayUCheckout(ctx, user, plan)
	case ProviderRazorpay:
		return s.createRazorpayCheckout(ctx, user, plan)
	default:
		return CheckoutResponse{}, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (s *Service) createRazorpayCheckout(ctx context.Context, user UserProfile, plan Plan) (CheckoutResponse, error) {
	receipt := fmt.Sprintf("order_%s", uuid.New().String())
	orderID, err := s.createRazorpayOrder(ctx, plan.PricePaise, receipt, user.ID, plan.Slug)
	if err != nil {
		return CheckoutResponse{}, err
	}

	if _, err := s.repo.CreatePendingPayment(ctx, user.ID, plan.ID, ProviderRazorpay, orderID, plan.PricePaise, plan.Currency); err != nil {
		return CheckoutResponse{}, err
	}

	resp := CheckoutResponse{
		OrderID:  orderID,
		Amount:   plan.PricePaise,
		Currency: plan.Currency,
		KeyID:    s.cfg.RazorpayKeyID,
		Provider: ProviderRazorpay,
		Prefill: &CheckoutPrefill{
			Name:  user.Name,
			Email: user.Email,
		},
	}
	if s.cfg.IsDevelopment() && !s.cfg.CheckoutReady() {
		resp.DevMode = true
	}
	return resp, nil
}

func (s *Service) createRazorpayOrder(ctx context.Context, amountPaise int64, receipt string, userID int64, planSlug string) (string, error) {
	if s.cfg.IsDevelopment() && !s.cfg.CheckoutReady() {
		return fmt.Sprintf("dev_order_%d_%s", userID, planSlug), nil
	}
	if s.cfg.RazorpayKeySecret == "" {
		return "", fmt.Errorf("RAZORPAY_KEY_SECRET is required for checkout")
	}
	if strings.TrimSpace(s.cfg.RazorpayKeyID) == "" {
		return "", fmt.Errorf("RAZORPAY_KEY_ID is required for checkout")
	}

	body := map[string]any{
		"amount":   amountPaise,
		"currency": "INR",
		"receipt":  receipt,
		"notes": map[string]any{
			"user_id": userID,
			"plan_id": planSlug,
		},
	}
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/orders", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(s.cfg.RazorpayKeyID, s.cfg.RazorpayKeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("razorpay order failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if parsed.ID == "" {
		return "", fmt.Errorf("razorpay returned empty order id")
	}
	return parsed.ID, nil
}

func (s *Service) createPayUCheckout(ctx context.Context, user UserProfile, plan Plan) (CheckoutResponse, error) {
	if s.cfg.PayUKey == "" || s.cfg.PayUSalt == "" {
		if s.cfg.IsDevelopment() {
			txnID := fmt.Sprintf("dev_payu_%s", uuid.New().String())
			_, _ = s.repo.CreatePendingPayment(ctx, user.ID, plan.ID, ProviderPayU, txnID, plan.PricePaise, plan.Currency)
			return CheckoutResponse{
				OrderID:  txnID,
				Amount:   plan.PricePaise,
				Currency: plan.Currency,
				Provider: ProviderPayU,
				PayuURL:  "https://test.payu.in/_payment",
				FormData: map[string]string{"txnid": txnID},
			}, nil
		}
		return CheckoutResponse{}, fmt.Errorf("PAYU_KEY and PAYU_SALT are required for PayU checkout")
	}

	txnID := uuid.New().String()
	amountStr := fmt.Sprintf("%.2f", float64(plan.PricePaise)/100.0)
	productInfo := plan.Name + " subscription"
	firstname := user.Name
	if firstname == "" {
		firstname = "User"
	}
	email := user.Email
	udf1 := fmt.Sprintf("%d", user.ID)
	udf2 := plan.Slug

	hashSeq := strings.Join([]string{
		s.cfg.PayUKey, txnID, amountStr, productInfo, firstname, email,
		udf1, udf2, "", "", "", "", "", "", "", "", s.cfg.PayUSalt,
	}, "|")
	hash := sha512Hex(hashSeq)

	if _, err := s.repo.CreatePendingPayment(ctx, user.ID, plan.ID, ProviderPayU, txnID, plan.PricePaise, plan.Currency); err != nil {
		return CheckoutResponse{}, err
	}

	form := map[string]string{
		"key":         s.cfg.PayUKey,
		"txnid":       txnID,
		"amount":      amountStr,
		"productinfo": productInfo,
		"firstname":   firstname,
		"email":       email,
		"phone":       "",
		"surl":        s.cfg.PayUWebhookURL,
		"furl":        s.cfg.PayUWebhookURL,
		"hash":        hash,
		"udf1":        udf1,
		"udf2":        udf2,
	}

	return CheckoutResponse{
		OrderID:  txnID,
		Amount:   plan.PricePaise,
		Currency: plan.Currency,
		Provider: ProviderPayU,
		PayuURL:  "https://secure.payu.in/_payment",
		FormData: form,
	}, nil
}

func sha512Hex(value string) string {
	sum := sha512.Sum512([]byte(value))
	return hex.EncodeToString(sum[:])
}
