package billing

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"studyapp/backend/internal/common"
	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
)

type Handler struct {
	service *Service
	db      *pgxpool.Pool
	logger  *slog.Logger
}

func NewHandler(service *Service, db *pgxpool.Pool) *Handler {
	return &Handler{service: service, db: db, logger: slog.Default()}
}

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/plans", h.GetPlans)
	r.Post("/billing/webhook/razorpay", h.HandleRazorpayWebhook)
	r.Post("/billing/webhook/payu", h.HandlePayUWebhook)
}

func (h *Handler) RegisterAuthRoutes(r chi.Router) {
	r.Post("/billing/checkout", h.CreateCheckout)
	r.Get("/users/me/subscription", h.GetMySubscription)
}

// RegisterDevRoutes mounts development-only billing helpers.
func (h *Handler) RegisterDevRoutes(r chi.Router) {
	r.Post("/billing/dev/complete", h.CompleteDevCheckout)
}

func (h *Handler) CompleteDevCheckout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	var req struct {
		OrderID string `json:"orderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if req.OrderID == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "orderId is required", nil)
		return
	}

	if err := h.service.CompleteDevCheckout(r.Context(), userID, req.OrderID); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodePaymentFailed, err.Error(), nil)
		return
	}

	ent, err := h.service.GetMySubscription(r.Context(), userID)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.writeJSON(w, http.StatusOK, ent)
}

func (h *Handler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.ListPlans(r.Context())
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.writeJSON(w, http.StatusOK, plans)
}

func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if req.PlanID == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "planId is required", nil)
		return
	}

	resp, err := h.service.CreateCheckout(r.Context(), userID, req)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodePaymentFailed, err.Error(), nil)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMySubscription(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	ent, err := h.service.GetMySubscription(r.Context(), userID)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.writeJSON(w, http.StatusOK, ent)
}

func (h *Handler) HandleRazorpayWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid payload", nil)
		return
	}
	err = h.service.HandleRazorpayWebhook(r.Context(), body, ParseRazorpaySignature(r), ParseRazorpayEventID(r))
	if errors.Is(err, ErrInvalidWebhookSignature) {
		h.logPaymentAudit(r, false, "razorpay", map[string]any{"reason": "invalid_signature"})
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid webhook signature", nil)
		return
	}
	if errors.Is(err, ErrEventAlreadyHandled) {
		h.writeJSON(w, http.StatusOK, map[string]string{"status": "already_processed"})
		return
	}
	if err != nil {
		h.logPaymentAudit(r, false, "razorpay", map[string]any{"reason": "processing_failed"})
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodePaymentFailed, "webhook processing failed", nil)
		return
	}
	h.logPaymentAudit(r, true, "razorpay", map[string]any{"event_id": ParseRazorpayEventID(r)})
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (h *Handler) HandlePayUWebhook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid form payload", nil)
		return
	}
	body, _ := json.Marshal(r.Form)
	err := h.service.HandlePayUWebhook(r.Context(), body, ParsePayUHash(r))
	if errors.Is(err, ErrInvalidWebhookSignature) {
		h.logPaymentAudit(r, false, "payu", map[string]any{"reason": "invalid_signature"})
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid webhook signature", nil)
		return
	}
	if errors.Is(err, ErrEventAlreadyHandled) {
		h.writeJSON(w, http.StatusOK, map[string]string{"status": "already_processed"})
		return
	}
	if err != nil {
		h.logPaymentAudit(r, false, "payu", map[string]any{"reason": "processing_failed"})
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodePaymentFailed, "webhook processing failed", nil)
		return
	}
	h.logPaymentAudit(r, true, "payu", nil)
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
}

func (h *Handler) logPaymentAudit(r *http.Request, success bool, provider string, details map[string]any) {
	if h.db == nil {
		return
	}
	if details == nil {
		details = map[string]any{}
	}
	details["provider"] = provider
	_ = common.LogAuditEvent(r.Context(), h.db, common.AuditEvent{
		Action:    "payment",
		Resource:  "payment/webhook",
		IPAddress: custommw.ClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   success,
		Details:   details,
	})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
