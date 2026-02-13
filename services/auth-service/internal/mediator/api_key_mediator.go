package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	tracing "github.com/augno/api/shared/tracing"
)

var apiKeyMedTracer = tracing.GetTracer("auth-service.api_key_mediator")

type apiKeyMedImpl struct {
	repos       domain.RepoFactory
	apiKeyUtils domain.APIKeyUtils
	coreClient  domain.AuthCoreClient
}

type APIKeyMedConfig struct {
	Repos       domain.RepoFactory
	APIKeyUtils domain.APIKeyUtils
	CoreClient  domain.AuthCoreClient
}

// WithDefaults returns a new APIKeyMedConfig with zero-value fields replaced by defaults.
func (c *APIKeyMedConfig) WithDefaults() *APIKeyMedConfig {
	if c == nil {
		c = &APIKeyMedConfig{}
	}
	return &APIKeyMedConfig{
		Repos:       c.Repos,
		APIKeyUtils: c.APIKeyUtils,
		CoreClient:  c.CoreClient,
	}
}

func (c *APIKeyMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("api key mediator: repos is required")
	}
	if c.APIKeyUtils == nil {
		return fmt.Errorf("api key mediator: api key utils is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("api key mediator: core client is required")
	}
	return nil
}

func NewAPIKeyMed(config *APIKeyMedConfig) domain.APIKeyMed {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &apiKeyMedImpl{
		repos:       config.Repos,
		apiKeyUtils: config.APIKeyUtils,
		coreClient:  config.CoreClient,
	}
}

func DefaultAPIKeyMedConfig(queries *sqlc.Queries, pepper []byte) *APIKeyMedConfig {
	return &APIKeyMedConfig{
		Repos:       repository.NewRepoFactory(queries),
		APIKeyUtils: apikey.NewAPIKeyUtils(&apikey.APIKeyConfig{Pepper: pepper}),
	}
}

func NewDefaultAPIKeyMed(queries *sqlc.Queries, pepper []byte) domain.APIKeyMed {
	return NewAPIKeyMed(DefaultAPIKeyMedConfig(queries, pepper))
}

func (s *apiKeyMedImpl) ParseKey(ctx context.Context, apiKey string) (*domain.ParsedAPIKey, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.parse")
	defer span.End()

	return s.apiKeyUtils.Parse(ctx, apiKey)
}

func (s *apiKeyMedImpl) FindAndValidate(ctx context.Context, apiKey string) (*domain.APIKey, *apierror.APIError) {
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
		if apierror.IsNotFound(err) {
			return nil, tracing.Trace(span, s.newAuthError(apiKey))
		}
		return nil, err
	}

	apiKeyModel.RedactedValue = apikey.RedactAPIKeyValue(apiKeyModel, parsedAPIKey.AccountMode)

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
		return nil, tracing.Trace(span, apierror.NewExpiredAPIKeyError(apikey.ErrAPIKeyExpired))
	}

	// The key has been revoked
	if apiKeyModel.RevokedAt != nil && apiKeyModel.RevokedAt.Before(time.Now().UTC()) {
		return nil, tracing.Trace(span, apierror.NewRevokedAPIKeyError(apikey.ErrAPIKeyRevoked))
	}

	return apiKeyModel, nil
}

