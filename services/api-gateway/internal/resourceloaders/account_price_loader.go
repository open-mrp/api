package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
)

func LoadAccountPrices(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAccountPrices should not be called — account prices are not used as expandable sub-resources",
	)
}
