package service

import (
	"context"
	"fmt"
	"math"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var itemSvcTracer = tracing.GetTracer("core-service.item_service")

type itemSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ItemSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
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
	}
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

	// Load attributes for each item
	repo := s.repos.NewItemRepo()
	for _, item := range result.Items {
		if apiErr := loadItemAttributes(ctx, repo, item); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return result, nil
}

// GetItem returns a single item by ID within the caller's account.
func (s *itemSvcImpl) GetItem(ctx context.Context, itemID string) (*domain.Item, *apierror.APIError) {
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

	accountID := identity.Target.AccountID
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

	totalCost := totalMaterial.Add(totalLabor).Add(totalOverhead)

	// 7. Update item's unit cost (side effect matching Dashboard behavior).
	unitID := targetStep.Production.Quantity.Unit.ID
	if updateErr := itemRepo.UpdateUnitCost(ctx, accountID, itemID, totalCost, unitID); updateErr != nil {
		return nil, tracing.Trace(span, updateErr)
	}

	return &domain.ItemCosts{
		DirectMaterialCost: totalMaterial.StringFixed(30),
		DirectLaborCost:    totalLabor.StringFixed(30),
		OverheadCost:       totalOverhead.StringFixed(30),
		TotalCost:          totalCost.StringFixed(30),
		UnitID:             unitID,
	}, nil
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

	// Labor time (seconds per piece).
	var laborTimeMeasure decimal.Decimal
	if step.LaborTime != nil {
		laborTimeMeasure, _ = decimal.NewFromString(step.LaborTime.Value)
	}

	// Labor rate ($ per second).
	var laborRateValue decimal.Decimal
	if step.LaborRate != nil {
		laborRateValue, _ = decimal.NewFromString(step.LaborRate.Value)
	}

	// Overhead rate ($ per second).
	var overheadRateValue decimal.Decimal
	if step.OverheadRate != nil {
		overheadRateValue, _ = decimal.NewFromString(step.OverheadRate.Value)
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
	for _, cons := range consumptions {
		if cons.ConsumedItemType == "part" || cons.ConsumedItemType == "product" {
			continue
		}
		totalUsed := cons.ConsumptionQuantity.Add(cons.WasteQuantity)
		result.material = result.material.Add(totalUsed.Mul(cons.UnitCost))
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

	return s.repos.NewItemRepo().GetTrends(ctx, identity.Target.AccountID, itemID, trendType)
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

// loadItemAttributes is a helper to load attributes for an item from a repo instance.
func loadItemAttributes(_ context.Context, _ domain.ItemRepo, _ *domain.Item) *apierror.APIError {
	// The Get method already loads attributes; this is for list results.
	// Attributes are loaded in the repository's loadItemAttributes method when using Get.
	// For list results, we skip attributes loading to optimize performance.
	// If needed, individual attribute loading can be done via Get.
	return nil
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
func (s *itemSvcImpl) AddItemAttribute(ctx context.Context, itemID, attributeID string) (*domain.Item, *apierror.APIError) {
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
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

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
func (s *itemSvcImpl) RemoveItemAttribute(ctx context.Context, itemID, attributeID string) (*domain.Item, *apierror.APIError) {
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
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

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

// ChangeItemCategory changes the category of an item and updates rate units.
func (s *itemSvcImpl) ChangeItemCategory(ctx context.Context, itemID, categoryID string) (*domain.Item, *apierror.APIError) {
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

			// Get the base unit of the new category
			baseUnitID, apiErr := txRepo.GetCategoryBaseUnitID(txCtx, categoryID)
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
			})
			if apiErr != nil {
				return apiErr
			}
			result = item

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
			invReservationRepo := txSvc.repos.NewInventoryReservationRepo()

			// Fetch current physical inventory (always uses the main account, matching Dashboard behavior).
			currentPhysical, apiErr := invQueryRepo.FetchPhysicalInventory(txCtx, params.ItemID, accountID)
			if apiErr != nil {
				return apiErr
			}

			var qc float64
			if params.QuantityChange != nil {
				qc = *params.QuantityChange
			}
			quantityChange := decimal.NewFromFloat(qc)

			reconcile := params.Reconcile != nil && *params.Reconcile

			var unitID string
			if params.UnitID != nil {
				unitID = *params.UnitID
			}

			// Calculate delta and final quantity based on reconcile mode.
			var delta, finalQty decimal.Decimal
			currentQty := decimal.NewFromFloat(currentPhysical)
			if reconcile {
				// Reconcile: set inventory to the exact value.
				finalQty = quantityChange
				delta = quantityChange.Sub(currentQty)
			} else {
				// Adjust: add the quantity change to current inventory.
				delta = quantityChange
				finalQty = currentQty.Add(quantityChange)
			}

			// Skip if no change and not reconciling.
			if delta.IsZero() && !reconcile {
				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, struct{}{})
			}

			if delta.GreaterThan(decimal.Zero) {
				// Positive delta: create inventory receipt.
				if apiErr := invMutRepo.CreateInventoryReceipt(txCtx, domain.CreateInventoryReceiptParams{
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

				// Allocate open issues against new receipt.
				if apiErr := invReservationRepo.AllocateOpenIssuesForItem(txCtx, accountID, params.ItemID); apiErr != nil {
					return apiErr
				}
			} else if delta.LessThan(decimal.Zero) {
				// Negative delta: create inventory issue.
				if apiErr := invMutRepo.CreateInventoryIssue(txCtx, domain.CreateInventoryIssueParams{
					AccountID:  accountID,
					ItemID:     params.ItemID,
					Measure:    delta,
					UnitID:     unitID,
					LocationID: params.LocationID,
					LotID:      lotID,
				}); apiErr != nil {
					return apiErr
				}

				// Allocate open issues.
				if apiErr := invReservationRepo.AllocateOpenIssuesForItem(txCtx, accountID, params.ItemID); apiErr != nil {
					return apiErr
				}
			}

			// Create inventory log (point-in-time snapshot).
			if apiErr := invMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
				AccountID: accountID,
				ItemID:    params.ItemID,
				Measure:   finalQty,
				UnitID:    unitID,
			}); apiErr != nil {
				return apiErr
			}

			// Create inventory change log (audit trail).
			if apiErr := invMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
				AccountID:         accountID,
				ItemID:            params.ItemID,
				Measure:           delta,
				UnitID:            unitID,
				ActionType:        "user_correction",
				ResponsibleUserID: responsibleUserID,
			}); apiErr != nil {
				return apiErr
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItem,
				ResourceID:   params.ItemID,
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
			results[i] = s.bulkCreateSingleItem(ctx, accountID, params.Type, input)
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

// bulkCreateSingleItem creates a single item within a bulk operation.
// Errors are captured in the result rather than propagated.
func (s *itemSvcImpl) bulkCreateSingleItem(ctx context.Context, accountID, itemType string, input domain.BulkCreateItemInput) domain.BulkCreateItemResult {
	errResult := func(sku string, msg string) domain.BulkCreateItemResult {
		return domain.BulkCreateItemResult{SKU: sku, Success: false, Error: &msg}
	}

	// Check SKU uniqueness.
	exists, apiErr := s.repos.NewItemRepo().CheckSKUExists(ctx, accountID, input.SKU, "")
	if apiErr != nil {
		return errResult(input.SKU, "Failed to check SKU uniqueness.")
	}
	if exists {
		return errResult(input.SKU, "An item with this SKU already exists.")
	}

	// Get base unit for rates from category.
	baseUnitID, apiErr := s.repos.NewItemRepo().GetCategoryBaseUnitID(ctx, input.ItemCategoryID)
	if apiErr != nil {
		return errResult(input.SKU, "Category not found.")
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
	productID, apiErr := id.GenID(id.ProductIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		txProductRepo := txSvc.repos.NewProductRepo()

		// Insert rates.
		if apiErr := txProductRepo.InsertRate(txCtx, unitValueRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txProductRepo.InsertRate(txCtx, unitCostRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txProductRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}

		// Insert item.
		params := domain.CreateProductParams{
			AccountID:       accountID,
			ItemID:          itemID,
			UnitValueRateID: unitValueRateID,
			UnitCostRateID:  unitCostRateID,
			BurnRateRateID:  burnRateRateID,
			SKU:             input.SKU,
			Description:     input.Description,
			ProductTypeCode: "sale",
			ProductLineID:   input.ProductLineID,
			CategoryID:      input.ItemCategoryID,
			IsPortalReady:   false,
		}
		if apiErr := txProductRepo.InsertItem(txCtx, itemID, params); apiErr != nil {
			return apiErr
		}

		// Insert product record.
		if _, apiErr := txProductRepo.Create(txCtx, productID, itemID, params); apiErr != nil {
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
	})
}

func (s *itemSvcImpl) bulkCreateMaterial(ctx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, input domain.BulkCreateItemInput) *apierror.APIError {
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

	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		txMaterialRepo := txSvc.repos.NewMaterialRepo()

		// Insert rates.
		if apiErr := txMaterialRepo.InsertRate(txCtx, unitValueRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txMaterialRepo.InsertRate(txCtx, unitCostRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txMaterialRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}

		// Insert item.
		params := domain.CreateMaterialParams{
			AccountID:       accountID,
			ItemID:          itemID,
			UnitValueRateID: unitValueRateID,
			UnitCostRateID:  unitCostRateID,
			BurnRateRateID:  burnRateRateID,
			OrderPointID:    orderPointQtyID,
			LeadTimeID:      leadTimeQtyID,
			SKU:             input.SKU,
			Description:     input.Description,
			CategoryID:      input.ItemCategoryID,
		}
		if apiErr := txMaterialRepo.InsertItem(txCtx, itemID, params); apiErr != nil {
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
		if apiErr := txMaterialRepo.Create(txCtx, materialID, params); apiErr != nil {
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
	})
}

func (s *itemSvcImpl) bulkCreatePart(ctx context.Context, accountID, itemID, unitValueRateID, unitCostRateID, burnRateRateID, baseUnitID string, input domain.BulkCreateItemInput) *apierror.APIError {
	partID, apiErr := id.GenID(id.PartIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		txPartRepo := txSvc.repos.NewPartRepo()

		// Insert rates.
		if apiErr := txPartRepo.InsertRate(txCtx, unitValueRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txPartRepo.InsertRate(txCtx, unitCostRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
			return apiErr
		}
		if apiErr := txPartRepo.InsertRate(txCtx, burnRateRateID, "0", baseUnitID, baseUnitID); apiErr != nil {
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
	})
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
				result.Errors = append(result.Errors, domain.ReconcileError{SKU: d.SKU, Error: fmt.Sprintf("Unit '%s' not found", d.Unit)})
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
		// Uses physical inventory (receipts - open issues) instead of ATP to match Dashboard's
		// physicalInventory metric.
		invQueryRepo := s.repos.NewInventoryQueryRepo()
		physicalInvMap := make(map[string]float64)
		for _, d := range validItems {
			item := itemMap[d.SKU]
			if _, already := physicalInvMap[item.ItemID]; already {
				continue
			}
			physInv, fetchErr := invQueryRepo.FetchPhysicalInventory(ctx, item.ItemID, accountID)
			if fetchErr != nil {
				// Skip items where inventory cannot be fetched, matching Dashboard behavior
				// where items with no currentInventory are silently skipped.
				continue
			}
			physicalInvMap[item.ItemID] = physInv
		}

		// Process in batches of 50
		batchSize := 50
		for batchStart := 0; batchStart < len(validItems); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(validItems) {
				batchEnd = len(validItems)
			}
			batch := validItems[batchStart:batchEnd]

			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
				invMutRepo := txSvc.repos.NewInventoryMutationRepo()

				for _, d := range batch {
					item := itemMap[d.SKU]

					currentQty, ok := physicalInvMap[item.ItemID]
					if !ok {
						// Item has no inventory data; skip silently (matches Dashboard behavior
						// where items without currentInventory are skipped without error).
						continue
					}

					var newQty, delta float64
					if params.ReconcileType == "force" {
						newQty = d.Quantity
						delta = d.Quantity - currentQty
					} else { // addition
						delta = d.Quantity
						newQty = currentQty + delta
					}

					measure := decimal.NewFromFloat(math.Abs(delta))
					unitID := item.BaseUnitID

					if delta > 0 {
						if apiErr := invMutRepo.CreateInventoryReceipt(txCtx, domain.CreateInventoryReceiptParams{
							AccountID: accountID, ItemID: item.ItemID, Measure: measure, UnitID: unitID,
						}); apiErr != nil {
							result.Errors = append(result.Errors, domain.ReconcileError{SKU: d.SKU, Error: "Failed to create receipt"})
							continue
						}
					} else if delta < 0 {
						if apiErr := invMutRepo.CreateInventoryIssue(txCtx, domain.CreateInventoryIssueParams{
							AccountID: accountID, ItemID: item.ItemID, Measure: measure, UnitID: unitID,
						}); apiErr != nil {
							result.Errors = append(result.Errors, domain.ReconcileError{SKU: d.SKU, Error: "Failed to create issue"})
							continue
						}
					}

					// Create inventory log
					logMeasure := decimal.NewFromFloat(newQty)
					if apiErr := invMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
						AccountID: accountID, ItemID: item.ItemID, Measure: logMeasure, UnitID: unitID,
					}); apiErr != nil {
						result.Errors = append(result.Errors, domain.ReconcileError{SKU: d.SKU, Error: "Failed to create log"})
						continue
					}

					// Create change log
					changeMeasure := decimal.NewFromFloat(delta)
					if apiErr := invMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
						AccountID: accountID, ItemID: item.ItemID, Measure: changeMeasure, UnitID: unitID,
						ActionType: "user_correction", ResponsibleUserID: params.ResponsibleUserID,
					}); apiErr != nil {
						result.Errors = append(result.Errors, domain.ReconcileError{SKU: d.SKU, Error: "Failed to create change log"})
						continue
					}

					result.ReconciledItems = append(result.ReconciledItems, domain.ReconciledItem{
						ItemID: item.ItemID, SKU: d.SKU,
						PreviousQuantity: currentQty, NewQuantity: newQty,
					})

					if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
						ServiceName:  domain.ServiceName,
						Action:       constants.AuditActionUpdate,
						ResourceType: constants.ObjectTypeItem,
						ResourceID:   item.ItemID,
					}); apiErr != nil {
						result.Errors = append(result.Errors, domain.ReconcileError{SKU: d.SKU, Error: "Failed to publish audit event"})
						continue
					}
				}

				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})

			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}
