package mediator

import (
	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/email"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/shared/contracts"
)

// FactoryConfig declares the dependencies needed to build mediators.
type MediatorFactoryConfig struct {
	JWTSecret             string
	APIKeyPepper          []byte
	NotificationPublisher domain.NotificationPublisher
	FrontendURL           string
	TemplatesDir          string
}

type mediatorFactoryImpl struct {
	jwtSecret             string
	apiKeyPepper          []byte
	notificationPublisher domain.NotificationPublisher
	frontendURL           string
	templateRenderer      email.TemplateRenderer
}

// NewMediatorFactory creates a mediator factory that mirrors the repository factory
// style: inject shared dependencies once, then build mediators bound to a
// specific repository factory (e.g., per transaction).
func NewMediatorFactory(config MediatorFactoryConfig) (domain.MediatorFactory, *contracts.APIError) {
	var templateRenderer email.TemplateRenderer
	if config.TemplatesDir != "" {
		tr, err := email.NewTemplateRendererFromDir(config.TemplatesDir)
		if err != nil {
			return nil, err
		}
		templateRenderer = tr
	}

	return &mediatorFactoryImpl{
		jwtSecret:             config.JWTSecret,
		apiKeyPepper:          config.APIKeyPepper,
		notificationPublisher: config.NotificationPublisher,
		frontendURL:           config.FrontendURL,
		templateRenderer:      templateRenderer,
	}, nil
}

func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
	refreshTokenMed := NewRefreshTokenMed(RefreshTokenMedConfig{
		Repos:            repoFactory,
		JWTUtils:         token.NewJWTUtils(token.DefaultJWTConfig(f.jwtSecret)),
		OpaqueTokenUtils: token.NewOpaqueTokenUtils(token.DefaultOpaqueTokenConfig()),
	})

	apiKeyMed := NewAPIKeyMed(APIKeyMedConfig{
		Repos:       repoFactory,
		APIKeyUtils: apikey.NewAPIKeyUtils(apikey.DefaultAPIKeyConfig(f.apiKeyPepper)),
	})

	accountUserMed := NewAccountUserMed(AccountUserMedConfig{
		Repos: repoFactory,
	})

	return domain.Mediators{
		User: NewUserMed(UserMedConfig{
			Repos:                 repoFactory,
			JWTSecret:             f.jwtSecret,
			RefreshTokenMed:       refreshTokenMed,
			APIKeyMed:             apiKeyMed,
			AccountUserMed:        accountUserMed,
			NotificationPublisher: f.notificationPublisher,
			TemplateRenderer:      f.templateRenderer,
		}),
		APIKey:      apiKeyMed,
		AccountUser: accountUserMed,
		Password: NewPasswordMed(PasswordMedConfig{
			Repos:                 repoFactory,
			RefreshTokenMed:       refreshTokenMed,
			JWTUtils:              token.NewJWTUtils(token.DefaultJWTConfig(f.jwtSecret)),
			OpaqueTokenUtils:      token.NewOpaqueTokenUtils(token.DefaultOpaqueTokenConfig()),
			NotificationPublisher: f.notificationPublisher,
			TemplateRenderer:      f.templateRenderer,
			FrontendURL:           f.frontendURL,
		}),
		RefreshToken: refreshTokenMed,
	}
}
