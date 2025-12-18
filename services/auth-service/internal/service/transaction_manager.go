package service

import (
	"context"
	"database/sql"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
)

// TransactionManager executes service logic inside a single ACID transaction
// while preserving clean architecture boundaries (no *sql.Tx leakage).
type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *contracts.APIError) *contracts.APIError
}

type transactionManagerImpl struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewTransactionManager(db *sql.DB, queries *sqlc.Queries) TransactionManager {
	return &transactionManagerImpl{db: db, queries: queries}
}

func (m *transactionManagerImpl) WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *contracts.APIError) *contracts.APIError {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.NewInternalError(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	qTx := m.queries.WithTx(tx)
	repoFactory := repository.NewRepoFactory(qTx)

	if err := fn(ctx, repoFactory); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return contracts.NewInternalError(err, "failed to commit transaction")
	}

	return nil
}
