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

type ReadAccessMed interface {
	// CheckReadAccess verifies that the actor account has owner-side read access
	// to the target account. Same-account access is always allowed. Cross-account
	// access requires an account_relation row in the actor→target direction
	// (the actor is the owner of the relation). Use this for endpoints that
	// expose the owner's view of a counterparty account (e.g. a merchant
	// reading data scoped to one of their customers).
	//
	//  1. Allow access when the actor and target accounts are the same.
	//  2. Require an actor→target account relation; otherwise return an authorization error.
	CheckReadAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError

	// CheckCounterpartyReadAccess verifies access in either direction. It is
	// intended for customer/supplier portal endpoints where the counterparty
	// (e.g. a customer) reads data on the owner's account (e.g. a vendor),
	// and the account_relation row is stored owner→counterparty.
	// Only use this on endpoints that explicitly scope returned data to the
	// counterparty — otherwise it leaks cross-tenant data.
	//
	//  1. Allow access when the actor and target accounts are the same.
	//  2. Check for an account relation in the actor→target direction.
	//  3. Fall back to checking the target→actor direction.
	//  4. Return an authorization error when no relation exists in either direction.
	CheckCounterpartyReadAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError
}

type EditAccessMed interface {
	// CheckEditAccess verifies that the actor account has edit access to the
	// target account. Same-account access is always allowed. Cross-account
	// access requires: the target has no active billing plan, a relation exists
	// between the accounts, and the target has no other owner relations.
	//
	//  1. Allow access when the actor and target accounts are the same.
	//  2. Reject when the target has an active billing plan.
	//  3. Require a relation between the actor and target accounts.
	//  4. Reject when the target has owner relations with other accounts.
	CheckEditAccess(ctx context.Context, actorAccountID, targetAccountID string) *apierror.APIError
}

type ProductionFlowMed interface {
	// LinkFlow recomputes all parent-child production step connections for a step
	// based on its current consumptions and productions.
	//
	//  1. Delegate to the production flow repository to rebuild the step's connections.
	LinkFlow(ctx context.Context, productionStepID, accountID string) *apierror.APIError

	// DisconnectSteps removes a specific parent-child connection between two steps.
	//
	//  1. Delegate to the production flow repository to remove the connection.
	DisconnectSteps(ctx context.Context, sourceID, targetID string) *apierror.APIError

	// FindSourceStepsByConsumption returns IDs of parent steps that should be disconnected
	// when a consumption is deleted.
	//
	//  1. Delegate to the production flow repository to find the matching parent step IDs.
	FindSourceStepsByConsumption(ctx context.Context, productionStepID, consumptionID, accountID string) ([]string, *apierror.APIError)

	// FindDownstreamStepByItem returns the ID of a downstream step connected via a specific
	// consumed item, if one exists.
	//
	//  1. Delegate to the production flow repository to find the connected downstream step.
	FindDownstreamStepByItem(ctx context.Context, productionStepID, itemID, accountID string) (*string, *apierror.APIError)
}

type BurnRateMed interface {
	// RecalculateFromHistory updates the item's burn_rate from consumption change logs
	// over the last 30 days. No-op when there is insufficient history.
	//
	//  1. Load the item and resolve its category's base unit.
	//  2. List the item's consumption change logs; no-op when fewer than two exist.
	//  3. Sum the absolute consumption quantities, converting each to the base unit.
	//  4. Divide the total by the days elapsed between the first and last log.
	//  5. Persist the resulting per-day rate to the item's burn rate.
	RecalculateFromHistory(ctx context.Context, accountID, itemID string) *apierror.APIError
}

type SandboxMed interface {
	// Create provisions a new sandbox account under the given owner account.
	//
	// 1. Verify the owner is a production account (not already a sandbox).
	// 2. Fetch the owner's plan code and sandbox limit.
	// 3. Check the current sandbox count against the plan limit; reject if at capacity.
	// 4. Generate unique IDs for the new account and sandbox type.
	// 5. Create the account record with sandbox type and the owner's plan code.
	// 6. Create supporting records: business address, account-user link, portal, system products, and branding.
	// 7. Insert the sandbox account record linking it to the owner.
	// 8. Re-fetch and return the created sandbox with populated owner metadata.
	Create(ctx context.Context, ownerAccountID, userID, name string) (*SandboxAccount, *apierror.APIError)

	// Delete removes a sandbox account and its underlying account record.
	//
	// 1. Find the sandbox by type ID and verify it belongs to the requesting owner account.
	// 2. Confirm the underlying account is actually a sandbox account.
	// 3. Delete the sandbox account record from the sandbox table.
	// 4. Delete the underlying account record.
	// 5. Return the deleted account ID for downstream purge processing.
	Delete(ctx context.Context, ownerAccountID, sandboxTypeID string) (accountID string, apiErr *apierror.APIError)
}
