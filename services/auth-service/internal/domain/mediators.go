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
	Resolve(ctx context.Context, sandboxAccountID string) (*GetOrCreateDocAPIKeyResult, *apierror.APIError)

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
	SyncRotatedAPIKey(ctx context.Context, input DocAPIKeySyncInput) *apierror.APIError
}

type APIKeyAccountAccess struct {
	APIKeyID    string
	AccountID   string
	RoleID      *string
	RoleType    *string
	Permissions map[string]bool
}

type APIKeyCreateInput struct {
	AccountMode    constants.AccountMode
	OwnerAccountID string
	RoleID         string
	Name           string
	ExpiresAt      *time.Time
}

type APIKeyRotateInput struct {
	AccountMode    constants.AccountMode
	APIKeyTypeID   string
	OwnerAccountID string
	ExpiresAt      *time.Time
	// RevokeAt schedules when the old key is revoked. Nil or past/now means immediate revocation; a future instant keeps the old key valid until then.
	RevokeAt *time.Time
}

type APIKeyRevokeInput struct {
	APIKeyTypeID   string
	OwnerAccountID string
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
	// 1. Parse the raw API key string to extract the key ID and secret.
	// 2. Look up the API key by its parsed ID.
	// 3. Verify the secret HMAC against the stored hash using the pepper.
	// 4. Check that the key is not expired or revoked.
	FindAndValidate(ctx context.Context, apiKey string) (*apikey.APIKey, *apierror.APIError)

	// ParseKey parses a raw API key string into its component parts.
	//
	// 1. Delegate to apikey.ParseAPIKey to extract the prefix, ID, and secret.
	ParseKey(ctx context.Context, apiKey string) (*apikey.ParsedAPIKey, *apierror.APIError)

	// TouchIfNotRecent touches a given API key if it has not been used in the last 24 hours.
	TouchIfNotRecent(ctx context.Context, apiKeyModel *apikey.APIKey) *apierror.APIError

	// Create creates a new API key for the requested account mode and persists it.
	//
	// 1. Generate a new parsed API key with a random secret for the given account mode.
	// 2. Compute the HMAC hash of the secret using the pepper.
	// 3. Generate a unique type ID and build the API key model.
	// 4. Persist the API key in the repository.
	// 5. Re-fetch the key to populate joined fields (role name, type code).
	Create(ctx context.Context, input APIKeyCreateInput) (string, *apikey.APIKey, *apierror.APIError)

	// Rotate revokes the specified API key and creates a replacement with the same name, owner account, and role.
	//
	// 1. Look up the existing API key by type ID.
	// 2. Revoke the existing key. By default revocation is immediate; a future RevokeAt schedules it (the old key keeps working until then) and is rejected with a validation error if more than 30 days out, while a past/now RevokeAt collapses to immediate.
	// 3. Create a new key using the old key's properties, with an optionally overridden expiration.
	//
	// Scoped to OwnerAccountID: returns a not-found error if the key does not exist for the requested owner.
	Rotate(ctx context.Context, input APIKeyRotateInput) (string, *apikey.APIKey, *apierror.APIError)

	// Revoke revokes an API key by its type ID.
	//
	// Scoped to ownerAccountID: returns a not-found error if the key does not exist for the given owner. This enforces tenant boundaries at the persistence layer as a backstop to service-layer ownership checks.
	Revoke(ctx context.Context, apiKeyTypeID string, ownerAccountID string) *apierror.APIError

	// List returns a paginated list of API keys for the given owner account and filters.
	//
	// 1. Query the API key repository with the provided filters and pagination parameters.
	// 2. Return the list of API keys and page info.
	List(ctx context.Context, input APIKeyListInput) (*ListAPIKeysResult, *apierror.APIError)

	// GetKeyAccountAccess returns the resolved account access for an API key targeting a specific account.
	//
	// 1. Look up the API key by its database ID.
	// 2. Verify the key's owner account matches the target account.
	// 3. Fetch the role permissions from core-service if a role is assigned.
	// 4. Return the access record with role and permission details.
	GetKeyAccountAccess(ctx context.Context, input APIKeyGetAccountAccessInput) (*APIKeyAccountAccess, *apierror.APIError)
}

