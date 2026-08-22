package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadConsumptions(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadConsumptions should not be called — consumptions are not used as expandable sub-resources",
	)
}
