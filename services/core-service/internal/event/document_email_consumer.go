package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DocumentEmailConsumer mails the customer-facing document a business event has made deliverable:
// the invoice behind a shipment, a sales order's acknowledgement, a purchase order's submission.
//
// The three facts are separate events on separate queues so a later subscriber to any of them binds
// beside this one rather than competing with it, but one consumer handles all three: the reaction is
// the same shape each time — render the document in core-service, hand the send to
// notification-service — and it is the reaction, not the fact, that this type is about.
//
// Rendering lives here rather than in the publisher because the publisher may be the dashboard API,
// which has neither the invoice PDF renderer nor a broker connection of its own.
type DocumentEmailConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	utilsSvc      domain.UtilsSvc
	tracer        trace.Tracer
}

func NewDocumentEmailConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	utilsSvc domain.UtilsSvc,
) *DocumentEmailConsumer {
	return &DocumentEmailConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		utilsSvc:      utilsSvc,
		tracer:        tracing.GetTracer("core-service.document_email_consumer"),
	}
}

func (c *DocumentEmailConsumer) Listen(ctx context.Context) error {
	if err := c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventInvoiceIssuedEmailQueue,
		c.inboxConsumer.Wrap("core.invoice_issued_email", c.handleInvoiceIssued)); err != nil {
		return err
	}

	if err := c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventSalesOrderAcknowledgedEmailQueue,
		c.inboxConsumer.Wrap("core.sales_order_acknowledged_email", c.handleSalesOrderAcknowledged)); err != nil {
		return err
	}

	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventPurchaseOrderSubmittedEmailQueue,
		c.inboxConsumer.Wrap("core.purchase_order_submitted_email", c.handlePurchaseOrderSubmitted))
}

func (c *DocumentEmailConsumer) handleInvoiceIssued(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.invoice_issued_email",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var evt domain.InvoiceIssuedEvent
	ctx, accountID, ok := decodeDocumentEvent(ctx, span, "invoice_issued", msg, &evt)
	if !ok {
		return nil
	}
	if evt.InvoiceID == "" {
		log.Printf("[invoice_issued] Empty invoice ID in event")
		return nil
	}

	span.SetAttributes(
		attribute.String("invoice.id", evt.InvoiceID),
		attribute.String("invoice.account_id", accountID),
	)

	if apiErr := c.utilsSvc.SendInvoiceEmail(ctx, domain.SendInvoiceEmailParams{
		AccountID:     accountID,
		InvoiceID:     evt.InvoiceID,
		EmailCustomer: evt.EmailCustomer,
		EmailSalesRep: evt.EmailSalesRep,
	}); apiErr != nil {
		log.Printf("[invoice_issued] Failed to email invoice %s: %s", evt.InvoiceID, apiErr.PublicMessage)
		return apiErr
	}

	return nil
}

func (c *DocumentEmailConsumer) handleSalesOrderAcknowledged(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.sales_order_acknowledged_email",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var evt domain.SalesOrderAcknowledgedEvent
	ctx, accountID, ok := decodeDocumentEvent(ctx, span, "sales_order_acknowledged", msg, &evt)
	if !ok {
		return nil
	}
	if evt.SalesOrderID == "" {
		log.Printf("[sales_order_acknowledged] Empty sales order ID in event")
		return nil
	}

	span.SetAttributes(
		attribute.String("sales_order.id", evt.SalesOrderID),
		attribute.String("sales_order.account_id", accountID),
	)

	if apiErr := c.utilsSvc.SendSalesOrderAcknowledgement(ctx, domain.SendSalesOrderAcknowledgementParams{
		AccountID:    accountID,
		SalesOrderID: evt.SalesOrderID,
	}); apiErr != nil {
		log.Printf("[sales_order_acknowledged] Failed to email acknowledgement for order %s: %s", evt.SalesOrderID, apiErr.PublicMessage)
		return apiErr
	}

	return nil
}

func (c *DocumentEmailConsumer) handlePurchaseOrderSubmitted(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.purchase_order_submitted_email",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var evt domain.PurchaseOrderSubmittedEvent
	ctx, accountID, ok := decodeDocumentEvent(ctx, span, "purchase_order_submitted", msg, &evt)
	if !ok {
		return nil
	}
	if evt.PurchaseOrderID == "" {
		log.Printf("[purchase_order_submitted] Empty purchase order ID in event")
		return nil
	}

	span.SetAttributes(
		attribute.String("purchase_order.id", evt.PurchaseOrderID),
		attribute.String("purchase_order.account_id", accountID),
	)

	if apiErr := c.utilsSvc.SendPurchaseOrderSubmission(ctx, domain.SendPurchaseOrderSubmissionParams{
		AccountID:       accountID,
		PurchaseOrderID: evt.PurchaseOrderID,
	}); apiErr != nil {
		log.Printf("[purchase_order_submitted] Failed to email submission for order %s: %s", evt.PurchaseOrderID, apiErr.PublicMessage)
		return apiErr
	}

	return nil
}

// decodeDocumentEvent unwraps the envelope into payload and account, and puts the caller's identity
// and request id back on the context. A message that cannot yield an account is unprocessable rather
// than retryable, so it reports false and the caller acks it away instead of cycling it to the
// dead-letter queue.
//
// Restoring the identity matches ExportConsumer and BulkOperationConsumer, and carries the
// originating request through to the outbox rows this work produces. It is not what stops a sandbox
// account mailing real customers — the identity a publisher writes need not carry an AccountMode at
// all — that guard lives in notification-service, which asks the account directly.
func decodeDocumentEvent(ctx context.Context, span trace.Span, name string, msg amqp.Delivery, payload any) (context.Context, string, bool) {
	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[%s] Failed to unmarshal message: %v", name, err)
		span.RecordError(err)
		return ctx, "", false
	}

	if err := json.Unmarshal(amqpMsg.Data, payload); err != nil {
		log.Printf("[%s] Failed to unmarshal event payload: %v", name, err)
		span.RecordError(err)
		return ctx, "", false
	}

	if amqpMsg.Identity == nil || amqpMsg.Identity.Target == nil || amqpMsg.Identity.Target.AccountID == "" {
		log.Printf("[%s] No account ID in message identity", name)
		return ctx, "", false
	}

	ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	return ctx, amqpMsg.Identity.Target.AccountID, true
}
