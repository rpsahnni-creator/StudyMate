package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInitAndMetricsHandler(t *testing.T) {
	reg := Init()
	RecordScanJob("completed")
	handler := Handler(reg)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "studyapp_scan_jobs_total") {
		t.Fatal("expected prometheus metric in output")
	}
}

func TestHTTPMiddlewareRecordsRequest(t *testing.T) {
	Init()
	handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	exporter := httptest.NewRecorder()
	Handler(nil).ServeHTTP(exporter, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(exporter.Body.String(), `status_code="201"`) {
		t.Fatal("expected recorded status code in metrics")
	}
}
