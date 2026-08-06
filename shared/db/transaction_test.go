package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	apierror "github.com/augno/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockQueries struct {
	db        *sql.DB
	tx        *sql.Tx
	txQueries *mockQueries
}

func (m *mockQueries) WithTx(tx *sql.Tx) *mockQueries {
	txQ := &mockQueries{db: m.db, tx: tx}
	m.txQueries = txQ
	return txQ
}

type mockFactory struct {
	queries *mockQueries
}

func TestTransactionManager_WithTx_Success(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	queries := &mockQueries{db: db}
	factoryCreate := func(q *mockQueries) *mockFactory {
		return &mockFactory{queries: q}
	}

	txMgr := NewTransactionManager(db, queries, factoryCreate)

	callbackCalled := false
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		callbackCalled = true
		assert.NotNil(t, f)
		assert.NotNil(t, f.queries.tx, "factory should receive tx-bound queries")
		return nil
	})

	assert.Nil(t, apiErr)
	assert.True(t, callbackCalled)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionManager_WithTx_RollbackOnError(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	queries := &mockQueries{db: db}
	factoryCreate := func(q *mockQueries) *mockFactory {
		return &mockFactory{queries: q}
	}

	txMgr := NewTransactionManager(db, queries, factoryCreate)

	expectedErr := apierror.NewInternalError(errors.New("test error"), "callback failed")
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		return expectedErr
	})

	assert.Equal(t, expectedErr, apiErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionManager_WithTx_BeginError(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	queries := &mockQueries{db: db}
	factoryCreate := func(q *mockQueries) *mockFactory {
		return &mockFactory{queries: q}
	}

	txMgr := NewTransactionManager(db, queries, factoryCreate)

	callbackCalled := false
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		callbackCalled = true
		return nil
	})

	assert.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInternalError, apiErr.Code)
	assert.False(t, callbackCalled, "callback should not be called when begin fails")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionManager_WithTx_CommitError(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	queries := &mockQueries{db: db}
	factoryCreate := func(q *mockQueries) *mockFactory {
		return &mockFactory{queries: q}
	}

	txMgr := NewTransactionManager(db, queries, factoryCreate)

	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		return nil
	})

	assert.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInternalError, apiErr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionManager_WithTx_FactoryReceivesTxQueries(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	queries := &mockQueries{db: db}
	var receivedQueries *mockQueries
	factoryCreate := func(q *mockQueries) *mockFactory {
		receivedQueries = q
		return &mockFactory{queries: q}
	}

	txMgr := NewTransactionManager(db, queries, factoryCreate)

	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		return nil
	})

	assert.Nil(t, apiErr)
	assert.NotNil(t, receivedQueries)
	assert.NotNil(t, receivedQueries.tx, "queries should have transaction set")
	assert.Equal(t, queries.txQueries, receivedQueries, "factory should receive the tx-bound queries")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A partial-success batch: one unit of work succeeds (SAVEPOINT then RELEASE) and one
// fails (SAVEPOINT then ROLLBACK TO SAVEPOINT). The failing unit's error surfaces from
// its Run, the transaction stays alive, and the outer WithTxSavepoint commits the rest.
func TestTransactionManager_WithTxSavepoint_PartialSuccess(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SAVEPOINT sp1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("RELEASE SAVEPOINT sp1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SAVEPOINT sp2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ROLLBACK TO SAVEPOINT sp2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	queries := &mockQueries{db: db}
	factoryCreate := func(q *mockQueries) *mockFactory { return &mockFactory{queries: q} }
	txMgr := NewTransactionManager(db, queries, factoryCreate)

	rowErr := apierror.NewValidationError("bad row")
	var okErr, failErr *apierror.APIError
	apiErr := txMgr.WithTxSavepoint(context.Background(), func(ctx context.Context, f *mockFactory, sp SavepointRunner) *apierror.APIError {
		okErr = sp.Run(ctx, func(context.Context) *apierror.APIError { return nil })
		failErr = sp.Run(ctx, func(context.Context) *apierror.APIError { return rowErr })
		return nil
	})

	assert.Nil(t, apiErr, "the batch commits even though one unit failed")
	assert.Nil(t, okErr)
	assert.Same(t, rowErr, failErr, "the failing unit's error surfaces from its Run")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// If ROLLBACK TO SAVEPOINT itself fails the transaction is doomed, so Run surfaces an
// internal error rather than the caller's, and the batch aborts.
func TestTransactionManager_WithTxSavepoint_RollbackFailureAborts(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SAVEPOINT sp1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ROLLBACK TO SAVEPOINT sp1").WillReturnError(errors.New("connection lost"))
	mock.ExpectRollback()

	queries := &mockQueries{db: db}
	factoryCreate := func(q *mockQueries) *mockFactory { return &mockFactory{queries: q} }
	txMgr := NewTransactionManager(db, queries, factoryCreate)

	apiErr := txMgr.WithTxSavepoint(context.Background(), func(ctx context.Context, f *mockFactory, sp SavepointRunner) *apierror.APIError {
		return sp.Run(ctx, func(context.Context) *apierror.APIError {
			return apierror.NewValidationError("bad row")
		})
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInternalError, apiErr.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
