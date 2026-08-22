package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadTransactions(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadTransactions should not be called — transactions are not used as expandable sub-resources",
	)
}
