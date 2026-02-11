package db

import (
	"context"
	"database/sql"

	apierror "github.com/augno/api/shared/errors"
)

type TxQuerier[Q any] interface {
	WithTx(tx *sql.Tx) Q
}

type TransactionManager[Q TxQuerier[Q], F any] interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, f F) *apierror.APIError) *apierror.APIError
}

type transactionManagerImpl[Q TxQuerier[Q], F any] struct {
	db            *sql.DB
	queries       Q
	factoryCreate func(Q) F
}

func NewTransactionManager[Q TxQuerier[Q], F any](
	db *sql.DB,
	queries Q,
	factoryCreate func(Q) F,
) TransactionManager[Q, F] {
	return &transactionManagerImpl[Q, F]{
		db:            db,
		queries:       queries,
		factoryCreate: factoryCreate,
	}
}

func (m *transactionManagerImpl[Q, F]) WithTx(
	ctx context.Context,
	fn func(ctx context.Context, f F) *apierror.APIError,
) *apierror.APIError {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return apierror.NewInternalError(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	qTx := m.queries.WithTx(tx)
	factory := m.factoryCreate(qTx)

	if apiErr := fn(ctx, factory); apiErr != nil {
		return apiErr
	}

	if err := tx.Commit(); err != nil {
		return apierror.NewInternalError(err, "failed to commit transaction")
	}

	return nil
}
