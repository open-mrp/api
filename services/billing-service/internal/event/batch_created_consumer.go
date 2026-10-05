package event

import (
	"context"

	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

type BatchCreatedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	batchHandler  *BatchCreatedHandler
	tracer        trace.Tracer
}

func NewBatchCreatedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	batchHandler *BatchCreatedHandler,
) *BatchCreatedConsumer {
	return &BatchCreatedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "billing-service"),
		batchHandler:  batchHandler,
		tracer:        tracing.GetTracer("billing-service.batch_created_consumer"),
	}
}

func (c *BatchCreatedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.BillingCmdReportBatchCreatedQueue,
		c.inboxConsumer.Wrap("billing.report_batch_created", c.batchHandler.Handle))
}
