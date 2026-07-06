package scan

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"studyapp/backend/internal/billing"
	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
)

// handlerService defines the minimal behavior needed by the HTTP layer.
type handlerService interface {
	CreateJob(ctx context.Context, userID int64, req CreateScanJobRequest) (ScanJobResponse, error)
	GetJob(ctx context.Context, jobID int64) (ScanJob, error)
	GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error)
	SetJobStrategy(ctx context.Context, jobID, userID int64, strategy string) error
	RecordConsent(ctx context.Context, userID int64) error
	UploadPageImage(ctx context.Context, jobID int64, pageNo int, reader io.Reader, size int64) error
}

// cacheInvalidator removes cached quiz entries by content hash.
type cacheInvalidator interface {
	Invalidate(ctx context.Context, contentHash string) error
}

// scanBilling enforces subscription scan limits.
type scanBilling interface {
	CheckScanLimit(ctx context.Context, userID int64) error
	RecordScan(ctx context.Context, userID int64) error
}

// Handler exposes scan-job REST endpoints.
type Handler struct {
	service       handlerService
	chunkUploader *ChunkUploadHandler
	cacheService  cacheInvalidator
	billing       scanBilling
	logger        *slog.Logger
}

func NewHandler(service handlerService) *Handler {
	return &Handler{service: service, logger: slog.Default()}
}

func (h *Handler) WithChunkUpload(uploader *ChunkUploadHandler) *Handler {
	h.chunkUploader = uploader
	return h
}

func (h *Handler) WithCacheService(cacheService cacheInvalidator) *Handler {
	h.cacheService = cacheService
	return h
}

func (h *Handler) WithBilling(billing scanBilling) *Handler {
	h.billing = billing
	return h
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/scan/jobs", h.CreateJob)
	r.Get("/scan/jobs/{jobID}", h.GetJob)
	r.Put("/scan/jobs/{jobID}/strategy", h.SetJobStrategy)
	r.Get("/scan/uploads/{jobID}/status", h.GetUploadStatus)
}

func (h *Handler) RegisterUploadRoutes(r chi.Router) {
	r.Post("/scan/upload", h.Upload)
	r.Post("/scan/jobs/{jobID}/pages/{pageNo}/image", h.UploadJobPage)
	if h.chunkUploader != nil {
		r.Post("/scan/upload/chunk", h.chunkUploader.HandleChunkUpload)
		r.Post("/scan/upload/complete", h.chunkUploader.HandleUploadComplete)
	}
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
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

	var req CreateScanJobRequest
	if err := json.Unmarshal(body, &req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}

	if vErrs := ValidateScanJob(ScanJobRequestFromCreate(req)); len(vErrs) > 0 {
		details := make(map[string]string, len(vErrs))
		for _, ve := range vErrs {
			details[ve.Field] = ve.Message
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "validation failed", details)
		return
	}
	if vErrs := ValidateScanMode(req.Mode); len(vErrs) > 0 {
		details := make(map[string]string, len(vErrs))
		for _, ve := range vErrs {
			details[ve.Field] = ve.Message
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "validation failed", details)
		return
	}

	if h.billing != nil {
		if err := h.billing.CheckScanLimit(r.Context(), userID); err != nil {
			if errors.Is(err, billing.ErrScanLimitExceeded) {
				apierrors.WriteError(w, http.StatusTooManyRequests, apierrors.ErrCodeScanLimitExceeded, "daily scan limit exceeded", nil)
				return
			}
			apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
			return
		}
	}

	resp, err := h.service.CreateJob(r.Context(), userID, req)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	if h.billing != nil {
		_ = h.billing.RecordScan(r.Context(), userID)
	}

	h.jsonResponse(w, http.StatusCreated, resp)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	_, fileReader, fileSize, err := ValidateUpload(r)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	var req CreateScanJobRequest
	req.Mode = r.FormValue("mode")
	if req.Mode == "" {
		req.Mode = "chapter"
	}
	req.Board = r.FormValue("board")
	req.ChapterTitle = r.FormValue("chapter_title")
	if bookID, err := strconv.ParseInt(r.FormValue("book_id"), 10, 64); err == nil {
		req.BookID = &bookID
	}
	if chapterID, err := strconv.ParseInt(r.FormValue("chapter_id"), 10, 64); err == nil {
		req.ChapterID = &chapterID
	}
	if pageNo, err := strconv.Atoi(r.FormValue("page_no")); err == nil {
		req.PageNo = pageNo
	}
	if accepted, err := strconv.ParseBool(r.FormValue("accepted_terms")); err == nil {
		req.AcceptedTerms = accepted
	} else if r.FormValue("accepted_terms") != "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "accepted_terms must be true or false", nil)
		return
	}

	if vErrs := ValidateScanJob(ScanJobRequestFromCreate(req)); len(vErrs) > 0 {
		details := make(map[string]string, len(vErrs))
		for _, ve := range vErrs {
			details[ve.Field] = ve.Message
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "validation failed", details)
		return
	}
	if vErrs := ValidateScanMode(req.Mode); len(vErrs) > 0 {
		details := make(map[string]string, len(vErrs))
		for _, ve := range vErrs {
			details[ve.Field] = ve.Message
		}
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "validation failed", details)
		return
	}

	resp, err := h.service.CreateJob(r.Context(), userID, req)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	pageNo := 1
	if req.PageNo > 0 {
		pageNo = req.PageNo
	}
	if err := h.service.UploadPageImage(r.Context(), resp.Job.ID, pageNo, fileReader, fileSize); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	h.jsonResponse(w, http.StatusCreated, resp)
}

