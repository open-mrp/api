package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

type RunConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	runner        domain.RunnerSvc
	tracer        trace.Tracer
}

func NewRunConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, runner domain.RunnerSvc) *RunConsumer {
	return &RunConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, domain.ServiceName),
		runner:        runner,
		tracer:        tracing.GetTracer("agent-service.run_consumer"),
	}
}

func (c *RunConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.AgentCmdExecuteRunQueue,
		c.inboxConsumer.Wrap("agent.execute_run", c.handleExecuteRun))
}

func (c *RunConsumer) ListenContinueRun(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.AgentCmdContinueRunQueue,
		c.inboxConsumer.Wrap("agent.continue_run", c.handleContinueRun))
}

func (c *RunConsumer) handleExecuteRun(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.execute_run",
		trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmarshal AMQP message: %w", err)
	}

	var data messaging.AgentExecuteRunData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmarshal execute run data: %w", err)
	}

	slog.Info("Processing agent run",
		"run_id", data.AgentRunID,
		"config_id", data.AgentConfigID,
		"account_id", data.AccountID,
		"trigger_type", data.TriggerType,
	)

	if err := c.runner.ExecuteRun(ctx, data.AgentRunID, data.AgentConfigID, data.AccountID, data.TriggerType); err != nil {
		span.RecordError(err)
		slog.Error("Agent run failed",
			"run_id", data.AgentRunID,
			"error", err,
		)
		// Don't return error — the run is already marked as failed in the DB.
		// Returning an error would cause the message to be requeued/retried,
		// but the runner already handles failure state.
		return nil
	}

	return nil
}

func (c *RunConsumer) handleContinueRun(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.continue_run",
		trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmarshal AMQP message: %w", err)
	}

	var data messaging.AgentContinueRunData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unmarshal continue run data: %w", err)
	}

	slog.Info("Processing agent continue run",
		"run_id", data.AgentRunID,
		"account_id", data.AccountID,
	)

	if err := c.runner.ContinueRun(ctx, data.AgentRunID, data.AccountID, data.Message, data.ApprovedToolSlugs, data.AllowedToolSlugs, data.ActorID, data.ActorType, data.ActorName); err != nil {
		span.RecordError(err)
		slog.Error("Agent continue run failed",
			"run_id", data.AgentRunID,
			"error", err,
		)
		return nil
	}

	return nil
}
