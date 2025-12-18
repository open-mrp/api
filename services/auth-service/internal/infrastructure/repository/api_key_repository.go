package repository

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

var apiKeyRepoTracer = tracing.GetTracer("auth-service.api_key_repository")

type apiKeyRepoImpl struct {
	db *sqlc.Queries
}

func NewAPIKeyRepo(db *sqlc.Queries) domain.APIKeyRepo {
	return &apiKeyRepoImpl{db: db}
}

func (r *apiKeyRepoImpl) Find(ctx context.Context, apiKeyID string) (*domain.APIKey, *contracts.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.find")
	defer span.End()

	apiKeyRow, err := r.db.FindAPIKeyWithRoleByID(ctx, apiKeyID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == contracts.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	roleTypeCode := ""
	if apiKeyRow.RoleTypeCode.Valid {
		roleTypeCode = apiKeyRow.RoleTypeCode.String
	}

	return &domain.APIKey{
		ID:             apiKeyRow.ID,
		Name:           apiKeyRow.Name.String,
		LastFour:       apiKeyRow.LastFour,
		SecretHash:     apiKeyRow.SecretHash,
		OwnerAccountID: apiKeyRow.OwnerAccountID,
		RoleID:         apiKeyRow.RoleID,
		RoleTypeCode:   roleTypeCode,
		CreatedAt:      apiKeyRow.CreatedAt,
		UpdatedAt:      apiKeyRow.UpdatedAt,
		LastUsedAt:     ptrutil.NullTimeToPtr(apiKeyRow.LastUsedAt),
		ExpiresAt:      ptrutil.NullTimeToPtr(apiKeyRow.ExpiresAt),
		RevokedAt:      ptrutil.NullTimeToPtr(apiKeyRow.RevokedAt),
	}, nil
}

func (r *apiKeyRepoImpl) Touch(ctx context.Context, apiKeyID string) *contracts.APIError {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.touch")
	defer span.End()

	err := r.db.TouchAPIKey(ctx, apiKeyID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
