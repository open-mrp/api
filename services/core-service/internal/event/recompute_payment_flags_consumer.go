package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RecomputePaymentFlagsConsumer re-derives transaction and invoice payment flags after a settlement or
// allocation changed, off the request that changed them.
//
// The command carries only identities and the flags are recomputed from current allocations under row
// locks, so repeated commands coalesce, a redelivery computes the same values, and two commands for the
// same invoice cannot leave behind the older one's answer.
type RecomputePaymentFlagsConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	txManager     db.TransactionManager[*sqlc.Queries, domain.RepoFactory]
	tracer        trace.Tracer
}

func NewRecomputePaymentFlagsConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	txManager db.TransactionManager[*sqlc.Queries, domain.RepoFactory],
) *RecomputePaymentFlagsConsumer {
	return &RecomputePaymentFlagsConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		txManager:     txManager,
		tracer:        tracing.GetTracer("core-service.recompute_payment_flags_consumer"),
	}
}

func (c *RecomputePaymentFlagsConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdRecomputePaymentFlagsQueue,
		c.inboxConsumer.Wrap("core.recompute_payment_flags", c.handleMessage))
}

func (c *RecomputePaymentFlagsConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.recompute_payment_flags",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		slog.ErrorContext(ctx, "recompute_payment_flags: failed to unmarshal envelope", "error", err)
		span.RecordError(err)
		return err
	}
	var evt domain.RecomputePaymentFlagsEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		slog.ErrorContext(ctx, "recompute_payment_flags: failed to unmarshal payload", "error", err)
		span.RecordError(err)
		return err
	}

	accountID := evt.AccountID
	if accountID == "" && amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}
	// A malformed command will never become well-formed: discard it as terminal rather than ACK it as
	// if the flags had been recomputed.
	switch {
	case accountID == "":
		slog.ErrorContext(ctx, "recompute_payment_flags: no account on event or identity")
		return c.inboxConsumer.Discard(ctx, "no account on event or identity")
	case len(evt.TransactionIDs) == 0 && len(evt.InvoiceIDs) == 0:
		slog.ErrorContext(ctx, "recompute_payment_flags: nothing to recompute", "account_id", accountID)
		return c.inboxConsumer.Discard(ctx, "no transactions or invoices on event")
	}

	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.Int("payment_flags.transactions", len(evt.TransactionIDs)),
		attribute.Int("payment_flags.invoices", len(evt.InvoiceIDs)),
	)

	apiErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := mediator.NewMediatorFactory().Build(f).PaymentFlags.Recompute(txCtx, accountID, evt.TransactionIDs, evt.InvoiceIDs); apiErr != nil {
			return apiErr
		}
		return completeInboxRecord(txCtx, f)
	})
	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}
