package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEvent captures a security-relevant action for the audit log.
type AuditEvent struct {
	UserID    string
	Action    string
	Resource  string
	IPAddress string
	UserAgent string
	Success   bool
	Details   map[string]any
}

// LogAuditEvent writes an audit record. Details must not contain sensitive data.
func LogAuditEvent(ctx context.Context, db *pgxpool.Pool, event AuditEvent) error {
	if db == nil {
		return fmt.Errorf("audit db is nil")
	}

	entityType, entityID := splitResource(event.Resource)
	actorID := parseActorUserID(event.UserID)

	detailsJSON, err := json.Marshal(sanitizeAuditDetails(event.Details))
	if err != nil {
		detailsJSON = []byte("{}")
	}

	_, err = db.Exec(ctx, `
		INSERT INTO audit_logs (
			actor_user_id, action, entity_type, entity_id,
			ip_address, user_agent, success, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
	`, actorID, event.Action, entityType, entityID,
		nullIfEmpty(event.IPAddress), nullIfEmpty(event.UserAgent), event.Success, detailsJSON)
	return err
}

func splitResource(resource string) (entityType, entityID string) {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return "system", ""
	}
	parts := strings.SplitN(resource, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func parseActorUserID(userID string) *int64 {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := strings.TrimSpace(s)
	return &v
}

func sanitizeAuditDetails(details map[string]any) map[string]any {
	if len(details) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(details))
	for k, v := range details {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
			continue
		}
		out[k] = v
	}
	return out
}

// RequestMeta extracts IP and user agent from an HTTP request.
func RequestMeta(ip, userAgent string) (string, string) {
	return strings.TrimSpace(ip), strings.TrimSpace(userAgent)
}
