package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var apiKeyRepoTracer = tracing.GetTracer("auth-service.api_key_repository")

type apiKeyRepoImpl struct {
	db *sqlc.Queries
}

func NewAPIKeyRepo(db *sqlc.Queries) domain.APIKeyRepo {
	return &apiKeyRepoImpl{db: db}
}

func (r *apiKeyRepoImpl) Find(ctx context.Context, apiKeyID string) (*apikey.APIKey, *apierror.APIError) {
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

	return &apikey.APIKey{
		ID:             apiKeyRow.ID,
		TypeID:         apiKeyRow.TypeID,
		KeyID:          apiKeyRow.KeyID,
		Name:           apiKeyRow.Name.String,
		SecretHash:     apiKeyRow.SecretHash,
		RedactedValue:  apiKeyRow.RedactedValue,
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

func (r *apiKeyRepoImpl) Create(ctx context.Context, apiKey *apikey.APIKey) (int64, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.create")
	defer span.End()

	result, err := r.db.CreateAPIKey(ctx, sqlc.CreateAPIKeyParams{
		TypeID:         apiKey.TypeID,
		KeyID:          apiKey.KeyID,
		Name:           db.NullString(apiKey.Name),
		SecretHash:     apiKey.SecretHash,
		RedactedValue:  apiKey.RedactedValue,
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

func (r *apiKeyRepoImpl) List(ctx context.Context, input domain.APIKeyListRepoInput) (*domain.APIKeyListRepoResult, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.list")
	defer span.End()

	searchQuery := ""
	if input.Query != nil {
		searchQuery = *input.Query
	}

	includeActive := false
	includeExpired := false
	includeRevoked := false
	for _, s := range input.Statuses {
		switch s {
		case constants.APIKeyStatusActive:
			includeActive = true
		case constants.APIKeyStatusExpired:
			includeExpired = true
		case constants.APIKeyStatusRevoked:
			includeRevoked = true
		}
	}

	var cursorDir *pagination.Direction

	if input.Cursor != nil {
		cur, err := pagination.DecodeCursor(*input.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.db.ListAPIKeysBackward(ctx, sqlc.ListAPIKeysBackwardParams{
				OwnerAccountID:  input.OwnerAccountID,
				Query:           searchQuery,
				IncludeActive:   includeActive,
				IncludeExpired:  includeExpired,
				IncludeRevoked:  includeRevoked,
				CursorCreatedAt: cur.CreatedAt,
				CursorID:        cur.ID,
				Limit:           input.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			apiKeys := mapBackwardAPIKeyRows(rows)
			result, pageInfo := pagination.BuildPage(apiKeys, input.Limit, cursorDir, apiKeyCreatedAt, apiKeyID)
			return &domain.APIKeyListRepoResult{APIKeys: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.db.ListAPIKeysForward(ctx, sqlc.ListAPIKeysForwardParams{
			OwnerAccountID:  input.OwnerAccountID,
			Query:           searchQuery,
			IncludeActive:   includeActive,
			IncludeExpired:  includeExpired,
			IncludeRevoked:  includeRevoked,
			CursorCreatedAt: gosql.NullTime{Time: cur.CreatedAt, Valid: true},
			CursorID:        gosql.NullInt64{Int64: cur.ID, Valid: true},
			Limit:           input.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		apiKeys := mapForwardAPIKeyRows(rows)
		result, pageInfo := pagination.BuildPage(apiKeys, input.Limit, cursorDir, apiKeyCreatedAt, apiKeyID)
		return &domain.APIKeyListRepoResult{APIKeys: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.db.ListAPIKeysForward(ctx, sqlc.ListAPIKeysForwardParams{
		OwnerAccountID: input.OwnerAccountID,
		Query:          searchQuery,
		IncludeActive:  includeActive,
		IncludeExpired: includeExpired,
		IncludeRevoked: includeRevoked,
		Limit:          input.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiKeys := mapForwardAPIKeyRows(rows)
	result, pageInfo := pagination.BuildPage(apiKeys, input.Limit, cursorDir, apiKeyCreatedAt, apiKeyID)
	return &domain.APIKeyListRepoResult{APIKeys: result, PageInfo: pageInfo}, nil
}

func apiKeyCreatedAt(k *apikey.APIKey) time.Time { return k.CreatedAt }
func apiKeyID(k *apikey.APIKey) int64            { return k.ID }

func mapSingleAPIKeyRow(id int64, typeID, keyID string, name gosql.NullString, secretHash []byte, redactedValue, ownerAccountID, roleID string, roleName, roleTypeCode gosql.NullString, createdAt, updatedAt time.Time, lastUsedAt, expiresAt, revokedAt gosql.NullTime) *apikey.APIKey {
	rn := ""
	if roleName.Valid {
		rn = roleName.String
	}
	rtc := ""
	if roleTypeCode.Valid {
		rtc = roleTypeCode.String
	}
	return &apikey.APIKey{
		ID:             id,
		TypeID:         typeID,
		KeyID:          keyID,
		Name:           name.String,
		SecretHash:     secretHash,
		RedactedValue:  redactedValue,
		OwnerAccountID: ownerAccountID,
		RoleID:         roleID,
		RoleName:       rn,
		RoleTypeCode:   rtc,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		LastUsedAt:     db.TimeFromNullTime(lastUsedAt),
		ExpiresAt:      db.TimeFromNullTime(expiresAt),
		RevokedAt:      db.TimeFromNullTime(revokedAt),
	}
}

func mapForwardAPIKeyRows(rows []sqlc.ListAPIKeysForwardRow) []*apikey.APIKey {
	keys := make([]*apikey.APIKey, len(rows))
	for i, r := range rows {
		keys[i] = mapSingleAPIKeyRow(r.ID, r.TypeID, r.KeyID, r.Name, r.SecretHash, r.RedactedValue, r.OwnerAccountID, r.RoleID, r.RoleName, r.RoleTypeCode, r.CreatedAt, r.UpdatedAt, r.LastUsedAt, r.ExpiresAt, r.RevokedAt)
	}
	return keys
}

func mapBackwardAPIKeyRows(rows []sqlc.ListAPIKeysBackwardRow) []*apikey.APIKey {
	keys := make([]*apikey.APIKey, len(rows))
	for i, r := range rows {
		keys[i] = mapSingleAPIKeyRow(r.ID, r.TypeID, r.KeyID, r.Name, r.SecretHash, r.RedactedValue, r.OwnerAccountID, r.RoleID, r.RoleName, r.RoleTypeCode, r.CreatedAt, r.UpdatedAt, r.LastUsedAt, r.ExpiresAt, r.RevokedAt)
	}
	return keys
}

func (r *apiKeyRepoImpl) FindByTypeID(ctx context.Context, typeID string) (*apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.find_by_type_id")
	defer span.End()

	apiKeyRow, err := r.db.FindAPIKeyWithRoleByTypeID(ctx, typeID)

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

	return &apikey.APIKey{
		ID:             apiKeyRow.ID,
		TypeID:         apiKeyRow.TypeID,
		KeyID:          apiKeyRow.KeyID,
		Name:           apiKeyRow.Name.String,
		SecretHash:     apiKeyRow.SecretHash,
		RedactedValue:  apiKeyRow.RedactedValue,
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

func (r *apiKeyRepoImpl) Revoke(ctx context.Context, typeID string) *apierror.APIError {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.revoke")
	defer span.End()

	err := r.db.RevokeAPIKeyByTypeID(ctx, typeID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *apiKeyRepoImpl) FindByDatabaseID(ctx context.Context, id int64) (*apikey.APIKey, *apierror.APIError) {
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

	return &apikey.APIKey{
		ID:             apiKeyRow.ID,
		TypeID:         apiKeyRow.TypeID,
		KeyID:          apiKeyRow.KeyID,
		Name:           apiKeyRow.Name.String,
		SecretHash:     apiKeyRow.SecretHash,
		RedactedValue:  apiKeyRow.RedactedValue,
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
