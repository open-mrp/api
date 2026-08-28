package timeutil

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBudgetedContext_ParentWithoutDeadline(t *testing.T) {
	t.Parallel()

	parent := context.Background()
	got, cancel := BudgetedContext(parent, FanOutReserve)

	if got != parent {
		t.Fatalf("BudgetedContext returned a derived context, want the parent unchanged")
	}
	if _, ok := got.Deadline(); ok {
		t.Error("returned context has a deadline, want none")
	}

	// The escape branches hand back a no-op cancel; invoking it must not kill the caller's context.
	cancel()
	if err := got.Err(); err != nil {
		t.Errorf("Err() after cancel = %v, want nil (cancel must be inert)", err)
	}
}

func TestBudgetedContext_ParentInsideReserve(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithTimeout(context.Background(), FanOutReserve/2)
	defer cancelParent()

	got, cancel := BudgetedContext(parent, FanOutReserve)
	if got != parent {
		t.Fatalf("BudgetedContext truncated a context already inside the reserve, want the parent unchanged")
	}

	parentDeadline, _ := parent.Deadline()
	gotDeadline, ok := got.Deadline()
	if !ok || !gotDeadline.Equal(parentDeadline) {
		t.Errorf("deadline = (%v, %v), want the parent's %v", gotDeadline, ok, parentDeadline)
	}

	cancel()
	if err := got.Err(); err != nil {
		t.Errorf("Err() after cancel = %v, want nil (cancel must be inert)", err)
	}
}

// A reserve equal to the whole remaining budget leaves nothing to spend: elapsed time makes the budget strictly negative.
func TestBudgetedContext_ReserveConsumesWholeBudget(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithTimeout(context.Background(), FanOutReserve)
	defer cancelParent()

	got, cancel := BudgetedContext(parent, FanOutReserve)
	defer cancel()

	if got != parent {
		t.Fatal("BudgetedContext derived a context from a zero budget, want the parent unchanged")
	}
}

func TestBudgetedContext_ParentAlreadyExpired(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelParent()

	got, cancel := BudgetedContext(parent, FanOutReserve)
	defer cancel()

	if got != parent {
		t.Fatal("BudgetedContext derived a context from an expired parent, want the parent unchanged")
	}
	if !errors.Is(got.Err(), context.DeadlineExceeded) {
		t.Errorf("Err() = %v, want context.DeadlineExceeded", got.Err())
	}
}

func TestBudgetedContext_LeavesReserveBeforeParentDeadline(t *testing.T) {
	t.Parallel()

	const parentTimeout = 10 * time.Second

	parent, cancelParent := context.WithTimeout(context.Background(), parentTimeout)
	defer cancelParent()

	got, cancel := BudgetedContext(parent, FanOutReserve)
	defer cancel()

	if got == parent {
		t.Fatal("BudgetedContext returned the parent unchanged, want a shortened context")
	}

	parentDeadline, _ := parent.Deadline()
	gotDeadline, ok := got.Deadline()
	if !ok {
		t.Fatal("derived context has no deadline")
	}

	// gap == reserve minus however long the call itself took, so it can only ever come in under the reserve.
	gap := parentDeadline.Sub(gotDeadline)
	if gap > FanOutReserve || gap < FanOutReserve-time.Second {
		t.Errorf("derived deadline sits %v before the parent's, want ~%v", gap, FanOutReserve)
	}
}

func TestBudgetedContext_HonorsCustomReserve(t *testing.T) {
	t.Parallel()

	const (
		parentTimeout = 10 * time.Second
		reserve       = 4 * time.Second
	)

	parent, cancelParent := context.WithTimeout(context.Background(), parentTimeout)
	defer cancelParent()

	got, cancel := BudgetedContext(parent, reserve)
	defer cancel()

	parentDeadline, _ := parent.Deadline()
	gotDeadline, ok := got.Deadline()
	if !ok {
		t.Fatal("derived context has no deadline")
	}

	gap := parentDeadline.Sub(gotDeadline)
	if gap > reserve || gap < reserve-time.Second {
		t.Errorf("derived deadline sits %v before the parent's, want ~%v", gap, reserve)
	}
}

func TestBudgetedContext_CancelStopsFanOutOnly(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelParent()

	got, cancel := BudgetedContext(parent, FanOutReserve)
	cancel()

	if !errors.Is(got.Err(), context.Canceled) {
		t.Errorf("derived Err() = %v, want context.Canceled", got.Err())
	}
	// The whole point of the budget is that the caller survives the fan-out to assemble a response.
	if err := parent.Err(); err != nil {
		t.Errorf("parent Err() = %v, want nil", err)
	}
}
