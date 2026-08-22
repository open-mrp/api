package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

// LoadContactMatches satisfies the resourcekit Definition's required loader for the contact_match resource. Contact matches are only ever produced as the top-level result of the find-by-email endpoint — they are never referenced by ID as a sub-resource — so this loader is never invoked and returns an empty result.
func LoadContactMatches(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}
