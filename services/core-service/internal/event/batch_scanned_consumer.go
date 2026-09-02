package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
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

// BatchScannedConsumer moves inventory in reaction to a scan. It credits what the batch produced, releases the reservations covering units that seconds and waste mean will never exist, and draws down the materials the step consumed.
//
// None of these writes is idempotent on its own, so the scan is applied exactly once by construction rather than by making each movement replayable. The whole reaction commits as one transaction, and the inbox recovery point commits inside it: a failed attempt rolls back with nothing applied, and an attempt that loses the race to a concurrent delivery rolls back too. Marking the message outside that transaction is what previously let a scan commit and then be applied a second time by a redelivery or a replay, because the marker was lost with the process.
type BatchScannedConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	repos         domain.RepoFactory
	txManager     db.TransactionManager[*sqlc.Queries, domain.RepoFactory]
	tracer        trace.Tracer
}

func NewBatchScannedConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	repos domain.RepoFactory,
	txManager db.TransactionManager[*sqlc.Queries, domain.RepoFactory],
) *BatchScannedConsumer {
	return &BatchScannedConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		repos:         repos,
		txManager:     txManager,
		tracer:        tracing.GetTracer("core-service.batch_scanned_consumer"),
	}
}

func (c *BatchScannedConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreEventBatchScannedInventoryQueue,
		c.inboxConsumer.Wrap("core.batch_scanned_inventory", c.handleMessage))
}

// ReplayMessage re-drives a single delivery through the same inbox-dedup wrapper Listen uses, so a maintenance tool can re-run a message that failed permanently without re-applying one that already succeeded: the wrapper skips any inbox record already marked processed.
func (c *BatchScannedConsumer) ReplayMessage(ctx context.Context, msg amqp.Delivery) error {
	return c.inboxConsumer.Wrap("core.batch_scanned_inventory", c.handleMessage)(ctx, msg)
}

func (c *BatchScannedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.batch_scanned_inventory",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		slog.ErrorContext(ctx, "batch_scanned: failed to unmarshal envelope", "error", err)
		span.RecordError(err)
		return err
	}

	var evt domain.BatchScannedEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		slog.ErrorContext(ctx, "batch_scanned: failed to unmarshal payload", "error", err)
		span.RecordError(err)
		return err
	}

	// The account is on the payload so a replay does not depend on the envelope, but an older
	// publisher may only set the identity.
	accountID := evt.AccountID
	if accountID == "" && amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}

	// A malformed event will never become well-formed. Discarding records the drop as terminal and surfaces it to the failure monitor; returning nil would have ACKed it and left the inbox claiming the scan was applied.
	switch {
	case accountID == "":
		slog.ErrorContext(ctx, "batch_scanned: no account on event or identity", "batch_id", evt.BatchID)
		return c.inboxConsumer.Discard(ctx, "no account on event or identity")
	case evt.ProductionStepID == "":
		slog.ErrorContext(ctx, "batch_scanned: no production step on event", "batch_id", evt.BatchID)
		return c.inboxConsumer.Discard(ctx, "no production step on event")
	case evt.BatchID == "":
		slog.ErrorContext(ctx, "batch_scanned: no batch on event")
		return c.inboxConsumer.Discard(ctx, "no batch on event")
	}

	span.SetAttributes(
		attribute.String("batch.id", evt.BatchID),
		attribute.String("batch.production_step_id", evt.ProductionStepID),
		attribute.String("batch.scanning_station_id", evt.ScanningStationID),
		attribute.String("batch.item_id", evt.ItemID),
		attribute.String("batch.account_id", accountID),
	)

	// The item set, resolved on the pool before the transaction opens. applyInventory reads the step
	// again inside the transaction and works from that; this read decides nothing except which roots to
	// take, and taking them after the transaction has written the ledger is the inversion (Corollary A).
	step, apiErr := c.repos.NewProductionStepQueryRepo().Find(ctx, accountID, evt.ProductionStepID)
	if apiErr != nil {
		span.RecordError(apiErr)
		return apiErr
	}
	itemIDs := append([]string{step.Production.ProducedItem.ID}, consumedItemIDs(step)...)

	apiErr = c.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txConsumer := &BatchScannedConsumer{
			rabbitmq:      c.rabbitmq,
			inboxConsumer: c.inboxConsumer,
			repos:         f,
			txManager:     c.txManager,
			tracer:        c.tracer,
		}
		scope, apiErr := ledgerlock.Acquire(txCtx, f.NewInventoryReservationRepo(), itemIDs)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := txConsumer.applyInventory(txCtx, scope, accountID, evt); apiErr != nil {
			return apiErr
		}
		return completeInboxRecord(txCtx, f)
	})
	if apiErr != nil {
		span.RecordError(apiErr)
		return discardIfPermanent(ctx, c.inboxConsumer, apiErr)
	}
	return nil
}

