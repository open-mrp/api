package service

import (
	"database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
)

type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries, payloads *repository.Payloads) TransactionManager {
	return db.NewTransactionManager(sqlDB, queries, func(q *sqlc.Queries) domain.RepoFactory {
		return repository.NewRepoFactory(q, payloads)
	})
}
