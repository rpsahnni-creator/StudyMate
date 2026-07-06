package notifications

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
)

// Handler handles notification HTTP endpoints
type Handler struct {
	service             *Service
	logger              Logger
	slog                *slog.Logger
	resendWebhookSecret string
	isDevelopment       bool
}

// NewHandler creates a new notification handler
func NewHandler(service *Service, logger Logger, opts ...HandlerOption) *Handler {
	h := &Handler{
		service: service,
		logger:  logger,
		slog:    slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// HandlerOption configures notification HTTP handlers.
type HandlerOption func(*Handler)

// WithResendWebhookSecret sets the signing secret for Resend webhook verification.
func WithResendWebhookSecret(secret string) HandlerOption {
	return func(h *Handler) {
		h.resendWebhookSecret = secret
	}
}

// WithDevelopmentMode skips webhook signature verification in development when secret is unset.
func WithDevelopmentMode(isDevelopment bool) HandlerOption {
	return func(h *Handler) {
		h.isDevelopment = isDevelopment
	}
}

// RegisterRoutes registers notification routes
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/auth/devices/register", h.RegisterDevice)
	router.Get("/user/preferences", h.GetUserPreferences)
	router.Put("/user/preferences", h.UpdateUserPreferences)
	router.Post("/notifications/send", h.SendNotification)
}

// RegisterPublicRoutes registers unauthenticated notification routes (webhooks).
func (h *Handler) RegisterPublicRoutes(router chi.Router) {
	router.Post("/webhooks/email/resend", h.HandleEmailWebhook)
}

// RegisterDeviceRequest represents device registration request
type RegisterDeviceRequest struct {
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	OSVersion  string `json:"os_version"`
}

// RegisterDevice registers a new FCM device token
func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request", nil)
		return
	}

	userID, err := userIDFromContext(r.Context())
	if err != nil {
		h.logger.Error("failed to resolve user id", "error", err)
		apierrors.WriteUnauthorized(w)
		return
	}

	if err := h.service.RegisterDevice(r.Context(), userID, req.Token, req.Platform, nil); err != nil {
		h.logger.Error("failed to register device", "error", err)
		apierrors.WriteInternal(w, h.slog, err, custommw.GetTraceID(r.Context()))
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]string{"message": "device registered"})
}

// GetUserPreferences retrieves user notification preferences
func (h *Handler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDFromContext(r.Context())
	if err != nil {
		apierrors.WriteUnauthorized(w)
		return
	}

	prefs, err := h.service.repo.GetUserPreferences(r.Context(), userID)
	if err != nil {
		apierrors.WriteInternal(w, h.slog, err, custommw.GetTraceID(r.Context()))
		return
	}

	h.jsonResponse(w, http.StatusOK, prefs)
}

// UpdateUserPreferencesRequest represents preference update request
type UpdateUserPreferencesRequest struct {
	PushEnabled     *bool   `json:"push_enabled"`
	EmailEnabled    *bool   `json:"email_enabled"`
	SMSEnabled      *bool   `json:"sms_enabled"`
	MaxPushPerDay   *int    `json:"max_push_per_day"`
	MaxEmailPerWeek *int    `json:"max_email_per_week"`
	QuietHoursStart *string `json:"quiet_hours_start"`
	QuietHoursEnd   *string `json:"quiet_hours_end"`
	QuietHoursTZ    *string `json:"quiet_hours_tz"`
}

// UpdateUserPreferences updates user notification preferences
func (h *Handler) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request", nil)
		return
	}

	userID, err := userIDFromContext(r.Context())
	if err != nil {
		apierrors.WriteUnauthorized(w)
		return
	}

	prefs, err := h.service.repo.GetUserPreferences(r.Context(), userID)
	if err != nil {
		apierrors.WriteInternal(w, h.slog, err, custommw.GetTraceID(r.Context()))
		return
	}

	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.EmailEnabled != nil {
		prefs.EmailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		prefs.SMSEnabled = *req.SMSEnabled
	}
	if req.MaxPushPerDay != nil {
		prefs.MaxPushPerDay = *req.MaxPushPerDay
	}
	if req.MaxEmailPerWeek != nil {
		prefs.MaxEmailPerWeek = *req.MaxEmailPerWeek
	}
	if req.QuietHoursStart != nil {
		prefs.QuietHoursStart = req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		prefs.QuietHoursEnd = req.QuietHoursEnd
	}
	if req.QuietHoursTZ != nil {
		prefs.QuietHoursTZ = *req.QuietHoursTZ
	}

	if err := h.service.repo.UpsertUserPreferences(r.Context(), prefs); err != nil {
		h.logger.Error("failed to update preferences", "error", err)
		apierrors.WriteInternal(w, h.slog, err, custommw.GetTraceID(r.Context()))
		return
	}

	h.jsonResponse(w, http.StatusOK, prefs)
}

// SendNotificationRequest represents notification send request
type SendNotificationRequest struct {
	Channel     string                 `json:"channel"`
	Category    string                 `json:"category"`
	TemplateKey string                 `json:"template_key"`
	Data        map[string]interface{} `json:"data"`
}

// SendNotification sends a notification
func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request", nil)
		return
	}

	userID, err := userIDFromContext(r.Context())
	if err != nil {
		apierrors.WriteUnauthorized(w)
		return
	}

	if err := h.service.SendNotification(r.Context(), userID, req.Channel, req.Category, req.TemplateKey, req.Data); err != nil {
		h.logger.Error("failed to send notification", "error", err)
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	h.jsonResponse(w, http.StatusAccepted, map[string]string{"message": "notification queued"})
}

// ResendWebhookPayload represents Resend webhook payload
type ResendWebhookPayload struct {
	Type            string                 `json:"type"`
	Email           string                 `json:"email"`
	JobID           string                 `json:"job_id"`
	ProviderEventID string                 `json:"provider_event_id"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// HandleEmailWebhook handles email delivery webhooks
func (h *Handler) HandleEmailWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid body", nil)
		return
	}
	defer r.Body.Close()

	headers := ResendWebhookHeaders{
		ID:        r.Header.Get("svix-id"),
		Timestamp: r.Header.Get("svix-timestamp"),
		Signature: r.Header.Get("svix-signature"),
	}

	if h.resendWebhookSecret != "" {
		if !VerifyResendSignature(body, headers, h.resendWebhookSecret) {
			apierrors.WriteError(w, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, "invalid webhook signature", nil)
			return
		}
	} else if !h.isDevelopment {
		apierrors.WriteError(w, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, "webhook secret not configured", nil)
		return
	}

	var payload ResendWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid payload", nil)
		return
	}

	h.logger.Info("email webhook received", "type", payload.Type, "email", payload.Email)

	switch payload.Type {
	case "bounce":
		_ = h.service.emailService.HandleEmailBounce(r.Context(), payload.Email, payload.Type)
	case "complaint":
		_ = h.service.emailService.HandleEmailComplaint(r.Context(), payload.Email)
	case "delivery":
		if jobID, err := uuid.Parse(payload.JobID); err == nil {
			_ = h.service.emailService.HandleEmailDelivery(r.Context(), jobID)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
