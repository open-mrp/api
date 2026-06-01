package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadInvoices(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadInvoices should not be called — invoices are not used as expandable sub-resources",
	)
}
