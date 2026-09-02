package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type SeedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	seeder        *repository.SandboxSeeder
	tracer        trace.Tracer
}

func NewSeedConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, seeder *repository.SandboxSeeder) *SeedConsumer {
	return &SeedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service").WithLeaseSeconds(jobInboxLeaseSeconds),
		seeder:        seeder,
		tracer:        tracing.GetTracer("core-service.seed_consumer"),
	}
}

func (c *SeedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdSeedSandboxQueue,
		c.inboxConsumer.Wrap("core.seed_sandbox", c.handleSeedMessage))
}

func (c *SeedConsumer) handleSeedMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.seed_sandbox",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[seed] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var payload SeedSandboxPayload
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		log.Printf("[seed] Failed to unmarshal seed payload: %v", err)
		span.RecordError(err)
		return err
	}

	if payload.AccountID == "" {
		log.Printf("[seed] Empty account ID in seed payload")
		return c.inboxConsumer.Discard(ctx, "no account on seed payload")
	}

	span.SetAttributes(attribute.String("seed.account_id", payload.AccountID))
	log.Printf("[seed] Starting sandbox seed for account %s", payload.AccountID)

	if err := c.seeder.Seed(ctx, payload.AccountID); err != nil {
		log.Printf("[seed] Failed to seed account %s: %v", payload.AccountID, err)
		span.RecordError(err)
		return err
	}

	log.Printf("[seed] Successfully seeded sandbox account %s", payload.AccountID)
	return nil
}
