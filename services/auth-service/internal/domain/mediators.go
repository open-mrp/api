package domain

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type APIKeyAccountAccess struct {
	APIKeyID     string
	AccountID    string
	RoleID       *string
	RoleTypeCode *string
	Permissions  map[string]bool
}

type APIKeyMed interface {
	// FindAndValidate takes in a raw API key string and validates it. If a valid API key
	// is found, we retreive it.
	//
	//  1. Parses the API key into its components.
	//  2. Finds the API key in the database.
	//  3. Verify the secret against the secret hash in the database.
	//  4. Make sure the key has not been revoked or expired.
	//  5. Returns the API key.
	FindAndValidate(ctx context.Context, apiKey string) (*APIKey, *apierror.APIError)

	// ParseKey parses a raw API key string into its components.
	//
	//  1. Parses the API key string into its components.
	//  2. Attempts to find the API key in the database.
	//  3. Verify the secret against the secret hash in the database.
	//  4. Make sure the key has not been revoked or expired.
	//  5. Returns the parsed API key.
	ParseKey(ctx context.Context, apiKey string) (*ParsedAPIKey, *apierror.APIError)

	// TouchIfNotRecent touches a given API key if it has not been used in the last 24 hours.
	//
	//  1. Checks if the API key has been used in the last 24 hours.
	//  2. If the API key has not been used in the last 24 hours, touches the API key.
	TouchIfNotRecent(ctx context.Context, apiKeyModel *APIKey) *apierror.APIError

	// Create creates a new API key for the given account mode.
	//
	//  1. Generates a new API key for the given account mode.
	//  2. Generates a HMAC for the secret.
	//  3. Generates a type ID for the API key.
	//  4. Inserts the API key into the database.
	//  5. Returns the new API key.
	Create(ctx context.Context, accountMode constants.AccountMode, ownerAccountID, roleID, name string, expiresAt *time.Time) (string, *APIKey, *apierror.APIError)

	// Rotate rotates a given API key.
	//
	//  1. Finds the API key in the database.
	//  2. Revokes the API key.
	//  3. Creates a new API key for the given account mode.
	//  4. Returns the new API key.
	Rotate(ctx context.Context, accountMode constants.AccountMode, apiKeyTypeID string, expiresAt *time.Time) (string, *APIKey, *apierror.APIError)

	// Revoke revokes an API key by its type ID.
	//
	//  1. Finds the API key in the database.
	//  2. Revokes the API key.
	Revoke(ctx context.Context, apiKeyTypeID string) *apierror.APIError

	// List lists all API keys for the given account mode.
	//
	//  1. Lists all API keys for the given account mode.
	//  2. Returns the API keys.
	List(ctx context.Context, accountMode constants.AccountMode, ownerAccountID string, cursor *string, limit int32, query *string, statuses []constants.APIKeyStatus) ([]*APIKey, int64, *apierror.APIError)

	// GetKeyAccountAccess gets the account access for a given API key.
	//
	//  1. Gets the account access for a given API key.
	//  2. Returns the account access.
	GetKeyAccountAccess(ctx context.Context, accountMode constants.AccountMode, apiKeyID int64, targetAccountID string) (*APIKeyAccountAccess, *apierror.APIError)
}

type PasswordMed interface {
	// RequestReset requests a password reset for a given identifier.
	//
	//  1. Checks if the identifier is valid.
	//  2. Generates a password reset token.
	//  3. Sends a password reset email to the user.
	RequestReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError

	// ValidatePasswordResetToken validates a password reset token and returns the user if it is valid.
	//
	//  1. Validates the password reset token.
	//  2. Returns the user if the password reset token is valid.
	ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *apierror.APIError)

	// Validate validates a given identifier and password and returns the user if it is valid.
	//
	//  1. Finds the user in the database.
	//  2. Makes sure the user has a hashed password.
	//  3. Compares the password against the hashed password.
	//  4. Returns the user if the identifier and password are valid.
	Validate(ctx context.Context, identifier, password string) (*types.User, *apierror.APIError)

	// Update updates a user's password.
	//
	//  1. Hashes the new password.
	//  2. Updates the user's password in the database.
	//  3. Revokes all refresh tokens for the user.
	//  4. Sends a password updated email to the user.
	Update(ctx context.Context, user *types.User, newPassword string) *apierror.APIError
}