// TouchIfNotRecent touches a given API key if it has not been used in the last 24 hours.
func (s *apiKeyMedImpl) TouchIfNotRecent(ctx context.Context, apiKeyModel *domain.APIKey) *apierror.APIError {
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

func (s *apiKeyMedImpl) Create(ctx context.Context, accountMode constants.AccountMode, ownerAccountID, roleID, name string, expiresAt *time.Time) (string, *domain.APIKey, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.create")
	defer span.End()

	// Generate a new API key
	parsedKey, apiErr := s.apiKeyUtils.Gen(ctx, accountMode)
	if apiErr != nil {
		return "", nil, apiErr
	}

	// Generate a HMAC for the secret
	secretHash, apiErr := s.apiKeyUtils.GenSecretHMAC(ctx, parsedKey.Secret)
	if apiErr != nil {
		return "", nil, apiErr
	}

	// Generate a type ID for the API key
	typeID, apiErr := id.GenID(id.APIKeyIDPrefix, nil)
	if apiErr != nil {
		return "", nil, tracing.Trace(span, apiErr)
	}

	// Create the API key model
	fullKey := parsedKey.String()
	apiKeyModel := &domain.APIKey{
		TypeID:         typeID,
		KeyID:          parsedKey.ID,
		Name:           name,
		SecretHash:     secretHash,
		LastFour:       fullKey[len(fullKey)-4:],
		OwnerAccountID: ownerAccountID,
		RoleID:         roleID,
		ExpiresAt:      expiresAt,
	}

	apiKeyModel.RedactedValue = apikey.RedactAPIKeyValue(apiKeyModel, accountMode)

	// Save the API key to the database
	apiKeyRepo := s.repos.NewAPIKeyRepo()
	id, apiErr := apiKeyRepo.Create(ctx, apiKeyModel)
	if apiErr != nil {
		return "", nil, apiErr
	}

	apiKeyModel.ID = id

	return parsedKey.String(), apiKeyModel, nil
}

func (s *apiKeyMedImpl) Revoke(ctx context.Context, apiKeyTypeID string) *apierror.APIError {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.revoke")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()

	if _, apiErr := apiKeyRepo.FindByTypeID(ctx, apiKeyTypeID); apiErr != nil {
		return apiErr
	}

	return apiKeyRepo.Revoke(ctx, apiKeyTypeID)
}

func (s *apiKeyMedImpl) Rotate(ctx context.Context, accountMode constants.AccountMode, apiKeyTypeID string, expiresAt *time.Time) (string, *domain.APIKey, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.rotate")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()

	oldKey, apiErr := apiKeyRepo.FindByTypeID(ctx, apiKeyTypeID)
	if apiErr != nil {
		return "", nil, apiErr
	}

	if apiErr := apiKeyRepo.Delete(ctx, apiKeyTypeID); apiErr != nil {
		return "", nil, apiErr
	}

	effectiveExpiresAt := oldKey.ExpiresAt
	if expiresAt != nil {
		effectiveExpiresAt = expiresAt
	}

	return s.Create(ctx, accountMode, oldKey.OwnerAccountID, oldKey.RoleID, oldKey.Name, effectiveExpiresAt)
}

func (s *apiKeyMedImpl) List(ctx context.Context, accountMode constants.AccountMode, ownerAccountID string, cursor *string, limit int32, query *string, statuses []constants.APIKeyStatus) ([]*domain.APIKey, int64, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.list")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()
	return apiKeyRepo.List(ctx, accountMode, ownerAccountID, cursor, limit, query, statuses)
}

func (s *apiKeyMedImpl) GetKeyAccountAccess(ctx context.Context, accountMode constants.AccountMode, apiKeyID int64, targetAccountID string) (*domain.APIKeyAccountAccess, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.get_key_account_access")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()
	apiKeyModel, err := apiKeyRepo.FindByDatabaseID(ctx, apiKeyID)
	if err != nil {
		if apierror.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if apiKeyModel.OwnerAccountID != targetAccountID {
		return nil, nil
	}

	permissions := map[string]bool{}
	if apiKeyModel.RoleID != "" {
		perms, err := s.coreClient.GetRolePermissions(ctx, apiKeyModel.RoleID)
		if err != nil {
			return nil, err
		}
		permissions = perms
	}

	roleID := &apiKeyModel.RoleID
	if apiKeyModel.RoleID == "" {
		roleID = nil
	}

	roleTypeCode := &apiKeyModel.RoleTypeCode
	if apiKeyModel.RoleTypeCode == "" {
		roleTypeCode = nil
	}

	apiKeyModel.RedactedValue = apikey.RedactAPIKeyValue(apiKeyModel, accountMode)

	return &domain.APIKeyAccountAccess{
		APIKeyID:     apiKeyModel.TypeID,
		AccountID:    targetAccountID,
		RoleID:       roleID,
		RoleTypeCode: roleTypeCode,
		Permissions:  permissions,
	}, nil
}

func (s *apiKeyMedImpl) newAuthError(apiKey string) *apierror.APIError {
	return apierror.NewAuthenticationError(
		fmt.Sprintf("%s: %s.", apikey.ErrAPIKeyInvalid, s.apiKeyUtils.SanitizeForDisplay(apiKey)),
	)
}
