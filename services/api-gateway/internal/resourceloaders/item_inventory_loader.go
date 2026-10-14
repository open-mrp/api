package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
)

func LoadItemInventories(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadItemInventories should not be called — item inventories are not used as expandable sub-resources",
	)
}
