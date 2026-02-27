package repository

import (
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/messaging"
)

type repoFactoryImpl struct {
	queries *sqlc.Queries
}

func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewAccountRepo() domain.AccountRepo {
	return NewAccountRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountUserRepo() domain.AccountUserRepo {
	return NewAccountUserRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountRelationRepo() domain.AccountRelationRepo {
	return NewAccountRelationRepo(r.queries)
}

func (r *repoFactoryImpl) NewRolePermissionRepo() domain.RolePermissionRepo {
	return NewRolePermissionRepo(r.queries)
}

func (r *repoFactoryImpl) NewSandboxAccountRepo() domain.SandboxAccountRepo {
	return NewSandboxAccountRepo(r.queries)
}

func (r *repoFactoryImpl) NewRegistrationRepo() domain.RegistrationRepo {
	return NewRegistrationRepo(r.queries)
}

func (r *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewUnitRepo() domain.UnitRepo {
	return NewUnitRepo(r.queries)
}

func (r *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(r.queries)
}
