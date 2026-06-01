package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadPurchaseOrders(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPurchaseOrders should not be called — purchase orders are not used as expandable sub-resources",
	)
}
