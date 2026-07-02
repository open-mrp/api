package service

import (
	"database/sql"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/repository"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
)

// TransactionManager runs a function against a transaction-bound RepoFactory so multi-repo writes (sequence allocation, message insert, conversation denormalization, notification rows, realtime outbox) commit atomically.
type TransactionManager = db.TransactionManager[*sqlc.Queries, domain.RepoFactory]

func NewTransactionManager(sqlDB *sql.DB, queries *sqlc.Queries) TransactionManager {
	return db.NewTransactionManager(sqlDB, queries, repository.NewRepoFactory)
}
