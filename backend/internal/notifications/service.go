package notifications

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service handles notification business logic
type Service struct {
	repo         *Repository
	fcmService   *FCMService
	emailService *EmailService
	logger       Logger

	// Rate limiting
	rateLimiters map[string]*TokenBucket
	rateMutex    sync.RWMutex
}

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// NewService creates a new notification service
func NewService(repo *Repository, fcmService *FCMService, emailService *EmailService, logger Logger) *Service {
	return &Service{
		repo:         repo,
		fcmService:   fcmService,
		emailService: emailService,
		logger:       logger,
		rateLimiters: make(map[string]*TokenBucket),
	}
}

// RegisterDevice registers a new FCM device token
func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, token, platform string, oldTokenID *uuid.UUID) error {

	// Check if token already exists
	tokens, _ := s.repo.GetActiveDeviceTokens(ctx, userID)
	for _, t := range tokens {
		if t == token {
			// Token already registered, just update last seen
			return s.repo.UpdateTokenLastSeen(ctx, uuid.New())
		}
	}

	// If old token provided, mark as replaced
	if oldTokenID != nil {
		_ = s.repo.MarkTokenInactiveByID(ctx, *oldTokenID)
	}

	// Create new token
	deviceToken := &FCMDeviceToken{
		ID:           uuid.New(),
		UserID:       userID,
		Token:        token,
		Platform:     platform,
		PushEnabled:  true,
		IsActive:     true,
		FailureCount: 0,
		LastSeen:     time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.repo.CreateFCMDeviceToken(ctx, deviceToken)
}

// SendNotification creates and queues a notification
func (s *Service) SendNotification(ctx context.Context, userID uuid.UUID, channel, category, templateKey string, data map[string]interface{}) error {

	// Validate category
	if _, exists := CategoryConfig[category]; !exists {
		return fmt.Errorf("invalid category: %s", category)
	}

	if data == nil {
		data = map[string]interface{}{}
	}
	if _, ok := data["auth_user_id"]; !ok {
		if authID, ok := ctx.Value("user_id").(int64); ok {
			data["auth_user_id"] = authID
		}
	}

	// Validate channel
	catConfig := CategoryConfig[category]
	validChannel := false
	for _, ch := range catConfig.AllowedChannels {
		if ch == channel {
			validChannel = true
			break
		}
	}
	if !validChannel {
		return fmt.Errorf("channel %s not allowed for category %s", channel, category)
	}

	// Generate idempotency key
	idempotencyKey := fmt.Sprintf("%s:%s:%s:%d", category, userID, templateKey, time.Now().Unix())

	// Check for duplicates
	exists, _ := s.repo.JobExistsByIdempotencyKey(ctx, idempotencyKey)
	if exists {
		s.logger.Debug("notification job already exists", "key", idempotencyKey)
		return nil
	}

	// Determine priority
	priority := catConfig.PushPriority
	if channel == "email" && priority == "high" {
		priority = "normal" // Emails don't use high priority
	}

	// Create notification job
	job := &NotificationJob{
		ID:             uuid.New(),
		UserID:         userID,
		Channel:        channel,
		Priority:       priority,
		Category:       category,
		TemplateKey:    templateKey,
		TemplateData:   JSONB(data),
		Status:         StatusPending,
		IdempotencyKey: idempotencyKey,
		RetryCount:     0,
		MaxRetries:     5,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return s.repo.CreateNotificationJob(ctx, job)
}

// ProcessNotifications runs the notification processing loop
func (s *Service) ProcessNotifications(ctx context.Context, batchSize int) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Process by priority (weighted)
			s.processPriority(ctx, "high", batchSize)
			s.processPriority(ctx, "normal", batchSize)
			s.processPriority(ctx, "low", batchSize)
		}
	}
}

