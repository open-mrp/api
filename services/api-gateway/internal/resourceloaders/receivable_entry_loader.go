package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadReceivableEntries(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadReceivableEntries should not be called — receivable entries are not used as expandable sub-resources",
	)
}
