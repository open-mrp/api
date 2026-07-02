package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var docAPIKeySvcTracer = tracing.GetTracer("auth-service.doc_api_key_service")

type docAPIKeySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type DocAPIKeySvcConfig struct {
	// Repos (required) is the repository factory for auth persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (optional; default: nil) wraps multi-step operations in database transactions. It is not validated at construction; transactional code paths panic at runtime if it is unset.
	TxManager TransactionManager
}

func (c *DocAPIKeySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("doc api key service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("doc api key service: mediator factory is required")
	}
	return nil
}

func NewDocAPIKeySvc(config *DocAPIKeySvcConfig) domain.DocAPIKeySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &docAPIKeySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func BuildDocAPIKeySvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient, encryptionKey []byte) *DocAPIKeySvcConfig {
	repoFactory := repository.NewRepoFactory(queries)

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:              jwtSecret,
		APIKeyPepper:           pepper,
		NotificationPublisher:  event.NewOutboxNotificationPublisher(),
		FrontendURL:            frontendURL,
		CoreClient:             coreClient,
		DocAPIKeyEncryptionKey: encryptionKey,
	})

	return &DocAPIKeySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
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
		}
		return fn(txCtx, txSvc)
	})
}

// GetOrCreateDocAPIKey returns an existing doc API key for the caller's sandbox account, or creates one if none exists.
//
//  1. Extract the identity from the context and verify the caller is an internal user
//     targeting a sandbox account.
//  2. Upsert an idempotency key; return the cached response if already finished.
//  3. Resolve the doc API key inside a transaction via the doc API key mediator.
//  4. Cache the success response and return the key with its plaintext secret.
//
// Preconditions:
//   - The caller must be an authenticated user with an internal actor type.
//   - The target account must be in sandbox mode.
func (s *docAPIKeySvcImpl) GetOrCreateDocAPIKey(ctx context.Context) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeySvcTracer.Start(ctx, "service.doc_api_key.get_or_create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsUser(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}
	if identity.AccountMode != constants.AccountModeSandbox {
		return nil, tracing.Trace(span, apierror.NewValidationError("A sandbox account ID is required. Production account IDs are not accepted."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:         identity.Actor.ID,
		IdentityType:    identity.Type,
		TargetAccountID: &identity.Target.AccountID,
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
		var result *domain.GetOrCreateDocAPIKeyResult
		resolveTx := func() *apierror.APIError {
			return s.withTx(ctx, func(txCtx context.Context, txSvc *docAPIKeySvcImpl) *apierror.APIError {
				txMeds := txSvc.mediators()
				var resolveErr *apierror.APIError
				result, resolveErr = txMeds.DocAPIKey.Resolve(txCtx, identity.Target.AccountID)
				if resolveErr != nil {
					return resolveErr
				}
				return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})
		}
		apiErr = resolveTx()
		if apiErr != nil && apiErr.Code == apierror.ErrorCodeResourceExists {
			// A concurrent request created the doc API key first (unique key on owner_account_id). Re-resolve in a fresh transaction to return the winner's key instead of propagating the conflict.
			apiErr = resolveTx()
		}
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}
