package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadCatalogProductLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadCatalogProductLines should not be called — catalog product lines are not used as expandable sub-resources",
	)
}
