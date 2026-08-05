package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/stripesync"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SyncStripeCustomerConsumer mirrors a customer create/update onto the account's connected Stripe integration.
//
// The mutation itself already committed — this is Stripe catching up behind it. Running out-of-band is what keeps a Stripe outage from failing a customer edit that has nothing to do with payments.
type SyncStripeCustomerConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	stripeSync    stripesync.Service
	tracer        trace.Tracer
}

func NewSyncStripeCustomerConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	stripeSync stripesync.Service,
) *SyncStripeCustomerConsumer {
	return &SyncStripeCustomerConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		stripeSync:    stripeSync,
		tracer:        tracing.GetTracer("core-service.sync_stripe_customer_consumer"),
	}
}

func (c *SyncStripeCustomerConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdSyncStripeCustomerQueue,
		c.inboxConsumer.Wrap("core.sync_stripe_customer", c.handleMessage))
}

func (c *SyncStripeCustomerConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.sync_stripe_customer",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[sync_stripe_customer] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var evt domain.SyncStripeCustomerEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		log.Printf("[sync_stripe_customer] Failed to unmarshal event payload: %v", err)
		span.RecordError(err)
		return err
	}

	if evt.OwnerAccountID == "" || evt.CustomerAccountID == "" {
		log.Printf("[sync_stripe_customer] Missing owner or customer account ID in event")
		return nil
	}

	span.SetAttributes(
		attribute.String("customer.account_id", evt.CustomerAccountID),
		attribute.String("customer.owner_account_id", evt.OwnerAccountID),
	)

	// Transient failures (Stripe rate limits, 5xx) are returned so the inbox retries.
	// Permanent ones — a customer deleted before the message was picked up, credentials
	// that no longer decrypt — are logged and swallowed: retrying them forever would
	// poison-loop the queue behind one unfixable customer.
	apiErr := c.stripeSync.SyncCustomer(ctx, evt.OwnerAccountID, evt.CustomerAccountID)
	if apiErr == nil {
		return nil
	}
	if apiErr.IsTransient {
		span.RecordError(apiErr)
		return apiErr
	}
	log.Printf("[sync_stripe_customer] Stripe sync permanently failed for customer %s (account %s): %v",
		evt.CustomerAccountID, evt.OwnerAccountID, apiErr)
	return nil
}
