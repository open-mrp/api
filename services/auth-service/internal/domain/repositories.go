package domain

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
)

type AccountRelationRepo interface {
	FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*AuthAccountRelation, *contracts.APIError)
	FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID, apiKeyID string) (*AuthAccountRelation, *contracts.APIError)
}

type AccountUserRepo interface {
	FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*AccountUser, *contracts.APIError)
	FindAccountAffiliationsByUserID(ctx context.Context, userID string) ([]AccountAffiliation, *contracts.APIError)
	FindLastUsedAccountIDByUserID(ctx context.Context, userID string) (string, *contracts.APIError)
	UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *contracts.APIError
}

type APIKeyRepo interface {
	Find(ctx context.Context, apiKeyID string) (*APIKey, *contracts.APIError)
	Touch(ctx context.Context, apiKeyID string) *contracts.APIError
}

type RefreshTokenRepo interface {
	Find(ctx context.Context, token string) (*RefreshToken, *contracts.APIError)
	Create(ctx context.Context, userID string, token string, expiresInDays int) (*RefreshToken, *contracts.APIError)
	Revoke(ctx context.Context, token string) *contracts.APIError
	RevokeAll(ctx context.Context, userID string) *contracts.APIError
}

type RolePermissionRepo interface {
	FindByRoleID(ctx context.Context, roleID string) (map[string]bool, *contracts.APIError)
}

type UserRepo interface {
	Find(ctx context.Context, identifier string) (*types.User, *contracts.APIError)
	Create(ctx context.Context, userID, email, name, hashedPassword string) (*types.User, *contracts.APIError)
	UpdatePassword(ctx context.Context, userID string, hashedPassword string) *contracts.APIError
}