type RefreshTokenMed interface {
	// Create creates a new refresh token for the given user ID that will expire in the given number of days.
	// If the number of days is not provided, it will default to 30 days.
	//
	//  1. Generates a new refresh token.
	//  2. Creates the refresh token in the database.
	//  3. Returns the refresh token.
	Create(ctx context.Context, userID string, expiresInDays *int) (*RefreshToken, *apierror.APIError)

	// Validate validates a refresh token and returns the user ID if it is valid.
	//
	//  1. Finds the refresh token in the database.
	//  2. Makes sure the refresh token has not been revoked or expired.
	//  3. Returns the user ID if the refresh token is valid.
	Validate(ctx context.Context, refreshToken string) (string, *apierror.APIError)

	// Revoke revokes a refresh token.
	//
	//  1. Finds the refresh token in the database.
	//  2. Makes sure the refresh token has not been revoked or expired.
	//  3. Revokes the refresh token.
	Revoke(ctx context.Context, refreshToken string) *apierror.APIError

	// RevokeAll revokes all refresh tokens associated with a user.
	//
	//  1. Revokes the refresh tokens for the given user ID.
	RevokeAll(ctx context.Context, userID string) *apierror.APIError
}

type RegisterUserInput struct {
	Name           string
	Email          string
	HashedPassword string
}

type UserMed interface {
	// GenAuthAccessToken mints an access token that can be used to authenticate requests to the API.
	//
	//  1. Encodes the user ID into a JWT.
	//  2. Returns the access token.
	GenAuthAccessToken(ctx context.Context, userID string) (string, *apierror.APIError)

	// GenPasswordResetAccessToken mints an access token that can be used to reset the password for the given user ID.
	//
	//  1. Encodes the user ID into a JWT.
	//  2. Returns the access token.
	GenPasswordResetAccessToken(ctx context.Context, userID string) (string, *apierror.APIError)

	// Register registers a new user.
	//
	//  1. Checks if the user already exists.
	//  2. Generates a new user ID.
	//  3. Creates the user in the database.
	//  4. Sends a welcome email to the user.
	//  5. Returns the user.
	Register(ctx context.Context, input RegisterUserInput) (*types.User, *apierror.APIError)

	// ValidateCredential validates a credentials provided by a request and returns an identity.
	//
	//  1. If the auth token is empty, we return an unauthenticated identity.
	//  2. If the auth token has the API key prefix, validate it as such.
	//  3. Otherwise, validate it as a user credential.
	//  4. Returns the identity.
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError)
}

type RequestIdentity struct {
	ActorID      string
	IdentityType types.IdentityType
}

type IdempotencyMed interface {
	// UpsertIdempotencyKey upserts the idempotency key for the request and returns the key.
	//
	//  1. Gets the idempotency key and handler name from the context.
	//  2. Generates a new type ID for the idempotency key.
	//  3. Computes the scope hash for the request.
	//  4. Finds the existing idempotency key in the database.
	//  5. If the idempotency key already exists, returns it.
	//  6. Otherwise, creates a new idempotency key and returns it.
	UpsertIdempotencyKey(ctx context.Context, identity *RequestIdentity) (*IdempotencyKey, *apierror.APIError)

	// CacheErrorResponse caches a non-transient error response and returns the original error.
	//
	//  1. If the error is transient, it returns the error without caching.
	//  2. Otherwise, caches the error response in the database.
	//  3. Returns the original error.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError

	// CacheSuccessResponse caches a successful response.
	//
	//  1. Marshals the response data into a JSON string.
	//  2. Caches the response in the database.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}
