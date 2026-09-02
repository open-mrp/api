package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

// kickOutbox wakes the outbox enqueuer so a just-committed allocation request is picked up
// immediately rather than on the enqueuer's next idle poll, which can be up to MaxPollInterval away.
// No-op when no notifier was injected. Call only after the writing transaction has committed —
// kicking while it is still open races the poll against a row it cannot yet see.
func (s *itemSvcImpl) kickOutbox() {
	if s.outboxNotifier != nil {
		s.outboxNotifier.Notify()
	}
}

// enqueueInventoryReceived hands allocation to the consumer that owns it.
//
// Written in the transaction that moved the stock, so the handoff commits with the movement or not
// at all — the same arrangement the dashboard has used since allocation stopped happening in-request.
func (s *itemSvcImpl) enqueueInventoryReceived(ctx context.Context, repos domain.RepoFactory, evt domain.InventoryReceivedEvent) *apierror.APIError {
	payload, err := json.Marshal(evt)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal inventory received event.")
	}

	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreEventInventoryReceived),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreEventInventoryReceived),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for inventory received.")
	}

	return nil
}

var itemSvcTracer = tracing.GetTracer("core-service.item_service")

type itemSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	outboxNotifier  messaging.OutboxNotifier
}

type ItemSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// OutboxNotifier (optional; default: nil) wakes the outbox enqueuer the instant an allocation request commits, so reconciled stock is offered to open demand on the next moment rather than on the enqueuer's next idle poll. When nil, the request is still picked up on the next poll.
	OutboxNotifier messaging.OutboxNotifier
}

func (c *ItemSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("item service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("item service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("item service: tx manager is required")
	}
	return nil
}

func NewItemSvc(config *ItemSvcConfig) domain.ItemSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &itemSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		outboxNotifier:  config.OutboxNotifier,
	}
}

func (s *itemSvcImpl) BatchGetItemsByIDs(ctx context.Context, ids []string) ([]*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	meds := s.mediators()
	if apiErr := authorizeCatalogBatchRead(ctx, identity, span, meds, func() *apierror.APIError {
		return identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead)
	}); apiErr != nil {
		return nil, apiErr
	}
	if len(ids) == 0 {
		return nil, nil
	}

	items, apiErr := s.repos.NewItemRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return items, nil
}

// ListItems returns a paginated list of items for the caller's account.
func (s *itemSvcImpl) ListItems(ctx context.Context, params domain.ListItemsParams) (*domain.ListItemsResult, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewItemRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// GetItem returns a single item by ID within the caller's account.
func (s *itemSvcImpl) GetItem(ctx context.Context, itemID string, includes []string) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
		AccountID: identity.Target.AccountID,
		ItemID:    itemID,
		Includes:  includes,
	})
}

// GetItemInventory returns inventory quantities for an item.
func (s *itemSvcImpl) GetItemInventory(ctx context.Context, itemID string) (*domain.ItemInventory, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.get_inventory")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewItemRepo().GetInventory(ctx, identity.Target.AccountID, itemID)
}

// GetItemCosts returns production cost breakdown for an item.
//
// This replicates the Dashboard's fetchCosts logic:
// 1. Find the production step that produces this item.
// 2. BFS backward through the production flow graph to find all contributing steps.
// 3. Calculate per-step costs (labor, overhead, material) using leveling factor and allowances.
// 4. Normalize and aggregate costs across the flow using a forward pass.
// 5. Update the item's unit cost and clear the dirty flag.
func (s *itemSvcImpl) GetItemCosts(ctx context.Context, itemID string) (*domain.ItemCosts, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.get_costs")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.ComputeItemCosts(ctx, identity.Target.AccountID, itemID)
}

// RecomputeItemCosts is GetItemCosts with the account named rather than read off the caller.
//
// Costing is also driven by events now — a material's price moving restates every part built from it
// — and a consumer has no caller to authorize. Rather than have it mint an identity that would
// satisfy the checks above, which is authorization theatre and would let a bug reach another
// tenant's data with a forged actor, the account travels on the event and is passed in here. The
// permission gate stays where a request enters.
func (s *itemSvcImpl) RecomputeItemCosts(ctx context.Context, accountID, itemID string) (*domain.ItemCosts, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.recompute_costs")
	defer span.End()

	costs, apiErr := s.ComputeItemCosts(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	total, err := decimal.NewFromString(costs.TotalCost)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Rolled-up cost is not a number."))
	}

	if updateErr := s.repos.NewItemRepo().UpdateUnitCost(ctx, accountID, itemID, total, costs.UnitID); updateErr != nil {
		return nil, tracing.Trace(span, updateErr)
	}

	return costs, nil
}

