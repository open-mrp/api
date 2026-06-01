package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadAuditEvents(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAuditEvents should not be called — audit events are not used as expandable sub-resources",
	)
}
