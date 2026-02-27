package repository

import (
	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
)

// repoFactoryImpl is the unexported concrete implementation used by the service.
type repoFactoryImpl struct {
	db *sqlc.Queries
}

func NewRepoFactory(db *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{db: db}
}

func (f *repoFactoryImpl) NewRequestLogRepo() domain.RequestLogRepo {
	return NewRequestLogRepo(f.db)
}
