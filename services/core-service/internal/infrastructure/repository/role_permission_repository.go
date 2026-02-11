package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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

	permissions, err := r.queries.FindRolePermissionStrings(ctx, roleID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return map[string]bool{}, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	permissionMap := make(map[string]bool)
	for _, p := range permissions {
		if str, ok := p.(string); ok {
			permissionMap[str] = true
		}
	}

	return permissionMap, nil
}
