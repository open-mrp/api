package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/event"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/mediator"
	pwdutil "github.com/augno/api/services/auth-service/internal/password"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var authSvcTracer = tracing.GetTracer("auth-service.auth_service")

type authSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type AuthSvcConfig struct {
	Repos                 domain.RepoFactory
	MediatorFactory       domain.MediatorFactory
	NotificationPublisher domain.NotificationPublisher
	TxManager             TransactionManager
}

func NewAuthSvc(config AuthSvcConfig) domain.AuthSvc {
	return &authSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func DefaultAuthSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, rabbitmq *messaging.RabbitMQ, templatesDir string) AuthSvcConfig {
	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewNotificationPublisher(rabbitmq)

	mediatorFactory, err := mediator.NewMediatorFactory(mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		TemplatesDir:          templatesDir,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create mediator factory: %v", err))
	}

	return AuthSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func NewDefaultAuthSvc(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, rabbitmq *messaging.RabbitMQ, templatesDir string) domain.AuthSvc {
	return NewAuthSvc(DefaultAuthSvcConfig(queries, jwtSecret, pepper, frontendURL, rabbitmq, templatesDir))
}

func (s *authSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *authSvcImpl) withTx(ctx context.Context, fn func(context.Context, *authSvcImpl) *contracts.APIError) *contracts.APIError {
	if s.txManager == nil {
		panic("txManager is nil") // Should never happen
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *contracts.APIError {
		txSvc := &authSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			notificationPublisher: s.notificationPublisher,
			txManager:             s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *authSvcImpl) Login(ctx context.Context, identifier, password string) (*domain.LoginResult, *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.login")
	defer span.End()

	user, apiErr := s.mediators().Password.Validate(ctx, identifier, password)
	if apiErr != nil {
		return nil, apiErr
	}

	accessToken, apiErr := s.mediators().User.GenAuthAccessToken(ctx, user.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	var refreshToken string
	apiErr = s.withTx(ctx, func(txCtx context.Context, svc *authSvcImpl) *contracts.APIError {
		refreshTokenModel, err := svc.mediators().RefreshToken.Create(txCtx, user.ID, nil)
		if err != nil {
			return err
		}
		refreshToken = refreshTokenModel.Token
		return nil
	})

	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authSvcImpl) Register(ctx context.Context, name, email, password string) (*domain.LoginResult, *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.register")
	defer span.End()

	hashedPassword, err := pwdutil.HashPassword(ctx, password)
	if err != nil {
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "failed to hash password"))
	}

	var refreshToken string
	var user *types.User
	apiErr := s.withTx(ctx, func(txCtx context.Context, svc *authSvcImpl) *contracts.APIError {
		meds := svc.mediators()
		user, err = meds.User.Register(txCtx, name, email, hashedPassword)
		if err != nil {
			return err
		}

		refreshTokenModel, err := meds.RefreshToken.Create(txCtx, user.ID, nil)
		if err != nil {
			return err
		}

		refreshToken = refreshTokenModel.Token
		return nil
	})

	if apiErr != nil {
		return nil, apiErr
	}

	accessToken, apiErr := s.mediators().User.GenAuthAccessToken(ctx, user.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, apiErr
}

func (s *authSvcImpl) ValidateCredential(ctx context.Context, authToken string, targetAccountID string) (*types.Identity, *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.validate_credential")
	defer span.End()

	return s.mediators().User.ValidateCredential(ctx, authToken, targetAccountID)
}

func (s *authSvcImpl) RefreshToken(ctx context.Context, refreshToken string) (*domain.RefreshTokenResult, *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.refresh_token")
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

func (s *authSvcImpl) RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) (apiErr *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.request_password_reset")
	defer span.End()

	apiErr = s.withTx(ctx, func(txCtx context.Context, svc *authSvcImpl) *contracts.APIError {
		meds := svc.mediators()
		apiErr = meds.Password.RequestReset(txCtx, identifier, accountSlug)
		return apiErr
	})

	return
}

func (s *authSvcImpl) ResetPassword(ctx context.Context, token, newPassword string) (*domain.LoginResult, *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.reset_password")
	defer span.End()

	user, apiErr := s.mediators().Password.ValidatePasswordResetToken(ctx, token)
	if apiErr != nil {
		return nil, apiErr
	}

	var accessToken string
	var refreshToken string
	apiErr = s.withTx(ctx, func(txCtx context.Context, svc *authSvcImpl) *contracts.APIError {
		meds := svc.mediators()
		err := meds.Password.Update(txCtx, user, newPassword)
		if err != nil {
			return err
		}

		refreshTokenModel, err := meds.RefreshToken.Create(txCtx, user.ID, nil)
		if err != nil {
			return err
		}

		refreshToken = refreshTokenModel.Token
		return nil
	})

	if apiErr != nil {
		return nil, apiErr
	}

	accessToken, apiErr = s.mediators().User.GenAuthAccessToken(ctx, user.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authSvcImpl) RevokeRefreshToken(ctx context.Context, refreshToken string) (apiErr *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.revoke_refresh_token")
	defer span.End()

	apiErr = s.withTx(ctx, func(txCtx context.Context, svc *authSvcImpl) *contracts.APIError {
		meds := svc.mediators()
		return meds.RefreshToken.Revoke(txCtx, refreshToken)
	})

	return
}

func (s *authSvcImpl) UpdatePassword(ctx context.Context, userID string, oldPassword, newPassword string) (apiErr *contracts.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.update_password")
	defer span.End()

	user, apiErr := s.mediators().Password.Validate(ctx, userID, oldPassword)
	if apiErr != nil {
		return apiErr
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, svc *authSvcImpl) *contracts.APIError {
		meds := svc.mediators()
		return meds.Password.Update(txCtx, user, newPassword)
	})

	return apiErr
}
