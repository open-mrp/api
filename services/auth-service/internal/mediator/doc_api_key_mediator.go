package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

const (
	docAPIKeyEncryptionKeyID = "doc-ak-key-id" // #nosec G101 -- identifier, not a credential
	docAPIKeyTTLDays         = 30
)

var docAPIKeyMedTracer = tracing.GetTracer("auth-service.doc_api_key_mediator")

type docAPIKeyMedImpl struct {
	repos         domain.RepoFactory
	encryptionKey []byte
	coreClient    domain.AuthCoreClient
	apiKeyMed     domain.APIKeyMed
}

type DocAPIKeyMedConfig struct {
	Repos         domain.RepoFactory
	EncryptionKey []byte
	CoreClient    domain.AuthCoreClient
	APIKeyMed     domain.APIKeyMed
}

func (c *DocAPIKeyMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("doc api key mediator: repos is required")
	}
	if len(c.EncryptionKey) == 0 {
		return fmt.Errorf("doc api key mediator: encryption key is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("doc api key mediator: core client is required")
	}
	if c.APIKeyMed == nil {
		return fmt.Errorf("doc api key mediator: api key mediator is required")
	}
	return nil
}

func NewDocAPIKeyMed(config *DocAPIKeyMedConfig) domain.DocAPIKeyMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &docAPIKeyMedImpl{
		repos:         config.Repos,
		encryptionKey: config.EncryptionKey,
		coreClient:    config.CoreClient,
		apiKeyMed:     config.APIKeyMed,
	}
}

// Resolve returns an existing doc API key for the given sandbox account, or creates one if needed.
//
// 1. Look up an existing doc API key for the sandbox account.
// 2. If none exists, create a new doc API key with the system admin role.
// 3. If the existing key is revoked, return an error indicating manual rotation is required.
// 4. If the existing key is expired, rotate it and return the new key.
// 5. Otherwise, decrypt and return the existing key's secret.
//
// Behavior:
//   - If a non-revoked, non-expired doc API key exists, it is returned.
//   - If the existing key is expired, a new key is created via rotation.
//   - If the existing key is revoked, returns an error indicating rotation is required.
func (m *docAPIKeyMedImpl) Resolve(ctx context.Context, sandboxAccountID string) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeyMedTracer.Start(ctx, "mediator.doc_api_key.resolve")
	defer span.End()

	existing, apiErr := m.repos.NewDocAPIKeyRepo().FindBySandboxAccountID(ctx, sandboxAccountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return m.createDocAPIKey(ctx, sandboxAccountID)
		}
		return nil, tracing.Trace(span, apiErr)
	}

	// We assume that if a user revokes this API key they would prefer to not auto-create a sandbox
	// API key. They can manually rotate it to allow this setting.
	if existing.IsAPIKeyRevoked() {
		return nil, tracing.Trace(span, apierror.NewValidationError("The documentation API key has been revoked. Please rotate it manually if you would like to continue using it."))
	}

	// If the existing key is expired, we rotate it.
	if existing.IsAPIKeyExpired() {
		return m.rotateDocAPIKey(ctx, sandboxAccountID, existing)
	}

	return m.returnExistingKey(ctx, existing)
}

// SyncRotatedAPIKey updates doc API key state after the underlying API key has been rotated.
//
// 1. Look up the existing doc API key by the old API key ID.
// 2. If no doc API key exists for the old key, return without error (no-op).
// 3. Delete the old doc API key record.
// 4. Encrypt the new secret using AES-GCM.
// 5. Create a new doc API key record pointing to the rotated API key.
//
// Behavior:
//   - No-op if no doc API key exists for the old API key.
//
// Side effects:
//   - Deletes the old doc API key record (if present).
//   - Creates a new doc API key record using the rotated API key and secret.
func (m *docAPIKeyMedImpl) SyncRotatedAPIKey(ctx context.Context, input domain.DocAPIKeySyncInput) *apierror.APIError {
	ctx, span := docAPIKeyMedTracer.Start(ctx, "mediator.doc_api_key.sync_rotated_api_key")
	defer span.End()

	existingDocKey, findErr := m.repos.NewDocAPIKeyRepo().FindByAPIKeyID(ctx, input.OldAPIKeyID)
	if findErr != nil {
		if findErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil
		}
		return tracing.Trace(span, findErr)
	}

	// Delete the old doc API key row so we can create a fresh one for the new key.
	if deleteErr := m.repos.NewDocAPIKeyRepo().Delete(ctx, existingDocKey.ID); deleteErr != nil {
		return tracing.Trace(span, deleteErr)
	}

	encrypted, encErr := crypto.EncryptAESGCM([]byte(input.NewSecret), m.encryptionKey, []byte(input.NewAPIKey.TypeID), docAPIKeyEncryptionKeyID)
	if encErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(encErr, "Failed to encrypt doc API key secret."))
	}

	typeID, genErr := id.GenID(id.DocAPIKeyIDPrefix, nil)
	if genErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(genErr, "Failed to generate doc API key ID."))
	}

	if _, createErr := m.repos.NewDocAPIKeyRepo().Create(ctx, &apikey.DocAPIKey{
		TypeID:          typeID,
		APIKeyID:        input.NewAPIKey.TypeID,
		EncryptedSecret: encrypted,
	}); createErr != nil {
		return tracing.Trace(span, createErr)
	}

	return nil
}

