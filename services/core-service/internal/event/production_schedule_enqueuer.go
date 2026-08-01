package event

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var scheduleEnqueuerTracer = tracing.GetTracer("core-service.production_schedule_enqueuer")

// outboxProductionScheduleEnqueuer writes the generate command to the outbox rather than publishing directly, so the placeholder schedule row and the message that will fill it in commit together. Publishing directly would let the message escape a transaction that then rolled back, and the consumer would solve into a row that does not exist.
type outboxProductionScheduleEnqueuer struct{}

func NewOutboxProductionScheduleEnqueuer() domain.ProductionScheduleEnqueuer {
	return &outboxProductionScheduleEnqueuer{}
}

func (p *outboxProductionScheduleEnqueuer) EnqueueGeneration(ctx context.Context, params domain.EnqueueGenerationParams) *apierror.APIError {
	ctx, span := scheduleEnqueuerTracer.Start(ctx, "event.production_schedule_enqueuer.enqueue_generation")
	defer span.End()

	repos, ok := GetReposFromContext(ctx)
	if !ok {
		return tracing.Trace(span, apierror.NewInternalError(nil, "RepoFactory not found in context for outbox publisher."))
	}

	dataJSON, err := json.Marshal(messaging.GenerateProductionScheduleData{
		AccountID:    params.AccountID,
		ScheduleID:   params.ScheduleID,
		PlanningAsOf: params.PlanningAsOf,
		AutoPublish:  params.AutoPublish,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal generate schedule data."))
	}

	if _, err := repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreCmdGenerateProductionSchedule),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdGenerateProductionSchedule),
		Payload:     contracts.AmqpMessage{Data: dataJSON},
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create outbox message."))
	}

	return nil
}
