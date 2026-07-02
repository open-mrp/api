package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MessagingConsumer processes notification.cmd.fanout intents into per-recipient notification rows (plus realtime pushes) via the messaging service.
type MessagingConsumer struct {
	rabbitmq      messaging.MessageBroker
	messagingSvc  domain.MessagingSvc
	inboxConsumer *messaging.InboxConsumer
	tracer        trace.Tracer
}

func NewMessagingConsumer(rabbitmq messaging.MessageBroker, messagingSvc domain.MessagingSvc, inboxRepo messaging.InboxRepo, tracer trace.Tracer) *MessagingConsumer {
	return &MessagingConsumer{
		rabbitmq:      rabbitmq,
		messagingSvc:  messagingSvc,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "notification-service"),
		tracer:        tracer,
	}
}

func (c *MessagingConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.NotificationCmdFanoutQueue, c.inboxConsumer.Wrap("notification.fanout", c.handleFanout))
}

func (c *MessagingConsumer) handleFanout(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.handle_fanout",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("notification.fanout: failed to unmarshal envelope: %v", err)
		span.RecordError(err)
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	var data messaging.AlertFanoutData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("notification.fanout: failed to unmarshal payload: %v", err)
		span.RecordError(err)
		return err
	}

	// Use the app-level message id as the idempotency seed so redelivery re-creates the same per-recipient notification ids (which then collide on the PK and are skipped).
	seed := amqpMsg.MessageID
	if seed == "" {
		seed = msg.MessageId
	}

	if apiErr := c.messagingSvc.FanOut(ctx, seed, data); apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}
