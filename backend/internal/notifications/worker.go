package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"studyapp/backend/internal/common/metrics"
)

const (
	queueKey      = "notif:queue"
	retryQueueKey = "notif:retry"
	dlqKey        = "notif:dlq"
	rateLimitKey  = "notif_rate:"
	maxConcurrent = 10
	maxAttempts   = 5
	brpopTimeout  = 5 * time.Second
)

var retryBackoffs = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	240 * time.Second,
	480 * time.Second,
}

// QueueJob is a Redis-queued notification delivery job.
type QueueJob struct {
	ID           string            `json:"id,omitempty"`
	UserID       int64             `json:"user_id"`
	TemplateID   TemplateID        `json:"template_id"`
	Data         map[string]string `json:"data"`
	Channels     []string          `json:"channels"`
	AttemptCount int               `json:"attempt_count,omitempty"`
}

// NotificationWorker processes notification jobs from Valkey/Redis.
type NotificationWorker struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	fcm    *FCMClient
	email  EmailClient
	repo   *Repository
	logger *slog.Logger
	sem    chan struct{}
}

// NewNotificationWorker creates a notification queue worker.
func NewNotificationWorker(
	db *pgxpool.Pool,
	cache *redis.Client,
	fcm *FCMClient,
	email EmailClient,
	repo *Repository,
	logger *slog.Logger,
) *NotificationWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &NotificationWorker{
		db:     db,
		cache:  cache,
		fcm:    fcm,
		email:  email,
		repo:   repo,
		logger: logger,
		sem:    make(chan struct{}, maxConcurrent),
	}
}

// Enqueue pushes a job onto the notification queue.
func (w *NotificationWorker) Enqueue(ctx context.Context, job QueueJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return w.cache.LPush(ctx, queueKey, payload).Err()
}

// Run starts the main queue loop and retry processor until ctx is cancelled.
func (w *NotificationWorker) Run(ctx context.Context) {
	w.logger.Info("notification worker starting")

	go w.runRetryProcessor(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("notification worker stopping")
			return
		default:
		}

		w.updateQueueDepth(ctx)

		result, err := w.cache.BRPop(ctx, brpopTimeout, queueKey).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			w.logger.Error("brpop failed", "error", err)
			continue
		}
		if len(result) < 2 {
			continue
		}

		var job QueueJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			w.logger.Error("invalid queue payload", "error", err)
			continue
		}

		w.sem <- struct{}{}
		go func(j QueueJob) {
			defer func() { <-w.sem }()
			if err := w.processJob(ctx, j); err != nil {
				w.logger.Error("process job failed", "job_id", j.ID, "error", err)
			}
		}(job)
	}
}

func (w *NotificationWorker) runRetryProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := w.cache.BRPop(ctx, brpopTimeout, retryQueueKey).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			w.logger.Error("retry brpop failed", "error", err)
			continue
		}
		if len(result) < 2 {
			continue
		}

		var job QueueJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			w.logger.Error("invalid retry payload", "error", err)
			continue
		}

		if job.AttemptCount >= maxAttempts {
			w.moveToDLQ(ctx, job, "max attempts exceeded")
			continue
		}

		delay := retryBackoffs[min(job.AttemptCount, len(retryBackoffs)-1)]
		w.logger.Info("retry scheduled", "job_id", job.ID, "attempt", job.AttemptCount, "delay", delay)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if err := w.processJob(ctx, job); err != nil {
			w.logger.Error("retry process failed", "job_id", job.ID, "error", err)
		}
	}
}

