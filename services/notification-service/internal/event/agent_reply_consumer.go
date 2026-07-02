package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// AgentReplyConsumer ingests an agent's chat reply (emitted by agent-service after a chat-triggered run) and posts it into the conversation as the agent participant via the chat service. Inbox-deduped on the message id; PostAgentReply is additionally idempotent on the client message id.
type AgentReplyConsumer struct {
	rabbitmq      messaging.MessageBroker
	chatSvc       domain.ConversationSvc
	inboxConsumer *messaging.InboxConsumer
	tracer        trace.Tracer
}

func NewAgentReplyConsumer(rabbitmq messaging.MessageBroker, chatSvc domain.ConversationSvc, inboxRepo messaging.InboxRepo, tracer trace.Tracer) *AgentReplyConsumer {
	return &AgentReplyConsumer{
		rabbitmq:      rabbitmq,
		chatSvc:       chatSvc,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "notification-service"),
		tracer:        tracer,
	}
}

func (c *AgentReplyConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.NotificationCmdAgentReplyQueue, c.inboxConsumer.Wrap("notification.agent_reply", c.handleAgentReply))
}

func (c *AgentReplyConsumer) handleAgentReply(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.handle_agent_reply",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("notification.agent_reply: failed to unmarshal envelope: %v", err)
		span.RecordError(err)
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	if amqpMsg.RequestID != "" {
		ctx = appctx.WithRequestID(ctx, amqpMsg.RequestID)
	}

	var data messaging.AgentReplyData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("notification.agent_reply: failed to unmarshal payload: %v", err)
		span.RecordError(err)
		return err
	}

	// A tool-approval notice posts as a senderless system_event timeline divider, not an agent bubble.
	if data.ApprovalEvent {
		if apiErr := c.chatSvc.PostConversationSystemEvent(ctx, domain.SystemEventInput{
			AccountID:       data.AccountID,
			ConversationID:  data.ConversationID,
			EventType:       "tool_approval",
			Body:            data.Body,
			ClientMessageID: data.ClientMessageID,
		}); apiErr != nil {
			span.RecordError(apiErr)
			return apiErr
		}
		return nil
	}

	in := domain.AgentReplyInput{
		AccountID:        data.AccountID,
		ConversationID:   data.ConversationID,
		AgentConfigID:    data.AgentConfigID,
		AgentName:        data.AgentName,
		AgentRunID:       data.AgentRunID,
		Body:             data.Body,
		ClientMessageID:  data.ClientMessageID,
		MessageID:        data.MessageID,
		ReplyToMessageID: data.ReplyToMessageID,
	}
	// A streaming reply spans two phases: "start" posts the empty bubble; "final" finalizes it. Phase "" is the legacy single-shot create-and-complete.
	var apiErr *apierror.APIError
	switch data.Phase {
	case "start":
		apiErr = c.chatSvc.PostAgentReplyStart(ctx, in)
	case "final":
		apiErr = c.chatSvc.PostAgentReplyComplete(ctx, in)
	default:
		apiErr = c.chatSvc.PostAgentReply(ctx, in)
	}
	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}
