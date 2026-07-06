package notifications

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// FCMMessage is the payload sent to a single device.
type FCMMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// BatchResult summarizes a multi-token send attempt.
type BatchResult struct {
	SuccessCount int
	FailureCount int
	FailedTokens []string
}

// FCMClient sends push notifications via the FCM HTTP v1 API.
type FCMClient struct {
	projectID   string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

// NewFCMClient loads Firebase credentials from FIREBASE_CREDENTIALS_PATH or
// base64-encoded FIREBASE_CREDENTIALS_JSON.
func NewFCMClient(credentialsPath string) (*FCMClient, error) {
	jsonBytes, err := loadFirebaseCredentials(credentialsPath)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, jsonBytes, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, fmt.Errorf("firebase credentials: %w", err)
	}

	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		var meta struct {
			ProjectID string `json:"project_id"`
		}
		if err := json.Unmarshal(jsonBytes, &meta); err == nil {
			projectID = meta.ProjectID
		}
	}
	if projectID == "" {
		return nil, fmt.Errorf("FIREBASE_PROJECT_ID is required")
	}

	return &FCMClient{
		projectID:   projectID,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		tokenSource: creds.TokenSource,
	}, nil
}

func loadFirebaseCredentials(pathOverride string) ([]byte, error) {
	if b64 := os.Getenv("FIREBASE_CREDENTIALS_JSON"); b64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decode FIREBASE_CREDENTIALS_JSON: %w", err)
		}
		return decoded, nil
	}

	path := pathOverride
	if path == "" {
		path = os.Getenv("FIREBASE_CREDENTIALS_PATH")
	}
	if path == "" {
		return nil, fmt.Errorf("FIREBASE_CREDENTIALS_PATH or FIREBASE_CREDENTIALS_JSON is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read firebase credentials: %w", err)
	}
	return data, nil
}

// SendToToken delivers a push notification to a single FCM registration token.
func (f *FCMClient) SendToToken(ctx context.Context, token string, msg FCMMessage) error {
	if f == nil {
		return fmt.Errorf("fcm client not configured")
	}
	if token == "" {
		return fmt.Errorf("empty device token")
	}

	oauthTok, err := f.tokenSource.Token()
	if err != nil {
		return &RetryableError{Err: fmt.Errorf("fcm auth: %w", err)}
	}

	data := msg.Data
	if data == nil {
		data = map[string]string{}
	}

	body := map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]string{
				"title": msg.Title,
				"body":  msg.Body,
			},
			"data":    data,
			"android": map[string]string{"priority": "high"},
			"apns":    map[string]any{"headers": map[string]string{"apns-priority": "10"}},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+oauthTok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return &RetryableError{Err: err}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("token invalid: %s", string(respBody))
	case http.StatusTooManyRequests:
		return &RetryableError{Err: fmt.Errorf("fcm rate limit: %s", string(respBody))}
	default:
		if resp.StatusCode >= 500 {
			return &RetryableError{Err: fmt.Errorf("fcm server error %d: %s", resp.StatusCode, string(respBody))}
		}
		return fmt.Errorf("fcm error %d: %s", resp.StatusCode, string(respBody))
	}
}

// SendToMultiple sends to each token individually (FCM v1 has no multicast).
func (f *FCMClient) SendToMultiple(ctx context.Context, tokens []string, msg FCMMessage) (BatchResult, error) {
	result := BatchResult{}
	for _, token := range tokens {
		err := f.SendToToken(ctx, token, msg)
		if err == nil {
			result.SuccessCount++
			continue
		}
		result.FailureCount++
		if isPermanentFCMError(err) {
			result.FailedTokens = append(result.FailedTokens, token)
		}
	}
	return result, nil
}

func isPermanentFCMError(err error) bool {
	if err == nil || IsRetryable(err) {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "token invalid") ||
		strings.Contains(errStr, "registrationtokennotregistered") ||
		strings.Contains(errStr, "not found")
}

// FCMService handles Firebase Cloud Messaging operations.
type FCMService struct {
	logger         Logger
	repo           *Repository
	client         *FCMClient
	rateLimiter    *TokenBucket
	circuitBreaker *CircuitBreaker
	mutex          sync.Mutex
}

// NewFCMService creates a new FCM service. client may be nil for log-only mode.
func NewFCMService(logger Logger, repo *Repository, client *FCMClient) *FCMService {
	return &FCMService{
		logger:         logger,
		repo:           repo,
		client:         client,
		rateLimiter:    NewTokenBucket(100),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
	}
}

// SendMulticast sends a message to multiple devices.
func (s *FCMService) SendMulticast(ctx context.Context, userID string, tokens []string, data map[string]string) (*MulticastResult, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no device tokens provided")
	}

	if s.circuitBreaker.IsOpen() {
		s.logger.Warn("FCM circuit breaker OPEN")
		return nil, fmt.Errorf("FCM circuit breaker open")
	}

	if !s.rateLimiter.Allow() {
		s.logger.Warn("FCM rate limit exceeded")
		return nil, &RetryableError{Err: fmt.Errorf("rate limited")}
	}

	title := data["title"]
	body := data["body"]
	if title == "" {
		title = "StudyApp"
	}
	if body == "" {
		body = data["category"]
	}

	msg := FCMMessage{Title: title, Body: body, Data: data}
	result := &MulticastResult{
		FailedTokens:    []string{},
		TransientErrors: []string{},
	}

	if s.client == nil {
		s.logger.Debug("fcm delivery skipped (no client)", "user_id", userID, "tokens", len(tokens))
		result.SuccessCount = len(tokens)
		return result, nil
	}

	batch, _ := s.client.SendToMultiple(ctx, tokens, msg)

	for _, token := range batch.FailedTokens {
		_ = s.repo.MarkTokenInactive(ctx, token)
		result.FailedTokens = append(result.FailedTokens, token)
	}

	result.SuccessCount = batch.SuccessCount
	result.FailureCount = batch.FailureCount

	if result.FailureCount > len(tokens)/2 {
		s.circuitBreaker.RecordFailure()
	} else {
		s.circuitBreaker.RecordSuccess()
	}

	return result, nil
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	state            string
	failureCount     int
	failureThreshold int
	successThreshold int
	lastFailureTime  time.Time
	recoveryTimeout  time.Duration
	mu               sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            "closed",
		failureThreshold: failureThreshold,
		successThreshold: 2,
		lastFailureTime:  time.Now(),
		recoveryTimeout:  recoveryTimeout,
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = "open"
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "half-open" {
		cb.successThreshold--
		if cb.successThreshold <= 0 {
			cb.state = "closed"
			cb.failureCount = 0
		}
	} else if cb.state == "closed" {
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != "open" {
		return false
	}

	if time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
		cb.state = "half-open"
		cb.successThreshold = 2
		return false
	}

	return true
}
