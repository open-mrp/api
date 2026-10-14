package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
)

func LoadProductionFlows(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadProductionFlows should not be called — production flows are not used as expandable sub-resources",
	)
}
