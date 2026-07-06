package featureflags

import (
	"context"
	"hash/fnv"
)

// Service resolves the final on/off state for a given user, and lets admins
// flip flags at runtime with no deploy needed.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ResolveForUser is what /me/features calls. It combines:
//  1. Global flag enabled/disabled
//  2. Rollout percentage (consistent per-user via hash, not random each request)
//  3. Per-user override (always wins, for beta testers / support cases)
func (s *Service) ResolveForUser(ctx context.Context, userID string) (ResolvedFlags, error) {
	flags, err := s.repo.GetAllFlags(ctx)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.GetUserOverrides(ctx, userID)
	if err != nil {
		return nil, err
	}

	overrideMap := make(map[FlagKey]bool, len(overrides))
	for _, o := range overrides {
		overrideMap[o.Key] = o.Enabled
	}

	resolved := make(ResolvedFlags, len(flags))
	for _, f := range flags {
		if val, ok := overrideMap[f.Key]; ok {
			resolved[f.Key] = val
			continue
		}
		if !f.Enabled {
			resolved[f.Key] = false
			continue
		}
		resolved[f.Key] = inRollout(userID, f.Key, f.RolloutPercentage)
	}
	return resolved, nil
}

// SetFlag is called from the admin panel toggle switch.
func (s *Service) SetFlag(ctx context.Context, key FlagKey, enabled bool, rollout int, adminID string) error {
	return s.repo.SetFlag(ctx, key, enabled, rollout, adminID)
}

// ListWithStats powers the admin dashboard's flag table.
func (s *Service) ListWithStats(ctx context.Context) ([]FlagWithStats, error) {
	return s.repo.GetAllFlagsWithStats(ctx)
}

// inRollout deterministically buckets a user into 0-99 based on a hash of
// their ID + the flag key, so the same user always gets the same result
// (no flickering features on/off between requests) until rollout % changes.
func inRollout(userID string, key FlagKey, percentage int) bool {
	if percentage >= 100 {
		return true
	}
	if percentage <= 0 {
		return false
	}
	h := fnv.New32a()
	h.Write([]byte(userID + ":" + string(key)))
	bucket := int(h.Sum32() % 100)
	return bucket < percentage
}
