package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadProductionSteps(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadProductionSteps should not be called — production steps are not used as expandable sub-resources",
	)
}

func LoadProductions(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadProductions should not be called — productions are not used as expandable sub-resources",
	)
}
