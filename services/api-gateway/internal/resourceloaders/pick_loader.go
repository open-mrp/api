package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadPicks(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPicks should not be called — picks are not used as expandable sub-resources",
	)
}

func LoadPickLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPickLines should not be called — pick lines are not used as expandable sub-resources",
	)
}
