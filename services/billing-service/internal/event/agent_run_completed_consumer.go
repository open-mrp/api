package event

import (
	"context"

	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

type AgentRunCompletedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	tokenHandler  *AgentTokenBillingHandler
	tracer        trace.Tracer
}

func NewAgentRunCompletedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	tokenHandler *AgentTokenBillingHandler,
) *AgentRunCompletedConsumer {
	return &AgentRunCompletedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "billing-service"),
		tokenHandler:  tokenHandler,
		tracer:        tracing.GetTracer("billing-service.agent_run_completed_consumer"),
	}
}

func (c *AgentRunCompletedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.AgentEventRunCompletedQueue,
		c.inboxConsumer.Wrap("billing.agent_run_completed", c.tokenHandler.Handle))
}
