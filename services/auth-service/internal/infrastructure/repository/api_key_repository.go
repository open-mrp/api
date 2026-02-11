package repository

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var apiKeyRepoTracer = tracing.GetTracer("auth-service.api_key_repository")

type apiKeyRepoImpl struct {
	db *sqlc.Queries
}

func NewAPIKeyRepo(db *sqlc.Queries) domain.APIKeyRepo {
	return &apiKeyRepoImpl{db: db}
}

func (r *apiKeyRepoImpl) Find(ctx context.Context, apiKeyID string) (*domain.APIKey, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.find")
	defer span.End()

	apiKeyRow, err := r.db.FindAPIKeyWithRoleByKeyID(ctx, apiKeyID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	roleTypeCode := ""
	if apiKeyRow.RoleTypeCode.Valid {
		roleTypeCode = apiKeyRow.RoleTypeCode.String
	}

	roleName := ""
	if apiKeyRow.RoleName.Valid {
		roleName = apiKeyRow.RoleName.String
	}

	return &domain.APIKey{
		ID:             apiKeyRow.ID,
		TypeID:         apiKeyRow.TypeID,
		KeyID:          apiKeyRow.KeyID,
		Name:           apiKeyRow.Name.String,
		LastFour:       apiKeyRow.LastFour,
		SecretHash:     apiKeyRow.SecretHash,
		OwnerAccountID: apiKeyRow.OwnerAccountID,
		RoleID:         apiKeyRow.RoleID,
		RoleName:       roleName,
		RoleTypeCode:   roleTypeCode,
		CreatedAt:      apiKeyRow.CreatedAt,
		UpdatedAt:      apiKeyRow.UpdatedAt,
		LastUsedAt:     db.TimeFromNullTime(apiKeyRow.LastUsedAt),
		ExpiresAt:      db.TimeFromNullTime(apiKeyRow.ExpiresAt),
		RevokedAt:      db.TimeFromNullTime(apiKeyRow.RevokedAt),
	}, nil
}

func (r *apiKeyRepoImpl) Touch(ctx context.Context, apiKeyID int64) *apierror.APIError {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.touch")
	defer span.End()

	err := r.db.TouchAPIKeyByID(ctx, apiKeyID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *apiKeyRepoImpl) Create(ctx context.Context, apiKey *domain.APIKey) (int64, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.create")
	defer span.End()

	result, err := r.db.CreateAPIKey(ctx, sqlc.CreateAPIKeyParams{
		TypeID:         apiKey.TypeID,
		KeyID:          apiKey.KeyID,
		Name:           db.NullString(apiKey.Name),
		SecretHash:     apiKey.SecretHash,
		LastFour:       apiKey.LastFour,
		OwnerAccountID: apiKey.OwnerAccountID,
		RoleID:         apiKey.RoleID,
		ExpiresAt:      db.NullTimePtr(apiKey.ExpiresAt),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "failed to get last insert id"))
	}

	return id, nil
}

func (r *apiKeyRepoImpl) List(ctx context.Context, accountMode constants.AccountMode, ownerAccountID string, cursor *string, limit int32, query *string) ([]*domain.APIKey, int64, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.list")
	defer span.End()

	searchQuery := ""
	if query != nil {
		searchQuery = *query
	}

	cursorVal := ""
	if cursor != nil {
		cursorVal = *cursor
	}

	rows, err := r.db.ListAPIKeys(ctx, sqlc.ListAPIKeysParams{
		OwnerAccountID: ownerAccountID,
		Cursor:         cursorVal,
		Query:          searchQuery,
		Limit:          limit,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, 0, tracing.Trace(span, apiErr)
	}

	total, err := r.db.CountAPIKeys(ctx, sqlc.CountAPIKeysParams{
		OwnerAccountID: ownerAccountID,
		Query:          searchQuery,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, 0, tracing.Trace(span, apiErr)
	}

	apiKeys := make([]*domain.APIKey, len(rows))
	for i, row := range rows {
		roleName := ""
		if row.RoleName.Valid {
			roleName = row.RoleName.String
		}
		roleTypeCode := ""
		if row.RoleTypeCode.Valid {
			roleTypeCode = row.RoleTypeCode.String
		}

		apiKeys[i] = &domain.APIKey{
			ID:             row.ID,
			TypeID:         row.TypeID,
			KeyID:          row.KeyID,
			Name:           row.Name.String,
			LastFour:       row.LastFour,
			SecretHash:     row.SecretHash,
			OwnerAccountID: row.OwnerAccountID,
			RoleID:         row.RoleID,
			RoleName:       roleName,
			RoleTypeCode:   roleTypeCode,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
			LastUsedAt:     db.TimeFromNullTime(row.LastUsedAt),
			ExpiresAt:      db.TimeFromNullTime(row.ExpiresAt),
			RevokedAt:      db.TimeFromNullTime(row.RevokedAt),
		}

		apiKeys[i].RedactedValue = apikey.RedactAPIKeyValue(apiKeys[i], accountMode)
	}

	return apiKeys, total, nil
}

func (r *apiKeyRepoImpl) FindByDatabaseID(ctx context.Context, id int64) (*domain.APIKey, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.find_by_database_id")
	defer span.End()

	apiKeyRow, err := r.db.FindAPIKeyWithRoleByDatabaseID(ctx, id)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	roleTypeCode := ""
	if apiKeyRow.RoleTypeCode.Valid {
		roleTypeCode = apiKeyRow.RoleTypeCode.String
	}

	roleName := ""
	if apiKeyRow.RoleName.Valid {
		roleName = apiKeyRow.RoleName.String
	}

	return &domain.APIKey{
		ID:             apiKeyRow.ID,
		TypeID:         apiKeyRow.TypeID,
		KeyID:          apiKeyRow.KeyID,
		Name:           apiKeyRow.Name.String,
		LastFour:       apiKeyRow.LastFour,
		SecretHash:     apiKeyRow.SecretHash,
		OwnerAccountID: apiKeyRow.OwnerAccountID,
		RoleID:         apiKeyRow.RoleID,
		RoleName:       roleName,
		RoleTypeCode:   roleTypeCode,
		CreatedAt:      apiKeyRow.CreatedAt,
		UpdatedAt:      apiKeyRow.UpdatedAt,
		LastUsedAt:     db.TimeFromNullTime(apiKeyRow.LastUsedAt),
		ExpiresAt:      db.TimeFromNullTime(apiKeyRow.ExpiresAt),
		RevokedAt:      db.TimeFromNullTime(apiKeyRow.RevokedAt),
	}, nil
}
