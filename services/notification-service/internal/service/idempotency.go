package service

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

// Recovery-point idempotency for the chat service's pure-create RPCs (which mint a fresh id per call
// and have no natural dedup key). This mirrors core-service's idempotency mediator; notification-service
// has no mediator layer, so the flow lives as helpers over the repo factory that a create method drives:
//
//	key, apiErr := upsertIdempotencyKey(ctx, s.repoFactory, identity)
//	switch domain.RecoveryPoint(key.RecoveryPoint) {
//	case domain.RecoveryPointFinished: return the cached response
//	case domain.RecoveryPointStarted:  run the mutation in WithTx and cacheSuccessResponse inside it
//	}
//
// The client's Idempotency-Key (forwarded by the gateway into gRPC metadata and lifted into context by
// the default IdempotencyKey interceptor) scopes the key; it falls back to the request id when absent.
var idempotencyTracer = tracing.GetTracer("notification-service.idempotency")

// upsertIdempotencyKey returns the existing idempotency key for the request scope, or creates one at the Started recovery point.
//
//  1. Resolve the idempotency key from context, falling back to the request id.
//  2. Compute the scope hash from actor, target account, service, handler, and key.
//  3. Return the existing key for the scope hash when one exists.
//  4. Otherwise persist a new key at Started, re-fetching if a concurrent request inserted the same scope hash first.
func upsertIdempotencyKey(ctx context.Context, repos domain.RepoFactory, identity *types.Identity) (*domain.IdempotencyKey, *apierror.APIError) {
	ctx, span := idempotencyTracer.Start(ctx, "service.idempotency.upsert")
	defer span.End()

	idempotencyKey, hasKey := appctx.GetIdempotencyKey(ctx)
	if !hasKey {
		if requestID, ok := appctx.GetRequestID(ctx); ok {
			idempotencyKey = requestID
		} else {
			return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Idempotency key required in context."))
		}
	}

	handler, ok := appctx.GetHandler(ctx)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Handler required in context."))
	}

	typeID, genErr := id.GenID(id.ServiceIdempotencyKeyIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	var actorID *string
	var targetAccountID *string
	identityType := string(types.IdentityActorTypeUnauthenticated)
	if identity != nil {
		actorID = &identity.Actor.ID
		identityType = string(identity.Type)
		if identity.Target != nil {
			targetAccountID = &identity.Target.AccountID
		}
	}

	scopeHash := idempotency.ComputeServiceScopeHash(actorID, targetAccountID, domain.ServiceName, handler, idempotencyKey)

	repo := repos.NewIdempotencyKeyRepo()
	existingKey, apiErr := repo.GetByScopeHash(ctx, scopeHash)
	if apiErr != nil && !apierror.IsNotFound(apiErr) {
		return nil, tracing.Trace(span, apiErr)
	}
	if existingKey != nil {
		return existingKey, nil
	}

	newKey, apiErr := repo.Create(ctx, &domain.IdempotencyKey{
		TypeID:         typeID,
		ServiceName:    domain.ServiceName,
		Handler:        handler,
		IdempotencyKey: idempotencyKey,
		ActorID:        actorID,
		IdentityType:   identityType,
		ScopeHash:      scopeHash,
		RecoveryPoint:  string(domain.RecoveryPointStarted),
	})
	if apiErr != nil {
		// A concurrent request may have inserted the same scope_hash first. Re-fetch instead of failing.
		if apiErr.Code == apierror.ErrorCodeResourceExists {
			existingKey, retryErr := repo.GetByScopeHash(ctx, scopeHash)
			if retryErr != nil {
				return nil, tracing.Trace(span, retryErr)
			}
			return existingKey, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return newKey, nil
}

// cacheSuccessResponse persists a successful response for the key and marks it finished. Call it inside the mutation's WithTx (with the transaction-scoped factory) so caching commits atomically with the business state.
func cacheSuccessResponse(ctx context.Context, repos domain.RepoFactory, typeID string, data any) *apierror.APIError {
	responseBody, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal response for caching.")
	}
	return repos.NewIdempotencyKeyRepo().SetResponse(ctx, typeID, 200, responseBody, domain.RecoveryPointFinished)
}

// cacheErrorResponse persists a non-transient error response for the key and returns the original error. Transient errors are returned uncached so the client can retry.
func cacheErrorResponse(ctx context.Context, repos domain.RepoFactory, typeID string, apiErr *apierror.APIError) *apierror.APIError {
	if apiErr == nil {
		return nil
	}
	if !apiErr.IsTransient {
		errorBody, _ := apiErr.ToJSON()
		_ = repos.NewIdempotencyKeyRepo().SetResponse(ctx, typeID, apierror.GetHTTPStatusCode(apiErr.Code), errorBody, domain.RecoveryPointFinished)
	}
	return apiErr
}
