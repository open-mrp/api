package event

import (
	"context"
	"log"

	"encoding/json"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// CustomerRegisteredConsumer turns core.event.customer_registered events into a bell notification to the seller's customer-service support-route group, so the team can follow up on new portal registrations.
type CustomerRegisteredConsumer struct {
	rabbitmq      messaging.MessageBroker
	messagingSvc  domain.MessagingSvc
	inboxConsumer *messaging.InboxConsumer
	tracer        trace.Tracer
}

func NewCustomerRegisteredConsumer(rabbitmq messaging.MessageBroker, messagingSvc domain.MessagingSvc, inboxRepo messaging.InboxRepo, tracer trace.Tracer) *CustomerRegisteredConsumer {
	return &CustomerRegisteredConsumer{
		rabbitmq:      rabbitmq,
		messagingSvc:  messagingSvc,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "notification-service"),
		tracer:        tracer,
	}
}

func (c *CustomerRegisteredConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventCustomerRegisteredQueue, c.inboxConsumer.Wrap("core.customer_registered", c.handleMessage))
}

func (c *CustomerRegisteredConsumer) handleMessage(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.customer_registered",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("core.customer_registered: failed to unmarshal envelope: %v", err)
		span.RecordError(err)
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	var data messaging.CustomerRegisteredData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("core.customer_registered: failed to unmarshal payload: %v", err)
		span.RecordError(err)
		return err
	}

	if data.SellerAccountID == "" || data.CustomerAccountID == "" {
		log.Printf("core.customer_registered: missing seller or customer account id; skipping")
		return c.inboxConsumer.Discard(ctx, "missing seller or customer account id")
	}

	span.SetAttributes(
		attribute.String("customer.seller_account_id", data.SellerAccountID),
		attribute.String("customer.account_id", data.CustomerAccountID),
	)

	// Use the app-level message id as the idempotency seed so redelivery re-creates the same per-recipient notification ids (which then collide on the PK and are skipped).
	seed := amqpMsg.MessageID
	if seed == "" {
		seed = msg.MessageId
	}

	if apiErr := c.messagingSvc.NotifyCustomerRegistered(ctx, seed, data); apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}
