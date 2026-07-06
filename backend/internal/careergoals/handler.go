package careergoals

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
	"studyapp/backend/internal/featureflags"
)

// Handler serves the career goals REST API. Discovery (GET /goals) is public;
// every other route is expected to be mounted behind RequireAuth +
// RequireCareerGoalsFlag by the caller (see cmd/api/main.go).
type Handler struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	flags  *featureflags.Service
	logger *slog.Logger
}

// NewHandler wires the module. The flag service is retained so callers can build
// the gating middleware; the cache is available for future response caching.
func NewHandler(db *pgxpool.Pool, cache *redis.Client, flags *featureflags.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{db: db, cache: cache, flags: flags, logger: logger}
}

// RegisterRoutes mounts the discovery route (public) and the flag-gated routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/goals", h.ListGoals)

	r.Group(func(r chi.Router) {
		r.Use(RequireCareerGoalsFlag(h.flags))
		r.Post("/goals/select", h.SelectGoal)
		r.Get("/goals/my", h.GetMyGoal)
		r.Delete("/goals/my", h.AbandonGoal)
		r.Get("/goals/my/practice/today", h.GetTodayPractice)
		r.Post("/goals/my/practice/{setId}/submit", h.SubmitPractice)
		r.Get("/goals/my/practice/history", h.GetPracticeHistory)
		r.Get("/goals/my/skills", h.GetSkillGaps)
	})
}

func (h *Handler) json(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, message string) {
	code := apierrors.ErrCodeValidation
	switch status {
	case http.StatusUnauthorized:
		code = apierrors.ErrCodeUnauthorized
	case http.StatusForbidden:
		code = apierrors.ErrCodeForbidden
	case http.StatusNotFound:
		code = apierrors.ErrCodeNotFound
	case http.StatusInternalServerError:
		code = apierrors.ErrCodeInternalError
	}
	apierrors.WriteError(w, status, code, message, nil)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	traceID := custommw.GetTraceID(r.Context())
	switch {
	case errors.Is(err, ErrNoActiveGoal):
		apierrors.WriteNotFound(w, "active goal")
	case errors.Is(err, ErrGoalNotFound):
		apierrors.WriteNotFound(w, "career goal")
	case errors.Is(err, ErrSetNotFound):
		apierrors.WriteNotFound(w, "practice set")
	case errors.Is(err, ErrSetForbidden):
		apierrors.WriteForbidden(w, err.Error())
	case errors.Is(err, ErrInvalidInput):
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
	default:
		apierrors.WriteInternal(w, h.logger, err, traceID)
	}
}

func (h *Handler) userID(r *http.Request) (int64, bool) {
	switch v := r.Context().Value("user_id").(type) {
	case int64:
		return v, true
	default:
		return 0, false
	}
}