type PasswordMed interface {
	// RequestReset initiates a password reset flow for the given identifier.
	//
	//  1. Look up the user by identifier; silently succeed if not found to avoid leaking information about registered identifiers.
	//  2. Generate a short-lived password reset JWT (15 minutes).
	//  3. Build the reset link, optionally scoped to an account slug.
	//  4. Send a password reset email with the link.
	//
	// Behavior:
	//   - Only returns an error for internal service failures; unknown identifiers
	//     succeed silently to prevent enumeration.
	//
	// Side effects:
	//   - Sends a password reset email.
	RequestReset(ctx context.Context, identifier string, accountSlug, portalBaseURL *string) *apierror.APIError

	// ValidatePasswordResetToken validates a password reset token and returns the associated user.
	//
	// 1. Decode and verify the JWT token as a password-reset type.
	// 2. Look up the user by the token's subject (user ID).
	// 3. Return an authentication error if the user is not found.
	ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *apierror.APIError)

	// Validate validates the identifier/password combination and returns the associated user.
	//
	//  1. Look up the user by identifier (email or user ID).
	//  2. If the user has no stored password hash, silently send a password reset email (when an email is on file) and return the generic invalid-credentials error so that the response is indistinguishable from a missing user or wrong password. This preserves recovery for legitimate passwordless users without leaking account state to unauthenticated callers.
	//  3. Compare the provided password against the stored hash.
	//  4. Return the user if the password matches; return an authentication error otherwise.
	Validate(ctx context.Context, identifier, password string) (*types.User, *apierror.APIError)

	// Update updates a user's password.
	//
	// 1. Hash the new password.
	// 2. Persist the updated password hash in the repository.
	// 3. Revoke all existing refresh tokens for the user.
	// 4. Send a password updated notification email.
	//
	// Side effects:
	//   - Updates the stored password hash.
	//   - Revokes all refresh tokens for the user.
	//   - Sends a password updated email.
	Update(ctx context.Context, user *types.User, newPassword string) *apierror.APIError
}

type RefreshTokenMed interface {
	// Create issues a new refresh token for the given user ID.
	//
	// 1. Generate a cryptographically random opaque token.
	// 2. Default expiration to 30 days if expiresInDays is nil.
	// 3. Persist the token in the repository with the computed expiration.
	Create(ctx context.Context, userID string, expiresInDays *int) (*RefreshToken, *apierror.APIError)

	// Validate validates a refresh token and returns the associated user ID.
	//
	// 1. Look up the refresh token in the repository.
	// 2. Verify the token is not revoked.
	// 3. Verify the token is not expired.
	// 4. Return the associated user ID.
	Validate(ctx context.Context, refreshToken string) (string, *apierror.APIError)

	// Revoke revokes a single refresh token, preventing it from being used to mint new access tokens.
	//
	// 1. Look up the refresh token in the repository.
	// 2. Verify the token is not already revoked or expired.
	// 3. Mark the token as revoked in the repository.
	Revoke(ctx context.Context, refreshToken string) *apierror.APIError

	// RevokeAll revokes all refresh tokens associated with a user.
	//
	// 1. Revoke all refresh tokens for the given user ID in the repository.
	//
	// Behavior:
	//   - Prevents stale tokens from being used after a password change.
	RevokeAll(ctx context.Context, userID string) *apierror.APIError
}

type RegisterUserInput struct {
	Name           string
	Email          string
	HashedPassword string
	AccountSlug    *string // Portal context for the "already registered" magic login link.
}

type UserMed interface {
	// GenAuthAccessToken mints an access token that can be used to authenticate requests to the API.
	GenAuthAccessToken(ctx context.Context, userID string) (string, *apierror.APIError)

	// Register registers a new user with the given input.
	//
	// 1. Check if a user with the given email already exists; return a validation error if so.
	// 2. Generate a unique user ID.
	// 3. Create the user record in the repository.
	// 4. Send a welcome email if the user has an email and name.
	//
	// Side effects:
	//   - Sends a welcome email.
	Register(ctx context.Context, input RegisterUserInput) (*types.User, *apierror.APIError)

	// ValidateMagicLoginToken validates a magic-login token and returns the associated user.
	ValidateMagicLoginToken(ctx context.Context, token string) (*types.User, *apierror.APIError)

	// SendAlreadyRegisteredEmail generates a magic login token and sends the "already registered" email so the user can log in with one click. This must be called outside a transaction so the outbox message is not rolled back.
	SendAlreadyRegisteredEmail(ctx context.Context, user *types.User, accountSlug, portalBaseURL *string)

	// ValidateCredential validates credentials provided by a request and returns an identity.
	//
	//  1. If authToken is empty, resolve the account mode for the target account (if provided) and return an unauthenticated identity.
	//  2. If authToken has the API key prefix, delegate to validateAPIKeyCredential.
	//  3. Otherwise, delegate to validateUserCredential for JWT-based validation.
	//
	// Behavior:
	//   - If authToken is empty, returns an unauthenticated identity.
	//   - If authToken is an API key credential, validates it as an API key.
	//   - Otherwise, validates it as a user credential (JWT).
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError)
}