// processPriority processes notifications of given priority
func (s *Service) processPriority(ctx context.Context, priority string, batchSize int) {
	jobs, err := s.repo.GetPendingJobs(ctx, priority, batchSize)
	if err != nil || len(jobs) == 0 {
		return
	}

	for _, job := range jobs {
		_ = s.processJob(ctx, job)
	}
}

// processJob processes a single notification job
func (s *Service) processJob(ctx context.Context, job *NotificationJob) error {

	// Check user preferences
	prefs, err := s.repo.GetUserPreferences(ctx, job.UserID)
	if err != nil {
		return s.repo.FailJob(ctx, job.ID, "preferences_error")
	}

	// Check if channel enabled
	if !s.isChannelEnabled(job.Channel, prefs) {
		return s.repo.UpdateJobStatus(ctx, job.ID, "skipped")
	}

	// Check if category enabled
	if !s.isCategoryEnabled(job.Category, prefs) {
		return s.repo.UpdateJobStatus(ctx, job.ID, "skipped")
	}

	// Check frequency capping
	if s.isFrequencyCapped(ctx, job.UserID, job.Channel, prefs) {
		return s.repo.RescheduleJob(ctx, job.ID, 24*time.Hour, "frequency_capped")
	}

	// Check quiet hours
	if s.isInQuietHours(ctx, job.UserID, job.Priority, prefs) && job.Priority != "high" {
		nextEnd := s.getNextQuietHourEnd(prefs)
		return s.repo.RescheduleJob(ctx, job.ID, time.Until(nextEnd), "quiet_hours")
	}

	// Update status to processing
	_ = s.repo.UpdateJobStatus(ctx, job.ID, StatusProcessing)

	// Route by channel
	switch job.Channel {
	case ChannelPush:
		return s.sendPushWithFallback(ctx, job)
	case ChannelEmail:
		return s.sendEmailWithRetry(ctx, job)
	}

	return nil
}

// sendPushWithFallback sends push notification with email fallback
func (s *Service) sendPushWithFallback(ctx context.Context, job *NotificationJob) error {

	tokens, err := s.repo.GetActiveDeviceTokens(ctx, job.UserID)
	if err != nil || len(tokens) == 0 {
		// No active devices, fallback to email
		job.Channel = ChannelEmail
		return s.sendEmailWithRetry(ctx, job)
	}

	// Send push to all devices
	data := map[string]string{
		"category": job.Category,
		"action":   s.extractAction(job.TemplateData),
	}

	result, err := s.fcmService.SendMulticast(ctx, job.UserID.String(), tokens, data)
	if err != nil {
		return s.retryWithBackoff(ctx, job)
	}

	// If all failed, try email
	if result.SuccessCount == 0 {
		job.Channel = ChannelEmail
		return s.sendEmailWithRetry(ctx, job)
	}

	return s.repo.UpdateJobStatus(ctx, job.ID, StatusSent)
}

// sendEmailWithRetry sends email with retry logic
func (s *Service) sendEmailWithRetry(ctx context.Context, job *NotificationJob) error {

	// Render template
	subject, body, err := s.renderTemplate(ctx, job.TemplateKey, job.TemplateData)
	if err != nil {
		return s.repo.FailJob(ctx, job.ID, fmt.Sprintf("template_error: %v", err))
	}

	// Send email
	email, err := s.lookupUserEmail(ctx, job)
	if err != nil {
		return s.repo.FailJob(ctx, job.ID, fmt.Sprintf("email_lookup: %v", err))
	}

	err = s.emailService.SendEmail(ctx, job.ID, email, subject, body)

	if err != nil {
		// Check if retry possible
		if job.RetryCount >= job.MaxRetries || time.Since(job.CreatedAt) > 24*time.Hour {
			return s.repo.FailJob(ctx, job.ID, fmt.Sprintf("max_retries: %v", err))
		}

		// Calculate exponential backoff
		backoff := time.Duration(math.Pow(2, float64(job.RetryCount))) * time.Second
		jitter := time.Duration(rand.Intn(int(backoff / 2)))

		return s.repo.RescheduleJob(ctx, job.ID, backoff+jitter, fmt.Sprintf("retry_%d", job.RetryCount+1))
	}

	return s.repo.UpdateJobStatus(ctx, job.ID, StatusSent)
}

