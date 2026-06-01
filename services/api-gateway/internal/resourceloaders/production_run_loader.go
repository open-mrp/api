package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadProductionRuns(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadProductionRuns should not be called — production runs are not used as expandable sub-resources",
	)
}
