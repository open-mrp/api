package repository

import (
	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
)

type repoFactoryImpl struct {
	db *sqlc.Queries
}

func NewRepoFactory(db *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{db: db}
}

func (f *repoFactoryImpl) NewEmailLogRepo() domain.EmailLogRepo {
	return NewEmailLogRepo(f.db)
}

func (f *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(f.db)
}
