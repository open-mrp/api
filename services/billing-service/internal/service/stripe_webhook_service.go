package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var stripeWebhookSvcTracer = tracing.GetTracer("billing-service.stripe_webhook_service")

type StripeWebhookSvcConfig struct {
	Repos        domain.RepoFactory
	StripeClient domain.StripeClient
}

type stripeWebhookSvcImpl struct {
	repos        domain.RepoFactory
	stripeClient domain.StripeClient
}

func (c *StripeWebhookSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("stripe webhook service: repos is required")
	}
	if c.StripeClient == nil {
		return fmt.Errorf("stripe webhook service: stripe client is required")
	}
	return nil
}

func NewStripeWebhookSvc(config *StripeWebhookSvcConfig) domain.StripeWebhookSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &stripeWebhookSvcImpl{
		repos:        config.Repos,
		stripeClient: config.StripeClient,
	}
}

// handledEventTypes is the set of Stripe event types we process.
// Unhandled types are acknowledged with a 200 but not enqueued.
var handledEventTypes = map[string]bool{
	"customer.subscription.updated": true,
	"customer.subscription.deleted": true,
	"customer.deleted":              true,
	"invoice.payment_failed":        true,
}

func (s *stripeWebhookSvcImpl) ProcessWebhookEvent(ctx context.Context, input domain.ProcessWebhookEventInput) (*domain.ProcessWebhookEventResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, stripeWebhookSvcTracer, "service.stripe_webhook.process_webhook_event")
	defer span.End()

	slog.InfoContext(ctx, "processing webhook event in billing service",
		"payload_size", len(input.RawPayload),
		"signature_present", input.StripeSignature != "",
		"signature_length", len(input.StripeSignature),
		"payload_empty", len(input.RawPayload) == 0,
	)

	// Verify the webhook signature to ensure the payload is from Stripe.
	event, err := s.stripeClient.VerifyWebhookSignature(input.RawPayload, input.StripeSignature)
	if err != nil {
		slog.ErrorContext(ctx, "webhook signature verification failed",
			"error", err.Error(),
			"payload_size", len(input.RawPayload),
			"signature_length", len(input.StripeSignature),
		)
		span.RecordError(err)
		return nil, apierror.NewValidationError("invalid webhook signature")
	}

	slog.InfoContext(ctx, "webhook signature verified",
		"event_id", event.ID,
		"event_type", event.Type,
		"object_id", event.ObjectID,
	)

	// Skip enqueuing events we don't handle.
	if !handledEventTypes[event.Type] {
		slog.InfoContext(ctx, "skipping unhandled webhook event type",
			"event_id", event.ID,
			"event_type", event.Type,
		)
		return &domain.ProcessWebhookEventResult{Success: true}, nil
	}

	// Enqueue the verified event for async processing via the outbox.
	msg := contracts.AmqpMessage{
		Data: input.RawPayload,
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.BillingEventStripeWebhook),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.BillingEventStripeWebhook),
		Payload:     msg,
	}

	outboxRepo := s.repos.NewOutboxRepo()
	if _, err := outboxRepo.Create(ctx, outboxInput); err != nil {
		slog.ErrorContext(ctx, "failed to enqueue webhook event",
			"error", err.Error(),
			"event_id", event.ID,
			"event_type", event.Type,
		)
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to enqueue webhook event")
	}

	slog.InfoContext(ctx, "webhook event enqueued",
		"event_id", event.ID,
		"event_type", event.Type,
	)

	return &domain.ProcessWebhookEventResult{
		Success: true,
	}, nil
}
