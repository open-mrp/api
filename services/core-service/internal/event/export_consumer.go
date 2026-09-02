package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// renders an accepted export and settles the job tracking it. Each entity's service
// supplies one — its BuildExport method value.
type ExportRenderer func(context.Context, domain.BulkOperationJobEvent) *apierror.APIError

// consumes one export command. An export carries the same {job_id} payload a bulk
// operation does, so the shape mirrors BulkOperationConsumer; only the work differs.
type ExportConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	queue         string
	handler       string
	name          string
	render        ExportRenderer
	tracer        trace.Tracer
}

// builds the consumer for one export, pairing its canonical identity with the service
// method that renders and uploads the file
func NewExportConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	op messaging.ExportOperation,
	render ExportRenderer,
) *ExportConsumer {
	return &ExportConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service").WithLeaseSeconds(jobInboxLeaseSeconds),
		queue:         op.Queue(),
		handler:       op.Handler(),
		name:          op.String(),
		render:        render,
		tracer:        tracing.GetTracer("core-service." + op.String() + "_consumer"),
	}
}

func (c *ExportConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, c.queue, c.inboxConsumer.Wrap(c.handler, c.handleMessage))
}

func (c *ExportConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
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

	span.SetAttributes(attribute.String("export.job_id", event.JobID))

	// Restore the originating identity: the render runs for the same account and reads
	// only what that caller was already allowed to see.
	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}

	// Guard the nil check explicitly: returning a typed-nil *apierror.APIError through the
	// error interface would read as non-nil.
	if apiErr := c.render(ctx, event); apiErr != nil {
		span.RecordError(apiErr)

		// Only a transient failure is worth another attempt. Returning the error hands the
		// delivery to the consumer's backoff retries and dead-letters it once those are
		// exhausted, so a blip resolves itself and a longer outage leaves a message to replay.
		if apiErr.IsTransient {
			log.Printf("[%s] Export job %s failed transiently, retrying: %v", c.name, event.JobID, apiErr)
			return apiErr
		}

		// A deterministic failure will fail identically, and the job already records it.
		log.Printf("[%s] Export job %s failed permanently: %v", c.name, event.JobID, apiErr)
		return nil
	}

	return nil
}
