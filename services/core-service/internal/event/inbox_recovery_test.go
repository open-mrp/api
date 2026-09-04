package event

import (
	"context"
	"errors"
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// The losing side of a completion race is recognized by the sentinel, not by the message text, and the sentinel has to survive being wrapped in an APIError to get there. If it stops unwrapping, a rolled-back duplicate is reported as a handler failure and retried forever instead of being acked.
func TestCompleteInboxRecord_SentinelSurvivesAPIErrorWrapping(t *testing.T) {
	t.Parallel()

	apiErr := apierror.NewInternalError(messaging.ErrInboxAlreadyCompleted, "Message was completed by a concurrent attempt.")

	if !errors.Is(apiErr, messaging.ErrInboxAlreadyCompleted) {
		t.Fatal("ErrInboxAlreadyCompleted did not survive the APIError wrapper")
	}
}

// Callers outside a wrapped delivery — tests, direct invocations — have no record to complete, and must not be forced to supply an inbox.
func TestCompleteInboxRecord_NoRecordOnContextIsANoOp(t *testing.T) {
	t.Parallel()

	if apiErr := completeInboxRecord(context.Background(), nil); apiErr != nil {
		t.Fatalf("expected no error without a record id, got %v", apiErr)
	}
}

// discardIfPermanent decides whether a message is ended for good, and getting that decision wrong is
// unrecoverable in one direction: a discarded record is terminal, handleDuplicate skips it, and
// replay cannot re-drive it. So only the handler's own drop signal may discard. Everything else
// retries and, if it keeps failing, lands on the dead-letter queue where it stays re-drivable.

func TestDiscardIfPermanent_NilErrorIsSuccess(t *testing.T) {
	t.Parallel()

	if err := discardIfPermanent(context.Background(), nil, nil); err != nil {
		t.Fatalf("a handler that succeeded has nothing to discard, got %v", err)
	}
}

// The failures an operator can fix. Every one of these is non-transient — IsTransient is false for
// validation, not-found and conflict alike — so classifying on that flag discards a message because
// a step was not visible yet or a unit conversion was missing, which are exactly the cases that
// succeed once someone fixes the data.
func TestDiscardIfPermanent_OperatorFixableErrorsAreRetriedNotDiscarded(t *testing.T) {
	t.Parallel()

	cases := map[string]*apierror.APIError{
		"not found":  apierror.NewResourceNotFoundError("Production step not found."),
		"validation": apierror.NewValidationError("Measure cannot be interpreted."),
		"conflict":   apierror.NewResourceConflictError("Item already reconciled."),
	}

	for name, apiErr := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// A nil consumer proves the point: reaching the discard path at all would panic.
			err := discardIfPermanent(context.Background(), nil, apiErr)
			if err == nil {
				t.Fatal("a fixable failure must be returned so the delivery retries, not swallowed")
			}
			if !errors.Is(err, apiErr) {
				t.Fatalf("the original error must be returned unchanged, got %v", err)
			}
		})
	}
}

// The one thing that does discard is the handler saying so.
func TestDiscardIfPermanent_RecognizesThePermanentDropSignal(t *testing.T) {
	t.Parallel()

	apiErr := newPermanentDropError("Production step no longer produces the scanned item.")

	if !errors.Is(apiErr, errPermanentDrop) {
		t.Fatal("the drop signal must survive the APIError wrapper, or nothing is ever discarded")
	}
	if errors.Is(apierror.NewValidationError("something else"), errPermanentDrop) {
		t.Fatal("an ordinary validation error must not read as a deliberate drop")
	}
}
