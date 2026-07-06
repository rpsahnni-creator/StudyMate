package billing

import (
	"context"
	"encoding/json"
)

// ListPlans returns all active subscription plans for the public catalog.
func (s *Service) ListPlans(ctx context.Context) ([]map[string]any, error) {
	plans, err := s.repo.ListActivePlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		out = append(out, planResponse(p))
	}
	return out, nil
}

// GetPlansHandlerData is a convenience wrapper for the HTTP handler.
func (s *Service) GetPlansHandlerData(ctx context.Context) (json.RawMessage, error) {
	plans, err := s.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(plans)
}
