package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadSupplierMaterials(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadSupplierMaterials should not be called — supplier materials are not used as expandable sub-resources",
	)
}
