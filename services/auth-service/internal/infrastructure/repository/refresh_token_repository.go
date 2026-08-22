package repository

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var refreshTokenRepoTracer = tracing.GetTracer("auth-service.refresh_token_repository")

type refreshTokenRepoImpl struct {
	db *sqlc.Queries
}

func NewRefreshTokenRepo(db *sqlc.Queries) domain.RefreshTokenRepo {
	return &refreshTokenRepoImpl{db: db}
}

func (r *refreshTokenRepoImpl) Find(ctx context.Context, token string) (*domain.RefreshToken, *apierror.APIError) {
	ctx, span := refreshTokenRepoTracer.Start(ctx, "repository.refresh_token.find")
	defer span.End()

	refreshTokenModel, err := r.db.FindRefreshToken(ctx, token)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.RefreshToken{
		Token:     token,
		UserID:    refreshTokenModel.UserID,
		ExpiresAt: refreshTokenModel.ExpiresAt,
		RevokedAt: db.TimeFromNullTime(refreshTokenModel.RevokedAt),
	}, nil
}

func (r *refreshTokenRepoImpl) Create(ctx context.Context, userID string, token string, expiresInDays int) (*domain.RefreshToken, *apierror.APIError) {
	ctx, span := refreshTokenRepoTracer.Start(ctx, "repository.refresh_token.create")
	defer span.End()

	expiresAt := time.Now().UTC().AddDate(0, 0, expiresInDays)
	err := r.db.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.RefreshToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
		RevokedAt: nil,
	}, nil
}

func (r *refreshTokenRepoImpl) Revoke(ctx context.Context, token string) *apierror.APIError {
	ctx, span := refreshTokenRepoTracer.Start(ctx, "repository.refresh_token.revoke")
	defer span.End()

	err := r.db.RevokeRefreshToken(ctx, token)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *refreshTokenRepoImpl) RevokeAll(ctx context.Context, userID string) *apierror.APIError {
	ctx, span := refreshTokenRepoTracer.Start(ctx, "repository.refresh_token.revoke_all")
	defer span.End()

	err := r.db.RevokeAllRefreshTokensByUserID(ctx, userID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
