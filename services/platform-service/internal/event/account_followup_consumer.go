package event

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
)

// AccountFollowupConsumer schedules a follow-up for each new self-serve registration and drafts the ones that come due.
type AccountFollowupConsumer struct {
	rabbitmq      messaging.MessageBroker
	svc           domain.AccountFollowupSvc
	inboxConsumer *messaging.InboxConsumer
	tracer        trace.Tracer
}

func NewAccountFollowupConsumer(
	rabbitmq messaging.MessageBroker,
	svc domain.AccountFollowupSvc,
	inboxRepo messaging.InboxRepo,
	tracer trace.Tracer,
) *AccountFollowupConsumer {
	return &AccountFollowupConsumer{
		rabbitmq:      rabbitmq,
		svc:           svc,
		tracer:        tracer,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, domain.ServiceName),
	}
}

// ListenSchedule consumes registrations. It runs whether or not drafting is enabled, so turning drafting on later still finds every registration since.
func (c *AccountFollowupConsumer) ListenSchedule(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(
		ctx,
		messaging.PlatformCmdScheduleAccountFollowupQueue,
		c.inboxConsumer.Wrap("platform.schedule_account_followup", c.handleSchedule),
	)
}

// ListenDraft consumes due follow-ups. Each draft is one LLM call; signups are few, so one at a time is plenty.
func (c *AccountFollowupConsumer) ListenDraft(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(
		ctx,
		messaging.PlatformCmdDraftAccountFollowupQueue,
		c.inboxConsumer.Wrap("platform.draft_account_followup", c.handleDraft),
	)
}

func (c *AccountFollowupConsumer) handleSchedule(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "event.account_followup.schedule")
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		return err
	}
	var payload messaging.AccountFollowupScheduleData
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		return err
	}

	if apiErr := c.svc.Schedule(ctx, payload); apiErr != nil {
		return apiErr
	}
	return nil
}

func (c *AccountFollowupConsumer) handleDraft(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "event.account_followup.draft")
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		return err
	}
	var payload messaging.AccountFollowupDraftData
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		return err
	}

	if apiErr := c.svc.Draft(ctx, payload.FollowupID); apiErr != nil {
		return apiErr
	}
	return nil
}
