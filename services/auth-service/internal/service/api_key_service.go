package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var apiKeySvcTracer = tracing.GetTracer("auth-service.api_key_service")

type apiKeySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type APIKeySvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *APIKeySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("api key service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("api key service: mediator factory is required")
	}
	return nil
}

func NewAPIKeySvc(config *APIKeySvcConfig) domain.APIKeySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &apiKeySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (c *APIKeySvcConfig) WithDefaults(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient, encryptionKey []byte) *APIKeySvcConfig {
	if c == nil {
		c = &APIKeySvcConfig{}
	}

	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:              jwtSecret,
		APIKeyPepper:           pepper,
		NotificationPublisher:  notificationPublisher,
		FrontendURL:            frontendURL,
		CoreClient:             coreClient,
		DocAPIKeyEncryptionKey: encryptionKey,
	})

	return &APIKeySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
	}
}

func (s *apiKeySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *apiKeySvcImpl) withTx(ctx context.Context, fn func(context.Context, *apiKeySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &apiKeySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *apiKeySvcImpl) GetAPIKey(ctx context.Context, apiKeyID string) (*apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkApiKeyAccess(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ownerAccountID := *identity.TargetAccountID

	key, apiErr := s.repos.NewAPIKeyRepo().FindByTypeID(ctx, apiKeyID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if key.OwnerAccountID != ownerAccountID {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("API key not found."))
	}

	return key, nil
}

func (s *apiKeySvcImpl) CreateAPIKey(ctx context.Context, input domain.CreateAPIKeyInput) (*domain.CreateAPIKeyResult, *apierror.APIError) {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkApiKeyAccess(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ownerAccountID := *identity.TargetAccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CreateAPIKeyResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.CreateAPIKeyResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *apiKeySvcImpl) *apierror.APIError {
			txMeds := svc.mediators()

			secret, apiKey, createErr := txMeds.APIKey.Create(txCtx, domain.APIKeyCreateInput{
				AccountMode:    identity.AccountMode,
				OwnerAccountID: ownerAccountID,
				RoleID:         input.RoleID,
				Name:           input.Name,
				ExpiresAt:      input.ExpiresAt,
			})

			if createErr != nil {
				return createErr
			}

			result = &domain.CreateAPIKeyResult{
				APIKeySecret: secret,
				APIKey:       apiKey,
			}

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

func (s *apiKeySvcImpl) ListAPIKeys(ctx context.Context, cursor *string, limit int32, query *string, statuses []constants.APIKeyStatus) (*domain.ListAPIKeysResult, *apierror.APIError) {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkApiKeyAccess(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ownerAccountID := *identity.TargetAccountID

	meds := s.mediators()

	result, apiErr := meds.APIKey.List(ctx, domain.APIKeyListInput{
		OwnerAccountID: ownerAccountID,
		Cursor:         cursor,
		Limit:          limit,
		Query:          query,
		Statuses:       statuses,
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *apiKeySvcImpl) RevokeAPIKey(ctx context.Context, input domain.RevokeAPIKeyInput) *apierror.APIError {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.revoke")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkApiKeyAccess(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
	})

	if apiErr != nil {
		return apiErr
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *apiKeySvcImpl) *apierror.APIError {
			txMeds := svc.mediators()

			// Revoke the API key. The doc API key row is intentionally kept so that
			// Resolve() sees the revocation and refuses to auto-regenerate.
			if revokeErr := txMeds.APIKey.Revoke(txCtx, input.APIKeyID); revokeErr != nil {
				return revokeErr
			}

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

func (s *apiKeySvcImpl) RotateAPIKey(ctx context.Context, input domain.RotateAPIKeyInput) (*domain.CreateAPIKeyResult, *apierror.APIError) {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.rotate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := checkApiKeyAccess(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:      identity.Actor.ID,
		IdentityType: identity.Type,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CreateAPIKeyResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.CreateAPIKeyResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *apiKeySvcImpl) *apierror.APIError {
			txMeds := svc.mediators()

			// Rotate the API key
			secret, apiKey, rotateErr := txMeds.APIKey.Rotate(txCtx, domain.APIKeyRotateInput{
				AccountMode:  identity.AccountMode,
				APIKeyTypeID: input.APIKeyID,
				ExpiresAt:    input.ExpiresAt,
			})
			if rotateErr != nil {
				return rotateErr
			}

			// Sync the doc API key if one exists for this API key
			if syncErr := txMeds.DocAPIKey.SyncRotatedAPIKey(txCtx, domain.DocAPIKeySyncInput{
				OldAPIKeyID: input.APIKeyID,
				NewSecret:   secret,
				NewAPIKey:   apiKey,
			}); syncErr != nil {
				return syncErr
			}

			result = &domain.CreateAPIKeyResult{
				APIKeySecret: secret,
				APIKey:       apiKey,
			}

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

// checkApiKeyAccess makes sure the identity has permissions to preform CRUD operations on API keys
func checkApiKeyAccess(identity *types.Identity) *apierror.APIError {
	// We want to make sure we are only targeting an internal account
	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return apiErr
	}

	// For now, we only allow admin roles to make changes to API keys or fetch them
	if apiErr := types.CheckIsAdmin(identity); apiErr != nil {
		return apiErr
	}

	// If we don't have a target account ID, this request should fail
	if identity.TargetAccountID == nil {
		return apierror.NewValidationError("Target account ID is required to perform this action.")
	}

	return nil
}
