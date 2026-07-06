package quiz

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
)

// Handler exposes the quiz REST API. All routes assume RequireAuth ran first.
type Handler struct {
	svc    *Service
	logger *slog.Logger
}

func NewHandler(svc *Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes mounts quiz routes on an already-authenticated router group.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/quizzes", func(r chi.Router) {
		r.Get("/{quizID}", h.GetQuiz)
		r.Post("/{quizID}/attempts", h.CreateAttempt)
		r.Put("/{quizID}/attempts/{attemptID}/submit", h.SubmitAttempt)
		r.Get("/{quizID}/attempts/{attemptID}/review", h.GetReview)
	})
	r.Get("/users/me/reports", h.GetReports)
	r.Get("/users/me/analytics", h.GetAnalytics)
	r.Get("/users/me/analytics/topics", h.GetTopicAnalytics)
}

func (h *Handler) GetQuiz(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	quizID, err := pathInt(r, "quizID")
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid quiz id", nil)
		return
	}

	detail, err := h.svc.GetQuiz(r.Context(), quizID, userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, detail)
}

func (h *Handler) CreateAttempt(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	quizID, err := pathInt(r, "quizID")
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid quiz id", nil)
		return
	}

	var req CreateAttemptRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON body", nil)
			return
		}
	}
	if err := req.Validate(); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	attempt, err := h.svc.CreateAttempt(r.Context(), quizID, userID, req.StartedAt)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusCreated, CreateAttemptResponse{
		AttemptID: attempt.ID,
		ExpiresAt: attempt.ExpiresAt,
	})
}

func (h *Handler) SubmitAttempt(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	attemptID, err := pathInt(r, "attemptID")
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid attempt id", nil)
		return
	}

	var req SubmitAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON body", nil)
		return
	}
	if err := req.Validate(); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	result, err := h.svc.SubmitAttempt(r.Context(), attemptID, userID, req.Answers, req.SubmittedAt)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, result)
}

func (h *Handler) GetReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	attemptID, err := pathInt(r, "attemptID")
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid attempt id", nil)
		return
	}

	review, err := h.svc.GetReview(r.Context(), attemptID, userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, review)
}

func (h *Handler) GetReports(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 10)

	reports, err := h.svc.GetUserReports(r.Context(), userID, page, limit)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, reports)
}

func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	analytics, err := h.svc.GetAnalytics(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, analytics)
}

func (h *Handler) GetTopicAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}
	topics, err := h.svc.GetTopicAnalytics(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, r, err)
		return
	}
	h.json(w, http.StatusOK, topics)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	traceID := custommw.GetTraceID(r.Context())
	switch {
	case errors.Is(err, ErrQuizNotFound):
		apierrors.WriteNotFound(w, "quiz")
	case errors.Is(err, ErrAttemptNotFound):
		apierrors.WriteNotFound(w, "attempt")
	case errors.Is(err, ErrAttemptForbidden):
		apierrors.WriteForbidden(w, err.Error())
	case errors.Is(err, ErrAttemptNotDone):
		apierrors.WriteError(w, http.StatusConflict, apierrors.ErrCodeConflict, err.Error(), nil)
	default:
		apierrors.WriteInternal(w, h.logger, err, traceID)
	}
}

func (h *Handler) json(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func userIDFromContext(r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value("user_id").(int64)
	return userID, ok
}

func pathInt(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, key), 10, 64)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
