package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	apierror "github.com/open-mrp/api/shared/errors"
)

// omitOnUnauthorized reports whether an expandable sub-resource load failed purely because the caller isn't authorized to view that resource class — e.g. a customer-portal actor resolving a sales order's internal-only related.shipments/pick/production_run, whose backing RPCs require an internal actor. Includes are best-effort expansions: one the caller can't see is simply absent from the response, matching how created_by and customer resolve (they follow the resource's own visibility rather than 403ing the parent). Loaders use this to omit such a sub-resource instead of failing the whole parent retrieve. Any other error (not-found, invariant, transport) still propagates.
func omitOnUnauthorized(apiErr *apierror.APIError) bool {
	return apiErr != nil && apiErr.Code == apierror.ErrorCodeInsufficientPerms
}

// loadReadable runs load as the caller. When the caller may not read that resource class it loads nothing and reports readable false, so the embedding record shows null instead of failing.
func loadReadable(ctx context.Context, load resourcekit.Loader, ids []string) (loaded map[string]any, readable bool, apiErr *apierror.APIError) {
	loaded, apiErr = load(ctx, ids)
	if omitOnUnauthorized(apiErr) {
		return nil, false, nil
	}
	if apiErr != nil {
		return nil, false, apiErr
	}
	return loaded, true, nil
}
