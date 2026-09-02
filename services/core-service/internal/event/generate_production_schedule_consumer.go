package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GenerateProductionScheduleConsumer runs the solve the generation cadence queued.
//
// The solve lives here rather than in the cadence tick because it takes minutes on a real tenant: running it inside the scheduler lease would block every other account behind whichever one is currently solving.
type GenerateProductionScheduleConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	scheduleSvc   domain.ProductionScheduleSvc
	repos         domain.RepoFactory
	tracer        trace.Tracer
}

func NewGenerateProductionScheduleConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	scheduleSvc domain.ProductionScheduleSvc,
	repos domain.RepoFactory,
) *GenerateProductionScheduleConsumer {
	return &GenerateProductionScheduleConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service").WithLeaseSeconds(jobInboxLeaseSeconds),
		scheduleSvc:   scheduleSvc,
		repos:         repos,
		tracer:        tracing.GetTracer("core-service.generate_production_schedule_consumer"),
	}
}

func (c *GenerateProductionScheduleConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdGenerateProductionScheduleQueue,
		c.inboxConsumer.Wrap("core.generate_production_schedule", c.handleMessage))
}

func (c *GenerateProductionScheduleConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.generate_production_schedule",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[generate_production_schedule] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.GenerateProductionScheduleData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[generate_production_schedule] Failed to unmarshal payload: %v", err)
		span.RecordError(err)
		return err
	}

	if data.AccountID == "" || data.ScheduleID == "" {
		// A malformed message can never succeed, so it is dropped rather than retried forever.
		log.Printf("[generate_production_schedule] Missing account or schedule ID; dropping")
		return c.inboxConsumer.Discard(ctx, "missing account or schedule id")
	}

	span.SetAttributes(
		attribute.String("production_schedule.id", data.ScheduleID),
		attribute.String("production_schedule.account_id", data.AccountID),
	)

	// The solve reads and writes through the repo factory, and the outbox publisher used by publish reads its factory from the context.
	ctx = WithRepos(ctx, c.repos)

	if apiErr := c.scheduleSvc.RunScheduledGeneration(ctx, domain.RunScheduledGenerationParams{
		AccountID:    data.AccountID,
		ScheduleID:   data.ScheduleID,
		PlanningAsOf: data.PlanningAsOf,
		AutoPublish:  data.AutoPublish,
	}); apiErr != nil {
		log.Printf("[generate_production_schedule] Generation failed for %s: %v", data.ScheduleID, apiErr)
		span.RecordError(apiErr)
		// The version has already been marked failed with the reason, so the merchant can see what happened. Returning nil stops an endless redelivery of a solve that will fail the same way every time.
		return nil
	}

	return nil
}
