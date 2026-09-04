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

// RecalcItemBurnRateConsumer recomputes an item's burn rate from its recent consumption history, off
// the transaction that recorded the consumption. That transaction is long, and recomputing inline
// held the shared rate row's X-lock for its whole length; doing it here holds that lock only for this
// consumer's own short transaction.
//
// The command carries only the item's identity and the rate is recomputed from current state, so
// repeated commands coalesce and a redelivery recomputes the same absolute value.
type RecalcItemBurnRateConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	txManager     db.TransactionManager[*sqlc.Queries, domain.RepoFactory]
	tracer        trace.Tracer
}

func NewRecalcItemBurnRateConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	txManager db.TransactionManager[*sqlc.Queries, domain.RepoFactory],
) *RecalcItemBurnRateConsumer {
	return &RecalcItemBurnRateConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		txManager:     txManager,
		tracer:        tracing.GetTracer("core-service.recalc_item_burn_rate_consumer"),
	}
}

func (c *RecalcItemBurnRateConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdRecalcItemBurnRateQueue,
		c.inboxConsumer.Wrap("core.recalc_item_burn_rate", c.handleMessage))
}

func (c *RecalcItemBurnRateConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.recalc_item_burn_rate",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		slog.ErrorContext(ctx, "recalc_item_burn_rate: failed to unmarshal envelope", "error", err)
		span.RecordError(err)
		return err
	}

	var evt domain.RecalcItemBurnRateEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		slog.ErrorContext(ctx, "recalc_item_burn_rate: failed to unmarshal payload", "error", err)
		span.RecordError(err)
		return err
	}

	// The account is on the payload so a replay does not depend on the envelope, but an older
	// publisher may only set the identity.
	accountID := evt.AccountID
	if accountID == "" && amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}

	// A malformed command will never become well-formed. Discarding records the drop as terminal and surfaces it to the failure monitor, rather than ACKing it as if the recalculation had run.
	switch {
	case accountID == "":
		slog.ErrorContext(ctx, "recalc_item_burn_rate: no account on event or identity", "item_id", evt.ItemID)
		return c.inboxConsumer.Discard(ctx, "no account on event or identity")
	case evt.ItemID == "":
		slog.ErrorContext(ctx, "recalc_item_burn_rate: no item on event")
		return c.inboxConsumer.Discard(ctx, "no item on event")
	}

	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("item.id", evt.ItemID),
	)

	apiErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := mediator.NewMediatorFactory().Build(f).BurnRate.RecalculateFromHistory(txCtx, accountID, evt.ItemID); apiErr != nil {
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
