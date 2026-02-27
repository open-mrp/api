package mediator

import (
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
)

// FactoryConfig declares the dependencies needed to build mediators.
type MediatorFactoryConfig struct {
	JWTSecret              string // #nosec G117 - Struct field, not a hardcoded credential
	APIKeyPepper           []byte
	NotificationPublisher  domain.NotificationPublisher
	FrontendURL            string
	CoreClient             domain.AuthCoreClient
	DocAPIKeyEncryptionKey []byte
}

type mediatorFactoryImpl struct {
	jwtSecret              string
	apiKeyPepper           []byte
	notificationPublisher  domain.NotificationPublisher
	frontendURL            string
	coreClient             domain.AuthCoreClient
	docAPIKeyEncryptionKey []byte
}

func (c *MediatorFactoryConfig) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("mediator factory: jwt secret is required")
	}
	if len(c.APIKeyPepper) == 0 {
		return fmt.Errorf("mediator factory: api key pepper is required")
	}
	if c.NotificationPublisher == nil {
		return fmt.Errorf("mediator factory: notification publisher is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("mediator factory: frontend url is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("mediator factory: core client is required")
	}
	return nil
}

// NewMediatorFactory creates a mediator factory that mirrors the repository factory
// style: inject shared dependencies once, then build mediators bound to a
// specific repository factory (e.g., per transaction).
func NewMediatorFactory(config *MediatorFactoryConfig) domain.MediatorFactory {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &mediatorFactoryImpl{
		jwtSecret:              config.JWTSecret,
		apiKeyPepper:           config.APIKeyPepper,
		notificationPublisher:  config.NotificationPublisher,
		frontendURL:            config.FrontendURL,
		coreClient:             config.CoreClient,
		docAPIKeyEncryptionKey: config.DocAPIKeyEncryptionKey,
	}
}

func (f *mediatorFactoryImpl) Build(repoFactory domain.RepoFactory) domain.Mediators {
	refreshTokenMed := NewRefreshTokenMed(&RefreshTokenMedConfig{
		Repos: repoFactory,
	})

	apiKeyMed := NewAPIKeyMed(&APIKeyMedConfig{
		Repos:      repoFactory,
		Pepper:     f.apiKeyPepper,
		CoreClient: f.coreClient,
	})

	userMed := NewUserMed(&UserMedConfig{
		Repos:                 repoFactory,
		JWTSecret:             f.jwtSecret,
		RefreshTokenMed:       refreshTokenMed,
		APIKeyMed:             apiKeyMed,
		CoreClient:            f.coreClient,
		NotificationPublisher: f.notificationPublisher,
	})

	meds := domain.Mediators{
		User:   userMed,
		APIKey: apiKeyMed,
		Password: NewPasswordMed(&PasswordMedConfig{
			Repos:                 repoFactory,
			RefreshTokenMed:       refreshTokenMed,
			JWTSecret:             f.jwtSecret,
			NotificationPublisher: f.notificationPublisher,
			FrontendURL:           f.frontendURL,
		}),
		RefreshToken: refreshTokenMed,
		Idempotency:  NewIdempotencyMed(&IdempotencyMedConfig{Repos: repoFactory}),
		Registration: NewRegistrationMed(&RegistrationMedConfig{
			Repos:                 repoFactory,
			NotificationPublisher: f.notificationPublisher,
			FrontendURL:           f.frontendURL,
			UserMed:               userMed,
			RefreshTokenMed:       refreshTokenMed,
		}),
	}

	if len(f.docAPIKeyEncryptionKey) > 0 {
		meds.DocAPIKey = NewDocAPIKeyMed(&DocAPIKeyMedConfig{
			Repos:         repoFactory,
			EncryptionKey: f.docAPIKeyEncryptionKey,
			CoreClient:    f.coreClient,
			APIKeyMed:     apiKeyMed,
		})
	}

	return meds
}
