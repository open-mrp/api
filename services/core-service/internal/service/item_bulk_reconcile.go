package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	// maxBulkReconcileRows bounds one request. Every row is written before the response returns, so
	// the cap is what keeps a request inside its deadline; larger counts are split by the caller.
	maxBulkReconcileRows = 1000
	// bulkReconcileBatchSize is how many rows share a ledger transaction.
	bulkReconcileBatchSize = 50
)

// reconcileRow is a submitted row whose SKU and unit resolved, its quantity carried into the item's base unit.
type reconcileRow struct {
	sku         string
	item        domain.ItemSKUInfo
	baseMeasure decimal.Decimal
}

// reconcileLookups is what the request's rows are resolved against.
type reconcileLookups struct {
	itemsBySKU map[string]domain.ItemSKUInfo
	// unitsByAbbreviation is keyed by lower-cased abbreviation: an account may define a unit sharing a built-in's.
	unitsByAbbreviation map[string][]*domain.Unit
	// groupUnits is each unit group's members, its base unit included.
	groupUnits map[string]map[string]bool
	factors    map[string]domain.UnitFactors
}

// BulkReconcileItems reconciles inventory for many items by SKU, converting each row into its item's base unit.
//
// Rows are written in batches, each its own ledger transaction. A batch that fails reports its rows
// as errors and the rest carry on: earlier batches are already committed, so failing the request
// would invite a resubmission that applies them twice.
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
	if len(params.Data) > maxBulkReconcileRows {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Cannot reconcile more than %d rows at a time.", maxBulkReconcileRows), "data"))
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
		lookups, apiErr := s.loadReconcileLookups(ctx, accountID, params.Data)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}

		rows, result := planBulkReconcile(params.Data, lookups)
		failures := runReconcileBatches(rows, result, func(batch []reconcileRow) ([]domain.ReconciledItem, *apierror.APIError) {
			return s.reconcileBatch(ctx, params, batch, lookups.factors)
		})
		for _, failure := range failures {
			tracing.Trace(span, failure)
		}

		// Cached once, after every batch has run, so a replay of the key returns what this call did.
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
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

