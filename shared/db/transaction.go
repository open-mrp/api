package db

import (
	"context"
	"database/sql"
	"fmt"

	apierror "github.com/augno/api/shared/errors"
)

type TxQuerier[Q any] interface {
	WithTx(tx *sql.Tx) Q
}

// SavepointRunner brackets a unit of work in a SAVEPOINT within an open transaction.
// Run releases the savepoint on success and rolls back to it on error — undoing only
// that unit's writes while the surrounding transaction stays alive — so a batch can let
// one item fail without discarding the rest. Obtain one from WithTxSavepoint.
type SavepointRunner interface {
	Run(ctx context.Context, fn func(ctx context.Context) *apierror.APIError) *apierror.APIError
}

type TransactionManager[Q TxQuerier[Q], F any] interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, f F) *apierror.APIError) *apierror.APIError
	// WithTxSavepoint is WithTx plus a SavepointRunner over the same transaction, for
	// partial-success batches: successful items and whatever the callback commits still
	// commit together at the end, and a mid-batch crash rolls the whole thing back.
	WithTxSavepoint(ctx context.Context, fn func(ctx context.Context, f F, sp SavepointRunner) *apierror.APIError) *apierror.APIError
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

func (m *transactionManagerImpl[Q, F]) WithTxSavepoint(
	ctx context.Context,
	fn func(ctx context.Context, f F, sp SavepointRunner) *apierror.APIError,
) *apierror.APIError {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return apierror.NewInternalError(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	qTx := m.queries.WithTx(tx)
	factory := m.factoryCreate(qTx)

	if apiErr := fn(ctx, factory, &savepointRunner{tx: tx}); apiErr != nil {
		return apiErr
	}

	if err := tx.Commit(); err != nil {
		return apierror.NewInternalError(err, "failed to commit transaction")
	}

	return nil
}

// savepointRunner issues named SAVEPOINTs on one transaction. Names are per-runner
// counters, so nested or concurrent use on the same runner is not supported.
type savepointRunner struct {
	tx *sql.Tx
	n  int
}

func (r *savepointRunner) Run(ctx context.Context, fn func(ctx context.Context) *apierror.APIError) *apierror.APIError {
	r.n++
	name := fmt.Sprintf("sp%d", r.n)

	if _, err := r.tx.ExecContext(ctx, "SAVEPOINT "+name); err != nil {
		return apierror.NewInternalError(err, "failed to create savepoint")
	}

	if apiErr := fn(ctx); apiErr != nil {
		if _, rbErr := r.tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+name); rbErr != nil {
			// The savepoint could not be rolled back — the transaction is doomed. Surface
			// this rather than fn's error so the batch aborts instead of pressing on
			// against a broken transaction.
			return apierror.NewInternalError(rbErr, "failed to roll back to savepoint")
		}
		return apiErr
	}

	if _, err := r.tx.ExecContext(ctx, "RELEASE SAVEPOINT "+name); err != nil {
		return apierror.NewInternalError(err, "failed to release savepoint")
	}
	return nil
}
