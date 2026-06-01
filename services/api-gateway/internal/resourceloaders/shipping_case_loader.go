package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadShippingCases(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadShippingCases should not be called — shipping cases are not used as expandable sub-resources",
	)
}
