package domain

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

type IdempotencyMed interface {
	// UpsertIdempotencyKey returns the existing idempotency key for the request scope,
	// or creates one if it does not exist.
	//
	// Side effects:
	//   - May persist a new idempotency key for the computed request scope.
	UpsertIdempotencyKey(ctx context.Context, identity *types.Identity) (*IdempotencyKey, *apierror.APIError)

	// CacheErrorResponse caches a non-transient error response for the given idempotency key
	// and returns the original error.
	//
	// Behavior:
	//   - Transient errors are not cached.
	//
	// Side effects:
	//   - Persists the error response for subsequent replays of the same idempotency key.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError

	// CacheSuccessResponse caches a successful response for the given idempotency key.
	//
	// Side effects:
	//   - Persists the response for subsequent replays of the same idempotency key.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}
