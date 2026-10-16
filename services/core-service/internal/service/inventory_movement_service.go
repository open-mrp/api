package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

// bulkMovementChunkSize is how many rows of a bulk movement job share a ledger transaction.
const bulkMovementChunkSize = 50

// createdMovement is what a create caches under its idempotency key: the movement it wrote.
type createdMovement struct {
	ID string `json:"id"`
}

// movementWrite is one hand-recorded movement, resolved and ready to write inside a ledger transaction.
type movementWrite struct {
	id         string
	itemID     string
	reconcile  bool
	measure    decimal.Decimal
	unitID     string
	ownerID    string
	locationID *string
	lotID      *string
	lotNumber  *string
	reason     *constants.InventoryMovementReason
	actor      actor.Actor
	// legacyUserID is the caller as the preview.9 change log records it.
	legacyUserID *string
}

// CreateInventoryMovement records an adjustment or reconciliation by hand.
//
//  1. Checks the item is stocked, the unit is one of its units, and the location, lot and customer may be used.
//  2. Under the item's ledger lock, measures the item's physical stock in that unit and works out the change: the
//     quantity for an adjustment, the difference from the count for a reconciliation.
//  3. Writes the receipt or issue, asks for allocation, and records the movement, even a reconciliation that changed
//     nothing (the count is the record).
func (s *itemSvcImpl) CreateInventoryMovement(ctx context.Context, params domain.CreateInventoryMovementParams) (*domain.InventoryChangeLog, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.create_inventory_movement")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	if params.LotID != nil && params.LotNumber != nil {
		return nil, tracing.Trace(span, apierror.NewFieldError("lot_number", apierror.CodeInvalidFormat, "Send lot_id or lot_number, not both."))
	}
	if !params.Reconcile && params.Measure.IsZero() {
		return nil, tracing.Trace(span, apierror.NewFieldError("quantity.value", apierror.CodeInvalidFormat, "An adjustment's quantity can't be 0."))
	}
	if params.Reconcile && params.Measure.IsNegative() {
		return nil, tracing.Trace(span, apierror.NewFieldError("quantity.value", apierror.CodeInvalidFormat, "A reconciliation's quantity can't be negative."))
	}

	if apiErr := s.checkMovementItemAndUnit(ctx, accountID, params.ItemID, params.UnitID, "quantity.unit_id"); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.CustomerID != nil {
		if apiErr := s.mediators().EditAccess.CheckEditAccess(ctx, accountID, *params.CustomerID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if params.LocationID != nil {
		if apiErr := requireMovementLocations(ctx, s.repos, accountID, []string{*params.LocationID}, func(int) string { return "location_id" }); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if params.LotID != nil {
		belongs, apiErr := s.repos.NewInventoryQueryRepo().LotBelongsToItem(ctx, accountID, params.ItemID, *params.LotID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if !belongs {
			return nil, tracing.Trace(span, apierror.NewFieldError("lot_id", apierror.CodeInvalidFormat, "No lot of this item has that ID."))
		}
	}

	who, ok := actor.FromIdentity(identity)
	if !ok {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("The caller can't be recorded as a movement's actor."))
	}
	write := movementWrite{
		itemID: params.ItemID, reconcile: params.Reconcile, measure: params.Measure, unitID: params.UnitID,
		ownerID: accountID, locationID: params.LocationID, lotID: params.LotID, lotNumber: params.LotNumber,
		reason: params.Reason, actor: who,
	}
	if params.CustomerID != nil {
		write.ownerID = *params.CustomerID
	}
	if identity.Actor != nil {
		write.legacyUserID = &identity.Actor.ID
	}

	meds := s.mediators()
	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var movementID string
	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[createdMovement](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		if cached.Error != nil {
			return nil, cached.Error
		}
		movementID = cached.Data.ID

	case domain.RecoveryPointStarted:
		write.id, apiErr = id.GenID(id.InventoryMovementIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *itemSvcImpl) *apierror.APIError {
			scope, apiErr := ledgerlock.Acquire(txCtx, txSvc.repos.NewInventoryMutationRepo(), []string{write.itemID})
			if apiErr != nil {
				return apiErr
			}
			physical, apiErr := txSvc.repos.NewInventoryQueryRepo().FetchPhysicalInventory(txCtx, write.itemID, accountID, write.unitID)
			if apiErr != nil {
				return apiErr
			}
			if _, apiErr := writeMovement(txCtx, txSvc.repos, scope, accountID, write, physical); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, createdMovement{ID: write.id})
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
		}
		movementID = write.id

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}

	movement, apiErr := s.repos.NewInventoryChangeLogRepo().Get(ctx, accountID, movementID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return movement, nil
}

// writeMovement applies one hand-recorded movement against the item's physical stock (in the movement's unit) and
// returns the stock after it. The caller holds the item's ledger lock.
func writeMovement(
	txCtx context.Context,
	repos domain.RepoFactory,
	scope *ledgerlock.Scope,
	accountID string,
	w movementWrite,
	physical decimal.Decimal,
) (decimal.Decimal, *apierror.APIError) {
	invMutRepo := repos.NewInventoryMutationRepo()

	lotID := w.lotID
	if w.lotNumber != nil && *w.lotNumber != "" {
		newLotID, apiErr := id.GenID(id.LotIDPrefix, nil)
		if apiErr != nil {
			return decimal.Zero, apiErr
		}
		existing, apiErr := repos.NewReceivingOrderRepo().UpsertLot(txCtx, newLotID, accountID, w.itemID, *w.lotNumber)
		if apiErr != nil {
			return decimal.Zero, apiErr
		}
		lotID = &existing
	}

	delta := w.measure
	movementType := constants.InventoryMovementTypeAdjustment
	if w.reconcile {
		delta = w.measure.Sub(physical)
		movementType = constants.InventoryMovementTypeReconciliation
	}
	next := physical.Add(delta)

	switch {
	case delta.IsPositive():
		if apiErr := invMutRepo.CreateInventoryReceipt(txCtx, scope, domain.CreateInventoryReceiptParams{
			AccountID: accountID, OwnerAccountID: w.ownerID, HolderAccountID: accountID,
			ItemID: w.itemID, Measure: delta, UnitID: w.unitID, LocationID: w.locationID, LotID: lotID,
		}); apiErr != nil {
			return decimal.Zero, apiErr
		}
	case delta.IsNegative():
		if apiErr := invMutRepo.CreateInventoryIssue(txCtx, scope, domain.CreateInventoryIssueParams{
			AccountID: accountID, ItemID: w.itemID, Measure: delta, UnitID: w.unitID, LocationID: w.locationID, LotID: lotID,
		}); apiErr != nil {
			return decimal.Zero, apiErr
		}
	}

	if !delta.IsZero() {
		// Allocation walks every open issue of the item; the consumer does it just after this commits.
		if apiErr := mediator.EnqueueAllocateOpenIssues(txCtx, repos, accountID, w.itemID); apiErr != nil {
			return decimal.Zero, apiErr
		}
	}

	if apiErr := invMutRepo.CreateInventoryLog(txCtx, domain.CreateInventoryLogParams{
		AccountID: accountID, ItemID: w.itemID, Measure: next, UnitID: w.unitID,
	}); apiErr != nil {
		return decimal.Zero, apiErr
	}
	if apiErr := invMutRepo.CreateInventoryChangeLog(txCtx, domain.CreateInventoryChangeLogParams{
		ID:                w.id,
		AccountID:         accountID,
		ItemID:            w.itemID,
		Measure:           delta,
		UnitID:            w.unitID,
		ActionType:        string(constants.InventoryActionTypeUserCorrection),
		ResponsibleUserID: w.legacyUserID,
		Movement: domain.MovementFields{
			Type:       movementType,
			Actor:      w.actor,
			LocationID: w.locationID,
			LotID:      lotID,
			Reason:     w.reason,
		},
	}); apiErr != nil {
		return decimal.Zero, apiErr
	}

	// Empty when the movement changed nothing; the publisher skips the event as a no-op.
	var changes []audit.FieldChange
	if !delta.IsZero() {
		changes = append(changes, audit.NewFieldChange("quantity", physical, next))
	}
	if apiErr := audit.NewPublisher().Publish(txCtx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeItem,
		ResourceID:   w.itemID,
		Changes:      changes,
	}); apiErr != nil {
		return decimal.Zero, apiErr
	}
	return next, nil
}

// checkMovementItemAndUnit refuses an item that isn't stocked and a unit that isn't the item's stocking unit or one
// of its order units.
func (s *itemSvcImpl) checkMovementItemAndUnit(ctx context.Context, accountID, itemID, unitID, unitParam string) *apierror.APIError {
	item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{AccountID: accountID, ItemID: itemID})
	if apiErr != nil {
		return apiErr
	}
	if item == nil {
		return apierror.NewNotFoundError("Item not found.")
	}
	if constants.ItemTypeCode(item.ItemTypeCode) == constants.ItemTypeCodeService {
		return apierror.NewFieldError("item_id", apierror.CodeInvalidFormat, "Services aren't stocked, so they have no inventory movements.")
	}
	stocking, apiErr := s.repos.NewItemRepo().GetStockingUnit(ctx, accountID, itemID)
	if apiErr != nil {
		return apiErr
	}
	if unitID != stocking.BaseUnitID && !slices.Contains(stocking.AllowedUnitIDs, unitID) {
		return apierror.NewFieldError(unitParam, apierror.CodeInvalidFormat,
			unitParam+" must be the item's stocking unit or one of its order units.")
	}
	return nil
}

