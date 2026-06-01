package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadShipments(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadShipments should not be called — shipments are not used as expandable sub-resources",
	)
}
