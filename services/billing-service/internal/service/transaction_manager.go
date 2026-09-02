package service

import (
	"database/sql"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	"github.com/open-mrp/api/services/billing-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/billing-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
)

// TransactionManager runs a function against a transaction-bound RepoFactory. Billing's writes are single statements, so the transaction exists to pair a write with the inbox recovery point that says it happened — the token accumulator is additive, and a duplicate delivery that lands without one overstates usage.
type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
	return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
