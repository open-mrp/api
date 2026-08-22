package repository

import (
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/messaging"
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

func (f *repoFactoryImpl) NewAuditEventRepo() domain.AuditEventRepo {
	return NewAuditEventRepo(f.db)
}

func (f *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(f.db)
}
