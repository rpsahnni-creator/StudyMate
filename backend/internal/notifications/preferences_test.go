package notifications

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestJSONBScan_NullValue(t *testing.T) {
	var j JSONB
	if err := j.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	if j == nil || len(j) != 0 {
		t.Fatalf("expected empty map, got %#v", j)
	}
}

func TestJSONBScan_EmptyBytes(t *testing.T) {
	var j JSONB
	if err := j.Scan([]byte{}); err != nil {
		t.Fatalf("scan empty bytes: %v", err)
	}
	if len(j) != 0 {
		t.Fatalf("expected empty map, got %#v", j)
	}
}

func TestJSONBValue_NilMap(t *testing.T) {
	var j JSONB
	val, err := j.Value()
	if err != nil {
		t.Fatalf("value nil map: %v", err)
	}
	b, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if string(b) != "{}" {
		t.Fatalf("expected {}, got %s", b)
	}
}

func TestDefaultNotificationPreferences(t *testing.T) {
	userID := uuid.New()
	prefs := DefaultNotificationPreferences(userID)
	if prefs.UserID != userID {
		t.Fatalf("unexpected user id")
	}
	if !prefs.PushEnabled || !prefs.EmailEnabled {
		t.Fatal("expected push and email enabled by default")
	}
	if prefs.MaxPushPerDay != defaultMaxPushPerDay {
		t.Fatalf("expected max push %d, got %d", defaultMaxPushPerDay, prefs.MaxPushPerDay)
	}
	if prefs.Preferences == nil {
		t.Fatal("expected non-nil preferences map")
	}
}

func TestNormalizeNotificationPreferences(t *testing.T) {
	prefs := &NotificationPreferences{}
	normalizeNotificationPreferences(prefs)
	if prefs.ID == uuid.Nil {
		t.Fatal("expected generated id")
	}
	if prefs.MaxPushPerDay != defaultMaxPushPerDay {
		t.Fatal("expected default max push")
	}
	if prefs.QuietHoursTZ != defaultQuietHoursTZ {
		t.Fatal("expected default timezone")
	}
}

func TestJSONBRoundTrip(t *testing.T) {
	original := JSONB{"channel": "email", "count": float64(2)}
	val, err := original.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}

	var decoded JSONB
	if err := decoded.Scan(val); err != nil {
		t.Fatalf("scan: %v", err)
	}

	b1, _ := json.Marshal(original)
	b2, _ := json.Marshal(decoded)
	if string(b1) != string(b2) {
		t.Fatalf("round trip mismatch: %s vs %s", b1, b2)
	}
}
