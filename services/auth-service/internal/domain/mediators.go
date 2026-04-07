package domain

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type DocAPIKeySyncInput struct {
	OldAPIKeyID string
	NewSecret   string
	NewAPIKey   *apikey.APIKey
}

type DocAPIKeyMed interface {
	// Resolve returns an existing doc API key for the given sandbox account, or creates one if needed.
	//
	// Behavior:
	//   - If a non-revoked, non-expired doc API key exists, it is returned.
	//   - If the existing key is expired, a new key is created and returned.
	//   - If the existing key is revoked, returns an error indicating rotation is required.
	Resolve(ctx context.Context, sandboxAccountID string) (*GetOrCreateDocAPIKeyResult, *apierror.APIError)

	// SyncRotatedAPIKey updates doc API key state after the underlying API key has been rotated.
	//
	// Behavior:
	//   - No-op if no doc API key exists for the old API key.
	//
	// Side effects:
	//   - Deletes the old doc API key record (if present).
	//   - Creates a new doc API key record using the rotated API key and secret.
	SyncRotatedAPIKey(ctx context.Context, input DocAPIKeySyncInput) *apierror.APIError
}

type APIKeyAccountAccess struct {
	APIKeyID     string
	AccountID    string
	RoleID       *string
	RoleTypeCode *string
	Permissions  map[string]bool
}

type APIKeyCreateInput struct {
	AccountMode    constants.AccountMode
	OwnerAccountID string
	RoleID         string
	Name           string
	ExpiresAt      *time.Time
}

type APIKeyRotateInput struct {
	AccountMode  constants.AccountMode
	APIKeyTypeID string
	ExpiresAt    *time.Time
}

type APIKeyRevokeInput struct {
	APIKeyTypeID string
}

type APIKeyListInput struct {
	OwnerAccountID string
	Cursor         *string
	Limit          int32
	Query          *string
	Statuses       []constants.APIKeyStatus
	Includes       []string
}

type APIKeyGetAccountAccessInput struct {
	AccountMode     constants.AccountMode
	APIKeyID        int64
	TargetAccountID string
}

type APIKeyMed interface {
	// FindAndValidate validates a raw API key string and returns the corresponding API key model.
	//
	// Validation includes secret verification and revocation/expiration checks.
	FindAndValidate(ctx context.Context, apiKey string) (*apikey.APIKey, *apierror.APIError)

	// ParseKey parses and validates a raw API key string and returns the parsed components.
	//
	// Validation includes secret verification and revocation/expiration checks.
	ParseKey(ctx context.Context, apiKey string) (*apikey.ParsedAPIKey, *apierror.APIError)

	// TouchIfNotRecent records usage for the API key if it has not been used recently.
	TouchIfNotRecent(ctx context.Context, apiKeyModel *apikey.APIKey) *apierror.APIError

	// Create creates a new API key for the requested account mode.
	Create(ctx context.Context, input APIKeyCreateInput) (string, *apikey.APIKey, *apierror.APIError)

	// Rotate revokes the specified API key and creates a replacement.
	Rotate(ctx context.Context, input APIKeyRotateInput) (string, *apikey.APIKey, *apierror.APIError)

	// Revoke revokes an API key by its type ID.
	Revoke(ctx context.Context, apiKeyTypeID string) *apierror.APIError

	// List returns a paginated list of API keys for the given owner account and filters.
	List(ctx context.Context, input APIKeyListInput) (*ListAPIKeysResult, *apierror.APIError)

	// GetKeyAccountAccess returns the resolved account access implied by the API key for a target account.
	GetKeyAccountAccess(ctx context.Context, input APIKeyGetAccountAccessInput) (*APIKeyAccountAccess, *apierror.APIError)
}

type PasswordMed interface {
	// RequestReset initiates a password reset flow for the identifier.
	//
	// Side effects:
	//   - Sends a password reset email.
	RequestReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError

	// ValidatePasswordResetToken validates a password reset token and returns the associated user.
	ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *apierror.APIError)

	// Validate validates the identifier/password combination and returns the associated user.
	Validate(ctx context.Context, identifier, password string) (*types.User, *apierror.APIError)

	// Update updates a user's password.
	//
	// Side effects:
	//   - Updates the stored password hash.
	//   - Revokes all refresh tokens for the user.
	//   - Sends a password updated email.
	Update(ctx context.Context, user *types.User, newPassword string) *apierror.APIError
}

