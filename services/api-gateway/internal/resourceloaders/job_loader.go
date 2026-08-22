package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

// LoadJobs exists to satisfy the registry's Load hook. Nothing embeds a job as a
// sub-resource — a job is only ever the root of its own response — so the resolver never
// batches a job by id, and there is no BatchGetJobsByIDs to call if it did.
func LoadJobs(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadJobs should not be called — a job is only ever the root of its own response, never a sub-resource",
	)
}
