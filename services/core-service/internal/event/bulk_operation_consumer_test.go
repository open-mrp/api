package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// bulkDelivery builds the AMQP delivery a bulk command arrives as: the job-id event
// wrapped in the standard message envelope.
func bulkDelivery(t *testing.T, jobID string) amqp.Delivery {
	t.Helper()

	payload, err := json.Marshal(domain.BulkOperationJobEvent{JobID: jobID})
	if err != nil {
		t.Fatalf("failed to encode the event: %v", err)
	}
	body, err := json.Marshal(contracts.AmqpMessage{Data: payload})
	if err != nil {
		t.Fatalf("failed to encode the message: %v", err)
	}
	return amqp.Delivery{Body: body, RoutingKey: "core.cmd.test", MessageId: "msg_1"}
}

// What the consumer does with a failed execution decides whether the delivery is
// retried and eventually dead-lettered, or acknowledged where it is. Returning an
// error hands it back to the broker-level backoff; returning nil acknowledges.
//
// Only a transient failure earns a retry. A deterministic one fails identically on
// every redelivery, and the execute phase has already recorded it on the job, so
// retrying would burn the backoff attempts and dead-letter a message that replaying
// could never settle differently.
func TestBulkOperationConsumer_RetriesOnlyTransientFailures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		executeErr *apierror.APIError
		wantRetry  bool
	}{
		{
			name:       "a successful execution is acknowledged",
			executeErr: nil,
			wantRetry:  false,
		},
		{
			name:       "an infrastructure failure is handed back for retry",
			executeErr: apierror.NewInternalError(errors.New("dial tcp: connection refused"), "The database is unreachable."),
			wantRetry:  true,
		},
		{
			name:       "a validation failure is acknowledged, not retried",
			executeErr: apierror.NewValidationError("Job items are not a bulk operation payload."),
			wantRetry:  false,
		},
		{
			name:       "a conflict is acknowledged, not retried",
			executeErr: apierror.NewResourceConflictError("The name matches a different existing unit."),
			wantRetry:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var executed int
			consumer := &BulkOperationConsumer{
				name:   "test_operation",
				tracer: tracing.GetTracer("test"),
				execute: func(_ context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
					executed++
					if event.JobID != "jo_test" {
						t.Errorf("the consumer must hand the executor the event's job, got %q", event.JobID)
					}
					return tc.executeErr
				},
			}

			err := consumer.handleMessage(context.Background(), bulkDelivery(t, "jo_test"))

			if executed != 1 {
				t.Errorf("expected the executor to run once, ran %d times", executed)
			}
			if tc.wantRetry && err == nil {
				t.Error("a transient failure must be returned so the delivery is retried")
			}
			if !tc.wantRetry && err != nil {
				t.Errorf("this delivery must be acknowledged rather than retried, got: %v", err)
			}
		})
	}
}

// A malformed delivery cannot name a job, so there is nothing to execute and nothing
// a retry could fix.
func TestBulkOperationConsumer_RejectsAnUnreadableMessage(t *testing.T) {
	t.Parallel()

	consumer := &BulkOperationConsumer{
		name:   "test_operation",
		tracer: tracing.GetTracer("test"),
		execute: func(context.Context, domain.BulkOperationJobEvent) *apierror.APIError {
			t.Error("the executor must not run for a message that does not parse")
			return nil
		},
	}

	if err := consumer.handleMessage(context.Background(), amqp.Delivery{Body: []byte("not json")}); err == nil {
		t.Error("expected an unreadable message to be reported")
	}
}
