package repository

import (
	"context"
	gosql "database/sql"
	"strings"
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

	return mapWithRoleRow(apiKeyRow.ID, apiKeyRow.TypeID, apiKeyRow.KeyID, apiKeyRow.Name, apiKeyRow.SecretHash, apiKeyRow.RedactedValue, apiKeyRow.OwnerAccountID, apiKeyRow.RoleID, apiKeyRow.RoleName, apiKeyRow.RoleTypeCode, apiKeyRow.CreatedAt, apiKeyRow.UpdatedAt, apiKeyRow.LastUsedAt, apiKeyRow.ExpiresAt, apiKeyRow.RevokedAt), nil
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

func (r *apiKeyRepoImpl) CountRoleForOwner(ctx context.Context, roleID string, ownerAccountID string) (int64, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.count_role_for_owner")
	defer span.End()

	count, err := r.db.CountRoleForOwner(ctx, sqlc.CountRoleForOwnerParams{
		RoleID:         roleID,
		OwnerAccountID: gosql.NullString{String: ownerAccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
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

	useRole := includesContains(input.Includes, "role")

	var cursorDir *pagination.Direction

	if input.Cursor != nil {
		cur, err := pagination.DecodeCursor(*input.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			if useRole {
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
			rows, err := r.db.ListAPIKeysBaseBackward(ctx, sqlc.ListAPIKeysBaseBackwardParams{
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
			apiKeys := mapBaseForwardAPIKeyRows(rows)
			result, pageInfo := pagination.BuildPage(apiKeys, input.Limit, cursorDir, apiKeyCreatedAt, apiKeyID)
			return &domain.APIKeyListRepoResult{APIKeys: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		if useRole {
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
		rows, err := r.db.ListAPIKeysBaseForward(ctx, sqlc.ListAPIKeysBaseForwardParams{
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
		apiKeys := mapBaseForwardAPIKeyRows(rows)
		result, pageInfo := pagination.BuildPage(apiKeys, input.Limit, cursorDir, apiKeyCreatedAt, apiKeyID)
		return &domain.APIKeyListRepoResult{APIKeys: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	if useRole {
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
	rows, err := r.db.ListAPIKeysBaseForward(ctx, sqlc.ListAPIKeysBaseForwardParams{
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

	apiKeys := mapBaseForwardAPIKeyRows(rows)
	result, pageInfo := pagination.BuildPage(apiKeys, input.Limit, cursorDir, apiKeyCreatedAt, apiKeyID)
	return &domain.APIKeyListRepoResult{APIKeys: result, PageInfo: pageInfo}, nil
}

func apiKeyCreatedAt(k *apikey.APIKey) time.Time { return k.CreatedAt }
func apiKeyID(k *apikey.APIKey) int64            { return k.ID }

// includesContains returns true when the given key is in the includes list, or when any include is a nested path rooted at that key (e.g. "role.permissions" implies "role"). When includes is nil (no include param), returns false.
func includesContains(includes []string, key string) bool {
	if includes == nil {
		return false
	}
	prefix := key + "."
	for _, v := range includes {
		if v == key || strings.HasPrefix(v, prefix) {
			return true
		}
	}
	return false
}

func mapWithRoleRow(id int64, typeID, keyID string, name gosql.NullString, secretHash []byte, redactedValue, ownerAccountID, roleID string, roleName, roleTypeCode gosql.NullString, createdAt, updatedAt time.Time, lastUsedAt, expiresAt, revokedAt gosql.NullTime) *apikey.APIKey {
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
		RoleType:       rtc,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		LastUsedAt:     db.TimeFromNullTime(lastUsedAt),
		ExpiresAt:      db.TimeFromNullTime(expiresAt),
		RevokedAt:      db.TimeFromNullTime(revokedAt),
	}
}

func mapBaseRow(row *sqlc.ApiKey) *apikey.APIKey {
	return &apikey.APIKey{
		ID:             row.ID,
		TypeID:         row.TypeID,
		KeyID:          row.KeyID,
		Name:           row.Name.String,
		SecretHash:     row.SecretHash,
		RedactedValue:  row.RedactedValue,
		OwnerAccountID: row.OwnerAccountID,
		RoleID:         row.RoleID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		LastUsedAt:     db.TimeFromNullTime(row.LastUsedAt),
		ExpiresAt:      db.TimeFromNullTime(row.ExpiresAt),
		RevokedAt:      db.TimeFromNullTime(row.RevokedAt),
	}
}

func mapForwardAPIKeyRows(rows []sqlc.ListAPIKeysForwardRow) []*apikey.APIKey {
	keys := make([]*apikey.APIKey, len(rows))
	for i, r := range rows {
		keys[i] = mapWithRoleRow(r.ID, r.TypeID, r.KeyID, r.Name, r.SecretHash, r.RedactedValue, r.OwnerAccountID, r.RoleID, r.RoleName, r.RoleTypeCode, r.CreatedAt, r.UpdatedAt, r.LastUsedAt, r.ExpiresAt, r.RevokedAt)
	}
	return keys
}

func mapBackwardAPIKeyRows(rows []sqlc.ListAPIKeysBackwardRow) []*apikey.APIKey {
	keys := make([]*apikey.APIKey, len(rows))
	for i, r := range rows {
		keys[i] = mapWithRoleRow(r.ID, r.TypeID, r.KeyID, r.Name, r.SecretHash, r.RedactedValue, r.OwnerAccountID, r.RoleID, r.RoleName, r.RoleTypeCode, r.CreatedAt, r.UpdatedAt, r.LastUsedAt, r.ExpiresAt, r.RevokedAt)
	}
	return keys
}

func mapBaseForwardAPIKeyRows(rows []sqlc.ApiKey) []*apikey.APIKey {
	keys := make([]*apikey.APIKey, len(rows))
	for i := range rows {
		keys[i] = mapBaseRow(&rows[i])
	}
	return keys
}

func (r *apiKeyRepoImpl) FindByTypeID(ctx context.Context, typeID string, includes []string) (*apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.find_by_type_id")
	defer span.End()

	if includesContains(includes, "role") {
		apiKeyRow, err := r.db.FindAPIKeyWithRoleByTypeID(ctx, typeID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, apiErr
			}
			return nil, tracing.Trace(span, apiErr)
		}
		return mapWithRoleRow(apiKeyRow.ID, apiKeyRow.TypeID, apiKeyRow.KeyID, apiKeyRow.Name, apiKeyRow.SecretHash, apiKeyRow.RedactedValue, apiKeyRow.OwnerAccountID, apiKeyRow.RoleID, apiKeyRow.RoleName, apiKeyRow.RoleTypeCode, apiKeyRow.CreatedAt, apiKeyRow.UpdatedAt, apiKeyRow.LastUsedAt, apiKeyRow.ExpiresAt, apiKeyRow.RevokedAt), nil
	}

	row, err := r.db.FindAPIKeyBaseByTypeID(ctx, typeID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return mapBaseRow(&row), nil
}

func (r *apiKeyRepoImpl) GetByIDs(ctx context.Context, ownerAccountID string, ids []string) ([]*apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.get_by_ids")
	defer span.End()

	rows, err := r.db.GetAPIKeysByIDs(ctx, sqlc.GetAPIKeysByIDsParams{
		Ids:            ids,
		OwnerAccountID: ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	keys := make([]*apikey.APIKey, len(rows))
	for i, row := range rows {
		keys[i] = mapBaseRow(&row)
	}
	return keys, nil
}

// Revoke revokes an API key. A nil revokeAt revokes immediately using the database clock — a service-supplied "now" can sit ahead of the database clock and would briefly count as a scheduled (still-active) revocation. A non-nil revokeAt schedules a future revocation.
func (r *apiKeyRepoImpl) Revoke(ctx context.Context, typeID string, ownerAccountID string, revokeAt *time.Time) *apierror.APIError {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.revoke")
	defer span.End()

	result, err := r.db.RevokeAPIKeyByTypeID(ctx, sqlc.RevokeAPIKeyByTypeIDParams{
		RevokedAt:      db.NullTimePtr(revokeAt),
		TypeID:         typeID,
		OwnerAccountID: ownerAccountID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "failed to read rows affected"))
	}
	if rows == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("API key not found."))
	}

	return nil
}

func (r *apiKeyRepoImpl) FindByDatabaseID(ctx context.Context, id int64, includes []string) (*apikey.APIKey, *apierror.APIError) {
	ctx, span := apiKeyRepoTracer.Start(ctx, "repository.api_key.find_by_database_id")
	defer span.End()

	if includesContains(includes, "role") {
		apiKeyRow, err := r.db.FindAPIKeyWithRoleByDatabaseID(ctx, id)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, apiErr
			}
			return nil, tracing.Trace(span, apiErr)
		}
		return mapWithRoleRow(apiKeyRow.ID, apiKeyRow.TypeID, apiKeyRow.KeyID, apiKeyRow.Name, apiKeyRow.SecretHash, apiKeyRow.RedactedValue, apiKeyRow.OwnerAccountID, apiKeyRow.RoleID, apiKeyRow.RoleName, apiKeyRow.RoleTypeCode, apiKeyRow.CreatedAt, apiKeyRow.UpdatedAt, apiKeyRow.LastUsedAt, apiKeyRow.ExpiresAt, apiKeyRow.RevokedAt), nil
	}

	row, err := r.db.FindAPIKeyBaseByDatabaseID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return mapBaseRow(&row), nil
}
