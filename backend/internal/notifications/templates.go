package notifications

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplateID identifies a notification template.
type TemplateID string

const (
	TmplQuizReady        TemplateID = "quiz_ready"
	TmplPaymentSuccess   TemplateID = "payment_success"
	TmplPaymentFailed    TemplateID = "payment_failed"
	TmplPracticeReminder TemplateID = "practice_reminder"
	TmplWelcome          TemplateID = "welcome"
	TmplPasswordReset    TemplateID = "password_reset"
)

// Template holds rendered push and email content.
type Template struct {
	PushTitle string
	PushBody  string
	EmailSubj string
	EmailHTML string
	EmailText string
}

type templateDef struct {
	PushTitle string
	PushBody  string
	EmailSubj string
	EmailHTML string
	EmailText string
}

var templateDefs = map[TemplateID]templateDef{
	TmplQuizReady: {
		PushTitle: "Your Quiz is Ready! 📚",
		PushBody:  "{{.subject}} quiz from your scan is ready. Tap to start!",
		EmailSubj: "Your StudyApp quiz is ready",
		EmailHTML: "<h2>Quiz Ready!</h2><p>Your {{.subject}} quiz has {{.count}} questions.</p><a href='{{.quizUrl}}'>Start Quiz</a>",
		EmailText: "Your {{.subject}} quiz has {{.count}} questions. Start here: {{.quizUrl}}",
	},
	TmplPaymentSuccess: {
		PushTitle: "Payment Successful ✅",
		PushBody:  "Your {{.plan}} plan is active until {{.expires}}.",
		EmailSubj: "Payment confirmed — StudyApp {{.plan}} plan",
		EmailHTML: "<h2>Payment Successful</h2><p>Your {{.plan}} subscription is active until {{.expires}}.</p>",
		EmailText: "Your {{.plan}} subscription is active until {{.expires}}.",
	},
	TmplPaymentFailed: {
		PushTitle: "Payment Failed",
		PushBody:  "We couldn't process your payment. Please try again.",
		EmailSubj: "Payment failed — please try again",
		EmailHTML: "<h2>Payment Failed</h2><p>We couldn't process your payment. Please open the app and try again.</p>",
		EmailText: "We couldn't process your payment. Please open the app and try again.",
	},
	TmplPracticeReminder: {
		PushTitle: "Time to Practice 📖",
		PushBody:  "Keep your streak going — a quick quiz is waiting for you.",
		EmailSubj: "Practice reminder from StudyApp",
		EmailHTML: "<h2>Practice Reminder</h2><p>Keep your streak going with a quick quiz today.</p>",
		EmailText: "Keep your streak going with a quick quiz today.",
	},
	TmplWelcome: {
		PushTitle: "Welcome to StudyApp 👋",
		PushBody:  "Scan your textbook and start learning with AI-powered quizzes.",
		EmailSubj: "Welcome to StudyApp",
		EmailHTML: "<h2>Welcome!</h2><p>Scan your textbook and start learning with AI-powered quizzes.</p>",
		EmailText: "Welcome to StudyApp! Scan your textbook and start learning with AI-powered quizzes.",
	},
	TmplPasswordReset: {
		EmailSubj: "Reset your StudyApp password",
		EmailHTML: "<h2>Password Reset</h2><p>Hi {{.name}},</p><p>We received a request to reset your password. Click the link below to choose a new password. This link expires in {{.expiryMinutes}} minutes.</p><p><a href='{{.resetUrl}}'>Reset Password</a></p><p>If you did not request this, you can safely ignore this email.</p>",
		EmailText: "Hi {{.name}},\n\nReset your StudyApp password using this link (expires in {{.expiryMinutes}} minutes):\n{{.resetUrl}}\n\nIf you did not request this, ignore this email.",
	},
}

// Render fills a template with the given data map.
func Render(tmpl TemplateID, data map[string]string) (*Template, error) {
	def, ok := templateDefs[tmpl]
	if !ok {
		return nil, fmt.Errorf("unknown template: %s", tmpl)
	}

	renderField := func(raw string) (string, error) {
		if raw == "" {
			return "", nil
		}
		t, err := template.New("field").Option("missingkey=zero").Parse(raw)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	pushTitle, err := renderField(def.PushTitle)
	if err != nil {
		return nil, fmt.Errorf("push title: %w", err)
	}
	pushBody, err := renderField(def.PushBody)
	if err != nil {
		return nil, fmt.Errorf("push body: %w", err)
	}
	emailSubj, err := renderField(def.EmailSubj)
	if err != nil {
		return nil, fmt.Errorf("email subject: %w", err)
	}
	emailHTML, err := renderField(def.EmailHTML)
	if err != nil {
		return nil, fmt.Errorf("email html: %w", err)
	}
	emailText, err := renderField(def.EmailText)
	if err != nil {
		return nil, fmt.Errorf("email text: %w", err)
	}

	return &Template{
		PushTitle: pushTitle,
		PushBody:  pushBody,
		EmailSubj: emailSubj,
		EmailHTML: emailHTML,
		EmailText: emailText,
	}, nil
}
