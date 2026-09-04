package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadBatchFlowNodes(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadBatchFlowNodes should not be called — flow nodes are only ever the root of a flow response and are never fetched by id",
	)
}
