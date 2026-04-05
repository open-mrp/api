package event

import (
	"context"

	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

type SeatChangeConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	seatHandler   *SeatChangeHandler
	tracer        trace.Tracer
}

func NewSeatChangeConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	seatHandler *SeatChangeHandler,
) *SeatChangeConsumer {
	return &SeatChangeConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "billing-service"),
		seatHandler:   seatHandler,
		tracer:        tracing.GetTracer("billing-service.seat_change_consumer"),
	}
}

func (c *SeatChangeConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.BillingCmdReportSeatChangeQueue,
		c.inboxConsumer.Wrap("billing.report_seat_change", c.seatHandler.Handle))
}
