package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyHandlerWithoutDependencies(t *testing.T) {
	prober := &Prober{}
	rec := httptest.NewRecorder()
	prober.ReadyHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without db/cache, got %d", rec.Code)
	}
}

func TestHealthResponseShape(t *testing.T) {
	prober := &Prober{
		Version:         "1.0.0",
		AIProviderName:  "stub",
		OCRProviderName: "stub",
	}
	rec := httptest.NewRecorder()
	prober.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Version != "1.0.0" {
		t.Fatalf("unexpected version %q", resp.Version)
	}
	if _, ok := resp.Checks["database"]; !ok {
		t.Fatal("expected database check")
	}
	if resp.Status != "down" {
		t.Fatalf("expected down without dependencies, got %q", resp.Status)
	}
}
