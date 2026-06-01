package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadVolumeDiscounts(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadVolumeDiscounts should not be called — volume discounts are not used as expandable sub-resources",
	)
}
