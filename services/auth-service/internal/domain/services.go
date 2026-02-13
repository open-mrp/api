package domain

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type LoginResult struct {
	User         *types.User
	RefreshToken string // #nosec G117 - Struct field, not a hardcoded credential
	AccessToken  string // #nosec G117 - Struct field, not a hardcoded credential
}

type RefreshTokenResult struct {
	AccessToken string // #nosec G117 - Struct field, not a hardcoded credential
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string // #nosec G117 - Struct field, not a hardcoded credential
}

type AuthSvc interface {
	// ValidateCredential validates a credentials provided by a request and returns an identity.
	//
	//  1. Validates the credentials.
	//  2. Returns the identity.
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError)
}

type TokenSvc interface {
	// RefreshToken exchanges a valid refresh token for a new short-lived access token.
	//
	//  1. Validates the refresh token.
	//  2. Generates a new access token.
	//  3. Returns the access token.
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResult, *apierror.APIError)

	// RevokeRefreshToken invalidates a refresh token so it can no longer be used to mint access tokens.
	//
	//  1. Revokes the refresh token.
	//  2. Caches the success response.
	RevokeRefreshToken(ctx context.Context, refreshToken string) *apierror.APIError
}

type UserSvc interface {
	// Login authenticates a user by identifier + password and returns a token pair (access + refresh) along with the user profile.
	//
	//  1. Validates the credentials.
	//  2. Generates a new access token.
	//  3. Generates a new refresh token.
	//  4. Returns the login result.
	Login(ctx context.Context, identifier, password string) (*LoginResult, *apierror.APIError)

	// Register creates a new user account and returns a token pair so the user is immediately logged in after registration.
	//
	//  1. Hashes the password.
	//  2. Calls the user mediator to register the user.
	//  3. Calls the refresh token mediator to create a new refresh token.
	//  4. Calls the user mediator to generate an access token.
	//  5. Caches the success response.
	//  6. Returns the login result.
	Register(ctx context.Context, input RegisterInput) (*LoginResult, *apierror.APIError)
}

type PasswordSvc interface {
	// UpdatePassword updates a user's password.
	//
	//  1. Hashes the new password.
	//  2. Calls the user mediator to update the user's password.
	//  3. Calls the refresh token mediator to revoke all refresh tokens for the user.
	//  4. Caches the success response.
	//  5. Returns the login result.
	UpdatePassword(ctx context.Context, userID string, oldPassword, newPassword string) *apierror.APIError

	// ResetPassword completes the password reset flow using the token from the reset email.
	//
	//  1. Validates the token.
	//  2. Hashes the new password.
	//  3. Calls the user mediator to update the user's password.
	//  4. Calls the refresh token mediator to create a new refresh token.
	//  5. Calls the user mediator to generate an access token.
	//  6. Caches the success response.
	//  7. Returns the login result.
	ResetPassword(ctx context.Context, token, newPassword string) (*LoginResult, *apierror.APIError)

	// RequestPasswordReset requests a password reset for a given identifier.
	//
	//  1. Calls the password mediator to request a password reset.
	//  2. Caches the success response.
	RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError
}

type CreateAPIKeyInput struct {
	RoleID    string
	Name      string
	ExpiresAt *time.Time
}

type CreateAPIKeyResult struct {
	APIKeySecret string
	APIKey       *APIKey
}

type RotateAPIKeyInput struct {
	APIKeyID  string
	ExpiresAt *time.Time
}

type RevokeAPIKeyInput struct {
	APIKeyID string
}

type ListAPIKeysResult struct {
	APIKeys    []*APIKey
	HasMore    bool
	NextCursor *string
}

type GetOrCreateDocAPIKeyResult struct {
	APIKeySecret string
	APIKey       *APIKey
}

type DocAPIKeySvc interface {
	// GetOrCreateDocAPIKey returns a sandbox API key for documentation.
	// Reuses an existing valid key, rotates an expired one, or creates a new one.
	//
	//  1. Get identity from context.
	//  2. Check if the identity is an internal actor, a user type, and an admin.
	//  3. Check if the identity has a target account ID.
	//  4. Resolve the sandbox account ID for the target account.
	//  5. Check for an existing doc API key for the sandbox account.
	//  6. If the key is expired, rotate it.
	//  7. If the key is valid, decrypt and return it.
	//  8. If no key exists, create a new one.
	//  9. Store the doc API key in the database.
	//  10. Return the doc API key secret and model.
	GetOrCreateDocAPIKey(ctx context.Context) (*GetOrCreateDocAPIKeyResult, *apierror.APIError)
}

type APIKeySvc interface {
	// CreateAPIKey creates a new API key for the given account mode.
	//
	//  1. Get identity from context.
	//  2. Check if the identity is an internal actor and an admin.
	//  3. Check if the identity has a target account ID.
	//  4. Create an API key for the given account mode.
	//  5. Return the API key secret and model.
	CreateAPIKey(ctx context.Context, input CreateAPIKeyInput) (*CreateAPIKeyResult, *apierror.APIError)

	// RotateAPIKey rotates a given API key.
	//
	//  1. Get identity from context.
	//  2. Check if the identity is an internal actor and an admin.
	//  3. Check if the identity has a target account ID.
	//  4. Rotate the API key.
	//  5. Return the API key secret and model.
	RotateAPIKey(ctx context.Context, input RotateAPIKeyInput) (*CreateAPIKeyResult, *apierror.APIError)

	// RevokeAPIKey revokes an API key without creating a replacement.
	//
	//  1. Get identity from context.
	//  2. Check if the identity is an internal actor and an admin.
	//  3. Check if the identity has a target account ID.
	//  4. Revoke the API key.
	RevokeAPIKey(ctx context.Context, input RevokeAPIKeyInput) *apierror.APIError

	// ListAPIKeys lists API keys for the target account.
	//
	//  1. Get identity from context.
	//  2. Check if the identity is an internal actor and an admin.
	//  3. Check if the identity has a target account ID.
	//  4. List API keys for the account.
	//  5. Return the list result with pagination info.
	ListAPIKeys(ctx context.Context, cursor *string, limit int32, query *string, statuses []constants.APIKeyStatus) (*ListAPIKeysResult, *apierror.APIError)
}
