package event

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/augno/api/services/billing-service/internal/domain"
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
	stripeEventLogRepo EventLogRepo
	coreClient         WebhookCoreClient
	stripeClient       domain.StripeClient
	notificationClient WebhookNotificationClient
	accountUsageRepo   WebhookAccountUsageRepo
	tracer             trace.Tracer
}

func NewStripeWebhookConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	stripeEventLogRepo EventLogRepo,
	coreClient WebhookCoreClient,
	stripeClient domain.StripeClient,
	notificationClient WebhookNotificationClient,
	accountUsageRepo WebhookAccountUsageRepo,
) *StripeWebhookConsumer {
	return &StripeWebhookConsumer{
		rabbitmq:           rabbitmq,
		inboxConsumer:      messaging.NewInboxConsumer(inboxRepo, "billing-service"),
		stripeEventLogRepo: stripeEventLogRepo,
		coreClient:         coreClient,
		stripeClient:       stripeClient,
		notificationClient: notificationClient,
		accountUsageRepo:   accountUsageRepo,
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

	// v2 thin events use related_object instead of data.object
	RelatedObject *relatedObject `json:"related_object,omitempty"`
}

// relatedObject holds the reference from a v2 thin event to the actual object.
type relatedObject struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
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

	// Resolve the object data: v2 thin events use related_object, v1 events use data.object
	var objectData json.RawMessage
	var objID objectIDExtractor
	isV2 := strings.HasPrefix(event.Type, "v2.")

	if isV2 && event.RelatedObject != nil {
		// v2 thin event: fetch the full object from the related_object URL
		objID.ID = event.RelatedObject.ID
		fetched, fetchErr := c.stripeClient.FetchObject(ctx, event.RelatedObject.URL)
		if fetchErr != nil {
			log.Printf("[stripe_webhook] Failed to fetch v2 related object %s: %v", event.RelatedObject.URL, fetchErr)
			span.RecordError(fetchErr)
			return fetchErr
		}
		objectData = fetched
	} else {
		// v1 event: parse data.object inline
		var data stripeEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			log.Printf("[stripe_webhook] Failed to unmarshal event data: %v", err)
			span.RecordError(err)
			return err
		}
		objectData = data.Object

		if err := json.Unmarshal(data.Object, &objID); err != nil {
			log.Printf("[stripe_webhook] Failed to extract object ID: %v", err)
			span.RecordError(err)
			return err
		}
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
	case "customer.deleted":
		handlerErr = c.handleCustomerDeleted(ctx, event.ID, objectData)

	// v2 pricing plan subscription events
	case "v2.billing.pricing_plan_subscription.servicing_activated":
		handlerErr = c.handleServicingActivated(ctx, event.ID, objectData)
	case "v2.billing.pricing_plan_subscription.servicing_canceled":
		handlerErr = c.handleServicingCanceled(ctx, event.ID, objectData)
	case "v2.billing.pricing_plan_subscription.collection_paused":
		handlerErr = c.handleCollectionPaused(ctx, event.ID, objectData)
	case "v2.billing.pricing_plan_subscription.collection_current":
		handlerErr = c.handleCollectionCurrent(ctx, event.ID, objectData)
	case "v2.billing.cadence.errored":
		handlerErr = c.handleCadenceErrored(ctx, event.ID, objectData)
	case "v2.billing.pricing_plan_subscription.servicing_paused":
		handlerErr = c.handleServicingPaused(ctx, event.ID, objectData)
	case "v2.billing.pricing_plan_subscription.collection_awaiting_customer_action":
		handlerErr = c.handleCollectionAwaitingCustomerAction(ctx, event.ID, objectData)
	case "v2.billing.cadence.billed":
		handlerErr = c.handleCadenceBilled(ctx, event.ID, objectData)
	case "v2.billing.cadence.canceled":
		handlerErr = c.handleCadenceCanceled(ctx, event.ID, objectData)
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
