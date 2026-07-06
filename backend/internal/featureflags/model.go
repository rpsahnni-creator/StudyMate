package featureflags

import "time"

// FlagKey is a strongly-typed key so modules can't be toggled by a typo'd string.
// Add new module keys here as the product grows (e.g. FlagAdaptiveEngine).
type FlagKey string

const (
	FlagScanQuiz    FlagKey = "scan_quiz_module"    // core module, always on in practice
	FlagCareerGoals FlagKey = "career_goals_module" // Phase 3 module, off by default
)

// Flag is the global on/off switch for a module, optionally rolled out gradually.
type Flag struct {
	Key               FlagKey   `json:"key" db:"flag_key"`
	Enabled           bool      `json:"enabled" db:"enabled"`
	RolloutPercentage int       `json:"rollout_percentage" db:"rollout_percentage"` // 0-100
	UpdatedBy         string    `json:"updated_by" db:"updated_by"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// UserOverride lets a specific user be opted in/out regardless of the global flag.
// Used for beta testers or support cases ("turn career goals on just for this user").
type UserOverride struct {
	UserID  string  `json:"user_id" db:"user_id"`
	Key     FlagKey `json:"key" db:"flag_key"`
	Enabled bool    `json:"enabled" db:"enabled"`
}

// ResolvedFlags is what gets sent to clients (mobile/web) after resolving
// global flag + rollout percentage + per-user override into simple booleans.
type ResolvedFlags map[FlagKey]bool

// FlagWithStats is what the admin dashboard renders — the flag's config
// plus enough usage data to make an informed toggle decision (this is the
// "soft-launch, watch server load/AI cost, then scale up" loop from the
// roadmap made concrete).
type FlagWithStats struct {
	Flag
	OverrideCount      int     `json:"override_count"`       // beta testers with manual overrides
	AICallCount24h     int     `json:"ai_call_count_24h"`     // AI generations in last 24h attributed to this module
	AICacheHitRate24h  float64 `json:"ai_cache_hit_rate_24h"` // 0.0-1.0, higher = cheaper
	AIEstimatedCost24h float64 `json:"ai_estimated_cost_24h_usd"`
}
