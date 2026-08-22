package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadOpenCreditEntries(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadOpenCreditEntries should not be called — open credit entries are not used as expandable sub-resources",
	)
}
