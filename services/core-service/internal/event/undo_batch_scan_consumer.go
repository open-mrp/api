package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
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
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	tracer        trace.Tracer
}

func NewUndoBatchScanConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
) *UndoBatchScanConsumer {
	return &UndoBatchScanConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		tracer:        tracing.GetTracer("core-service.undo_batch_scan_consumer"),
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

func (c *UndoBatchScanConsumer) undoBatchScan(ctx context.Context, accountID string, evt domain.UndoBatchScanEvent) error {
	deltas, apiErr := c.repos.NewInventoryMutationRepo().ReverseInventoryForBatch(ctx, domain.ReverseInventoryForBatchParams{
		AccountID:         accountID,
		BatchID:           evt.BatchID,
		ScanningStationID: evt.ScanningStationID,
		ResponsibleUserID: evt.ResponsibleUserID,
	})
	if apiErr != nil {
		// The delete checked that nothing had drawn on the batch's output, but that was before this
		// message was picked up. Failing here parks the message in the dead-letter queue with the
		// batch on it, which is recoverable; reversing half a ledger is not. The public message is
		// logged explicitly because a validation error carries no internal one to print.
		log.Printf("[undo_batch_scan] Failed to reverse inventory for batch %s: %s", evt.BatchID, apiErr.PublicMessage)
		return apiErr
	}

	if apiErr := c.restoreShortfallReservations(ctx, accountID, evt); apiErr != nil {
		log.Printf("[undo_batch_scan] Failed to restore reservations for batch %s: %v", evt.BatchID, apiErr)
		return apiErr
	}

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

	itemIDs := make(map[string]bool, len(deltas))
	for _, delta := range deltas {
		itemIDs[delta.ItemID] = true

		mediator.RecordInventoryAuditTrailOrLog(
			ctx,
			c.repos,
			accountID,
			delta.ItemID,
			delta.Measure,
			delta.UnitID,
			string(constants.InventoryActionTypeUserCorrection),
			scanningStationID,
			responsibleUserID,
		)
	}

	// Freed receipts can now cover issues that were short, so allocation runs again for whatever the
	// reversal touched.
	reservationRepo := c.repos.NewInventoryReservationRepo()
	for itemID := range itemIDs {
		if apiErr := reservationRepo.AllocateOpenIssuesForItem(ctx, accountID, itemID); apiErr != nil {
			log.Printf("[undo_batch_scan] Failed to re-allocate open issues for item %s: %v", itemID, apiErr)
			return apiErr
		}
	}

	log.Printf("[undo_batch_scan] Completed: batch=%s account=%s corrections=%d", evt.BatchID, accountID, len(deltas))
	return nil
}

// restoreShortfallReservations puts back the reservations the scan released for the units its scrap
// meant would never be produced. The amounts were snapshotted at delete time, since the lineage they
// are read from does not survive the batch.
func (c *UndoBatchScanConsumer) restoreShortfallReservations(ctx context.Context, accountID string, evt domain.UndoBatchScanEvent) *apierror.APIError {
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

	reservationRepo := c.repos.NewInventoryReservationRepo()

	// CreateMaterialReservation is the inverse of ReduceReservedForOrderItem: a reservation is a
	// plain row, and the one that held this quantity was deleted rather than shrunk.
	if apiErr := reservationRepo.CreateMaterialReservation(ctx, domain.CreateMaterialReservationParams{
		AccountID: accountID,
		ItemID:    evt.ProducedItemID,
		Measure:   shortfall,
		UnitID:    evt.ShortfallUnitID,
		OrderID:   evt.OrderID,
	}); apiErr != nil {
		return apiErr
	}

	// The upstream materials that would have gone into those units move with them.
	demands, apiErr := c.repos.NewMaterialDemandRepo().GetMaterialDemand(ctx, accountID, evt.ProducedItemID, shortfall, evt.ShortfallUnitID)
	if apiErr != nil {
		return apiErr
	}

	for _, demand := range demands {
		if apiErr := reservationRepo.CreateMaterialReservation(ctx, domain.CreateMaterialReservationParams{
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
