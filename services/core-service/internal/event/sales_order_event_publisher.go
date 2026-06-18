package event

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var salesOrderEventPublisherTracer = tracing.GetTracer("core-service.sales_order_event_publisher")

// outboxSalesOrderEventPublisher writes sales-order domain events to the outbox table instead of publishing directly to RabbitMQ, so the event commits atomically with the order in the same transaction.
type outboxSalesOrderEventPublisher struct{}

// NewOutboxSalesOrderEventPublisher creates a sales-order event publisher that writes to the outbox table for reliable message delivery.
func NewOutboxSalesOrderEventPublisher() domain.SalesOrderEventPublisher {
	return &outboxSalesOrderEventPublisher{}
}

func (p *outboxSalesOrderEventPublisher) PublishSalesOrderCreated(ctx context.Context, data messaging.SalesOrderCreatedData) *apierror.APIError {
	ctx, span := salesOrderEventPublisherTracer.Start(ctx, "event.outbox_sales_order_event_publisher.publish_sales_order_created")
	defer span.End()

	repos, ok := GetReposFromContext(ctx)
	if !ok {
		return tracing.Trace(span, apierror.NewInternalError(nil, "RepoFactory not found in context for outbox publisher."))
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal sales order created data."))
	}

	msg := contracts.AmqpMessage{
		Data: dataJSON,
	}

	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreEventSalesOrderCreated),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreEventSalesOrderCreated),
		Payload:     msg,
	}

	outboxRepo := repos.NewOutboxRepo()
	if _, err := outboxRepo.Create(ctx, outboxInput); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create outbox message."))
	}

	return nil
}
