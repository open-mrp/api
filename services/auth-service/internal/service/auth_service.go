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
	apierror "github.com/augno/api/shared/errors"
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

func (c *AuthSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("auth service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("auth service: mediator factory is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("auth service: notification publisher is required")
	}
	return nil
}

func NewAuthSvc(config *AuthSvcConfig) domain.AuthSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &authSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		notificationPublisher: config.NotificationPublisher,
		txManager:             config.TxManager,
	}
}

func (c *AuthSvcConfig) WithDefaults(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) *AuthSvcConfig {
	if c == nil {
		c = &AuthSvcConfig{}
	}

	repoFactory := repository.NewRepoFactory(queries)
	notificationPublisher := event.NewOutboxNotificationPublisher()

	mediatorFactory := mediator.NewMediatorFactory(&mediator.MediatorFactoryConfig{
		JWTSecret:             jwtSecret,
		APIKeyPepper:          pepper,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           frontendURL,
		CoreClient:            coreClient,
	})

	return &AuthSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		NotificationPublisher: notificationPublisher,
	}
}

func (s *authSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

// ValidateCredential validates an auth token and returns the resulting identity.
//
// 1. Delegate to the user mediator's ValidateCredential method.
//
// Behavior:
//   - If authToken is empty, returns an unauthenticated identity.
//   - If authToken is an API key credential, validates it as an API key.
//   - Otherwise, validates it as a user credential (JWT).
func (s *authSvcImpl) ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.validate_credential")
	defer span.End()

	return s.mediators().User.ValidateCredential(ctx, authToken, targetAccountID, actorAccountID)
}
