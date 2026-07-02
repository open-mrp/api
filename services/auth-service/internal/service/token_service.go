package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var tokenSvcTracer = tracing.GetTracer("auth-service.token_service")

type tokenSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type TokenSvcConfig struct {
	// Repos (required) is the repository factory for auth persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher

	// TxManager (optional; default: nil) wraps multi-step operations in database transactions. It is not validated at construction; transactional code paths panic at runtime if it is unset.
	TxManager TransactionManager
}

func (c *TokenSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("token service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("token service: mediator factory is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("token service: notification publisher is required")
	}
	return nil
}

func NewTokenSvc(config *TokenSvcConfig) domain.TokenSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &tokenSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func BuildTokenSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) *TokenSvcConfig {
	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &TokenSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func (s *tokenSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *tokenSvcImpl) withTx(ctx context.Context, fn func(context.Context, *tokenSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil") // Should never happen
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &tokenSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// RefreshToken validates a refresh token and mints a new access token.
//
// 1. Validate the refresh token and extract the associated user ID.
// 2. Mint a new access token for the user.
func (s *tokenSvcImpl) RefreshToken(ctx context.Context, refreshToken string) (*domain.RefreshTokenResult, *apierror.APIError) {
	ctx, span := tokenSvcTracer.Start(ctx, "service.token.refresh_token")
	defer span.End()

	userID, apiErr := s.mediators().RefreshToken.Validate(ctx, refreshToken)
	if apiErr != nil {
		return nil, apiErr
	}

	accessToken, apiErr := s.mediators().User.GenAuthAccessToken(ctx, userID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.RefreshTokenResult{AccessToken: accessToken}, nil
}

// RevokeRefreshToken revokes a refresh token so it can no longer be used to mint access tokens.
//
// 1. Upsert an idempotency key; return the cached response if already finished.
// 2. Revoke the refresh token inside a transaction via the refresh token mediator.
// 3. Cache the success response.
func (s *tokenSvcImpl) RevokeRefreshToken(ctx context.Context, refreshToken string) *apierror.APIError {
	ctx, span := tokenSvcTracer.Start(ctx, "service.token.revoke_refresh_token")
	defer span.End()

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *tokenSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()

			if err := txMeds.RefreshToken.Revoke(txCtx, refreshToken); err != nil {
				return err
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