func (w *NotificationWorker) processJob(ctx context.Context, job QueueJob) error {
	userUUID := UserIDToUUID(job.UserID)

	prefs, err := w.repo.GetUserPreferences(ctx, userUUID)
	if err != nil {
		prefs = DefaultNotificationPreferences(userUUID)
	}

	tmpl, err := Render(job.TemplateID, job.Data)
	if err != nil {
		return w.recordJob(ctx, job, StatusFailed, err.Error())
	}

	if !w.checkRateLimit(ctx, job.UserID) {
		w.logger.Warn("notification rate limited", "user_id", job.UserID)
		return w.recordJob(ctx, job, StatusSkipped, "rate_limited")
	}

	pushOK := false
	emailOK := false
	var lastErr error

	wantPush := channelWanted(job.Channels, ChannelPush) && prefs.PushEnabled
	wantEmail := channelWanted(job.Channels, ChannelEmail) && prefs.EmailEnabled

	if wantPush {
		tokens, err := w.repo.GetActiveDeviceTokens(ctx, userUUID)
		if err != nil {
			lastErr = err
			metrics.RecordNotificationSent(ChannelPush, StatusFailed)
		} else if len(tokens) == 0 {
			w.logger.Debug("no fcm tokens", "user_id", job.UserID)
			metrics.RecordNotificationSent(ChannelPush, StatusSkipped)
		} else if w.fcm != nil {
			msg := FCMMessage{
				Title: tmpl.PushTitle,
				Body:  tmpl.PushBody,
				Data:  job.Data,
			}
			for _, token := range tokens {
				sendErr := w.fcm.SendToToken(ctx, token, msg)
				if sendErr == nil {
					pushOK = true
					metrics.RecordNotificationSent(ChannelPush, StatusDelivered)
					continue
				}
				if isPermanentFCMError(sendErr) {
					_ = w.repo.MarkTokenInactive(ctx, token)
					continue
				}
				if IsRetryable(sendErr) {
					return w.scheduleRetry(ctx, job, sendErr)
				}
				lastErr = sendErr
			}
		} else {
			w.logger.Info("push skipped (no fcm client)", "user_id", job.UserID)
			pushOK = true
			metrics.RecordNotificationSent(ChannelPush, StatusSkipped)
		}
	}

	if wantEmail {
		email, err := w.repo.GetUserEmail(ctx, job.UserID)
		if err != nil {
			w.logger.Error("user email lookup failed", "user_id", job.UserID, "error", err)
		} else if w.email != nil {
			sendErr := w.email.SendTransactional(ctx, EmailRequest{
				To:      email,
				Subject: tmpl.EmailSubj,
				HTML:    tmpl.EmailHTML,
				Text:    tmpl.EmailText,
				Tags:    []string{string(job.TemplateID)},
			})
			if sendErr != nil {
				w.logger.Error("email send failed", "user_id", job.UserID, "error", sendErr)
				lastErr = sendErr
				metrics.RecordNotificationSent(ChannelEmail, StatusFailed)
			} else {
				emailOK = true
				metrics.RecordNotificationSent(ChannelEmail, StatusDelivered)
			}
		}
	}

	status := StatusDelivered
	reason := ""
	if wantPush && !pushOK && wantEmail && !emailOK {
		status = StatusFailed
		reason = "all channels failed"
		if lastErr != nil {
			reason = lastErr.Error()
		}
	} else if wantPush && !pushOK && !wantEmail {
		status = StatusFailed
		if lastErr != nil {
			reason = lastErr.Error()
		} else {
			reason = "push failed"
		}
	} else if !wantPush && !wantEmail {
		status = StatusSkipped
		reason = "channels disabled"
	}

	return w.recordJob(ctx, job, status, reason)
}

func (w *NotificationWorker) updateQueueDepth(ctx context.Context) {
	if w.cache == nil {
		return
	}
	mainLen, err := w.cache.LLen(ctx, queueKey).Result()
	if err != nil {
		return
	}
	retryLen, _ := w.cache.LLen(ctx, retryQueueKey).Result()
	metrics.SetNotificationQueueDepth(float64(mainLen + retryLen))
}

func (w *NotificationWorker) checkRateLimit(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("%s%d", rateLimitKey, userID)
	count, err := w.cache.Incr(ctx, key).Result()
	if err != nil {
		return true
	}
	if count == 1 {
		_ = w.cache.Expire(ctx, key, time.Hour).Err()
	}
	return count <= 10
}

func (w *NotificationWorker) scheduleRetry(ctx context.Context, job QueueJob, err error) error {
	job.AttemptCount++
	if job.AttemptCount >= maxAttempts {
		return w.moveToDLQ(ctx, job, err.Error())
	}
	payload, marshalErr := json.Marshal(job)
	if marshalErr != nil {
		return marshalErr
	}
	return w.cache.LPush(ctx, retryQueueKey, payload).Err()
}

func (w *NotificationWorker) moveToDLQ(ctx context.Context, job QueueJob, reason string) error {
	w.logger.Error("moving job to dlq", "job_id", job.ID, "reason", reason)
	payload, err := json.Marshal(map[string]any{
		"job":    job,
		"reason": reason,
	})
	if err != nil {
		return err
	}
	if err := w.cache.LPush(ctx, dlqKey, payload).Err(); err != nil {
		return err
	}
	return w.recordJob(ctx, job, StatusFailed, reason)
}

