package repository

import (
	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
)

// RepoFactoryImpl is the concrete implementation used by the service.
type RepoFactoryImpl struct {
	db *sqlc.Queries
}

func NewRepoFactory(db *sqlc.Queries) domain.RepoFactory {
	return &RepoFactoryImpl{db: db}
}

func (f *RepoFactoryImpl) NewRequestLogRepo() domain.RequestLogRepo {
	return NewRequestLogRepo(f.db)
}
