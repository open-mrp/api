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

var passwordSvcTracer = tracing.GetTracer("auth-service.password_service")

type passwordSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type PasswordSvcConfig struct {
	Repos                 domain.RepoFactory
	MediatorFactory       domain.MediatorFactory
	NotificationPublisher domain.NotificationPublisher
	TxManager             TransactionManager
}

// WithDefaults returns a new PasswordSvcConfig with zero-value fields replaced by defaults.
func (c *PasswordSvcConfig) WithDefaults() *PasswordSvcConfig {
	if c == nil {
		c = &PasswordSvcConfig{}
	}
	return &PasswordSvcConfig{
		Repos:                 c.Repos,
		MediatorFactory:       c.MediatorFactory,
		NotificationPublisher: c.NotificationPublisher,
		TxManager:             c.TxManager,
	}
}

func (c *PasswordSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("password service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("password service: mediator factory is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("password service: notification publisher is required")
	}
	return nil
}

func NewPasswordSvc(config *PasswordSvcConfig) domain.PasswordSvc {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &passwordSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func DefaultPasswordSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) *PasswordSvcConfig {
	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &PasswordSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func NewDefaultPasswordSvc(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) domain.PasswordSvc {
	return NewPasswordSvc(DefaultPasswordSvcConfig(queries, jwtSecret, pepper, frontendURL, coreClient))
}

func (s *passwordSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *passwordSvcImpl) withTx(ctx context.Context, fn func(context.Context, *passwordSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil") // Should never happen
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &passwordSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *passwordSvcImpl) RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError {
	ctx, span := passwordSvcTracer.Start(ctx, "service.password.request_password_reset")
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
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *passwordSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			if err := txMeds.Password.RequestReset(txCtx, identifier, accountSlug); err != nil {
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

func (s *passwordSvcImpl) ResetPassword(ctx context.Context, token, newPassword string) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := passwordSvcTracer.Start(ctx, "service.password.reset_password")
	defer span.End()

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch idempotencyKey.RecoveryPoint {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.LoginResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Error != nil {
			return nil, cached.Error
		}
		return cached.Data, nil

	case domain.RecoveryPointStarted:
		user, apiErr := meds.Password.ValidatePasswordResetToken(ctx, token)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		accessToken, apiErr := meds.User.GenAuthAccessToken(ctx, user.ID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var refreshToken string
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *passwordSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			err := txMeds.Password.Update(txCtx, user, newPassword)
			if err != nil {
				return err
			}

			refreshTokenModel, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
			if err != nil {
				return err
			}

			refreshToken = refreshTokenModel.Token

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, domain.LoginResult{
				User:         user,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			})
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		result := &domain.LoginResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

func (s *passwordSvcImpl) UpdatePassword(ctx context.Context, userID string, oldPassword, newPassword string) *apierror.APIError {
	ctx, span := passwordSvcTracer.Start(ctx, "service.password.update_password")
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
		user, apiErr := meds.Password.Validate(ctx, userID, oldPassword)
		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *passwordSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			if err := txMeds.Password.Update(txCtx, user, newPassword); err != nil {
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