// scrapMeasure is the part of the scan that will never ship, in the unit the scan was recorded in.
func scrapMeasure(evt domain.BatchScannedEvent) (decimal.Decimal, *apierror.APIError) {
	seconds, err := evt.SecondsDecimal()
	if err != nil {
		return decimal.Zero, apierror.NewInternalError(err, "Invalid seconds measure.")
	}
	waste, err := evt.WasteDecimal()
	if err != nil {
		return decimal.Zero, apierror.NewInternalError(err, "Invalid waste measure.")
	}
	return seconds.Add(waste), nil
}

// applyInventory is the inventory reaction to one scan, and runs inside the transaction.
func (c *BatchScannedConsumer) applyInventory(ctx context.Context, scope *ledgerlock.Scope, accountID string, evt domain.BatchScannedEvent) *apierror.APIError {
	step, apiErr := c.repos.NewProductionStepQueryRepo().Find(ctx, accountID, evt.ProductionStepID)
	if apiErr != nil {
		return apiErr
	}

	// A step that no longer produces what was scanned cannot be reasoned about — the routing changed under the batch. Acking would silently drop the scan's inventory, so this rolls back; it is permanent rather than transient, so it is discarded and alerted on instead of retried three times into the DLQ.
	if step.Production.ProducedItem.ID != evt.ItemID {
		return newPermanentDropError(
			"Production step " + evt.ProductionStepID + " no longer produces the scanned item " + evt.ItemID + ".")
	}
	if step.Production.Quantity.Measure.IsZero() {
		return newPermanentDropError(
			"Production step " + evt.ProductionStepID + " has a zero production quantity.")
	}

	scanned, parseErr := decimal.NewFromString(evt.Measure)
	if parseErr != nil {
		return apierror.NewInternalError(parseErr, "Invalid scanned measure.")
	}

	unitConvRepo := c.repos.NewUnitConversionRepo()
	producedUnitID := step.Production.Quantity.Unit.ID

	// The operator scans in the unit the station works in, which need not be the unit the step is
	// defined in — dozens off a knitting machine against a step written in pairs. Everything below is
	// in the step's unit, so the conversion happens once, here.
	convertedMeasure, apiErr := unitConvRepo.ConvertValue(ctx, scanned, evt.UnitID, producedUnitID)
	if apiErr != nil {
		return apiErr
	}

	producedMultiplier := convertedMeasure.Div(step.Production.Quantity.Measure)
	producedMeasure := step.Production.Quantity.Measure.Mul(producedMultiplier)

	// Materials are burned by everything the step ran, not just the part that came out saleable. A
	// batch that scrapped a third of its output still consumed yarn for that third, so consumption is
	// scaled by good output plus scrap while production is scaled by good output alone.
	consumptionMultiplier := producedMultiplier
	scrap, apiErr := scrapMeasure(evt)
	if apiErr != nil {
		return apiErr
	}
	if scrap.GreaterThan(decimal.Zero) {
		convertedScrap, apiErr := unitConvRepo.ConvertValue(ctx, scrap, evt.UnitID, producedUnitID)
		if apiErr != nil {
			return apiErr
		}
		consumptionMultiplier = convertedMeasure.Add(convertedScrap).Div(step.Production.Quantity.Measure)
	}

	// One collector for the whole scan. Every movement below is applied immediately but logged only
	// once all of them are written, so the level each records is the item's final physical inventory
	// for the scan and comes from a single batched read rather than one aggregation per movement.
	audit := &inventoryAuditCollector{}

	if apiErr := audit.mutate(ctx, c.repos, accountID, domain.InventoryUpdateParams{
		AccountID:         accountID,
		ItemID:            step.Production.ProducedItem.ID,
		Measure:           producedMeasure,
		UnitID:            producedUnitID,
		ActionType:        "scan",
		ScanningStationID: evt.ScanningStationID,
		ResponsibleUserID: evt.ResponsibleUserID,
		BatchID:           &evt.BatchID,
	}); apiErr != nil {
		return apiErr
	}

	// One lineage walk for the whole scan. It answers both questions that follow — which production
	// run ordered the work, and how much scrap accumulated along the way — and resolving them
	// separately meant re-walking the same ancestry for every material on the step.
	lineage, apiErr := c.repos.NewBatchRepo().FindLineageShortfall(ctx, evt.BatchID)
	if apiErr != nil {
		return apiErr
	}

	var orderID *string
	if lineage != nil && lineage.ProductionRunID != "" {
		orderID, apiErr = c.repos.NewOrderQueryRepo().FindIDByProductionRun(ctx, accountID, lineage.ProductionRunID)
		if apiErr != nil {
			return apiErr
		}
	}

	if apiErr := c.releaseShortfallReservations(ctx, scope, accountID, step, lineage, orderID, producedUnitID); apiErr != nil {
		return apiErr
	}

	if apiErr := c.applyConsumptions(ctx, scope, accountID, evt, step, consumptionMultiplier, orderID, audit); apiErr != nil {
		return apiErr
	}

	if apiErr := c.enqueueOpenIssueAllocation(ctx, accountID, step); apiErr != nil {
		return apiErr
	}

	// Levels are stamped last, once every mutation this scan makes is written, so each is the item's
	// final physical inventory for the scan.
	return audit.finalize(ctx, c.repos, accountID)
}

