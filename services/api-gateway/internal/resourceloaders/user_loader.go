package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadUsers(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadUsers should not be called — users are not used as expandable sub-resources",
	)
}
