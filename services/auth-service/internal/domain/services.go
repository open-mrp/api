package domain

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/apikey"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
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
	Name        string
	Email       string
	Password    string  // #nosec G117 - Struct field, not a hardcoded credential
	AccountSlug *string // Portal context for the "already registered" magic login link.
}

type AuthSvc interface {
	// ValidateCredential validates an auth token and returns the resulting identity.
	//
	// Behavior:
	//   - If authToken is empty, returns an unauthenticated identity.
	//   - If authToken is an API key credential, validates it as an API key.
	//   - Otherwise, validates it as a user credential.
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string, actorAccountID *string) (*types.Identity, *apierror.APIError)
}

type TokenSvc interface {
	// RefreshToken exchanges a valid refresh token for a new short-lived access token.
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResult, *apierror.APIError)

	// RevokeRefreshToken invalidates a refresh token so it can no longer be used to mint access tokens.
	RevokeRefreshToken(ctx context.Context, refreshToken string) *apierror.APIError
}

type UserSvc interface {
	// Login authenticates a user and returns a token pair (access + refresh) plus the user profile.
	Login(ctx context.Context, identifier, password string) (*LoginResult, *apierror.APIError)

	// Register creates a new user and returns a token pair so the user is immediately logged in.
	Register(ctx context.Context, input RegisterInput) (*LoginResult, *apierror.APIError)

	// MagicLogin exchanges a magic-login token for a token pair, logging the user in without a password.
	MagicLogin(ctx context.Context, token string) (*LoginResult, *apierror.APIError)
}

type PasswordSvc interface {
	// UpdatePassword updates a user's password.
	//
	// Side effects:
	//   - Revokes existing refresh tokens for the user.
	UpdatePassword(ctx context.Context, oldPassword, newPassword string) *apierror.APIError

	// ResetPassword completes the password reset flow and returns a token pair plus the user profile.
	//
	// Side effects:
	//   - Revokes existing refresh tokens for the user.
	ResetPassword(ctx context.Context, token, newPassword string) (*LoginResult, *apierror.APIError)

	// RequestPasswordReset initiates a password reset flow for the identifier.
	RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError
}

type CreateAPIKeyInput struct {
	RoleID    string
	Name      string
	ExpiresAt *time.Time
}

type CreateAPIKeyResult struct {
	APIKeySecret string
	APIKey       *apikey.APIKey
}

type RotateAPIKeyInput struct {
	APIKeyID  string
	ExpiresAt *time.Time
	// RevokeAt schedules when the old key is revoked. Nil or past/now means immediate revocation; a future instant keeps the old key valid until then.
	RevokeAt *time.Time
}

type RevokeAPIKeyInput struct {
	APIKeyID string
}

type ListAPIKeysResult struct {
	APIKeys  []*apikey.APIKey
	PageInfo pagination.PageInfo
}

type GetOrCreateDocAPIKeyResult struct {
	APIKeySecret string
	APIKey       *apikey.APIKey
}

type DocAPIKeySvc interface {
	// GetOrCreateDocAPIKey returns a documentation API key for the caller's target account.
	//
	// Authorization:
	//   - Requires an internal identity with a target account in context.
	//
	// Behavior:
	//   - Reuses an existing valid key.
	//   - Rotates and replaces an expired key.
	//
	// Side effects:
	//   - May rotate (revoke and replace) an existing doc API key.
	GetOrCreateDocAPIKey(ctx context.Context) (*GetOrCreateDocAPIKeyResult, *apierror.APIError)
}

type APIKeySvc interface {
	// GetAPIKey returns a single API key's metadata by its type ID.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	//   - The key must belong to the caller's target account.
	GetAPIKey(ctx context.Context, apiKeyID string, includes []string) (*apikey.APIKey, *apierror.APIError)

	// CreateAPIKey creates a new API key for the caller's target account.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	CreateAPIKey(ctx context.Context, input CreateAPIKeyInput) (*CreateAPIKeyResult, *apierror.APIError)

	// RotateAPIKey rotates (revokes and replaces) an API key.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	//
	// Side effects:
	//   - Revokes the prior API key.
	RotateAPIKey(ctx context.Context, input RotateAPIKeyInput) (*CreateAPIKeyResult, *apierror.APIError)

	// RevokeAPIKey revokes an API key without creating a replacement.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	RevokeAPIKey(ctx context.Context, input RevokeAPIKeyInput) *apierror.APIError

	// ListAPIKeys returns a paginated list of API keys for the caller's target account.
	//
	// Authorization:
	//   - Requires an internal admin identity with a target account in context.
	//
	// Pagination:
	//   - If cursor is non-nil, results begin after the provided cursor.
	//   - limit controls the maximum number of results returned.
	ListAPIKeys(ctx context.Context, cursor *string, limit int32, query *string, statuses []constants.APIKeyStatus, includes []string) (*ListAPIKeysResult, *apierror.APIError)

	BatchGetAPIKeysByIDs(ctx context.Context, ids []string) ([]*apikey.APIKey, *apierror.APIError)
}
