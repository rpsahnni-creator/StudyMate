package notifications

import (
	"strings"
	"testing"
)

func TestRenderQuizReadyTemplate(t *testing.T) {
	tmpl, err := Render(TmplQuizReady, map[string]string{
		"subject": "Physics",
		"count":   "12",
		"quizUrl": "/quiz/99",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(tmpl.PushBody, "Physics") {
		t.Fatalf("expected subject in push body, got %q", tmpl.PushBody)
	}
	if !strings.Contains(tmpl.EmailHTML, "12") {
		t.Fatalf("expected count in email html, got %q", tmpl.EmailHTML)
	}
	if !strings.Contains(tmpl.EmailHTML, "/quiz/99") {
		t.Fatalf("expected quiz url in email html, got %q", tmpl.EmailHTML)
	}
}

func TestRenderPaymentSuccessTemplate(t *testing.T) {
	tmpl, err := Render(TmplPaymentSuccess, map[string]string{
		"plan":    "Pro",
		"expires": "5 Aug 2026",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(tmpl.PushBody, "Pro") || !strings.Contains(tmpl.EmailText, "5 Aug 2026") {
		t.Fatalf("unexpected render: push=%q email=%q", tmpl.PushBody, tmpl.EmailText)
	}
}

func TestRenderPasswordResetTemplate(t *testing.T) {
	tmpl, err := Render(TmplPasswordReset, map[string]string{
		"name":          "Asha",
		"resetUrl":      "https://app.studyapp.in/reset-password?token=abc",
		"expiryMinutes": "30",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(tmpl.EmailHTML, "Asha") || !strings.Contains(tmpl.EmailText, "abc") {
		t.Fatalf("unexpected render: html=%q text=%q", tmpl.EmailHTML, tmpl.EmailText)
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	_, err := Render(TemplateID("missing"), nil)
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}
