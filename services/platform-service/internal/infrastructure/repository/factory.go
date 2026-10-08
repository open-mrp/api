package repository

import (
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/messaging"
)

// repoFactoryImpl is the unexported concrete implementation used by the service.
type repoFactoryImpl struct {
	db       *sqlc.Queries
	payloads *blobstore.Store
}

// NewRepoFactory builds the repositories. payloads may be nil when no payloads bucket is configured.
func NewRepoFactory(db *sqlc.Queries, payloads *blobstore.Store) domain.RepoFactory {
	return &repoFactoryImpl{db: db, payloads: payloads}
}

func (f *repoFactoryImpl) NewRequestLogRepo() domain.RequestLogRepo {
	return NewRequestLogRepo(f.db, f.payloads)
}

func (f *repoFactoryImpl) NewAuditEventRepo() domain.AuditEventRepo {
	return NewAuditEventRepo(f.db)
}

func (f *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(f.db)
}

func (f *repoFactoryImpl) NewAccountFollowupRepo() domain.AccountFollowupRepo {
	return NewAccountFollowupRepo(f.db)
}
