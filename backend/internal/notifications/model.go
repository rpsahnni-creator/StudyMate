package notifications

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FCMDeviceToken represents a Firebase Cloud Messaging device token
type FCMDeviceToken struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	UserID      uuid.UUID  `db:"user_id" json:"user_id"`
	Token       string     `db:"token" json:"token"`
	Platform    string     `db:"platform" json:"platform"` // 'ios' | 'android'
	AppVersion  string     `db:"app_version" json:"app_version"`
	OSVersion   string     `db:"os_version" json:"os_version"`
	PushEnabled bool       `db:"push_enabled" json:"push_enabled"`
	LastSeen    time.Time  `db:"last_seen" json:"last_seen"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	FailureCount int       `db:"failure_count" json:"failure_count"`
	ReplacedBy  *uuid.UUID `db:"replaced_by" json:"replaced_by"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// NotificationJob represents a queued notification to be sent
type NotificationJob struct {
	ID              uuid.UUID           `db:"id" json:"id"`
	UserID          uuid.UUID           `db:"user_id" json:"user_id"`
	Channel         string              `db:"channel" json:"channel"` // 'push' | 'email' | 'sms'
	Priority        string              `db:"priority" json:"priority"`
	Category        string              `db:"category" json:"category"`
	TemplateKey     string              `db:"template_key" json:"template_key"`
	TemplateData    JSONB               `db:"template_data" json:"template_data"`
	Status          string              `db:"status" json:"status"`
	IdempotencyKey  string              `db:"idempotency_key" json:"idempotency_key"`
	RetryCount      int                 `db:"retry_count" json:"retry_count"`
	MaxRetries      int                 `db:"max_retries" json:"max_retries"`
	LastError       string              `db:"last_error" json:"last_error"`
	NextRetryAt     *time.Time          `db:"next_retry_at" json:"next_retry_at"`
	SentAt          *time.Time          `db:"sent_at" json:"sent_at"`
	DeliveredAt     *time.Time          `db:"delivered_at" json:"delivered_at"`
	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at" json:"updated_at"`
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences struct {
	ID               uuid.UUID           `db:"id" json:"id"`
	UserID           uuid.UUID           `db:"user_id" json:"user_id"`
	PushEnabled      bool                `db:"push_enabled" json:"push_enabled"`
	EmailEnabled     bool                `db:"email_enabled" json:"email_enabled"`
	SMSEnabled       bool                `db:"sms_enabled" json:"sms_enabled"`
	Preferences      JSONB               `db:"preferences" json:"preferences"`
	MaxPushPerDay    int                 `db:"max_push_per_day" json:"max_push_per_day"`
	MaxEmailPerWeek  int                 `db:"max_email_per_week" json:"max_email_per_week"`
	QuietHoursStart  *string             `db:"quiet_hours_start" json:"quiet_hours_start"`
	QuietHoursEnd    *string             `db:"quiet_hours_end" json:"quiet_hours_end"`
	QuietHoursTZ     string              `db:"quiet_hours_tz" json:"quiet_hours_tz"`
	CreatedAt        time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time           `db:"updated_at" json:"updated_at"`
}

// EmailEvent represents email delivery events (bounce, complaint, etc.)
type EmailEvent struct {
	ID                uuid.UUID           `db:"id" json:"id"`
	UserID            *uuid.UUID          `db:"user_id" json:"user_id"`
	EmailAddress      string              `db:"email_address" json:"email_address"`
	EventType         string              `db:"event_type" json:"event_type"` // 'bounce' | 'complaint' | 'delivery'
	ProviderEventID   string              `db:"provider_event_id" json:"provider_event_id"`
	Metadata          JSONB               `db:"metadata" json:"metadata"`
	CreatedAt         time.Time           `db:"created_at" json:"created_at"`
}

// NotificationTemplate represents email/push templates
type NotificationTemplate struct {
	ID        uuid.UUID           `db:"id" json:"id"`
	Key       string              `db:"key" json:"key"`
	Subject   string              `db:"subject" json:"subject"`
	BodyHTML  string              `db:"body_html" json:"body_html"`
	BodyText  string              `db:"body_text" json:"body_text"`
	BodyI18n  JSONB               `db:"body_i18n" json:"body_i18n"`
	Variables JSONB               `db:"variables" json:"variables"`
	CreatedAt time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt time.Time           `db:"updated_at" json:"updated_at"`
}

// NotificationDeliveryLog tracks delivery status
type NotificationDeliveryLog struct {
	ID              uuid.UUID           `db:"id" json:"id"`
	NotificationID  uuid.UUID           `db:"notification_job_id" json:"notification_job_id"`
	UserID          uuid.UUID           `db:"user_id" json:"user_id"`
	Channel         string              `db:"channel" json:"channel"`
	ProviderID      string              `db:"provider_id" json:"provider_id"`
	Status          string              `db:"status" json:"status"`
	StatusTimestamp time.Time           `db:"status_timestamp" json:"status_timestamp"`
	Metadata        JSONB               `db:"metadata" json:"metadata"`
	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
}

// MulticastResult represents result of sending to multiple devices
type MulticastResult struct {
	SuccessCount    int      `json:"success_count"`
	FailureCount    int      `json:"failure_count"`
	FailedTokens    []string `json:"failed_tokens"`
	TransientErrors []string `json:"transient_errors"`
}

// JSONB is a custom type for JSON data in database
type JSONB map[string]interface{}

// Value implements driver.Valuer
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = JSONB{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*j = JSONB{}
			return nil
		}
		return json.Unmarshal(v, j)
	case string:
		if v == "" {
			*j = JSONB{}
			return nil
		}
		return json.Unmarshal([]byte(v), j)
	default:
		return json.Unmarshal([]byte("{}"), j)
	}
}

