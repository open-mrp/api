package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Names the Stripe meter that bills invoice volume; it must match the meter configured in the Stripe dashboard.
const invoiceCreatedMeterEventName = "openmrp_invoices"

// Declares the account usage lookups the invoice-created handler needs.
type InvoiceCreatedAccountUsageRepo interface {
	GetStripeCustomerIDByAccountID(ctx context.Context, accountID string) (*string, *apierror.APIError)
}

// Declares the Stripe operations the invoice-created handler needs.
type InvoiceCreatedStripeClient interface {
	ReportMeterEvent(ctx context.Context, eventName, stripeCustomerID string, value int, idempotencyKey string) error
}

type InvoiceCreatedHandler struct {
	tracer       trace.Tracer
	usageRepo    InvoiceCreatedAccountUsageRepo
	stripeClient InvoiceCreatedStripeClient
}

func NewInvoiceCreatedHandler(
	usageRepo InvoiceCreatedAccountUsageRepo,
	stripeClient InvoiceCreatedStripeClient,
) *InvoiceCreatedHandler {
	return &InvoiceCreatedHandler{
		tracer:       tracing.GetTracer("billing-service.invoice_created_handler"),
		usageRepo:    usageRepo,
		stripeClient: stripeClient,
	}
}

// Reports one invoice-created usage event to Stripe, skipping accounts with no Stripe customer (free tier).
// The AMQP message ID is the Stripe idempotency key, so redelivery does not double-count the invoice.
func (h *InvoiceCreatedHandler) Handle(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := h.tracer.Start(ctx, "handler.invoice_created",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[invoice_created] Failed to unmarshal AMQP message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.InvoiceCreatedReportData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[invoice_created] Failed to unmarshal invoice created data: %v", err)
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.String("billing.account_id", data.AccountID),
		attribute.String("billing.invoice_id", data.InvoiceID),
	)

	stripeCustomerID, apiErr := h.usageRepo.GetStripeCustomerIDByAccountID(ctx, data.AccountID)
	if apiErr != nil {
		return fmt.Errorf("failed to get Stripe customer ID: %w", apiErr)
	}
	if stripeCustomerID == nil {
		log.Printf("[invoice_created] No Stripe customer for account %s (free tier), skipping meter event", data.AccountID)
		return nil
	}

	span.SetAttributes(attribute.String("billing.stripe_customer_id", *stripeCustomerID))

	if err := h.stripeClient.ReportMeterEvent(ctx, invoiceCreatedMeterEventName, *stripeCustomerID, 1, msg.MessageId); err != nil {
		return fmt.Errorf("failed to report meter event: %w", err)
	}

	log.Printf("[invoice_created] Reported invoice %s for account %s (customer %s)",
		data.InvoiceID, data.AccountID, *stripeCustomerID)

	return nil
}
