package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadInventoryChangeLogs(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadInventoryChangeLogs should not be called — inventory change logs are not used as expandable sub-resources",
	)
}
