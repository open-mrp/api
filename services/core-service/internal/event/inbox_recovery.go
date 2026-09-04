package event

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// completeInboxRecord commits this delivery's inbox recovery point inside the caller's transaction, so the marker and the work it describes commit together or not at all.
//
// Handlers whose entire effect is local writes call this as the last statement of their transaction. That is what makes them exactly-once: the alternative — letting InboxConsumer mark the message after the handler returns — leaves a window where the work is committed and the marker is not, and a record in that state is indistinguishable from one whose handler never ran, so a redelivery or a replay applies it again.
//
// A zero-row update means a concurrent attempt completed the message first. Returning an error rolls this transaction back, discarding the duplicate work.
func completeInboxRecord(ctx context.Context, f domain.RepoFactory) *apierror.APIError {
	lease, ok := messaging.InboxLeaseFromContext(ctx)
	if !ok {
		return nil
	}

	completed, err := f.NewInboxRepo().Complete(ctx, lease.RecordID, lease.Owner)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to record inbox recovery point.")
	}
	if !completed {
		return apierror.NewInternalError(messaging.ErrInboxAlreadyCompleted, "Message was completed by a concurrent attempt.")
	}

	return nil
}

// errPermanentDrop marks the errors a handler raises deliberately to end a message. Only these are discarded; see discardIfPermanent for why the error's transience is not enough on its own.
var errPermanentDrop = errors.New("message describes work that can never be done")

// newPermanentDropError builds the failure a handler returns when the message describes work that can never be done — state the event no longer matches, a measure that cannot be interpreted. Returning it from inside a transaction rolls that transaction back; discardIfPermanent then records the message as terminal.
func newPermanentDropError(reason string) *apierror.APIError {
	return apierror.NewValidationError(reason).WithInternal(fmt.Errorf("%w: %s", errPermanentDrop, reason))
}

// discardIfPermanent ends a message only when the handler said to, and otherwise lets it retry.
//
// The classification cannot come from IsTransient. That flag is false for every validation, not-found and conflict error, so keying on it discards a message because a repository lookup missed — a step not yet visible, a unit conversion an operator has not configured — and those are exactly the cases that succeed once someone fixes the data. A discarded record is terminal and skipped by handleDuplicate, so replay cannot re-drive it either: the message is simply gone.
//
// So only errPermanentDrop discards. Everything else is returned unchanged and retries, reaching the dead-letter queue if it keeps failing, where it stays visible and re-drivable.
func discardIfPermanent(ctx context.Context, inbox *messaging.InboxConsumer, apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}
	if !errors.Is(apiErr, errPermanentDrop) {
		return apiErr
	}
	return inbox.Discard(ctx, apierror.Describe(apiErr))
}
