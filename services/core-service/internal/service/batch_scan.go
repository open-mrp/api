package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"slices"
	"strconv"
	"time"

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

// The operations a scanning station performs on batches.
//
// Each is a port of the dashboard's scanning service, which the floor has run on: the checks, the
// order they run in, their messages, and the quantity math are its. What is Go's own sits around them —
// idempotency, the audit trail, and every write a scan owes (the batch, the batch_scanned event that
// moves its inventory, the usage meter) committing in one transaction.

// baseUnitIDByType is the unit a quantity of each type is totalled in when its terms share no unit.
var baseUnitIDByType = map[string]string{
	string(constants.UnitTypeCurrency):    "dollar",
	string(constants.UnitTypeMass):        "gram",
	string(constants.UnitTypeQuantity):    "each",
	string(constants.UnitTypeTime):        "hour",
	string(constants.UnitTypeVolume):      "liter",
	string(constants.UnitTypeLength):      "meter",
	string(constants.UnitTypeArea):        "meters_squared",
	string(constants.UnitTypeTemperature): "celcius",
}

// batchActor resolves the internal user a batch operation runs as and checks the batches permission.
func batchActor(ctx context.Context, action types.Action) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainBatches, action); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// idempotentScan runs a scan once per idempotency key: a repeat of a finished request returns what it
// returned the first time, and an error worth keeping is cached in its place.
func (s *batchSvcImpl) idempotentScan(ctx context.Context, identity *types.Identity, run func(key *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError)) (*domain.BaseBatch, *apierror.APIError) {
	meds := s.mediators()

	key, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(key.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.BaseBatch](ctx, key.ResponseCode, key.ResponseBody)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Issue unmarshalling cached response.")
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := run(key)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, key.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, apierror.NewInvariantViolationError("Unexpected recovery point: " + key.RecoveryPoint)
	}
}

// heldBatch is what a batch row read while holding it says: whether it still exists, and when it was scanned.
type heldBatch struct {
	exists    bool
	scannedAt *time.Time
}

// holdBatches holds the rows of the batches a scan or an undo works on until its transaction ends, and
// must be the transaction's first read. Two stations can scan one batch at once, and an undo can land
// while a scan of the same batch is under way; held rows make the later one wait and then see what the
// earlier one committed, as if it had come after it. The rows are taken in id order, so operations over
// overlapping batches queue behind each other rather than deadlock.
func holdBatches(ctx context.Context, repo domain.BatchRepo, accountID string, idSets ...[]string) (map[string]heldBatch, *apierror.APIError) {
	held := make(map[string]heldBatch)
	var ids []string
	for _, set := range idSets {
		for _, id := range set {
			if _, seen := held[id]; !seen {
				held[id] = heldBatch{}
				ids = append(ids, id)
			}
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		scannedAt, exists, apiErr := repo.LockScan(ctx, accountID, id)
		if apiErr != nil {
			return nil, apiErr
		}
		held[id] = heldBatch{exists: exists, scannedAt: scannedAt}
	}
	return held, nil
}

// ---------------------------------------------------------------------------
// Initialize
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) InitializeBatch(ctx context.Context, params domain.InitializeBatchParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.initialize")
	defer span.End()

	identity, apiErr := batchActor(ctx, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := s.idempotentScan(ctx, identity, func(key *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
		return s.initializeBatch(ctx, identity, accountID, params, key)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

func (s *batchSvcImpl) initializeBatch(ctx context.Context, identity *types.Identity, accountID string, params domain.InitializeBatchParams, key *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
	stationType, apiErr := s.repos.NewScanningStationQueryRepo().FindType(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Scanning Station not found.")
		}
		return nil, apiErr
	}
	if stationType == "" {
		return nil, apierror.NewResourceNotFoundError("Scanning Station not found.")
	}

	batchRepo := s.repos.NewBatchRepo()
	batch, apiErr := batchRepo.Find(ctx, accountID, params.BatchID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Batch not found.")
		}
		return nil, apiErr
	}
	if apiErr := checkInitializable(batch); apiErr != nil {
		return nil, apiErr
	}

	init := string(constants.ScanningStationTypeInitBatch)
	isOverriddenInit := params.TypeOverride != nil && *params.TypeOverride == init && stationType != init

	// Scanning as another station type, or recording a scan without its materials, is a supervisor's
	// correction rather than an operator's scan.
	if params.TypeOverride != nil && *params.TypeOverride != stationType {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionUpdate); apiErr != nil {
			return nil, apiErr
		}
	}
	if params.ConsumeMaterials != nil && !*params.ConsumeMaterials {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionUpdate); apiErr != nil {
			return nil, apiErr
		}
	}

	if batch.ProductionRun == nil {
		return nil, apierror.NewResourceNotFoundError("Production run not found.")
	}

	productionStepID, apiErr := s.resolveInitStep(ctx, accountID, params, batch, isOverriddenInit)
	if apiErr != nil {
		return nil, apiErr
	}

	if apiErr := enforceBatchScansPerPeriodLimit(ctx, s.repos, accountID); apiErr != nil {
		return nil, apiErr
	}

	// Materials are what the step's execution is for, so declining to consume them means there is no
	// scan to hand off — only a batch stamped as scanned.
	consume := params.ConsumeMaterials == nil || *params.ConsumeMaterials

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		if _, apiErr := holdBatches(txCtx, txBatchRepo, accountID, []string{batch.ID}); apiErr != nil {
			return apiErr
		}
		held, apiErr := txBatchRepo.Find(txCtx, accountID, batch.ID)
		if apiErr != nil {
			if apierror.IsNotFound(apiErr) {
				return apierror.NewResourceNotFoundError("Batch not found.")
			}
			return apiErr
		}
		if apiErr := checkInitializable(held); apiErr != nil {
			return apiErr
		}

		if apiErr := txBatchRepo.MarkAsScanned(txCtx, accountID, batch.ID); apiErr != nil {
			return apiErr
		}
		if apiErr := txBatchRepo.ConnectProductionStep(txCtx, accountID, batch.ID, productionStepID); apiErr != nil {
			return apiErr
		}
		if apiErr := txBatchRepo.ConnectScanningStation(txCtx, accountID, batch.ID, params.ScanningStationID); apiErr != nil {
			return apiErr
		}
		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, batch.ID, productionStepID); apiErr != nil {
			return apiErr
		}

		// Read back so the event carries the stamp the row was just given: an undo clears it and leaves
		// the batch scannable again, so the stamp is what tells this scan from a later one of the row.
		updated, apiErr := txBatchRepo.Find(txCtx, accountID, batch.ID)
		if apiErr != nil {
			return apiErr
		}
		result = baseBatchOf(updated)

		if consume {
			if apiErr := publishBatchScanned(txCtx, txSvc.repos, identity, accountID, productionStepID, params.ScanningStationID, result); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   result.ID,
			Changes:      audit.ComputeChanges(baseBatchOf(batch), result),
		}); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
	})
	if apiErr != nil {
		return nil, apiErr
	}

	// After the transaction commits, so the close check sees every other scan that has committed;
	// inside it, two last scans racing would each miss the other and neither would close the run.
	runRepo := s.repos.NewProductionRunQueryRepo()
	runID := batch.ProductionRun.ID
	if apiErr := runRepo.Start(ctx, accountID, runID); apiErr != nil {
		slog.ErrorContext(ctx, "Failed to start production run after batch scan", "error", apiErr, "production_run_id", runID, "batch_id", batch.ID)
	}
	if apiErr := runRepo.CloseIfAllBatchesScannedOrDeleted(ctx, accountID, runID); apiErr != nil {
		slog.ErrorContext(ctx, "Failed to close production run after batch scan", "error", apiErr, "production_run_id", runID, "batch_id", batch.ID)
	}

	return result, nil
}

