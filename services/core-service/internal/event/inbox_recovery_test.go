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
