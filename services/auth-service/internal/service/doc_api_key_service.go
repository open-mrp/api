package service

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
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
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var docAPIKeySvcTracer = tracing.GetTracer("auth-service.doc_api_key_service")

const docAPIKeyTTLDays = 30

type docAPIKeySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	coreClient      domain.AuthCoreClient
	encryptionKey   []byte
}

type DocAPIKeySvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
	CoreClient      domain.AuthCoreClient
	EncryptionKey   []byte
}

// WithDefaults returns a new DocAPIKeySvcConfig with zero-value fields replaced by defaults.
func (c *DocAPIKeySvcConfig) WithDefaults() *DocAPIKeySvcConfig {
	if c == nil {
		c = &DocAPIKeySvcConfig{}
	}
	return &DocAPIKeySvcConfig{
		Repos:           c.Repos,
		MediatorFactory: c.MediatorFactory,
		TxManager:       c.TxManager,
		CoreClient:      c.CoreClient,
		EncryptionKey:   c.EncryptionKey,
	}
}

func (c *DocAPIKeySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("doc api key service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("doc api key service: mediator factory is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("doc api key service: core client is required")
	}
	if len(c.EncryptionKey) == 0 {
		return fmt.Errorf("doc api key service: encryption key is required")
	}
	return nil
}

func NewDocAPIKeySvc(config *DocAPIKeySvcConfig) domain.DocAPIKeySvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &docAPIKeySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		coreClient:      config.CoreClient,
		encryptionKey:   config.EncryptionKey,
	}
}

func DefaultDocAPIKeySvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient, encryptionKey []byte) *DocAPIKeySvcConfig {
	repoFactory := repository.NewRepoFactory(queries)

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: event.NewOutboxNotificationPublisher(),
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &DocAPIKeySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		CoreClient:      coreClient,
		EncryptionKey:   encryptionKey,
	}
}

func (s *docAPIKeySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *docAPIKeySvcImpl) withTx(ctx context.Context, fn func(context.Context, *docAPIKeySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &docAPIKeySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			coreClient:      s.coreClient,
			encryptionKey:   s.encryptionKey,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *docAPIKeySvcImpl) GetOrCreateDocAPIKey(ctx context.Context) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeySvcTracer.Start(ctx, "service.doc_api_key.get_or_create")
	defer span.End()

	identity, apiErr := s.validateDocAPIKeyAccess(ctx)
	if apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.GetOrCreateDocAPIKeyResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := s.resolveDocAPIKey(ctx, *identity.TargetAccountID, idempotencyKey.TypeID, meds)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

func (s *docAPIKeySvcImpl) validateDocAPIKeyAccess(ctx context.Context) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := types.CheckIsUser(identity); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := types.CheckIsAdmin(identity); apiErr != nil {
		return nil, apiErr
	}
	if identity.TargetAccountID == nil {
		return nil, apierror.NewValidationError("Target account ID is required.")
	}
	return identity, nil
}

func (s *docAPIKeySvcImpl) encryptSecret(secret string) (string, *apierror.APIError) {
	encrypted, err := crypto.EncryptAESGCM([]byte(secret), s.encryptionKey)
	if err != nil {
		return "", apierror.NewInternalError(err, "failed to encrypt doc API key secret")
	}
	return encrypted, nil
}

func (s *docAPIKeySvcImpl) resolveDocAPIKey(ctx context.Context, ownerAccountID string, idempotencyKeyTypeID string, meds domain.Mediators) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	sandboxAccountID, apiErr := s.coreClient.GetSandboxAccountByOwner(ctx, ownerAccountID)
	if apiErr != nil {
		return nil, apiErr
	}

	existing, apiErr := s.repos.NewDocAPIKeyRepo().FindBySandboxAccountID(ctx, sandboxAccountID)
	if apiErr != nil {
		return nil, apiErr
	}

	if existing != nil {
		isRevoked := existing.APIKeyRevokedAt != nil
		isExpired := existing.APIKeyExpiresAt != nil && existing.APIKeyExpiresAt.Before(time.Now().UTC())

		if isRevoked || isExpired {
			return s.rotateDocAPIKey(ctx, existing, sandboxAccountID, idempotencyKeyTypeID)
		}
		return s.returnExistingKey(ctx, existing, idempotencyKeyTypeID, meds)
	}

	return s.createDocAPIKey(ctx, sandboxAccountID, idempotencyKeyTypeID)
}

