package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadDeliveries(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadDeliveries should not be called — deliveries are not used as expandable sub-resources",
	)
}
