package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadSettlementSummaries(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadSettlementSummaries should not be called — settlement summaries are not used as expandable sub-resources",
	)
}
