package event

import (
	"context"
	"encoding/json"
	"log"
	"sort"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UndoBatchScanConsumer reverses the inventory a scan recorded against a batch that has been deleted: the receipts it produced, the issues it consumed, and the reservations it drew down.
//
// The delete itself already happened synchronously — this is the ledger catching up behind it, the mirror of ExecuteProductionStepConsumer running behind a scan.
type UndoBatchScanConsumer struct {
	rabbitmq       messaging.MessageBroker
	inboxConsumer  *messaging.InboxConsumer
	repos          domain.RepoFactory
	txManager      db.TransactionManager[*sqlc.Queries, domain.RepoFactory]
	outboxNotifier messaging.OutboxNotifier
	tracer         trace.Tracer
}

// outboxNotifier is optional: when nil the allocation request this consumer writes is still picked up
// on the enqueuer's next poll, just not on the next instant.
func NewUndoBatchScanConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	txManager db.TransactionManager[*sqlc.Queries, domain.RepoFactory],
	outboxNotifier messaging.OutboxNotifier,
) *UndoBatchScanConsumer {
	return &UndoBatchScanConsumer{
		rabbitmq:       rabbitmq,
		inboxConsumer:  messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:          repos,
		txManager:      txManager,
		outboxNotifier: outboxNotifier,
		tracer:         tracing.GetTracer("core-service.undo_batch_scan_consumer"),
	}
}

// kickOutbox wakes the outbox enqueuer so a just-committed allocation request is picked up
// immediately rather than on the enqueuer's next idle poll, which can be up to MaxPollInterval away.
// No-op when no notifier was injected. Call only after the writing transaction has committed —
// kicking while it is still open races the poll against a row it cannot yet see.
func (c *UndoBatchScanConsumer) kickOutbox() {
	if c.outboxNotifier != nil {
		c.outboxNotifier.Notify()
	}
}

func (c *UndoBatchScanConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdUndoBatchScanQueue,
		c.inboxConsumer.Wrap("core.undo_batch_scan", c.handleMessage))
}

func (c *UndoBatchScanConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.undo_batch_scan",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[undo_batch_scan] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var evt domain.UndoBatchScanEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		log.Printf("[undo_batch_scan] Failed to unmarshal event payload: %v", err)
		span.RecordError(err)
		return err
	}

	if evt.BatchID == "" {
		log.Printf("[undo_batch_scan] Empty batch ID in event")
		return nil
	}

	accountID := ""
	if amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}
	if accountID == "" {
		log.Printf("[undo_batch_scan] No account ID in message identity")
		return nil
	}

	span.SetAttributes(
		attribute.String("batch.id", evt.BatchID),
		attribute.String("batch.account_id", accountID),
	)

	return c.undoBatchScan(ctx, accountID, evt)
}

// undoBatchScan reverses the batch's ledger rows in one transaction, then offers the stock it freed
// to demand that was short.
//
// The reversal is transactional and the re-allocation deliberately is not, and the split is the
// point. ReverseInventoryForBatch deletes allocations, deletes their quantity and rate rows, frees
// the receipts and restores the issues as seven separate set-based statements; until this consumer
// was given a transaction manager every one of them autocommitted, so any failure part-way left a
// ledger that is reversed on one side and not the other. That is not hypothetical — the invariant
// check finds an allocation in production whose issue and receipt still exist while all three of its
// satellite rows are gone, which is what a half-applied delete looks like. It is also why the
// FOR UPDATE this path used to take bought nothing here: single statements in autocommit release
// their locks immediately.
//
// Re-allocation stays outside because it is not part of the correction. The reversal is complete and
// correct on its own; covering the freed stock is opportunistic follow-up that any later allocation
// pass for the item would do anyway. Rolling a good reversal back because allocation failed would be
// the wrong trade, and pulling an unbounded walk over every open issue into this transaction would
// put it under the same server time limit that has already killed one allocation transaction.
func (c *UndoBatchScanConsumer) undoBatchScan(ctx context.Context, accountID string, evt domain.UndoBatchScanEvent) error {
	var deltas []domain.InventoryReversalDelta

	// The item set is resolved here, on the pool, before the transaction opens. Discovering it inside
	// the transaction would mean taking the ordering root after the reversal already held ledger row
	// locks, which is itself an inversion (Corollary A) — the precise mistake this rule exists to fix.
	itemIDs, apiErr := c.repos.NewInventoryMutationRepo().ListItemIDsForBatchReversal(ctx, accountID, evt.BatchID)
	if apiErr != nil {
		log.Printf("[undo_batch_scan] Failed to resolve the item set for batch %s: %v", evt.BatchID, apiErr)
		return apiErr
	}

	apiErr = c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		mutationRepo := f.NewInventoryMutationRepo()
		scope, apiErr := ledgerlock.Acquire(txCtx, mutationRepo, itemIDs)
		if apiErr != nil {
			return apiErr
		}

		reversed, apiErr := mutationRepo.ReverseInventoryForBatch(txCtx, scope, domain.ReverseInventoryForBatchParams{
			AccountID:         accountID,
			BatchID:           evt.BatchID,
			ScanningStationID: evt.ScanningStationID,
			ResponsibleUserID: evt.ResponsibleUserID,
		})
		if apiErr != nil {
			// The delete checked that nothing had drawn on the batch's output, but that was before this
			// message was picked up. Failing here parks the message in the dead-letter queue with the
			// batch on it, which is recoverable. The public message is logged explicitly because a
			// validation error carries no internal one to print.
			log.Printf("[undo_batch_scan] Failed to reverse inventory for batch %s: %s", evt.BatchID, apiErr.PublicMessage)
			return apiErr
		}

		if apiErr := c.restoreShortfallReservations(txCtx, scope, f, accountID, evt); apiErr != nil {
			log.Printf("[undo_batch_scan] Failed to restore reservations for batch %s: %v", evt.BatchID, apiErr)
			return apiErr
		}

		recordReversalAuditTrail(txCtx, f, accountID, evt, reversed)

		// Assigned, not appended: the callback re-runs on a lock conflict and the second run must
		// start from the same state the first did.
		deltas = reversed
		return nil
	})
	if apiErr != nil {
		return apiErr
	}

	// Freed receipts can now cover issues that were short, so allocation is asked for again for
	// whatever the reversal touched. Sorted because two of these taking the same items in different
	// orders is a deadlock nobody would be able to explain from the logs.
	//
	// Its own transaction, after the reversal has committed: an allocation request that survives a
	// reversal which did not is a request to cover demand from stock that was never freed.
	requestIDs := reversedItemIDs(deltas)
	if len(requestIDs) > 0 {
		if apiErr := c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			return mediator.EnqueueAllocateOpenIssues(txCtx, f, accountID, requestIDs...)
		}); apiErr != nil {
			log.Printf("[undo_batch_scan] Failed to request allocation for batch %s: %v", evt.BatchID, apiErr)
			return apiErr
		}
		c.kickOutbox()
	}

	log.Printf("[undo_batch_scan] Completed: batch=%s account=%s corrections=%d", evt.BatchID, accountID, len(deltas))
	return nil
}

