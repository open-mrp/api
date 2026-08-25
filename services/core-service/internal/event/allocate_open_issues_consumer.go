package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const allocateOpenIssuesPageSize = 200

type AllocateOpenIssuesConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	txManager     db.TransactionManager[*sqlc.Queries, domain.RepoFactory]
	tracer        trace.Tracer
}

func NewAllocateOpenIssuesConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	txManager db.TransactionManager[*sqlc.Queries, domain.RepoFactory],
) *AllocateOpenIssuesConsumer {
	return &AllocateOpenIssuesConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		txManager:     txManager,
		tracer:        tracing.GetTracer("core-service.allocate_open_issues_consumer"),
	}
}

func (c *AllocateOpenIssuesConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdAllocateOpenIssuesQueue,
		c.inboxConsumer.Wrap("core.allocate_open_issues", c.handleMessage))
}

func (c *AllocateOpenIssuesConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.allocate_open_issues",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		slog.ErrorContext(ctx, "allocate_open_issues: failed to unmarshal envelope", "error", err)
		span.RecordError(err)
		return err
	}

	var evt domain.AllocateOpenIssuesEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		slog.ErrorContext(ctx, "allocate_open_issues: failed to unmarshal payload", "error", err)
		span.RecordError(err)
		return err
	}

	accountID := evt.AccountID
	if accountID == "" && amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}

	switch {
	case accountID == "":
		slog.ErrorContext(ctx, "allocate_open_issues: no account on event or identity", "item_id", evt.ItemID)
		return nil
	case evt.ItemID == "":
		slog.ErrorContext(ctx, "allocate_open_issues: no item on event")
		return nil
	}

	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("item.id", evt.ItemID),
		attribute.String("cursor.after_id", evt.AfterID),
	)

	apiErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return allocateOpenIssuesPage(txCtx, f, accountID, evt.ItemID, evt.AfterCreatedAt, evt.AfterID, allocateOpenIssuesPageSize)
	})
	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}

func allocateOpenIssuesPage(ctx context.Context, f domain.RepoFactory, accountID, itemID string, afterCreatedAt time.Time, afterID string, pageSize int32) *apierror.APIError {
	lastCreatedAt, lastID, count, apiErr := f.NewInventoryReservationRepo().
		AllocateOpenIssuesForItemPage(ctx, accountID, itemID, afterCreatedAt, afterID, pageSize)
	if apiErr != nil {
		return apiErr
	}

	if count == int(pageSize) {
		return enqueueAllocateOpenIssues(ctx, f.NewOutboxRepo(), accountID, itemID, lastCreatedAt, lastID)
	}
	return nil
}

func enqueueAllocateOpenIssues(ctx context.Context, outboxRepo messaging.OutboxRepo, accountID, itemID string, afterCreatedAt time.Time, afterID string) *apierror.APIError {
	payload, err := json.Marshal(domain.AllocateOpenIssuesEvent{
		AccountID:      accountID,
		ItemID:         itemID,
		AfterCreatedAt: afterCreatedAt,
		AfterID:        afterID,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal allocate open issues event.")
	}

	msg := contracts.AmqpMessage{
		Data: payload,
	}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreCmdAllocateOpenIssues),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdAllocateOpenIssues),
		Payload:     msg,
	}

	if _, err := outboxRepo.Create(ctx, outboxInput); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for allocate open issues.")
	}

	return nil
}
