package billing

import (
	"context"
)

// GetUserEntitlements returns the current subscription entitlements for a user.
func GetUserEntitlements(ctx context.Context, repo Repository, userID int64) (*Entitlements, error) {
	ent, err := repo.GetEntitlements(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetMySubscription is a service wrapper for the authenticated subscription endpoint.
func (s *Service) GetMySubscription(ctx context.Context, userID int64) (*Entitlements, error) {
	return GetUserEntitlements(ctx, s.repo, userID)
}
