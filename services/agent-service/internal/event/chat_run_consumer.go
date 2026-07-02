package event

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

// ChatRunConsumer handles chat-run commands from notification-service: create a chat-linked agent run and execute it. Inbox-deduped so a redelivery doesn't create a duplicate run.
type ChatRunConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	agentDefSvc   domain.AgentDefinitionSvc
	tracer        trace.Tracer
}

func NewChatRunConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, agentDefSvc domain.AgentDefinitionSvc) *ChatRunConsumer {
	return &ChatRunConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, domain.ServiceName),
		agentDefSvc:   agentDefSvc,
		tracer:        tracing.GetTracer("agent-service.chat_run_consumer"),
	}
}

func (c *ChatRunConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.AgentCmdChatRunQueue,
		c.inboxConsumer.Wrap("agent.chat_run", c.handleChatRun))
}

func (c *ChatRunConsumer) handleChatRun(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.chat_run", trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmarshal AMQP message: %w", err)
	}

	var data messaging.AgentChatRunData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmarshal chat run data: %w", err)
	}

	history := make([]domain.ChatHistoryMessage, 0, len(data.History))
	for _, h := range data.History {
		history = append(history, domain.ChatHistoryMessage{Role: h.Role, Name: h.Name, AgentConfigID: h.AgentConfigID, Body: h.Body})
	}

	if apiErr := c.agentDefSvc.CreateChatRun(ctx, domain.ChatRunInput{
		AccountID:         data.AccountID,
		AgentDefinitionID: data.AgentDefinitionID,
		ConversationID:    data.ConversationID,
		TriggerMessageID:  data.TriggerMessageID,
		Message:           data.Message,
		History:           history,
		ContinueRunID:     data.ContinueRunID,
	}); apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}
