package repository

import (
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
)

type repoFactoryImpl struct {
	queries *sqlc.Queries
}

// NewRepoFactory creates a factory that instantiates repositories backed by the
// provided sqlc query executor (raw DB or transactional).
func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewUserRepo() domain.UserRepo {
	return NewUserRepo(r.queries)
}

func (r *repoFactoryImpl) NewRefreshTokenRepo() domain.RefreshTokenRepo {
	return NewRefreshTokenRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountUserRepo() domain.AccountUserRepo {
	return NewAccountUserRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountRelationRepo() domain.AccountRelationRepo {
	return NewAccountRelationRepo(r.queries)
}

func (r *repoFactoryImpl) NewAPIKeyRepo() domain.APIKeyRepo {
	return NewAPIKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewRolePermissionRepo() domain.RolePermissionRepo {
	return NewRolePermissionRepo(r.queries)
}