// checkInitializable refuses a batch an init station cannot stamp.
func checkInitializable(batch *domain.Batch) *apierror.APIError {
	switch {
	case batch.ClosedAt != nil:
		return apierror.NewValidationError("This batch is closed.")
	case batch.ScannedAt != nil:
		return apierror.NewValidationError("This batch has been scanned already.")
	}
	return nil
}

// resolveInitStep picks the step a batch is initialized into. A step the operator chose must be one
// this station runs for the batch's item. Scanning as an init station when the station is not one has
// no entry step to fall back on, so the station's steps for the item must settle it alone. Otherwise it
// is the station's entry step for the item: one with nothing upstream.
func (s *batchSvcImpl) resolveInitStep(ctx context.Context, accountID string, params domain.InitializeBatchParams, batch *domain.Batch, isOverriddenInit bool) (string, *apierror.APIError) {
	if (params.ProductionStepID != nil && *params.ProductionStepID != "") || isOverriddenInit {
		candidates, apiErr := s.repos.NewBatchRepo().FindPossibleInitSteps(ctx, accountID, params.ScanningStationID, batch.ID)
		if apiErr != nil {
			return "", apiErr
		}

		if params.ProductionStepID != nil && *params.ProductionStepID != "" {
			for _, candidate := range candidates {
				if candidate.ID == *params.ProductionStepID {
					return candidate.ID, nil
				}
			}
			return "", apierror.NewValidationError("Selected production step is not valid for this scanning station and part.")
		}

		switch len(candidates) {
		case 0:
			return "", apierror.NewResourceNotFoundError("Production step not found.")
		case 1:
			return candidates[0].ID, nil
		default:
			return "", apierror.NewValidationError("Multiple production steps match. Please select one.")
		}
	}

	stepID, apiErr := s.repos.NewProductionStepQueryRepo().FindIDByScanningStationAndProducedBlock(ctx, accountID, params.ScanningStationID, batch.Item.ID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return "", apierror.NewResourceNotFoundError("Production step not found.")
		}
		return "", apiErr
	}
	return stepID, nil
}

