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

// Validate validates a refresh token and returns the user ID if it is valid.
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

// TODO: We should hash rather than use opaque tokens.

// Create creates a new refresh token for the given user ID that will expire in the given number of days.
// If the number of days is not provided, it will default to 30 days.
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

// Revoke revokes a refresh token. This will prevent the refresh token from being used to mint
// a new access token.
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

// RevokeAll revokes all refresh tokens associated with a user. This will prevent stale tokens from
// being minted after a password change.
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
