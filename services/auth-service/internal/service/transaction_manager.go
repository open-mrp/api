package service

import (
	"database/sql"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
)

type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
	return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
