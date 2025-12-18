package mediator

import (
	"context"
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/token"
	"github.com/augno/api/shared/contracts"
	tracing "github.com/augno/api/shared/tracing"
)

var refreshTokenMedTracer = tracing.GetTracer("auth-service.refresh_token_mediator")

type refreshTokenMedImpl struct {
	repos            domain.RepoFactory
	jwtUtils         domain.JWTUtils
	opaqueTokenUtils domain.OpaqueTokenUtils
}

type RefreshTokenMedConfig struct {
	Repos            domain.RepoFactory
	JWTUtils         domain.JWTUtils
	OpaqueTokenUtils domain.OpaqueTokenUtils
}

func NewRefreshTokenMed(config RefreshTokenMedConfig) domain.RefreshTokenMed {
	return &refreshTokenMedImpl{
		repos:            config.Repos,
		jwtUtils:         config.JWTUtils,
		opaqueTokenUtils: config.OpaqueTokenUtils,
	}
}

func DefaultRefreshTokenMedConfig(queries *sqlc.Queries, jwtSecret string) RefreshTokenMedConfig {
	return RefreshTokenMedConfig{
		Repos:            repository.NewRepoFactory(queries),
		JWTUtils:         token.NewJWTUtils(token.DefaultJWTConfig(jwtSecret)),
		OpaqueTokenUtils: token.NewOpaqueTokenUtils(token.DefaultOpaqueTokenConfig()),
	}
}

func NewDefaultRefreshTokenMed(queries *sqlc.Queries, jwtSecret string) domain.RefreshTokenMed {
	return NewRefreshTokenMed(DefaultRefreshTokenMedConfig(queries, jwtSecret))
}

// Validate validates a refresh token and returns the user ID if it is valid.
func (s *refreshTokenMedImpl) Validate(ctx context.Context, refreshToken string) (userID string, err *contracts.APIError) {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.validate")
	defer span.End()

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	refreshTokenModel, err := refreshTokenRepo.Find(ctx, refreshToken)
	if err != nil || refreshTokenModel == nil {
		return "", tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidRefreshToken))
	}

	if refreshTokenModel.RevokedAt != nil && refreshTokenModel.RevokedAt.Before(time.Now().UTC()) {
		return "", tracing.Trace(span, contracts.NewAuthenticationError(ErrRefreshTokenRevoked))
	}

	if refreshTokenModel.ExpiresAt.Before(time.Now().UTC()) {
		return "", tracing.Trace(span, contracts.NewAuthenticationError(ErrExpiredRefreshToken))
	}

	return refreshTokenModel.UserID, nil
}

// Create creates a new refresh token for the given user ID that will expire in the given number of days.
// If the number of days is not provided, it will default to 30 days.
func (s *refreshTokenMedImpl) Create(ctx context.Context, userID string, expiresInDays *int) (*domain.RefreshToken, *contracts.APIError) {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.create")
	defer span.End()

	opaqueToken, err := s.opaqueTokenUtils.Gen(ctx)
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
func (s *refreshTokenMedImpl) Revoke(ctx context.Context, refreshToken string) *contracts.APIError {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.revoke")
	defer span.End()

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	refreshTokenModel, err := refreshTokenRepo.Find(ctx, refreshToken)
	if err != nil {
		return tracing.Trace(span, contracts.NewAuthenticationError(ErrInvalidRefreshToken))
	}

	if refreshTokenModel.RevokedAt != nil && refreshTokenModel.RevokedAt.Before(time.Now().UTC()) {
		return tracing.Trace(span, contracts.NewAuthenticationError(ErrRefreshTokenRevoked))
	}

	if refreshTokenModel.ExpiresAt.Before(time.Now().UTC()) {
		return tracing.Trace(span, contracts.NewAuthenticationError(ErrExpiredRefreshToken))
	}

	err = refreshTokenRepo.Revoke(ctx, refreshToken)
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to revoke refresh token."))
	}
	return nil
}

// RevokeAll revokes all refresh tokens associated with a user. This will prevent stale tokens from
// being minted after a password change.
func (s *refreshTokenMedImpl) RevokeAll(ctx context.Context, userID string) *contracts.APIError {
	ctx, span := refreshTokenMedTracer.Start(ctx, "mediator.refresh_token.revoke_all")
	defer span.End()

	refreshTokenRepo := s.repos.NewRefreshTokenRepo()
	err := refreshTokenRepo.RevokeAll(ctx, userID)
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to revoke all refresh tokens."))
	}
	return nil
}
