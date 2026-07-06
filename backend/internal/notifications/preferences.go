package notifications

import (
	"time"

	"github.com/google/uuid"
)

const (
	defaultMaxPushPerDay   = 10
	defaultMaxEmailPerWeek = 5
	defaultQuietHoursTZ    = "Asia/Kolkata"
)

// DefaultNotificationPreferences returns platform defaults for a user without a stored row.
func DefaultNotificationPreferences(userID uuid.UUID) *NotificationPreferences {
	now := time.Now().UTC()
	return &NotificationPreferences{
		ID:              uuid.New(),
		UserID:          userID,
		PushEnabled:     true,
		EmailEnabled:    true,
		SMSEnabled:      false,
		Preferences:     JSONB{},
		MaxPushPerDay:   defaultMaxPushPerDay,
		MaxEmailPerWeek: defaultMaxEmailPerWeek,
		QuietHoursTZ:    defaultQuietHoursTZ,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// normalizeNotificationPreferences fills zero-value fields before persistence.
func normalizeNotificationPreferences(prefs *NotificationPreferences) {
	if prefs == nil {
		return
	}
	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}
	if prefs.Preferences == nil {
		prefs.Preferences = JSONB{}
	}
	if prefs.MaxPushPerDay <= 0 {
		prefs.MaxPushPerDay = defaultMaxPushPerDay
	}
	if prefs.MaxEmailPerWeek <= 0 {
		prefs.MaxEmailPerWeek = defaultMaxEmailPerWeek
	}
	if prefs.QuietHoursTZ == "" {
		prefs.QuietHoursTZ = defaultQuietHoursTZ
	}
	now := time.Now().UTC()
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = now
	}
	prefs.UpdatedAt = now
}
