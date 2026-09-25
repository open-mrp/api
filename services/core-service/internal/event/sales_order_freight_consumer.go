package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SalesOrderFreightConsumer adds the freight line of an order created before its carrier rate was
// cached, so creating the order never waits on the carrier. Orders that got their freight line at
// create are a no-op.
type SalesOrderFreightConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	freight       domain.FreightFinisher
	tracer        trace.Tracer
}

func NewSalesOrderFreightConsumer(rabbitmq messaging.MessageBroker, inboxRepo messaging.InboxRepo, freight domain.FreightFinisher) *SalesOrderFreightConsumer {
	return &SalesOrderFreightConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		freight:       freight,
		tracer:        tracing.GetTracer("core-service.sales_order_freight_consumer"),
	}
}

func (c *SalesOrderFreightConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventSalesOrderFreightQueue,
		c.inboxConsumer.Wrap("core.sales_order_freight", c.handleMessage))
}

func (c *SalesOrderFreightConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.sales_order_freight",
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
	var data messaging.SalesOrderCreatedData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		span.RecordError(err)
		return c.inboxConsumer.Discard(ctx, "unreadable payload: "+err.Error())
	}
	if data.SalesOrderID == "" || data.AccountID == "" {
		return c.inboxConsumer.Discard(ctx, "missing sales order or account id")
	}
	// The freight line is attributed to whoever created the order.
	if amqpMsg.Identity != nil {
		ctx = appctx.WithIdentity(ctx, amqpMsg.Identity)
	}
	span.SetAttributes(
		attribute.String("sales_order.id", data.SalesOrderID),
		attribute.String("sales_order.account_id", data.AccountID),
	)

	return c.finish(ctx, data.AccountID, data.SalesOrderID)
}

// finish retries a carrier outage through the inbox. A carrier that rejects the shipment outright
// will not accept it on retry either, so the order stops waiting and its freight is left to be
// quoted by hand.
func (c *SalesOrderFreightConsumer) finish(ctx context.Context, accountID, salesOrderID string) error {
	apiErr := c.freight.FinishPendingFreight(ctx, accountID, salesOrderID)
	if apiErr == nil {
		return nil
	}
	if apiErr.IsTransient {
		return apiErr
	}
	log.Printf("[sales_order_freight] carrier could not quote order %s (account %s), leaving freight to be quoted by hand: %v",
		salesOrderID, accountID, apiErr)
	if abandonErr := c.freight.AbandonPendingFreight(ctx, accountID, salesOrderID); abandonErr != nil {
		return abandonErr
	}
	return c.inboxConsumer.Discard(ctx, apierror.Describe(apiErr))
}
