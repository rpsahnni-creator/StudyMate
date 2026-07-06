package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"studyapp/backend/internal/common"
	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
)

// Handler provides HTTP handlers for authentication endpoints.
type Handler struct {
	service Service
	db      *pgxpool.Pool
}

// NewHandler creates a new authentication handler.
func NewHandler(service Service, db *pgxpool.Pool) *Handler {
	return &Handler{service: service, db: db}
}

// RegisterRoutes registers all auth routes on the router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/register/send-otp", h.SendRegistrationOTP)
	r.Post("/auth/register/verify-otp", h.VerifyRegistrationOTP)
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.RefreshToken)
	r.Post("/auth/forgot-password", h.ForgotPassword)
	r.Post("/auth/reset-password", h.ResetPassword)
}

func (h *Handler) SendRegistrationOTP(w http.ResponseWriter, r *http.Request) {
	var req SendRegistrationOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "email is required", nil)
		return
	}
	resp, err := h.service.SendRegistrationOTP(r.Context(), req.Email)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}
	h.jsonResponse(w, http.StatusOK, resp)
}

func (h *Handler) VerifyRegistrationOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyRegistrationOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.OTP) == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "email and otp are required", nil)
		return
	}
	resp, err := h.service.VerifyRegistrationOTP(r.Context(), &req)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}
	h.jsonResponse(w, http.StatusOK, resp)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if custommw.HandleBodyTooLarge(w, err) {
			return
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request body", nil)
		return
	}
	defer r.Body.Close()

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}

	req.Name = SanitizeUserInput(req.Name)
	if req.VerificationToken == "" || req.Name == "" || req.Email == "" || req.Class == "" || req.Mobile == "" || req.Password == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "missing required fields", nil)
		return
	}

	token, err := h.service.Register(r.Context(), &req)
	if err != nil {
		h.logAudit(r, "", "register", "user", false, map[string]any{"email": req.Email})
		apierrors.WriteError(w, http.StatusConflict, apierrors.ErrCodeConflict, err.Error(), nil)
		return
	}

	h.logAudit(r, strconv.FormatInt(token.User.ID, 10), "register", "user/"+strconv.FormatInt(token.User.ID, 10), true, nil)
	h.jsonResponse(w, http.StatusCreated, token)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if custommw.HandleBodyTooLarge(w, err) {
			return
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request body", nil)
		return
	}
	defer r.Body.Close()

	var req LoginRequest
	if err := json.Unmarshal(body, &req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}

	if req.Email == "" || req.Password == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "email and password are required", nil)
		return
	}

	token, err := h.service.Login(r.Context(), &req)
	if err != nil {
		h.logAudit(r, "", "login", "user", false, map[string]any{"email": req.Email})
		apierrors.WriteError(w, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, "invalid email or password", nil)
		return
	}

	h.logAudit(r, strconv.FormatInt(token.User.ID, 10), "login", "user/"+strconv.FormatInt(token.User.ID, 10), true, nil)
	h.jsonResponse(w, http.StatusOK, token)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if custommw.HandleBodyTooLarge(w, err) {
			return
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request body", nil)
		return
	}
	defer r.Body.Close()

	var req RefreshTokenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}

	token, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		apierrors.WriteUnauthorized(w)
		return
	}

	h.jsonResponse(w, http.StatusOK, token)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if err := h.service.InitiatePasswordReset(r.Context(), req.Email); err != nil {
		h.logAudit(r, "", "password_reset_request", "user", false, map[string]any{"email": req.Email})
	} else {
		h.logAudit(r, "", "password_reset_request", "user", true, map[string]any{"email": req.Email})
	}
	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "if the account exists, a reset link has been sent"})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Token, req.NewPassword, req.NewPasswordConfirm); err != nil {
		h.logAudit(r, "", "password_reset", "user", false, nil)
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}
	h.logAudit(r, "", "password_reset", "user", true, nil)
	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "password reset successfully"})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if custommw.HandleBodyTooLarge(w, err) {
			return
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request body", nil)
		return
	}
	defer r.Body.Close()

	var req ChangePasswordRequest
	if err := json.Unmarshal(body, &req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}

	if err := h.service.ChangePassword(r.Context(), userID, &req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		apierrors.WriteNotFound(w, "user")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"id":             user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"role":           user.Role,
		"status":         user.Status,
		"email_verified": user.EmailVerified,
	})
}

func (h *Handler) logAudit(r *http.Request, userID, action, resource string, success bool, details map[string]any) {
	if h.db == nil {
		return
	}
	_ = common.LogAuditEvent(r.Context(), h.db, common.AuditEvent{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		IPAddress: custommw.ClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   success,
		Details:   details,
	})
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
