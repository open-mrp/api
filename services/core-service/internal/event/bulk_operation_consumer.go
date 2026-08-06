package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BulkOperationExecutor performs the write phase of an enqueued bulk operation: it loads
// the job named by the event and runs its resolved rows. Each entity's service supplies
// one — its Execute method value (e.g. unitGroupSvc.ExecuteBulkUpsertUnitGroups).
type BulkOperationExecutor func(context.Context, domain.BulkOperationJobEvent) *apierror.APIError

// BulkOperationConsumer is the single consumer behind every async bulk operation. Each
// bulk endpoint enqueues a {job_id} command; this consumer loads the job (whose resolved,
// validated payload was stored at enqueue time), restores the originating identity, and
// hands off to the entity's executor. The only variance between operations is data — the
// queue, the inbox handler key, and the executor — so there is one implementation instead
// of one near-identical file per operation. The message inbox de-dupes redeliveries so
// retries converge on one visible outcome.
type BulkOperationConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	queue         string
	handler       string
	name          string
	execute       BulkOperationExecutor
	tracer        trace.Tracer
}

// builds the consumer for one bulk operation, pairing the operation's canonical identity
// with the entity service's Execute method
func NewBulkOperationConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	op messaging.BulkOperation,
	execute BulkOperationExecutor,
) *BulkOperationConsumer {
	return &BulkOperationConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		queue:         op.Queue(),
		handler:       op.Handler(),
		name:          op.String(),
		execute:       execute,
		tracer:        tracing.GetTracer("core-service." + op.String() + "_consumer"),
	}
}

func (c *BulkOperationConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, c.queue, c.inboxConsumer.Wrap(c.handler, c.handleMessage))
}

func (c *BulkOperationConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer."+c.name,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[%s] Failed to unmarshal message: %v", c.name, err)
		span.RecordError(err)
		return err
	}

	var event domain.BulkOperationJobEvent
	if err := json.Unmarshal(amqpMsg.Data, &event); err != nil {
		log.Printf("[%s] Failed to unmarshal event payload: %v", c.name, err)
		span.RecordError(err)
		return err
	}

	span.SetAttributes(attribute.String("bulk_operation.job_id", event.JobID))

	// Restore the originating identity: the write path runs for the same account and is
	// attributed to the same actor that enqueued the command. The authority to do the work
	// was settled when that request was accepted.
	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}

	// Guard the nil check explicitly: returning a typed-nil *apierror.APIError through the
	// error interface would read as non-nil.
	if apiErr := c.execute(ctx, event); apiErr != nil {
		span.RecordError(apiErr)

		// Only a transient failure is worth another attempt.
		// Returning the error hands the delivery back to the consumer's backoff retries,
		// and dead-letters it once those are exhausted, so a blip resolves itself and a
		// longer outage leaves a message to replay.
		if apiErr.IsTransient {
			log.Printf("[%s] Bulk operation job %s failed transiently, retrying: %v", c.name, event.JobID, apiErr)
			return apiErr
		}

		// A deterministic failure by definition will fail identically.
		log.Printf("[%s] Bulk operation job %s failed permanently: %v", c.name, event.JobID, apiErr)
		return nil
	}

	return nil
}
