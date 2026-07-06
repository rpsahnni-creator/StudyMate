package admin

import "time"

// AdminUser is a user row enriched with subscription + usage for the admin table.
type AdminUser struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	IsSuspended    bool       `json:"is_suspended"`
	Plan           string     `json:"plan"`
	SubStatus      string     `json:"subscription_status,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	ScanCountToday int        `json:"scan_count_today"`
	CreatedAt      time.Time  `json:"created_at"`
}

// UsersPage is the paginated response for the users list.
type UsersPage struct {
	Items []AdminUser `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// AdminJob is a scan job enriched with the owner's email for monitoring.
type AdminJob struct {
	ID           int64      `json:"id"`
	UserEmail    string     `json:"user_email"`
	Status       string     `json:"status"`
	Progress     int        `json:"progress"`
	Provider     string     `json:"provider,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DurationMs   int64      `json:"duration_ms"`
}

type JobsPage struct {
	Items []AdminJob `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

// AICostRow is a daily provider/model cost aggregation.
type AICostRow struct {
	Day          string  `json:"day"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	RequestCount int     `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost_estimate"`
}

type AICostSummary struct {
	TotalCost     float64     `json:"total_cost_estimate"`
	TotalRequests int         `json:"total_requests"`
	TotalTokens   int64       `json:"total_tokens"`
	AvgCost       float64     `json:"avg_cost_per_request"`
	Rows          []AICostRow `json:"rows"`
	From          string      `json:"from"`
	To            string      `json:"to"`
}

// AuditLogEntry mirrors an audit_logs row with actor email.
type AuditLogEntry struct {
	ID         int64     `json:"id"`
	ActorEmail string    `json:"actor_email,omitempty"`
	ActorID    *int64    `json:"actor_user_id,omitempty"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuditLogPage struct {
	Items []AuditLogEntry `json:"items"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

// ContentFlag mirrors a content_flags row for the moderation queue.
type ContentFlag struct {
	ID              int64      `json:"id"`
	QuestionID      int64      `json:"question_id"`
	ContentHash     string     `json:"content_hash,omitempty"`
	Reason          string     `json:"reason"`
	ReportedByEmail string     `json:"reported_by_email,omitempty"`
	Status          string     `json:"status"`
	ResolutionReason string    `json:"resolution_reason,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ContentFlagsPage struct {
	Items []ContentFlag `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}
