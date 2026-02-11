package domain

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
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
	ValidateCredential(ctx context.Context, authToken string, targetAccountID *string) (*types.Identity, *apierror.APIError)
}

type TokenSvc interface {
	RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenResult, *apierror.APIError)
	RevokeRefreshToken(ctx context.Context, refreshToken string) *apierror.APIError
}

type UserSvc interface {
	Login(ctx context.Context, identifier, password string) (*LoginResult, *apierror.APIError)
	Register(ctx context.Context, input RegisterInput) (*LoginResult, *apierror.APIError)
}

type PasswordSvc interface {
	UpdatePassword(ctx context.Context, userID string, oldPassword, newPassword string) *apierror.APIError
	ResetPassword(ctx context.Context, token, newPassword string) (*LoginResult, *apierror.APIError)
	RequestPasswordReset(ctx context.Context, identifier string, accountSlug *string) *apierror.APIError
}
