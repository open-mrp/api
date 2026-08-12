package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SalesOrderShippingUpdatedConsumer re-syncs an order's existing shipment records
// (carrier / service level / ship-to) after the order's shipping fields changed,
// out-of-band from the update response. This mirrors legacy updateCarrierByOrder /
// updateShipToByOrder, which cascaded these edits to shipments synchronously.
type SalesOrderShippingUpdatedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	transitWarmer domain.TransitWarmer
	tracer        trace.Tracer
}

func NewSalesOrderShippingUpdatedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	transitWarmer domain.TransitWarmer,
) *SalesOrderShippingUpdatedConsumer {
	return &SalesOrderShippingUpdatedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		transitWarmer: transitWarmer,
		tracer:        tracing.GetTracer("core-service.sales_order_shipping_updated_consumer"),
	}
}

func (c *SalesOrderShippingUpdatedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventSalesOrderShippingUpdatedQueue,
		c.inboxConsumer.Wrap("core.sales_order_shipping_updated", c.handleMessage))
}

func (c *SalesOrderShippingUpdatedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.sales_order_shipping_updated",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[sales_order_shipping_updated] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.SalesOrderShippingUpdatedData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[sales_order_shipping_updated] Failed to unmarshal event payload: %v", err)
		span.RecordError(err)
		return err
	}

	if data.SalesOrderID == "" || data.AccountID == "" {
		log.Printf("[sales_order_shipping_updated] Missing sales order ID or account ID in event")
		return nil
	}

	span.SetAttributes(
		attribute.String("sales_order.id", data.SalesOrderID),
		attribute.String("sales_order.account_id", data.AccountID),
	)

	// Re-fetch the order so we sync shipments to its current shipping fields (the event
	// carries only identifiers; this stays correct even if several updates coalesced).
	order, apiErr := c.repos.NewSalesOrderRepo().Get(ctx, data.AccountID, data.SalesOrderID)
	if apiErr != nil {
		// A missing order (deleted since the event) is terminal, not retryable.
		if apiErr.IsTransient {
			return apiErr
		}
		log.Printf("[sales_order_shipping_updated] order %s (account %s) not syncable: %v", data.SalesOrderID, data.AccountID, apiErr)
		return nil
	}

	// The carrier, service level or ship-to just moved, which is exactly what a transit lane is keyed on, so the order's old lane no longer describes it. Warming here is what keeps a lane ready for orders whose carrier is chosen after create rather than during it.
	c.warmTransit(ctx, data.AccountID, data.SalesOrderID)

	// Shipments require a carrier; if the order has none there is nothing to cascade.
	if order.CarrierID == nil {
		return nil
	}

	apiErr = c.repos.NewShipmentRepo().SyncShippingForOrder(ctx, domain.SyncShipmentShippingParams{
		AccountID:         data.AccountID,
		SalesOrderID:      data.SalesOrderID,
		CarrierID:         *order.CarrierID,
		ServiceLevelID:    order.ServiceLevelID,
		ShippingAddressID: order.ShippingAddressID,
	})
	if apiErr != nil {
		if apiErr.IsTransient {
			return apiErr
		}
		log.Printf("[sales_order_shipping_updated] shipment sync permanently failed for order %s (account %s): %v", data.SalesOrderID, data.AccountID, apiErr)
		return nil
	}
	return nil
}

// warmTransit refreshes the cached carrier transit for the order's current lane. Failures are swallowed for the same reason as on create: the estimate is a cache with a fallback, and retrying the message would redo the shipment sync below to refill it.
func (c *SalesOrderShippingUpdatedConsumer) warmTransit(ctx context.Context, accountID, salesOrderID string) {
	if c.transitWarmer == nil {
		return
	}
	if apiErr := c.transitWarmer.WarmForOrder(ctx, accountID, salesOrderID); apiErr != nil {
		log.Printf("[sales_order_shipping_updated] transit warm failed for order %s (account %s): %v", salesOrderID, accountID, apiErr)
	}
}
