# Notifications Module

Production-ready notification system with push (FCM) and email support.

## Architecture Overview

```
┌─ HTTP Handlers (handler.go)
│  - POST /auth/devices/register
│  - GET/PUT /user/preferences
│  - POST /notifications/send
│  - POST /webhooks/email/resend
│
├─ Business Logic (service.go)
│  - SendNotification()
│  - RegisterDevice()
│  - ProcessNotifications()
│  - Preference management
│  - Rate limiting + quiet hours
│
├─ Database (repository.go)
│  - CRUD for tokens, jobs, preferences
│  - Queries for pending jobs
│  - Cleanup tasks
│
├─ Channels
│  ├─ FCM (fcm.go)
│  │  - SendMulticast() to multiple devices
│  │  - Error classification
│  │  - Circuit breaker + rate limiting
│  │
│  └─ Email (email.go)
│     - SendEmail() via Resend/SES
│     - Bounce/complaint handling
│     - Signature verification
│
└─ Background Worker (worker.go)
   - Process notification queue
   - Weighted priority scheduling
   - Heartbeat + health monitoring
   - Cleanup cronjob
```

## Models (model.go)

### Core Models
- `FCMDeviceToken` - Device registration with lifecycle tracking
- `NotificationJob` - Queued notification with retry logic
- `NotificationPreferences` - User notification settings
- `EmailEvent` - Bounce/complaint tracking
- `NotificationTemplate` - Email/push templates

### Constants
- Categories: `quiz_ready`, `payment_alert`, `weekly_digest`, etc.
- Channels: `push`, `email`, `sms`
- Priorities: `high`, `normal`, `low`
- Status: `pending`, `processing`, `sent`, `delivered`, `failed`

## Features

### ✅ Implemented
- Device token registration with rotation support
- Multi-channel notifications (push/email/SMS)
- User preference management
- Frequency capping (max notifications per day)
- Quiet hours (no notifications during sleep time)
- Timezone support for quiet hours
- Exponential backoff retry (up to 24h)
- Circuit breaker for FCM failures
- Rate limiting (token bucket)
- Worker health monitoring (heartbeat)
- Weighted priority scheduling (prevent starvation)
- Deduplication via idempotency keys
- Row-level locking for concurrent safety

### 🔄 To Implement
- Firebase Cloud Messaging API integration
- Resend/AWS SES email integration
- Template hot-reload from database
- Webhook signature verification
- i18n template support
- Delivery status tracking
- Analytics dashboard

## Usage

### 1. Register a Device

```bash
curl -X POST http://localhost:8080/auth/devices/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "token": "FCM_DEVICE_TOKEN",
    "platform": "ios",
    "app_version": "1.2.3",
    "os_version": "14.0"
  }'
```

### 2. Get User Preferences

```bash
curl http://localhost:8080/user/preferences \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. Update Preferences

```bash
curl -X PUT http://localhost:8080/user/preferences \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "push_enabled": true,
    "email_enabled": true,
    "max_push_per_day": 10,
    "quiet_hours_start": "21:00",
    "quiet_hours_end": "08:00",
    "quiet_hours_tz": "Asia/Kolkata"
  }'
```

### 4. Send a Notification

```bash
curl -X POST http://localhost:8080/notifications/send \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "channel": "push",
    "category": "quiz_ready",
    "template_key": "quiz_completion_notification",
    "data": {
      "quiz_id": "abc123",
      "score": "85",
      "subject": "Physics"
    }
  }'
```

## Database Schema

Migration: `migrations/0004_notifications.sql`

### Tables
- `fcm_device_tokens` - Device token management
- `notification_jobs` - Notification queue
- `notification_preferences` - User settings
- `email_events` - Email delivery tracking
- `notification_templates` - Email/push templates
- `notification_delivery_logs` - Delivery status history

## Configuration

### Environment Variables

```bash
# FCM Configuration
FCM_PROJECT_ID=your-firebase-project
FCM_SERVICE_ACCOUNT_KEY=path/to/service-account.json

# Email Configuration (Resend)
RESEND_API_KEY=your-resend-api-key
RESEND_WEBHOOK_SECRET=your-webhook-secret

# Worker Configuration
NOTIFICATION_WORKER_COUNT=3
NOTIFICATION_BATCH_SIZE=50
NOTIFICATION_POLL_INTERVAL=1s

# Cleanup Configuration
CLEANUP_INTERVAL=1h
```

### Rate Limits

- FCM: 100 requests/second
- Email: Depends on provider (Resend ~50/sec)
- Retry: Max 5 attempts over 24 hours

### Frequency Limits (Per User)

- Push: 10 notifications/day (configurable)
- Email: 5 notifications/week (configurable)
- SMS: 1 notification/day (configurable)

## Error Handling

### FCM Errors

| Error Type | Permanent? | Action |
|-----------|-----------|--------|
| `token_invalid` | Yes | Mark token inactive |
| `device_uninstalled` | Yes | Mark token inactive |
| `fcm_config_error` | Yes | Alert ops |
| `rate_limited` | No | Retry with backoff |
| `timeout` | No | Retry with backoff |

### Email Errors

| Error Type | Permanent? | Action |
|-----------|-----------|--------|
| `invalid_email` | Yes | Fail, don't retry |
| `bounced` | Yes | Mark user email as bounced |
| `complained` | Yes | Mark user email as complained |
| `timeout` | No | Retry with backoff |

## Monitoring

### Metrics to Track

- Notifications sent/hour (by channel)
- Delivery rate (%)
- Bounce rate (%)
- Average processing time
- Queue depth (pending jobs)
- Worker health (heartbeat)

### Alerting

```
Alert if:
  - Delivery rate < 95%
  - Bounce rate > 5%
  - Queue depth > 1000
  - Worker heartbeat missing > 30s
  - FCM circuit breaker open
```

## Testing

### Unit Tests

```bash
cd /e:/study-app/backend
go test ./internal/notifications -v
```

### Integration Tests

```bash
# Test with real database and Resend
go test ./internal/notifications -v -tags=integration
```

### Load Test

```bash
# Simulate 1000 notifications/sec for 5 minutes
go test ./internal/notifications -bench=BenchmarkProcessNotifications -benchmem
```

## Deployment

### Prerequisites
1. PostgreSQL 13+ with JSON support
2. Firebase Cloud Messaging project
3. Resend email account
4. Redis (for rate limiting + worker coordination)

### Steps

1. **Run migration:**
   ```bash
   psql -U postgres study_app < migrations/0004_notifications.sql
   ```

2. **Set environment variables:**
   ```bash
   export FCM_PROJECT_ID=...
   export RESEND_API_KEY=...
   # ... other vars
   ```

3. **Start worker:**
   ```bash
   go run cmd/api/main.go --enable-notification-worker
   ```

## Roadmap

### Phase 1 (MVP)
- ✅ Token management
- ✅ Preference management
- ✅ FCM rate limiting + circuit breaker
- ✅ Email retry logic + webhooks
- ✅ Worker + monitoring

### Phase 2
- [ ] Firebase Cloud Messaging API
- [ ] Resend email API
- [ ] Template hot-reload
- [ ] i18n support
- [ ] Delivery analytics

### Phase 3
- [ ] SMS channel (Twilio/AWS SNS)
- [ ] In-app notification center
- [ ] Notification history
- [ ] Advanced segmentation
- [ ] A/B testing

## Support

For issues or questions, refer to:
- Architecture: See Section 7 in production roadmap
- Issues: Check GitHub issues
- Docs: See README in backend folder
