package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// AccountStripePayoutConsumer reconciles payout.paid events the account Stripe webhook verified and
// queued, so the webhook answers Stripe without reading every charge in the payout first.
type AccountStripePayoutConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	reconciler    domain.StripePayoutReconciler
	tracer        trace.Tracer
}

func NewAccountStripePayoutConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, reconciler domain.StripePayoutReconciler) *AccountStripePayoutConsumer {
	return &AccountStripePayoutConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		reconciler:    reconciler,
		tracer:        tracing.GetTracer("core-service.account_stripe_payout_consumer"),
	}
}

func (c *AccountStripePayoutConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventAccountStripePayoutPaidQueue,
		c.inboxConsumer.Wrap("core.account_stripe_payout_paid", c.handleMessage))
}

func (c *AccountStripePayoutConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.account_stripe_payout_paid",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		span.RecordError(err)
		return c.inboxConsumer.Discard(ctx, "unreadable message: "+err.Error())
	}
	var data messaging.AccountStripePayoutPaidData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		span.RecordError(err)
		return c.inboxConsumer.Discard(ctx, "unreadable payload: "+err.Error())
	}
	if data.AccountID == "" || len(data.Event) == 0 {
		return c.inboxConsumer.Discard(ctx, "missing account id or event")
	}
	span.SetAttributes(
		attribute.String("account.id", data.AccountID),
		attribute.String("stripe.event_id", data.EventID),
	)

	// Stamping funds_received_at is an overwrite, so a redelivered payout is harmless. Stripe outages
	// retry through the inbox; an event Stripe or the account will never accept is dropped.
	apiErr := c.reconciler.ReconcileAccountStripePayout(ctx, data.AccountID, data.Event)
	if apiErr == nil || apiErr.IsTransient {
		return errOrNil(apiErr)
	}
	log.Printf("[account_stripe_payout] payout %s for account %s could not be reconciled: %v", data.EventID, data.AccountID, apiErr)
	return c.inboxConsumer.Discard(ctx, apierror.Describe(apiErr))
}

// errOrNil keeps a nil *APIError from becoming a non-nil error interface.
func errOrNil(apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}
	return apiErr
}
