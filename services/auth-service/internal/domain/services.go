package domain

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/contracts"
)

type LoginResult struct {
	User         *types.User
	RefreshToken string
	AccessToken  string
}

type RefreshTokenResult struct {
	AccessToken string
}

type AuthSvc interface {
	Login(ctx context.Context, identifier, password string) (*LoginResult, *contracts.APIError)
	Register(ctx context.Context, name, email, password string) (*LoginResult, *contracts.APIError)
	ValidateCredential(ctx context.Context, authToken string, targetAccountID string) (*types.Identity, *contracts.APIError)
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResult, *contracts.APIError)
	RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) *contracts.APIError
	ResetPassword(ctx context.Context, token, newPassword string) (*LoginResult, *contracts.APIError)
	RevokeRefreshToken(ctx context.Context, refreshToken string) *contracts.APIError
	UpdatePassword(ctx context.Context, userID string, oldPassword, newPassword string) *contracts.APIError
}
