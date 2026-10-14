package event

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"

	"github.com/open-mrp/apikit/tracing"
	"github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/trace"
)

type AuditEventConsumer struct {
	rabbitmq      messaging.MessageBroker
	auditSvc      domain.AuditEventSvc
	inboxConsumer *messaging.InboxConsumer
	tracer        trace.Tracer
}

func NewAuditEventConsumer(
	rabbitmq messaging.MessageBroker,
	auditSvc domain.AuditEventSvc,
	inboxRepo messaging.InboxRepo,
	tracer trace.Tracer,
) *AuditEventConsumer {
	return &AuditEventConsumer{
		rabbitmq:      rabbitmq,
		auditSvc:      auditSvc,
		tracer:        tracer,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "platform-service"),
	}
}

func (c *AuditEventConsumer) Listen(ctx context.Context) error {
	// Audit events are independent rows with inbox-based dedup, so they can be persisted concurrently — no cross-message ordering requirement.
	return c.rabbitmq.ConsumeMessages(
		ctx,
		messaging.PlatformEventAuditLogQueue,
		c.inboxConsumer.Wrap("platform.audit_event", c.handleMessage),
		messaging.WithConcurrency(persistenceConsumerConcurrency),
	)
}

func (c *AuditEventConsumer) handleMessage(ctx context.Context, msg amqp091.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "event.audit_event.consume")
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		return err
	}

	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}

	var payload audit.PublishedEvent
	if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
		return err
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
	}
	if identity.Actor == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity actor not found in context."))
	}
	// A system actor is shown to merchants by its ID, so only a process OpenMRP names may be recorded as one.
	if identity.IsSystem() {
		if _, ok := constants.ParseSystemActorID(identity.Actor.ID); !ok {
			return tracing.Trace(span, apierror.NewInvariantViolationError("System identity names an unknown process: "+identity.Actor.ID))
		}
	}

	var metadataRaw json.RawMessage
	if payload.Metadata != nil {
		b, err := json.Marshal(payload.Metadata)
		if err != nil {
			return err
		}
		metadataRaw = b
	}

	changes := make([]domain.AuditFieldChange, len(payload.Changes))
	for i, ch := range payload.Changes {
		changes[i] = domain.AuditFieldChange{
			Field:    ch.Field,
			OldValue: ch.OldValue,
			NewValue: ch.NewValue,
		}
	}

	var requestID *string
	if amqpMsg.RequestID != "" {
		s := amqpMsg.RequestID
		requestID = &s
	}

	targetAccountID := identity.Target.AccountID

	event := &domain.AuditEvent{
		ID: payload.TypeID,

		ActorID:         identity.Actor.ID,
		ActorType:       string(identity.Actor.RelationType),
		IdentityType:    string(identity.Type),
		AccountID:       identity.Target.AccountID,
		TargetAccountID: &targetAccountID,

		Action:           payload.Action,
		ResourceType:     constants.ObjectType(payload.ResourceType),
		ResourceID:       payload.ResourceID,
		RootResourceType: constants.ObjectType(payload.RootResourceType),
		RootResourceID:   payload.RootResourceID,
		Changes:          changes,
		Metadata:         metadataRaw,

		ServiceName: payload.ServiceName,
		RequestID:   requestID,
		// idempotency_key_id is nullable; payload already uses *string.
		IdempotencyKeyID: payload.IdempotencyKeyID,
		SourceIP:         payload.SourceIP,

		OccurredAt: payload.OccurredAt,
		CreatedAt:  time.Now().UTC(),
	}

	if apiErr := c.auditSvc.SaveAuditEvent(ctx, event); apiErr != nil {
		return apiErr
	}

	return nil
}
