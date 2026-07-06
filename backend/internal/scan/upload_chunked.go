package scan

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	apierrors "studyapp/backend/internal/common/errors"
	"studyapp/backend/internal/scan/storage"
)

const (
	chunkSizeBytes = 512 * 1024
	chunkTimeout   = 60 * time.Second
)

var maxChunksPerPage = storage.MaxPageUploadSize() / chunkSizeBytes

// ChunkUploadHandler serves resumable multipart page uploads.
type ChunkUploadHandler struct {
	repo    Repository
	storage storage.Client
}

func NewChunkUploadHandler(repo Repository, store storage.Client) *ChunkUploadHandler {
	return &ChunkUploadHandler{repo: repo, storage: store}
}

func (h *ChunkUploadHandler) HandleChunkUpload(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), chunkTimeout)
	defer cancel()

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		writeUploadError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	jobID, pageNo, chunkIndex, totalChunks, checksum, err := parseChunkHeaders(r)
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, err.Error())
		return
	}

	if totalChunks <= 0 || totalChunks > maxChunksPerPage {
		writeUploadError(w, http.StatusBadRequest, fmt.Sprintf("total chunks must be between 1 and %d", maxChunksPerPage))
		return
	}
	if chunkIndex < 0 || chunkIndex >= totalChunks {
		writeUploadError(w, http.StatusBadRequest, "invalid chunk index")
		return
	}

	if _, err := h.repo.GetJobForUser(ctx, jobID, userID); err != nil {
		writeUploadError(w, http.StatusNotFound, "job not found")
		return
	}

	page, err := h.repo.EnsurePage(ctx, jobID, pageNo)
	if err != nil {
		writeUploadError(w, http.StatusInternalServerError, "failed to resolve scan page")
		return
	}
	if page.UploadStatus == "uploaded" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "already_uploaded"})
		return
	}

	meta := decodeChunkMetadata(page.ChunkMetadata)
	if meta.TotalChunks == 0 {
		meta.TotalChunks = totalChunks
	}
	if meta.TotalChunks != totalChunks {
		writeUploadError(w, http.StatusBadRequest, "total chunk count mismatch")
		return
	}
	if chunkAlreadyReceived(meta, chunkIndex) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "already_received"})
		return
	}

	if err := r.ParseMultipartForm(chunkSizeBytes + (32 << 10)); err != nil {
		writeUploadError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	file, _, err := r.FormFile("chunk")
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, "chunk file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, chunkSizeBytes+1))
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, "failed to read chunk")
		return
	}
	if len(data) > chunkSizeBytes {
		writeUploadError(w, http.StatusBadRequest, "chunk exceeds 512KB limit")
		return
	}
	if chunkIndex == 0 && int64(totalChunks)*chunkSizeBytes > int64(storage.MaxPageUploadSize()) {
		writeUploadError(w, http.StatusBadRequest, "upload exceeds 10MB page limit")
		return
	}

	if got := md5Hex(data); !strings.EqualFold(got, checksum) {
		writeUploadError(w, http.StatusBadRequest, "chunk checksum mismatch")
		return
	}

	chunkKey := chunkObjectKey(jobID, pageNo, chunkIndex)
	if err := h.storage.PutObject(ctx, chunkKey, data); err != nil {
		writeUploadError(w, http.StatusInternalServerError, "failed to store chunk")
		return
	}

	meta = recordChunkReceipt(meta, chunkIndex, checksum)
	if err := h.repo.UpdatePageChunkMetadata(ctx, page.ID, meta); err != nil {
		writeUploadError(w, http.StatusInternalServerError, "failed to record chunk metadata")
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":       "received",
		"chunk_index":  chunkIndex,
		"received":     len(meta.Received),
		"total_chunks": meta.TotalChunks,
	})
}

