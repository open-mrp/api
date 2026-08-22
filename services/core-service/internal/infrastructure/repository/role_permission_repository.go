package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var rolePermissionRepoTracer = tracing.GetTracer("core-service.role_permission_repository")

type rolePermissionRepoImpl struct {
	queries *sqlc.Queries
}

func NewRolePermissionRepo(queries *sqlc.Queries) domain.RolePermissionRepo {
	return &rolePermissionRepoImpl{queries: queries}
}

func (r *rolePermissionRepoImpl) FindByRoleID(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError) {
	ctx, span := rolePermissionRepoTracer.Start(ctx, "repository.role_permission.find_by_role_id")
	defer span.End()

	rows, err := r.queries.FindRolePermissionFlags(ctx, roleID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return map[string]bool{}, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	permissionMap := make(map[string]bool, len(rows)*4)
	for _, row := range rows {
		for verb, granted := range map[string]bool{
			"create": row.Create,
			"read":   row.Read,
			"update": row.Update,
			"delete": row.Delete,
		} {
			if granted {
				permissionMap[row.PermissionCode+":"+verb] = true
			}
		}
	}

	return permissionMap, nil
}

func (r *rolePermissionRepoImpl) ListByRoleID(ctx context.Context, roleID string) ([]*domain.RolePermission, *apierror.APIError) {
	ctx, span := rolePermissionRepoTracer.Start(ctx, "repository.role_permission.list_by_role_id")
	defer span.End()

	rows, err := r.queries.ListRolePermissionsByRoleID(ctx, roleID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	permissions := make([]*domain.RolePermission, len(rows))
	for i, row := range rows {
		permissions[i] = &domain.RolePermission{
			ID:             row.ID,
			PermissionCode: row.PermissionCode,
			Create:         row.Create,
			Read:           row.Read,
			Update:         row.Update,
			Delete:         row.Delete,
			RoleID:         row.RoleID,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}

	return permissions, nil
}

func (r *rolePermissionRepoImpl) ListByRoleIDs(ctx context.Context, roleIDs []string) (map[string][]*domain.RolePermission, *apierror.APIError) {
	ctx, span := rolePermissionRepoTracer.Start(ctx, "repository.role_permission.list_by_role_ids")
	defer span.End()

	if len(roleIDs) == 0 {
		return map[string][]*domain.RolePermission{}, nil
	}

	rows, err := r.queries.ListRolePermissionsByRoleIDs(ctx, roleIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make(map[string][]*domain.RolePermission)
	for _, row := range rows {
		result[row.RoleID] = append(result[row.RoleID], &domain.RolePermission{
			ID:             row.ID,
			PermissionCode: row.PermissionCode,
			Create:         row.Create,
			Read:           row.Read,
			Update:         row.Update,
			Delete:         row.Delete,
			RoleID:         row.RoleID,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		})
	}

	return result, nil
}

func (r *rolePermissionRepoImpl) Create(ctx context.Context, permID, roleID string, input domain.CreateRolePermissionInput) *apierror.APIError {
	ctx, span := rolePermissionRepoTracer.Start(ctx, "repository.role_permission.create")
	defer span.End()

	err := r.queries.InsertRolePermission(ctx, sqlc.InsertRolePermissionParams{
		ID:             permID,
		PermissionCode: input.PermissionCode,
		Create:         input.Create,
		Read:           input.Read,
		Update:         input.Update,
		Delete:         input.Delete,
		RoleID:         roleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *rolePermissionRepoImpl) DeleteByRoleID(ctx context.Context, roleID string) *apierror.APIError {
	ctx, span := rolePermissionRepoTracer.Start(ctx, "repository.role_permission.delete_by_role_id")
	defer span.End()

	err := r.queries.DeleteRolePermissionsByRoleID(ctx, roleID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
