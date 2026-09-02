package event

import (
	"context"

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
	recordID, ok := messaging.InboxRecordIDFromContext(ctx)
	if !ok {
		return nil
	}

	completed, err := f.NewInboxRepo().Complete(ctx, recordID)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to record inbox recovery point.")
	}
	if !completed {
		return apierror.NewInternalError(messaging.ErrInboxAlreadyCompleted, "Message was completed by a concurrent attempt.")
	}

	return nil
}

// newPermanentDropError builds the non-transient failure a handler returns when the message describes work that can never be done — state the event no longer matches, a measure that cannot be interpreted. Returning it from inside a transaction rolls that transaction back; discardIfPermanent then records the message as terminal.
func newPermanentDropError(reason string) *apierror.APIError {
	return apierror.NewValidationError(reason)
}

// discardIfPermanent decides a failed message's fate from the error's own classification, which is the only thing that knows whether a retry could ever succeed.
//
// A transient failure is returned unchanged so the delivery retries. A permanent one is recorded as discarded: terminal, alerted on by the failure monitor, and not retried. The alternative these replaced — logging and returning nil — marked the message processed, so a sync that permanently failed and a sync that worked were indistinguishable afterwards.
func discardIfPermanent(ctx context.Context, inbox *messaging.InboxConsumer, apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}
	if apiErr.IsTransient {
		return apiErr
	}
	return inbox.Discard(ctx, apierror.Describe(apiErr))
}
