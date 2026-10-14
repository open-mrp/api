package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
)

func LoadTransactionAllocations(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadTransactionAllocations should not be called — transaction allocations are not used as expandable sub-resources",
	)
}
