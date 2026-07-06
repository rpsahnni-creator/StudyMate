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
	if len(errs) == 0 {
		t.Fatal("expected board validation error")
	}
}

func TestValidateScanJob_InvalidBoard(t *testing.T) {
	errs := ValidateScanJob(ScanJobRequest{Board: "invalid", PageCount: 1})
	if len(errs) == 0 {
		t.Fatal("expected invalid board error")
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
