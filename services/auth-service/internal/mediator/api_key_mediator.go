package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	tracing "github.com/augno/api/shared/tracing"
)

var apiKeyMedTracer = tracing.GetTracer("auth-service.api_key_mediator")

type apiKeyMedImpl struct {
	repos      domain.RepoFactory
	pepper     []byte
	coreClient domain.AuthCoreClient
}

type APIKeyMedConfig struct {
	Repos      domain.RepoFactory
	Pepper     []byte
	CoreClient domain.AuthCoreClient
}

func (c *APIKeyMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("api key mediator: repos is required")
	}
	if len(c.Pepper) == 0 {
		return fmt.Errorf("api key mediator: pepper is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("api key mediator: core client is required")
	}
	return nil
}

func NewAPIKeyMed(config *APIKeyMedConfig) domain.APIKeyMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &apiKeyMedImpl{
		repos:      config.Repos,
		pepper:     config.Pepper,
		coreClient: config.CoreClient,
	}
}

// ParseKey parses a raw API key string into its component parts.
//
// 1. Delegate to apikey.ParseAPIKey to extract the prefix, ID, and secret.
func (s *apiKeyMedImpl) ParseKey(ctx context.Context, apiKey string) (*apikey.ParsedAPIKey, *apierror.APIError) {
	_, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.parse")
	defer span.End()

	return apikey.ParseAPIKey(apiKey)
}

// FindAndValidate validates a raw API key string and returns the corresponding API key model.
//
// 1. Parse the raw API key string to extract the key ID and secret.
// 2. Look up the API key by its parsed ID.
// 3. Verify the secret HMAC against the stored hash using the pepper.
// 4. Check that the key is not expired or revoked.
func (s *apiKeyMedImpl) FindAndValidate(ctx context.Context, apiKey string) (*apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.find")
	defer span.End()

	parsedAPIKey, err := apikey.ParseAPIKey(apiKey)
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

	if !parsedAPIKey.VerifySecretHMAC(s.pepper, apiKeyModel.SecretHash) {
		return nil, tracing.Trace(span, s.newAuthError(apiKey))
	}

	if apiKeyModel.IsExpired() {
		return nil, tracing.Trace(span, apierror.NewExpiredAPIKeyError(apikey.ErrAPIKeyExpired))
	}

	if apiKeyModel.IsRevoked() {
		return nil, tracing.Trace(span, apierror.NewRevokedAPIKeyError(apikey.ErrAPIKeyRevoked))
	}

	return apiKeyModel, nil
}

// TouchIfNotRecent touches a given API key if it has not been used in the last 24 hours.
func (s *apiKeyMedImpl) TouchIfNotRecent(ctx context.Context, apiKeyModel *apikey.APIKey) *apierror.APIError {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.touch_if_not_recent")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()

	now := time.Now().UTC()
	threshold := now.Add(-24 * time.Hour)

	if apiKeyModel.ShouldTouch(threshold) {
		return apiKeyRepo.Touch(ctx, apiKeyModel.ID)
	}

	return nil
}

// Create creates a new API key for the requested account mode and persists it.
//
// 1. Generate a new parsed API key with a random secret for the given account mode.
// 2. Compute the HMAC hash of the secret using the pepper.
// 3. Generate a unique type ID and build the API key model.
// 4. Persist the API key in the repository.
// 5. Re-fetch the key to populate joined fields (role name, type code).
func (s *apiKeyMedImpl) Create(ctx context.Context, input domain.APIKeyCreateInput) (string, *apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.create")
	defer span.End()

	parsedKey, apiErr := apikey.GenParsedAPIKey(input.AccountMode, nil)
	if apiErr != nil {
		return "", nil, apiErr
	}

	secretHash := parsedKey.GenSecretHMAC(s.pepper)

	typeID, apiErr := id.GenID(id.APIKeyIDPrefix, nil)
	if apiErr != nil {
		return "", nil, tracing.Trace(span, apiErr)
	}

	fullKey := parsedKey.String()
	apiKeyModel := &apikey.APIKey{
		TypeID:         typeID,
		KeyID:          parsedKey.ID,
		Name:           input.Name,
		SecretHash:     secretHash,
		RedactedValue:  parsedKey.RedactedValue(),
		OwnerAccountID: input.OwnerAccountID,
		RoleID:         input.RoleID,
		ExpiresAt:      input.ExpiresAt,
	}

	apiKeyRepo := s.repos.NewAPIKeyRepo()
	_, apiErr = apiKeyRepo.Create(ctx, apiKeyModel)
	if apiErr != nil {
		return "", nil, apiErr
	}

	// Re-fetch to populate role name and type code from the JOIN.
	created, apiErr := apiKeyRepo.FindByTypeID(ctx, apiKeyModel.TypeID, nil)
	if apiErr != nil {
		return "", nil, apiErr
	}

	return fullKey, created, nil
}