// NotificationCategory defines a category with its settings
type NotificationCategory struct {
	Key                string   `json:"key"`
	DefaultEnabled     bool     `json:"default_enabled"`
	AllowedChannels    []string `json:"allowed_channels"`
	PushPriority       string   `json:"push_priority"` // 'high' | 'normal' | 'low'
	MaxFrequencyPerDay int      `json:"max_frequency_per_day"`
	RequiresConsent    bool     `json:"requires_consent"`
}

// Constants for notification categories
const (
	CategoryQuizReady      = "quiz_ready"
	CategoryQuizReminder   = "quiz_reminder"
	CategoryPaymentAlert   = "payment_alert"
	CategoryOTPVerify      = "otp_verify"
	CategoryWeeklyDigest   = "weekly_digest"
	CategoryNewContent     = "new_content"
	CategorySkillUpdate    = "skill_update"
	CategoryPromotion      = "promotion"
	CategoryFeatureAnnounce = "feature_announce"
	CategoryFeedback       = "feedback_request"
)

// Constants for channels
const (
	ChannelPush  = "push"
	ChannelEmail = "email"
	ChannelSMS   = "sms"
)

// Constants for priorities
const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// Constants for status
const (
	StatusPending     = "pending"
	StatusProcessing  = "processing"
	StatusSent        = "sent"
	StatusDelivered   = "delivered"
	StatusFailed      = "failed"
	StatusSkipped     = "skipped"
	StatusQueued      = "queued"
)

// CategoryConfig maps category names to their configuration
var CategoryConfig = map[string]NotificationCategory{
	CategoryQuizReady: {
		Key:                CategoryQuizReady,
		DefaultEnabled:     true,
		AllowedChannels:    []string{ChannelPush, ChannelEmail},
		PushPriority:       PriorityHigh,
		MaxFrequencyPerDay: 20,
		RequiresConsent:    false,
	},
	CategoryQuizReminder: {
		Key:                CategoryQuizReminder,
		DefaultEnabled:     true,
		AllowedChannels:    []string{ChannelPush, ChannelEmail},
		PushPriority:       PriorityNormal,
		MaxFrequencyPerDay: 5,
		RequiresConsent:    false,
	},
	CategoryPaymentAlert: {
		Key:                CategoryPaymentAlert,
		DefaultEnabled:     true,
		AllowedChannels:    []string{ChannelPush, ChannelEmail},
		PushPriority:       PriorityHigh,
		MaxFrequencyPerDay: 10,
		RequiresConsent:    false,
	},
	CategoryOTPVerify: {
		Key:                CategoryOTPVerify,
		DefaultEnabled:     true,
		AllowedChannels:    []string{ChannelPush, ChannelEmail, ChannelSMS},
		PushPriority:       PriorityHigh,
		MaxFrequencyPerDay: 20,
		RequiresConsent:    false,
	},
	CategoryWeeklyDigest: {
		Key:                CategoryWeeklyDigest,
		DefaultEnabled:     true,
		AllowedChannels:    []string{ChannelEmail},
		PushPriority:       PriorityLow,
		MaxFrequencyPerDay: 1,
		RequiresConsent:    false,
	},
	CategoryPromotion: {
		Key:                CategoryPromotion,
		DefaultEnabled:     false,
		AllowedChannels:    []string{ChannelPush, ChannelEmail},
		PushPriority:       PriorityLow,
		MaxFrequencyPerDay: 2,
		RequiresConsent:    true,
	},
}
