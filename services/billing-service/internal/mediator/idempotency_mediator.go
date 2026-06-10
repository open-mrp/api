package mediator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var idempotencyMedTracer = tracing.GetTracer("billing-service.idempotency_mediator")

type idempotencyMedImpl struct {
	repos domain.RepoFactory
}

type IdempotencyMedConfig struct {
	// Repos (required) is the repository factory for idempotency persistence.
	Repos domain.RepoFactory
}

func (c *IdempotencyMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("idempotency mediator: repos is required")
	}
	return nil
}

func NewIdempotencyMed(config *IdempotencyMedConfig) domain.IdempotencyMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &idempotencyMedImpl{
		repos: config.Repos,
	}
}

// UpsertIdempotencyKey returns the existing idempotency key for the request scope,
// or creates one if it does not exist.
//
//  1. Resolve the idempotency key from the request context, falling back to the request ID.
//  2. Compute the scope hash from the actor, target account, service, handler, and key.
//  3. Return the existing key for the scope hash when one exists.
//  4. Otherwise persist a new key at the Started recovery point, re-fetching the
//     existing row if a concurrent request inserted the same scope hash first.
func (m *idempotencyMedImpl) UpsertIdempotencyKey(ctx context.Context, identity *types.Identity) (*domain.IdempotencyKey, *apierror.APIError) {
	ctx, span := idempotencyMedTracer.Start(ctx, "mediator.idempotency.upsert_idempotency_key")
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

	var actorID *string = nil
	var targetAccountID *string = nil
	var identityType = string(types.IdentityActorTypeUnauthenticated)
	if identity != nil {
		actorID = &identity.Actor.ID
		identityType = string(identity.Type)
		if identity.Target != nil {
			targetAccountID = &identity.Target.AccountID
		}
	}

	scopeHash := idempotency.ComputeServiceScopeHash(actorID, targetAccountID, domain.ServiceName, handler, idempotencyKey)

	repo := m.repos.NewIdempotencyKeyRepo()
	existingKey, apiErr := repo.GetByScopeHash(ctx, scopeHash)
	if apiErr != nil && !apierror.IsNotFound(apiErr) {
		return nil, apiErr
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
		// A concurrent request may have inserted the same scope_hash first.
		// Re-fetch the existing row instead of propagating the duplicate error.
		if apiErr.Code == apierror.ErrorCodeResourceExists {
			existingKey, retryErr := repo.GetByScopeHash(ctx, scopeHash)
			if retryErr != nil {
				return nil, retryErr
			}
			return existingKey, nil
		}
		return nil, apiErr
	}

	return newKey, nil
}

// CacheErrorResponse caches a non-transient error response for the given idempotency key
// and returns the original error.
//
//  1. Return transient errors uncached so the client can retry.
//  2. Persist non-transient errors as the cached response and mark the key finished.
func (m *idempotencyMedImpl) CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError {
	if apiErr == nil {
		return nil
	}
	if !apiErr.IsTransient {
		errorBody, _ := apiErr.ToJSON()
		_ = m.repos.NewIdempotencyKeyRepo().SetResponse(ctx, typeID, apierror.GetHTTPStatusCode(apiErr.Code), errorBody, domain.RecoveryPointFinished)
	}
	return apiErr
}

// CacheSuccessResponse caches a successful response for the given idempotency key.
//
//  1. Marshal the response data to JSON.
//  2. Persist it as the cached response and mark the key finished.
func (m *idempotencyMedImpl) CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError {
	responseBody, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal response for caching.")
	}
	return m.repos.NewIdempotencyKeyRepo().SetResponse(ctx, typeID, 200, responseBody, domain.RecoveryPointFinished)
}
