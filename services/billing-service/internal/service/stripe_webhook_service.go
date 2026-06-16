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
	// Repos (required) is the repository factory for billing persistence.
	Repos domain.RepoFactory

	// StripeClient (required) is the Stripe API client used to verify webhooks.
	StripeClient domain.StripeClient

	// VerboseErrors (optional; default: false) includes the underlying Stripe
	// error in 400 responses when true (e.g. in dev).
	VerboseErrors bool
}

type stripeWebhookSvcImpl struct {
	repos         domain.RepoFactory
	stripeClient  domain.StripeClient
	verboseErrors bool
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
		repos:         config.Repos,
		stripeClient:  config.StripeClient,
		verboseErrors: config.VerboseErrors,
	}
}

// handledEventTypes is the set of Stripe event types we process.
// Unhandled types are acknowledged with a 200 but not enqueued.
var handledEventTypes = map[string]bool{
	"customer.deleted": true,
	// Sales order checkout payment completion.
	"checkout.session.completed": true,
	// v2 pricing plan subscription events
	"v2.billing.pricing_plan_subscription.servicing_activated": true,
	"v2.billing.pricing_plan_subscription.servicing_canceled":  true,
	"v2.billing.pricing_plan_subscription.collection_paused":   true,
	"v2.billing.pricing_plan_subscription.collection_current":  true,
	"v2.billing.cadence.errored":                               true,
	// New v2 events
	"v2.billing.pricing_plan_subscription.servicing_paused":                    true,
	"v2.billing.pricing_plan_subscription.collection_awaiting_customer_action": true,
	"v2.billing.cadence.billed":                                                true,
	"v2.billing.cadence.canceled":                                              true,
}

// ProcessWebhookEvent verifies and enqueues an incoming Stripe webhook event for async processing.
//
// 1. Verify the webhook signature against the raw payload to confirm authenticity.
// 2. If the event type is not in the handled set, acknowledge it without enqueuing.
// 3. Wrap the verified payload in an outbox message for the billing webhook queue.
// 4. Persist the outbox message for later dispatch via RabbitMQ.
// 5. Return a success result.
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
		publicMsg := "invalid webhook signature"
		if s.verboseErrors {
			publicMsg = "invalid webhook signature: " + err.Error()
		}
		apiErr := apierror.NewValidationError(publicMsg)
		apiErr.InternalMessage = err.Error()
		return nil, apiErr
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