// ComputeItemCosts rolls an item's cost up from its production steps and the unit costs of what they
// consume, and returns it without storing anything.
//
// Separate from RecomputeItemCosts because the read path must not write: GET item costs served the
// rollup and persisted it on the way out, so every page view restated the item, and a rollup defect
// reached the database as fast as anyone could look at a page.
func (s *itemSvcImpl) ComputeItemCosts(ctx context.Context, accountID, itemID string) (*domain.ItemCosts, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.compute_costs")
	defer span.End()

	flowRepo := s.repos.NewProductionFlowRepo()
	itemRepo := s.repos.NewItemRepo()

	// 1. Find the production step(s) that produce this item.
	initialStepIDs, apiErr := flowRepo.FindStepsByProducedItem(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(initialStepIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production flow not found."))
	}

	// 2. Get the full edge graph for the account and BFS backward.
	edges, apiErr := flowRepo.GetAllStepEdgesForAccount(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	parentMap := make(map[string][]string)
	childMap := make(map[string][]string)
	for _, edge := range edges {
		parentMap[edge.ChildStepID] = append(parentMap[edge.ChildStepID], edge.ParentStepID)
		childMap[edge.ParentStepID] = append(childMap[edge.ParentStepID], edge.ChildStepID)
	}

	relevantStepIDs := make(map[string]bool)
	queue := make([]string, 0, len(initialStepIDs))
	queue = append(queue, initialStepIDs...)

	for len(queue) > 0 {
		stepID := queue[0]
		queue = queue[1:]
		if relevantStepIDs[stepID] {
			continue
		}
		relevantStepIDs[stepID] = true
		for _, parentID := range parentMap[stepID] {
			if !relevantStepIDs[parentID] {
				queue = append(queue, parentID)
			}
		}
	}

	// 3. Fetch all step data.
	stepQueryRepo := s.repos.NewProductionStepQueryRepo()
	type flowStepData struct {
		step         *domain.ProductionFlowStep
		consumptions []domain.CostFlowConsumption
	}
	stepDataMap := make(map[string]*flowStepData, len(relevantStepIDs))

	for stepID := range relevantStepIDs {
		step, apiErr := flowRepo.GetFlowStep(ctx, accountID, stepID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Get consumptions with item type and unit cost for cost calculation.
		consumptions, apiErr := itemRepo.GetCostFlowConsumptions(ctx, stepID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Also get structural consumptions for the flow graph.
		stepDetail, apiErr := stepQueryRepo.Find(ctx, accountID, stepID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		step.Consumptions = stepDetail.Consumptions

		// Compute InStepIDs/OutStepIDs.
		inIDs := make([]string, 0)
		for _, parentID := range parentMap[stepID] {
			if relevantStepIDs[parentID] {
				inIDs = append(inIDs, parentID)
			}
		}
		step.InStepIDs = inIDs

		outIDs := make([]string, 0)
		for _, childID := range childMap[stepID] {
			if relevantStepIDs[childID] {
				outIDs = append(outIDs, childID)
			}
		}
		step.OutStepIDs = outIDs

		stepDataMap[stepID] = &flowStepData{step: step, consumptions: consumptions}
	}

	// 4. Calculate per-step costs.
	costMap := make(map[string]*itemStepCost, len(stepDataMap))

	for stepID, data := range stepDataMap {
		cost := calculateStepCost(data.step, data.consumptions)
		costMap[stepID] = cost
	}

	// 5. Calculate normalization factors.
	// The target step is the one that produces our item.
	targetStepID := initialStepIDs[0]
	targetStep := stepDataMap[targetStepID].step
	targetProdQty := targetStep.Production.Quantity.Measure

	normMap := make(map[string]decimal.Decimal, len(stepDataMap))

	// Target step normalization: 1 / production quantity (cost per 1 unit of output).
	if targetProdQty.IsZero() {
		return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Target production quantity is zero."))
	}
	normMap[targetStepID] = decimal.NewFromInt(1).Div(targetProdQty)

	// BFS backward from target to compute normalization factors.
	// For each parent step, the normalization factor is:
	// parentNorm = childNorm * (consumption quantity of parent's produced item in child step / parent's production quantity)
	normQueue := []string{targetStepID}
	normVisited := map[string]bool{targetStepID: true}

	for len(normQueue) > 0 {
		currentID := normQueue[0]
		normQueue = normQueue[1:]
		currentData := stepDataMap[currentID]
		currentNorm := normMap[currentID]

		for _, parentID := range currentData.step.InStepIDs {
			if normVisited[parentID] {
				continue
			}
			normVisited[parentID] = true

			parentData := stepDataMap[parentID]
			parentProdQty := parentData.step.Production.Quantity.Measure

			if parentProdQty.IsZero() {
				normMap[parentID] = decimal.Zero
				normQueue = append(normQueue, parentID)
				continue
			}

			// Find how much of the parent's produced item is consumed by the current step.
			consumedQty := decimal.Zero
			parentProducedItemID := parentData.step.Production.ProducedItem.ID
			for _, cons := range currentData.step.Consumptions {
				if cons.ConsumedItem.ID == parentProducedItemID {
					consumedQty = consumedQty.Add(cons.Quantity.Measure)
					break
				}
			}

			// parentNorm = currentNorm * consumedQty / parentProdQty
			normMap[parentID] = currentNorm.Mul(consumedQty).Div(parentProdQty)
			normQueue = append(normQueue, parentID)
		}
	}

	// 6. Forward pass: aggregate normalized costs from leaf nodes to target.
	totalMaterial := decimal.Zero
	totalLabor := decimal.Zero
	totalOverhead := decimal.Zero

	for stepID, cost := range costMap {
		norm, ok := normMap[stepID]
		if !ok {
			norm = decimal.Zero
		}
		totalMaterial = totalMaterial.Add(cost.material.Mul(norm))
		totalLabor = totalLabor.Add(cost.labor.Mul(norm))
		totalOverhead = totalOverhead.Add(cost.overhead.Mul(norm))
	}

	// 7. Restate the costs against the unit the item is stocked in, and write the total back as the item's unit cost.
	stepUnitID := targetStep.Production.Quantity.Unit.ID
	stocking, apiErr := itemRepo.GetStockingUnit(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := ValidateCostDenominatorInUnitGroup(ctx, s.repos.NewUnitRepo(), stocking.UnitGroupID, stepUnitID, "unit_cost"); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	perStockingUnit, apiErr := stepUnitsPerStockingUnit(ctx, s.repos.NewUnitConversionRepo(), stepUnitID, stocking.BaseUnitID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	totalMaterial = totalMaterial.Mul(perStockingUnit)
	totalLabor = totalLabor.Mul(perStockingUnit)
	totalOverhead = totalOverhead.Mul(perStockingUnit)
	totalCost := totalMaterial.Add(totalLabor).Add(totalOverhead)

	return &domain.ItemCosts{
		DirectMaterialCost: totalMaterial.StringFixed(30),
		DirectLaborCost:    totalLabor.StringFixed(30),
		OverheadCost:       totalOverhead.StringFixed(30),
		TotalCost:          totalCost.StringFixed(30),
		UnitID:             stocking.BaseUnitID,
		NumeratorUnitID:    stocking.CostNumeratorUnitID,
	}, nil
}

// stepUnitsPerStockingUnit is the factor that carries a per-step-unit amount onto a per-stocking-unit footing. A rate's denominator scales the opposite way to a measure's, so this is the quantity conversion run backwards.
func stepUnitsPerStockingUnit(ctx context.Context, conv domain.UnitConversionRepo, stepUnitID, stockingUnitID string) (decimal.Decimal, *apierror.APIError) {
	if stepUnitID == stockingUnitID {
		return decimal.NewFromInt(1), nil
	}
	return conv.ConvertValue(ctx, decimal.NewFromInt(1), stockingUnitID, stepUnitID)
}

// baseUnitRatio reads a unit's base ratio, defaulting to 1 so a rate with no unit recorded prices the
// same as it always did rather than collapsing the term to zero.
func baseUnitRatio(ratio string) decimal.Decimal {
	parsed, err := decimal.NewFromString(ratio)
	if err != nil || parsed.IsZero() {
		return decimal.NewFromInt(1)
	}
	return parsed
}

// itemStepCost holds the cost breakdown for a single production step.
type itemStepCost struct {
	material decimal.Decimal
	labor    decimal.Decimal
	overhead decimal.Decimal
	total    decimal.Decimal
}

// calculateStepCost computes the raw cost for a single production step.
// This mirrors the Dashboard's LightProductionStepUtils.fetchLightCostOfStep.
func calculateStepCost(step *domain.ProductionFlowStep, consumptions []domain.CostFlowConsumption) *itemStepCost {
	result := &itemStepCost{
		material: decimal.Zero,
		labor:    decimal.Zero,
		overhead: decimal.Zero,
		total:    decimal.Zero,
	}

	prodQty := step.Production.Quantity.Measure
	if prodQty.IsZero() {
		return result
	}

	// Parse step parameters.
	levelingFactor, _ := decimal.NewFromString(step.LevelingFactor)
	allowances, _ := decimal.NewFromString(step.Allowances)

	// Labor time, carried into the base time unit. A duration and the rate pricing it are each entered
	// in whatever unit suited whoever entered them — seconds a piece against dollars an hour — so both
	// go to base units before they meet, exactly as the material term does. Multiplying them raw prices
	// an hour's labor for every second of it.
	var laborTimeMeasure decimal.Decimal
	if step.LaborTime != nil {
		laborTimeMeasure, _ = decimal.NewFromString(step.LaborTime.Value)
		laborTimeMeasure = laborTimeMeasure.Mul(baseUnitRatio(step.LaborTime.NumeratorRatio))
	}

	// Labor rate, per base time unit.
	var laborRateValue decimal.Decimal
	if step.LaborRate != nil {
		laborRateValue, _ = decimal.NewFromString(step.LaborRate.Value)
		laborRateValue = laborRateValue.Div(baseUnitRatio(step.LaborRate.DenominatorRatio))
	}

	// Overhead rate, per base time unit.
	var overheadRateValue decimal.Decimal
	if step.OverheadRate != nil {
		overheadRateValue, _ = decimal.NewFromString(step.OverheadRate.Value)
		overheadRateValue = overheadRateValue.Div(baseUnitRatio(step.OverheadRate.DenominatorRatio))
	}

	// Corrective factor: (levelingFactor * allowances) + levelingFactor + allowances + 1
	correctiveFactor := levelingFactor.Mul(allowances).Add(levelingFactor).Add(allowances).Add(decimal.NewFromInt(1))

	// Corrected labor time per piece.
	correctedLaborTime := laborTimeMeasure.Mul(correctiveFactor)

	// Total labor time for the batch.
	totalLaborTime := prodQty.Mul(correctedLaborTime)

	// Labor cost = totalLaborTime * laborRate.
	result.labor = totalLaborTime.Mul(laborRateValue)

	// Overhead cost = totalLaborTime * overheadRate.
	result.overhead = totalLaborTime.Mul(overheadRateValue)

	// Material cost: sum of raw material consumption costs (exclude parts and products).
	//
	// Quantity and cost are each recorded in whatever unit was entered, so both go to base units before they meet: eight eaches drawn against a per-carton cost is one carton's worth of money, not eight.
	for _, cons := range consumptions {
		if cons.ConsumedItemType == "part" || cons.ConsumedItemType == "product" {
			continue
		}
		if cons.UnitCostDenominatorRatio.IsZero() {
			continue
		}
		usedInBaseUnits := cons.ConsumptionQuantity.Mul(cons.ConsumptionUnitRatio).Add(cons.WasteQuantity.Mul(cons.WasteUnitRatio))
		costPerBaseUnit := cons.UnitCost.Div(cons.UnitCostDenominatorRatio)
		result.material = result.material.Add(usedInBaseUnits.Mul(costPerBaseUnit))
	}

	result.total = result.material.Add(result.labor).Add(result.overhead)
	return result
}

// GetItemTrends returns historical trend data for an item.
func (s *itemSvcImpl) GetItemTrends(ctx context.Context, itemID string, trendType string) (*domain.ItemTrends, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.get_trends")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Reject unknown trend types — matches Dashboard's guard rail.
	if !constants.ItemTrendType(trendType).IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam(
			"Unsupported trend type. Supported values: "+strings.Join(constants.ItemTrendType("").EnumValues(), ", "),
			"trend_type",
		))
	}

	itemRepo := s.repos.NewItemRepo()

	// An item with nothing logged and an item that does not exist both produce an empty series, so without this read the endpoint would answer for another account's item ID as readily as for a real one of your own.
	if _, apiErr := itemRepo.Get(ctx, domain.GetItemParams{
		AccountID: identity.Target.AccountID,
		ItemID:    itemID,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return itemRepo.GetTrends(ctx, identity.Target.AccountID, itemID, trendType)
}

// ExportItems returns all items with on-hand inventory for the caller's account.
func (s *itemSvcImpl) ExportItems(ctx context.Context) (*domain.ExportItemsResult, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.export")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewItemRepo().ExportWithInventory(ctx, identity.Target.AccountID)
}

func (s *itemSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *itemSvcImpl) withTx(ctx context.Context, fn func(context.Context, *itemSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &itemSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// itemAuditIncludes merges user-requested includes with the includes required for correct audit change tracking (attributes).
func itemAuditIncludes(userIncludes []string) []string {
	auditRequired := []string{"attributes"}
	merged := make([]string, len(auditRequired))
	copy(merged, auditRequired)
	for _, inc := range userIncludes {
		if !slices.Contains(merged, inc) {
			merged = append(merged, inc)
		}
	}
	return merged
}

// UpdateItem partially updates an item (sku, description, notes).
func (s *itemSvcImpl) UpdateItem(ctx context.Context, params domain.UpdateItemParams) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Item](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Item
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: params.AccountID,
				ItemID:    params.ItemID,
			})
			if apiErr != nil {
				return apiErr
			}

			// Check SKU uniqueness if being updated
			if params.SKU != nil {
				exists, apiErr := txRepo.CheckSKUExists(txCtx, params.AccountID, *params.SKU, params.ItemID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam(fmt.Sprintf("Item sku %s already exists.", *params.SKU), "sku")
				}
			}

			if apiErr := txRepo.Update(txCtx, params); apiErr != nil {
				return apiErr
			}

			item, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: params.AccountID,
				ItemID:    params.ItemID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItem,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// AddItemAttribute adds an attribute to an item.
func (s *itemSvcImpl) AddItemAttribute(ctx context.Context, itemID, attributeID string, includes []string) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.add_attribute")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Item](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Item
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemRepo()

			auditIncs := itemAuditIncludes(includes)

			old, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  auditIncs,
			})
			if apiErr != nil {
				return apiErr
			}

			if apiErr := validateAttributesForCategory(txCtx, txSvc.repos, accountID, old.ItemCategoryID, []string{attributeID}, "attribute_id"); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.AddAttribute(txCtx, domain.AddItemAttributeParams{
				AccountID:   accountID,
				ItemID:      itemID,
				AttributeID: attributeID,
			}); apiErr != nil {
				return apiErr
			}

			item, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  auditIncs,
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

			changes := audit.ComputeChanges(old, item)

			// Adding an already-associated attribute is a documented no-op; skip the publish when nothing actually changed.
			if len(changes) > 0 {
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeItem,
					ResourceID:   item.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// RemoveItemAttribute removes an attribute from an item.
func (s *itemSvcImpl) RemoveItemAttribute(ctx context.Context, itemID, attributeID string, includes []string) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.remove_attribute")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Item](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Item
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemRepo()

			auditIncs := itemAuditIncludes(includes)

			old, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  auditIncs,
			})
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.RemoveAttribute(txCtx, domain.RemoveItemAttributeParams{
				AccountID:   accountID,
				ItemID:      itemID,
				AttributeID: attributeID,
			}); apiErr != nil {
				return apiErr
			}

			item, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  auditIncs,
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

			changes := audit.ComputeChanges(old, item)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItem,
				ResourceID:   item.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func categoryTypeMatchesItem(itemTypeCode, categoryTypeCode string) bool {
	switch constants.ItemTypeCode(itemTypeCode) {
	case constants.ItemTypeCodeMaterial:
		return categoryTypeCode == string(constants.ItemCategoryTypeMaterial)
	case constants.ItemTypeCodeProduct, constants.ItemTypeCodePart:
		return categoryTypeCode == string(constants.ItemCategoryTypeProduct)
	default:
		return false
	}
}

func validateChangeItemCategoryTypes(item *domain.Item, category *domain.ItemCategoryFull) *apierror.APIError {
	if item == nil || category == nil {
		return apierror.NewInvariantViolationError("Item and category are required for category change validation.")
	}
	if !categoryTypeMatchesItem(item.ItemTypeCode, category.ItemCategoryTypeCode) {
		return apierror.NewValidationErrorWithParam("This category type cannot be assigned to this item type.", "category_id")
	}
	return nil
}

// ChangeItemCategory changes the category of an item and updates rate units.
func (s *itemSvcImpl) ChangeItemCategory(ctx context.Context, itemID, categoryID string, includes []string) (*domain.Item, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.change_category")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Item](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Item
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewItemRepo()

			auditIncs := itemAuditIncludes(includes)

			itemForValidation, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  auditIncs,
			})
			if apiErr != nil {
				return apiErr
			}
			category, apiErr := txSvc.repos.NewItemCategoryRepo().Get(txCtx, domain.GetItemCategoryParams{
				AccountID:      accountID,
				ItemCategoryID: categoryID,
			})
			if apiErr != nil {
				return apiErr
			}
			if apiErr := validateChangeItemCategoryTypes(itemForValidation, category); apiErr != nil {
				return apiErr
			}
			if apiErr := validateCategoryCarriesItemAttributes(txCtx, txSvc.repos, itemForValidation, categoryID, "category_id"); apiErr != nil {
				return apiErr
			}

			// Get the base unit of the new category (type already validated above).
			baseUnitID, _, apiErr := txRepo.GetCategoryBaseUnitID(txCtx, categoryID)
			if apiErr != nil {
				return apiErr
			}

			// Update the item's category
			if apiErr := txRepo.ChangeCategory(txCtx, domain.ChangeItemCategoryParams{
				AccountID:  accountID,
				ItemID:     itemID,
				CategoryID: categoryID,
			}); apiErr != nil {
				return apiErr
			}

			// Update all rate units to the new category's base unit
			if apiErr := txRepo.UpdateRateUnits(txCtx, accountID, itemID, baseUnitID); apiErr != nil {
				return apiErr
			}

			// Update material order point unit (no-op if item is not a material)
			if apiErr := txRepo.UpdateMaterialOrderPointUnit(txCtx, accountID, itemID, baseUnitID); apiErr != nil {
				return apiErr
			}

			// Update consumption and production quantity units
			if apiErr := txRepo.UpdateConsumptionProductionQuantityUnits(txCtx, accountID, itemID, baseUnitID); apiErr != nil {
				return apiErr
			}

			item, apiErr := txRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  auditIncs,
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

			changes := audit.ComputeChanges(itemForValidation, item)

			// Re-assigning the item's current category is a no-op; skip the publish when nothing actually changed.
			if len(changes) > 0 {
				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeItem,
					ResourceID:   item.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// ListInventories returns all items with their on-hand inventory quantities.
func (s *itemSvcImpl) ListInventories(ctx context.Context, params domain.ListInventoriesParams) (*domain.ListInventoriesResult, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.list_inventories")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	// Fetch items with pagination and query
	itemResult, apiErr := s.repos.NewItemRepo().List(ctx, domain.ListItemsParams{
		AccountID: accountID,
		Cursor:    params.Cursor,
		Limit:     params.Limit,
		Query:     params.Query,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(itemResult.Items) == 0 {
		return &domain.ListInventoriesResult{Items: nil, Count: 0, PageInfo: itemResult.PageInfo}, nil
	}

	// Collect item IDs
	itemIDs := make([]string, len(itemResult.Items))
	for i, item := range itemResult.Items {
		itemIDs[i] = item.ID
	}

	// Fetch bulk on-hand inventory
	inventoryData, apiErr := s.repos.NewInventoryQueryRepo().FetchOnHandInventoryBulk(ctx, itemIDs, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build inventory map
	invMap := make(map[string]*domain.BulkOnHandInventory, len(inventoryData))
	for _, inv := range inventoryData {
		invMap[inv.ItemID] = inv
	}

	// Merge items with inventory
	results := make([]*domain.InventoryItemResult, len(itemResult.Items))
	for i, item := range itemResult.Items {
		result := &domain.InventoryItemResult{Item: item}
		if inv, ok := invMap[item.ID]; ok {
			result.OnHandQuantity = inv.OnHandQuantity
			result.OnHandUnitID = inv.UnitID
			result.OnHandUnitAbbrev = inv.UnitAbbreviation
			result.OnHandUnitType = inv.UnitType
		}
		results[i] = result
	}

	return &domain.ListInventoriesResult{
		Items:    results,
		Count:    int64(len(results)),
		PageInfo: itemResult.PageInfo,
	}, nil
}

// UpdateItemInventory adjusts or reconciles inventory for an item.
func (s *itemSvcImpl) UpdateItemInventory(ctx context.Context, params domain.UpdateItemInventoryParams) *apierror.APIError {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.update_item_inventory")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	params.AccountID = accountID
	inventoryOwnerAccountID := accountID
	if params.CustomerID != nil {
		inventoryOwnerAccountID = *params.CustomerID
	}

	var responsibleUserID *string
	if identity.Actor != nil {
		responsibleUserID = &identity.Actor.ID
	}

	// Verify item exists.
	item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
		AccountID: accountID,
		ItemID:    params.ItemID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if item == nil {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Item not found."))
	}

	// If customerID is provided, verify edit access.
	if params.CustomerID != nil {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, accountID, *params.CustomerID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// If lotNumber is provided, upsert the lot.
	var lotID *string
	if params.LotNumber != nil && *params.LotNumber != "" {
		generatedLotID, genErr := id.GenID(id.LotIDPrefix, nil)
		if genErr != nil {
			return tracing.Trace(span, genErr)
		}
		actualLotID, apiErr := s.repos.NewReceivingOrderRepo().UpsertLot(ctx, generatedLotID, accountID, params.ItemID, *params.LotNumber)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		lotID = &actualLotID
	}

	// If locationID is provided, verify it belongs to the account.
	if params.LocationID != nil {
		inAccount, apiErr := s.repos.NewLocationRepo().IsInAccount(ctx, accountID, *params.LocationID)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if !inAccount {
			return tracing.Trace(span, apierror.NewResourceNotFoundError("Location not found."))
		}
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			invQueryRepo := txSvc.repos.NewInventoryQueryRepo()
			invMutRepo := txSvc.repos.NewInventoryMutationRepo()
			// One item, known before the transaction opened, so the root is its first statement.
			scope, apiErr := ledgerlock.Acquire(txCtx, invMutRepo, []string{params.ItemID})
			if apiErr != nil {
				return apiErr
			}

			quantityChange := params.Measure
			reconcile := params.Reconcile != nil && *params.Reconcile
			unitID := params.UnitID

			// Read in that same unit: a reconcile subtracts the two, and a target in pairs measured
			// against a level in each is a correction to a number the operator never saw.
			// (Always the main account, matching Dashboard behavior.)
			currentPhysical, apiErr := invQueryRepo.FetchPhysicalInventory(txCtx, params.ItemID, accountID, unitID)
			if apiErr != nil {
				return apiErr
			}

			// Calculate delta based on reconcile mode.
			var delta decimal.Decimal
			if reconcile {
				delta = quantityChange.Sub(currentPhysical)
			} else {
				delta = quantityChange
			}

			// Skip if no change and not reconciling.
			if delta.IsZero() && !reconcile {
				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
			}

			if delta.GreaterThan(decimal.Zero) {
				// Positive delta: create inventory receipt.
				if apiErr := invMutRepo.CreateInventoryReceipt(txCtx, scope, domain.CreateInventoryReceiptParams{
					AccountID:       accountID,
					OwnerAccountID:  inventoryOwnerAccountID,
					HolderAccountID: accountID,
					ItemID:          params.ItemID,
					Measure:         delta,
					UnitID:          unitID,
					LocationID:      params.LocationID,
					LotID:           lotID,
				}); apiErr != nil {
					return apiErr
				}
			} else if delta.LessThan(decimal.Zero) {
				// Negative delta: create inventory issue.
				if apiErr := invMutRepo.CreateInventoryIssue(txCtx, scope, domain.CreateInventoryIssueParams{
					AccountID:  accountID,
					ItemID:     params.ItemID,
					Measure:    delta,
					UnitID:     unitID,
					LocationID: params.LocationID,
					LotID:      lotID,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Allocation walks every open issue for the item and locks the receipts it draws on, which
			// is not work to hold a request open for. The message commits with the movement that
			// caused it, and the consumer does it just after.
			if apiErr := txSvc.enqueueInventoryReceived(txCtx, txSvc.repos, domain.InventoryReceivedEvent{
				AccountID: accountID,
				ItemIDs:   []string{params.ItemID},
				Reason:    "inventory_updated",
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := mediator.RecordInventoryAuditTrail(
				txCtx,
				txSvc.repos,
				accountID,
				params.ItemID,
				delta,
				unitID,
				"user_correction",
				nil,
				responsibleUserID,
			); apiErr != nil {
				return apiErr
			}

			// Empty when delta is zero (reconcile to the same quantity), in which case the publisher skips the event as a no-op.
			var changes []audit.FieldChange
			if !delta.IsZero() {
				changes = append(changes, audit.NewFieldChange("quantity", currentPhysical, currentPhysical.Add(delta)))
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItem,
				ResourceID:   params.ItemID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// BulkCreateItems creates multiple items in a single operation.
func (s *itemSvcImpl) BulkCreateItems(ctx context.Context, params domain.BulkCreateItemsParams) ([]domain.BulkCreateItemResult, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.bulk_create_items")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	// The duplicate-SKU path in bulkCreateSingleItem updates an existing item in place. That is an update-class mutation, so it must additionally require items:update; otherwise a create-only actor could modify existing catalog records by submitting bulk-create rows with existing SKUs.
	canUpdateItems := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionUpdate) == nil

	// Validate item type.
	switch params.Type {
	case "product", "material", "part":
	default:
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid item type. Must be 'product', 'material', or 'part'.", "type"))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[[]domain.BulkCreateItemResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return *cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		results := make([]domain.BulkCreateItemResult, len(params.Items))

		for i, input := range params.Items {
			results[i] = s.bulkCreateSingleItem(ctx, accountID, params.Type, input, canUpdateItems)
		}

		cacheErr := s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, results)
		})
		if cacheErr != nil {
			// Caching failure is not fatal; return results anyway.
			_ = cacheErr
		}

		return results, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// connects the given attributes to an item (additive), rejecting any the account doesn't own or whose property the item's category doesn't carry
func attachItemAttributesInTx(txCtx context.Context, repos domain.RepoFactory, accountID, categoryID, itemID string, attributeIDs []string) *apierror.APIError {
	ids := make([]string, 0, len(attributeIDs))
	for _, attrID := range attributeIDs {
		if attrID != "" {
			ids = append(ids, attrID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	if apiErr := validateAttributesForCategory(txCtx, repos, accountID, categoryID, ids, "attribute_ids"); apiErr != nil {
		return apiErr
	}

	itemRepo := repos.NewItemRepo()
	for _, attrID := range ids {
		if apiErr := itemRepo.AddAttribute(txCtx, domain.AddItemAttributeParams{
			AccountID:   accountID,
			ItemID:      itemID,
			AttributeID: attrID,
		}); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// validateAttributesForCategory rejects attributes the account doesn't own, and ones whose property is not linked to the item category they would be attached under. The category's properties are what the catalog offers as the item's describable axes, and an item's attributes are read back straight off the join table, so an attribute from an unrelated property would render on the item as a value of a property the item is not supposed to have.
func validateAttributesForCategory(ctx context.Context, repos domain.RepoFactory, accountID, categoryID string, attributeIDs []string, param string) *apierror.APIError {
	if len(attributeIDs) == 0 {
		return nil
	}

	attributes, apiErr := repos.NewAttributeRepo().GetByIDs(ctx, accountID, attributeIDs)
	if apiErr != nil {
		return apiErr
	}
	byID := make(map[string]*domain.Attribute, len(attributes))
	for _, attr := range attributes {
		byID[attr.ID] = attr
	}

	linked, apiErr := categoryPropertyIDs(ctx, repos, categoryID)
	if apiErr != nil {
		return apiErr
	}

	for _, attrID := range attributeIDs {
		attr, ok := byID[attrID]
		if !ok {
			return apierror.NewResourceNotFoundError("Attribute not found.")
		}
		if _, ok := linked[attr.PropertyID]; !ok {
			return apierror.NewValidationErrorWithParam(fmt.Sprintf("Attribute %q is not available on this item's category; link its property to the category first.", attr.Value), param)
		}
	}

	return nil
}

// categoryPropertyIDs returns the set of property ids the category carries.
func categoryPropertyIDs(ctx context.Context, repos domain.RepoFactory, categoryID string) (map[string]struct{}, *apierror.APIError) {
	properties, apiErr := repos.NewItemCategoryRepo().GetProperties(ctx, categoryID)
	if apiErr != nil {
		return nil, apiErr
	}
	linked := make(map[string]struct{}, len(properties))
	for _, property := range properties {
		linked[property.ID] = struct{}{}
	}
	return linked, nil
}

// validateCategoryCarriesItemAttributes rejects a move to a category that doesn't carry the properties of the attributes already on the item. The move leaves those links in place, so permitting it would open the back door to exactly the state validateAttributesForCategory refuses to create at link time.
func validateCategoryCarriesItemAttributes(ctx context.Context, repos domain.RepoFactory, item *domain.Item, categoryID, param string) *apierror.APIError {
	// Re-assigning the item's own category is a documented no-op, and it strands nothing even when the item predates this rule.
	if len(item.Attributes) == 0 || item.ItemCategoryID == categoryID {
		return nil
	}

	linked, apiErr := categoryPropertyIDs(ctx, repos, categoryID)
	if apiErr != nil {
		return apiErr
	}

	for _, attr := range item.Attributes {
		if _, ok := linked[attr.PropertyID]; !ok {
			return apierror.NewValidationErrorWithParam(fmt.Sprintf("The item carries the attribute %q, whose property the target category does not have; unlink the attribute or add its property to the category first.", attr.Value), param)
		}
	}

	return nil
}

// applyItemRatesInTx writes the unit_value / unit_cost rate values and units when
// supplied, enforcing the currency-numerator / non-currency-denominator rule used by
// the single create/update endpoints. Used by bulk update paths (create sets rates
// directly when inserting them).
func applyItemRatesInTx(txCtx context.Context, repos domain.RepoFactory, unitValueRateID, unitCostRateID string, unitPrice, unitCost *domain.CreateRateParams) *apierror.APIError {
	unitRepo := repos.NewUnitRepo()
	rateRepo := repos.NewRateRepo()

	if unitPrice != nil {
		if apiErr := ValidateCostRateUnits(txCtx, unitRepo, unitPrice.NumeratorUnitID, unitPrice.DenominatorUnitID, "unit_price"); apiErr != nil {
			return apiErr
		}
		if _, apiErr := rateRepo.Update(txCtx, domain.UpdateRateParams{
			RateID:            unitValueRateID,
			Value:             &unitPrice.Value,
			NumeratorUnitID:   &unitPrice.NumeratorUnitID,
			DenominatorUnitID: &unitPrice.DenominatorUnitID,
		}); apiErr != nil {
			return apiErr
		}
	}

	if unitCost != nil {
		if apiErr := ValidateCostRateUnits(txCtx, unitRepo, unitCost.NumeratorUnitID, unitCost.DenominatorUnitID, "unit_cost"); apiErr != nil {
			return apiErr
		}
		if _, apiErr := rateRepo.Update(txCtx, domain.UpdateRateParams{
			RateID:            unitCostRateID,
			Value:             &unitCost.Value,
			NumeratorUnitID:   &unitCost.NumeratorUnitID,
			DenominatorUnitID: &unitCost.DenominatorUnitID,
		}); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// updates an existing item in place during a bulk-create, matching Dashboard's updateExistingProduct
// behavior: writes description, product_line_id, category_id and the unit_value rate instead of erroring on a duplicate SKU.
func (s *itemSvcImpl) bulkUpsertExistingItem(ctx context.Context, accountID, itemID, unitValueRateID string, input domain.BulkCreateItemInput) *apierror.APIError {
	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		return txSvc.bulkUpsertExistingItemInTx(txCtx, accountID, itemID, unitValueRateID, input)
	})
}

// bulkUpsertExistingItemInTx updates an existing item in place within an existing
// transaction (description, category, product line, attributes).
func (s *itemSvcImpl) bulkUpsertExistingItemInTx(txCtx context.Context, accountID, itemID, unitValueRateID string, input domain.BulkCreateItemInput) *apierror.APIError {
	{
		txSvc := s
		txItemRepo := txSvc.repos.NewItemRepo()

		// Description / notes / sku are handled via the generic item update.
		updateDesc := input.Description != nil
		if apiErr := txItemRepo.Update(txCtx, domain.UpdateItemParams{
			AccountID:         accountID,
			ItemID:            itemID,
			Description:       input.Description,
			UpdateDescription: updateDesc,
		}); apiErr != nil {
			return apiErr
		}

		// Move the item between categories when the input supplies a new category, keeping rate units consistent with the new category's base unit.
		if input.ItemCategoryID != "" {
			item, apiErr := txItemRepo.Get(txCtx, domain.GetItemParams{
				AccountID: accountID,
				ItemID:    itemID,
				Includes:  []string{"attributes"},
			})
			if apiErr != nil {
				return apiErr
			}
			category, apiErr := txSvc.repos.NewItemCategoryRepo().Get(txCtx, domain.GetItemCategoryParams{
				AccountID:      accountID,
				ItemCategoryID: input.ItemCategoryID,
			})
			if apiErr != nil {
				return apiErr
			}
			if apiErr := validateChangeItemCategoryTypes(item, category); apiErr != nil {
				return apiErr
			}
			if apiErr := validateCategoryCarriesItemAttributes(txCtx, txSvc.repos, item, input.ItemCategoryID, "item_category_id"); apiErr != nil {
				return apiErr
			}
			if apiErr := txItemRepo.ChangeCategory(txCtx, domain.ChangeItemCategoryParams{
				AccountID:  accountID,
				ItemID:     itemID,
				CategoryID: input.ItemCategoryID,
			}); apiErr != nil {
				return apiErr
			}
			baseUnitID, _, apiErr := txItemRepo.GetCategoryBaseUnitID(txCtx, input.ItemCategoryID)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := txItemRepo.UpdateRateUnits(txCtx, accountID, itemID, baseUnitID); apiErr != nil {
				return apiErr
			}
		}

		// Move the product to the new product line when supplied. Only applies to product-type bulk uploads; materials/parts don't have a product line.
		if input.ProductLineID != nil && *input.ProductLineID != "" {
			if _, apiErr := txSvc.repos.NewProductRepo().ChangeProductLine(txCtx, domain.ChangeProductProductLineParams{
				AccountID:     accountID,
				ProductID:     itemID,
				ProductLineID: *input.ProductLineID,
			}); apiErr != nil {
				// Non-product items don't have a product-line row; ignore not-found here.
				if !apierror.IsNotFound(apiErr) {
					return apiErr
				}
			}
		}

		// Note: bulk input currently doesn't carry a unit price, so there's nothing to write into the unit_value rate. unitValueRateID is accepted on the signature so callers can add a price field later without changing the repo surface.
		_ = unitValueRateID

		// Replace attributes when supplied: clears existing, then re-adds from input.
		// Matches Dashboard's updateExistingProduct attribute handling.
		if len(input.AttributeIDs) > 0 {
			categoryID := input.ItemCategoryID
			if categoryID == "" {
				item, apiErr := txItemRepo.Get(txCtx, domain.GetItemParams{
					AccountID: accountID,
					ItemID:    itemID,
				})
				if apiErr != nil {
					return apiErr
				}
				categoryID = item.ItemCategoryID
			}
			if apiErr := attachItemAttributesInTx(txCtx, txSvc.repos, accountID, categoryID, itemID, input.AttributeIDs); apiErr != nil {
				return apiErr
			}
		}

		return nil
	}
}

// bulkCreateSingleItem creates a single item within a bulk operation.
// Errors are captured in the result rather than propagated.
func (s *itemSvcImpl) bulkCreateSingleItem(ctx context.Context, accountID, itemType string, input domain.BulkCreateItemInput, canUpdateItems bool) domain.BulkCreateItemResult {
	errResult := func(sku string, msg string) domain.BulkCreateItemResult {
		return domain.BulkCreateItemResult{SKU: sku, Success: false, Error: &msg}
	}

	// When the SKU already exists in the account, upsert the existing item's description + unit_value rather than failing (matches Dashboard's bulk behavior).
	existingItemID, existingRateID, apiErr := s.repos.NewItemRepo().FindBySKU(ctx, accountID, input.SKU)
	if apiErr != nil {
		return errResult(input.SKU, "Failed to look up existing SKU.")
	}
	if existingItemID != nil && existingRateID != nil {
		// The upsert path mutates an existing item; gate it on items:update so a create-only actor cannot modify existing catalog records by passing an existing SKU.
		if !canUpdateItems {
			return errResult(input.SKU, "You do not have permission to update an existing item with this SKU.")
		}
		if apiErr := s.bulkUpsertExistingItem(ctx, accountID, *existingItemID, *existingRateID, input); apiErr != nil {
			return errResult(input.SKU, apiErr.PublicMessage)
		}
		return domain.BulkCreateItemResult{SKU: input.SKU, Success: true, ItemID: existingItemID}
	}

	// Get base unit for rates from category, and enforce the item-type/category-type
	// rule (materials → material categories; parts and products → product categories).
	baseUnitID, categoryTypeCode, apiErr := s.repos.NewItemRepo().GetCategoryBaseUnitID(ctx, input.ItemCategoryID)
	if apiErr != nil {
		return errResult(input.SKU, "Category not found.")
	}
	if !categoryTypeMatchesItem(itemType, categoryTypeCode) {
		return errResult(input.SKU, "This category type cannot be assigned to this item type.")
	}

	// Generate IDs.
	itemID, apiErr := id.GenID(id.ItemIDPrefix, nil)
	if apiErr != nil {
		return errResult(input.SKU, "Failed to generate item ID.")
	}

	unitValueRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return errResult(input.SKU, "Failed to generate rate ID.")
	}
	unitCostRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return errResult(input.SKU, "Failed to generate rate ID.")
	}
	burnRateRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return errResult(input.SKU, "Failed to generate rate ID.")
	}

	var createErr *apierror.APIError

	switch itemType {
	case "product":
		createErr = s.bulkCreateProduct(ctx, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID, input)
	case "material":
		createErr = s.bulkCreateMaterial(ctx, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID, input)
	case "part":
		createErr = s.bulkCreatePart(ctx, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID, input)
	}

	if createErr != nil {
		msg := createErr.PublicMessage
		return errResult(input.SKU, msg)
	}

	return domain.BulkCreateItemResult{SKU: input.SKU, Success: true, ItemID: &itemID}
}

func (s *itemSvcImpl) bulkCreateProduct(ctx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, input domain.BulkCreateItemInput) *apierror.APIError {
	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		return txSvc.bulkCreateProductInTx(txCtx, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID, decimal.Zero, input)
	})
}

// bulkCreateProductInTx inserts a product-type item and its rates within an
// existing transaction. The wrapper bulkCreateProduct opens its own tx; bulk
// upsert calls this directly inside its batch tx. openingQty seeds the initial
// on-hand inventory (zero for the bulk-create path).
func (s *itemSvcImpl) bulkCreateProductInTx(txCtx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, openingQty decimal.Decimal, input domain.BulkCreateItemInput) *apierror.APIError {
	productID, apiErr := id.GenID(id.ProductIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	{
		txSvc := s
		txProductRepo := txSvc.repos.NewProductRepo()

		// Insert rates.
		if apiErr := txProductRepo.InsertRate(txCtx, unitValueRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txProductRepo.InsertRate(txCtx, unitCostRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txProductRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, "day"); apiErr != nil {
			return apiErr
		}

		// Insert item. Default IsPortalReady=true to match Dashboard's bulk-create behavior
		// (customer-facing by default; clients can toggle off after import if needed).
		params := domain.CreateProductParams{
			AccountID:       accountID,
			SKU:             input.SKU,
			Description:     input.Description,
			ProductTypeCode: "sale",
			ProductLineID:   input.ProductLineID,
			CategoryID:      input.ItemCategoryID,
			IsPortalReady:   true,
		}
		if apiErr := txProductRepo.InsertItem(txCtx, domain.InsertProductItemParams{
			ItemID:          itemID,
			AccountID:       accountID,
			SKU:             input.SKU,
			Description:     input.Description,
			CategoryID:      input.ItemCategoryID,
			UnitValueRateID: unitValueRateID,
			UnitCostRateID:  unitCostRateID,
			BurnRateRateID:  burnRateRateID,
		}); apiErr != nil {
			return apiErr
		}

		// Insert product record.
		if _, apiErr := txProductRepo.Create(txCtx, productID, itemID, params); apiErr != nil {
			return apiErr
		}

		// Initialize inventory tracking to match the regular CreateProduct path.
		// openingQty is zero for bulk-create; bulk-upsert may seed a starting on-hand.
		txInvMutRepo := txSvc.repos.NewInventoryMutationRepo()
		if apiErr := txInvMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
			AccountID: accountID,
			ItemID:    itemID,
			Measure:   openingQty,
			UnitID:    baseUnitID,
		}); apiErr != nil {
			return apiErr
		}
		if apiErr := txInvMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
			AccountID:  accountID,
			ItemID:     itemID,
			Measure:    openingQty,
			UnitID:     baseUnitID,
			ActionType: "user_action",
		}); apiErr != nil {
			return apiErr
		}

		createdItem, apiErr := txSvc.repos.NewItemRepo().Get(txCtx, domain.GetItemParams{
			AccountID: accountID,
			ItemID:    itemID,
		})
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, createdItem)

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeItem,
			ResourceID:   itemID,
			Changes:      changes,
		})
	}
}

func (s *itemSvcImpl) bulkCreateMaterial(ctx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, input domain.BulkCreateItemInput) *apierror.APIError {
	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		return txSvc.bulkCreateMaterialInTx(txCtx, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID, decimal.Zero, input)
	})
}

// bulkCreateMaterialInTx inserts a material-type item within an existing transaction.
// A non-zero openingQty seeds initial on-hand inventory (bulk upsert only).
func (s *itemSvcImpl) bulkCreateMaterialInTx(txCtx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, openingQty decimal.Decimal, input domain.BulkCreateItemInput) *apierror.APIError {
	materialID, apiErr := id.GenID(id.MaterialIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	orderPointQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	leadTimeQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	{
		txSvc := s
		txMaterialRepo := txSvc.repos.NewMaterialRepo()

		// Insert rates.
		if apiErr := txMaterialRepo.InsertRate(txCtx, unitValueRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txMaterialRepo.InsertRate(txCtx, unitCostRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txMaterialRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, "day"); apiErr != nil {
			return apiErr
		}

		// Insert item.
		if apiErr := txMaterialRepo.InsertItem(txCtx, domain.InsertMaterialItemParams{
			ItemID:          itemID,
			AccountID:       accountID,
			SKU:             input.SKU,
			Description:     input.Description,
			CategoryID:      input.ItemCategoryID,
			UnitValueRateID: unitValueRateID,
			UnitCostRateID:  unitCostRateID,
			BurnRateRateID:  burnRateRateID,
		}); apiErr != nil {
			return apiErr
		}

		// Insert order point and lead time quantities with defaults.
		if apiErr := txMaterialRepo.InsertQuantity(txCtx, orderPointQtyID, "0", baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txMaterialRepo.InsertQuantity(txCtx, leadTimeQtyID, "0", baseUnitID); apiErr != nil {
			return apiErr
		}

		// Insert material record.
		if apiErr := txMaterialRepo.Create(txCtx, materialID, itemID, orderPointQtyID, leadTimeQtyID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.seedOpeningInventoryInTx(txCtx, accountID, itemID, baseUnitID, openingQty); apiErr != nil {
			return apiErr
		}

		createdItem, apiErr := txSvc.repos.NewItemRepo().Get(txCtx, domain.GetItemParams{
			AccountID: accountID,
			ItemID:    itemID,
		})
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, createdItem)

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeItem,
			ResourceID:   itemID,
			Changes:      changes,
		})
	}
}

// seedOpeningInventoryInTx records an opening on-hand inventory log + change log for a
// newly created item. No-op when the quantity is zero, preserving the bulk-create
// path's existing behavior (products already seed a zero log in their own helper).
func (s *itemSvcImpl) seedOpeningInventoryInTx(txCtx context.Context, accountID, itemID, baseUnitID string, openingQty decimal.Decimal) *apierror.APIError {
	if openingQty.IsZero() {
		return nil
	}
	txInvMutRepo := s.repos.NewInventoryMutationRepo()
	if apiErr := txInvMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
		AccountID: accountID,
		ItemID:    itemID,
		Measure:   openingQty,
		UnitID:    baseUnitID,
	}); apiErr != nil {
		return apiErr
	}
	return txInvMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
		AccountID:  accountID,
		ItemID:     itemID,
		Measure:    openingQty,
		UnitID:     baseUnitID,
		ActionType: "user_action",
	})
}

func (s *itemSvcImpl) bulkCreatePart(ctx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, input domain.BulkCreateItemInput) *apierror.APIError {
	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		return txSvc.bulkCreatePartInTx(txCtx, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID, decimal.Zero, input)
	})
}

// bulkCreatePartInTx inserts a part-type item within an existing transaction.
// A non-zero openingQty seeds initial on-hand inventory (bulk upsert only).
func (s *itemSvcImpl) bulkCreatePartInTx(txCtx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, openingQty decimal.Decimal, input domain.BulkCreateItemInput) *apierror.APIError {
	partID, apiErr := id.GenID(id.PartIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	{
		txSvc := s
		txPartRepo := txSvc.repos.NewPartRepo()

		// Insert rates.
		if apiErr := txPartRepo.InsertRate(txCtx, unitValueRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txPartRepo.InsertRate(txCtx, unitCostRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txPartRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, "day"); apiErr != nil {
			return apiErr
		}

		// Insert item.
		params := domain.CreatePartParams{
			AccountID:   accountID,
			SKU:         input.SKU,
			Description: input.Description,
			CategoryID:  input.ItemCategoryID,
		}
		if apiErr := txPartRepo.InsertItem(txCtx, itemID, params, unitValueRateID, burnRateRateID, unitCostRateID); apiErr != nil {
			return apiErr
		}

		// Insert part record.
		if _, apiErr := txPartRepo.Create(txCtx, partID, itemID, params); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.seedOpeningInventoryInTx(txCtx, accountID, itemID, baseUnitID, openingQty); apiErr != nil {
			return apiErr
		}

		createdItem, apiErr := txSvc.repos.NewItemRepo().Get(txCtx, domain.GetItemParams{
			AccountID: accountID,
			ItemID:    itemID,
		})
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, createdItem)

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeItem,
			ResourceID:   itemID,
			Changes:      changes,
		})
	}
}

// BulkReconcileItems reconciles inventory for multiple items by SKU.
func (s *itemSvcImpl) BulkReconcileItems(ctx context.Context, params domain.BulkReconcileItemsParams) (*domain.BulkReconcileItemsResult, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.bulk_reconcile_items")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainItems, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	params.AccountID = accountID
	if identity.Actor != nil {
		params.ResponsibleUserID = &identity.Actor.ID
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.BulkReconcileItemsResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result := &domain.BulkReconcileItemsResult{}

		// Extract unique SKUs and units
		uniqueSKUs := make(map[string]bool)
		uniqueUnits := make(map[string]bool)
		for _, d := range params.Data {
			uniqueSKUs[d.SKU] = true
			uniqueUnits[d.Unit] = true
		}
		skuList := make([]string, 0, len(uniqueSKUs))
		for sku := range uniqueSKUs {
			skuList = append(skuList, sku)
		}
		unitList := make([]string, 0, len(uniqueUnits))
		for u := range uniqueUnits {
			unitList = append(unitList, u)
		}

		// Batch-fetch items and units
		items, apiErr := s.repos.NewItemRepo().FetchItemsBySKU(ctx, accountID, skuList)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}
		units, apiErr := s.repos.NewUnitRepo().FindByAbbreviations(ctx, accountID, unitList)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}

		// Build lookup maps
		itemMap := make(map[string]domain.ItemSKUInfo, len(items))
		for _, item := range items {
			itemMap[item.SKU] = item
		}
		unitMap := make(map[string]*domain.Unit, len(units))
		for _, unit := range units {
			unitMap[unit.Abbreviation] = unit
		}

		// Categorize data
		var validItems []domain.BulkReconcileItemInput
		for _, d := range params.Data {
			if _, ok := itemMap[d.SKU]; !ok {
				result.SkippedItems = append(result.SkippedItems, domain.SkippedItem{SKU: d.SKU, Reason: "Item not found"})
				continue
			}
			if _, ok := unitMap[d.Unit]; !ok {
				result.Errors = append(result.Errors, domain.ReconcileError{ItemID: itemMap[d.SKU].ItemID, SKU: d.SKU, Error: fmt.Sprintf("Unit '%s' not found", d.Unit)})
				continue
			}
			validItems = append(validItems, d)
		}

		if len(validItems) == 0 {
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}
			return result, nil
		}

		// Fetch physical inventory for all valid items before batch processing.
		// This matches the Dashboard pattern which bulk-fetches inventory before processing.
		// Uses physical inventory (receipts - open issues) instead of ATP to match Dashboard's physicalInventory metric.
		invQueryRepo := s.repos.NewInventoryQueryRepo()
		physicalInvMap := make(map[string]decimal.Decimal)
		for _, d := range validItems {
			item := itemMap[d.SKU]
			if _, already := physicalInvMap[item.ItemID]; already {
				continue
			}
			// In the base unit, which is what the rows below are written in.
			physInv, fetchErr := invQueryRepo.FetchPhysicalInventory(ctx, item.ItemID, accountID, item.BaseUnitID)
			if fetchErr != nil {
				// Skip items where inventory cannot be fetched, matching Dashboard behavior where items with no currentInventory are silently skipped.
				continue
			}
			physicalInvMap[item.ItemID] = physInv
		}

		// Process in batches of 50
		batchSize := 50
		for batchStart := 0; batchStart < len(validItems); batchStart += batchSize {
			batchEnd := min(batchStart+batchSize, len(validItems))
			batch := validItems[batchStart:batchEnd]

			// Built inside the callback and assigned out once, then merged into the caller's result here.
			// transaction.go's contract is that the callback re-runs on a lock conflict, so appending
			// straight to `result` from inside it meant a retried batch reported every one of its rows
			// twice. tools/txaudit enforces this shape.
			var batchErrors []domain.ReconcileError
			var batchReconciled []domain.ReconciledItem

			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
				var errs []domain.ReconcileError
				var reconciled []domain.ReconciledItem
				invMutRepo := txSvc.repos.NewInventoryMutationRepo()
				// Every item in the batch, so the roots are the transaction's first statements. The set
				// comes from `batch`, which was sliced before the transaction opened (Corollary A).
				batchItemIDs := make([]string, 0, len(batch))
				for _, d := range batch {
					batchItemIDs = append(batchItemIDs, itemMap[d.SKU].ItemID)
				}
				scope, apiErr := ledgerlock.Acquire(txCtx, invMutRepo, batchItemIDs)
				if apiErr != nil {
					return apiErr
				}

				for _, d := range batch {
					item := itemMap[d.SKU]

					currentQty, ok := physicalInvMap[item.ItemID]
					if !ok {
						// Item has no inventory data; skip silently (matches Dashboard behavior where items without currentInventory are skipped without error).
						continue
					}

					var newQty, delta decimal.Decimal
					if params.ReconcileType == "force" {
						newQty = d.Measure
						delta = d.Measure.Sub(currentQty)
					} else { // addition
						delta = d.Measure
						newQty = currentQty.Add(delta)
					}

					measure := delta.Abs()
					unitID := item.BaseUnitID

					if delta.GreaterThan(decimal.Zero) {
						if apiErr := invMutRepo.CreateInventoryReceipt(txCtx, scope, domain.CreateInventoryReceiptParams{
							AccountID: accountID, ItemID: item.ItemID, Measure: measure, UnitID: unitID,
						}); apiErr != nil {
							errs = append(errs, domain.ReconcileError{SKU: d.SKU, Error: "Failed to create receipt"})
							continue
						}
					} else if delta.LessThan(decimal.Zero) {
						if apiErr := invMutRepo.CreateInventoryIssue(txCtx, scope, domain.CreateInventoryIssueParams{
							AccountID: accountID, ItemID: item.ItemID, Measure: measure, UnitID: unitID,
						}); apiErr != nil {
							errs = append(errs, domain.ReconcileError{SKU: d.SKU, Error: "Failed to create issue"})
							continue
						}
					}

					if apiErr := mediator.RecordInventoryAuditTrail(
						txCtx,
						txSvc.repos,
						accountID,
						item.ItemID,
						delta,
						unitID,
						"user_correction",
						nil,
						params.ResponsibleUserID,
					); apiErr != nil {
						errs = append(errs, domain.ReconcileError{ItemID: item.ItemID, SKU: d.SKU, Error: "Failed to record inventory audit trail"})
						continue
					}

					reconciled = append(reconciled, domain.ReconciledItem{
						ItemID: item.ItemID, SKU: d.SKU,
						PreviousMeasure: currentQty, NewMeasure: newQty,
						UnitID: item.BaseUnitID,
					})

					// Empty when the reconciled quantity equals the current quantity; the publisher skips the event as a no-op.
					var changes []audit.FieldChange
					if !delta.IsZero() {
						changes = append(changes, audit.NewFieldChange("quantity", currentQty, newQty))
					}

					if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
						ServiceName:  domain.ServiceName,
						Action:       constants.AuditActionUpdate,
						ResourceType: constants.ObjectTypeItem,
						ResourceID:   item.ItemID,
						Changes:      changes,
					}); apiErr != nil {
						errs = append(errs, domain.ReconcileError{SKU: d.SKU, Error: "Failed to publish audit event"})
						continue
					}
				}

				// Reconciling writes receipts and issues and then never offered that stock to anything
				// waiting on it: an item adjusted upward stayed short against its own open demand until
				// something unrelated happened to trigger allocation. Inside this transaction, so the
				// request exists if and only if the rows that justify it do.
				requestIDs := make([]string, 0, len(reconciled))
				for _, item := range reconciled {
					requestIDs = append(requestIDs, item.ItemID)
				}
				if apiErr := mediator.EnqueueAllocateOpenIssues(txCtx, txSvc.repos, accountID, requestIDs...); apiErr != nil {
					return apiErr
				}

				// Assigned, not appended: the callback re-runs on a lock conflict and the second run must
				// replace the first run's rows rather than add to them.
				batchErrors = errs
				batchReconciled = reconciled
				return nil
			})

			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}

			result.Errors = append(result.Errors, batchErrors...)
			result.ReconciledItems = append(result.ReconciledItems, batchReconciled...)
		}

		// Cached once, after every batch has committed, rather than once per batch from inside the
		// callback. It was being handed the accumulating slice mid-run, so a retried batch cached a
		// response body containing its rows twice — and the cached body is what a replay of the
		// idempotency key returns.
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		s.kickOutbox()

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// --- Export ---

