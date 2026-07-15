package event

import (
	"context"
	"encoding/json"

	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var customerRegisteredPublisherTracer = tracing.GetTracer("core-service.customer_registered_publisher")

// CustomerRegisteredPublisher writes customer-registered events to the outbox so the notification-service can notify the seller's customer-service group out-of-band from the registration response.
type CustomerRegisteredPublisher struct{}

// NewCustomerRegisteredPublisher creates a publisher for customer-registered events.
func NewCustomerRegisteredPublisher() *CustomerRegisteredPublisher {
	return &CustomerRegisteredPublisher{}
}

// Publish enqueues a customer-registered event via the outbox. It is safe to call inside a service transaction: the outboxRepo write commits atomically with the registration mutation.
func (p *CustomerRegisteredPublisher) Publish(ctx context.Context, outboxRepo messaging.OutboxRepo, data messaging.CustomerRegisteredData) *apierror.APIError {
	ctx, span := customerRegisteredPublisherTracer.Start(ctx, "event.customer_registered_publisher.publish")
	defer span.End()

	if outboxRepo == nil {
		return tracing.Trace(span, apierror.NewInternalError(nil, "Customer-registered publisher: outbox repo is required."))
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal customer-registered data."))
	}

	msg := contracts.AmqpMessage{Data: dataJSON}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := outboxRepo.Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreEventCustomerRegistered),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreEventCustomerRegistered),
		Payload:     msg,
	}); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create customer-registered outbox message."))
	}

	return nil
}