type RegistrationMed interface {
	// CreateSession creates a new registration session or returns an existing active session for the given email (idempotent).
	//
	//  1. Check if the user already exists (noted but does not prevent session creation).
	//  2. Look for an existing non-expired session for the email; if found, update the plan code if different and resend the verification email.
	//  3. Generate a unique type ID and verification token.
	//  4. Create a new registration session record.
	//  5. Send the verification email.
	//
	// Side effects:
	//   - Sends a verification email to the user.
	CreateSession(ctx context.Context, input CreateRegistrationSessionInput) (*CreateRegistrationSessionResult, *apierror.APIError)

	// ResendVerificationEmail regenerates the verification token and resends the verification email.
	//
	// 1. Look up the session by type ID.
	// 2. Validate the session is not completed and email is not already verified.
	// 3. Generate a new verification token and update the session.
	// 4. Send the verification email with the new token.
	//
	// Side effects:
	//   - Rotates the verification token.
	//   - Sends a new verification email.
	ResendVerificationEmail(ctx context.Context, sessionID string) *apierror.APIError

	// VerifyToken verifies the email verification token and marks the session as email-verified.
	//
	// 1. Look up the session by verification token.
	// 2. Reject completed sessions.
	// 3. Check token expiry (24-hour TTL from last update).
	// 4. If already verified, return the current session without changes (idempotent).
	// 5. Check if a user already exists for the session's email.
	// 6. Mark the email as verified and advance the step to user_details.
	// 7. Re-fetch and return the updated session.
	VerifyToken(ctx context.Context, token string) (*RegistrationSession, *apierror.APIError)

	// CreateUserForSession creates a new user for the registration session and returns the user ID with auth tokens.
	//
	//  1. Look up the session by type ID and validate it is not completed and email is verified.
	//  2. If the session already has a user, generate tokens for the existing user (idempotent).
	//  3. Reject if an account already exists for the session's email — pre-existing accounts must authenticate via login, not by holding a verified session id.
	//  4. Hash the password and create a new user record.
	//  5. Associate the user with the session and update session data with the user name.
	//  6. Advance the session step to account_details.
	//  7. Generate and return an access token and refresh token.
	//
	// Side effects:
	//   - May create a new user record.
	//   - Associates the user with the session.
	//   - Advances the session step to account_details.
	CreateUserForSession(ctx context.Context, input CreateUserForRegistrationInput) (*CreateUserForRegistrationOutput, *apierror.APIError)

	// UpdateSession updates an in-progress registration session's step and form data.
	//
	// 1. Look up the session by type ID and validate it is not completed.
	// 2. Validate the step transition allows only forward progression.
	// 3. Merge the provided session data into the existing data (non-nil fields only).
	// 4. Persist the updated step and data.
	// 5. Re-fetch and return the refreshed session.
	UpdateSession(ctx context.Context, sessionID string, step *constants.RegistrationStep, sessionData *UpdateRegistrationSessionData) (*RegistrationSession, *apierror.APIError)

	// GetSession returns the registration session for the given type ID.
	//
	// 1. Look up and return the session by its type ID.
	GetSession(ctx context.Context, sessionID string) (*RegistrationSession, *apierror.APIError)

	// CompleteSession marks a registration session as completed and records the account ID.
	//
	// 1. Look up the session by type ID.
	// 2. Mark the session as completed with the provided account ID in the repository.
	CompleteSession(ctx context.Context, sessionID, accountID string) *apierror.APIError
}

type RequestIdentity struct {
	ActorID         string
	IdentityType    types.IdentityActorType
	TargetAccountID *string
}

type IdempotencyMed interface {
	// UpsertIdempotencyKey upserts and returns the idempotency key for the request scope.
	//
	//  1. Resolve the idempotency key from the request context, falling back to the request ID.
	//  2. Compute the scope hash from the actor, target account, service, handler, and key.
	//  3. Return the existing key for the scope hash when one exists.
	//  4. Otherwise persist a new key at the Started recovery point, re-fetching the
	//     existing row if a concurrent request inserted the same scope hash first.
	UpsertIdempotencyKey(ctx context.Context, identity *RequestIdentity) (*IdempotencyKey, *apierror.APIError)

	// CacheErrorResponse caches a non-transient error response for the idempotency key.
	//
	//  1. Return transient errors uncached so the client can retry.
	//  2. Persist non-transient errors as the cached response and mark the key finished.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError

	// CacheSuccessResponse caches a successful response for the idempotency key.
	//
	//  1. Marshal the response data to JSON.
	//  2. Persist it as the cached response and mark the key finished.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}
