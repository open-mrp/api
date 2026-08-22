package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadValidatedAddresses(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadValidatedAddresses should not be called — validated addresses are not used as expandable sub-resources",
	)
}
