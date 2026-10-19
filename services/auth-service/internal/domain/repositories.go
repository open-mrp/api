package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/apikey"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
)

type APIKeyListRepoInput struct {
	OwnerAccountID string
	Cursor         *string
	Limit          int32
	Query          *string
	Statuses       []constants.APIKeyStatus
	Includes       []string
}

type APIKeyListRepoResult struct {
	APIKeys  []*apikey.APIKey
	PageInfo pagination.PageInfo
}

type APIKeyRepo interface {
	Find(ctx context.Context, apiKeyID string) (*apikey.APIKey, *apierror.APIError)
	FindByDatabaseID(ctx context.Context, id int64, includes []string) (*apikey.APIKey, *apierror.APIError)
	FindByTypeID(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError)
	Touch(ctx context.Context, apiKeyID int64) *apierror.APIError
	Create(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError)
	// CountRoleForOwner returns the number of roles matching roleID that are visible to ownerAccountID (system roles with a NULL account_id, or roles owned by the account). Used to validate a referenced role_id before persisting an API key.
	CountRoleForOwner(ctx context.Context, roleID string, ownerAccountID string) (int64, *apierror.APIError)
	GetByIDs(ctx context.Context, ownerAccountID string, ids []string) ([]*apikey.APIKey, *apierror.APIError)
	// Revoke marks an API key as revoked. A nil revokeAt revokes immediately using the database clock; a non-nil revokeAt schedules a future revocation. Scoped to ownerAccountID; returns a not-found error if the key does not exist for the given owner.
	Revoke(ctx context.Context, typeID string, ownerAccountID string, revokeAt *time.Time) *apierror.APIError
	List(ctx context.Context, input APIKeyListRepoInput) (*APIKeyListRepoResult, *apierror.APIError)
}

type RefreshTokenRepo interface {
	Find(ctx context.Context, token string) (*RefreshToken, *apierror.APIError)
	Create(ctx context.Context, userID string, token string, expiresInDays int) (*RefreshToken, *apierror.APIError)
	Revoke(ctx context.Context, token string) *apierror.APIError
	RevokeAll(ctx context.Context, userID string) *apierror.APIError
}

type UserRepo interface {
	Find(ctx context.Context, identifier string) (*types.User, *apierror.APIError)
	Create(ctx context.Context, userID, email, name, hashedPassword string) (*types.User, *apierror.APIError)
	UpdatePassword(ctx context.Context, userID string, hashedPassword string) *apierror.APIError
	// FindLastUsedAccountID returns the active account the user last used, or nil when they belong to none.
	FindLastUsedAccountID(ctx context.Context, userID string) (*string, *apierror.APIError)
}

type DocAPIKeyRepo interface {
	FindBySandboxAccountID(ctx context.Context, sandboxAccountID string) (*apikey.DocAPIKey, *apierror.APIError)
	FindByAPIKeyID(ctx context.Context, apiKeyID string) (*apikey.DocAPIKey, *apierror.APIError)
	Create(ctx context.Context, docAPIKey *apikey.DocAPIKey) (int64, *apierror.APIError)
	Update(ctx context.Context, docAPIKey *apikey.DocAPIKey) *apierror.APIError
	Delete(ctx context.Context, id int64) *apierror.APIError
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