// loadReconcileLookups reads the items, units, unit-group members and conversion factors the rows name.
func (s *itemSvcImpl) loadReconcileLookups(ctx context.Context, accountID string, data []domain.BulkReconcileItemInput) (*reconcileLookups, *apierror.APIError) {
	skuSet := make(map[string]struct{}, len(data))
	abbrevSet := make(map[string]struct{}, len(data))
	skus := make([]string, 0, len(data))
	abbreviations := make([]string, 0, len(data))
	for _, d := range data {
		if _, seen := skuSet[d.SKU]; !seen {
			skuSet[d.SKU] = struct{}{}
			skus = append(skus, d.SKU)
		}
		if _, seen := abbrevSet[d.Unit]; !seen {
			abbrevSet[d.Unit] = struct{}{}
			abbreviations = append(abbreviations, d.Unit)
		}
	}

	items, apiErr := s.repos.NewItemRepo().FetchItemsBySKU(ctx, accountID, skus)
	if apiErr != nil {
		return nil, apiErr
	}
	units, apiErr := s.repos.NewUnitRepo().FindByAbbreviations(ctx, accountID, abbreviations)
	if apiErr != nil {
		return nil, apiErr
	}

	lookups := &reconcileLookups{
		itemsBySKU:          make(map[string]domain.ItemSKUInfo, len(items)),
		unitsByAbbreviation: make(map[string][]*domain.Unit, len(units)),
		groupUnits:          make(map[string]map[string]bool),
	}
	unitIDs := make([]string, 0, len(units)+len(items))
	groupIDs := make([]string, 0, len(items))
	for _, item := range items {
		lookups.itemsBySKU[item.SKU] = item
		if lookups.groupUnits[item.UnitGroupID] == nil {
			lookups.groupUnits[item.UnitGroupID] = map[string]bool{}
			groupIDs = append(groupIDs, item.UnitGroupID)
		}
		lookups.groupUnits[item.UnitGroupID][item.BaseUnitID] = true
		unitIDs = append(unitIDs, item.BaseUnitID)
	}
	for _, unit := range units {
		key := strings.ToLower(unit.Abbreviation)
		lookups.unitsByAbbreviation[key] = append(lookups.unitsByAbbreviation[key], unit)
		unitIDs = append(unitIDs, unit.ID)
	}

	members, apiErr := s.repos.NewUnitGroupRepo().FindUnitsByGroupIDs(ctx, groupIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	for _, member := range members {
		if lookups.groupUnits[member.UnitGroupID] != nil {
			lookups.groupUnits[member.UnitGroupID][member.UnitID] = true
		}
	}

	lookups.factors, apiErr = s.repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, unitIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	return lookups, nil
}

// planBulkReconcile sorts the submitted rows, in order, into those to write, unknown SKUs (skipped)
// and unusable units (errors). A row's unit must belong to its item's unit group: membership is what
// defines how many of one unit make the item's base unit.
func planBulkReconcile(data []domain.BulkReconcileItemInput, lookups *reconcileLookups) ([]reconcileRow, *domain.BulkReconcileItemsResult) {
	result := &domain.BulkReconcileItemsResult{}
	rows := make([]reconcileRow, 0, len(data))
	for _, d := range data {
		item, ok := lookups.itemsBySKU[d.SKU]
		if !ok {
			result.SkippedItems = append(result.SkippedItems, domain.SkippedItem{SKU: d.SKU, Reason: "Item not found"})
			continue
		}
		rowError := func(msg string) {
			result.Errors = append(result.Errors, domain.ReconcileError{ItemID: item.ItemID, SKU: d.SKU, Error: msg})
		}

		candidates := lookups.unitsByAbbreviation[strings.ToLower(d.Unit)]
		if len(candidates) == 0 {
			rowError(fmt.Sprintf("Unit '%s' not found", d.Unit))
			continue
		}
		var unit *domain.Unit
		for _, candidate := range candidates {
			if lookups.groupUnits[item.UnitGroupID][candidate.ID] {
				unit = candidate
				break
			}
		}
		if unit == nil {
			rowError(fmt.Sprintf("Unit '%s' is not in this item's unit group; use its base unit or another unit its category allows", d.Unit))
			continue
		}

		baseMeasure, ok := convertToUnit(d.Measure, unit.ID, item.BaseUnitID, lookups.factors)
		if !ok {
			rowError(fmt.Sprintf("Unit '%s' cannot be converted into this item's base unit", d.Unit))
			continue
		}
		rows = append(rows, reconcileRow{sku: d.SKU, item: item, baseMeasure: baseMeasure})
	}
	return rows, result
}

// convertToUnit carries measure from one unit to another through their dimension's base unit.
func convertToUnit(measure decimal.Decimal, fromUnitID, toUnitID string, factors map[string]domain.UnitFactors) (decimal.Decimal, bool) {
	if fromUnitID == toUnitID {
		return measure, true
	}
	from, okFrom := factors[fromUnitID]
	to, okTo := factors[toUnitID]
	if !okFrom || !okTo || from.DimensionCode != to.DimensionCode {
		return decimal.Zero, false
	}
	return to.FromBase(from.ToBase(measure)), true
}

// runReconcileBatches writes rows bulkReconcileBatchSize at a time into result. A batch that fails is
// reported row by row in result.Errors and the next one still runs; the failures are returned.
func runReconcileBatches(rows []reconcileRow, result *domain.BulkReconcileItemsResult, write func([]reconcileRow) ([]domain.ReconciledItem, *apierror.APIError)) []*apierror.APIError {
	var failures []*apierror.APIError
	for start := 0; start < len(rows); start += bulkReconcileBatchSize {
		batch := rows[start:min(start+bulkReconcileBatchSize, len(rows))]
		reconciled, apiErr := write(batch)
		if apiErr != nil {
			failures = append(failures, apiErr)
			result.Errors = append(result.Errors, failedBatchErrors(batch, apiErr)...)
			continue
		}
		result.ReconciledItems = append(result.ReconciledItems, reconciled...)
	}
	return failures
}

// failedBatchErrors reports every row of a batch whose transaction rolled back.
func failedBatchErrors(batch []reconcileRow, apiErr *apierror.APIError) []domain.ReconcileError {
	errs := make([]domain.ReconcileError, len(batch))
	for i, row := range batch {
		errs[i] = domain.ReconcileError{
			ItemID: row.item.ItemID,
			SKU:    row.sku,
			Error:  "Not reconciled; nothing in its batch was written: " + apiErr.PublicMessage,
		}
	}
	return errs
}

// reconcileBatch writes one batch in one transaction. The level each row is measured against is read
// after the batch's items are locked, and carried row to row, so a SKU repeated in the batch applies
// to what the row before it left.
func (s *itemSvcImpl) reconcileBatch(ctx context.Context, params domain.BulkReconcileItemsParams, batch []reconcileRow, factors map[string]domain.UnitFactors) ([]domain.ReconciledItem, *apierror.APIError) {
	accountID := params.AccountID
	itemIDs := make([]string, len(batch))
	for i, row := range batch {
		itemIDs[i] = row.item.ItemID
	}

	// Assigned, not appended: the callback re-runs on a lock conflict and the rerun replaces the first run's rows.
	var reconciled []domain.ReconciledItem
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
		invMutRepo := txSvc.repos.NewInventoryMutationRepo()
		scope, apiErr := ledgerlock.Acquire(txCtx, invMutRepo, itemIDs)
		if apiErr != nil {
			return apiErr
		}

		physicalBase, apiErr := txSvc.repos.NewInventoryQueryRepo().FetchPhysicalInventoryBaseForItems(txCtx, accountID, itemIDs)
		if apiErr != nil {
			return apiErr
		}
		levels := make(map[string]decimal.Decimal, len(batch))
		for _, row := range batch {
			if _, seen := levels[row.item.ItemID]; !seen {
				levels[row.item.ItemID] = factors[row.item.BaseUnitID].FromBase(physicalBase[row.item.ItemID])
			}
		}

		rows := make([]domain.ReconciledItem, 0, len(batch))
		moved := make([]string, 0, len(batch))
		for _, row := range batch {
			itemID, unitID := row.item.ItemID, row.item.BaseUnitID
			current := levels[itemID]
			next := current.Add(row.baseMeasure)
			if params.ReconcileType == string(constants.ItemReconcileTypeForce) {
				next = row.baseMeasure
			}
			delta := next.Sub(current)
			levels[itemID] = next

			switch {
			case delta.IsPositive():
				if apiErr := invMutRepo.CreateInventoryReceipt(txCtx, scope, domain.CreateInventoryReceiptParams{
					AccountID: accountID, ItemID: itemID, Measure: delta, UnitID: unitID,
				}); apiErr != nil {
					return apiErr
				}
			case delta.IsNegative():
				if apiErr := invMutRepo.CreateInventoryIssue(txCtx, scope, domain.CreateInventoryIssueParams{
					AccountID: accountID, ItemID: itemID, Measure: delta.Abs(), UnitID: unitID,
				}); apiErr != nil {
					return apiErr
				}
			}

			if apiErr := mediator.RecordInventoryAuditTrailWithLevel(txCtx, txSvc.repos, accountID, itemID, delta, next, unitID,
				"user_correction", nil, params.ResponsibleUserID); apiErr != nil {
				return apiErr
			}

			// Empty when the row changed nothing; the publisher skips the event as a no-op.
			var changes []audit.FieldChange
			if !delta.IsZero() {
				changes = append(changes, audit.NewFieldChange("quantity", current, next))
				moved = append(moved, itemID)
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeItem,
				ResourceID:   itemID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			rows = append(rows, domain.ReconciledItem{
				ItemID: itemID, SKU: row.sku,
				PreviousMeasure: current, NewMeasure: next,
				UnitID: unitID,
			})
		}

		// Stock moved upward is offered to whatever demand is waiting on it, in the transaction that moved it.
		if apiErr := mediator.EnqueueAllocateOpenIssues(txCtx, txSvc.repos, accountID, moved...); apiErr != nil {
			return apiErr
		}

		reconciled = rows
		return nil
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return reconciled, nil
}
