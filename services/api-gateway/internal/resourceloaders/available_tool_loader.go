package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadAvailableTools(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAvailableTools should not be called — available tools are not used as expandable sub-resources",
	)
}

func LoadToolGroups(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadToolGroups should not be called — tool groups are not used as expandable sub-resources",
	)
}
