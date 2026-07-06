package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// RegisterRoutes mounts admin endpoints. Caller must apply RequireAuth + RequireAdmin.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/users", h.GetUsers)
	r.Put("/admin/users/{userID}/suspend", h.SuspendUser)
	r.Put("/admin/users/{userID}/unsuspend", h.UnsuspendUser)
	r.Get("/admin/jobs", h.GetJobs)
	r.Post("/admin/jobs/{jobID}/retry", h.RetryJob)
	r.Get("/admin/ai-costs", h.GetAICosts)
	r.Get("/admin/audit-logs", h.GetAuditLogs)
	r.Get("/admin/content-flags", h.GetContentFlags)
	r.Put("/admin/content-flags/{flagID}/resolve", h.ResolveContentFlag)
}

func adminIDFromContext(r *http.Request) int64 {
	if v, ok := r.Context().Value("user_id").(int64); ok {
		return v
	}
	return 0
}

func queryInt(r *http.Request, key string, fallback int) int {
	if raw := strings.TrimSpace(r.URL.Query().Get(key)); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			return v
		}
	}
	return fallback
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)

	result, err := h.service.GetUsers(r.Context(), search, page, limit)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type suspendRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	adminID := adminIDFromContext(r)
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid user id", nil)
		return
	}
	if userID == adminID {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "cannot suspend your own account", nil)
		return
	}
	var req suspendRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Reason) == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "reason is required", nil)
		return
	}
	if err := h.service.SuspendUser(r.Context(), adminID, userID, req.Reason); err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.logAdminAction(r, adminID, "admin_action", "user/"+strconv.FormatInt(userID, 10), true, map[string]any{"action": "suspend"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) UnsuspendUser(w http.ResponseWriter, r *http.Request) {
	adminID := adminIDFromContext(r)
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid user id", nil)
		return
	}
	if err := h.service.UnsuspendUser(r.Context(), adminID, userID); err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.logAdminAction(r, adminID, "admin_action", "user/"+strconv.FormatInt(userID, 10), true, map[string]any{"action": "unsuspend"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func (h *Handler) GetJobs(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)

	result, err := h.service.GetJobs(r.Context(), status, page, limit)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RetryJob(w http.ResponseWriter, r *http.Request) {
	adminID := adminIDFromContext(r)
	jobID, err := strconv.ParseInt(chi.URLParam(r, "jobID"), 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid job id", nil)
		return
	}
	ok, err := h.service.RetryJob(r.Context(), adminID, jobID)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	if !ok {
		apierrors.WriteError(w, http.StatusConflict, apierrors.ErrCodeConflict, "job is not in a retryable state", nil)
		return
	}
	h.logAdminAction(r, adminID, "admin_action", "scan_job/"+strconv.FormatInt(jobID, 10), true, map[string]any{"action": "retry"})
	writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
}

func (h *Handler) GetAICosts(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	to := now.AddDate(0, 0, 1)
	from := now.AddDate(0, 0, -30)

	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			from = parsed
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			to = parsed.AddDate(0, 0, 1)
		}
	}

	result, err := h.service.GetAICosts(r.Context(), from, to)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	actorUserID := int64(queryInt(r, "userId", 0))
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 50)

	result, err := h.service.GetAuditLogs(r.Context(), actorUserID, page, limit)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetContentFlags(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", 20)

	result, err := h.service.GetContentFlags(r.Context(), status, page, limit)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type resolveFlagRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func (h *Handler) ResolveContentFlag(w http.ResponseWriter, r *http.Request) {
	adminID := adminIDFromContext(r)
	flagID, err := strconv.ParseInt(chi.URLParam(r, "flagID"), 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid flag id", nil)
		return
	}
	var req resolveFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid request body", nil)
		return
	}
	if req.Action != "approved" && req.Action != "removed" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "action must be 'approved' or 'removed'", nil)
		return
	}
	if err := h.service.ResolveContentFlag(r.Context(), adminID, flagID, req.Action, req.Reason); err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.logAdminAction(r, adminID, "admin_action", "content_flag/"+strconv.FormatInt(flagID, 10), true, map[string]any{"action": req.Action})
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Action})
}

func (h *Handler) logAdminAction(r *http.Request, adminID int64, action, resource string, success bool, details map[string]any) {
	if h.db == nil {
		return
	}
	_ = common.LogAuditEvent(r.Context(), h.db, common.AuditEvent{
		UserID:    strconv.FormatInt(adminID, 10),
		Action:    action,
		Resource:  resource,
		IPAddress: custommw.ClientIP(r),
		UserAgent: r.UserAgent(),
		Success:   success,
		Details:   details,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
