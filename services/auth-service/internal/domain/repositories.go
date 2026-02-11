package domain

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type APIKeyRepo interface {
	Find(ctx context.Context, apiKeyID string) (*APIKey, *apierror.APIError)
	FindByDatabaseID(ctx context.Context, id int64) (*APIKey, *apierror.APIError)
	Touch(ctx context.Context, apiKeyID int64) *apierror.APIError
	Create(ctx context.Context, apiKey *APIKey) (int64, *apierror.APIError)
	List(ctx context.Context, accountMode constants.AccountMode, ownerAccountID string, cursor *string, limit int32, query *string) ([]*APIKey, int64, *apierror.APIError)
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
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (string, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
