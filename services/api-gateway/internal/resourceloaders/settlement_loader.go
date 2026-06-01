package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadSettlements(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadSettlements should not be called — settlements are not used as expandable sub-resources",
	)
}
