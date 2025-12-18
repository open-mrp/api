package repository

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/tracing"
)

var rolePermissionRepoTracer = tracing.GetTracer("auth-service.role_permission_repository")

type rolePermissionRepoImpl struct {
	db *sqlc.Queries
}

func NewRolePermissionRepo(db *sqlc.Queries) domain.RolePermissionRepo {
	return &rolePermissionRepoImpl{db: db}
}

func (r *rolePermissionRepoImpl) FindByRoleID(ctx context.Context, roleID string) (map[string]bool, *contracts.APIError) {
	ctx, span := rolePermissionRepoTracer.Start(ctx, "repository.role_permission.findByRoleID")
	defer span.End()

	permissions, err := r.db.FindRolePermissionStrings(ctx, roleID)

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == contracts.ErrorCodeResourceNotFound {
			return nil, nil
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
