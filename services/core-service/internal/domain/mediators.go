package domain

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

type RequestIdentity struct {
	ActorID      string
	IdentityType types.IdentityType
}
type IdempotencyMed interface {
	// UpsertIdempotencyKey returns the existing idempotency key for the request scope,
	// or creates one if it does not exist.
	//
	// Side effects:
	//   - May persist a new idempotency key for the computed request scope.
	UpsertIdempotencyKey(ctx context.Context, identity *RequestIdentity) (*IdempotencyKey, *apierror.APIError)

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

type SandboxMed interface {
	// Create creates a new sandbox account for the owner and grants the
	// specified user admin access.
	//
	// Behavior:
	//   - Fails if the owner has reached the maximum number of sandbox accounts.
	//
	// Side effects:
	//   - Persists a new account, business address, portal, account-user link,
	//     and sandbox_account record.
	Create(ctx context.Context, ownerAccountID, userID, name string) (*SandboxAccount, *apierror.APIError)

	// Delete removes a sandbox account and its underlying account record.
	//
	// Behavior:
	//   - Fails if the owner has only one sandbox remaining.
	//
	// Returns:
	//   - The account ID of the deleted sandbox (needed for async purge).
	Delete(ctx context.Context, ownerAccountID, sandboxTypeID string) (accountID string, apiErr *apierror.APIError)
}
