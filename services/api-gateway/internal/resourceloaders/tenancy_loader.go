package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadTenancies(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadTenancies should not be called — tenancies are not used as expandable sub-resources",
	)
}
