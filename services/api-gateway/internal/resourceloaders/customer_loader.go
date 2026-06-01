package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadCustomers(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadCustomers should not be called — customers are not used as expandable sub-resources",
	)
}
