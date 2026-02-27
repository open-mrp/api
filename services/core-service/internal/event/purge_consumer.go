package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/core-service/internal/infrastructure/repository"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PurgeConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	purgeRepo     *repository.PurgeRepo
	tracer        trace.Tracer
}

func NewPurgeConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, purgeRepo *repository.PurgeRepo) *PurgeConsumer {
	return &PurgeConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		purgeRepo:     purgeRepo,
		tracer:        tracing.GetTracer("core-service.purge_consumer"),
	}
}

func (c *PurgeConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdPurgeAccountDataQueue,
		c.inboxConsumer.Wrap("core.purge_account_data", c.handlePurgeMessage))
}

func (c *PurgeConsumer) handlePurgeMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.purge_account_data",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[purge] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var payload PurgeAccountDataPayload
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		log.Printf("[purge] Failed to unmarshal purge payload: %v", err)
		span.RecordError(err)
		return err
	}

	if payload.AccountID == "" {
		log.Printf("[purge] Empty account ID in purge payload")
		return nil
	}

	span.SetAttributes(attribute.String("purge.account_id", payload.AccountID))
	log.Printf("[purge] Starting account data purge for account %s", payload.AccountID)

	if err := c.purgeRepo.VerifyAccountIsSandboxOrDeleted(ctx, payload.AccountID); err != nil {
		log.Printf("[purge] BLOCKED: %v", err)
		span.RecordError(err)
		return err
	}

	if err := c.purgeRepo.PurgeAccountData(ctx, payload.AccountID); err != nil {
		log.Printf("[purge] Failed to purge account data for %s: %v", payload.AccountID, err)
		span.RecordError(err)
		return err
	}

	log.Printf("[purge] Successfully purged account data for account %s", payload.AccountID)
	return nil
}
