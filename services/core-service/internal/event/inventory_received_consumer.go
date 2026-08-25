package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InventoryReceivedConsumer offers newly available stock to the demand that went short waiting for
// it.
//
// An issue goes open when it was asked for more than the shelf could cover; it stays short until
// stock arrives, and this is what notices that it has. The walk itself is unbounded — an item can
// carry any number of open issues — so it is handed to the paged allocate-open-issues consumer
// rather than run here, keeping this handler's transaction to a few outbox rows.
type InventoryReceivedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	txManager     db.TransactionManager[*sqlc.Queries, domain.RepoFactory]
	tracer        trace.Tracer
}

func NewInventoryReceivedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	txManager db.TransactionManager[*sqlc.Queries, domain.RepoFactory],
) *InventoryReceivedConsumer {
	return &InventoryReceivedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		txManager:     txManager,
		tracer:        tracing.GetTracer("core-service.inventory_received_consumer"),
	}
}

func (c *InventoryReceivedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventInventoryReceivedAllocationQueue,
		c.inboxConsumer.Wrap("core.inventory_received_allocation", c.handleMessage))
}

func (c *InventoryReceivedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.inventory_received_allocation",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attribute.String("messaging.message_id", msg.MessageId)),
	)
	defer span.End()

	evt, accountID, err := decodeEvent[domain.InventoryReceivedEvent](msg, func(e domain.InventoryReceivedEvent) string {
		return e.AccountID
	})
	if err != nil {
		span.RecordError(err)
		return err
	}
	if accountID == "" {
		slog.ErrorContext(ctx, "inventory_received: no account on event or identity")
		return nil
	}
	if len(evt.ItemIDs) == 0 {
		return nil
	}

	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.Int("items.count", len(evt.ItemIDs)),
		attribute.String("event.reason", evt.Reason),
	)

	apiErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return enqueueAllocationForItems(txCtx, f, accountID, evt.ItemIDs)
	})
	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}

// enqueueAllocationForItems asks for each item's newly available stock to be offered to the demand
// waiting on it.
//
// Items are deduplicated because one cause routinely names the same item twice — a step consuming a
// material in two places, a batch whose output is also one of its inputs — and allocating twice for
// one arrival would walk the same open issues again for nothing.
func enqueueAllocationForItems(ctx context.Context, repos domain.RepoFactory, accountID string, itemIDs []string) *apierror.APIError {
	outboxRepo := repos.NewOutboxRepo()

	seen := make(map[string]bool, len(itemIDs))
	for _, itemID := range itemIDs {
		if itemID == "" || seen[itemID] {
			continue
		}
		seen[itemID] = true
		if apiErr := enqueueAllocateOpenIssues(ctx, outboxRepo, accountID, itemID, time.Time{}, ""); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// decodeEvent unwraps the AMQP envelope and resolves the account, preferring the payload's own so a
// replay does not depend on the envelope surviving intact.
func decodeEvent[T any](msg amqp.Delivery, accountOf func(T) string) (T, string, error) {
	var zero T

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		return zero, "", err
	}

	var evt T
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		return zero, "", err
	}

	accountID := accountOf(evt)
	if accountID == "" && amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}
	return evt, accountID, nil
}
