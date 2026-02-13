package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var apiKeySvcTracer = tracing.GetTracer("auth-service.api_key_service")

type apiKeySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	encryptionKey   []byte
}

type APIKeySvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
	EncryptionKey   []byte
}

// WithDefaults returns a new APIKeySvcConfig with zero-value fields replaced by defaults.
func (c *APIKeySvcConfig) WithDefaults() *APIKeySvcConfig {
	if c == nil {
		c = &APIKeySvcConfig{}
	}
	return &APIKeySvcConfig{
		Repos:           c.Repos,
		MediatorFactory: c.MediatorFactory,
		TxManager:       c.TxManager,
		EncryptionKey:   c.EncryptionKey,
	}
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
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &apiKeySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		encryptionKey:   config.EncryptionKey,
	}
}

func DefaultAPIKeySvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient, encryptionKey []byte) *APIKeySvcConfig {
	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &APIKeySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		EncryptionKey:   encryptionKey,
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
			encryptionKey:   s.encryptionKey,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *apiKeySvcImpl) CreateAPIKey(ctx context.Context, input domain.CreateAPIKeyInput) (*domain.CreateAPIKeyResult, *apierror.APIError) {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := types.CheckIsAdmin(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Target account ID is required to create an API key."))
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

			secret, apiKey, createErr := txMeds.APIKey.Create(txCtx, identity.AccountMode, ownerAccountID, input.RoleID, input.Name, input.ExpiresAt)
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

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := types.CheckIsAdmin(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Target account ID is required to list API keys."))
	}

	ownerAccountID := *identity.TargetAccountID

	meds := s.mediators()

	apiKeys, _, apiErr := meds.APIKey.List(ctx, identity.AccountMode, ownerAccountID, cursor, limit, query, statuses)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	hasMore := len(apiKeys) == int(limit)
	var nextCursor *string
	if hasMore && len(apiKeys) > 0 {
		last := apiKeys[len(apiKeys)-1].TypeID
		nextCursor = &last
	}

	return &domain.ListAPIKeysResult{
		APIKeys:    apiKeys,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (s *apiKeySvcImpl) RevokeAPIKey(ctx context.Context, input domain.RevokeAPIKeyInput) *apierror.APIError {
	ctx, span := apiKeySvcTracer.Start(ctx, "service.api_key.revoke")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := types.CheckIsAdmin(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if identity.TargetAccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("Target account ID is required to revoke an API key."))
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

			if revokeErr := txMeds.APIKey.Revoke(txCtx, input.APIKeyID); revokeErr != nil {
				return revokeErr
			}

			if deleteErr := svc.repos.NewDocAPIKeyRepo().DeleteByAPIKeyID(txCtx, input.APIKeyID); deleteErr != nil {
				return deleteErr
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

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := types.CheckIsAdmin(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("Target account ID is required to rotate an API key."))
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
			docAPIKeyRepo := svc.repos.NewDocAPIKeyRepo()

			existingDocKey, findErr := docAPIKeyRepo.FindByAPIKeyID(txCtx, input.APIKeyID)
			if findErr != nil {
				return findErr
			}

			secret, apiKey, rotateErr := txMeds.APIKey.Rotate(txCtx, identity.AccountMode, input.APIKeyID, input.ExpiresAt)
			if rotateErr != nil {
				return rotateErr
			}

			if existingDocKey != nil {
				encrypted, encErr := crypto.EncryptAESGCM([]byte(secret), svc.encryptionKey)
				if encErr != nil {
					return apierror.NewInternalError(encErr, "failed to encrypt doc API key secret")
				}
				existingDocKey.APIKeyID = apiKey.TypeID
				existingDocKey.EncryptedSecret = encrypted
				if updateErr := docAPIKeyRepo.Update(txCtx, existingDocKey); updateErr != nil {
					return updateErr
				}
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
