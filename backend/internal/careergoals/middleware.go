package careergoals

import (
	"net/http"
	"strconv"

	apierrors "studyapp/backend/internal/common/errors"
	custommw "studyapp/backend/internal/common/middleware"
	"studyapp/backend/internal/featureflags"
)

// RequireCareerGoalsFlag gates every route in this module behind the
// career_goals_module feature flag, resolved per authenticated user.
func RequireCareerGoalsFlag(flagService *featureflags.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := contextUserID(r)
			if userID == "" {
				apierrors.WriteUnauthorized(w)
				return
			}

			flags, err := flagService.ResolveForUser(r.Context(), userID)
			if err != nil {
				apierrors.WriteInternal(w, nil, err, custommw.GetTraceID(r.Context()))
				return
			}

			if !flags[featureflags.FlagCareerGoals] {
				apierrors.WriteError(w, http.StatusForbidden, apierrors.ErrCodeFeatureGated, "feature not available", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func contextUserID(r *http.Request) string {
	switch v := r.Context().Value("user_id").(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		return v
	default:
		return ""
	}
}
