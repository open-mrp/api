package ledgerlock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apierror "github.com/open-mrp/api/shared/errors"
)

type recordingLocker struct {
	acquired []string
	fail     *apierror.APIError
}

func (l *recordingLocker) LockItemForLedger(_ context.Context, itemID string) *apierror.APIError {
	if l.fail != nil {
		return l.fail
	}
	l.acquired = append(l.acquired, itemID)
	return nil
}

// One key, one direction. Two transactions wanting the same two items must take them in the same
// order or the root is just another pair of rows to deadlock over — and several flows collect their
// item ids by ranging a Go map, whose iteration order is randomised per run.
func TestAcquire_TakesItemsInAscendingOrderOnce(t *testing.T) {
	t.Parallel()

	l := &recordingLocker{}
	scope, apiErr := Acquire(context.Background(), l, []string{"it_c", "it_a", "", "it_c", "it_b"})

	require.Nil(t, apiErr)
	require.Equal(t, []string{"it_a", "it_b", "it_c"}, l.acquired,
		"blanks dropped, duplicates collapsed, and ascending — the order is the whole mechanism")
	for _, id := range []string{"it_a", "it_b", "it_c"} {
		require.True(t, scope.Holds(id))
	}
	require.False(t, scope.Holds("it_z"))
}

// A conforming caller makes the repository backstop a no-op. If it ever stops being one, every ledger
// write pays for a redundant acquisition on a lock it already holds.
func TestEnsureLocked_IsANoOpForAHeldItem(t *testing.T) {
	t.Parallel()

	l := &recordingLocker{}
	scope, apiErr := Acquire(context.Background(), l, []string{"it_a"})
	require.Nil(t, apiErr)

	require.Nil(t, scope.EnsureLocked(context.Background(), l, "it_a"))
	require.Equal(t, []string{"it_a"}, l.acquired, "a held item must not be acquired a second time")
}

// The honest residual of compiler enforcement: &Scope{} is constructible by anyone, and a forged one
// holds nothing.
//
// It must degrade to a LATE acquisition — taken, logged at ERROR, and paged on — rather than to a
// silent bypass or a panic on the nil map inside it. A forged scope that quietly skipped the lock
// would be worse than no enforcement at all, because the type would read as evidence that the root was
// held.
func TestEnsureLocked_ForgedScopeAcquiresLateRatherThanBypassing(t *testing.T) {
	t.Parallel()

	l := &recordingLocker{}
	forged := &Scope{}

	require.Nil(t, forged.EnsureLocked(context.Background(), l, "it_a"))
	require.Equal(t, []string{"it_a"}, l.acquired, "a forged scope must still take the lock")
	require.True(t, forged.Holds("it_a"), "and must record it, so the next write is not a second late acquisition")
}

// A nil scope is the other way the plumbing can be wrong, and it takes the same path.
func TestEnsureLocked_NilScopeAcquiresLate(t *testing.T) {
	t.Parallel()

	l := &recordingLocker{}
	var scope *Scope

	require.Nil(t, scope.EnsureLocked(context.Background(), l, "it_a"))
	require.Equal(t, []string{"it_a"}, l.acquired)
}

// A failed acquisition is the caller's error, not something to carry on past holding a partial set.
func TestAcquire_StopsAtTheFirstFailure(t *testing.T) {
	t.Parallel()

	l := &recordingLocker{fail: apierror.NewInternalError(nil, "boom")}
	scope, apiErr := Acquire(context.Background(), l, []string{"it_a", "it_b"})

	require.NotNil(t, apiErr)
	require.Nil(t, scope, "a partial scope would read as evidence of locks the transaction does not hold")
}
