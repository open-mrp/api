package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Names the Stripe meter that bills batch volume; it must match the meter configured in the Stripe dashboard.
const batchCreatedMeterEventName = "openmrp_batches"

// errBatchMeterRejected marks a meter event Stripe refused for good. Batch metering has always been best
// effort, so the consumer records the refusal instead of retrying it into the dead-letter queue.
var errBatchMeterRejected = errors.New("batch meter event rejected")

// Declares the account usage lookups the batch-created handler needs.
type BatchCreatedAccountUsageRepo interface {
	GetStripeCustomerIDByAccountID(ctx context.Context, accountID string) (*string, *apierror.APIError)
}

// Declares the Stripe operations the batch-created handler needs.
type BatchCreatedStripeClient interface {
	ReportMeterEvent(ctx context.Context, eventName, stripeCustomerID string, value int, idempotencyKey string) error
}

type BatchCreatedHandler struct {
	tracer       trace.Tracer
	usageRepo    BatchCreatedAccountUsageRepo
	stripeClient BatchCreatedStripeClient
}

func NewBatchCreatedHandler(
	usageRepo BatchCreatedAccountUsageRepo,
	stripeClient BatchCreatedStripeClient,
) *BatchCreatedHandler {
	return &BatchCreatedHandler{
		tracer:       tracing.GetTracer("billing-service.batch_created_handler"),
		usageRepo:    usageRepo,
		stripeClient: stripeClient,
	}
}

// Reports one batch-created usage event to Stripe, skipping accounts with no Stripe customer (free tier).
// The AMQP message ID is the Stripe idempotency key, so redelivery does not double-count the batch.
func (h *BatchCreatedHandler) Handle(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := h.tracer.Start(ctx, "handler.batch_created",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[batch_created] Failed to unmarshal AMQP message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.BatchCreatedReportData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[batch_created] Failed to unmarshal batch created data: %v", err)
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.String("billing.account_id", data.AccountID),
		attribute.String("billing.batch_id", data.BatchID),
	)

	stripeCustomerID, apiErr := h.usageRepo.GetStripeCustomerIDByAccountID(ctx, data.AccountID)
	if apiErr != nil {
		return fmt.Errorf("failed to get Stripe customer ID: %w", apiErr)
	}
	if stripeCustomerID == nil {
		log.Printf("[batch_created] No Stripe customer for account %s (free tier), skipping meter event", data.AccountID)
		return nil
	}

	span.SetAttributes(attribute.String("billing.stripe_customer_id", *stripeCustomerID))

	if err := h.stripeClient.ReportMeterEvent(ctx, batchCreatedMeterEventName, *stripeCustomerID, 1, msg.MessageId); err != nil {
		var permanent interface{ Permanent() bool }
		if errors.As(err, &permanent) && permanent.Permanent() {
			log.Printf("[batch_created] Stripe rejected the meter event for batch %s (account %s): %v", data.BatchID, data.AccountID, err)
			span.RecordError(err)
			return fmt.Errorf("%w: %v", errBatchMeterRejected, err)
		}
		return fmt.Errorf("failed to report meter event: %w", err)
	}

	log.Printf("[batch_created] Reported batch %s for account %s (customer %s)",
		data.BatchID, data.AccountID, *stripeCustomerID)

	return nil
}
