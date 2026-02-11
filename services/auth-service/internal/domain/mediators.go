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
	FindAndValidate(ctx context.Context, apiKey string) (*APIKey, *apierror.APIError)
	ParseKey(ctx context.Context, apiKey string) (*ParsedAPIKey, *apierror.APIError)
	TouchIfNotRecent(ctx context.Context, apiKeyModel *APIKey) *apierror.APIError
	Create(ctx context.Context, accountMode constants.AccountMode, ownerAccountID, roleID, name string, expiresAt *time.Time) (string, *APIKey, *apierror.APIError)
	List(ctx context.Context, accountMode constants.AccountMode, ownerAccountID string, cursor *string, limit int32, query *string) ([]*APIKey, int64, *apierror.APIError)
	GetKeyAccountAccess(ctx context.Context, accountMode constants.AccountMode, apiKeyID int64, targetAccountID string) (*APIKeyAccountAccess, *apierror.APIError)
}

type PasswordMed interface {
	RequestReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError
	ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *apierror.APIError)
	Validate(ctx context.Context, identifier, password string) (*types.User, *apierror.APIError)
	Update(ctx context.Context, user *types.User, newPassword string) *apierror.APIError
}

type RefreshTokenMed interface {
	Create(ctx context.Context, userID string, expiresInDays *int) (*RefreshToken, *apierror.APIError)
	Validate(ctx context.Context, refreshToken string) (string, *apierror.APIError)
	Revoke(ctx context.Context, refreshToken string) *apierror.APIError
	RevokeAll(ctx context.Context, userID string) *apierror.APIError
}

type RegisterUserInput struct {
	Name           string
	Email          string
	HashedPassword string
}

type UserMed interface {
	GenAuthAccessToken(ctx context.Context, userID string) (string, *apierror.APIError)
	GenPasswordResetAccessToken(ctx context.Context, userID string) (string, *apierror.APIError)
	Register(ctx context.Context, input RegisterUserInput) (*types.User, *apierror.APIError)
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError)
}

type RequestIdentity struct {
	ActorID      string
	IdentityType types.IdentityType
}

type IdempotencyMed interface {
	// UpsertIdempotencyKey upserts the idempotency key for the request and returns the key.
	UpsertIdempotencyKey(ctx context.Context, identity *RequestIdentity) (*IdempotencyKey, *apierror.APIError)
	// CacheErrorResponse caches a non-transient error response and returns the original error.
	// If the error is transient, it returns the error without caching.
	CacheErrorResponse(ctx context.Context, typeID string, apiErr *apierror.APIError) *apierror.APIError
	// CacheSuccessResponse caches a successful response with HTTP 200 status.
	CacheSuccessResponse(ctx context.Context, typeID string, data any) *apierror.APIError
}
