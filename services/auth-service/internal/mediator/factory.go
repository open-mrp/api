package mediator

import (
	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/token"
)

// FactoryConfig declares the dependencies needed to build mediators.
type MediatorFactoryConfig struct {
	JWTSecret             string // #nosec G117 - Struct field, not a hardcoded credential
	APIKeyPepper          []byte
	NotificationPublisher domain.NotificationPublisher
	FrontendURL           string
	CoreClient            domain.AuthCoreClient
}

type mediatorFactoryImpl struct {
	jwtSecret             string
	apiKeyPepper          []byte
	notificationPublisher domain.NotificationPublisher
	frontendURL           string
	coreClient            domain.AuthCoreClient
}

// NewMediatorFactory creates a mediator factory that mirrors the repository factory
// style: inject shared dependencies once, then build mediators bound to a
// specific repository factory (e.g., per transaction).
func NewMediatorFactory(config MediatorFactoryConfig) domain.MediatorFactory {
	return &mediatorFactoryImpl{
		jwtSecret:             config.JWTSecret,
		apiKeyPepper:          config.APIKeyPepper,
		notificationPublisher: config.NotificationPublisher,
		frontendURL:           config.FrontendURL,
		coreClient:            config.CoreClient,
	}
}

func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
	refreshTokenMed := NewRefreshTokenMed(RefreshTokenMedConfig{
		Repos:    repoFactory,
		JWTUtils: token.NewJWTUtils(&token.JWTConfig{Secret: f.jwtSecret}),
	})

	apiKeyMed := NewAPIKeyMed(APIKeyMedConfig{
		Repos:       repoFactory,
		APIKeyUtils: apikey.NewAPIKeyUtils(&apikey.APIKeyConfig{Pepper: f.apiKeyPepper}),
		CoreClient:  f.coreClient,
	})

	userMed := NewUserMed(UserMedConfig{
		Repos:                 repoFactory,
		JWTSecret:             f.jwtSecret,
		RefreshTokenMed:       refreshTokenMed,
		APIKeyMed:             apiKeyMed,
		CoreClient:            f.coreClient,
		NotificationPublisher: f.notificationPublisher,
	})

	return domain.Mediators{
		User:   userMed,
		APIKey: apiKeyMed,
		Password: NewPasswordMed(PasswordMedConfig{
			Repos:                 repoFactory,
			RefreshTokenMed:       refreshTokenMed,
			JWTUtils:              token.NewJWTUtils(&token.JWTConfig{Secret: f.jwtSecret}),
			NotificationPublisher: f.notificationPublisher,
			FrontendURL:           f.frontendURL,
		}),
		RefreshToken: refreshTokenMed,
		Idempotency:  NewIdempotencyMed(IdempotencyMedConfig{Repos: repoFactory}),
	}
}
