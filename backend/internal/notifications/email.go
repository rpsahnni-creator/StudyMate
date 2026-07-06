package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"
)

// EmailRequest is a transactional email send request.
type EmailRequest struct {
	To      string
	Subject string
	HTML    string
	Text    string
	Tags    []string
}

// EmailConfig configures the outbound email provider.
type EmailConfig struct {
	Provider     string
	ResendAPIKey string
	From         string
	AWSRegion    string
}

// EmailClient sends transactional email.
type EmailClient interface {
	SendTransactional(ctx context.Context, req EmailRequest) error
}

// StubClient logs emails without sending.
type StubClient struct {
	logger *slog.Logger
}

func (c *StubClient) SendTransactional(ctx context.Context, req EmailRequest) error {
	if c.logger != nil {
		c.logger.Info("email stub send",
			"to", req.To,
			"subject", req.Subject,
			"text", req.Text,
			"tags", req.Tags,
		)
		for _, tag := range req.Tags {
			if tag == "registration_otp" {
				c.logger.Warn("REGISTRATION OTP (stub — not sent to inbox)", "to", req.To, "text", req.Text)
				break
			}
		}
	}
	return nil
}

// ResendClient sends email via the Resend HTTP API.
type ResendClient struct {
	apiKey     string
	from       string
	httpClient *http.Client
}

func NewResendClient(apiKey, from string) *ResendClient {
	return &ResendClient{
		apiKey:     apiKey,
		from:       formatFrom(from),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *ResendClient) SendTransactional(ctx context.Context, req EmailRequest) error {
	payload := map[string]any{
		"from":    c.from,
		"to":      []string{req.To},
		"subject": req.Subject,
		"html":    req.HTML,
		"text":    req.Text,
	}
	if len(req.Tags) > 0 {
		tags := make([]map[string]string, 0, len(req.Tags))
		for _, tag := range req.Tags {
			tags = append(tags, map[string]string{"name": tag, "value": "true"})
		}
		payload["tags"] = tags
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("resend error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SESClient sends email via AWS SES v2.
type SESClient struct {
	client *sesv2.Client
	from   string
}

func NewSESClient(ctx context.Context, region, from string) (*SESClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &SESClient{
		client: sesv2.NewFromConfig(cfg),
		from:   from,
	}, nil
}

func (c *SESClient) SendTransactional(ctx context.Context, req EmailRequest) error {
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(formatFrom(c.from)),
		Destination: &types.Destination{
			ToAddresses: []string{req.To},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(req.Subject)},
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(req.HTML)},
					Text: &types.Content{Data: aws.String(req.Text)},
				},
			},
		},
	}
	_, err := c.client.SendEmail(ctx, input)
	return err
}

// NewEmailClient selects the provider from config.
func NewEmailClient(ctx context.Context, cfg EmailConfig, logger *slog.Logger) (EmailClient, error) {
	from := cfg.From
	if from == "" {
		from = "noreply@studyapp.in"
	}

	switch strings.ToLower(cfg.Provider) {
	case "resend":
		if cfg.ResendAPIKey == "" {
			return nil, fmt.Errorf("RESEND_API_KEY is required when EMAIL_PROVIDER=resend")
		}
		return NewResendClient(cfg.ResendAPIKey, from), nil
	case "ses":
		region := cfg.AWSRegion
		if region == "" {
			region = "ap-south-1"
		}
		return NewSESClient(ctx, region, from)
	default:
		return &StubClient{logger: logger}, nil
	}
}

func formatFrom(addr string) string {
	if strings.Contains(addr, "<") {
		return addr
	}
	return fmt.Sprintf("StudyApp <%s>", addr)
}

// EmailProvider defines the legacy outbound email provider interface.
type EmailProvider interface {
	Send(ctx context.Context, to, subject, body string) error
}

// NoOpEmailProvider is the default provider used in local/dev mode.
type NoOpEmailProvider struct{}

func (p NoOpEmailProvider) Send(ctx context.Context, to, subject, body string) error {
	return nil
}

type emailProviderAdapter struct {
	client EmailClient
}

func (a emailProviderAdapter) Send(ctx context.Context, to, subject, body string) error {
	return a.client.SendTransactional(ctx, EmailRequest{
		To:      to,
		Subject: subject,
		HTML:    body,
		Text:    body,
	})
}

// EmailService handles email sending operations.
type EmailService struct {
	logger   Logger
	repo     *Repository
	provider EmailProvider
	client   EmailClient
}

