package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadQuantities(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadQuantities should not be called — quantities are not used as expandable sub-resources",
	)
}
