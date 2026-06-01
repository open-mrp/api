package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadSalesOrders(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadSalesOrders should not be called — sales orders are not used as expandable sub-resources",
	)
}
