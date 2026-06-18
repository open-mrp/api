package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SalesOrderCreatedConsumer processes sales-order-created events and runs the
// out-of-band side effects that should not block the create response — currently
// dispatching CRM sync for accounts with a connected integration (e.g. HubSpot).
type SalesOrderCreatedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	tracer        trace.Tracer
}

func NewSalesOrderCreatedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
) *SalesOrderCreatedConsumer {
	return &SalesOrderCreatedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
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
		return nil
	}

	span.SetAttributes(
		attribute.String("sales_order.id", data.SalesOrderID),
		attribute.String("sales_order.account_id", data.AccountID),
		attribute.String("sales_order.buyer_account_id", data.BuyerAccountID),
	)

	return c.dispatchIntegrations(ctx, data)
}

// dispatchIntegrations runs each connected third-party integration's reaction to a new
// sales order. Each integration is independent and idempotent on msg replay (the inbox
// guarantees at-most-once delivery to this handler; integrations should additionally use
// data.SalesOrderID as their upstream idempotency key).
func (c *SalesOrderCreatedConsumer) dispatchIntegrations(ctx context.Context, data messaging.SalesOrderCreatedData) error {
	return c.syncHubspot(ctx, data)
}

// syncHubspot pushes the new sales order to HubSpot when the account has the HubSpot
// integration connected.
//
// TODO: implement the actual HubSpot sync once a HubSpot API client exists. The wiring
// below establishes the trigger point and the enabled-check; the remaining work is to
// (1) build a HubSpot client from the account's encrypted credentials
// (AccountIntegrationRepo.GetEncryptedCredentials), (2) re-fetch the full order via the
// sales-order repo, and (3) upsert the corresponding HubSpot deal/line items keyed on
// data.SalesOrderID for idempotency.
func (c *SalesOrderCreatedConsumer) syncHubspot(ctx context.Context, data messaging.SalesOrderCreatedData) error {
	hasHubspot, apiErr := c.repos.NewAccountIntegrationRepo().HasIntegration(ctx, data.AccountID, constants.IntegrationCodeHubspot)
	if apiErr != nil {
		return apiErr
	}
	if !hasHubspot {
		return nil
	}

	log.Printf("[sales_order_created] HubSpot integration enabled for account %s; sync for order %s not yet implemented",
		data.AccountID, data.SalesOrderID)
	return nil
}
