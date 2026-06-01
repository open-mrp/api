package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

func LoadAgentDefinitions(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAgentDefinitions should not be called — agent definitions are not used as expandable sub-resources",
	)
}