// isChannelEnabled checks if channel is enabled in preferences
func (s *Service) isChannelEnabled(channel string, prefs *NotificationPreferences) bool {
	switch channel {
	case ChannelPush:
		return prefs.PushEnabled
	case ChannelEmail:
		return prefs.EmailEnabled
	case ChannelSMS:
		return prefs.SMSEnabled
	}
	return false
}

// isCategoryEnabled checks if category is enabled in preferences
func (s *Service) isCategoryEnabled(category string, prefs *NotificationPreferences) bool {
	if catConfig, exists := CategoryConfig[category]; exists && !catConfig.DefaultEnabled {
		// Opt-in category, check user preferences
		if prefs.Preferences == nil {
			return false
		}
		prefMap := map[string]interface{}(prefs.Preferences)
		if val, exists := prefMap[category]; exists {
			if catPrefs, ok := val.(map[string]interface{}); ok {
				if enabled, ok := catPrefs["enabled"].(bool); ok {
					return enabled
				}
			}
		}
		return false
	}
	return true
}

// isFrequencyCapped checks if user has hit frequency limit
func (s *Service) isFrequencyCapped(ctx context.Context, userID uuid.UUID, channel string, prefs *NotificationPreferences) bool {

	var maxPerDay int
	switch channel {
	case ChannelPush:
		maxPerDay = prefs.MaxPushPerDay
	case ChannelEmail:
		maxPerDay = prefs.MaxEmailPerWeek / 7
	}

	if maxPerDay == 0 {
		return false
	}

	count, _ := s.repo.CountNotifications(ctx, userID, channel, []string{StatusPending, StatusProcessing, StatusSent, StatusDelivered}, 24*time.Hour)
	return count >= maxPerDay
}

// isInQuietHours checks if current time is in quiet hours
func (s *Service) isInQuietHours(ctx context.Context, userID uuid.UUID, priority string, prefs *NotificationPreferences) bool {

	// High priority bypasses quiet hours
	if priority == "high" {
		return false
	}

	if prefs.QuietHoursStart == nil || prefs.QuietHoursEnd == nil {
		return false
	}

	// Load user's timezone
	loc, err := time.LoadLocation(prefs.QuietHoursTZ)
	if err != nil {
		return false
	}

	userNow := time.Now().In(loc)
	start, _ := time.Parse("15:04", *prefs.QuietHoursStart)
	end, _ := time.Parse("15:04", *prefs.QuietHoursEnd)

	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()
	nowMinutes := userNow.Hour()*60 + userNow.Minute()

	if endMinutes > startMinutes {
		// Normal case: 21:00 - 08:00 doesn't wrap
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}

	// Wraps midnight
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}

// getNextQuietHourEnd calculates next quiet hour end time
func (s *Service) getNextQuietHourEnd(prefs *NotificationPreferences) time.Time {
	if prefs.QuietHoursEnd == nil {
		return time.Now()
	}

	loc, _ := time.LoadLocation(prefs.QuietHoursTZ)
	userNow := time.Now().In(loc)

	end, _ := time.Parse("15:04", *prefs.QuietHoursEnd)
	nextEnd := time.Date(userNow.Year(), userNow.Month(), userNow.Day(),
		end.Hour(), end.Minute(), 0, 0, loc)

	if nextEnd.Before(userNow) {
		nextEnd = nextEnd.AddDate(0, 0, 1) // Tomorrow
	}

	return nextEnd.In(time.UTC)
}

