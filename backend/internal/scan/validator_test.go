package scan

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateScanJob_BoardRequired(t *testing.T) {
	errs := ValidateScanJob(ScanJobRequest{PageCount: 1})
	if len(errs) != 0 {
		t.Fatalf("empty board should default to ncert, got %+v", errs)
	}
}

func TestValidateScanJob_InvalidBoard(t *testing.T) {
	errs := ValidateScanJob(ScanJobRequest{Board: "invalid", PageCount: 1})
	if len(errs) == 0 {
		t.Fatal("expected invalid board error")
	}
}

func TestNormalizeBoard_AcceptsCommonInput(t *testing.T) {
	cases := map[string]string{
		"":                "ncert",
		"NCERT":           "ncert",
		"cbse":            "cbse",
		"ICSE":            "icse",
		"Jharkhand Board": "jharkhand_board",
		"Bihar Board":     "bihar_board",
		"state board":     "state_board",
	}
	for in, want := range cases {
		if got := NormalizeBoard(in); got != want {
			t.Fatalf("NormalizeBoard(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateScanJob_JharkhandBoard(t *testing.T) {
	errs := ValidateScanJob(ScanJobRequest{Board: "jharkhand_board", PageCount: 1})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

func TestValidateScanJob_StateBoardLabel(t *testing.T) {
	errs := ValidateScanJob(ScanJobRequest{Board: "state board", PageCount: 1})
	if len(errs) != 0 {
		t.Fatalf("expected no errors for state board label, got %+v", errs)
	}
}

func TestValidateUpload_RejectsNonImage(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("not an image file content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/scan/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, _, _, err = ValidateUpload(req)
	if err == nil {
		t.Fatal("expected non-image rejection")
	}
}

func TestValidateUpload_AcceptsJPEG(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'})
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/scan/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, _, _, err = ValidateUpload(req)
	if err != nil {
		t.Fatalf("expected jpeg acceptance, got %v", err)
	}
}

func TestValidateUpload_RejectsFakeImageContentType(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "fake.jpg")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("<?php echo 'bad'; ?>"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/scan/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, _, _, err = ValidateUpload(req)
	if err == nil {
		t.Fatal("expected magic-byte rejection")
	}
}
