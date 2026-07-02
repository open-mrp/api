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

// AgentReplyPatchConsumer streams partial-body updates into an in-flight agent reply (emitted by agent-service as a chat-triggered run produces tokens). Each patch carries the full accumulated body, so it is idempotent and not inbox-deduped: the latest patch wins and the final reply reconciles the row. Best-effort — a failed patch is dropped (logged, acked) rather than retried, since a fresher patch or the finalize will supersede it.
type AgentReplyPatchConsumer struct {
	rabbitmq messaging.MessageBroker
	chatSvc  domain.ConversationSvc
	tracer   trace.Tracer
}

func NewAgentReplyPatchConsumer(rabbitmq messaging.MessageBroker, chatSvc domain.ConversationSvc, tracer trace.Tracer) *AgentReplyPatchConsumer {
	return &AgentReplyPatchConsumer{
		rabbitmq: rabbitmq,
		chatSvc:  chatSvc,
		tracer:   tracer,
	}
}

func (c *AgentReplyPatchConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.NotificationCmdAgentReplyPatchQueue, c.handlePatch)
}

func (c *AgentReplyPatchConsumer) handlePatch(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.handle_agent_reply_patch",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("notification.agent_reply_patch: failed to unmarshal envelope: %v", err)
		span.RecordError(err)
		return nil // malformed — drop (best-effort)
	}
	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	var data messaging.AgentReplyPatchData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("notification.agent_reply_patch: failed to unmarshal payload: %v", err)
		span.RecordError(err)
		return nil // malformed — drop (best-effort)
	}

	if apiErr := c.chatSvc.PostAgentReplyPatch(ctx, domain.AgentReplyPatchInput{
		AccountID:      data.AccountID,
		ConversationID: data.ConversationID,
		MessageID:      data.MessageID,
		Body:           data.Body,
	}); apiErr != nil {
		// Best-effort: log and ack rather than requeue. A later patch or the finalize reconciles.
		log.Printf("notification.agent_reply_patch: patch failed (dropped): %v", apiErr)
		span.RecordError(apiErr)
	}
	return nil
}
