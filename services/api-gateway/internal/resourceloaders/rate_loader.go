package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadRates(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadRates should not be called — rates are not used as expandable sub-resources",
	)
}
