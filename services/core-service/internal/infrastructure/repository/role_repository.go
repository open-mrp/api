package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var roleRepoTracer = tracing.GetTracer("core-service.role_repository")

type roleRepoImpl struct {
	queries *sqlc.Queries
}

func NewRoleRepo(queries *sqlc.Queries) domain.RoleRepo {
	return &roleRepoImpl{queries: queries}
}

func roleCreatedAt(r *domain.Role) time.Time { return r.CreatedAt }
func roleID(r *domain.Role) string           { return r.ID }

func mapRoleRow(row sqlc.Role) *domain.Role {
	var acctID *string
	if row.AccountID.Valid {
		acctID = &row.AccountID.String
	}
	return &domain.Role{
		ID:        row.ID,
		Name:      row.Name,
		RoleType:  row.RoleTypeCode,
		AccountID: acctID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func buildRoleSearchQuery(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	sanitized := db.SanitizeFulltextBoolean(*query)
	if sanitized == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: sanitized + "*", Valid: true}
}

func (r *roleRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Role, *apierror.APIError) {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetRolesFullByIDs(ctx, sqlc.GetRolesFullByIDsParams{
		Ids:       ids,
		AccountID: db.NullString(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	roles := make([]*domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = mapRoleRow(row)
	}
	return roles, nil
}

func (r *roleRepoImpl) GetByID(ctx context.Context, roleID string) (*domain.RoleInfo, *apierror.APIError) {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.get_by_id")
	defer span.End()

	row, err := r.queries.GetRoleByID(ctx, roleID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.RoleInfo{
		ID:       row.ID,
		Name:     row.Name,
		RoleType: row.RoleTypeCode,
	}, nil
}

func (r *roleRepoImpl) FindByTypeCode(ctx context.Context, typeCode string, accountID string) (*domain.RoleInfo, *apierror.APIError) {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.find_by_type_code")
	defer span.End()

	row, err := r.queries.FindRoleByTypeCode(ctx, sqlc.FindRoleByTypeCodeParams{
		RoleTypeCode: typeCode,
		AccountID:    db.NullString(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.RoleInfo{
		ID:       row.ID,
		Name:     row.Name,
		RoleType: row.RoleTypeCode,
	}, nil
}

func (r *roleRepoImpl) List(ctx context.Context, params domain.ListRolesParams) (*domain.ListRolesPage, *apierror.APIError) {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.list")
	defer span.End()

	searchQuery := buildRoleSearchQuery(params.Query)
	includeRoleTypeFilter := len(params.RoleTypes) > 0
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListRolesBackward(ctx, sqlc.ListRolesBackwardParams{
				AccountID:             db.NullString(params.AccountID),
				SearchQuery:           searchQuery,
				IncludeRoleTypeFilter: includeRoleTypeFilter,
				RoleTypeCodes:         params.RoleTypes,
				CursorCreatedAt:       cur.OccurredAt,
				CursorID:              cur.ID,
				Limit:                 params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			roles := make([]*domain.Role, len(rows))
			for i, row := range rows {
				roles[i] = mapRoleRow(row)
			}
			result, pageInfo := pagination.BuildPageString(roles, params.Limit, cursorDir, roleCreatedAt, roleID)
			return &domain.ListRolesPage{Roles: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListRolesForward(ctx, sqlc.ListRolesForwardParams{
			AccountID:             db.NullString(params.AccountID),
			SearchQuery:           searchQuery,
			IncludeRoleTypeFilter: includeRoleTypeFilter,
			RoleTypeCodes:         params.RoleTypes,
			CursorCreatedAt:       gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:              gosql.NullString{String: cur.ID, Valid: true},
			Limit:                 params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		roles := make([]*domain.Role, len(rows))
		for i, row := range rows {
			roles[i] = mapRoleRow(row)
		}
		result, pageInfo := pagination.BuildPageString(roles, params.Limit, cursorDir, roleCreatedAt, roleID)
		return &domain.ListRolesPage{Roles: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListRolesForward(ctx, sqlc.ListRolesForwardParams{
		AccountID:             db.NullString(params.AccountID),
		SearchQuery:           searchQuery,
		IncludeRoleTypeFilter: includeRoleTypeFilter,
		RoleTypeCodes:         params.RoleTypes,
		Limit:                 params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	roles := make([]*domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = mapRoleRow(row)
	}
	result, pageInfo := pagination.BuildPageString(roles, params.Limit, cursorDir, roleCreatedAt, roleID)
	return &domain.ListRolesPage{Roles: result, PageInfo: pageInfo}, nil
}

func (r *roleRepoImpl) Get(ctx context.Context, roleID, accountID string) (*domain.Role, *apierror.APIError) {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.get")
	defer span.End()

	row, err := r.queries.GetRoleByIDAndAccount(ctx, sqlc.GetRoleByIDAndAccountParams{
		ID:        roleID,
		AccountID: db.NullString(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapRoleRow(row), nil
}

func (r *roleRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.exists_by_name")
	defer span.End()

	var excID gosql.NullString
	if excludeID != nil {
		excID = gosql.NullString{String: *excludeID, Valid: true}
	}

	exists, err := r.queries.ExistsRoleByName(ctx, sqlc.ExistsRoleByNameParams{
		Name:      name,
		AccountID: db.NullString(accountID),
		ExcludeID: excID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return exists, nil
}

func (r *roleRepoImpl) Create(ctx context.Context, roleID string, params domain.CreateRoleParams) *apierror.APIError {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.create")
	defer span.End()

	err := r.queries.InsertRole(ctx, sqlc.InsertRoleParams{
		ID:           roleID,
		Name:         params.Name,
		RoleTypeCode: string(constants.RoleTypeCustom),
		AccountID:    db.NullString(params.AccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *roleRepoImpl) UpdateName(ctx context.Context, roleID, accountID, name string) *apierror.APIError {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.update_name")
	defer span.End()

	err := r.queries.UpdateRoleName(ctx, sqlc.UpdateRoleNameParams{
		Name:      name,
		ID:        roleID,
		AccountID: db.NullString(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *roleRepoImpl) Delete(ctx context.Context, roleID, accountID string) *apierror.APIError {
	ctx, span := roleRepoTracer.Start(ctx, "repository.role.delete")
	defer span.End()

	err := r.queries.DeleteRoleByID(ctx, sqlc.DeleteRoleByIDParams{
		ID:        roleID,
		AccountID: db.NullString(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
