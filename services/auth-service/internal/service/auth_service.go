package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/event"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/auth-service/internal/mediator"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var authSvcTracer = tracing.GetTracer("auth-service.auth_service")

type authSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	notificationPublisher domain.NotificationPublisher
	txManager             TransactionManager
}

type AuthSvcConfig struct {
	// Repos (required) is the repository factory for auth persistence.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// NotificationPublisher (required) publishes notification messages to the outbox.
	NotificationPublisher domain.NotificationPublisher

	// TxManager (optional; default: nil) wraps multi-step operations in database transactions. It is not validated at construction; transactional code paths panic at runtime if it is unset.
	TxManager TransactionManager
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

func BuildAuthSvcConfig(queries *sqlc.Queries, jwtSecret string, pepper []byte, frontendURL string, coreClient domain.AuthCoreClient) *AuthSvcConfig {
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
// Behavior:
//   - If authToken is empty, returns an unauthenticated identity.
//   - If authToken is an API key credential, validates it as an API key.
//   - Otherwise, validates it as a user credential (JWT).
func (s *authSvcImpl) ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError) {
	ctx, span := authSvcTracer.Start(ctx, "service.auth.validate_credential")
	defer span.End()

	return s.mediators().User.ValidateCredential(ctx, authToken, targetAccountID, actorAccountID)
}