// recordReversalAuditTrail stamps one correction per reversed row, signed opposite to the scan.
func recordReversalAuditTrail(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID string,
	evt domain.UndoBatchScanEvent,
	deltas []domain.InventoryReversalDelta,
) {
	var scanningStationID *string
	if evt.ScanningStationID != "" {
		stationID := evt.ScanningStationID
		scanningStationID = &stationID
	}
	var responsibleUserID *string
	if evt.ResponsibleUserID != "" {
		userID := evt.ResponsibleUserID
		responsibleUserID = &userID
	}

	for _, delta := range deltas {
		mediator.RecordInventoryAuditTrailOrLog(
			ctx,
			repos,
			accountID,
			delta.ItemID,
			delta.Measure,
			delta.UnitID,
			string(constants.InventoryActionTypeUserCorrection),
			scanningStationID,
			responsibleUserID,
		)
	}
}

// reversedItemIDs is the deduplicated item set of a reversal, in a stable order.
//
// Sorted rather than ranged over as a map: Go randomises map iteration, so two undo transactions
// covering the same items would take their locks in different orders and deadlock each other at
// random. The same hazard is open in four other item loops; this is the one in reach here.
func reversedItemIDs(deltas []domain.InventoryReversalDelta) []string {
	seen := make(map[string]bool, len(deltas))
	out := make([]string, 0, len(deltas))
	for _, delta := range deltas {
		if seen[delta.ItemID] {
			continue
		}
		seen[delta.ItemID] = true
		out = append(out, delta.ItemID)
	}
	sort.Strings(out)
	return out
}

// restoreShortfallReservations puts back the reservations the scan released for the units its scrap
// meant would never be produced. The amounts were snapshotted at delete time, since the lineage they
// are read from does not survive the batch.
func (c *UndoBatchScanConsumer) restoreShortfallReservations(ctx context.Context, scope *ledgerlock.Scope, repos domain.RepoFactory, accountID string, evt domain.UndoBatchScanEvent) *apierror.APIError {
	if evt.OrderID == "" || evt.ShortfallMeasure == "" {
		return nil
	}

	shortfall, err := decimal.NewFromString(evt.ShortfallMeasure)
	if err != nil {
		log.Printf("[undo_batch_scan] Invalid shortfall measure %q: %v", evt.ShortfallMeasure, err)
		return nil
	}
	if shortfall.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	reservationRepo := repos.NewInventoryReservationRepo()

	// CreateMaterialReservation is the inverse of ReduceReservedForOrderItem: a reservation is a
	// plain row, and the one that held this quantity was deleted rather than shrunk.
	if apiErr := reservationRepo.CreateMaterialReservation(ctx, scope, domain.CreateMaterialReservationParams{
		AccountID: accountID,
		ItemID:    evt.ProducedItemID,
		Measure:   shortfall,
		UnitID:    evt.ShortfallUnitID,
		OrderID:   evt.OrderID,
	}); apiErr != nil {
		return apiErr
	}

	// The upstream materials that would have gone into those units move with them.
	demands, apiErr := repos.NewMaterialDemandRepo().GetMaterialDemand(ctx, accountID, evt.ProducedItemID, shortfall, evt.ShortfallUnitID)
	if apiErr != nil {
		return apiErr
	}

	for _, demand := range demands {
		if apiErr := reservationRepo.CreateMaterialReservation(ctx, scope, domain.CreateMaterialReservationParams{
			AccountID: accountID,
			ItemID:    demand.ItemID,
			Measure:   demand.Measure,
			UnitID:    demand.UnitID,
			OrderID:   evt.OrderID,
		}); apiErr != nil {
			return apiErr
		}
	}

	return nil
}
