package event

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
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

	// The id this delivery arrived under, read the way messaging.InboxConsumer reads it. It seeds the
	// continuation's id below, which is what makes a retry of THIS invocation republish the same
	// continuation while a genuinely new chain for the same item gets a new one.
	parentMessageID := msg.MessageId
	if parentMessageID == "" {
		parentMessageID = amqpMsg.MessageID
	}

	if apiErr := c.allocateItem(ctx, parentMessageID, accountID, evt.ItemID, evt.AfterCreatedAt, evt.AfterID, allocateOpenIssuesPageSize); apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	return nil
}

// allocateItem covers a page of an item's open demand, one transaction per issue.
//
// Discovery names up to 200 issues in a single index-only read that takes no locks; each issue is
// then claimed and covered in its own short transaction. The page is still a message boundary, but it
// is no longer a transaction boundary, which is what takes the 20-second platform ceiling and the
// item-wide lock footprint out of the picture at once.
func (c *AllocateOpenIssuesConsumer) allocateItem(ctx context.Context, parentMessageID, accountID, itemID string, afterCreatedAt time.Time, afterID string, pageSize int32) *apierror.APIError {
	reservationRepo := c.repos.NewInventoryReservationRepo()

	// An item with nothing to draw on costs one read rather than a transaction per issue. The busiest
	// items in this database have hundreds of open issues against zero available receipts.
	available, apiErr := reservationRepo.CountAvailableReceiptsForItem(ctx, accountID, itemID)
	if apiErr != nil {
		return apiErr
	}
	if available == 0 {
		return nil
	}

	refs, apiErr := reservationRepo.ListOpenIssueIDsForItem(ctx, accountID, itemID, afterCreatedAt, afterID, pageSize)
	if apiErr != nil {
		return apiErr
	}

	lastCreatedAt, lastID := afterCreatedAt, afterID
	var failed int
	var firstErr *apierror.APIError

	for _, ref := range refs {
		txErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			repo := f.NewInventoryReservationRepo()
			// The ordering root, as the callback's first statement. The item set is one id and it is
			// known before the transaction opens, which is what Corollary A asks for.
			scope, apiErr := ledgerlock.Acquire(txCtx, repo, []string{itemID})
			if apiErr != nil {
				return apiErr
			}
			return repo.AllocateOneOpenIssue(txCtx, scope, accountID, itemID, ref.ID)
		})
		if txErr != nil {
			// One issue failing is not the page failing: only its own writes roll back, the row stays
			// `open`, and stopping here would let one poisoned issue starve every issue behind it. It is
			// still counted and still reported below, because swallowing it would take away the signal
			// that surfaced this incident in the first place.
			slog.ErrorContext(ctx, "allocate_open_issues: issue failed",
				"account_id", accountID, "item_id", itemID, "issue_id", ref.ID,
				"error_code", txErr.Code, "error", txErr)
			failed++
			if firstErr == nil {
				firstErr = txErr
			}
		}
		lastCreatedAt, lastID = ref.CreatedAt, ref.ID
	}

	if len(refs) == int(pageSize) {
		if apiErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			return mediator.EnqueueAllocateOpenIssuesFrom(txCtx, f.NewOutboxRepo(), accountID, itemID,
				lastCreatedAt, lastID,
				continuationMessageID(parentMessageID, accountID, itemID, lastCreatedAt, lastID))
		}); apiErr != nil {
			return apiErr
		}
	}

	if failed > 0 {
		slog.ErrorContext(ctx, "allocate_open_issues: page finished with failures",
			"account_id", accountID, "item_id", itemID, "failed", failed, "of", len(refs))
		return firstErr
	}
	return nil
}

// continuationMessageID derives a continuation's id from the delivery that produced it, so a retry of
// that delivery republishes the same message instead of forking the chain.
//
// The page and its continuation are no longer one transaction, so a handler retry — four delivery
// attempts, each re-invoking the handler — republishes the continuation. With a random id that is
// four chains for one item, each free to fork again at its own failing page. With this id the inbox's
// (handler, message_id) unique key makes every republish a no-op. Duplicate rows in the OUTBOX are
// fine and expected; the dedup that matters is on the consuming side.
//
// The parent id is in the hash, and it is what keeps this safe. Keyed on the cursor alone the id
// would be stable for all time, so the second chain ever to reach a given page — after a new receipt
// lands, say — would be deduped against the first and simply stop, leaving every issue past that page
// uncovered forever. Each chain starts from a producer's own random id, so each gets its own series.
func continuationMessageID(parentMessageID, accountID, itemID string, afterCreatedAt time.Time, afterID string) string {
	sum := sha256.Sum256([]byte(parentMessageID + "\x00" + accountID + "\x00" + itemID + "\x00" +
		strconv.FormatInt(afterCreatedAt.UnixMilli(), 10) + "\x00" + afterID))
	return string(id.MessageIDPrefix) + "_" + hex.EncodeToString(sum[:11])
}
