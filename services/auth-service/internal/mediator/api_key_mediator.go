package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	tracing "github.com/augno/api/shared/tracing"
)

var apiKeyMedTracer = tracing.GetTracer("auth-service.api_key_mediator")

type apiKeyMedImpl struct {
	repos       domain.RepoFactory
	apiKeyUtils domain.APIKeyUtils
}

type APIKeyMedConfig struct {
	Repos       domain.RepoFactory
	APIKeyUtils domain.APIKeyUtils
}

func NewAPIKeyMed(config APIKeyMedConfig) domain.APIKeyMed {
	return &apiKeyMedImpl{
		repos:       config.Repos,
		apiKeyUtils: config.APIKeyUtils,
	}
}

func DefaultAPIKeyMedConfig(queries *sqlc.Queries, pepper []byte) APIKeyMedConfig {
	return APIKeyMedConfig{
		Repos:       repository.NewRepoFactory(queries),
		APIKeyUtils: apikey.NewAPIKeyUtils(apikey.DefaultAPIKeyConfig(pepper)),
	}
}

func NewDefaultAPIKeyMed(queries *sqlc.Queries, pepper []byte) domain.APIKeyMed {
	return NewAPIKeyMed(DefaultAPIKeyMedConfig(queries, pepper))
}

func (s *apiKeyMedImpl) FindAndValidate(ctx context.Context, apiKey string) (*domain.APIKey, *contracts.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.find")
	defer span.End()

	// Parse the received API key into its components
	parsedAPIKey, err := s.apiKeyUtils.Parse(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	apiKeyRepo := s.repos.NewAPIKeyRepo()
	apiKeyModel, err := apiKeyRepo.Find(ctx, parsedAPIKey.ID)
	if err != nil {
		return nil, err
	}

	// The key was not found in the database, so it's invalid
	if apiKeyModel == nil {
		return nil, tracing.Trace(span, s.newAuthError(apiKey))
	}

	valid, err := s.apiKeyUtils.VerifySecretHMAC(ctx, parsedAPIKey.Secret, apiKeyModel.SecretHash)
	if err != nil {
		return nil, err
	}

	// The secret hash doesn't match the secret in the database, so it's invalid
	if !valid {
		return nil, tracing.Trace(span, s.newAuthError(apiKey))
	}

	// The key has expired
	if apiKeyModel.ExpiresAt != nil && apiKeyModel.ExpiresAt.Before(time.Now().UTC()) {
		return nil, tracing.Trace(span, contracts.NewExpiredAPIKeyError(apikey.ErrAPIKeyExpired))
	}

	// The key has been revoked
	if apiKeyModel.RevokedAt != nil && apiKeyModel.RevokedAt.Before(time.Now().UTC()) {
		return nil, tracing.Trace(span, contracts.NewRevokedAPIKeyError(apikey.ErrAPIKeyRevoked))
	}

	return apiKeyModel, nil
}

// TouchIfNotRecent touches a given API key if it has not been used in the last 24 hours.
func (s *apiKeyMedImpl) TouchIfNotRecent(ctx context.Context, apiKeyModel *domain.APIKey) *contracts.APIError {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.touch_if_not_recent")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()

	now := time.Now().UTC()
	threshold := now.Add(-24 * time.Hour)

	// The API key has not been used before, so let's mark it as used now
	if apiKeyModel.LastUsedAt == nil {
		return apiKeyRepo.Touch(ctx, apiKeyModel.ID)
	}

	// We only want to note that a API key has made a request for an account once a day at most
	lastUsed := *apiKeyModel.LastUsedAt
	if lastUsed.Before(threshold) {
		return apiKeyRepo.Touch(ctx, apiKeyModel.ID)
	}

	return nil
}

func (s *apiKeyMedImpl) newAuthError(apiKey string) *contracts.APIError {
	return contracts.NewAuthenticationError(
		fmt.Sprintf("%s: %s.", apikey.ErrAPIKeyInvalid, s.apiKeyUtils.SanitizeForDisplay(apiKey)),
	)
}
