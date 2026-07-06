package scan

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"studyapp/backend/internal/scan/storage"
)

type uploadFakeRepo struct {
	fakeRepository
	job         ScanJob
	page        ScanPage
	metaUpdates []PageChunkMetadata
	uploaded    bool
	uploadedKey string
	jobPending  bool
}

func (r *uploadFakeRepo) GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error) {
	return r.job, nil
}

func (r *uploadFakeRepo) EnsurePage(ctx context.Context, jobID int64, pageNo int) (ScanPage, error) {
	return r.page, nil
}

func (r *uploadFakeRepo) GetPageByJobAndPageNo(ctx context.Context, jobID int64, pageNo int) (ScanPage, error) {
	return r.page, nil
}

func (r *uploadFakeRepo) UpdatePageChunkMetadata(ctx context.Context, pageID int64, metadata PageChunkMetadata) error {
	r.metaUpdates = append(r.metaUpdates, metadata)
	raw, _ := json.Marshal(metadata)
	r.page.ChunkMetadata = raw
	return nil
}

func (r *uploadFakeRepo) MarkPageUploaded(ctx context.Context, pageID int64, storageKey string) error {
	r.uploaded = true
	r.uploadedKey = storageKey
	r.page.UploadStatus = "uploaded"
	return nil
}

func (r *uploadFakeRepo) AllPagesUploaded(ctx context.Context, jobID int64) (bool, error) {
	return true, nil
}

func (r *uploadFakeRepo) UpdateJobStatus(ctx context.Context, jobID int64, status ScanJobStatus, progress int, errMsg *string) error {
	if status == ScanJobPending {
		r.jobPending = true
	}
	return nil
}

func TestChunkUploadDuplicateReturns409(t *testing.T) {
	repo := &uploadFakeRepo{
		job:  ScanJob{ID: 9, UserID: 42, Status: ScanJobUploading},
		page: ScanPage{ID: 3, ScanJobID: 9, PageNo: 1, UploadStatus: "uploading"},
	}
	store := storage.NewLocalClient()
	handler := NewChunkUploadHandler(repo, store)

	body, headers := chunkMultipart(t, []byte("hello chunk"))
	req := httptest.NewRequest(http.MethodPost, "/scan/upload/chunk", body)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(42)))

	rec := httptest.NewRecorder()
	handler.HandleChunkUpload(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first chunk 200, got %d", rec.Code)
	}

	body2, headers2 := chunkMultipart(t, []byte("hello chunk"))
	req2 := httptest.NewRequest(http.MethodPost, "/scan/upload/chunk", body2)
	for key, value := range headers2 {
		req2.Header.Set(key, value)
	}
	req2 = req2.WithContext(context.WithValue(req2.Context(), "user_id", int64(42)))

	rec2 := httptest.NewRecorder()
	handler.HandleChunkUpload(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected duplicate chunk 409, got %d", rec2.Code)
	}
}

func TestUploadCompleteAssemblesChunks(t *testing.T) {
	repo := &uploadFakeRepo{
		job:  ScanJob{ID: 11, UserID: 42, Status: ScanJobUploading},
		page: ScanPage{ID: 5, ScanJobID: 11, PageNo: 1, UploadStatus: "uploading"},
	}
	store := storage.NewLocalClient()
	handler := NewChunkUploadHandler(repo, store)

	// JPEG magic bytes required for content-type validation.
	partA := []byte{0xFF, 0xD8, 0xFF, 0xE0, 'p', 'a', 'r', 't', '-', 'a', '-'}
	partB := []byte{'p', 'a', 'r', 't', '-', 'b'}
	_ = store.PutObject(context.Background(), chunkObjectKey(11, 1, 0), partA)
	_ = store.PutObject(context.Background(), chunkObjectKey(11, 1, 1), partB)

	meta, _ := json.Marshal(PageChunkMetadata{
		TotalChunks: 2,
		Received:    []int{0, 1},
		Checksums:   map[string]string{"0": md5Hex(partA), "1": md5Hex(partB)},
	})
	repo.page.ChunkMetadata = meta

	reqBody := `{"jobId":11,"pageNumber":1,"totalChunks":2}`
	req := httptest.NewRequest(http.MethodPost, "/scan/upload/complete", io.NopCloser(bytes.NewBufferString(reqBody)))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(42)))

	rec := httptest.NewRecorder()
	handler.HandleUploadComplete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected complete 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !repo.uploaded {
		t.Fatal("expected page marked uploaded")
	}
	if !strings.HasPrefix(repo.uploadedKey, "temp/11/1/") {
		t.Fatalf("expected temp key prefix, got %q", repo.uploadedKey)
	}
	if !repo.jobPending {
		t.Fatal("expected job moved to pending")
	}

	// Composed object stored under temp/{job}/{page}/{uuid}.ext
	// Verify via repo that a storage key was set (fake repo doesn't expose it,
	// but MarkPageUploaded was called which is sufficient).
}

func chunkMultipart(t *testing.T, data []byte) (*bytes.Buffer, map[string]string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("chunk", "chunk.part")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	return &body, map[string]string{
		"Content-Type":       writer.FormDataContentType(),
		"X-Job-ID":           "9",
		"X-Page-Number":      "1",
		"X-Chunk-Index":      "0",
		"X-Total-Chunks":     "1",
		"X-Chunk-Checksum":   hex.EncodeToString(sum[:]),
	}
}