// Revoke revokes an API key by its type ID.
//
// 1. Verify the API key exists by looking it up.
// 2. Mark the API key as revoked in the repository.
func (s *apiKeyMedImpl) Revoke(ctx context.Context, apiKeyTypeID string) *apierror.APIError {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.revoke")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()

	if _, apiErr := apiKeyRepo.FindByTypeID(ctx, apiKeyTypeID, nil); apiErr != nil {
		return apiErr
	}

	return apiKeyRepo.Revoke(ctx, apiKeyTypeID)
}

// Rotate revokes the specified API key and creates a replacement with the same name,
// owner account, and role.
//
// 1. Look up the existing API key by type ID.
// 2. Revoke the existing key.
// 3. Create a new key using the old key's properties, with an optionally overridden expiration.
func (s *apiKeyMedImpl) Rotate(ctx context.Context, input domain.APIKeyRotateInput) (string, *apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.rotate")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()

	oldKey, apiErr := apiKeyRepo.FindByTypeID(ctx, input.APIKeyTypeID, nil)
	if apiErr != nil {
		return "", nil, apiErr
	}

	if apiErr := apiKeyRepo.Revoke(ctx, input.APIKeyTypeID); apiErr != nil {
		return "", nil, apiErr
	}

	effectiveExpiresAt := oldKey.ExpiresAt
	if input.ExpiresAt != nil {
		effectiveExpiresAt = input.ExpiresAt
	}

	return s.Create(ctx, domain.APIKeyCreateInput{
		AccountMode:    input.AccountMode,
		OwnerAccountID: oldKey.OwnerAccountID,
		RoleID:         oldKey.RoleID,
		Name:           oldKey.Name,
		ExpiresAt:      effectiveExpiresAt,
	})
}

// List returns a paginated list of API keys for the given owner account and filters.
//
// 1. Query the API key repository with the provided filters and pagination parameters.
// 2. Return the list of API keys and page info.
func (s *apiKeyMedImpl) List(ctx context.Context, input domain.APIKeyListInput) (*domain.ListAPIKeysResult, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.list")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()
	result, apiErr := apiKeyRepo.List(ctx, domain.APIKeyListRepoInput(input))
	if apiErr != nil {
		return nil, apiErr
	}
	return &domain.ListAPIKeysResult{
		APIKeys:  result.APIKeys,
		PageInfo: result.PageInfo,
	}, nil
}

// GetKeyAccountAccess returns the resolved account access for an API key targeting a specific account.
//
// 1. Look up the API key by its database ID.
// 2. Verify the key's owner account matches the target account.
// 3. Fetch the role permissions from core-service if a role is assigned.
// 4. Return the access record with role and permission details.
func (s *apiKeyMedImpl) GetKeyAccountAccess(ctx context.Context, input domain.APIKeyGetAccountAccessInput) (*domain.APIKeyAccountAccess, *apierror.APIError) {
	ctx, span := apiKeyMedTracer.Start(ctx, "mediator.api_key.get_key_account_access")
	defer span.End()

	apiKeyRepo := s.repos.NewAPIKeyRepo()
	apiKeyModel, err := apiKeyRepo.FindByDatabaseID(ctx, input.APIKeyID, nil)
	if err != nil {
		return nil, err
	}

	if apiKeyModel.OwnerAccountID != input.TargetAccountID {
		return nil, apierror.NewResourceNotFoundError("API key not found.")
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

	roleTypeCode := &apiKeyModel.RoleType
	if apiKeyModel.RoleType == "" {
		roleTypeCode = nil
	}

	return &domain.APIKeyAccountAccess{
		APIKeyID:    apiKeyModel.TypeID,
		AccountID:   input.TargetAccountID,
		RoleID:      roleID,
		RoleType:    roleTypeCode,
		Permissions: permissions,
	}, nil
}

func (s *apiKeyMedImpl) newAuthError(apiKey string) *apierror.APIError {
	return apierror.NewAuthenticationError(
		fmt.Sprintf("%s: %s.", apikey.ErrAPIKeyInvalid, apikey.SanitizeAPIKey(apiKey)),
	)
}
