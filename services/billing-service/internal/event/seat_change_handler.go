package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SeatChangeAccountUsageRepo defines the account usage operations needed by the seat change handler.
type SeatChangeAccountUsageRepo interface {
	GetStripeCustomerIDByAccountID(ctx context.Context, accountID string) (*string, *apierror.APIError)
	CountUsersByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
}

// SeatChangeStripeClient defines the Stripe operations needed by the seat change handler.
type SeatChangeStripeClient interface {
	ReportMeterEvent(ctx context.Context, eventName, stripeCustomerID string, value int, idempotencyKey string) error
}

type SeatChangeHandler struct {
	tracer       trace.Tracer
	usageRepo    SeatChangeAccountUsageRepo
	stripeClient SeatChangeStripeClient
}

func NewSeatChangeHandler(
	usageRepo SeatChangeAccountUsageRepo,
	stripeClient SeatChangeStripeClient,
) *SeatChangeHandler {
	return &SeatChangeHandler{
		tracer:       tracing.GetTracer("billing-service.seat_change_handler"),
		usageRepo:    usageRepo,
		stripeClient: stripeClient,
	}
}

// Handle consumes a seat-change event and reports the account's current seat count to Stripe as a metered usage event. Accounts with no Stripe customer (free tier) are skipped. The AMQP message ID is passed as the Stripe idempotency key, so redelivery of the same message does not double-report usage.
func (h *SeatChangeHandler) Handle(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := h.tracer.Start(ctx, "handler.seat_change",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[seat_change] Failed to unmarshal AMQP message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.SeatChangeReportData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[seat_change] Failed to unmarshal seat change data: %v", err)
		span.RecordError(err)
		return err
	}

	span.SetAttributes(attribute.String("billing.account_id", data.AccountID))

	log.Printf("[seat_change] Received seat change report for account %s", data.AccountID)

	stripeCustomerID, apiErr := h.usageRepo.GetStripeCustomerIDByAccountID(ctx, data.AccountID)
	if apiErr != nil {
		return fmt.Errorf("failed to get Stripe customer ID: %w", apiErr)
	}
	if stripeCustomerID == nil {
		log.Printf("[seat_change] No Stripe customer for account %s (free tier), skipping meter event", data.AccountID)
		return nil
	}

	seatCount, apiErr := h.usageRepo.CountUsersByAccountID(ctx, data.AccountID)
	if apiErr != nil {
		return fmt.Errorf("failed to count users: %w", apiErr)
	}

	span.SetAttributes(
		attribute.String("billing.stripe_customer_id", *stripeCustomerID),
		attribute.Int("billing.seat_count", seatCount),
	)

	if err := h.stripeClient.ReportMeterEvent(ctx, "seat_count", *stripeCustomerID, seatCount, msg.MessageId); err != nil {
		return fmt.Errorf("failed to report meter event: %w", err)
	}

	log.Printf("[seat_change] Reported seat count %d for account %s (customer %s)",
		seatCount, data.AccountID, *stripeCustomerID)

	return nil
}
