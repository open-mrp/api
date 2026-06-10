package mediator

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/token"
	apierror "github.com/augno/api/shared/errors"
	tracing "github.com/augno/api/shared/tracing"
)

var refreshTokenMedTracer = tracing.GetTracer("auth-service.refresh_token_mediator")

type refreshTokenMedImpl struct {
	repos domain.RepoFactory
}

type RefreshTokenMedConfig struct {
	// Repos (required) is the repository factory for refresh token persistence.
	Repos domain.RepoFactory
}

func (c *RefreshTokenMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("refresh token mediator: repos is required")
	}
	return nil
}

func NewRefreshTokenMed(config *RefreshTokenMedConfig) domain.RefreshTokenMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &refreshTokenMedImpl{
		repos: config.Repos,
	}
}

// Validate validates a refresh token and returns the associated user ID.
//
// 1. Look up the refresh token in the repository.
// 2. Verify the token is not revoked.
// 3. Verify the token is not expired.
// 4. Return the associated user ID.
func (s *refreshTokenMedImpl) Validate(ctx context.Context, refreshToken string) (userID string, err *apierror.APIError) {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.validate")
	defer span.End()

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	refreshTokenModel, err := refreshTokenRepo.Find(ctx, refreshToken)
	if err != nil {
		if apierror.IsNotFound(err) {
			return "", tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidRefreshToken))
		}
		return "", tracing.Trace(span, err)
	}

	if refreshTokenModel.IsRevoked() {
		return "", tracing.Trace(span, apierror.NewAuthenticationError(ErrRefreshTokenRevoked))
	}

	if refreshTokenModel.IsExpired() {
		return "", tracing.Trace(span, apierror.NewAuthenticationError(ErrExpiredRefreshToken))
	}

	return refreshTokenModel.UserID, nil
}

// ! NOTE: We should hash rather than use opaque tokens.

// Create issues a new refresh token for the given user ID.
//
// 1. Generate a cryptographically random opaque token.
// 2. Default expiration to 30 days if expiresInDays is nil.
// 3. Persist the token in the repository with the computed expiration.
func (s *refreshTokenMedImpl) Create(ctx context.Context, userID string, expiresInDays *int) (*domain.RefreshToken, *apierror.APIError) {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.create")
	defer span.End()

	opaqueToken, err := token.GenOpaqueToken(ctx)
	if err != nil {
		return nil, err
	}

	expirationDays := 30
	if expiresInDays != nil {
		expirationDays = *expiresInDays
	}

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	refreshTokenModel, err := refreshTokenRepo.Create(ctx, userID, opaqueToken, expirationDays)
	if err != nil {
		return nil, err
	}

	return refreshTokenModel, nil
}

// Revoke revokes a single refresh token, preventing it from being used to mint new access tokens.
//
// 1. Look up the refresh token in the repository.
// 2. Verify the token is not already revoked or expired.
// 3. Mark the token as revoked in the repository.
func (s *refreshTokenMedImpl) Revoke(ctx context.Context, refreshToken string) *apierror.APIError {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.revoke")
	defer span.End()

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	refreshTokenModel, err := refreshTokenRepo.Find(ctx, refreshToken)
	if err != nil {
		if apierror.IsNotFound(err) {
			return tracing.Trace(span, apierror.NewAuthenticationError(ErrInvalidRefreshToken))
		}
		return tracing.Trace(span, err)
	}

	if refreshTokenModel.IsRevoked() {
		return tracing.Trace(span, apierror.NewAuthenticationError(ErrRefreshTokenRevoked))
	}

	if refreshTokenModel.IsExpired() {
		return tracing.Trace(span, apierror.NewAuthenticationError(ErrExpiredRefreshToken))
	}

	err = refreshTokenRepo.Revoke(ctx, refreshToken)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to revoke refresh token."))
	}
	return nil
}

// RevokeAll revokes all refresh tokens associated with a user.
//
// 1. Revoke all refresh tokens for the given user ID in the repository.
//
// Behavior:
//   - Prevents stale tokens from being used after a password change.
func (s *refreshTokenMedImpl) RevokeAll(ctx context.Context, userID string) *apierror.APIError {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.revoke_all")
	defer span.End()

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	err := refreshTokenRepo.RevokeAll(ctx, userID)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to revoke all refresh tokens."))
	}
	return nil
}