// UploadJobPage attaches an image to an existing scan job page (mobile-friendly multipart).
func (h *Handler) UploadJobPage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	jobID, err := strconv.ParseInt(chi.URLParam(r, "jobID"), 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid job id", nil)
		return
	}
	pageNo, err := strconv.Atoi(chi.URLParam(r, "pageNo"))
	if err != nil || pageNo < 1 {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid page number", nil)
		return
	}

	if _, err := h.service.GetJobForUser(r.Context(), jobID, userID); err != nil {
		apierrors.WriteNotFound(w, "job")
		return
	}

	_, fileReader, fileSize, err := ValidateUpload(r)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	if err := h.service.UploadPageImage(r.Context(), jobID, pageNo, fileReader, fileSize); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, err.Error(), nil)
		return
	}

	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]any{
		"job_id":  jobID,
		"page_no": pageNo,
		"status":  string(job.Status),
	})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	jobIDParam := chi.URLParam(r, "jobID")
	jobID, err := strconv.ParseInt(jobIDParam, 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid job id", nil)
		return
	}

	job, err := h.service.GetJobForUser(r.Context(), jobID, userID)
	if err != nil {
		apierrors.WriteNotFound(w, "job")
		return
	}

	h.jsonResponse(w, http.StatusOK, job)
}

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Delete("/admin/cache/{contentHash}", h.InvalidateCache)
}

func (h *Handler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	if h.cacheService == nil {
		apierrors.WriteError(w, http.StatusServiceUnavailable, apierrors.ErrCodeInternalError, "cache service not configured", nil)
		return
	}
	contentHash := chi.URLParam(r, "contentHash")
	if contentHash == "" {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "content hash is required", nil)
		return
	}
	if err := h.cacheService.Invalidate(r.Context(), contentHash); err != nil {
		apierrors.WriteInternal(w, h.logger, err, custommw.GetTraceID(r.Context()))
		return
	}
	h.jsonResponse(w, http.StatusOK, map[string]string{
		"message":      "cache entry invalidated",
		"content_hash": contentHash,
	})
}

func (h *Handler) GetUploadStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	jobIDParam := chi.URLParam(r, "jobID")
	jobID, err := strconv.ParseInt(jobIDParam, 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid job id", nil)
		return
	}

	job, err := h.service.GetJobForUser(r.Context(), jobID, userID)
	if err != nil {
		apierrors.WriteNotFound(w, "job")
		return
	}

	status := UploadStatusResponse{
		Status:   string(job.Status),
		Progress: job.Progress,
		CanRetry: job.Status == ScanJobFailed,
		Message:  "upload is being processed",
		QuizID:   job.QuizID,
	}
	if job.DetectedPageType != nil {
		status.DetectedPageType = *job.DetectedPageType
	}
	if job.ChapterSummary != nil {
		status.ChapterSummary = *job.ChapterSummary
	}
	if job.Status == ScanJobNeedsStrategy {
		status.NeedsStrategy = true
		status.Message = "mixed page detected — choose extract questions or generate from chapter"
	}
	if job.Status == ScanJobFailed {
		status.Message = "upload failed; retry is available"
	}
	if job.Status == ScanJobQuizReady || job.Status == ScanJobCompleted {
		status.Message = "upload completed successfully"
	}
	if job.Status == ScanJobOCRComplete {
		status.Message = "ocr complete; generating quiz"
	}

	h.jsonResponse(w, http.StatusOK, status)
}

func (h *Handler) SetJobStrategy(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		apierrors.WriteUnauthorized(w)
		return
	}

	jobIDParam := chi.URLParam(r, "jobID")
	jobID, err := strconv.ParseInt(jobIDParam, 10, 64)
	if err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid job id", nil)
		return
	}

	var req SetJobStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "invalid JSON", nil)
		return
	}
	if !ValidGenerationStrategy(req.GenerationStrategy) {
		apierrors.WriteError(w, http.StatusBadRequest, apierrors.ErrCodeValidation, "generation_strategy must be extract_questions or generate_from_chapter", nil)
		return
	}

	if err := h.service.SetJobStrategy(r.Context(), jobID, userID, req.GenerationStrategy); err != nil {
		if errors.Is(err, ErrJobNotAwaitingStrategy) {
			apierrors.WriteError(w, http.StatusConflict, apierrors.ErrCodeValidation, err.Error(), nil)
			return
		}
		apierrors.WriteNotFound(w, "job")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":              "pending",
		"generation_strategy": req.GenerationStrategy,
		"message":             "strategy saved; quiz generation will resume",
	})
}

func ValidGenerationStrategy(s string) bool {
	return s == "extract_questions" || s == "generate_from_chapter"
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