func (s *docAPIKeySvcImpl) returnExistingKey(ctx context.Context, existing *domain.DocAPIKey, idempotencyKeyTypeID string, meds domain.Mediators) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeySvcTracer.Start(ctx, "service.doc_api_key.return_existing")
	defer span.End()

	secret, err := crypto.DecryptAESGCM(existing.EncryptedSecret, s.encryptionKey)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "failed to decrypt doc API key secret"))
	}

	apiKey, apiErr := s.repos.NewAPIKeyRepo().FindByTypeID(ctx, existing.APIKeyID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiKey.RedactedValue = apikey.RedactAPIKeyValue(apiKey, constants.AccountModeSandbox)

	result := &domain.GetOrCreateDocAPIKeyResult{
		APIKeySecret: string(secret),
		APIKey:       apiKey,
	}

	if cacheErr := meds.Idempotency.CacheSuccessResponse(ctx, idempotencyKeyTypeID, result); cacheErr != nil {
		return nil, cacheErr
	}

	return result, nil
}

func (s *docAPIKeySvcImpl) rotateDocAPIKey(ctx context.Context, existing *domain.DocAPIKey, sandboxAccountID string, idempotencyKeyTypeID string) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeySvcTracer.Start(ctx, "service.doc_api_key.rotate")
	defer span.End()

	expiresAt := time.Now().UTC().AddDate(0, 0, docAPIKeyTTLDays)

	var result *domain.GetOrCreateDocAPIKeyResult
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *docAPIKeySvcImpl) *apierror.APIError {
		txMeds := txSvc.mediators()

		if deleteErr := txSvc.repos.NewDocAPIKeyRepo().DeleteAllBySandboxAccountID(txCtx, sandboxAccountID); deleteErr != nil {
			return deleteErr
		}

		newSecret, newAPIKey, rotateErr := txMeds.APIKey.Rotate(txCtx, constants.AccountModeSandbox, existing.APIKeyID, &expiresAt)
		if rotateErr != nil {
			return rotateErr
		}

		encrypted, encErr := txSvc.encryptSecret(newSecret)
		if encErr != nil {
			return encErr
		}

		typeID, genErr := id.GenID(id.DocAPIKeyIDPrefix, nil)
		if genErr != nil {
			return genErr
		}

		_, createErr := txSvc.repos.NewDocAPIKeyRepo().Create(txCtx, &domain.DocAPIKey{
			TypeID:          typeID,
			APIKeyID:        newAPIKey.TypeID,
			EncryptedSecret: encrypted,
		})
		if createErr != nil {
			return createErr
		}

		result = &domain.GetOrCreateDocAPIKeyResult{
			APIKeySecret: newSecret,
			APIKey:       newAPIKey,
		}

		return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, result)
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *docAPIKeySvcImpl) createDocAPIKey(ctx context.Context, sandboxAccountID string, idempotencyKeyTypeID string) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeySvcTracer.Start(ctx, "service.doc_api_key.create")
	defer span.End()

	roleID, apiErr := s.coreClient.GetAdminRole(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	expiresAt := time.Now().UTC().AddDate(0, 0, docAPIKeyTTLDays)

	var result *domain.GetOrCreateDocAPIKeyResult
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *docAPIKeySvcImpl) *apierror.APIError {
		txMeds := txSvc.mediators()

		secret, apiKey, createErr := txMeds.APIKey.Create(txCtx, constants.AccountModeSandbox, sandboxAccountID, roleID, "Documentation API Key [System Generated]", &expiresAt)
		if createErr != nil {
			return createErr
		}

		encrypted, encErr := txSvc.encryptSecret(secret)
		if encErr != nil {
			return encErr
		}

		typeID, genErr := id.GenID(id.DocAPIKeyIDPrefix, nil)
		if genErr != nil {
			return genErr
		}

		_, createDocErr := txSvc.repos.NewDocAPIKeyRepo().Create(txCtx, &domain.DocAPIKey{
			TypeID:          typeID,
			APIKeyID:        apiKey.TypeID,
			EncryptedSecret: encrypted,
		})
		if createDocErr != nil {
			return createDocErr
		}

		result = &domain.GetOrCreateDocAPIKeyResult{
			APIKeySecret: secret,
			APIKey:       apiKey,
		}

		return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, result)
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}
