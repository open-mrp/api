package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadReceivingOrders(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadReceivingOrders should not be called — receiving orders are not used as expandable sub-resources",
	)
}
