package domain

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
)

type AccountUserMed interface {
	MarkUsedIfNotRecent(ctx context.Context, accountUser *AccountUser) *contracts.APIError
}

type APIKeyMed interface {
	FindAndValidate(ctx context.Context, apiKey string) (*APIKey, *contracts.APIError)
	TouchIfNotRecent(ctx context.Context, apiKeyModel *APIKey) *contracts.APIError
}

type PasswordMed interface {
	RequestReset(ctx context.Context, identifier string, accountSlug *string) *contracts.APIError
	ValidatePasswordResetToken(ctx context.Context, token string) (*types.User, *contracts.APIError)
	Validate(ctx context.Context, identifier, password string) (*types.User, *contracts.APIError)
	Update(ctx context.Context, user *types.User, newPassword string) *contracts.APIError
}

type RefreshTokenMed interface {
	Create(ctx context.Context, userID string, expiresInDays *int) (*RefreshToken, *contracts.APIError)
	Validate(ctx context.Context, refreshToken string) (string, *contracts.APIError)
	Revoke(ctx context.Context, refreshToken string) *contracts.APIError
	RevokeAll(ctx context.Context, userID string) *contracts.APIError
}

type UserMed interface {
	GenAuthAccessToken(ctx context.Context, userID string) (string, *contracts.APIError)
	GenPasswordResetAccessToken(ctx context.Context, userID string) (string, *contracts.APIError)
	Register(ctx context.Context, name, email, hashedPassword string) (*types.User, *contracts.APIError)
	ValidateCredential(ctx context.Context, authToken string, targetAccountID string) (*types.Identity, *contracts.APIError)
}
