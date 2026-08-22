package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/api/shared/errors"
)

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *apierror.APIError) *apierror.APIError
}

type pgxTxManager struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewTransactionManager(pool *pgxpool.Pool, queries *sqlc.Queries) TransactionManager {
	return &pgxTxManager{pool: pool, queries: queries}
}

func (m *pgxTxManager) WithTx(ctx context.Context, fn func(ctx context.Context, f domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return apierror.NewInternalError(err, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	txQueries := m.queries.WithTx(tx)
	factory := repository.NewRepoFactory(txQueries)

	if apiErr := fn(ctx, factory); apiErr != nil {
		return apiErr
	}

	if err := tx.Commit(ctx); err != nil {
		return apierror.NewInternalError(err, "failed to commit transaction")
	}

	return nil
}