// retryWithBackoff handles exponential backoff retry
func (s *Service) retryWithBackoff(ctx context.Context, job *NotificationJob) error {
	if job.RetryCount >= job.MaxRetries || time.Since(job.CreatedAt) > 24*time.Hour {
		return s.repo.FailJob(ctx, job.ID, "max_retries_exceeded")
	}

	backoff := time.Duration(math.Pow(2, float64(job.RetryCount))) * time.Second
	jitter := time.Duration(rand.Intn(int(backoff / 2)))

	return s.repo.RescheduleJob(ctx, job.ID, backoff+jitter, fmt.Sprintf("retry_%d", job.RetryCount+1))
}

// renderTemplate renders a notification template.
func (s *Service) renderTemplate(ctx context.Context, key string, data JSONB) (subject, body string, err error) {
	stringData := make(map[string]string, len(data))
	for k, v := range data {
		stringData[k] = fmt.Sprintf("%v", v)
	}

	if rendered, renderErr := Render(TemplateID(key), stringData); renderErr == nil {
		return rendered.EmailSubj, rendered.EmailHTML, nil
	}

	subject = fmt.Sprintf("Notification: %s", key)
	body = fmt.Sprintf("A notification for %s is ready.", key)
	if len(data) > 0 {
		subject = fmt.Sprintf("Notification: %s for {{name}}", key)
		body = fmt.Sprintf("A notification for %s is ready for {{name}}.", key)
	}

	if s.repo != nil {
		tmpl, err := s.repo.GetTemplate(ctx, key)
		if err == nil {
			subject = tmpl.Subject
			body = tmpl.BodyHTML
			if body == "" {
				body = tmpl.BodyText
			}
		}
	}

	for k, v := range data {
		placeholder := fmt.Sprintf("{{%s}}", k)
		value := fmt.Sprintf("%v", v)
		subject = strings.ReplaceAll(subject, placeholder, value)
		body = strings.ReplaceAll(body, placeholder, value)
	}

	return subject, body, nil
}

// BuildPushPayload creates a consistent payload shape for push delivery.
func (s *Service) BuildPushPayload(job *NotificationJob) map[string]string {
	payload := map[string]string{
		"category":     job.Category,
		"template_key": job.TemplateKey,
		"action":       s.extractAction(job.TemplateData),
	}
	for key, value := range job.TemplateData {
		payload[key] = fmt.Sprintf("%v", value)
	}
	return payload
}

// extractAction extracts action from template data
func userIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, fmt.Errorf("context is nil")
	}

	if userID, ok := ctx.Value("user_id").(int64); ok {
		return UserIDToUUID(userID), nil
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		parsed, err := uuid.Parse(userID)
		if err == nil {
			return parsed, nil
		}
		id, err := strconv.ParseInt(userID, 10, 64)
		if err == nil {
			return UserIDToUUID(id), nil
		}
		return uuid.Nil, fmt.Errorf("invalid user_id in context")
	}
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID, nil
	}
	return uuid.Nil, fmt.Errorf("user id not found in context")
}

func (s *Service) extractAction(data JSONB) string {
	if action, ok := data["action"]; ok {
		return fmt.Sprintf("%v", action)
	}
	return "open_app"
}

func (s *Service) lookupUserEmail(ctx context.Context, job *NotificationJob) (string, error) {
	if raw, ok := job.TemplateData["auth_user_id"]; ok {
		id, err := strconv.ParseInt(fmt.Sprintf("%v", raw), 10, 64)
		if err == nil {
			return s.repo.GetUserEmail(ctx, id)
		}
	}
	return "", fmt.Errorf("auth_user_id missing from template data")
}

// TokenBucket implements rate limiting
type TokenBucket struct {
	capacity   int
	tokens     float64
	lastRefill time.Time
	rate       float64 // tokens per second
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rps int) *TokenBucket {
	return &TokenBucket{
		capacity:   rps,
		tokens:     float64(rps),
		lastRefill: time.Now(),
		rate:       float64(rps),
	}
}

// Allow checks if operation is allowed
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// Refill tokens
	tb.tokens = math.Min(float64(tb.capacity), tb.tokens+elapsed*tb.rate)

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}
