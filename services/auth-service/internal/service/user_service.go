package service

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	pwdutil "github.com/augno/api/services/auth-service/internal/password"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var userSvcTracer = tracing.GetTracer("auth-service.user_service")

type userSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type UserSvcConfig struct {
	Repos                 domain.RepoFactory
	MediatorFactory       domain.MediatorFactory
	NotificationPublisher domain.NotificationPublisher
	TxManager             TransactionManager
}

func NewUserSvc(config UserSvcConfig) domain.UserSvc {
	return &userSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func DefaultUserSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) UserSvcConfig {
	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return UserSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func NewDefaultUserSvc(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) domain.UserSvc {
	return NewUserSvc(DefaultUserSvcConfig(queries, jwtSecret, pepper, frontendURL, coreClient))
}

func (s *userSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *userSvcImpl) withTx(ctx context.Context, fn func(context.Context, *userSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil") // Should never happen
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &userSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *userSvcImpl) Login(ctx context.Context, identifier, password string) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.login")
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
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		user, apiErr := meds.Password.Validate(ctx, identifier, password)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		accessToken, apiErr := meds.User.GenAuthAccessToken(ctx, user.ID)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var refreshToken string
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			refreshTokenModel, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
			if err != nil {
				return err
			}
			refreshToken = refreshTokenModel.Token

			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, &domain.LoginResult{
				User:         user,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			})
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return &domain.LoginResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint.String()))
	}
}

func (s *userSvcImpl) Register(ctx context.Context, input domain.RegisterInput) (*domain.LoginResult, *apierror.APIError) {
	ctx, span := userSvcTracer.Start(ctx, "service.user.register")
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
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		hashedPassword, err := pwdutil.HashPassword(ctx, input.Password)
		if err != nil {
			apiErr := apierror.NewInternalError(err, "failed to hash password")
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		var result *domain.LoginResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, svc *userSvcImpl) *apierror.APIError {
			txMeds := svc.mediators()
			user, regErr := txMeds.User.Register(txCtx, domain.RegisterUserInput{
				Name:           input.Name,
				Email:          input.Email,
				HashedPassword: hashedPassword,
			})
			if regErr != nil {
				return regErr
			}

			refreshTokenModel, err := txMeds.RefreshToken.Create(txCtx, user.ID, nil)
			if err != nil {
				return err
			}

			accessToken, tokenErr := txMeds.User.GenAuthAccessToken(txCtx, user.ID)
			if tokenErr != nil {
				return tokenErr
			}

			result = &domain.LoginResult{
				User:         user,
				AccessToken:  accessToken,
				RefreshToken: refreshTokenModel.Token,
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
