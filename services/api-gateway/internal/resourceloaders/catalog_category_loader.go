package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadCatalogCategories(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadCatalogCategories should not be called — catalog categories are not used as expandable sub-resources",
	)
}
