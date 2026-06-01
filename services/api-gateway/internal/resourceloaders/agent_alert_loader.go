package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadAgentAlerts(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAgentAlerts should not be called — agent alerts are not used as expandable sub-resources",
	)
}