func (h *ChunkUploadHandler) HandleUploadComplete(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), chunkTimeout)
	defer cancel()

	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		writeUploadError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req UploadCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeUploadError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.JobID <= 0 || req.PageNumber <= 0 || req.TotalChunks <= 0 {
		writeUploadError(w, http.StatusBadRequest, "jobId, pageNumber, and totalChunks are required")
		return
	}

	job, err := h.repo.GetJobForUser(ctx, req.JobID, userID)
	if err != nil {
		writeUploadError(w, http.StatusNotFound, "job not found")
		return
	}

	page, err := h.repo.GetPageByJobAndPageNo(ctx, req.JobID, req.PageNumber)
	if err != nil {
		writeUploadError(w, http.StatusNotFound, "scan page not found")
		return
	}

	// Idempotent: page already composed and uploaded.
	if page.UploadStatus == "uploaded" && page.TempStorageKey != nil && *page.TempStorageKey != "" {
		jobStatus := string(job.Status)
		if ready, err := h.repo.AllPagesUploaded(ctx, req.JobID); err == nil && ready && job.Status == ScanJobUploading {
			if err := h.repo.UpdateJobStatus(ctx, req.JobID, ScanJobPending, 0, nil); err == nil {
				jobStatus = string(ScanJobPending)
			}
		}
		writeJSON(w, http.StatusOK, UploadCompleteResponse{
			JobID:  req.JobID,
			Status: jobStatus,
		})
		return
	}

	meta := decodeChunkMetadata(page.ChunkMetadata)
	if meta.TotalChunks == 0 {
		meta.TotalChunks = req.TotalChunks
	}
	if meta.TotalChunks != req.TotalChunks {
		writeUploadError(w, http.StatusBadRequest, "total chunk count mismatch")
		return
	}
	if len(meta.Received) < meta.TotalChunks {
		writeUploadError(w, http.StatusBadRequest, "missing chunks")
		return
	}

	sort.Ints(meta.Received)
	for i := 0; i < meta.TotalChunks; i++ {
		if meta.Received[i] != i {
			writeUploadError(w, http.StatusBadRequest, fmt.Sprintf("missing chunk %d", i))
			return
		}
	}

	var composed []byte
	for i := 0; i < meta.TotalChunks; i++ {
		part, err := h.storage.GetObject(ctx, chunkObjectKey(req.JobID, req.PageNumber, i))
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, fmt.Sprintf("missing stored chunk %d", i))
			return
		}
		composed = append(composed, part...)
	}
	if len(composed) > storage.MaxPageUploadSize() {
		writeUploadError(w, http.StatusBadRequest, "upload exceeds 10MB page limit")
		return
	}

	contentType, ext, err := detectImageContentType(composed)
	if err != nil {
		writeUploadError(w, http.StatusBadRequest, err.Error())
		return
	}

	finalKey := fmt.Sprintf("temp/%d/%d/%s%s", req.JobID, req.PageNumber, uuid.New().String(), ext)
	if err := h.storage.PutObjectStream(ctx, finalKey, contentType, bytes.NewReader(composed), int64(len(composed))); err != nil {
		writeUploadError(w, http.StatusInternalServerError, "failed to store composed upload")
		return
	}
	composed = nil

	chunkPrefix := fmt.Sprintf("temp/chunks/%d/%d/", req.JobID, req.PageNumber)
	_ = h.storage.DeletePrefix(ctx, chunkPrefix)

	if err := h.repo.MarkPageUploaded(ctx, page.ID, finalKey); err != nil {
		writeUploadError(w, http.StatusInternalServerError, "failed to update scan page")
		return
	}

	jobStatus := string(job.Status)
	if ready, err := h.repo.AllPagesUploaded(ctx, req.JobID); err == nil && ready {
		if err := h.repo.UpdateJobStatus(ctx, req.JobID, ScanJobPending, 0, nil); err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to update job status")
			return
		}
		jobStatus = string(ScanJobPending)
	}

	writeJSON(w, http.StatusOK, UploadCompleteResponse{
		JobID:  req.JobID,
		Status: jobStatus,
	})
}

func parseChunkHeaders(r *http.Request) (jobID int64, pageNo, chunkIndex, totalChunks int, checksum string, err error) {
	jobID, err = strconv.ParseInt(strings.TrimSpace(r.Header.Get("X-Job-ID")), 10, 64)
	if err != nil || jobID <= 0 {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid X-Job-ID header")
	}
	pageNo, err = strconv.Atoi(strings.TrimSpace(r.Header.Get("X-Page-Number")))
	if err != nil || pageNo <= 0 {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid X-Page-Number header")
	}
	chunkIndex, err = strconv.Atoi(strings.TrimSpace(r.Header.Get("X-Chunk-Index")))
	if err != nil {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid X-Chunk-Index header")
	}
	totalChunks, err = strconv.Atoi(strings.TrimSpace(r.Header.Get("X-Total-Chunks")))
	if err != nil {
		return 0, 0, 0, 0, "", fmt.Errorf("invalid X-Total-Chunks header")
	}
	checksum = strings.TrimSpace(r.Header.Get("X-Chunk-Checksum"))
	if checksum == "" {
		return 0, 0, 0, 0, "", fmt.Errorf("missing X-Chunk-Checksum header")
	}
	return jobID, pageNo, chunkIndex, totalChunks, checksum, nil
}

func chunkObjectKey(jobID int64, pageNo, chunkIndex int) string {
	return fmt.Sprintf("temp/chunks/%d/%d/%d.part", jobID, pageNo, chunkIndex)
}

func decodeChunkMetadata(raw json.RawMessage) PageChunkMetadata {
	if len(raw) == 0 {
		return PageChunkMetadata{Checksums: map[string]string{}}
	}
	var meta PageChunkMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return PageChunkMetadata{Checksums: map[string]string{}}
	}
	if meta.Checksums == nil {
		meta.Checksums = map[string]string{}
	}
	return meta
}

func chunkAlreadyReceived(meta PageChunkMetadata, chunkIndex int) bool {
	for _, received := range meta.Received {
		if received == chunkIndex {
			return true
		}
	}
	return false
}

func recordChunkReceipt(meta PageChunkMetadata, chunkIndex int, checksum string) PageChunkMetadata {
	if meta.Checksums == nil {
		meta.Checksums = map[string]string{}
	}
	if chunkAlreadyReceived(meta, chunkIndex) {
		return meta
	}
	meta.Received = append(meta.Received, chunkIndex)
	meta.Checksums[strconv.Itoa(chunkIndex)] = checksum
	return meta
}

func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func detectImageContentType(data []byte) (contentType, ext string, err error) {
	if len(data) < 12 {
		return "", "", fmt.Errorf("file too small to be a valid image")
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg", ".jpg", nil
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "image/png", ".png", nil
	case string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp", ".webp", nil
	default:
		return "", "", fmt.Errorf("unsupported content type: only image/jpeg, image/png, image/webp allowed")
	}
}

func writeUploadError(w http.ResponseWriter, status int, message string) {
	switch status {
	case http.StatusUnauthorized:
		apierrors.WriteUnauthorized(w)
	case http.StatusNotFound:
		apierrors.WriteError(w, status, apierrors.ErrCodeNotFound, message, nil)
	case http.StatusInternalServerError:
		apierrors.WriteError(w, status, apierrors.ErrCodeInternalError, "an internal error occurred", nil)
	default:
		apierrors.WriteError(w, status, apierrors.ErrCodeValidation, message, nil)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