func (c *BatchScannedConsumer) enqueueOpenIssueAllocation(ctx context.Context, accountID string, step *domain.ProductionStepDetail) *apierror.APIError {
	outboxRepo := c.repos.NewOutboxRepo()

	seen := map[string]bool{}
	for _, itemID := range append([]string{step.Production.ProducedItem.ID}, consumedItemIDs(step)...) {
		if itemID == "" || seen[itemID] {
			continue
		}
		seen[itemID] = true
		if apiErr := mediator.EnqueueAllocateOpenIssuesFrom(ctx, outboxRepo, accountID, itemID, time.Time{}, "", ""); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func consumedItemIDs(step *domain.ProductionStepDetail) []string {
	ids := make([]string, 0, len(step.Consumptions))
	for _, consumption := range step.Consumptions {
		ids = append(ids, consumption.ConsumedItem.ID)
	}
	return ids
}

// releaseShortfallReservations hands back the reservations covering units seconds and waste mean the
// batch will never deliver.
func (c *BatchScannedConsumer) releaseShortfallReservations(
	ctx context.Context,
	scope *ledgerlock.Scope,
	accountID string,
	step *domain.ProductionStepDetail,
	lineage *domain.LineageShortfall,
	orderID *string,
	producedUnitID string,
) *apierror.APIError {
	if lineage == nil || orderID == nil {
		return nil
	}

	shortfall := lineage.Total()
	if shortfall.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	reservationRepo := c.repos.NewInventoryReservationRepo()
	if apiErr := reservationRepo.ReduceReservedForOrderItem(ctx, scope, domain.OrderReservationReductionParams{
		OrderID:   *orderID,
		AccountID: accountID,
		ItemID:    step.Production.ProducedItem.ID,
		Measure:   shortfall,
		UnitID:    producedUnitID,
	}); apiErr != nil {
		return apiErr
	}

	demands, apiErr := c.repos.NewMaterialDemandRepo().GetMaterialDemand(
		ctx, accountID, step.Production.ProducedItem.ID, shortfall, producedUnitID)
	if apiErr != nil {
		return apiErr
	}
	if len(demands) == 0 {
		return nil
	}

	return reservationRepo.ReduceReservedForOrderMaterials(ctx, scope, *orderID, accountID, demands)
}

// applyConsumptions draws down the materials the step consumed.
//
// The production run and its order are resolved once for the whole step. Resolving them per
// consumption walked the same batch lineage again for every material on the step, which for a step
// with a long bill of materials was the bulk of the work.
func (c *BatchScannedConsumer) applyConsumptions(
	ctx context.Context,
	scope *ledgerlock.Scope,
	accountID string,
	evt domain.BatchScannedEvent,
	step *domain.ProductionStepDetail,
	multiplier decimal.Decimal,
	orderID *string,
	audit *inventoryAuditCollector,
) *apierror.APIError {
	if len(step.Consumptions) == 0 {
		return nil
	}

	reservationRepo := c.repos.NewInventoryReservationRepo()

	for _, consumption := range step.Consumptions {
		// Waste on the consumption is material the step burns without it reaching the product, so it
		// is drawn down alongside what the product actually takes.
		perUnit := consumption.Quantity.Measure.Add(consumption.WasteQuantity.Measure)
		consumedMeasure := perUnit.Mul(multiplier)
		if consumedMeasure.IsZero() {
			continue
		}
		consumedUnitID := consumption.Quantity.Unit.ID

		// With no order behind the batch there is no reservation to draw from, so the material comes
		// straight off the shelf.
		if orderID == nil {
			if apiErr := audit.mutate(ctx, c.repos, accountID, domain.InventoryUpdateParams{
				AccountID:         accountID,
				ItemID:            consumption.ConsumedItem.ID,
				Measure:           consumedMeasure.Neg(),
				UnitID:            consumedUnitID,
				ActionType:        "scan",
				ScanningStationID: evt.ScanningStationID,
				ResponsibleUserID: evt.ResponsibleUserID,
				BatchID:           &evt.BatchID,
			}); apiErr != nil {
				return apiErr
			}
			continue
		}

		result, apiErr := reservationRepo.AllocateReservationsForConsumption(ctx, scope, domain.ConsumptionAllocationParams{
			OrderID:         *orderID,
			AccountID:       accountID,
			ItemID:          consumption.ConsumedItem.ID,
			Measure:         consumedMeasure,
			UnitID:          consumedUnitID,
			ProducedBatchID: evt.BatchID,
		})
		if apiErr != nil {
			return apiErr
		}

		// Whatever the reservations did not cover is taken from open stock.
		if result == nil || result.RemainingMeasure.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if apiErr := audit.mutate(ctx, c.repos, accountID, domain.InventoryUpdateParams{
			AccountID:         accountID,
			ItemID:            consumption.ConsumedItem.ID,
			Measure:           result.RemainingMeasure.Neg(),
			UnitID:            result.RemainingUnitID,
			ActionType:        "scan",
			ScanningStationID: evt.ScanningStationID,
			ResponsibleUserID: evt.ResponsibleUserID,
			BatchID:           &evt.BatchID,
		}); apiErr != nil {
			return apiErr
		}
	}

	return nil
}
