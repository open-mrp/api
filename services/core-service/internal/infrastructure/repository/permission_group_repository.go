package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var permissionGroupRepoTracer = tracing.GetTracer("core-service.permission_group_repository")

type permissionGroupRepoImpl struct {
	queries *sqlc.Queries
}

func NewPermissionGroupRepo(queries *sqlc.Queries) domain.PermissionGroupRepo {
	return &permissionGroupRepoImpl{queries: queries}
}

func pgCreatedAt(pg *domain.PermissionGroup) time.Time { return pg.CreatedAt }
func pgID(pg *domain.PermissionGroup) string           { return pg.ID }

// mapPermissionGroupRow maps one permission_group row to the domain.
//
// sqlc emits a distinct row struct per query even when the column lists are identical, so this takes
// one of them and the other call sites convert. The conversion is compiler-checked: a query whose
// columns ever diverge from this shape fails the build here rather than mapping the wrong field.
func mapPermissionGroupRow(row sqlc.GetPermissionGroupsByIDsRow) *domain.PermissionGroup {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	return &domain.PermissionGroup{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Description: description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapPermissionRow(row sqlc.ListPermissionsByGroupCodesRow) *domain.Permission {
	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	return &domain.Permission{
		ID:                  row.ID,
		Code:                row.Code,
		Name:                row.Name,
		Description:         description,
		PermissionGroupCode: row.PermissionGroupCode,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func (r *permissionGroupRepoImpl) GetByIDs(ctx context.Context, ids []string) ([]*domain.PermissionGroup, *apierror.APIError) {
	ctx, span := permissionGroupRepoTracer.Start(ctx, "repository.permission_group.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetPermissionGroupsByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	groups := make([]*domain.PermissionGroup, len(rows))
	for i, row := range rows {
		groups[i] = mapPermissionGroupRow(row)
	}

	if apiErr := r.loadPermissions(ctx, groups); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return groups, nil
}

func (r *permissionGroupRepoImpl) List(ctx context.Context, params domain.ListPermissionGroupsParams) (*domain.ListPermissionGroupsResult, *apierror.APIError) {
	ctx, span := permissionGroupRepoTracer.Start(ctx, "repository.permission_group.list")
	defer span.End()

	searchQuery := gosql.NullString{}
	if params.Query != nil && *params.Query != "" {
		searchQuery = gosql.NullString{String: "%" + db.EscapeLike(*params.Query) + "%", Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListPermissionGroupsBackward(ctx, sqlc.ListPermissionGroupsBackwardParams{
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			groups := make([]*domain.PermissionGroup, len(rows))
			for i, row := range rows {
				groups[i] = mapPermissionGroupRow(sqlc.GetPermissionGroupsByIDsRow(row))
			}
			result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, pgCreatedAt, pgID)
			if apiErr := r.loadPermissions(ctx, result); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			return &domain.ListPermissionGroupsResult{PermissionGroups: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListPermissionGroupsForward(ctx, sqlc.ListPermissionGroupsForwardParams{
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		groups := make([]*domain.PermissionGroup, len(rows))
		for i, row := range rows {
			groups[i] = mapPermissionGroupRow(sqlc.GetPermissionGroupsByIDsRow(row))
		}
		result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, pgCreatedAt, pgID)
		if apiErr := r.loadPermissions(ctx, result); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListPermissionGroupsResult{PermissionGroups: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListPermissionGroupsForward(ctx, sqlc.ListPermissionGroupsForwardParams{
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	groups := make([]*domain.PermissionGroup, len(rows))
	for i, row := range rows {
		groups[i] = mapPermissionGroupRow(sqlc.GetPermissionGroupsByIDsRow(row))
	}
	result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, pgCreatedAt, pgID)
	if apiErr := r.loadPermissions(ctx, result); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListPermissionGroupsResult{PermissionGroups: result, PageInfo: pageInfo}, nil
}

// loadPermissions batch-loads permissions for the given permission groups and attaches them.
func (r *permissionGroupRepoImpl) loadPermissions(ctx context.Context, groups []*domain.PermissionGroup) *apierror.APIError {
	if len(groups) == 0 {
		return nil
	}

	codes := make([]string, len(groups))
	for i, g := range groups {
		codes[i] = g.Code
	}

	rows, err := r.queries.ListPermissionsByGroupCodes(ctx, codes)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}

	// Group permissions by permission_group_code
	permsByCode := make(map[string][]*domain.Permission)
	for _, row := range rows {
		perm := mapPermissionRow(row)
		permsByCode[perm.PermissionGroupCode] = append(permsByCode[perm.PermissionGroupCode], perm)
	}

	for _, g := range groups {
		g.Permissions = permsByCode[g.Code]
		if g.Permissions == nil {
			g.Permissions = []*domain.Permission{}
		}
	}

	return nil
}
