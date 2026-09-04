package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/hubspotsync"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SalesOrderCreatedConsumer processes sales-order-created events and runs the out-of-band side effects that should not block the create response — dispatching CRM sync for accounts with a connected integration (e.g. HubSpot), and warming the carrier transit estimate for the order's lane.
type SalesOrderCreatedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	hubspotSync   hubspotsync.Service
	transitWarmer domain.TransitWarmer
	tracer        trace.Tracer
}

func NewSalesOrderCreatedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	hubspotSync hubspotsync.Service,
	transitWarmer domain.TransitWarmer,
) *SalesOrderCreatedConsumer {
	return &SalesOrderCreatedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		hubspotSync:   hubspotSync,
		transitWarmer: transitWarmer,
		tracer:        tracing.GetTracer("core-service.sales_order_created_consumer"),
	}
}

func (c *SalesOrderCreatedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventSalesOrderCreatedQueue,
		c.inboxConsumer.Wrap("core.sales_order_created", c.handleMessage))
}

func (c *SalesOrderCreatedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.sales_order_created",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[sales_order_created] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.SalesOrderCreatedData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[sales_order_created] Failed to unmarshal event payload: %v", err)
		span.RecordError(err)
		return err
	}

	if data.SalesOrderID == "" || data.AccountID == "" {
		log.Printf("[sales_order_created] Missing sales order ID or account ID in event")
		return c.inboxConsumer.Discard(ctx, "missing sales order or account id")
	}

	span.SetAttributes(
		attribute.String("sales_order.id", data.SalesOrderID),
		attribute.String("sales_order.account_id", data.AccountID),
		attribute.String("sales_order.buyer_account_id", data.BuyerAccountID),
	)

	c.warmTransit(ctx, data.AccountID, data.SalesOrderID)

	return c.dispatchIntegrations(ctx, data)
}

// warmTransit caches the carrier's transit for this order's lane, so issuing the order later can work its ship-by date back from a promised delivery date without calling a carrier.
//
// Every failure is swallowed. The estimate is a cache with a fallback: a lane that does not warm leaves the order stamping against the service level's default, or against the promised date unadjusted, and the next order on the same lane tries again. Returning the error instead would retry the whole message and re-run the CRM sync above it — paying a third-party call to refill a cache that costs nothing to miss.
func (c *SalesOrderCreatedConsumer) warmTransit(ctx context.Context, accountID, salesOrderID string) {
	if c.transitWarmer == nil {
		return
	}
	if apiErr := c.transitWarmer.WarmForOrder(ctx, accountID, salesOrderID); apiErr != nil {
		log.Printf("[sales_order_created] transit warm failed for order %s (account %s): %v", salesOrderID, accountID, apiErr)
	}
}

// dispatchIntegrations runs each connected third-party integration's reaction to a new sales order. Each integration is independent and idempotent on msg replay (the inbox guarantees at-most-once delivery to this handler; integrations should additionally use data.SalesOrderID as their upstream idempotency key).
func (c *SalesOrderCreatedConsumer) dispatchIntegrations(ctx context.Context, data messaging.SalesOrderCreatedData) error {
	return c.syncHubspot(ctx, data)
}

// syncHubspot pushes the new sales order to HubSpot when the account has the HubSpot integration connected.
// The engine no-ops when HubSpot isn't connected. Transient failures (rate limits, 5xx) are returned so the inbox retries; permanent failures (e.g. HubSpot 4xx on bad data) are logged and swallowed so a single bad order can't poison-loop the queue.
func (c *SalesOrderCreatedConsumer) syncHubspot(ctx context.Context, data messaging.SalesOrderCreatedData) error {
	apiErr := c.hubspotSync.SyncOrder(ctx, data.AccountID, data.SalesOrderID)
	if apiErr == nil {
		return nil
	}
	if apiErr.IsTransient {
		return apiErr
	}
	log.Printf("[sales_order_created] HubSpot sync permanently failed for order %s (account %s): %v",
		data.SalesOrderID, data.AccountID, apiErr)
	return c.inboxConsumer.Discard(ctx, apierror.Describe(apiErr))
}