// NewEmailService creates a new email service.
func NewEmailService(logger Logger, repo *Repository, client EmailClient) *EmailService {
	svc := &EmailService{
		logger: logger,
		repo:   repo,
		client: client,
	}
	if client != nil {
		svc.provider = emailProviderAdapter{client: client}
	} else {
		svc.provider = NoOpEmailProvider{}
	}
	return svc
}

// SetProvider overrides the default provider for real integrations like Resend or SES.
func (s *EmailService) SetProvider(provider EmailProvider) {
	if provider != nil {
		s.provider = provider
	}
}

// SendEmail sends an email notification.
func (s *EmailService) SendEmail(ctx context.Context, jobID uuid.UUID, to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("email address required")
	}

	s.logger.Info("sending email",
		"job_id", jobID,
		"to", to,
		"subject", subject,
	)

	if strings.Contains(strings.ToLower(body), "{{") {
		return fmt.Errorf("template placeholders were not fully resolved")
	}

	if err := s.provider.Send(ctx, to, subject, body); err != nil {
		return err
	}

	if s.repo != nil {
		_ = s.repo.LogEmailEvent(ctx, &EmailEvent{
			ID:              uuid.New(),
			EmailAddress:    to,
			EventType:       "delivery",
			ProviderEventID: jobID.String(),
			Metadata:        JSONB{"job_id": jobID.String()},
			CreatedAt:       time.Now(),
		})
	}

	return nil
}

// SendTransactional sends a fully rendered transactional email.
func (s *EmailService) SendTransactional(ctx context.Context, req EmailRequest) error {
	if s.client == nil {
		return s.provider.Send(ctx, req.To, req.Subject, req.HTML)
	}
	return s.client.SendTransactional(ctx, req)
}

// HandleEmailBounce handles bounce events.
func (s *EmailService) HandleEmailBounce(ctx context.Context, email string, reason string) error {
	s.logger.Info("email bounced", "email", email, "reason", reason)
	if s.repo != nil {
		_ = s.repo.LogEmailEvent(ctx, &EmailEvent{
			ID:           uuid.New(),
			EmailAddress: email,
			EventType:    "bounce",
			Metadata:     JSONB{"reason": reason},
			CreatedAt:    time.Now(),
		})
	}
	return nil
}

// HandleEmailComplaint handles complaint events.
func (s *EmailService) HandleEmailComplaint(ctx context.Context, email string) error {
	s.logger.Info("email complained", "email", email)
	if s.repo != nil {
		_ = s.repo.LogEmailEvent(ctx, &EmailEvent{
			ID:           uuid.New(),
			EmailAddress: email,
			EventType:    "complaint",
			CreatedAt:    time.Now(),
		})
	}
	return nil
}

// HandleEmailDelivery marks email as delivered.
func (s *EmailService) HandleEmailDelivery(ctx context.Context, jobID uuid.UUID) error {
	s.logger.Info("email delivered", "job_id", jobID)
	if s.repo != nil {
		_ = s.repo.LogEmailEvent(ctx, &EmailEvent{
			ID:              uuid.New(),
			EmailAddress:    "",
			EventType:       "delivery",
			ProviderEventID: jobID.String(),
			Metadata:        JSONB{"job_id": jobID.String()},
			CreatedAt:       time.Now(),
		})
	}
	return nil
}

// ResendWebhookHeaders holds Svix verification headers from Resend webhooks.
type ResendWebhookHeaders struct {
	ID        string
	Timestamp string
	Signature string
}

const resendWebhookTolerance = 5 * time.Minute

// VerifyResendSignature verifies Resend (Svix) webhook signatures using HMAC-SHA256.
func VerifyResendSignature(payload []byte, headers ResendWebhookHeaders, secret string) bool {
	if secret == "" || headers.ID == "" || headers.Timestamp == "" || headers.Signature == "" {
		return false
	}

	ts, err := strconv.ParseInt(headers.Timestamp, 10, 64)
	if err != nil {
		return false
	}
	eventTime := time.Unix(ts, 0)
	now := time.Now()
	if eventTime.Before(now.Add(-resendWebhookTolerance)) || eventTime.After(now.Add(resendWebhookTolerance)) {
		return false
	}

	key, err := decodeWebhookSecret(secret)
	if err != nil {
		return false
	}

	signedContent := headers.ID + "." + headers.Timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signedContent))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, part := range strings.Split(headers.Signature, " ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		sig := part
		if idx := strings.Index(part, ","); idx >= 0 {
			sig = part[idx+1:]
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) == 1 {
			return true
		}
	}
	return false
}

func decodeWebhookSecret(secret string) ([]byte, error) {
	encoded := strings.TrimPrefix(secret, "whsec_")
	return base64.StdEncoding.DecodeString(encoded)
}
