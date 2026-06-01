package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadInventoryItems(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadInventoryItems should not be called — inventory items are not used as expandable sub-resources",
	)
}
