package mediator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	tracing "github.com/augno/api/shared/tracing"
)

var idempotencyMedTracer = tracing.GetTracer("auth-service.login_idempotency_mediator")

type idempotencyMedImpl struct {
	repos domain.RepoFactory
}

type IdempotencyMedConfig struct {
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

func (m *idempotencyMedImpl) UpsertIdempotencyKey(ctx context.Context, identity *domain.RequestIdentity) (*domain.IdempotencyKey, *apierror.APIError) {
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
		actorID = &identity.ActorID
		identityType = string(identity.IdentityType)
		targetAccountID = identity.TargetAccountID
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
		RecoveryPoint:  domain.RecoveryPointStarted,
	})

	if apiErr != nil {
		return nil, apiErr
	}

	return newKey, nil
}

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

func (m *idempotencyMedImpl) CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError {
	responseBody, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal response for caching.")
	}
	return m.repos.NewIdempotencyKeyRepo().SetResponse(ctx, typeID, 200, responseBody, domain.RecoveryPointFinished)
}