// requireMovementLocations refuses a location that isn't the account's or is archived; param names each one's field.
func requireMovementLocations(ctx context.Context, repos domain.RepoFactory, accountID string, locationIDs []string, param func(int) string) *apierror.APIError {
	found, apiErr := repos.NewLocationRepo().GetByIDs(ctx, accountID, uniqueIDs(locationIDs))
	if apiErr != nil {
		return apiErr
	}
	byID := make(map[string]*domain.Location, len(found))
	for _, l := range found {
		byID[l.ID] = l
	}
	for i, locationID := range locationIDs {
		l, ok := byID[locationID]
		if !ok {
			return apierror.NewFieldError(param(i), apierror.CodeInvalidFormat, fmt.Sprintf("No location has the ID %q.", locationID))
		}
		if apiErr := domain.RequireAssignable(param(i), "location", l.Status); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// bulkMovementInput is one submitted bulk row with the request's operation.
type bulkMovementInput struct {
	Row       domain.BulkCreateInventoryMovementRow
	Reconcile bool
}

func validateBulkMovementRows(rows []bulkMovementInput) *apierror.APIError {
	var rowErrs apierror.RowErrors
	for i, in := range rows {
		r := in.Row
		hasID, hasSKU := r.ItemID != nil && *r.ItemID != "", r.SKU != nil && *r.SKU != ""
		if hasID == hasSKU {
			rowErrs.AddField(i, "item_id", apierror.CodeInvalidFormat, "send item_id or sku, exactly one")
		}
		if r.UnitID == "" {
			rowErrs.AddField(i, "quantity.unit_id", apierror.CodeInvalidFormat, "is required")
		}
		if !in.Reconcile && r.Measure.IsZero() {
			rowErrs.AddField(i, "quantity.value", apierror.CodeInvalidFormat, "an adjustment's quantity can't be 0")
		}
		if in.Reconcile && r.Measure.IsNegative() {
			rowErrs.AddField(i, "quantity.value", apierror.CodeInvalidFormat, "a reconciliation's quantity can't be negative")
		}
		if r.Reason != nil && !r.Reason.IsValid() {
			rowErrs.AddField(i, "reason", apierror.CodeInvalidFormat, "must be damaged, count_correction, found, scrap or other")
		}
	}
	return rowErrs.Summary("movements")
}

// resolveBulkMovementRows resolves each row's item and checks its locations. A SKU or item ID that matches no item
// resolves to no item, and the write reports that row as failed; the rest still apply.
func resolveBulkMovementRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []bulkMovementInput) ([]domain.ResolvedInventoryMovementRow, *apierror.APIError) {
	var skus, ids, locationIDs []string
	var locationRows []int
	for i, in := range rows {
		if in.Row.SKU != nil {
			skus = append(skus, *in.Row.SKU)
		}
		if in.Row.ItemID != nil {
			ids = append(ids, *in.Row.ItemID)
		}
		if in.Row.LocationID != nil {
			locationIDs = append(locationIDs, *in.Row.LocationID)
			locationRows = append(locationRows, i)
		}
	}
	if len(locationIDs) > 0 {
		if apiErr := requireMovementLocations(ctx, repos, accountID, locationIDs, func(i int) string {
			return fmt.Sprintf("movements[%d].location_id", locationRows[i])
		}); apiErr != nil {
			return nil, apiErr
		}
	}

	bySKU := map[string]string{}
	if len(skus) > 0 {
		found, apiErr := repos.NewItemRepo().FetchItemsBySKU(ctx, accountID, uniqueIDs(skus))
		if apiErr != nil {
			return nil, apiErr
		}
		for _, it := range found {
			bySKU[it.SKU] = it.ItemID
		}
	}
	byID := map[string]bool{}
	if len(ids) > 0 {
		found, apiErr := repos.NewItemRepo().GetByIDs(ctx, accountID, uniqueIDs(ids))
		if apiErr != nil {
			return nil, apiErr
		}
		for _, it := range found {
			byID[it.ID] = true
		}
	}

	resolved := make([]domain.ResolvedInventoryMovementRow, len(rows))
	for i, in := range rows {
		movementID, apiErr := id.GenID(id.InventoryMovementIDPrefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		r := domain.ResolvedInventoryMovementRow{
			ID: movementID, Reconcile: in.Reconcile, Measure: in.Row.Measure, UnitID: in.Row.UnitID,
			LocationID: in.Row.LocationID, LotNumber: in.Row.LotNumber, Reason: in.Row.Reason,
		}
		switch {
		case in.Row.SKU != nil:
			r.SKU = *in.Row.SKU
			r.ItemID = bySKU[*in.Row.SKU]
		case in.Row.ItemID != nil && byID[*in.Row.ItemID]:
			r.ItemID = *in.Row.ItemID
		}
		resolved[i] = r
	}
	return resolved, nil
}

// writeBulkMovements writes one chunk under the ledger locks of all its items. Each row runs in its own savepoint and
// measures against what the rows before it left, so a SKU repeated in the chunk applies in order. A row whose movement
// already exists was written by an earlier delivery and is reported again rather than applied twice.
func (s *itemSvcImpl) writeBulkMovements(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedInventoryMovementRow) (BulkWriteResult, *apierror.APIError) {
	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError

	identity, _ := appctx.Identity[*types.Identity](txCtx)
	who, ok := actor.FromIdentity(identity)
	if !ok {
		who = actor.System()
	}
	var legacyUserID *string
	if identity != nil && identity.Actor != nil {
		legacyUserID = &identity.Actor.ID
	}

	var itemIDs []string
	for _, r := range rows {
		if r.ItemID != "" {
			itemIDs = append(itemIDs, r.ItemID)
		}
	}
	scope, apiErr := ledgerlock.Acquire(txCtx, txRepos.NewInventoryMutationRepo(), itemIDs)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	stocking := map[string]*domain.ItemStockingUnit{}
	physical := map[string]decimal.Decimal{}
	for i, r := range rows {
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			if r.ItemID == "" {
				if r.SKU != "" {
					return apierror.NewFieldError("sku", apierror.CodeInvalidFormat, fmt.Sprintf("No item has the SKU %q.", r.SKU))
				}
				return apierror.NewFieldError("item_id", apierror.CodeInvalidFormat, "No item has that ID.")
			}
			if _, apiErr := txRepos.NewInventoryChangeLogRepo().Get(spCtx, accountID, r.ID); apiErr == nil {
				return nil
			} else if apiErr.Code != apierror.CodeResourceNotFound {
				return apiErr
			}
			unit, ok := stocking[r.ItemID]
			if !ok {
				var apiErr *apierror.APIError
				if unit, apiErr = txRepos.NewItemRepo().GetStockingUnit(spCtx, accountID, r.ItemID); apiErr != nil {
					return apiErr
				}
				stocking[r.ItemID] = unit
			}
			if r.UnitID != unit.BaseUnitID && !slices.Contains(unit.AllowedUnitIDs, r.UnitID) {
				return apierror.NewFieldError("quantity.unit_id", apierror.CodeInvalidFormat, "quantity.unit_id must be the item's stocking unit or one of its order units.")
			}
			key := r.ItemID + "|" + r.UnitID
			level, ok := physical[key]
			if !ok {
				var apiErr *apierror.APIError
				if level, apiErr = txRepos.NewInventoryQueryRepo().FetchPhysicalInventory(spCtx, r.ItemID, accountID, r.UnitID); apiErr != nil {
					return apiErr
				}
			}
			next, apiErr := writeMovement(spCtx, txRepos, scope, accountID, movementWrite{
				id: r.ID, itemID: r.ItemID, reconcile: r.Reconcile, measure: r.Measure, unitID: r.UnitID, ownerID: accountID,
				locationID: r.LocationID, lotNumber: r.LotNumber, reason: r.Reason, actor: who, legacyUserID: legacyUserID,
			}, level)
			if apiErr != nil {
				return apiErr
			}
			// Another unit's cached level is now stale; reread it if this item comes up again in that unit.
			for k := range physical {
				if len(k) > len(r.ItemID) && k[:len(r.ItemID)+1] == r.ItemID+"|" {
					delete(physical, k)
				}
			}
			physical[key] = next
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.RowError{Index: i, Error: rowErr.Object()})
			continue
		}
		results = append(results, newRowResult(i, r.ID, true))
	}
	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}

