package event

import (
	"context"

	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

type InvoiceCreatedConsumer struct {
	rabbitmq       messaging.MessageBroker
	inboxConsumer  *messaging.InboxConsumer
	invoiceHandler *InvoiceCreatedHandler
	tracer         trace.Tracer
}

func NewInvoiceCreatedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	invoiceHandler *InvoiceCreatedHandler,
) *InvoiceCreatedConsumer {
	return &InvoiceCreatedConsumer{
		rabbitmq:       rabbitmq,
		inboxConsumer:  messaging.NewInboxConsumer(inboxRepo, "billing-service"),
		invoiceHandler: invoiceHandler,
		tracer:         tracing.GetTracer("billing-service.invoice_created_consumer"),
	}
}

func (c *InvoiceCreatedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.BillingCmdReportInvoiceCreatedQueue,
		c.inboxConsumer.Wrap("billing.report_invoice_created", c.invoiceHandler.Handle))
}
