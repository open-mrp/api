package domain

import (
	"context"

	types "github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// IdempotencyMed provides idempotency logic for service operations.
type IdempotencyMed interface {
	// UpsertIdempotencyKey upserts and returns the idempotency key for the request scope.
	UpsertIdempotencyKey(ctx context.Context, identity *types.Identity) (*IdempotencyKey, *apierror.APIError)

	// CacheErrorResponse caches a non-transient error response for the idempotency key.
	//
	// Behavior:
	//   - If the error is transient, it is returned without caching.
	//
	// Side effects:
	//   - Persists the error response for subsequent replays of the same idempotency key.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError

	// CacheSuccessResponse caches a successful response for the idempotency key.
	//
	// Side effects:
	//   - Persists the success response for subsequent replays of the same idempotency key.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}

// Mediators groups all mediator instances.
type Mediators struct {
	Idempotency IdempotencyMed
}

// MediatorFactory builds mediator instances from a RepoFactory.
type MediatorFactory interface {
	Build(repoFactory RepoFactory) Mediators
}