func (s *itemSvcImpl) bulkMovementSpec() bulkOperationSpec[bulkMovementInput, domain.ResolvedInventoryMovementRow] {
	return bulkOperationSpec[bulkMovementInput, domain.ResolvedInventoryMovementRow]{
		JobType:          constants.JobTypeBulkCreate,
		ResourceType:     constants.ObjectTypeInventoryMovement,
		RoutingKey:       messaging.BulkCreateInventoryMovements.RoutingKey(),
		PermissionDomain: types.PermissionDomainItems,
		Actions:          []types.Action{types.ActionCreate},
		EntityName:       "movements",
		Validate:         validateBulkMovementRows,
		Resolve:          resolveBulkMovementRows,
		AcceptResults: func(resolved []domain.ResolvedInventoryMovementRow) []domain.RowResult {
			out := make([]domain.RowResult, 0, len(resolved))
			for i, r := range resolved {
				if r.ItemID != "" {
					out = append(out, newRowResult(i, r.ID, true))
				}
			}
			return out
		},
		Write:     s.writeBulkMovements,
		ChunkSize: bulkMovementChunkSize,
	}
}

// BulkCreateInventoryMovements validates and resolves the rows, records them on a job and returns the job to poll.
func (s *itemSvcImpl) BulkCreateInventoryMovements(ctx context.Context, params domain.BulkCreateInventoryMovementsParams) (*domain.Job, *apierror.APIError) {
	rows := make([]bulkMovementInput, len(params.Rows))
	for i, r := range params.Rows {
		rows[i] = bulkMovementInput{Row: r, Reconcile: params.Reconcile}
	}
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkMovementSpec(), rows)
}

