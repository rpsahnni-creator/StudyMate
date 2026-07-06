package featureflags

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, logger: slog.Default()}
}

func userIDFromContext(r *http.Request) string {
	switch v := r.Context().Value("user_id").(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		return v
	default:
		return ""
	}
}

func (h *Handler) GetMyFeatures(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	if userID == "" {
		apierrors.WriteUnauthorized(w)
		return
	}

	flags, err := h.service.ResolveForUser(r.Context(), userID)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flags)
}

func (h *Handler) AdminListFlags(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.ListWithStats(r.Context())
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

type AdminSetFlagRequest struct {
	Key               FlagKey `json:"key"`
	Enabled           bool    `json:"enabled"`
	RolloutPercentage int     `json:"rollout_percentage"`
}

func (h *Handler) AdminSetFlag(w http.ResponseWriter, r *http.Request) {
	adminID := userIDFromContext(r)

	var req AdminSetFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request body", nil)
		return
	}

	if err := h.service.SetFlag(r.Context(), req.Key, req.Enabled, req.RolloutPercentage, adminID); err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}

	w.WriteHeader(http.StatusOK)
}
