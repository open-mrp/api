package repository

import (
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/messaging"
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

func (r *repoFactoryImpl) NewAPIKeyRepo() domain.APIKeyRepo {
	return NewAPIKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewDocAPIKeyRepo() domain.DocAPIKeyRepo {
	return NewDocAPIKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewRegistrationSessionRepo() domain.RegistrationSessionRepo {
	return NewRegistrationSessionRepo(r.queries)
}

func (r *repoFactoryImpl) NewRegistrationQueueRepo() domain.RegistrationQueueRepo {
	return NewRegistrationQueueRepo(r.queries)
}

func (r *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(r.queries)
}