// ExecuteBulkCreateInventoryMovements writes an accepted bulk movement job.
func (s *itemSvcImpl) ExecuteBulkCreateInventoryMovements(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkMovementSpec(), event)
}

// stockedItemTypes are the item types inventory levels report on; services are never stocked.
var stockedItemTypes = []string{string(constants.ItemTypeCodeMaterial), string(constants.ItemTypeCodePart), string(constants.ItemTypeCodeProduct)}

// ListInventoryLevels returns a page of stocked items with their stock positions, at the given locations or sites
// when named, now or as last logged at an instant.
func (s *itemSvcImpl) ListInventoryLevels(ctx context.Context, params domain.ListInventoryLevelsParams) (*domain.ListInventoryLevelsResult, *apierror.APIError) {
	ctx, span := itemSvcTracer.Start(ctx, "service.item.list_inventory_levels")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
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

	if params.AsOf != nil && (len(params.LocationIDs) > 0 || len(params.SiteIDs) > 0) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("as_of", "as_of can't be combined with location_ids or site_ids: past levels are logged per item, not per location."))
	}

	locationIDs := uniqueIDs(params.LocationIDs)
	if len(params.SiteIDs) > 0 {
		inSites, apiErr := s.mediators().Site.LocationIDsInSites(ctx, accountID, uniqueIDs(params.SiteIDs))
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(locationIDs) > 0 {
			locationIDs = intersectIDs(locationIDs, inSites)
		} else {
			locationIDs = inSites
		}
		if len(locationIDs) == 0 {
			// The sites hold no locations of the account, so nothing is stocked there; an empty filter would read all.
			locationIDs = []string{""}
		}
	}

	page, apiErr := s.repos.NewItemRepo().List(ctx, domain.ListItemsParams{
		AccountID: accountID,
		Cursor:    params.Cursor,
		Limit:     params.Limit,
		Types:     stockedItemTypes,
		IDs:       uniqueIDs(params.ItemIDs),
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	itemIDs := make([]string, len(page.Items))
	for i, it := range page.Items {
		itemIDs[i] = it.ID
	}

	var levels map[string]*domain.StockLevel
	if params.AsOf != nil {
		levels, apiErr = s.repos.NewInventoryQueryRepo().FetchStockLevelsAsOf(ctx, accountID, itemIDs, *params.AsOf)
	} else {
		levels, apiErr = s.repos.NewInventoryQueryRepo().FetchStockLevels(ctx, accountID, itemIDs, locationIDs)
	}
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// A level names its location only when it covers exactly one; otherwise it's the sum over the locations asked for.
	var location *string
	if len(params.LocationIDs) == 1 && len(params.SiteIDs) == 0 {
		location = &params.LocationIDs[0]
	}
	out := make([]*domain.StockLevel, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		if l, ok := levels[itemID]; ok {
			l.LocationID = location
			out = append(out, l)
		}
	}
	return &domain.ListInventoryLevelsResult{Levels: out, PageInfo: page.PageInfo}, nil
}

// intersectIDs is the ids in a that are also in b, in a's order.
func intersectIDs(a, b []string) []string {
	var out []string
	for _, v := range a {
		if slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}
