package scan

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type stubScanService struct {
	created ScanJobResponse
	job     ScanJob
	err     error
}

func (s *stubScanService) CreateJob(ctx context.Context, userID int64, req CreateScanJobRequest) (ScanJobResponse, error) {
	return s.created, s.err
}

func (s *stubScanService) GetJob(ctx context.Context, jobID int64) (ScanJob, error) {
	return s.job, s.err
}

func (s *stubScanService) GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error) {
	return s.job, s.err
}

func (s *stubScanService) SetJobStrategy(ctx context.Context, jobID, userID int64, strategy string) error {
	return s.err
}

func (s *stubScanService) RecordConsent(ctx context.Context, userID int64) error {
	return nil
}

func (s *stubScanService) UploadPageImage(ctx context.Context, jobID int64, pageNo int, reader io.Reader, size int64) error {
	return s.err
}

func TestCreateJobHandlerReturnsCreated(t *testing.T) {
	service := &stubScanService{created: ScanJobResponse{Job: ScanJob{ID: 7, Mode: "chapter"}}}
	h := NewHandler(service)

	body := `{"mode":"chapter","board":"ncert","book_id":1,"chapter_id":2,"accepted_terms":true}`
	req := httptest.NewRequest(http.MethodPost, "/scan/jobs", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(42)))
	rec := httptest.NewRecorder()

	h.CreateJob(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if payload["job"] == nil {
		t.Fatalf("expected job payload in response")
	}
}

func TestUploadJobPageHandlerReturnsOK(t *testing.T) {
	service := &stubScanService{
		job: ScanJob{ID: 7, Status: ScanJobPending},
	}
	h := NewHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "page.jpg")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'})
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/scan/jobs/7/pages/1/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(42)))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "7")
	rctx.URLParams.Add("pageNo", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	h.UploadJobPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if payload["job_id"] != float64(7) {
		t.Fatalf("expected job_id 7, got %v", payload["job_id"])
	}
}
