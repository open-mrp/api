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
	tracer        trace.Tracer
}

func NewSalesOrderShippingUpdatedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
) *SalesOrderShippingUpdatedConsumer {
	return &SalesOrderShippingUpdatedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
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
