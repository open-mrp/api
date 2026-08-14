package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

// LoadCustomerPricingFindings exists only so the finding can register its expandable sub-resources. A finding is computed per request and has no row to fetch, so it is never loaded as anyone's child.
func LoadCustomerPricingFindings(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadCustomerPricingFindings should not be called — pricing findings are computed, not stored",
	)
}

// LoadRealizedMarginFindings is the same stub for realized-margin findings.
func LoadRealizedMarginFindings(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadRealizedMarginFindings should not be called — margin findings are computed, not stored",
	)
}
