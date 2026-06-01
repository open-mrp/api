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

type ReadAccessMed interface {
	// CheckReadAccess verifies that the actor account has owner-side read access
	// to the target account. Same-account access is always allowed. Cross-account
	// access requires an account_relation row in the actor→target direction
	// (the actor is the owner of the relation). Use this for endpoints that
	// expose the owner's view of a counterparty account (e.g. a merchant
	// reading data scoped to one of their customers).
	CheckReadAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError

	// CheckCounterpartyReadAccess verifies access in either direction. It is
	// intended for customer/supplier portal endpoints where the counterparty
	// (e.g. a customer) reads data on the owner's account (e.g. a vendor),
	// and the account_relation row is stored owner→counterparty.
	// Only use this on endpoints that explicitly scope returned data to the
	// counterparty — otherwise it leaks cross-tenant data.
	CheckCounterpartyReadAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError
}

type EditAccessMed interface {
	// CheckEditAccess verifies that the actor account has edit access to the
	// target account. Same-account access is always allowed. Cross-account
	// access requires: the target has no active billing plan, a relation exists
	// between the accounts, and the target has no other owner relations.
	CheckEditAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError
}

type ProductionFlowMed interface {
	// LinkFlow recomputes all parent-child production step connections for a step
	// based on its current consumptions and productions.
	LinkFlow(ctx context.Context, productionStepID, accountID string) *apierror.APIError

	// DisconnectSteps removes a specific parent-child connection between two steps.
	DisconnectSteps(ctx context.Context, sourceID, targetID string) *apierror.APIError

	// FindSourceStepsByConsumption returns IDs of parent steps that should be disconnected
	// when a consumption is deleted.
	FindSourceStepsByConsumption(ctx context.Context, productionStepID, consumptionID, accountID string) ([]string, *apierror.APIError)

	// FindDownstreamStepByItem returns the ID of a downstream step connected via a specific
	// consumed item, if one exists.
	FindDownstreamStepByItem(ctx context.Context, productionStepID, itemID, accountID string) (*string, *apierror.APIError)
}

type BurnRateMed interface {
	// RecalculateFromHistory updates the item's burn_rate from consumption change logs
	// over the last 30 days. No-op when there is insufficient history.
	RecalculateFromHistory(ctx context.Context, accountID, itemID string) *apierror.APIError
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