// returnExistingKey decrypts and returns the existing doc API key secret and model.
func (m *docAPIKeyMedImpl) returnExistingKey(ctx context.Context, existing *apikey.DocAPIKey) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeyMedTracer.Start(ctx, "mediator.doc_api_key.return_existing")
	defer span.End()

	secret, err := crypto.DecryptAESGCM(existing.EncryptedSecret, m.encryptionKey, []byte(existing.APIKeyID))
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "failed to decrypt doc API key secret"))
	}

	apiKey, apiErr := m.repos.NewAPIKeyRepo().FindByTypeID(ctx, existing.APIKeyID, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.GetOrCreateDocAPIKeyResult{
		APIKeySecret: string(secret),
		APIKey:       apiKey,
	}, nil
}

func (m *docAPIKeyMedImpl) rotateDocAPIKey(ctx context.Context, sandboxAccountID string, existing *apikey.DocAPIKey) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeyMedTracer.Start(ctx, "mediator.doc_api_key.rotate")
	defer span.End()

	// Delete the existing doc API key so we can create a fresh one after rotation.
	if deleteErr := m.repos.NewDocAPIKeyRepo().Delete(ctx, existing.ID); deleteErr != nil {
		return nil, tracing.Trace(span, deleteErr)
	}

	expiresAt := time.Now().UTC().AddDate(0, 0, docAPIKeyTTLDays)

	newSecret, newAPIKey, rotateErr := m.apiKeyMed.Rotate(ctx, domain.APIKeyRotateInput{
		AccountMode:    constants.AccountModeSandbox,
		APIKeyTypeID:   existing.APIKeyID,
		OwnerAccountID: sandboxAccountID,
		ExpiresAt:      &expiresAt,
	})
	if rotateErr != nil {
		return nil, tracing.Trace(span, rotateErr)
	}

	// Encrypt the new secret key so we can retrieve it at will
	encrypted, encErr := crypto.EncryptAESGCM([]byte(newSecret), m.encryptionKey, []byte(newAPIKey.TypeID), docAPIKeyEncryptionKeyID)
	if encErr != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(encErr, "failed to encrypt doc API key secret"))
	}

	typeID, genErr := id.GenID(id.DocAPIKeyIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	// Create a new doc API key record for the sandbox account.
	if _, createErr := m.repos.NewDocAPIKeyRepo().Create(ctx, &apikey.DocAPIKey{
		TypeID:          typeID,
		APIKeyID:        newAPIKey.TypeID,
		EncryptedSecret: encrypted,
	}); createErr != nil {
		return nil, tracing.Trace(span, createErr)
	}

	return &domain.GetOrCreateDocAPIKeyResult{
		APIKeySecret: newSecret,
		APIKey:       newAPIKey,
	}, nil
}

// createDocAPIKey creates a new doc API key for the sandbox account. We use the system-defined admin role for this purpose.
func (m *docAPIKeyMedImpl) createDocAPIKey(ctx context.Context, sandboxAccountID string) (*domain.GetOrCreateDocAPIKeyResult, *apierror.APIError) {
	ctx, span := docAPIKeyMedTracer.Start(ctx, "mediator.doc_api_key.create")
	defer span.End()

	// Get the admin role ID for the sandbox account.
	roleID, apiErr := m.coreClient.GetAdminRole(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	expiresAt := time.Now().UTC().AddDate(0, 0, docAPIKeyTTLDays)

	// Create a new API key for the sandbox account.
	secret, apiKey, createErr := m.apiKeyMed.Create(ctx, domain.APIKeyCreateInput{
		AccountMode:    constants.AccountModeSandbox,
		OwnerAccountID: sandboxAccountID,
		RoleID:         roleID,
		Name:           "Doc API Key [System Generated]",
		ExpiresAt:      &expiresAt,
	})
	if createErr != nil {
		return nil, tracing.Trace(span, createErr)
	}

	// Encrypt the new secret key so we can retrieve it at will
	encrypted, encErr := crypto.EncryptAESGCM([]byte(secret), m.encryptionKey, []byte(apiKey.TypeID), docAPIKeyEncryptionKeyID)
	if encErr != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(encErr, "failed to encrypt doc API key secret"))
	}

	typeID, genErr := id.GenID(id.DocAPIKeyIDPrefix, nil)
	if genErr != nil {
		return nil, tracing.Trace(span, genErr)
	}

	// Create a new doc API key record for the sandbox account.
	if _, createDocErr := m.repos.NewDocAPIKeyRepo().Create(ctx, &apikey.DocAPIKey{
		TypeID:          typeID,
		APIKeyID:        apiKey.TypeID,
		EncryptedSecret: encrypted,
	}); createDocErr != nil {
		return nil, tracing.Trace(span, createDocErr)
	}

	return &domain.GetOrCreateDocAPIKeyResult{
		APIKeySecret: secret,
		APIKey:       apiKey,
	}, nil
}
