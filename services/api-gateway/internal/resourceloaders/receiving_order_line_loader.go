package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadReceivingOrderLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadReceivingOrderLines should not be called — receiving order lines are not used as expandable sub-resources",
	)
}