func (w *NotificationWorker) recordJob(ctx context.Context, job QueueJob, status, lastError string) error {
	if w.repo == nil {
		return nil
	}

	category := templateCategory(job.TemplateID)
	templateData := make(JSONB, len(job.Data))
	for k, v := range job.Data {
		templateData[k] = v
	}

	dbJob := &NotificationJob{
		ID:             uuid.New(),
		UserID:         UserIDToUUID(job.UserID),
		Channel:        primaryChannel(job.Channels),
		Priority:       PriorityNormal,
		Category:       category,
		TemplateKey:    string(job.TemplateID),
		TemplateData:   templateData,
		Status:         status,
		IdempotencyKey: job.ID,
		LastError:      lastError,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if status == StatusDelivered {
		now := time.Now()
		dbJob.DeliveredAt = &now
	}

	return w.repo.CreateNotificationJob(ctx, dbJob)
}

func channelWanted(channels []string, channel string) bool {
	for _, ch := range channels {
		if ch == channel {
			return true
		}
	}
	return false
}

func primaryChannel(channels []string) string {
	if len(channels) == 0 {
		return ChannelPush
	}
	return channels[0]
}

func templateCategory(id TemplateID) string {
	switch id {
	case TmplQuizReady:
		return CategoryQuizReady
	case TmplPaymentSuccess, TmplPaymentFailed:
		return CategoryPaymentAlert
	case TmplPracticeReminder:
		return CategoryQuizReminder
	default:
		return CategoryNewContent
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Worker processes notifications in the background (legacy DB polling worker).
type Worker struct {
	service       *Service
	logger        Logger
	redisClient   *redis.Client
	instanceID    string
	heartbeatMu   sync.Mutex
	lastHeartbeat time.Time
}

// NewWorker creates a new notification worker.
func NewWorker(service *Service, logger Logger, redisClient *redis.Client) *Worker {
	return &Worker{
		service:     service,
		logger:      logger,
		redisClient: redisClient,
		instanceID:  uuid.New().String(),
	}
}

// Start starts the legacy DB-polling worker.
func (w *Worker) Start(ctx context.Context, numWorkers int) {
	w.logger.Info("notification worker starting", "instance_id", w.instanceID, "workers", numWorkers)

	for i := 0; i < numWorkers; i++ {
		go w.processQueue(ctx)
	}

	go w.metricsReporter(ctx)
	go w.heartbeatWriter(ctx)
}

func (w *Worker) processQueue(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker stopping")
			return
		case <-ticker.C:
			w.processPriority(ctx, "high", 20)
			w.processPriority(ctx, "normal", 14)
			w.processPriority(ctx, "low", 6)
		}
	}
}

func (w *Worker) processPriority(ctx context.Context, priority string, batchSize int) {
	jobs, err := w.service.repo.GetPendingJobs(ctx, priority, batchSize)
	if err != nil {
		w.logger.Error("failed to get pending jobs", "priority", priority, "error", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	w.logger.Debug("processing jobs", "priority", priority, "count", len(jobs))

	for _, job := range jobs {
		_ = w.service.processJob(ctx, job)
	}
}

func (w *Worker) heartbeatWriter(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.redisClient.Del(ctx, fmt.Sprintf("worker_heartbeat:%s", w.instanceID))
			return
		case <-ticker.C:
			w.heartbeatMu.Lock()
			w.lastHeartbeat = time.Now()
			w.heartbeatMu.Unlock()

			key := fmt.Sprintf("worker_heartbeat:%s", w.instanceID)
			w.redisClient.Set(ctx, key, time.Now().Unix(), 30*time.Second)

			w.logger.Debug("heartbeat written", "instance_id", w.instanceID)
		}
	}
}

// HealthCheck checks if worker is healthy.
func (w *Worker) HealthCheck(ctx context.Context) bool {
	w.heartbeatMu.Lock()
	defer w.heartbeatMu.Unlock()
	return time.Since(w.lastHeartbeat) < 30*time.Second
}

func (w *Worker) metricsReporter(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.logger.Debug("metrics collected", "instance_id", w.instanceID)
		}
	}
}

// CleanupCronjob handles periodic cleanup tasks.
type CleanupCronjob struct {
	repo   *Repository
	logger Logger
}

// NewCleanupCronjob creates a new cleanup cronjob.
func NewCleanupCronjob(repo *Repository, logger Logger) *CleanupCronjob {
	return &CleanupCronjob{repo: repo, logger: logger}
}

// Start starts the cleanup cronjob.
func (c *CleanupCronjob) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runCleanup(ctx)
		}
	}
}

func (c *CleanupCronjob) runCleanup(ctx context.Context) {
	c.logger.Info("running cleanup tasks")

	if err := c.repo.CleanupStaleTokens(ctx); err != nil {
		c.logger.Error("failed to cleanup stale tokens", "error", err)
	}
	if err := c.repo.HardDeleteOldTokens(ctx); err != nil {
		c.logger.Error("failed to delete old tokens", "error", err)
	}

	c.logger.Info("cleanup tasks completed")
}

// HealthCheckCronjob checks worker health.
type HealthCheckCronjob struct {
	redisClient *redis.Client
	logger      Logger
}

// NewHealthCheckCronjob creates a new health check cronjob.
func NewHealthCheckCronjob(redisClient *redis.Client, logger Logger) *HealthCheckCronjob {
	return &HealthCheckCronjob{redisClient: redisClient, logger: logger}
}

// Start starts the health check cronjob.
func (hc *HealthCheckCronjob) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkWorkerHealth(ctx)
		}
	}
}

func (hc *HealthCheckCronjob) checkWorkerHealth(ctx context.Context) {
	keys, err := hc.redisClient.Keys(ctx, "worker_heartbeat:*").Result()
	if err != nil {
		hc.logger.Error("failed to get worker heartbeats", "error", err)
		return
	}

	if len(keys) == 0 {
		hc.logger.Warn("no worker heartbeats found")
		return
	}

	for _, key := range keys {
		ttl := hc.redisClient.TTL(ctx, key).Val()
		if ttl < 0 {
			hc.logger.Warn("worker heartbeat expired", "worker", key)
		}
	}

	hc.logger.Debug("health check completed", "active_workers", len(keys))
}
