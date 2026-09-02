package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var batchSvcTracer = tracing.GetTracer("core-service.batch_service")

type batchSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type BatchSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *BatchSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("batch service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("batch service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("batch service: tx manager is required")
	}
	return nil
}

func NewBatchSvc(config *BatchSvcConfig) domain.BatchSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &batchSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *batchSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *batchSvcImpl) withTx(ctx context.Context, fn func(context.Context, *batchSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &batchSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// enqueueExecuteProductionStep writes an outbox message for the execute production step side-effect.
func (s *batchSvcImpl) enqueueExecuteProductionStep(ctx context.Context, repos domain.RepoFactory, evt domain.ExecuteProductionStepEvent) *apierror.APIError {
	payload, err := json.Marshal(evt)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal execute production step event.")
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
		MessageType: string(contracts.CoreCmdExecuteProductionStep),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdExecuteProductionStep),
		Payload:     msg,
	}

	outboxRepo := repos.NewOutboxRepo()
	if _, err := outboxRepo.Create(ctx, outboxInput); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for execute production step.")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Read-only methods (no idempotency)
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) GetBatchFlow(ctx context.Context, batchID string) ([]domain.BatchFlowNode, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_batch_flow")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewBatchRepo().FindBatchFlow(ctx, identity.Target.AccountID, batchID)
}

func (s *batchSvcImpl) ListBatchesByScanningStation(ctx context.Context, params domain.ListBatchesByScanningStationParams) (*domain.ListBatchesByScanningStationResult, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.list_by_scanning_station")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	return s.repos.NewBatchRepo().FindByScanningStation(ctx, params)
}

func (s *batchSvcImpl) GetPossibleNextSteps(ctx context.Context, scanningStationID, batchID string) ([]domain.ScanningProductionStepInfo, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_possible_next_steps")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewBatchRepo().FindPossibleNextSteps(ctx, identity.Target.AccountID, scanningStationID, batchID)
}

func (s *batchSvcImpl) GetPossibleInitSteps(ctx context.Context, scanningStationID, batchID string) ([]domain.ScanningProductionStepInfo, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_possible_init_steps")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewBatchRepo().FindPossibleInitSteps(ctx, identity.Target.AccountID, scanningStationID, batchID)
}

func (s *batchSvcImpl) AnalyzeOpenBatches(ctx context.Context, itemIDs, productLineIDs []string) ([]domain.OpenBatchSummary, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.analyze_open_batches")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewBatchRepo().FindOpenBatches(ctx, identity.Target.AccountID, itemIDs, productLineIDs)
}

func (s *batchSvcImpl) GetRemainingQuantityToSplit(ctx context.Context, batchIDs []string, productionStepID string) (*domain.BatchQuantity, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_remaining_quantity_to_split")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	batchRepo := s.repos.NewBatchRepo()
	stepRepo := s.repos.NewProductionStepQueryRepo()

	if len(batchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("No batches to split."))
	}

	if len(batchIDs) == 1 {
		return s.getRemainingQuantitySinglePart(ctx, accountID, batchIDs[0], productionStepID, batchRepo, stepRepo)
	}
	return s.getRemainingQuantityMultiPart(ctx, accountID, batchIDs, productionStepID, batchRepo, stepRepo)
}

