package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
)

func LoadAddressSuggestions(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAddressSuggestions should not be called — address suggestions are not used as expandable sub-resources",
	)
}
