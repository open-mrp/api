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
	//  1. Resolve the idempotency key from the request context, falling back to the request ID.
	//  2. Compute the scope hash from the actor, target account, service, handler, and key.
	//  3. Return the existing key for the scope hash when one exists.
	//  4. Otherwise persist a new key at the Started recovery point, re-fetching the
	//     existing row if a concurrent request inserted the same scope hash first.
	UpsertIdempotencyKey(ctx context.Context, identity *types.Identity) (*IdempotencyKey, *apierror.APIError)

	// CacheErrorResponse caches a non-transient error response for the given idempotency key
	// and returns the original error.
	//
	//  1. Return transient errors uncached so the client can retry.
	//  2. Persist non-transient errors as the cached response and mark the key finished.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError

	// CacheSuccessResponse caches a successful response for the given idempotency key.
	//
	//  1. Marshal the response data to JSON.
	//  2. Persist it as the cached response and mark the key finished.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}
