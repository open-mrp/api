package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadEmailLogs(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadEmailLogs should not be called — email logs are not used as expandable sub-resources",
	)
}