func (s *batchSvcImpl) getRemainingQuantitySinglePart(ctx context.Context, accountID, batchID, productionStepID string, batchRepo domain.BatchRepo, stepRepo domain.ProductionStepQueryRepo) (*domain.BatchQuantity, *apierror.APIError) {
	// Find the available source batch in the flow.
	sourceBatch, apiErr := batchRepo.FindNextAvailableBatchInFlow(ctx, accountID, batchID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Calculate the expected output quantity.
	nextStepResult, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, sourceBatch.Item.ID, sourceBatch.Quantity, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Get output batches and sum their quantities.
	outputBatches, apiErr := batchRepo.FindOutputBatches(ctx, accountID, sourceBatch.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	totalUsed := decimal.Zero
	for _, ob := range outputBatches {
		totalUsed = totalUsed.Add(ob.Quantity.Measure)
		if ob.Seconds != nil {
			totalUsed = totalUsed.Add(ob.Seconds.Measure)
		}
		if ob.Waste != nil {
			totalUsed = totalUsed.Add(ob.Waste.Measure)
		}
	}

	remaining := nextStepResult.Quantity.Sub(totalUsed)

	producedUnit, apiErr := stepRepo.FindProducedUnit(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.BatchQuantity{
		Measure: remaining,
		Unit:    *producedUnit,
	}, nil
}

func (s *batchSvcImpl) getRemainingQuantityMultiPart(ctx context.Context, accountID string, batchIDs []string, productionStepID string, batchRepo domain.BatchRepo, stepRepo domain.ProductionStepQueryRepo) (*domain.BatchQuantity, *apierror.APIError) {
	// Find all available source batches.
	sourceBatches, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, batchIDs, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Get the production step to verify all consumptions have matching batches.
	productionStep, apiErr := stepRepo.Find(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if productionStep == nil {
		return nil, apierror.NewResourceNotFoundError("Production step not found.")
	}

	// Group by item and sum quantities per group, then calculate output quantity.
	type blockGroup struct {
		itemID        string
		totalQuantity decimal.Decimal
		unitID        string
	}
	groupMap := make(map[string]*blockGroup)

	for _, b := range sourceBatches {
		g, exists := groupMap[b.Item.ID]
		if !exists {
			g = &blockGroup{itemID: b.Item.ID, totalQuantity: decimal.Zero, unitID: b.Quantity.Unit.ID}
			groupMap[b.Item.ID] = g
		}
		g.totalQuantity = g.totalQuantity.Add(b.Quantity.Measure)
	}

	// Verify all required consumptions have corresponding batches.
	for _, consumption := range productionStep.Consumptions {
		blockBatches, exists := groupMap[consumption.ConsumedItem.ID]
		if !exists || blockBatches.totalQuantity.IsZero() {
			return nil, apierror.NewValidationErrorWithParam("Missing required part: "+consumption.ConsumedItem.SKU, "batch_ids")
		}
	}

	// Calculate the output quantity for each block and validate they're all equal.
	var outputQuantity decimal.Decimal
	var producedUnitID string
	first := true
	for _, group := range groupMap {
		bq := domain.BatchQuantity{Measure: group.totalQuantity, Unit: domain.LightUnit{ID: group.unitID}}
		result, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, group.itemID, bq, productionStepID)
		if apiErr != nil {
			return nil, apiErr
		}

		if first {
			outputQuantity = result.Quantity
			producedUnitID = result.ProducedUnitID
			first = false
		} else if !outputQuantity.Equal(result.Quantity) {
			return nil, apierror.NewValidationError("Calculated output quantities do not match across input blocks.")
		}
	}

	// Collect all unique output batch IDs from all source batches.
	outputBatchMap := make(map[string]bool)
	var allOutputBatches []domain.BaseBatch
	for _, b := range sourceBatches {
		outputs, apiErr := batchRepo.FindOutputBatches(ctx, accountID, b.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, ob := range outputs {
			if !outputBatchMap[ob.ID] {
				outputBatchMap[ob.ID] = true
				allOutputBatches = append(allOutputBatches, ob)
			}
		}
	}

	// In the multi-part path, only sum the primary quantity (not seconds/waste).
	totalUsed := decimal.Zero
	for _, ob := range allOutputBatches {
		totalUsed = totalUsed.Add(ob.Quantity.Measure)
	}

	remaining := outputQuantity.Sub(totalUsed)

	producedUnit, apiErr := stepRepo.FindProducedUnit(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	_ = producedUnitID // already used producedUnit from FindProducedUnit

	return &domain.BatchQuantity{
		Measure: remaining,
		Unit:    *producedUnit,
	}, nil
}

func (s *batchSvcImpl) GetScanningStationConsumption(ctx context.Context, params domain.GetConsumptionParams) ([]domain.ScanningConsumption, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_scanning_station_consumption")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	ssRepo := s.repos.NewScanningStationQueryRepo()
	stepRepo := s.repos.NewProductionStepQueryRepo()
	batchRepo := s.repos.NewBatchRepo()
	invRepo := s.repos.NewInventoryQueryRepo()

	// Validate scanning station in account.
	inAccount, apiErr := ssRepo.IsInAccount(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
	}

	// Get scanning station type.
	stationType, apiErr := ssRepo.FindType(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	ssType := constants.ScanningStationType(stationType)

	switch ssType {
	case constants.ScanningStationTypeInitBatch:
		return s.getInitBatchConsumption(ctx, accountID, params, stepRepo, batchRepo, invRepo)
	case constants.ScanningStationTypeMoveBatch:
		return s.getMoveBatchConsumption(ctx, accountID, params, stepRepo, batchRepo, invRepo)
	case constants.ScanningStationTypeSplitBatch:
		return s.getSplitBatchConsumption(ctx, accountID, params, stepRepo, batchRepo, invRepo)
	case constants.ScanningStationTypeMergeBatch:
		return s.getMergeBatchConsumption(ctx, accountID, params, stepRepo, batchRepo, invRepo)
	default:
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid scanning station type."))
	}
}

func (s *batchSvcImpl) getInitBatchConsumption(ctx context.Context, accountID string, params domain.GetConsumptionParams, stepRepo domain.ProductionStepQueryRepo, batchRepo domain.BatchRepo, invRepo domain.InventoryQueryRepo) ([]domain.ScanningConsumption, *apierror.APIError) {
	if len(params.BatchIDs) == 0 {
		return nil, apierror.NewValidationError("At least one batch ID is required.")
	}

	// Get the batch to find the item.
	batch, apiErr := batchRepo.Find(ctx, accountID, params.BatchIDs[0])
	if apiErr != nil {
		return nil, apiErr
	}

	// Find the production step by scanning station + produced block.
	step, apiErr := stepRepo.FindOneByScanningStationAndProducedBlock(ctx, accountID, params.ScanningStationID, batch.Item.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Calculate execution multiplier: batchQuantity / productionStepQuantity.
	if step.Production.Quantity.Measure.IsZero() {
		return nil, apierror.NewInternalError(nil, "Production step quantity is zero.")
	}
	multiplier := batch.Quantity.Measure.Div(step.Production.Quantity.Measure)

	return s.calculateConsumptionDemand(ctx, accountID, step, multiplier, invRepo)
}

func (s *batchSvcImpl) getMoveBatchConsumption(ctx context.Context, accountID string, params domain.GetConsumptionParams, stepRepo domain.ProductionStepQueryRepo, batchRepo domain.BatchRepo, invRepo domain.InventoryQueryRepo) ([]domain.ScanningConsumption, *apierror.APIError) {
	if params.ProductionStepID == nil {
		return nil, apierror.NewValidationError("Production step ID is required for move consumption.")
	}

	step, apiErr := stepRepo.Find(ctx, accountID, *params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	isMultiPart := len(step.Consumptions) > 1

	if isMultiPart || len(params.BatchIDs) > 1 {
		return s.getManyBatchConsumption(ctx, accountID, params, step, stepRepo, batchRepo, invRepo)
	}

	// Single batch move: find furthest right batch and calculate.
	furthestBatch, apiErr := batchRepo.FindFurthestRightBatchInFlow(ctx, accountID, params.BatchIDs[0])
	if apiErr != nil {
		return nil, apiErr
	}
	if furthestBatch.ClosedAt != nil {
		return nil, apierror.NewValidationError("Batch is closed.")
	}

	nextStepResult, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, furthestBatch.Item.ID, furthestBatch.Quantity, *params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	bq := domain.BatchQuantity{Measure: nextStepResult.Quantity, Unit: domain.LightUnit{ID: nextStepResult.ProducedUnitID}}
	if step.Production.Quantity.Measure.IsZero() {
		return nil, apierror.NewInternalError(nil, "Production step quantity is zero.")
	}
	multiplier := bq.Measure.Div(step.Production.Quantity.Measure)

	return s.calculateConsumptionDemand(ctx, accountID, step, multiplier, invRepo)
}

func (s *batchSvcImpl) getSplitBatchConsumption(ctx context.Context, accountID string, params domain.GetConsumptionParams, stepRepo domain.ProductionStepQueryRepo, batchRepo domain.BatchRepo, invRepo domain.InventoryQueryRepo) ([]domain.ScanningConsumption, *apierror.APIError) {
	if params.ProductionStepID == nil {
		return nil, apierror.NewValidationError("Production step ID is required for split consumption.")
	}
	if params.SplitQuantity == nil {
		return nil, apierror.NewValidationError("Split quantity is required for split consumption.")
	}

	step, apiErr := stepRepo.Find(ctx, accountID, *params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Match Dashboard routing: isMultiPart or multiple batches go to many-batch path.
	isMultiPart := len(step.Consumptions) > 1
	if isMultiPart || len(params.BatchIDs) > 1 {
		return s.getManyBatchConsumption(ctx, accountID, params, step, stepRepo, batchRepo, invRepo)
	}

	// Single batch split: find furthest right batch and validate not closed.
	if len(params.BatchIDs) == 0 {
		return nil, apierror.NewValidationError("At least one batch ID is required.")
	}
	furthestBatch, apiErr := batchRepo.FindFurthestRightBatchInFlow(ctx, accountID, params.BatchIDs[0])
	if apiErr != nil {
		return nil, apiErr
	}
	if furthestBatch.ClosedAt != nil {
		return nil, apierror.NewValidationError("Batch is closed.")
	}

	if step.Production.Quantity.Measure.IsZero() {
		return nil, apierror.NewInternalError(nil, "Production step quantity is zero.")
	}
	multiplier := params.SplitQuantity.Measure.Div(step.Production.Quantity.Measure)

	return s.calculateConsumptionDemand(ctx, accountID, step, multiplier, invRepo)
}

func (s *batchSvcImpl) getMergeBatchConsumption(ctx context.Context, accountID string, params domain.GetConsumptionParams, stepRepo domain.ProductionStepQueryRepo, batchRepo domain.BatchRepo, invRepo domain.InventoryQueryRepo) ([]domain.ScanningConsumption, *apierror.APIError) {
	if params.ProductionStepID == nil {
		return nil, apierror.NewValidationError("Production step ID is required for merge consumption.")
	}

	step, apiErr := stepRepo.Find(ctx, accountID, *params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	return s.getManyBatchConsumption(ctx, accountID, params, step, stepRepo, batchRepo, invRepo)
}

func (s *batchSvcImpl) getManyBatchConsumption(ctx context.Context, accountID string, params domain.GetConsumptionParams, step *domain.ProductionStepDetail, stepRepo domain.ProductionStepQueryRepo, batchRepo domain.BatchRepo, invRepo domain.InventoryQueryRepo) ([]domain.ScanningConsumption, *apierror.APIError) {
	if len(params.BatchIDs) == 0 {
		return nil, apierror.NewValidationError("At least one batch ID is required.")
	}

	// Find available batches in flow for each batch ID.
	sourceBatches, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, params.BatchIDs, *params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Validate no duplicate batches.
	seen := make(map[string]struct{}, len(sourceBatches))
	for _, b := range sourceBatches {
		if _, dup := seen[b.ID]; dup {
			return nil, apierror.NewValidationError("Duplicate batches provided.")
		}
		seen[b.ID] = struct{}{}
	}

	// Validate all requested batches were found.
	if len(sourceBatches) != len(params.BatchIDs) {
		return nil, apierror.NewResourceNotFoundError("One or more batches not found.")
	}

	// Calculate next step quantities for each batch individually and sum them, matching the Dashboard behavior of summing per-batch output quantities.
	totalOutputQuantity := decimal.Zero
	for _, b := range sourceBatches {
		result, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, b.Item.ID, b.Quantity, *params.ProductionStepID)
		if apiErr != nil {
			return nil, apiErr
		}
		totalOutputQuantity = totalOutputQuantity.Add(result.Quantity)
	}

	if step.Production.Quantity.Measure.IsZero() {
		return nil, apierror.NewInternalError(nil, "Production step quantity is zero.")
	}
	multiplier := totalOutputQuantity.Div(step.Production.Quantity.Measure)

	return s.calculateConsumptionDemand(ctx, accountID, step, multiplier, invRepo)
}

func (s *batchSvcImpl) calculateConsumptionDemand(ctx context.Context, accountID string, step *domain.ProductionStepDetail, multiplier decimal.Decimal, invRepo domain.InventoryQueryRepo) ([]domain.ScanningConsumption, *apierror.APIError) {
	results := make([]domain.ScanningConsumption, 0, len(step.Consumptions))

	for _, c := range step.Consumptions {
		demandQuantity := c.Quantity.Measure.Mul(multiplier)

		inventory, apiErr := invRepo.FetchCurrentInventory(ctx, c.ConsumedItem.ID, accountID)
		if apiErr != nil {
			return nil, apiErr
		}

		sc := domain.ScanningConsumption{
			SKU:              c.ConsumedItem.SKU,
			DemandMeasure:    formatQuantity(demandQuantity),
			DemandUnit:       c.Quantity.Unit.Abbreviation,
			InventoryMeasure: formatQuantity(inventory.AvailableToPromiseMeasure),
			InventoryUnit:    inventory.AvailableToPromiseUnitAbbreviation,
			Instructions:     c.Instructions,
		}

		results = append(results, sc)
	}

	return results, nil
}

// formatQuantity formats a decimal for display by removing trailing zeros.
func formatQuantity(d decimal.Decimal) string {
	return d.StringFixed(4)
}

// ---------------------------------------------------------------------------
// Mutation methods (with idempotency)
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) InitializeBatch(ctx context.Context, batchID, scanningStationID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.initialize")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionCreate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.BaseBatch](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Validate scanning station.
		ssRepo := s.repos.NewScanningStationQueryRepo()
		inAccount, apiErr := ssRepo.IsInAccount(ctx, accountID, scanningStationID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if !inAccount {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
		}

		// Validate batch.
		batchRepo := s.repos.NewBatchRepo()
		batch, apiErr := batchRepo.Find(ctx, accountID, batchID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if batch.ClosedAt != nil {
			return nil, tracing.Trace(span, apierror.NewValidationError("This batch is closed."))
		}
		if batch.ScannedAt != nil {
			return nil, tracing.Trace(span, apierror.NewValidationError("This batch has been scanned already."))
		}
		if batch.ProductionRun == nil {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production run not found."))
		}

		// Find the production step for this scanning station + item.
		stepRepo := s.repos.NewProductionStepQueryRepo()
		productionStepID, apiErr := stepRepo.FindIDByScanningStationAndProducedBlock(ctx, accountID, scanningStationID, batch.Item.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.BaseBatch
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txSvc.repos)
			txBatchRepo := txSvc.repos.NewBatchRepo()

			// Mark as scanned.
			if apiErr := txBatchRepo.MarkAsScanned(txCtx, accountID, batchID); apiErr != nil {
				return apiErr
			}

			// Connect production step and scanning station.
			if apiErr := txBatchRepo.ConnectProductionStep(txCtx, accountID, batchID, productionStepID); apiErr != nil {
				return apiErr
			}
			if apiErr := txBatchRepo.ConnectScanningStation(txCtx, accountID, batchID, scanningStationID); apiErr != nil {
				return apiErr
			}

			// Close if last step.
			if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, batchID, productionStepID); apiErr != nil {
				return apiErr
			}

			// Enqueue outbox message for executeProductionStep.
			evt := domain.ExecuteProductionStepEvent{
				ProductionStepID:  productionStepID,
				ScanningStationID: scanningStationID,
				ItemID:            batch.Item.ID,
				BatchQuantityID:   batch.Quantity.ID,
				BatchMeasure:      batch.Quantity.Measure.String(),
				BatchUnitID:       batch.Quantity.Unit.ID,
				ProducedBatchID:   &batchID,
				ProduceInventory:  true,
			}
			if identity.Actor != nil {
				evt.ResponsibleUserID = &identity.Actor.ID
			}
			if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
				return apiErr
			}

			// Fetch the updated batch.
			updatedRow, apiErr := txSvc.repos.NewBatchRepo().Find(ctx, accountID, batchID)
			if apiErr != nil {
				return apiErr
			}
			result = &domain.BaseBatch{
				ID:              updatedRow.ID,
				Item:            updatedRow.Item,
				Quantity:        updatedRow.Quantity,
				Seconds:         updatedRow.Seconds,
				Waste:           updatedRow.Waste,
				ScanningStation: updatedRow.ScanningStation,
				ProductionStep:  updatedRow.ProductionStep,
				ProductionRun:   updatedRow.ProductionRun,
				ClosedAt:        updatedRow.ClosedAt,
				ScannedAt:       updatedRow.ScannedAt,
				CreatedAt:       updatedRow.CreatedAt,
				UpdatedAt:       updatedRow.UpdatedAt,
			}

			oldBatch := &domain.BaseBatch{
				ID:              batch.ID,
				Item:            batch.Item,
				Quantity:        batch.Quantity,
				Seconds:         batch.Seconds,
				Waste:           batch.Waste,
				ScanningStation: batch.ScanningStation,
				ProductionStep:  batch.ProductionStep,
				ProductionRun:   batch.ProductionRun,
				ClosedAt:        batch.ClosedAt,
				ScannedAt:       batch.ScannedAt,
				CreatedAt:       batch.CreatedAt,
				UpdatedAt:       batch.UpdatedAt,
			}

			changes := audit.ComputeChanges(oldBatch, result)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeBatch,
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

		// After transaction: start production run, close if all batches scanned.
		if batch.ProductionRun != nil {
			runRepo := s.repos.NewProductionRunQueryRepo()
			_ = runRepo.Start(ctx, accountID, batch.ProductionRun.ID)
			_ = runRepo.CloseIfAllBatchesScannedOrDeleted(ctx, accountID, batch.ProductionRun.ID)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *batchSvcImpl) MoveBatches(ctx context.Context, params domain.MoveBatchesParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.move")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	meds := s.mediators()

	// Validate scanning station and production step in account.
	ssRepo := s.repos.NewScanningStationQueryRepo()
	inAccount, apiErr := ssRepo.IsInAccount(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
	}

	stepRepo := s.repos.NewProductionStepQueryRepo()
	inAccount, apiErr = stepRepo.IsInAccount(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	if len(params.BatchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("At least one batch ID is required."))
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.BaseBatch](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		if len(params.BatchIDs) == 1 {
			result, apiErr := s.moveSinglePartBatch(ctx, accountID, identity, params, stepRepo)
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
			}
			return result, nil
		}
		result, apiErr := s.moveMultiPartBatch(ctx, accountID, identity, params, stepRepo)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *batchSvcImpl) moveSinglePartBatch(ctx context.Context, accountID string, identity *types.Identity, params domain.MoveBatchesParams, stepRepo domain.ProductionStepQueryRepo) (*domain.BaseBatch, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()
	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	// Find the available source batch in the flow.
	sourceBatch, apiErr := batchRepo.FindNextAvailableBatchInFlow(ctx, accountID, params.BatchIDs[0], params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if sourceBatch.ClosedAt != nil {
		return nil, apierror.NewValidationError("Source batch is closed.")
	}

	// Calculate next step quantities.
	nextStep, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, sourceBatch.Item.ID, sourceBatch.Quantity, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		created, apiErr := txBatchRepo.Create(txCtx, newBatchID, domain.CreateBatchParams{
			AccountID:         accountID,
			ItemID:            nextStep.ItemID,
			Quantity:          domain.CreateQuantityParams{Measure: nextStep.Quantity, UnitID: nextStep.ProducedUnitID},
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.ConnectOneToOne(txCtx, accountID, sourceBatch.ID, newBatchID, true); apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, params.ProductionStepID); apiErr != nil {
			return apiErr
		}

		// Enqueue outbox.
		evt := domain.ExecuteProductionStepEvent{
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
			ItemID:            nextStep.ItemID,
			BatchQuantityID:   created.Quantity.ID,
			BatchMeasure:      created.Quantity.Measure.String(),
			BatchUnitID:       created.Quantity.Unit.ID,
			ProducedBatchID:   &newBatchID,
			ProduceInventory:  true,
		}
		if identity.Actor != nil {
			evt.ResponsibleUserID = &identity.Actor.ID
		}
		if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		result = created
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})

	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) moveMultiPartBatch(ctx context.Context, accountID string, identity *types.Identity, params domain.MoveBatchesParams, stepRepo domain.ProductionStepQueryRepo) (*domain.BaseBatch, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()
	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	// Find all available source batches.
	sourceBatches, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, params.BatchIDs, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Validate no duplicates.
	seen := make(map[string]bool)
	for _, b := range sourceBatches {
		if seen[b.ID] {
			return nil, apierror.NewValidationError("Duplicate batch found in flow.")
		}
		seen[b.ID] = true
		if b.ClosedAt != nil {
			return nil, apierror.NewValidationError("Source batch is closed.")
		}
	}

	// Calculate next step quantities for each and validate all are the same.
	var outputQuantity decimal.Decimal
	var outputItemID string
	var outputUnitID string
	first := true
	for _, b := range sourceBatches {
		nextStep, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, b.Item.ID, b.Quantity, params.ProductionStepID)
		if apiErr != nil {
			return nil, apiErr
		}
		if first {
			outputQuantity = nextStep.Quantity
			outputItemID = nextStep.ItemID
			outputUnitID = nextStep.ProducedUnitID
			first = false
		} else if !outputQuantity.Equal(nextStep.Quantity) {
			return nil, apierror.NewValidationError("Calculated output quantities do not match across input batches.")
		}
	}

	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	sourceBatchIDs := make([]string, len(sourceBatches))
	for i, b := range sourceBatches {
		sourceBatchIDs[i] = b.ID
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		created, apiErr := txBatchRepo.Create(txCtx, newBatchID, domain.CreateBatchParams{
			AccountID:         accountID,
			ItemID:            outputItemID,
			Quantity:          domain.CreateQuantityParams{Measure: outputQuantity, UnitID: outputUnitID},
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.ConnectManyToOne(txCtx, accountID, sourceBatchIDs, newBatchID, true); apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, params.ProductionStepID); apiErr != nil {
			return apiErr
		}

		evt := domain.ExecuteProductionStepEvent{
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
			ItemID:            outputItemID,
			BatchQuantityID:   created.Quantity.ID,
			BatchMeasure:      created.Quantity.Measure.String(),
			BatchUnitID:       created.Quantity.Unit.ID,
			ProducedBatchID:   &newBatchID,
			ProduceInventory:  true,
		}
		if identity.Actor != nil {
			evt.ResponsibleUserID = &identity.Actor.ID
		}
		if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		result = created
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})

	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) MergeBatches(ctx context.Context, params domain.MergeBatchesParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.merge")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	meds := s.mediators()

	// Validate scanning station.
	ssRepo := s.repos.NewScanningStationQueryRepo()
	inAccount, apiErr := ssRepo.IsInAccount(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
	}

	if len(params.BatchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("At least one batch ID is required."))
	}

	// Check for duplicate batch IDs.
	seen := make(map[string]struct{}, len(params.BatchIDs))
	for _, bid := range params.BatchIDs {
		if _, exists := seen[bid]; exists {
			return nil, tracing.Trace(span, apierror.NewValidationError("Duplicate batches provided."))
		}
		seen[bid] = struct{}{}
	}

	stepRepo := s.repos.NewProductionStepQueryRepo()
	isMultiPart, apiErr := stepRepo.IsMultiPart(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.BaseBatch](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.BaseBatch
		if isMultiPart {
			result, apiErr = s.mergeMultiPartBatches(ctx, accountID, identity, params, stepRepo, idempotencyKey)
		} else {
			result, apiErr = s.mergeSinglePartBatches(ctx, accountID, identity, params, stepRepo, idempotencyKey)
		}
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *batchSvcImpl) mergeSinglePartBatches(ctx context.Context, accountID string, identity *types.Identity, params domain.MergeBatchesParams, stepRepo domain.ProductionStepQueryRepo, idempotencyKey *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()
	meds := s.mediators()

	sourceBatches, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, params.BatchIDs, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Validate all batches have the same item.
	if len(sourceBatches) == 0 {
		return nil, apierror.NewValidationError("No available batches found.")
	}
	expectedItemID := sourceBatches[0].Item.ID
	totalQuantity := decimal.Zero
	for _, b := range sourceBatches {
		if b.Item.ID != expectedItemID {
			return nil, apierror.NewValidationError("All batches must have the same item for single-part merge.")
		}
		totalQuantity = totalQuantity.Add(b.Quantity.Measure)
	}

	// Calculate next step quantities.
	bq := domain.BatchQuantity{Measure: totalQuantity, Unit: sourceBatches[0].Quantity.Unit}
	nextStep, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, expectedItemID, bq, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	sourceBatchIDs := make([]string, len(sourceBatches))
	for i, b := range sourceBatches {
		sourceBatchIDs[i] = b.ID
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		created, apiErr := txBatchRepo.Create(txCtx, newBatchID, domain.CreateBatchParams{
			AccountID:         accountID,
			ItemID:            nextStep.ItemID,
			Quantity:          domain.CreateQuantityParams{Measure: nextStep.Quantity, UnitID: nextStep.ProducedUnitID},
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.ConnectManyToOne(txCtx, accountID, sourceBatchIDs, newBatchID, true); apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, params.ProductionStepID); apiErr != nil {
			return apiErr
		}

		evt := domain.ExecuteProductionStepEvent{
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
			ItemID:            nextStep.ItemID,
			BatchQuantityID:   created.Quantity.ID,
			BatchMeasure:      created.Quantity.Measure.String(),
			BatchUnitID:       created.Quantity.Unit.ID,
			ProducedBatchID:   &newBatchID,
			ProduceInventory:  true,
		}
		if identity.Actor != nil {
			evt.ResponsibleUserID = &identity.Actor.ID
		}
		if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		result = created
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})

	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) mergeMultiPartBatches(ctx context.Context, accountID string, identity *types.Identity, params domain.MergeBatchesParams, stepRepo domain.ProductionStepQueryRepo, idempotencyKey *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()
	meds := s.mediators()

	sourceBatches, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, params.BatchIDs, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(sourceBatches) == 0 {
		return nil, apierror.NewValidationError("No available batches found.")
	}

	// Fetch production step to validate all required consumptions are covered.
	productionStep, apiErr := stepRepo.Find(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if productionStep == nil {
		return nil, apierror.NewResourceNotFoundError("Production step not found.")
	}

	// Group by item and sum.
	type blockGroup struct {
		itemID        string
		totalQuantity decimal.Decimal
		unit          domain.LightUnit
	}
	groupMap := make(map[string]*blockGroup)
	for _, b := range sourceBatches {
		g, exists := groupMap[b.Item.ID]
		if !exists {
			g = &blockGroup{itemID: b.Item.ID, totalQuantity: decimal.Zero, unit: b.Quantity.Unit}
			groupMap[b.Item.ID] = g
		}
		g.totalQuantity = g.totalQuantity.Add(b.Quantity.Measure)
	}

	// Verify all required parts (consumptions) have corresponding batches.
	for _, consumption := range productionStep.Consumptions {
		if _, exists := groupMap[consumption.ConsumedItem.ID]; !exists {
			return nil, apierror.NewValidationError("Missing required part: " + consumption.ConsumedItem.SKU)
		}
	}

	// Calculate output quantity for each block and validate they're all the same.
	var outputQuantity decimal.Decimal
	var outputItemID string
	var outputUnitID string
	first := true
	for _, group := range groupMap {
		bq := domain.BatchQuantity{Measure: group.totalQuantity, Unit: group.unit}
		result, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, group.itemID, bq, params.ProductionStepID)
		if apiErr != nil {
			return nil, apiErr
		}
		if first {
			outputQuantity = result.Quantity
			outputItemID = result.ItemID
			outputUnitID = result.ProducedUnitID
			first = false
		} else if !outputQuantity.Equal(result.Quantity) {
			return nil, apierror.NewValidationError("Calculated output quantities do not match across input blocks.")
		}
	}

	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	sourceBatchIDs := make([]string, len(sourceBatches))
	for i, b := range sourceBatches {
		sourceBatchIDs[i] = b.ID
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		created, apiErr := txBatchRepo.Create(txCtx, newBatchID, domain.CreateBatchParams{
			AccountID:         accountID,
			ItemID:            outputItemID,
			Quantity:          domain.CreateQuantityParams{Measure: outputQuantity, UnitID: outputUnitID},
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.ConnectManyToOne(txCtx, accountID, sourceBatchIDs, newBatchID, true); apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, params.ProductionStepID); apiErr != nil {
			return apiErr
		}

		evt := domain.ExecuteProductionStepEvent{
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
			ItemID:            outputItemID,
			BatchQuantityID:   created.Quantity.ID,
			BatchMeasure:      created.Quantity.Measure.String(),
			BatchUnitID:       created.Quantity.Unit.ID,
			ProducedBatchID:   &newBatchID,
			ProduceInventory:  true,
		}
		if identity.Actor != nil {
			evt.ResponsibleUserID = &identity.Actor.ID
		}
		if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		result = created
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})

	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) SplitBatch(ctx context.Context, params domain.SplitBatchParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.split")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	meds := s.mediators()

	// Validate scanning station.
	ssRepo := s.repos.NewScanningStationQueryRepo()
	inAccount, apiErr := ssRepo.IsInAccount(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning station not found."))
	}

	if len(params.BatchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("At least one batch ID is required."))
	}

	// Validate at least one non-zero quantity.
	hasNonZero := !params.Firsts.Measure.IsZero()
	if params.Seconds != nil && !params.Seconds.Measure.IsZero() {
		hasNonZero = true
	}
	if params.Waste != nil && !params.Waste.Measure.IsZero() {
		hasNonZero = true
	}
	if !hasNonZero {
		return nil, tracing.Trace(span, apierror.NewValidationError("At least one of firsts, seconds, or waste must have a non-zero quantity."))
	}

	stepRepo := s.repos.NewProductionStepQueryRepo()
	isMultiPart, apiErr := stepRepo.IsMultiPart(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.BaseBatch](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.BaseBatch
		if isMultiPart {
			result, apiErr = s.splitMultiPartBatch(ctx, accountID, identity, params, stepRepo, idempotencyKey)
		} else {
			result, apiErr = s.splitSinglePartBatch(ctx, accountID, identity, params, stepRepo, idempotencyKey)
		}
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *batchSvcImpl) splitSinglePartBatch(ctx context.Context, accountID string, identity *types.Identity, params domain.SplitBatchParams, stepRepo domain.ProductionStepQueryRepo, idempotencyKey *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
	if len(params.BatchIDs) != 1 {
		return nil, apierror.NewValidationError("Cannot split multiple batches for a single-part production step.")
	}

	batchRepo := s.repos.NewBatchRepo()
	meds := s.mediators()

	// Find the produced item ID for the production step.
	producedItemID, apiErr := stepRepo.FindProducedItemID(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Find the available source batch.
	sourceBatch, apiErr := batchRepo.FindNextAvailableBatchInFlow(ctx, accountID, params.BatchIDs[0], params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	producedUnit, apiErr := stepRepo.FindProducedUnit(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	createParams := domain.CreateBatchParams{
		AccountID:         accountID,
		ItemID:            producedItemID,
		Quantity:          domain.CreateQuantityParams{Measure: params.Firsts.Measure, UnitID: params.Firsts.Unit.ID},
		ProductionStepID:  params.ProductionStepID,
		ScanningStationID: params.ScanningStationID,
	}
	if params.Seconds != nil && !params.Seconds.Measure.IsZero() {
		createParams.Seconds = &domain.CreateQuantityParams{Measure: params.Seconds.Measure, UnitID: params.Seconds.Unit.ID}
	}
	if params.Waste != nil && !params.Waste.Measure.IsZero() {
		createParams.Waste = &domain.CreateQuantityParams{Measure: params.Waste.Measure, UnitID: params.Waste.Unit.ID}
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		created, apiErr := txBatchRepo.Create(txCtx, newBatchID, createParams)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.ConnectOneToOne(txCtx, accountID, sourceBatch.ID, newBatchID, params.CloseBatch); apiErr != nil {
			return apiErr
		}

		// If not explicitly closing the batch, check if fully used.
		if !params.CloseBatch {
			if apiErr := txBatchRepo.CloseIfFullyUsed(txCtx, accountID, *sourceBatch, *producedUnit, params.ProductionStepID); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, params.ProductionStepID); apiErr != nil {
			return apiErr
		}

		// Enqueue outbox for firsts.
		evt := domain.ExecuteProductionStepEvent{
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
			ItemID:            producedItemID,
			BatchQuantityID:   created.Quantity.ID,
			BatchMeasure:      created.Quantity.Measure.String(),
			BatchUnitID:       created.Quantity.Unit.ID,
			ProducedBatchID:   &newBatchID,
			ProduceInventory:  true,
		}
		if identity.Actor != nil {
			evt.ResponsibleUserID = &identity.Actor.ID
		}
		if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
			return apiErr
		}

		// If seconds or waste exist, enqueue a SECOND outbox message with combined quantity and produceInventory=false.
		secondsWasteTotal := decimal.Zero
		if created.Seconds != nil {
			secondsWasteTotal = secondsWasteTotal.Add(created.Seconds.Measure)
		}
		if created.Waste != nil {
			secondsWasteTotal = secondsWasteTotal.Add(created.Waste.Measure)
		}
		if secondsWasteTotal.GreaterThan(decimal.Zero) {
			evt2 := domain.ExecuteProductionStepEvent{
				ProductionStepID:  params.ProductionStepID,
				ScanningStationID: params.ScanningStationID,
				ItemID:            producedItemID,
				BatchQuantityID:   created.Quantity.ID,
				BatchMeasure:      secondsWasteTotal.String(),
				BatchUnitID:       created.Quantity.Unit.ID,
				ProduceInventory:  false,
			}
			if identity.Actor != nil {
				evt2.ResponsibleUserID = &identity.Actor.ID
			}
			if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt2); apiErr != nil {
				return apiErr
			}
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		result = created
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})

	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) splitMultiPartBatch(ctx context.Context, accountID string, identity *types.Identity, params domain.SplitBatchParams, stepRepo domain.ProductionStepQueryRepo, idempotencyKey *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
	if len(params.BatchIDs) <= 1 {
		return nil, apierror.NewValidationError("Cannot split a single batch for a multi-part production step.")
	}

	batchRepo := s.repos.NewBatchRepo()
	meds := s.mediators()

	// Find all available source batches.
	sourceBatches, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, params.BatchIDs, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(sourceBatches) == 0 {
		return nil, apierror.NewValidationError("No available batches found.")
	}

	// Check for duplicate resolved batches.
	seen := make(map[string]bool, len(sourceBatches))
	for _, b := range sourceBatches {
		if seen[b.ID] {
			return nil, apierror.NewValidationError("Duplicate batches provided.")
		}
		seen[b.ID] = true
	}

	producedItemID, apiErr := stepRepo.FindProducedItemID(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	sourceBatchIDs := make([]string, len(sourceBatches))
	for i, b := range sourceBatches {
		sourceBatchIDs[i] = b.ID
	}

	createParams := domain.CreateBatchParams{
		AccountID:         accountID,
		ItemID:            producedItemID,
		Quantity:          domain.CreateQuantityParams{Measure: params.Firsts.Measure, UnitID: params.Firsts.Unit.ID},
		ProductionStepID:  params.ProductionStepID,
		ScanningStationID: params.ScanningStationID,
	}
	if params.Seconds != nil && !params.Seconds.Measure.IsZero() {
		createParams.Seconds = &domain.CreateQuantityParams{Measure: params.Seconds.Measure, UnitID: params.Seconds.Unit.ID}
	}
	if params.Waste != nil && !params.Waste.Measure.IsZero() {
		createParams.Waste = &domain.CreateQuantityParams{Measure: params.Waste.Measure, UnitID: params.Waste.Unit.ID}
	}

	producedUnit, apiErr := stepRepo.FindProducedUnit(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		created, apiErr := txBatchRepo.Create(txCtx, newBatchID, createParams)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.ConnectManyToOne(txCtx, accountID, sourceBatchIDs, newBatchID, params.CloseBatch); apiErr != nil {
			return apiErr
		}

		// If not explicitly closing, check if fully used (check against first source batch).
		if !params.CloseBatch {
			if apiErr := txBatchRepo.CloseIfFullyUsed(txCtx, accountID, sourceBatches[0], *producedUnit, params.ProductionStepID); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, params.ProductionStepID); apiErr != nil {
			return apiErr
		}

		evt := domain.ExecuteProductionStepEvent{
			ProductionStepID:  params.ProductionStepID,
			ScanningStationID: params.ScanningStationID,
			ItemID:            producedItemID,
			BatchQuantityID:   created.Quantity.ID,
			BatchMeasure:      created.Quantity.Measure.String(),
			BatchUnitID:       created.Quantity.Unit.ID,
			ProducedBatchID:   &newBatchID,
			ProduceInventory:  true,
		}
		if identity.Actor != nil {
			evt.ResponsibleUserID = &identity.Actor.ID
		}
		if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt); apiErr != nil {
			return apiErr
		}

		// Second outbox message for seconds + waste if present.
		secondsWasteTotal := decimal.Zero
		if created.Seconds != nil {
			secondsWasteTotal = secondsWasteTotal.Add(created.Seconds.Measure)
		}
		if created.Waste != nil {
			secondsWasteTotal = secondsWasteTotal.Add(created.Waste.Measure)
		}
		if secondsWasteTotal.GreaterThan(decimal.Zero) {
			evt2 := domain.ExecuteProductionStepEvent{
				ProductionStepID:  params.ProductionStepID,
				ScanningStationID: params.ScanningStationID,
				ItemID:            producedItemID,
				BatchQuantityID:   created.Quantity.ID,
				BatchMeasure:      secondsWasteTotal.String(),
				BatchUnitID:       created.Quantity.Unit.ID,
				ProduceInventory:  false,
			}
			if identity.Actor != nil {
				evt2.ResponsibleUserID = &identity.Actor.ID
			}
			if apiErr := txSvc.enqueueExecuteProductionStep(txCtx, txSvc.repos, evt2); apiErr != nil {
				return apiErr
			}
		}

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		result = created
		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
	})

	if apiErr != nil {
		return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) CloseBatch(ctx context.Context, batchID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.close")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	batchRepo := s.repos.NewBatchRepo()

	old, apiErr := batchRepo.Find(ctx, accountID, batchID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		closed, apiErr := txSvc.repos.NewBatchRepo().Close(txCtx, accountID, batchID)
		if apiErr != nil {
			return apiErr
		}
		result = closed

		changes := audit.ComputeChanges(old, closed)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   old.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) DeleteBatch(ctx context.Context, batchID string) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	batchRepo := s.repos.NewBatchRepo()

	// Find the batch to get production run ID for post-delete handling.
	batch, apiErr := batchRepo.Find(ctx, accountID, batchID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeBatch, batchID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This batch has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := s.undoBatch(ctx, identity, accountID, batch)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *batchSvcImpl) DeleteManyBatches(ctx context.Context, batchIDs []string) *apierror.APIError {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.delete_many")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	batchRepo := s.repos.NewBatchRepo()
	var foundBatches []*domain.Batch
	for _, bid := range batchIDs {
		batch, apiErr := batchRepo.Find(ctx, accountID, bid)
		if apiErr != nil {
			continue // Skip batches that can't be found.
		}
		foundBatches = append(foundBatches, batch)
	}

	if len(foundBatches) == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Batches not found."))
	}

	ordered, apiErr := s.orderForUndo(ctx, foundBatches)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	for _, batch := range ordered {
		if _, apiErr := s.undoBatch(ctx, identity, accountID, batch); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

// orderForUndo sorts a selection so a batch is undone before the batches that fed it. A batch
// something downstream still consumes cannot be undone, so a selection holding a whole chain has to
// be worked from its end backwards.
func (s *batchSvcImpl) orderForUndo(ctx context.Context, batches []*domain.Batch) ([]*domain.Batch, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()

	byID := make(map[string]*domain.Batch, len(batches))
	for _, batch := range batches {
		byID[batch.ID] = batch
	}

	// Only the edges inside the selection matter: an input outside it is not being undone. The map
	// is built the consuming way round — who feeds on this batch — because that is the order the
	// undo has to respect.
	consumers := make(map[string][]string, len(batches))
	for _, batch := range batches {
		inputIDs, apiErr := batchRepo.FindInputBatchIDs(ctx, batch.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, inputID := range inputIDs {
			if _, selected := byID[inputID]; selected {
				consumers[inputID] = append(consumers[inputID], batch.ID)
			}
		}
	}

	var ordered []*domain.Batch
	visited := make(map[string]bool, len(batches))
	var visit func(id string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		for _, consumerID := range consumers[id] {
			visit(consumerID)
		}
		if batch, ok := byID[id]; ok {
			ordered = append(ordered, batch)
		}
	}
	for _, batch := range batches {
		visit(batch.ID)
	}

	return ordered, nil
}

// undoBatch undoes the scan that produced a batch: the batches it consumed are released, the run it
// belongs to reopens, and an outbox message goes out to reverse the inventory the scan recorded.
//
// A batch created by a scan (move, merge, split) is deleted outright. A batch that a production run
// created and an init station merely stamped keeps its row — deleting it would quietly take a unit of
// work off the run instead of putting it back in the queue.
//
// The ledger reversal is deliberately not part of this: it fans out across receipts, issues,
// allocations and reservations, and holding a scanning station's request open for it is the reason
// the forward path already runs behind a message too.
func (s *batchSvcImpl) undoBatch(ctx context.Context, identity *types.Identity, accountID string, batch *domain.Batch) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.undo")
	defer span.End()

	batchRepo := s.repos.NewBatchRepo()

	downstream, apiErr := batchRepo.CountDownstreamBatches(ctx, batch.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if downstream > 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("This batch has already been used by a later scan. Delete that batch first."))
	}

	allocated, apiErr := s.repos.NewInventoryMutationRepo().CountAllocatedReceiptsForBatch(ctx, accountID, batch.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if allocated > 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("Inventory produced by this batch has already been used and cannot be reversed."))
	}

	// An unscanned batch never moved inventory or joined the flow: it is a planned unit of work, and
	// deleting it is just a delete.
	if batch.ScannedAt == nil {
		return s.deleteBatchRow(ctx, identity, accountID, batch)
	}

	inputBatchIDs, apiErr := batchRepo.FindInputBatchIDs(ctx, batch.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	undoEvent, apiErr := s.planScanUndo(ctx, identity, accountID, batch)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The row is deleted only when a scan created it; an init-station batch belongs to its run.
	isInitScan := len(inputBatchIDs) == 0 && batch.ProductionRun != nil

	var producedUnit *domain.LightUnit
	if len(inputBatchIDs) > 0 && batch.ProductionStep != nil {
		producedUnit, apiErr = s.repos.NewProductionStepQueryRepo().FindProducedUnit(ctx, accountID, batch.ProductionStep.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		if isInitScan {
			unscanned, apiErr := txBatchRepo.Unscan(txCtx, accountID, batch.ID)
			if apiErr != nil {
				return apiErr
			}
			result = unscanned

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeBatch,
				ResourceID:   batch.ID,
				Changes:      audit.ComputeChanges(batch, unscanned),
			}); apiErr != nil {
				return apiErr
			}
		} else {
			if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeBatch, batch.ID, batch); apiErr != nil {
				return apiErr
			}

			deleted, apiErr := txBatchRepo.Delete(txCtx, accountID, batch.ID)
			if apiErr != nil {
				return apiErr
			}
			result = deleted

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionDelete,
				ResourceType: constants.ObjectTypeBatch,
				ResourceID:   batch.ID,
				Changes:      audit.ComputeChanges(batch, (*domain.Batch)(nil)),
			}); apiErr != nil {
				return apiErr
			}
		}

		// With the batch gone its inputs are no longer spoken for, so any that were closed only
		// because this batch consumed them open back up.
		if producedUnit != nil {
			for _, inputBatchID := range inputBatchIDs {
				inputBatch, apiErr := txBatchRepo.Find(txCtx, accountID, inputBatchID)
				if apiErr != nil {
					continue
				}
				if apiErr := txBatchRepo.ReopenIfNotFullyUsed(txCtx, accountID, domain.BaseBatch{
					ID:       inputBatch.ID,
					Item:     inputBatch.Item,
					Quantity: inputBatch.Quantity,
				}, *producedUnit, batch.ProductionStep.ID); apiErr != nil {
					return apiErr
				}
			}
		}

		if batch.ProductionRun != nil {
			if apiErr := txSvc.repos.NewProductionRunQueryRepo().Reopen(txCtx, accountID, batch.ProductionRun.ID); apiErr != nil {
				return apiErr
			}
		}

		return txSvc.enqueueUndoBatchScan(txCtx, txSvc.repos, undoEvent)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// planScanUndo snapshots what the reversal will need once the batch is gone. The ledger rows keep
// pointing at the batch after it is deleted, but the flow edges do not, so the scrap the scan
// released reservations for has to be read while the lineage still exists.
func (s *batchSvcImpl) planScanUndo(ctx context.Context, identity *types.Identity, accountID string, batch *domain.Batch) (domain.UndoBatchScanEvent, *apierror.APIError) {
	evt := domain.UndoBatchScanEvent{BatchID: batch.ID}
	if batch.ScanningStation != nil {
		evt.ScanningStationID = batch.ScanningStation.ID
	}
	if identity.Actor != nil {
		evt.ResponsibleUserID = identity.Actor.ID
	}

	if batch.ProductionStep == nil {
		return evt, nil
	}

	lineage, apiErr := s.repos.NewBatchRepo().FindLineageShortfall(ctx, batch.ID)
	if apiErr != nil {
		return evt, apiErr
	}
	shortfall := lineage.Total()
	if lineage.ProductionRunID == "" || shortfall.LessThanOrEqual(decimal.Zero) {
		return evt, nil
	}

	orderID, apiErr := s.repos.NewOrderQueryRepo().FindIDByProductionRun(ctx, accountID, lineage.ProductionRunID)
	if apiErr != nil {
		return evt, apiErr
	}
	if orderID == nil {
		return evt, nil
	}

	step, apiErr := s.repos.NewProductionStepQueryRepo().Find(ctx, accountID, batch.ProductionStep.ID)
	if apiErr != nil {
		return evt, apiErr
	}

	evt.OrderID = *orderID
	evt.ProducedItemID = step.Production.ProducedItem.ID
	evt.ShortfallMeasure = shortfall.String()
	evt.ShortfallUnitID = step.Production.Quantity.Unit.ID

	return evt, nil
}

// deleteBatchRow removes a batch that was never scanned. Nothing to unwind: it is a ticket the floor
// never ran.
func (s *batchSvcImpl) deleteBatchRow(ctx context.Context, _ *types.Identity, accountID string, batch *domain.Batch) (*domain.BaseBatch, *apierror.APIError) {
	var deleted *domain.BaseBatch

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeBatch, batch.ID, batch); apiErr != nil {
			return apiErr
		}

		var apiErr *apierror.APIError
		deleted, apiErr = txSvc.repos.NewBatchRepo().Delete(txCtx, accountID, batch.ID)
		if apiErr != nil {
			return apiErr
		}

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   batch.ID,
			Changes:      audit.ComputeChanges(batch, (*domain.Batch)(nil)),
		})
	})
	if apiErr != nil {
		return nil, apiErr
	}

	if batch.ProductionRun != nil {
		_ = s.repos.NewProductionRunQueryRepo().CloseIfAllBatchesScannedOrDeleted(ctx, accountID, batch.ProductionRun.ID)
	}

	return deleted, nil
}

// enqueueUndoBatchScan writes an outbox message for the ledger reversal that follows a deleted scan.
func (s *batchSvcImpl) enqueueUndoBatchScan(ctx context.Context, repos domain.RepoFactory, evt domain.UndoBatchScanEvent) *apierror.APIError {
	payload, err := json.Marshal(evt)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal undo batch scan event.")
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
		MessageType: string(contracts.CoreCmdUndoBatchScan),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdUndoBatchScan),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for undo batch scan.")
	}

	return nil
}
