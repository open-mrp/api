package event

import (
	"context"
	"encoding/json"
	"log"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ExecuteProductionStepConsumer processes execute-production-step commands. It calculates inventory changes from batch mutations (initialize, move, merge, split) and creates inventory receipts/issues accordingly.
type ExecuteProductionStepConsumer struct {
	rabbitmq      messaging.MessageBroker
	inboxConsumer *messaging.InboxConsumer
	queries       *sqlc.Queries
	repos         domain.RepoFactory
	tracer        trace.Tracer
}

func NewExecuteProductionStepConsumer(
	rabbitmq messaging.MessageBroker,
	inboxRepo messaging.InboxRepo,
	queries *sqlc.Queries,
	repos domain.RepoFactory,
) *ExecuteProductionStepConsumer {
	return &ExecuteProductionStepConsumer{
		rabbitmq:      rabbitmq,
		inboxConsumer: messaging.NewInboxConsumer(inboxRepo, "core-service"),
		queries:       queries,
		repos:         repos,
		tracer:        tracing.GetTracer("core-service.execute_production_step_consumer"),
	}
}

func (c *ExecuteProductionStepConsumer) Listen(ctx context.Context) error {
	return c.rabbitmq.ConsumeMessages(ctx, messaging.CoreCmdExecuteProductionStepQueue,
		c.inboxConsumer.Wrap("core.execute_production_step", c.handleMessage))
}

func (c *ExecuteProductionStepConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := c.tracer.Start(ctx, "consumer.execute_production_step",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[execute_production_step] Failed to unmarshal message: %v", err)
		span.RecordError(err)
		return err
	}

	var evt domain.ExecuteProductionStepEvent
	if err := json.Unmarshal(amqpMsg.Data, &evt); err != nil {
		log.Printf("[execute_production_step] Failed to unmarshal event payload: %v", err)
		span.RecordError(err)
		return err
	}

	if evt.ProductionStepID == "" {
		log.Printf("[execute_production_step] Empty production step ID in event")
		return nil
	}

	// Extract account ID from identity.
	accountID := ""
	if amqpMsg.Identity != nil && amqpMsg.Identity.Target != nil {
		accountID = amqpMsg.Identity.Target.AccountID
	}
	if accountID == "" {
		log.Printf("[execute_production_step] No account ID in message identity")
		return nil
	}

	span.SetAttributes(
		attribute.String("batch.production_step_id", evt.ProductionStepID),
		attribute.String("batch.scanning_station_id", evt.ScanningStationID),
		attribute.String("batch.item_id", evt.ItemID),
		attribute.String("batch.account_id", accountID),
		attribute.Bool("batch.produce_inventory", evt.ProduceInventory),
	)

	log.Printf("[execute_production_step] Processing: step=%s item=%s account=%s produceInventory=%v",
		evt.ProductionStepID, evt.ItemID, accountID, evt.ProduceInventory)

	return c.executeProductionStep(ctx, accountID, evt)
}

// executeProductionStep implements the core logic ported from dashboard/apps/api/src/repositories/production-step.repo.ts:1142-1434.
//
// Since batch service events always use partUsageType=produced and undo=false, this consumer only implements that path.
func (c *ExecuteProductionStepConsumer) executeProductionStep(ctx context.Context, accountID string, evt domain.ExecuteProductionStepEvent) error {
	stepRepo := c.repos.NewProductionStepQueryRepo()
	unitConvRepo := c.repos.NewUnitConversionRepo()

	// 1. Fetch the production step.
	step, apiErr := stepRepo.Find(ctx, accountID, evt.ProductionStepID)
	if apiErr != nil {
		log.Printf("[execute_production_step] Failed to find production step %s: %v", evt.ProductionStepID, apiErr)
		return apiErr
	}

	// 2. Parse batch measure.
	batchMeasure, err := decimal.NewFromString(evt.BatchMeasure)
	if err != nil {
		log.Printf("[execute_production_step] Failed to parse batch measure %q: %v", evt.BatchMeasure, err)
		return err
	}

	// 3. Calculate execution multiplier by production.
	// multiplier = convertedMeasure / step.production.quantity.measure
	if step.Production.ProducedItem.ID != evt.ItemID {
		log.Printf("[execute_production_step] Item mismatch: step produces %s but event has %s",
			step.Production.ProducedItem.ID, evt.ItemID)
		return nil
	}

	convertedMeasure, apiErr := unitConvRepo.ConvertValue(ctx, batchMeasure, evt.BatchUnitID, step.Production.Quantity.Unit.ID)
	if apiErr != nil {
		log.Printf("[execute_production_step] Unit conversion failed: %v", apiErr)
		return apiErr
	}

	if step.Production.Quantity.Measure.IsZero() {
		log.Printf("[execute_production_step] Production quantity measure is zero for step %s", evt.ProductionStepID)
		return nil
	}

	multiplier := convertedMeasure.Div(step.Production.Quantity.Measure)

	// 4. Handle produced inventory.
	if evt.ProduceInventory {
		producedMeasure := step.Production.Quantity.Measure.Mul(multiplier)
		producedUnitID := step.Production.Quantity.Unit.ID

		// Create inventory receipt for the produced item (positive = receipt).
		apiErr = c.updateInventoryWithAudit(ctx, accountID, domain.InventoryUpdateParams{
			AccountID:         accountID,
			ItemID:            step.Production.ProducedItem.ID,
			Measure:           producedMeasure,
			UnitID:            producedUnitID,
			ActionType:        "scan",
			ScanningStationID: evt.ScanningStationID,
			ResponsibleUserID: evt.ResponsibleUserID,
			BatchID:           evt.ProducedBatchID,
		})
		if apiErr != nil {
			log.Printf("[execute_production_step] Failed to update produced inventory: %v", apiErr)
			return apiErr
		}

		// If there's a produced batch, handle seconds/waste shortfall and order reservations.
		if evt.ProducedBatchID != nil {
			if err := c.handleProducedBatchShortfall(ctx, accountID, evt, step, producedMeasure, producedUnitID); err != nil {
				log.Printf("[execute_production_step] Failed to handle shortfall: %v", err)
				return err
			}
		}
	}

	// 5. Handle consumptions.
	for _, consumption := range step.Consumptions {
		// consumedQuantity = (consumption.quantity + consumption.wasteQuantity) * multiplier
		totalConsumptionMeasure := consumption.Quantity.Measure.Add(consumption.WasteQuantity.Measure)
		consumedMeasure := totalConsumptionMeasure.Mul(multiplier)

		// Negate: consumption decreases inventory (not undo, so we negate).
		consumedMeasure = consumedMeasure.Neg()
		consumedUnitID := consumption.Quantity.Unit.ID

		if evt.ProducedBatchID != nil {
			if err := c.handleConsumptionWithOrder(ctx, accountID, evt, consumption, consumedMeasure, consumedUnitID); err != nil {
				log.Printf("[execute_production_step] Failed to handle consumption with order: %v", err)
				return err
			}
		} else {
			// No produced batch — direct inventory update.
			apiErr = c.updateInventoryWithAudit(ctx, accountID, domain.InventoryUpdateParams{
				AccountID:         accountID,
				ItemID:            consumption.ConsumedItem.ID,
				Measure:           consumedMeasure,
				UnitID:            consumedUnitID,
				ActionType:        "scan",
				ScanningStationID: evt.ScanningStationID,
				ResponsibleUserID: evt.ResponsibleUserID,
				BatchID:           evt.ProducedBatchID,
			})
			if apiErr != nil {
				return apiErr
			}
		}
	}

	log.Printf("[execute_production_step] Completed: step=%s account=%s", evt.ProductionStepID, accountID)
	return nil
}

// handleProducedBatchShortfall handles the seconds/waste shortfall logic.
//
// When a batch has seconds or waste, the produced quantity is reduced, and we need to reduce the reservation on the associated order by the shortfall amount.
func (c *ExecuteProductionStepConsumer) handleProducedBatchShortfall(
	ctx context.Context,
	accountID string,
	evt domain.ExecuteProductionStepEvent,
	step *domain.ProductionStepDetail,
	producedMeasure decimal.Decimal,
	producedUnitID string,
) error {
	if evt.ProducedBatchID == nil {
		return nil
	}

	// BFS up the batch lineage to find productionRunID and accumulate seconds/waste.
	productionRunID, secondsSum, wasteSum, err := c.bfsForProductionRunAndShortfall(ctx, *evt.ProducedBatchID)
	if err != nil {
		return err
	}

	if productionRunID == "" {
		return nil
	}

	shortfall := secondsSum.Add(wasteSum)
	if shortfall.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	// Find the order for this production run.
	orderQueryRepo := c.repos.NewOrderQueryRepo()
	orderID, apiErr := orderQueryRepo.FindIDByProductionRun(ctx, accountID, productionRunID)
	if apiErr != nil {
		return apiErr
	}
	if orderID == nil {
		return nil
	}

	// Reduce reserved quantity for the order item.
	reservationRepo := c.repos.NewInventoryReservationRepo()
	apiErr = reservationRepo.ReduceReservedForOrderItem(ctx, domain.OrderReservationReductionParams{
		OrderID:   *orderID,
		AccountID: accountID,
		ItemID:    step.Production.ProducedItem.ID,
		Measure:   shortfall,
		UnitID:    producedUnitID,
	})
	if apiErr != nil {
		return apiErr
	}

	// Release upstream material reservations proportionally.
	materialRepo := c.repos.NewMaterialDemandRepo()
	demands, apiErr := materialRepo.GetMaterialDemand(ctx, accountID, step.Production.ProducedItem.ID, shortfall, producedUnitID)
	if apiErr != nil {
		return apiErr
	}

	if len(demands) > 0 {
		apiErr = reservationRepo.ReduceReservedForOrderMaterials(ctx, *orderID, accountID, demands)
		if apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// handleConsumptionWithOrder handles consumption inventory changes when there's an associated order. It tries to allocate from existing reservations first, then falls back to direct inventory update.
func (c *ExecuteProductionStepConsumer) handleConsumptionWithOrder(
	ctx context.Context,
	accountID string,
	evt domain.ExecuteProductionStepEvent,
	consumption domain.StepConsumption,
	consumedMeasure decimal.Decimal,
	consumedUnitID string,
) error {
	// BFS to find productionRunID.
	productionRunID, err := c.bfsForProductionRunID(ctx, *evt.ProducedBatchID)
	if err != nil {
		return err
	}

	if productionRunID == "" {
		// No production run found — direct inventory update.
		apiErr := c.updateInventoryWithAudit(ctx, accountID, domain.InventoryUpdateParams{
			AccountID:         accountID,
			ItemID:            consumption.ConsumedItem.ID,
			Measure:           consumedMeasure,
			UnitID:            consumedUnitID,
			ActionType:        "scan",
			ScanningStationID: evt.ScanningStationID,
			ResponsibleUserID: evt.ResponsibleUserID,
			BatchID:           evt.ProducedBatchID,
		})
		if apiErr != nil {
			return apiErr
		}
		return nil
	}

	// Find order.
	orderQueryRepo := c.repos.NewOrderQueryRepo()
	orderID, apiErr := orderQueryRepo.FindIDByProductionRun(ctx, accountID, productionRunID)
	if apiErr != nil {
		return apiErr
	}

	if orderID == nil {
		// No order — direct inventory update.
		apiErr = c.updateInventoryWithAudit(ctx, accountID, domain.InventoryUpdateParams{
			AccountID:         accountID,
			ItemID:            consumption.ConsumedItem.ID,
			Measure:           consumedMeasure,
			UnitID:            consumedUnitID,
			ActionType:        "scan",
			ScanningStationID: evt.ScanningStationID,
			ResponsibleUserID: evt.ResponsibleUserID,
			BatchID:           evt.ProducedBatchID,
		})
		if apiErr != nil {
			return apiErr
		}
		return nil
	}

	// Try to allocate from existing reservations.
	reservationRepo := c.repos.NewInventoryReservationRepo()
	absMeasure := consumedMeasure.Abs()
	result, apiErr := reservationRepo.AllocateReservationsForConsumption(ctx, domain.ConsumptionAllocationParams{
		OrderID:   *orderID,
		AccountID: accountID,
		ItemID:    consumption.ConsumedItem.ID,
		Measure:   absMeasure,
		UnitID:    consumedUnitID,
	})
	if apiErr != nil {
		return apiErr
	}

	// If there's remaining quantity not covered by reservations, update inventory directly.
	if result != nil && result.RemainingMeasure.GreaterThan(decimal.Zero) {
		remaining := result.RemainingMeasure.Neg() // Negate for consumption.
		apiErr = c.updateInventoryWithAudit(ctx, accountID, domain.InventoryUpdateParams{
			AccountID:         accountID,
			ItemID:            consumption.ConsumedItem.ID,
			Measure:           remaining,
			UnitID:            result.RemainingUnitID,
			ActionType:        "scan",
			ScanningStationID: evt.ScanningStationID,
			ResponsibleUserID: evt.ResponsibleUserID,
			BatchID:           evt.ProducedBatchID,
		})
		if apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// bfsForProductionRunID performs BFS up the batch lineage to find a productionRunID.
func (c *ExecuteProductionStepConsumer) bfsForProductionRunID(ctx context.Context, startBatchID string) (string, error) {
	visited := make(map[string]bool)
	queue := []string{startBatchID}

	for len(queue) > 0 {
		// Fetch batch of IDs.
		currentBatch := queue
		queue = nil

		// Filter unvisited.
		var toFetch []string
		for _, id := range currentBatch {
			if !visited[id] {
				visited[id] = true
				toFetch = append(toFetch, id)
			}
		}
		if len(toFetch) == 0 {
			break
		}

		rows, err := c.queries.FindBatchProductionRunIDAncestry(ctx, toFetch)
		if err != nil {
			return "", err
		}

		for _, row := range rows {
			if row.ProductionRunID.Valid && row.ProductionRunID.String != "" {
				return row.ProductionRunID.String, nil
			}
			if row.ParentID.Valid && row.ParentID.String != "" && !visited[row.ParentID.String] {
				queue = append(queue, row.ParentID.String)
			}
		}
	}

	return "", nil
}

// bfsForProductionRunAndShortfall performs BFS up the batch lineage to find productionRunID and accumulate seconds/waste quantities.
func (c *ExecuteProductionStepConsumer) bfsForProductionRunAndShortfall(
	ctx context.Context, startBatchID string,
) (productionRunID string, secondsSum, wasteSum decimal.Decimal, err error) {
	secondsSum = decimal.Zero
	wasteSum = decimal.Zero
	visited := make(map[string]bool)
	queue := []string{startBatchID}

	for len(queue) > 0 {
		currentBatch := queue
		queue = nil

		var toFetch []string
		for _, id := range currentBatch {
			if !visited[id] {
				visited[id] = true
				toFetch = append(toFetch, id)
			}
		}
		if len(toFetch) == 0 {
			break
		}

		rows, queryErr := c.queries.FindBatchProductionRunIDAncestry(ctx, toFetch)
		if queryErr != nil {
			err = queryErr
			return
		}

		for _, row := range rows {
			if productionRunID == "" && row.ProductionRunID.Valid && row.ProductionRunID.String != "" {
				productionRunID = row.ProductionRunID.String
			}

			// Fetch seconds/waste for this batch.
			swRow, swErr := c.queries.GetBatchSecondsAndWaste(ctx, row.ID)
			if swErr == nil {
				if swRow.SecondsValue.Valid {
					val, _ := decimal.NewFromString(swRow.SecondsValue.String)
					secondsSum = secondsSum.Add(val)
				}
				if swRow.WasteValue.Valid {
					val, _ := decimal.NewFromString(swRow.WasteValue.String)
					wasteSum = wasteSum.Add(val)
				}
			}

			if row.ParentID.Valid && row.ParentID.String != "" && !visited[row.ParentID.String] {
				queue = append(queue, row.ParentID.String)
			}
		}
	}

	return
}
