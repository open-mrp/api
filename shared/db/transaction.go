package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
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

// TransactionManager runs a unit of work in a database transaction.
//
// A transaction that loses a lock conflict is re-run, so callbacks must be safe to
// execute more than once. In practice that means a callback may only write to the database:
// its writes are rolled back with the transaction, so a second run starts from the same state
// the first one did. Anything that escapes the database does not get undone — an HTTP call to
// a payment provider, a message published straight to the broker, a value appended to a slice
// declared outside the callback — and would happen twice.
//
// That is why events are written to the outbox rather than published inline, and why results
// are assembled inside the callback and handed out at the end. `make tx-audit` checks these
// rules across the codebase.
type TransactionManager[Q TxQuerier[Q], F any] interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, f F) *apierror.APIError) *apierror.APIError
	// WithTxSavepoint is WithTx plus a SavepointRunner over the same transaction, for
	// partial-success batches: successful items and whatever the callback commits still
	// commit together at the end, and a mid-batch crash rolls the whole thing back.
	WithTxSavepoint(ctx context.Context, fn func(ctx context.Context, f F, sp SavepointRunner) *apierror.APIError) *apierror.APIError
}

const (
	// deadlockMaxAttempts bounds how many times a transaction is re-run after losing to a lock
	// conflict. Two transactions deadlocking resolve on the first retry, because the one that
	// survived has committed by then. Needing more than this means sustained contention, which
	// re-running cannot fix — better to surface it than to keep holding the request open.
	//
	// "Lock conflict" and not just "deadlock": IsRetryableLockConflict also covers 1205, the
	// lock-wait timeout. That is the same contention arbitrated the other way — nobody is picked as
	// a victim, the waiter simply gives up — and it is the failure mode that replaces 1213 wherever
	// contending transactions are given a consistent lock order. Retrying one and not the other
	// would mean this loop quietly stopped working at exactly the point the ordering work landed.
	//
	// A 1205 costs the full innodb_lock_wait_timeout before it is seen (20s on PlanetScale), so a
	// retried one is slow where a retried 1213 is immediate. Still worth retrying: the alternative
	// is failing work that would have succeeded, and the caller above is a message that will be
	// redelivered anyway.
	deadlockMaxAttempts = 3

	// deadlockBaseBackoff is the wait before the first retry, doubling thereafter.
	//
	// InnoDB detects a deadlock and rolls the victim back immediately, so there is nothing to
	// wait for — the pause exists only to stagger the retry away from whatever else is
	// contending for the same rows. Milliseconds, not the seconds a network retry would use:
	// the request deadline is still ticking, and the work itself takes longer than the wait.
	deadlockBaseBackoff = 5 * time.Millisecond

	// deadlockBackoffJitter spreads each wait by ±40% so two victims of the same deadlock do
	// not wake together and collide again.
	deadlockBackoffJitter = 0.4
)

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
	return m.run(ctx, func(ctx context.Context, _ *sql.Tx, f F) *apierror.APIError {
		return fn(ctx, f)
	})
}

func (m *transactionManagerImpl[Q, F]) WithTxSavepoint(
	ctx context.Context,
	fn func(ctx context.Context, f F, sp SavepointRunner) *apierror.APIError,
) *apierror.APIError {
	return m.run(ctx, func(ctx context.Context, tx *sql.Tx, f F) *apierror.APIError {
		return fn(ctx, f, &savepointRunner{tx: tx})
	})
}

// run executes fn in a transaction, re-running it when the database rejects it over a lock
// conflict — chosen as a deadlock victim (1213), or timed out waiting for a lock (1205).
//
// Neither is a failure of the work — both are the database arbitrating between two transactions
// that wanted the same rows, and the loser is asked to try again. The losing transaction is rolled
// back completely, so the retry starts from the same state the first attempt did. Surfacing it
// instead would make every caller implement this loop, and a 500 for a condition resolved in five
// milliseconds is not an answer worth giving.
func (m *transactionManagerImpl[Q, F]) run(
	ctx context.Context,
	fn func(ctx context.Context, tx *sql.Tx, f F) *apierror.APIError,
) *apierror.APIError {
	for attempt := 0; ; attempt++ {
		apiErr, conflicted := m.attempt(ctx, fn)
		if apiErr == nil || !conflicted || attempt >= deadlockMaxAttempts-1 {
			if conflicted && apiErr != nil {
				slog.WarnContext(ctx, "transaction abandoned after repeated lock conflicts",
					"attempts", attempt+1,
					"error", apiErr.Error(),
				)
			}
			return apiErr
		}

		// A retry that outlives the caller's deadline helps nobody, and the lock conflict is the
		// more useful thing to report than the cancellation that followed it.
		if !sleepFor(ctx, deadlockBackoff(attempt)) {
			return apiErr
		}

		// Logged rather than swallowed: retries hide contention, and a path that conflicts
		// constantly needs its lock ordering fixed, not its symptoms absorbed.
		slog.WarnContext(ctx, "retrying transaction after lock conflict", "attempt", attempt+1)
	}
}

// attempt runs fn once in its own transaction and reports whether it failed to a lock conflict. It is
// a separate function so the rollback is deferred per attempt rather than accumulating across
// the retry loop.
func (m *transactionManagerImpl[Q, F]) attempt(
	ctx context.Context,
	fn func(ctx context.Context, tx *sql.Tx, f F) *apierror.APIError,
) (*apierror.APIError, bool) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return apierror.NewInternalError(err, "failed to begin transaction"), IsRetryableLockConflict(err)
	}
	defer tx.Rollback()

	qTx := m.queries.WithTx(tx)
	factory := m.factoryCreate(qTx)

	if apiErr := fn(ctx, tx, factory); apiErr != nil {
		// MapSQLError keeps the driver error underneath, so the lock conflict is still visible
		// through the APIError the repository layer returned.
		return apiErr, IsRetryableLockConflict(apiErr)
	}

	// Committing is where a lock conflict most often surfaces, since that is when InnoDB resolves
	// the locks the transaction has been holding.
	if err := tx.Commit(); err != nil {
		return apierror.NewInternalError(err, "failed to commit transaction"), IsRetryableLockConflict(err)
	}

	return nil, false
}

// deadlockBackoff returns the wait before the given retry: an exponential base with symmetric
// jitter, so two transactions that deadlocked with each other separate rather than collide again.
func deadlockBackoff(attempt int) time.Duration {
	base := float64(deadlockBaseBackoff) * math.Pow(2, float64(attempt))
	return time.Duration(base * (1 + deadlockBackoffJitter*(randFraction()*2-1)))
}

// randFraction returns a random value in [0, 1). It reads from crypto/rand to avoid math/rand's
// global lock on a path every write transaction can reach; a failed read yields the midpoint,
// which costs a little jitter rather than panicking inside a transaction.
func randFraction() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0.5
	}
	return float64(binary.BigEndian.Uint64(b[:])>>11) / (1 << 53)
}

// sleepFor waits for d, reporting false if the context ended first.
func sleepFor(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
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