type RefreshTokenMed interface {
	// Create issues a refresh token for the user.
	//
	// If expiresInDays is nil, a default expiration is applied.
	Create(ctx context.Context, userID string, expiresInDays *int) (*RefreshToken, *apierror.APIError)

	// Validate validates a refresh token and returns the associated user ID.
	Validate(ctx context.Context, refreshToken string) (string, *apierror.APIError)

	// Revoke revokes a refresh token.
	Revoke(ctx context.Context, refreshToken string) *apierror.APIError

	// RevokeAll revokes all refresh tokens associated with a user.
	RevokeAll(ctx context.Context, userID string) *apierror.APIError
}

type RegisterUserInput struct {
	Name           string
	Email          string
	HashedPassword string
	AccountSlug    *string // Portal context for the "already registered" magic login link.
}

type UserMed interface {
	// GenAuthAccessToken mints an access token for authenticating API requests.
	GenAuthAccessToken(ctx context.Context, userID string) (string, *apierror.APIError)

	// Register registers a new user.
	//
	// Side effects:
	//   - Sends a welcome email.
	Register(ctx context.Context, input RegisterUserInput) (*types.User, *apierror.APIError)

	// ValidateMagicLoginToken validates a magic-login token and returns the associated user.
	ValidateMagicLoginToken(ctx context.Context, token string) (*types.User, *apierror.APIError)

	// SendAlreadyRegisteredEmail sends a magic login email to an existing user.
	// Must be called outside a transaction.
	SendAlreadyRegisteredEmail(ctx context.Context, user *types.User, accountSlug *string)

	// ValidateCredential validates an auth token and returns the resulting identity.
	//
	// Behavior:
	//   - If authToken is empty, returns an unauthenticated identity.
	//   - If authToken is an API key credential, validates it as an API key.
	//   - Otherwise, validates it as a user credential.
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError)
}

type RegistrationMed interface {
	// CreateSession creates a new registration session or returns an existing
	// active session for the given email (idempotent).
	//
	// Side effects:
	//   - Sends a verification email to the user.
	CreateSession(ctx context.Context, input CreateRegistrationSessionInput) (*CreateRegistrationSessionResult, *apierror.APIError)

	// ResendVerificationEmail regenerates the verification token and resends
	// the verification email for the given session.
	//
	// Side effects:
	//   - Rotates the verification token.
	//   - Sends a new verification email to the user.
	ResendVerificationEmail(ctx context.Context, sessionID string) *apierror.APIError

	// VerifyToken verifies the email token and marks the session as email-verified.
	// Advances the step to user_details. Idempotent: if the session is already
	// verified, returns the current session without error.
	VerifyToken(ctx context.Context, token string) (*RegistrationSession, *apierror.APIError)

	// CreateUserForSession creates or resolves a user for the registration
	// session and returns the user ID with auth tokens. If the session already
	// has a user, tokens are generated for the existing user (idempotent).
	//
	// Side effects:
	//   - May create a new user record.
	//   - Associates the user with the session.
	//   - Advances the session step to account_details.
	CreateUserForSession(ctx context.Context, input CreateUserForRegistrationInput) (*CreateUserForRegistrationOutput, *apierror.APIError)

	// UpdateSession updates an in-progress registration session's step and
	// form data. Only non-nil parameters are applied.
	// Returns the refreshed session after all updates.
	UpdateSession(ctx context.Context, sessionID string, step *constants.RegistrationStep, sessionData *UpdateRegistrationSessionData) (*RegistrationSession, *apierror.APIError)

	// GetSession returns the registration session for the given type ID.
	GetSession(ctx context.Context, sessionID string) (*RegistrationSession, *apierror.APIError)

	// CompleteSession marks a registration session as completed and records
	// the account ID.
	CompleteSession(ctx context.Context, sessionID, accountID string) *apierror.APIError
}

type RequestIdentity struct {
	ActorID         string
	IdentityType    types.IdentityActorType
	TargetAccountID *string
}

type IdempotencyMed interface {
	// UpsertIdempotencyKey upserts and returns the idempotency key for the request scope.
	UpsertIdempotencyKey(ctx context.Context, identity *RequestIdentity) (*IdempotencyKey, *apierror.APIError)

	// CacheErrorResponse caches a non-transient error response for the idempotency key.
	//
	// Behavior:
	//   - If the error is transient, it is returned without caching.
	//
	// Side effects:
	//   - Persists the error response for subsequent replays of the same idempotency key.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError

	// CacheSuccessResponse caches a successful response for the idempotency key.
	//
	// Side effects:
	//   - Persists the success response for subsequent replays of the same idempotency key.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}
