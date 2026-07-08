package resourceloaders

import apierror "github.com/augno/api/shared/errors"

// omitOnUnauthorized reports whether an expandable sub-resource load failed purely because the caller isn't authorized to view that resource class — e.g. a customer-portal actor resolving a sales order's internal-only related.shipments/pick/production_run, whose backing RPCs require an internal actor. Includes are best-effort expansions: one the caller can't see is simply absent from the response, matching how created_by and customer resolve (they follow the resource's own visibility rather than 403ing the parent). Loaders use this to omit such a sub-resource instead of failing the whole parent retrieve. Any other error (not-found, invariant, transport) still propagates.
func omitOnUnauthorized(apiErr *apierror.APIError) bool {
	return apiErr != nil && apiErr.Code == apierror.ErrorCodeInsufficientPerms
}
