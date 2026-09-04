package event

import (
	"context"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// completeInboxRecord commits this delivery's inbox recovery point inside the caller's transaction, so the marker and the work it describes commit together or not at all.
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
