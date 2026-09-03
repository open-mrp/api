package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadPurchaseOrderLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPurchaseOrderLines should not be called — purchase order lines arrive with their order and are never fetched by id",
	)
}
