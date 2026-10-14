package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
)

func LoadDeliveryLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadDeliveryLines should not be called — delivery lines arrive with their delivery and are never fetched by id",
	)
}