// lists the fixed columns every item sheet carries; extra lands between the
// category and the money columns, where a type's own fields read best
func itemBaseColumns(extra ...excel.ColumnSpec) []excel.ColumnSpec {
	columns := []excel.ColumnSpec{
		{Header: "ID", Key: "id", Width: 24},
		{Header: "SKU", Key: "sku", Width: 32},
		{Header: "Description", Key: "description", Width: 32},
		{Header: "Notes", Key: "notes", Width: 32},
		{Header: "Category", Key: "category", Width: 18},
	}
	columns = append(columns, extra...)
	// No unit columns beside these: the importer reads the account's currency and
	// base unit itself, and an unrecognised header becomes a property.
	return append(columns,
		excel.ColumnSpec{Header: "Unit Price", Key: "unit_price", Width: 14},
		excel.ColumnSpec{Header: "Unit Cost", Key: "unit_cost", Width: 14},
	)
}

// fills the fixed item cells shared by the product, part and material sheets
func addItemBaseCells(row excel.Row, rowID string, item *domain.Item) {
	row["id"] = rowID
	if item == nil {
		return
	}
	row["sku"] = item.SKU
	row["description"] = excel.Str(item.Description)
	row["notes"] = excel.Str(item.Notes)
	row["category"] = item.CategoryName
	row["unit_price"] = rateValue(item.UnitValue)
	row["unit_cost"] = rateValue(item.UnitCost)
	addItemPropertyCells(row, item)
}

// reads a rate's amount, blank where the rate was never set
func rateValue(rate *domain.Rate) string {
	if rate == nil {
		return ""
	}
	return decimalCell(rate.Value)
}

// reads a quantity's bare amount; the unit stays out of the cell because the
// importer parses the number alone and supplies the unit itself
func quantityValue(quantity *domain.Quantity) string {
	if quantity == nil {
		return ""
	}
	return decimalCell(quantity.Value)
}
