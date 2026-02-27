package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/billing-service/internal/domain"
	billinggrpc "github.com/augno/api/services/billing-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/billing-service/internal/infrastructure/repository"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type StripeWebhookConsumer struct {
	rabbitmq           messaging.MessageBroker
	inboxConsumer      *messaging.InboxConsumer
	stripeEventLogRepo *repository.StripeEventLogRepo
	coreClient         *billinggrpc.BillingCoreClient
	stripeClient       domain.StripeClient
	tracer             trace.Tracer
}

func NewStripeWebhookConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	stripeEventLogRepo *repository.StripeEventLogRepo,
	coreClient *billinggrpc.BillingCoreClient,
	stripeClient domain.StripeClient,
) *StripeWebhookConsumer {
	return &StripeWebhookConsumer{
		rabbitmq:           rabbitmq,
		inboxConsumer:      messaging.NewInboxConsumer(inboxRepo, "billing-service"),
		stripeEventLogRepo: stripeEventLogRepo,
		coreClient:         coreClient,
		stripeClient:       stripeClient,
		tracer:             tracing.GetTracer("billing-service.stripe_webhook_consumer"),
	}
}

func (c *StripeWebhookConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.BillingEventStripeWebhookQueue,
		c.inboxConsumer.Wrap("billing.stripe_webhook", c.handleStripeWebhook))
}

// stripeEventEnvelope is a lightweight struct for extracting fields from a raw
// Stripe event JSON payload without pulling in the full Stripe SDK types.
type stripeEventEnvelope struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// stripeEventData wraps the data.object as raw JSON for flexible parsing.
type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

// objectIDExtractor extracts just the ID from a data.object payload.
type objectIDExtractor struct {
	ID string `json:"id"`
}

func (c *StripeWebhookConsumer) handleStripeWebhook(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.stripe_webhook",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[stripe_webhook] Failed to unmarshal AMQP message: %v", err)
		span.RecordError(err)
		return err
	}

	var event stripeEventEnvelope
	if err := json.Unmarshal(amqpMsg.Data, &event); err != nil {
		log.Printf("[stripe_webhook] Failed to unmarshal Stripe event: %v", err)
		span.RecordError(err)
		return err
	}

	// Parse data.object to extract the object ID
	var data stripeEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		log.Printf("[stripe_webhook] Failed to unmarshal event data: %v", err)
		span.RecordError(err)
		return err
	}

	var objID objectIDExtractor
	if err := json.Unmarshal(data.Object, &objID); err != nil {
		log.Printf("[stripe_webhook] Failed to extract object ID: %v", err)
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.String("stripe.event_id", event.ID),
		attribute.String("stripe.event_type", event.Type),
		attribute.String("stripe.object_id", objID.ID),
	)

	// Stripe-level dedup: check if this event was already fully processed
	// (handles Stripe webhook retries arriving as different AMQP messages)
	exists, err := c.stripeEventLogRepo.Exists(ctx, event.ID, objID.ID)
	if err != nil {
		log.Printf("[stripe_webhook] Failed to check stripe event log: %v", err)
		span.RecordError(err)
		return err
	}
	if exists {
		log.Printf("[stripe_webhook] Stripe event %s (type=%s, object=%s) already processed, skipping", event.ID, event.Type, objID.ID)
		return nil
	}

	// Dispatch to handler based on event type
	var handlerErr error
	switch event.Type {
	case "customer.subscription.updated":
		handlerErr = c.handleSubscriptionUpdated(ctx, event.ID, data.Object)
	case "customer.subscription.deleted":
		handlerErr = c.handleSubscriptionDeleted(ctx, event.ID, data.Object)
	case "customer.deleted":
		handlerErr = c.handleCustomerDeleted(ctx, event.ID, data.Object)
	case "invoice.payment_failed":
		handlerErr = c.handleInvoicePaymentFailed(ctx, event.ID, data.Object)
	default:
		log.Printf("[stripe_webhook] Unhandled event type %s (event=%s)", event.Type, event.ID)
		return nil
	}

	if handlerErr != nil {
		return handlerErr
	}

	// Mark event as fully processed after successful handler completion.
	// If this insert fails, the handler will safely re-execute on retry
	// because the gRPC mutations are idempotent.
	if insertErr := c.stripeEventLogRepo.Insert(ctx, event.ID, event.Type, objID.ID); insertErr != nil {
		log.Printf("[stripe_webhook] Failed to insert stripe event log: %v", insertErr)
		span.RecordError(insertErr)
		return insertErr
	}

	return nil
}
