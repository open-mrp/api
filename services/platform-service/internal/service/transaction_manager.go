package service

import (
	"database/sql"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
)

// TransactionManager runs a function against a transaction-bound RepoFactory, so a state change and the outbox message announcing it commit together.
type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
	return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