// enforceBatchScansPerPeriodLimit rejects a scan that would exceed the account plan's per-billing-period
// batch cap. Sandboxes, accounts on no plan, and plans with no cap are exempt.
func enforceBatchScansPerPeriodLimit(ctx context.Context, repos domain.RepoFactory, accountID string) *apierror.APIError {
	max, start, apiErr := resolveAccountPlanLimit(ctx, repos, accountID, constants.AccountPlanLimitBatchesMaximum)
	if apiErr != nil || max == nil {
		return apiErr
	}

	count, apiErr := repos.NewBatchRepo().CountScannedSince(ctx, accountID, start)
	if apiErr != nil {
		return apiErr
	}
	if count >= int64(*max) {
		plural := "s"
		if *max == 1 {
			plural = ""
		}
		return apierror.NewValidationError(fmt.Sprintf("Your plan allows a maximum of %d batch scan%s per billing period", *max, plural))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Move
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) MoveBatches(ctx context.Context, params domain.MoveBatchesParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.move")
	defer span.End()

	identity, apiErr := batchActor(ctx, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	if apiErr := s.checkStationInAccount(ctx, accountID, params.ScanningStationID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	stepRepo := s.repos.NewProductionStepQueryRepo()
	stepInAccount, apiErr := stepRepo.IsInAccount(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !stepInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}
	isMultiPart, apiErr := stepRepo.IsMultiPart(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(params.BatchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("No batches to move."))
	}

	result, apiErr := s.idempotentScan(ctx, identity, func(key *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
		return s.planAndWriteScan(ctx, identity, accountID, key, params.BatchIDs, func(ctx context.Context, svc *batchSvcImpl) (*scanOutput, *apierror.APIError) {
			if isMultiPart {
				return svc.planMultiPartMove(ctx, accountID, params)
			}
			if len(params.BatchIDs) != 1 {
				return nil, apierror.NewValidationError("Single-part production step can only accept one batch at a time.")
			}
			return svc.planSinglePartMove(ctx, accountID, params)
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

func (s *batchSvcImpl) planSinglePartMove(ctx context.Context, accountID string, params domain.MoveBatchesParams) (*scanOutput, *apierror.APIError) {
	source, apiErr := s.repos.NewBatchRepo().FindNextAvailableBatchInFlow(ctx, accountID, params.BatchIDs[0], params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if source.ClosedAt != nil {
		return nil, apierror.NewValidationError("Batch is closed.")
	}

	next, apiErr := s.repos.NewProductionStepQueryRepo().CalculateNextStepQuantities(ctx, accountID, source.Item.ID, source.Quantity, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &scanOutput{
		productionStepID:  params.ProductionStepID,
		scanningStationID: params.ScanningStationID,
		itemID:            next.ItemID,
		quantity:          domain.CreateQuantityParams{Measure: next.Quantity, UnitID: next.ProducedUnitID},
		sourceIDs:         []string{source.ID},
		closeSources:      true,
	}, nil
}

// planMultiPartMove brings one batch of each part together. Every batch is stepped through on its own
// and all of them must make the same amount: parts that do not match are not one assembly.
func (s *batchSvcImpl) planMultiPartMove(ctx context.Context, accountID string, params domain.MoveBatchesParams) (*scanOutput, *apierror.APIError) {
	sources, apiErr := s.findScanSources(ctx, accountID, params.BatchIDs, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	step, apiErr := s.findScanStep(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := checkRequiredParts(step, sources); apiErr != nil {
		return nil, apiErr
	}

	stepRepo := s.repos.NewProductionStepQueryRepo()
	var next *domain.NextStepQuantitiesResult
	measures := make([]decimal.Decimal, 0, len(sources))
	for _, source := range sources {
		if source.ClosedAt != nil {
			return nil, apierror.NewValidationError("Batch is closed.")
		}
		next, apiErr = stepRepo.CalculateNextStepQuantities(ctx, accountID, source.Item.ID, source.Quantity, params.ProductionStepID)
		if apiErr != nil {
			return nil, apiErr
		}
		measures = append(measures, next.Quantity)
	}
	if apiErr := checkSameMeasure(measures); apiErr != nil {
		return nil, apiErr
	}
	if measures[0].IsZero() {
		return nil, apierror.NewResourceNotFoundError("Calculated measure not found.")
	}

	return &scanOutput{
		productionStepID:  params.ProductionStepID,
		scanningStationID: params.ScanningStationID,
		itemID:            next.ItemID,
		quantity:          domain.CreateQuantityParams{Measure: measures[0], UnitID: next.ProducedUnitID},
		sourceIDs:         batchIDsOf(sources),
		closeSources:      true,
	}, nil
}

// ---------------------------------------------------------------------------
// Merge
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) MergeBatches(ctx context.Context, params domain.MergeBatchesParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.merge")
	defer span.End()

	identity, apiErr := batchActor(ctx, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	if apiErr := s.checkStationInAccount(ctx, accountID, params.ScanningStationID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	isMultiPart, apiErr := s.repos.NewProductionStepQueryRepo().IsMultiPart(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(params.BatchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("No batches to merge."))
	}

	result, apiErr := s.idempotentScan(ctx, identity, func(key *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
		return s.planAndWriteScan(ctx, identity, accountID, key, params.BatchIDs, func(ctx context.Context, svc *batchSvcImpl) (*scanOutput, *apierror.APIError) {
			sources, apiErr := svc.findScanSources(ctx, accountID, params.BatchIDs, params.ProductionStepID)
			if apiErr != nil {
				return nil, apiErr
			}
			if isMultiPart {
				return svc.planMultiPartMerge(ctx, accountID, params, sources)
			}
			return svc.planSinglePartMerge(ctx, accountID, params, sources)
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// planSinglePartMerge combines batches of one part into one, totalled in the part's base unit and
// stepped through as a whole.
func (s *batchSvcImpl) planSinglePartMerge(ctx context.Context, accountID string, params domain.MergeBatchesParams, sources []domain.BaseBatch) (*scanOutput, *apierror.APIError) {
	itemID := sources[0].Item.ID
	for _, source := range sources {
		if source.Item.ID != itemID {
			return nil, apierror.NewValidationError("All batches must be of the same type for single part merging")
		}
	}

	unitGroup, apiErr := s.findItemUnitGroup(ctx, accountID, itemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if unitGroup == nil {
		return nil, apierror.NewResourceNotFoundError("Unit group not found.")
	}

	total, apiErr := s.totalInUnit(ctx, sources, unitGroup.BaseUnit)
	if apiErr != nil {
		return nil, apiErr
	}

	next, apiErr := s.repos.NewProductionStepQueryRepo().CalculateNextStepQuantities(ctx, accountID, itemID, total, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &scanOutput{
		productionStepID:  params.ProductionStepID,
		scanningStationID: params.ScanningStationID,
		itemID:            next.ItemID,
		quantity:          domain.CreateQuantityParams{Measure: next.Quantity, UnitID: next.ProducedUnitID},
		sourceIDs:         batchIDsOf(sources),
		closeSources:      true,
	}, nil
}

// planMultiPartMerge combines batches of several parts: each part's batches are totalled in its base
// unit and stepped through together, and every part must come out at the same amount.
func (s *batchSvcImpl) planMultiPartMerge(ctx context.Context, accountID string, params domain.MergeBatchesParams, sources []domain.BaseBatch) (*scanOutput, *apierror.APIError) {
	step, apiErr := s.findScanStep(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	groups, apiErr := s.groupByPart(ctx, accountID, sources)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := checkRequiredParts(step, sources); apiErr != nil {
		return nil, apiErr
	}

	next, measures, apiErr := s.stepGroupsThrough(ctx, accountID, groups, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := checkSameMeasure(measures); apiErr != nil {
		return nil, apiErr
	}
	if measures[0].IsZero() {
		return nil, apierror.NewResourceNotFoundError("Output quantity not found.")
	}

	return &scanOutput{
		productionStepID:  params.ProductionStepID,
		scanningStationID: params.ScanningStationID,
		itemID:            next.ItemID,
		quantity:          domain.CreateQuantityParams{Measure: measures[0], UnitID: next.ProducedUnitID},
		sourceIDs:         batchIDsOf(sources),
		closeSources:      true,
	}, nil
}

// ---------------------------------------------------------------------------
// Split
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) SplitBatch(ctx context.Context, params domain.SplitBatchParams) (*domain.BaseBatch, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.split")
	defer span.End()

	identity, apiErr := batchActor(ctx, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	if apiErr := s.checkStationInAccount(ctx, accountID, params.ScanningStationID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	isMultiPart, apiErr := s.repos.NewProductionStepQueryRepo().IsMultiPart(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(params.BatchIDs) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationError("No batches to split."))
	}

	result, apiErr := s.idempotentScan(ctx, identity, func(key *domain.IdempotencyKey) (*domain.BaseBatch, *apierror.APIError) {
		return s.planAndWriteScan(ctx, identity, accountID, key, params.BatchIDs, func(ctx context.Context, svc *batchSvcImpl) (*scanOutput, *apierror.APIError) {
			return svc.planSplit(ctx, accountID, params, isMultiPart)
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// planSplit takes part of what a batch makes at a step — firsts, and the seconds and waste that came
// with them — leaving the source open until what has been split off accounts for all of it, unless the
// operator closes it now.
func (s *batchSvcImpl) planSplit(ctx context.Context, accountID string, params domain.SplitBatchParams, isMultiPart bool) (*scanOutput, *apierror.APIError) {
	sources, apiErr := s.findScanSources(ctx, accountID, params.BatchIDs, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	if params.Firsts.Measure.IsZero() &&
		(params.Seconds == nil || params.Seconds.Measure.IsZero()) &&
		(params.Waste == nil || params.Waste.Measure.IsZero()) {
		return nil, apierror.NewValidationError("No quantity to split.")
	}

	if isMultiPart && len(sources) <= 1 {
		return nil, apierror.NewValidationError("Cannot split a single batch.")
	}
	if !isMultiPart && len(sources) != 1 {
		return nil, apierror.NewValidationError("Cannot split multiple batches.")
	}

	itemID, apiErr := s.repos.NewProductionStepQueryRepo().FindProducedItemID(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	// The split is counted in the unit the operator entered firsts in, so that is the unit the source is
	// measured against when deciding whether anything is left of it.
	firstsUnit, apiErr := s.findScanUnit(ctx, accountID, params.Firsts.Unit.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	out := &scanOutput{
		productionStepID:  params.ProductionStepID,
		scanningStationID: params.ScanningStationID,
		itemID:            itemID,
		quantity:          domain.CreateQuantityParams{Measure: params.Firsts.Measure, UnitID: firstsUnit.ID},
		closeSources:      params.CloseBatch,
	}
	if params.Seconds != nil {
		unit, apiErr := s.findScanUnit(ctx, accountID, params.Seconds.Unit.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		out.seconds = &domain.CreateQuantityParams{Measure: params.Seconds.Measure, UnitID: unit.ID}
	}
	if params.Waste != nil {
		unit, apiErr := s.findScanUnit(ctx, accountID, params.Waste.Unit.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		out.waste = &domain.CreateQuantityParams{Measure: params.Waste.Measure, UnitID: unit.ID}
	}

	if isMultiPart {
		out.sourceIDs = batchIDsOf(sources)
	} else {
		out.sourceIDs = []string{sources[0].ID}
	}
	if !params.CloseBatch {
		// A multi-part split is measured against its first part alone; every part makes the same amount.
		out.closeWhenFullyUsed = &sources[0]
		out.closeWhenFullyUsedUnit = *firstsUnit
	}

	return out, nil
}

// ---------------------------------------------------------------------------
// Remaining to split
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) GetRemainingQuantityToSplit(ctx context.Context, batchIDs []string, productionStepID string) (*domain.BatchQuantity, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_remaining_quantity_to_split")
	defer span.End()

	identity, apiErr := batchActor(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	var result *domain.BatchQuantity
	switch len(batchIDs) {
	case 0:
		apiErr = apierror.NewValidationError("No batches to split.")
	case 1:
		result, apiErr = s.remainingToSplitOne(ctx, accountID, batchIDs[0], productionStepID)
	default:
		result, apiErr = s.remainingToSplitMany(ctx, accountID, batchIDs, productionStepID)
	}
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Not a stored quantity, but one the response can still be addressed by, so its unit can be
	// included like any other.
	result.ID, apiErr = id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// remainingToSplitOne is what a batch has left to split at a step, in the base unit of what the step
// makes.
func (s *batchSvcImpl) remainingToSplitOne(ctx context.Context, accountID, batchID, productionStepID string) (*domain.BatchQuantity, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()

	source, apiErr := batchRepo.FindNextAvailableBatchInFlow(ctx, accountID, batchID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	producedUnit, apiErr := s.repos.NewProductionStepQueryRepo().FindProducedUnit(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	remaining, apiErr := batchRepo.RemainingToSplit(ctx, accountID, *source, *producedUnit, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.BatchQuantity{Measure: remaining, Unit: *producedUnit}, nil
}

// remainingToSplitMany is what a set of parts has left to split at a step, in the step's production
// unit: what they make together less the firsts already split off any of them.
func (s *batchSvcImpl) remainingToSplitMany(ctx context.Context, accountID string, batchIDs []string, productionStepID string) (*domain.BatchQuantity, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()

	sources, apiErr := batchRepo.FindAvailableBatchesInFlow(ctx, accountID, batchIDs, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	step, apiErr := s.findScanStep(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	groups, apiErr := s.groupByPart(ctx, accountID, sources)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := checkRequiredParts(step, sources); apiErr != nil {
		return nil, apiErr
	}

	next, measures, apiErr := s.stepGroupsThrough(ctx, accountID, groups, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := checkSameMeasure(measures); apiErr != nil {
		return nil, apiErr
	}
	if next == nil {
		return nil, apierror.NewResourceNotFoundError("Produced block or unit not found.")
	}

	seen := make(map[string]bool)
	var outputs []domain.BaseBatch
	for _, source := range sources {
		batchOutputs, apiErr := batchRepo.FindOutputBatches(ctx, accountID, source.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, output := range batchOutputs {
			if !seen[output.ID] {
				seen[output.ID] = true
				outputs = append(outputs, output)
			}
		}
	}

	producedUnit, apiErr := s.findScanUnit(ctx, accountID, next.ProducedUnitID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Produced unit not found.")
		}
		return nil, apiErr
	}

	// Only firsts: seconds and waste split off one part are not split off the assembly.
	used := decimal.Zero
	conv := s.repos.NewUnitConversionRepo()
	for _, output := range outputs {
		measure, apiErr := domain.ConvertScanQuantity(ctx, conv, output.Quantity.Measure, output.Quantity.Unit, *producedUnit)
		if apiErr != nil {
			return nil, apiErr
		}
		used = used.Add(measure)
	}

	if measures[0].IsZero() {
		return nil, apierror.NewResourceNotFoundError("Quantity to split not found.")
	}

	return &domain.BatchQuantity{Measure: measures[0].Sub(used), Unit: *producedUnit}, nil
}

// ---------------------------------------------------------------------------
// Consumption preview
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) GetScanningStationConsumption(ctx context.Context, params domain.GetConsumptionParams) ([]domain.ScanningConsumption, *apierror.APIError) {
	ctx, span := batchSvcTracer.Start(ctx, "service.batch.get_scanning_station_consumption")
	defer span.End()

	identity, apiErr := batchActor(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := s.scanningStationConsumption(ctx, identity, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

func (s *batchSvcImpl) scanningStationConsumption(ctx context.Context, identity *types.Identity, params domain.GetConsumptionParams) ([]domain.ScanningConsumption, *apierror.APIError) {
	accountID := identity.Target.AccountID

	if len(params.BatchIDs) == 0 {
		return nil, apierror.NewValidationError("No batches provided.")
	}

	if apiErr := s.checkStationInAccount(ctx, accountID, params.ScanningStationID); apiErr != nil {
		return nil, apiErr
	}
	stationType, apiErr := s.repos.NewScanningStationQueryRepo().FindType(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Scanning Station type not found.")
		}
		return nil, apiErr
	}
	if stationType == "" {
		return nil, apierror.NewResourceNotFoundError("Scanning Station type not found.")
	}
	if !isScanningStationType(stationType) {
		return nil, apierror.NewValidationError("Invalid scanning station type.")
	}

	effectiveType := stationType
	if params.TypeOverride != nil && *params.TypeOverride != stationType {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionUpdate); apiErr != nil {
			return nil, apiErr
		}
		if !isScanningStationType(*params.TypeOverride) {
			return nil, apierror.NewValidationError("Invalid scanning station type.")
		}
		effectiveType = *params.TypeOverride
	}

	if effectiveType == string(constants.ScanningStationTypeInitBatch) {
		return s.initConsumption(ctx, accountID, params)
	}

	if params.ProductionStepID == nil || *params.ProductionStepID == "" {
		return nil, apierror.NewResourceNotFoundError("Production step not found.")
	}
	stepID := *params.ProductionStepID

	isMultiPart, apiErr := s.repos.NewProductionStepQueryRepo().IsMultiPart(ctx, accountID, stepID)
	if apiErr != nil {
		return nil, apiErr
	}

	if isMultiPart || effectiveType == string(constants.ScanningStationTypeMergeBatch) || len(params.BatchIDs) > 1 {
		return s.manyBatchConsumption(ctx, accountID, params.BatchIDs, stepID)
	}

	switch effectiveType {
	case string(constants.ScanningStationTypeMoveBatch):
		return s.moveConsumption(ctx, accountID, params.BatchIDs[0], stepID)
	case string(constants.ScanningStationTypeSplitBatch):
		return s.splitConsumption(ctx, accountID, params.BatchIDs[0], stepID, params.SplitQuantity)
	default:
		return nil, apierror.NewValidationError("Invalid scanning station type.")
	}
}

// initConsumption is what initializing a batch would draw: the entry step's materials for the batch's
// own quantity. A step the operator chose among several is the one previewed, when it is one the
// station runs for the batch's item.
func (s *batchSvcImpl) initConsumption(ctx context.Context, accountID string, params domain.GetConsumptionParams) ([]domain.ScanningConsumption, *apierror.APIError) {
	batch, apiErr := s.repos.NewBatchRepo().Find(ctx, accountID, params.BatchIDs[0])
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Batch not found.")
		}
		return nil, apiErr
	}

	stepRepo := s.repos.NewProductionStepQueryRepo()
	var step *domain.ProductionStepDetail
	if params.ProductionStepID != nil && *params.ProductionStepID != "" {
		candidates, apiErr := s.repos.NewBatchRepo().FindPossibleInitSteps(ctx, accountID, params.ScanningStationID, batch.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, candidate := range candidates {
			if candidate.ID == *params.ProductionStepID {
				step, apiErr = s.findScanStep(ctx, accountID, candidate.ID)
				if apiErr != nil {
					return nil, apiErr
				}
				break
			}
		}
	}
	if step == nil {
		step, apiErr = stepRepo.FindOneByScanningStationAndProducedBlock(ctx, accountID, params.ScanningStationID, batch.Item.ID)
		if apiErr != nil {
			if apierror.IsNotFound(apiErr) {
				return nil, apierror.NewResourceNotFoundError("Production step not found.")
			}
			return nil, apiErr
		}
	}

	return s.consumptionsForQuantity(ctx, accountID, step, batch.Quantity)
}

// moveConsumption is what moving a batch would draw: the step's materials for what the batch's furthest
// open descendant makes at it.
func (s *batchSvcImpl) moveConsumption(ctx context.Context, accountID, batchID, stepID string) ([]domain.ScanningConsumption, *apierror.APIError) {
	furthest, apiErr := s.furthestOpenBatch(ctx, accountID, batchID)
	if apiErr != nil {
		return nil, apiErr
	}

	next, apiErr := s.repos.NewProductionStepQueryRepo().CalculateNextStepQuantities(ctx, accountID, furthest.Item.ID, furthest.Quantity, stepID)
	if apiErr != nil {
		return nil, apiErr
	}
	producedUnit, apiErr := s.findScanUnit(ctx, accountID, next.ProducedUnitID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Produced unit not found.")
		}
		return nil, apiErr
	}

	step, apiErr := s.findScanStep(ctx, accountID, stepID)
	if apiErr != nil {
		return nil, apiErr
	}

	return s.consumptionsForQuantity(ctx, accountID, step, domain.BatchQuantity{Measure: next.Quantity, Unit: *producedUnit})
}

// splitConsumption is what a split would draw: the step's materials for the quantity being split off.
func (s *batchSvcImpl) splitConsumption(ctx context.Context, accountID, batchID, stepID string, splitQuantity *domain.BatchQuantity) ([]domain.ScanningConsumption, *apierror.APIError) {
	if apiErr := s.checkBatchExists(ctx, accountID, batchID); apiErr != nil {
		return nil, apiErr
	}
	if splitQuantity == nil {
		return nil, apierror.NewValidationError("Split quantity not found.")
	}

	if _, apiErr := s.furthestOpenBatch(ctx, accountID, batchID); apiErr != nil {
		return nil, apiErr
	}

	step, apiErr := s.findScanStep(ctx, accountID, stepID)
	if apiErr != nil {
		return nil, apiErr
	}

	unit, apiErr := s.findScanUnit(ctx, accountID, splitQuantity.Unit.ID)
	if apiErr != nil {
		return nil, apiErr
	}

	return s.consumptionsForQuantity(ctx, accountID, step, domain.BatchQuantity{Measure: splitQuantity.Measure, Unit: *unit})
}

// furthestOpenBatch follows a batch to the newest open batch in its flow, which is the one a move or
// split at the station would take from.
func (s *batchSvcImpl) furthestOpenBatch(ctx context.Context, accountID, batchID string) (*domain.BaseBatch, *apierror.APIError) {
	if apiErr := s.checkBatchExists(ctx, accountID, batchID); apiErr != nil {
		return nil, apiErr
	}

	furthest, apiErr := s.repos.NewBatchRepo().FindFurthestRightBatchInFlow(ctx, accountID, batchID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Batch not found.")
		}
		return nil, apiErr
	}
	if furthest.ClosedAt != nil {
		return nil, apierror.NewValidationError("Batch is closed.")
	}
	return furthest, nil
}

func (s *batchSvcImpl) checkBatchExists(ctx context.Context, accountID, batchID string) *apierror.APIError {
	if _, apiErr := s.repos.NewBatchRepo().Find(ctx, accountID, batchID); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return apierror.NewResourceNotFoundError("Batch not found.")
		}
		return apiErr
	}
	return nil
}

// consumptionsForQuantity scales each of the step's consumptions by how many of the step's runs the
// quantity is, once it is restated in the production's unit.
func (s *batchSvcImpl) consumptionsForQuantity(ctx context.Context, accountID string, step *domain.ProductionStepDetail, quantity domain.BatchQuantity) ([]domain.ScanningConsumption, *apierror.APIError) {
	inProductionUnit, apiErr := domain.ConvertScanQuantity(ctx, s.repos.NewUnitConversionRepo(), quantity.Measure, quantity.Unit, step.Production.Quantity.Unit)
	if apiErr != nil {
		return nil, apiErr
	}

	demand := func(c domain.StepConsumption) float64 {
		return scanDemand(c.Quantity.Measure, inProductionUnit, step.Production.Quantity.Measure)
	}
	return s.consumptionRows(ctx, accountID, step, demand)
}

// manyBatchConsumption is what a scan of several batches would draw: the step's materials for what all
// of them make at it together.
func (s *batchSvcImpl) manyBatchConsumption(ctx context.Context, accountID string, batchIDs []string, stepID string) ([]domain.ScanningConsumption, *apierror.APIError) {
	sources, apiErr := s.findScanSources(ctx, accountID, batchIDs, stepID)
	if apiErr != nil {
		return nil, apiErr
	}

	step, apiErr := s.findScanStep(ctx, accountID, stepID)
	if apiErr != nil {
		return nil, apiErr
	}

	stepRepo := s.repos.NewProductionStepQueryRepo()
	total := 0.0
	for _, source := range sources {
		next, apiErr := stepRepo.CalculateNextStepQuantities(ctx, accountID, source.Item.ID, source.Quantity, stepID)
		if apiErr != nil {
			return nil, apiErr
		}
		total += next.Quantity.InexactFloat64()
	}

	// The step's runs are counted as the reciprocal of runs-per-output, so no output at all draws
	// nothing rather than failing.
	multiplier := scanRunsFor(step.Production.Quantity.Measure, total)
	demand := func(c domain.StepConsumption) float64 {
		if math.IsInf(multiplier, 0) || math.IsNaN(multiplier) {
			return c.Quantity.Measure.InexactFloat64() * multiplier
		}
		return c.Quantity.Measure.Mul(decimal.NewFromFloat(multiplier)).InexactFloat64()
	}
	return s.consumptionRows(ctx, accountID, step, demand)
}

func (s *batchSvcImpl) consumptionRows(ctx context.Context, accountID string, step *domain.ProductionStepDetail, demand func(domain.StepConsumption) float64) ([]domain.ScanningConsumption, *apierror.APIError) {
	results := make([]domain.ScanningConsumption, 0, len(step.Consumptions))
	for _, c := range step.Consumptions {
		onHand, unit, apiErr := s.availableToPromiseForScan(ctx, accountID, c.ConsumedItem.ID)
		if apiErr != nil {
			return nil, apiErr
		}

		results = append(results, domain.ScanningConsumption{
			SKU:              c.ConsumedItem.SKU,
			DemandMeasure:    formatQuantity(demand(c)),
			DemandUnit:       c.Quantity.Unit.Abbreviation,
			InventoryMeasure: formatQuantity(onHand.InexactFloat64()),
			InventoryUnit:    unit,
			Instructions:     c.Instructions,
		})
	}
	return results, nil
}

// availableToPromiseForScan is an item's available-to-promise as the scanning station has always shown
// it: in the unit its stock was received in when all of it shares one, otherwise in its type's base
// unit.
func (s *batchSvcImpl) availableToPromiseForScan(ctx context.Context, accountID, itemID string) (decimal.Decimal, string, *apierror.APIError) {
	invRepo := s.repos.NewInventoryQueryRepo()

	snapshot, apiErr := invRepo.FetchCurrentInventory(ctx, itemID, accountID)
	if apiErr != nil {
		return decimal.Zero, "", apiErr
	}

	// The snapshot is in the item's unit-group base unit.
	unitGroup, apiErr := s.findItemUnitGroup(ctx, accountID, itemID)
	if apiErr != nil {
		return decimal.Zero, "", apiErr
	}
	if unitGroup == nil {
		return snapshot.AvailableToPromiseMeasure, snapshot.AvailableToPromiseUnitAbbreviation, nil
	}

	receiptUnitIDs, apiErr := invRepo.ListAvailableReceiptUnitIDs(ctx, itemID, accountID)
	if apiErr != nil {
		return decimal.Zero, "", apiErr
	}
	targetID := baseUnitIDByType[unitGroup.BaseUnit.Type]
	if len(receiptUnitIDs) == 1 {
		targetID = receiptUnitIDs[0]
	}
	if targetID == "" || targetID == unitGroup.BaseUnit.ID {
		return snapshot.AvailableToPromiseMeasure, unitGroup.BaseUnit.Abbreviation, nil
	}

	target, apiErr := s.findScanUnit(ctx, accountID, targetID)
	if apiErr != nil {
		return decimal.Zero, "", apiErr
	}
	measure, apiErr := domain.ConvertScanQuantity(ctx, s.repos.NewUnitConversionRepo(), snapshot.AvailableToPromiseMeasure, unitGroup.BaseUnit, *target)
	if apiErr != nil {
		return decimal.Zero, "", apiErr
	}
	return measure, target.Abbreviation, nil
}

// scanDemand is a consumption's measure scaled by how many of the step's runs a quantity is. A step
// that produces nothing per run makes the count unbounded, which reads out as it always has.
func scanDemand(consumed, quantity, produced decimal.Decimal) float64 {
	if produced.IsZero() {
		return consumed.InexactFloat64() * (quantity.InexactFloat64() / 0)
	}
	return consumed.Mul(quantity.Div(produced)).InexactFloat64()
}

// scanRunsFor is how many of the step's runs make total, taken as the reciprocal of runs-per-output.
func scanRunsFor(produced decimal.Decimal, total float64) float64 {
	if total == 0 {
		return 0
	}
	perOutput := produced.Div(decimal.NewFromFloat(total))
	if perOutput.IsZero() {
		return math.Inf(1)
	}
	return decimal.NewFromInt(1).Div(perOutput).InexactFloat64()
}

// formatQuantity reads a measure out for an operator: whole numbers as they are, anything else rounded
// to four places with the trailing zeros dropped.
func formatQuantity(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	}
	if v == math.Trunc(v) {
		if v == 0 {
			return "0"
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	}

	// Rounded from the float's exact value, half away from zero, then printed in its shortest form.
	exact, err := decimal.NewFromString(new(big.Float).SetFloat64(v).Text('f', 1100))
	if err != nil {
		return strconv.FormatFloat(v, 'f', 4, 64)
	}
	rounded := exact.Round(4).InexactFloat64()
	if rounded == 0 {
		return "0"
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// ---------------------------------------------------------------------------
// Writing a scan
// ---------------------------------------------------------------------------

// scanOutput is a batch a move, merge or split creates, and what creating it does to its sources.
type scanOutput struct {
	productionStepID  string
	scanningStationID string
	itemID            string
	quantity          domain.CreateQuantityParams
	seconds           *domain.CreateQuantityParams
	waste             *domain.CreateQuantityParams
	sourceIDs         []string
	// closeSources closes each source as the output is connected to it.
	closeSources bool
	// closeWhenFullyUsed is a source closed once what has been split off it accounts for all it makes at
	// the step, measured in closeWhenFullyUsedUnit.
	closeWhenFullyUsed     *domain.BaseBatch
	closeWhenFullyUsedUnit domain.LightUnit
}

// writeScanOutput creates the batch a scan makes, already scanned, and in the same transaction connects
// it to its sources, closes what the scan used up, and writes the batch_scanned event that moves its
// inventory, the usage meter, and the audit record. A scan is all of these or none of them.
// scanPlan works out what a scan makes and which batches it takes from, reading through svc.
type scanPlan func(ctx context.Context, svc *batchSvcImpl) (*scanOutput, *apierror.APIError)

// errScanSourcesMoved marks a scan whose plan, made again once its batches were held, takes from a batch
// it did not hold.
var errScanSourcesMoved = apierror.NewResourceConflictError("These batches changed while they were being scanned. Scan them again.")

// maxScanPlans bounds how often a scan starts over because the batches it takes from moved under it.
const maxScanPlans = 3

// planAndWriteScan plans a scan to learn which batches it takes from, then writes it holding them.
func (s *batchSvcImpl) planAndWriteScan(ctx context.Context, identity *types.Identity, accountID string, key *domain.IdempotencyKey, scannedIDs []string, plan scanPlan) (*domain.BaseBatch, *apierror.APIError) {
	for range maxScanPlans {
		out, apiErr := plan(ctx, s)
		if apiErr != nil {
			return nil, apiErr
		}
		result, apiErr := s.writeScanOutput(ctx, identity, accountID, key, scannedIDs, out, plan)
		if apiErr != errScanSourcesMoved {
			return result, apiErr
		}
	}
	return nil, errScanSourcesMoved
}

// writeScanOutput writes a scan's batch. It first holds the batches the operator scanned and the ones
// the plan takes from, then plans again: a scan of them that committed while this one waited may have
// used them up or closed them, and this one must then go as it would have gone after it.
func (s *batchSvcImpl) writeScanOutput(ctx context.Context, identity *types.Identity, accountID string, key *domain.IdempotencyKey, scannedIDs []string, out *scanOutput, plan scanPlan) (*domain.BaseBatch, *apierror.APIError) {
	newBatchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	meter, apiErr := s.meterBatches(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}

	var result *domain.BaseBatch
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchSvcImpl) *apierror.APIError {
		txCtx = event.WithRepos(txCtx, txSvc.repos)
		txBatchRepo := txSvc.repos.NewBatchRepo()

		held, apiErr := holdBatches(txCtx, txBatchRepo, accountID, scannedIDs, out.sourceIDs)
		if apiErr != nil {
			return apiErr
		}
		out, apiErr = plan(txCtx, txSvc)
		if apiErr != nil {
			return apiErr
		}
		for _, sourceID := range out.sourceIDs {
			if _, isHeld := held[sourceID]; !isHeld {
				return errScanSourcesMoved
			}
		}

		if _, apiErr := txBatchRepo.Create(txCtx, newBatchID, domain.CreateBatchParams{
			AccountID:         accountID,
			ItemID:            out.itemID,
			Quantity:          out.quantity,
			Seconds:           out.seconds,
			Waste:             out.waste,
			ProductionStepID:  out.productionStepID,
			ScanningStationID: out.scanningStationID,
		}); apiErr != nil {
			return apiErr
		}
		if apiErr := txBatchRepo.MarkAsScanned(txCtx, accountID, newBatchID); apiErr != nil {
			return apiErr
		}

		if len(out.sourceIDs) == 1 {
			if apiErr := txBatchRepo.ConnectOneToOne(txCtx, accountID, out.sourceIDs[0], newBatchID, out.closeSources); apiErr != nil {
				return apiErr
			}
		} else {
			if apiErr := txBatchRepo.ConnectManyToOne(txCtx, accountID, out.sourceIDs, newBatchID, out.closeSources); apiErr != nil {
				return apiErr
			}
		}

		if out.closeWhenFullyUsed != nil {
			if apiErr := txBatchRepo.CloseIfFullyUsed(txCtx, accountID, *out.closeWhenFullyUsed, out.closeWhenFullyUsedUnit, out.productionStepID); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txBatchRepo.CloseIfLastStep(txCtx, accountID, newBatchID, out.productionStepID); apiErr != nil {
			return apiErr
		}

		created, apiErr := txBatchRepo.Find(txCtx, accountID, newBatchID)
		if apiErr != nil {
			return apiErr
		}
		result = baseBatchOf(created)

		if apiErr := publishBatchScanned(txCtx, txSvc.repos, identity, accountID, out.productionStepID, out.scanningStationID, result); apiErr != nil {
			return apiErr
		}

		if meter {
			// Metering must never fail a scan; the command rides the outbox, so a rolled-back scan never meters.
			if apiErr := txSvc.billingPub.PublishReportBatchCreated(txCtx, accountID, newBatchID); apiErr != nil {
				slog.WarnContext(txCtx, "batch usage metering failed; scanning anyway",
					"account_id", accountID, "batch_id", newBatchID, "error", apiErr.Error())
			}
		}

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeBatch,
			ResourceID:   result.ID,
			Changes:      audit.ComputeChanges(nil, result),
		}); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
	})
	if apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

// meterBatches reports whether the batches this account's scans create are metered for billing.
// Sandboxes are not billed.
func (s *batchSvcImpl) meterBatches(ctx context.Context, accountID string) (bool, *apierror.APIError) {
	if s.billingPub == nil {
		return false, nil
	}
	accountCtx, apiErr := s.repos.NewAccountRepo().GetAccountContext(ctx, accountID)
	if apiErr != nil {
		return false, apiErr
	}
	return accountCtx == nil || !accountCtx.IsSandbox, nil
}

// publishBatchScanned writes the event that a batch was scanned at a station, which is what the
// inventory consumer moves stock on: the produced receipt, the reservations seconds and waste free, and
// the materials the step consumes.
func publishBatchScanned(ctx context.Context, repos domain.RepoFactory, identity *types.Identity, accountID, productionStepID, scanningStationID string, batch *domain.BaseBatch) *apierror.APIError {
	evt := domain.BatchScannedEvent{
		AccountID:         accountID,
		BatchID:           batch.ID,
		ProductionStepID:  productionStepID,
		ScanningStationID: scanningStationID,
		ItemID:            batch.Item.ID,
		Measure:           batch.Quantity.Measure.String(),
		UnitID:            batch.Quantity.Unit.ID,
	}
	if batch.Seconds != nil {
		evt.SecondsMeasure = batch.Seconds.Measure.String()
	}
	if batch.Waste != nil {
		evt.WasteMeasure = batch.Waste.Measure.String()
	}
	if identity.Actor != nil {
		evt.ResponsibleUserID = &identity.Actor.ID
	}
	if batch.ScannedAt == nil {
		return apierror.NewInvariantViolationError("A scanned batch has no scan time.")
	}
	evt.ScannedAt = *batch.ScannedAt

	payload, err := json.Marshal(evt)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal batch scanned event.")
	}

	msg := contracts.AmqpMessage{Data: payload}
	if ctxIdentity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = ctxIdentity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	if _, err := repos.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreEventBatchScanned),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreEventBatchScanned),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for batch scanned event.")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shared lookups and checks
// ---------------------------------------------------------------------------

func (s *batchSvcImpl) checkStationInAccount(ctx context.Context, accountID, scanningStationID string) *apierror.APIError {
	inAccount, apiErr := s.repos.NewScanningStationQueryRepo().IsInAccount(ctx, accountID, scanningStationID)
	if apiErr != nil {
		return apiErr
	}
	if !inAccount {
		return apierror.NewResourceNotFoundError("Scanning Station not found.")
	}
	return nil
}

// findScanSources resolves each scanned batch forward to the batch waiting at the step. Each must be a
// different batch, and each id must resolve.
func (s *batchSvcImpl) findScanSources(ctx context.Context, accountID string, batchIDs []string, productionStepID string) ([]domain.BaseBatch, *apierror.APIError) {
	sources, apiErr := s.repos.NewBatchRepo().FindAvailableBatchesInFlow(ctx, accountID, batchIDs, productionStepID)
	if apiErr != nil {
		return nil, apiErr
	}

	seen := make(map[string]bool, len(sources))
	for _, source := range sources {
		if seen[source.ID] {
			return nil, apierror.NewValidationError("Duplicate batches provided.")
		}
		seen[source.ID] = true
	}
	if len(sources) != len(batchIDs) {
		return nil, apierror.NewResourceNotFoundError("One or more batches not found.")
	}
	return sources, nil
}

func (s *batchSvcImpl) findScanStep(ctx context.Context, accountID, productionStepID string) (*domain.ProductionStepDetail, *apierror.APIError) {
	step, apiErr := s.repos.NewProductionStepQueryRepo().Find(ctx, accountID, productionStepID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Production step not found.")
		}
		return nil, apiErr
	}
	if step == nil {
		return nil, apierror.NewResourceNotFoundError("Production step not found.")
	}
	return step, nil
}

func (s *batchSvcImpl) findScanUnit(ctx context.Context, accountID, unitID string) (*domain.LightUnit, *apierror.APIError) {
	unit, apiErr := s.repos.NewUnitQueryRepo().Find(ctx, accountID, unitID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewResourceNotFoundError("Unit not found.")
		}
		return nil, apiErr
	}
	return unit, nil
}

// findItemUnitGroup is the item's unit group, or nil when it has none.
func (s *batchSvcImpl) findItemUnitGroup(ctx context.Context, accountID, itemID string) (*domain.UnitGroup, *apierror.APIError) {
	unitGroup, apiErr := s.repos.NewUnitGroupQueryRepo().FindByItem(ctx, accountID, itemID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, nil
		}
		return nil, apiErr
	}
	return unitGroup, nil
}

// totalInUnit adds the batches' quantities in one unit.
func (s *batchSvcImpl) totalInUnit(ctx context.Context, batches []domain.BaseBatch, unit domain.LightUnit) (domain.BatchQuantity, *apierror.APIError) {
	conv := s.repos.NewUnitConversionRepo()
	total := decimal.Zero
	for _, b := range batches {
		measure, apiErr := domain.ConvertScanQuantity(ctx, conv, b.Quantity.Measure, b.Quantity.Unit, unit)
		if apiErr != nil {
			return domain.BatchQuantity{}, apiErr
		}
		total = total.Add(measure)
	}
	return domain.BatchQuantity{Measure: total, Unit: unit}, nil
}

// partGroup is the batches of one part in a multi-part scan.
type partGroup struct {
	itemID    string
	batches   []domain.BaseBatch
	unitGroup *domain.UnitGroup
}

// groupByPart groups batches by part, in the order the parts first appear.
func (s *batchSvcImpl) groupByPart(ctx context.Context, accountID string, batches []domain.BaseBatch) ([]*partGroup, *apierror.APIError) {
	var groups []*partGroup
	byItem := make(map[string]*partGroup)
	for _, b := range batches {
		group, ok := byItem[b.Item.ID]
		if !ok {
			unitGroup, apiErr := s.findItemUnitGroup(ctx, accountID, b.Item.ID)
			if apiErr != nil {
				return nil, apiErr
			}
			group = &partGroup{itemID: b.Item.ID, unitGroup: unitGroup}
			byItem[b.Item.ID] = group
			groups = append(groups, group)
		}
		group.batches = append(group.batches, b)
	}
	return groups, nil
}

// stepGroupsThrough totals each part's batches in its base unit and steps the total through, returning
// what each part makes and the last result for the produced item and unit.
func (s *batchSvcImpl) stepGroupsThrough(ctx context.Context, accountID string, groups []*partGroup, productionStepID string) (*domain.NextStepQuantitiesResult, []decimal.Decimal, *apierror.APIError) {
	stepRepo := s.repos.NewProductionStepQueryRepo()
	var next *domain.NextStepQuantitiesResult
	measures := make([]decimal.Decimal, 0, len(groups))
	for _, group := range groups {
		if group.unitGroup == nil {
			return nil, nil, apierror.NewResourceNotFoundError("Unit group not found.")
		}
		total, apiErr := s.totalInUnit(ctx, group.batches, group.unitGroup.BaseUnit)
		if apiErr != nil {
			return nil, nil, apiErr
		}
		next, apiErr = stepRepo.CalculateNextStepQuantities(ctx, accountID, group.itemID, total, productionStepID)
		if apiErr != nil {
			return nil, nil, apiErr
		}
		measures = append(measures, next.Quantity)
	}
	return next, measures, nil
}

// checkRequiredParts requires a batch of every part the step consumes. Materials are consumed, not
// scanned in, so they are not required.
func checkRequiredParts(step *domain.ProductionStepDetail, batches []domain.BaseBatch) *apierror.APIError {
	scanned := make(map[string]bool, len(batches))
	for _, b := range batches {
		scanned[b.Item.ID] = true
	}
	for _, c := range step.Consumptions {
		if c.ConsumedItem.Type != string(constants.ItemTypeCodePart) {
			continue
		}
		if !scanned[c.ConsumedItem.ID] {
			return apierror.NewValidationError("Missing required part: " + c.ConsumedItem.SKU)
		}
	}
	return nil
}

// checkSameMeasure requires every part of a multi-part scan to make the same amount.
func checkSameMeasure(measures []decimal.Decimal) *apierror.APIError {
	for _, m := range measures {
		if !m.Equal(measures[0]) {
			return apierror.NewValidationError("All batches must have the same measure.")
		}
	}
	return nil
}

func isScanningStationType(t string) bool {
	switch constants.ScanningStationType(t) {
	case constants.ScanningStationTypeInitBatch,
		constants.ScanningStationTypeMoveBatch,
		constants.ScanningStationTypeSplitBatch,
		constants.ScanningStationTypeMergeBatch:
		return true
	}
	return false
}

func batchIDsOf(batches []domain.BaseBatch) []string {
	ids := make([]string, len(batches))
	for i, b := range batches {
		ids[i] = b.ID
	}
	return ids
}

// baseBatchOf is the mutation-response view of a batch.
func baseBatchOf(b *domain.Batch) *domain.BaseBatch {
	base := &domain.BaseBatch{
		ID:              b.ID,
		Item:            b.Item,
		Quantity:        b.Quantity,
		Seconds:         b.Seconds,
		Waste:           b.Waste,
		ScanningStation: b.ScanningStation,
		DepartmentID:    b.DepartmentID,
		DepartmentName:  b.DepartmentName,
		ProductionStep:  b.ProductionStep,
		ProductionRun:   b.ProductionRun,
		Machines:        b.Machines,
		Lots:            b.Lots,
		ClosedAt:        b.ClosedAt,
		ScannedAt:       b.ScannedAt,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
	if b.ProductionRun != nil {
		runID := b.ProductionRun.ID
		base.ProductionRunID = &runID
	}
	return base
}
