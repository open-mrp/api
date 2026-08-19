package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	apierror "github.com/augno/api/shared/errors"
	"github.com/go-sql-driver/mysql"
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

// ──────────────────────────────────────────────
// Deadlock retry
// ──────────────────────────────────────────────

// deadlockErr is what the MySQL driver returns to the transaction chosen as the victim.
func deadlockErr() error {
	return &mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock; try restarting transaction"}
}

func newDeadlockTestManager(t *testing.T) (*sql.DB, sqlmock.Sqlmock, TransactionManager[*mockQueries, *mockFactory]) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	return db, mock, NewTransactionManager(db, &mockQueries{db: db}, func(q *mockQueries) *mockFactory {
		return &mockFactory{queries: q}
	})
}

// The retry exists so a deadlock does not reach the caller at all: the second attempt starts
// from the state the first one did, because the database rolled it back.
func TestTransactionManager_WithTx_RetriesAfterDeadlock(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectCommit()

	attempts := 0
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		attempts++
		if attempts == 1 {
			return MapSQLError(deadlockErr())
		}
		return nil
	})

	assert.Nil(t, apiErr, "the second attempt succeeded, so the caller sees success")
	assert.Equal(t, 2, attempts, "the callback runs again on the retry")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A deadlock at commit is the common case — that is when InnoDB resolves the locks the
// transaction has been holding — so it has to be retried like one raised mid-transaction.
func TestTransactionManager_WithTx_RetriesADeadlockAtCommit(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	// No rollback between the two: database/sql considers the transaction finished once Commit
	// returns, so the deferred rollback never reaches the driver.
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(deadlockErr())
	mock.ExpectBegin()
	mock.ExpectCommit()

	attempts := 0
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		attempts++
		return nil
	})

	assert.Nil(t, apiErr)
	assert.Equal(t, 2, attempts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Retrying forever would hold a request open against contention no retry can resolve.
func TestTransactionManager_WithTx_GivesUpAfterRepeatedDeadlocks(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	for range deadlockMaxAttempts {
		mock.ExpectBegin()
		mock.ExpectRollback()
	}

	attempts := 0
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		attempts++
		return MapSQLError(deadlockErr())
	})

	require.NotNil(t, apiErr, "a deadlock that never clears must still reach the caller")
	assert.Equal(t, deadlockMaxAttempts, attempts)
	assert.True(t, IsDeadlock(apiErr), "the surfaced error must still identify itself as a deadlock")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Only deadlocks are re-run. Anything else is the work failing, and running it again would
// turn one rejection into several.
func TestTransactionManager_WithTx_DoesNotRetryOtherFailures(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	attempts := 0
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		attempts++
		return apierror.NewValidationError("nope")
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, 1, attempts, "a validation failure is not retried")
	assert.Equal(t, "nope", apiErr.PublicMessage)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A duplicate-key violation is deterministic: the row is already there, and it will still be
// there on a second attempt.
func TestTransactionManager_WithTx_DoesNotRetryADuplicateKey(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	attempts := 0
	apiErr := txMgr.WithTx(context.Background(), func(ctx context.Context, f *mockFactory) *apierror.APIError {
		attempts++
		return MapSQLError(&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"})
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, 1, attempts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Retrying past the caller's deadline helps nobody, and the deadlock is the more useful thing
// to report than the cancellation that followed it.
func TestTransactionManager_WithTx_StopsRetryingOnceTheContextEnds(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	apiErr := txMgr.WithTx(ctx, func(c context.Context, f *mockFactory) *apierror.APIError {
		attempts++
		cancel()
		return MapSQLError(deadlockErr())
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, 1, attempts, "no retry once the caller has gone")
	assert.True(t, IsDeadlock(apiErr), "the deadlock is reported, not the cancellation")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The savepoint variant shares the retry, so a batch that deadlocks is re-run whole rather than
// leaving the caller with a partially applied one.
func TestTransactionManager_WithTxSavepoint_RetriesAfterDeadlock(t *testing.T) {
	t.Parallel()
	_, mock, txMgr := newDeadlockTestManager(t)

	mock.ExpectBegin()
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectCommit()

	attempts := 0
	apiErr := txMgr.WithTxSavepoint(context.Background(), func(ctx context.Context, f *mockFactory, sp SavepointRunner) *apierror.APIError {
		attempts++
		if attempts == 1 {
			return MapSQLError(deadlockErr())
		}
		return nil
	})

	assert.Nil(t, apiErr)
	assert.Equal(t, 2, attempts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The wait is short enough to be worth taking inside a request, and spread so two victims of
// the same deadlock do not wake together and collide again.
func TestDeadlockBackoff_IsShortAndSpread(t *testing.T) {
	t.Parallel()

	seen := map[time.Duration]bool{}
	for range 50 {
		d := deadlockBackoff(0)
		assert.Positive(t, d)
		assert.Less(t, d, 20*time.Millisecond, "the first retry must not stall the request")
		seen[d] = true
	}
	assert.Greater(t, len(seen), 1, "identical waits would make two victims collide again")

	assert.Greater(t, deadlockBackoff(2), deadlockBaseBackoff, "later retries back off further")
}
