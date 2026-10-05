package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var batchSvcTracer = tracing.GetTracer("core-service.batch_service")

type batchSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	billingPub      domain.BillingPublisher
}

type BatchSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// BillingPub (optional) meters the batches scans create for usage billing. Nil skips metering.
	BillingPub domain.BillingPublisher
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
		billingPub:      config.BillingPub,
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
			billingPub:      s.billingPub,
		}
		return fn(txCtx, txSvc)
	})
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

			if apiErr := txSvc.mediators().ProductionRunActivity.NotifyBatchDeleted(txCtx, identity, accountID, batch); apiErr != nil {
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
func (s *batchSvcImpl) deleteBatchRow(ctx context.Context, identity *types.Identity, accountID string, batch *domain.Batch) (*domain.BaseBatch, *apierror.APIError) {
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

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   batch.ID,
			Changes:      audit.ComputeChanges(batch, (*domain.Batch)(nil)),
		}); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().ProductionRunActivity.NotifyBatchDeleted(txCtx, identity, accountID, batch)
	})
	if apiErr != nil {
		return nil, apiErr
	}

	if batch.ProductionRun != nil {
		if apiErr := s.repos.NewProductionRunQueryRepo().CloseIfAllBatchesScannedOrDeleted(ctx, accountID, batch.ProductionRun.ID); apiErr != nil {
			slog.ErrorContext(ctx, "Failed to close production run after batch delete", "error", apiErr, "production_run_id", batch.ProductionRun.ID, "batch_id", batch.ID)
		}
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
