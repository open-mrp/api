package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
)

// maxSetOrderUnitsItems bounds how many items one Set Order Units job writes.
const maxSetOrderUnitsItems = 1000

// setOrderUnitsChunkSize is how many items one Set Order Units transaction writes. Each item costs a handful of
// short statements, so a chunk commits in about a second.
const setOrderUnitsChunkSize = 100

// SetItemOrderUnits validates the request, matches the items and records them on a job to poll.
func (s *itemSvcImpl) SetItemOrderUnits(ctx context.Context, params domain.SetOrderUnitsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.setOrderUnitsSpec(), []domain.SetOrderUnitsParams{params})
}

// ExecuteSetItemOrderUnits writes an accepted Set Order Units job.
func (s *itemSvcImpl) ExecuteSetItemOrderUnits(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.setOrderUnitsSpec(), event)
}

func (s *itemSvcImpl) setOrderUnitsSpec() bulkOperationSpec[domain.SetOrderUnitsParams, domain.ResolvedSetOrderUnitsRow] {
	return bulkOperationSpec[domain.SetOrderUnitsParams, domain.ResolvedSetOrderUnitsRow]{
		JobType:          constants.JobTypeSetOrderUnits,
		ResourceType:     constants.ObjectTypeItem,
		RoutingKey:       messaging.SetItemOrderUnits.RoutingKey(),
		PermissionDomain: types.PermissionDomainItems,
		Actions:          []types.Action{types.ActionUpdate},
		EntityName:       "items",
		Validate:         validateSetOrderUnits,
		Resolve:          resolveSetOrderUnits,
		Write:            writeSetOrderUnits,
		ChunkSize:        setOrderUnitsChunkSize,
	}
}

// validateSetOrderUnits checks the request's shape: some filter, and each order unit named once.
func validateSetOrderUnits(rows []domain.SetOrderUnitsParams) *apierror.APIError {
	p := rows[0]
	f := p.Filter
	if len(f.ItemIDs)+len(f.ProductLineIDs)+len(f.CategoryIDs)+len(f.AttributeIDs) == 0 {
		return apierror.NewFieldError("filter", apierror.CodeMissingField,
			"Name the items with at least one of filter.item_ids, filter.product_line_ids, filter.category_ids or filter.attribute_ids.")
	}
	seen := make(map[string]bool, len(p.OrderUnits))
	for i, ou := range p.OrderUnits {
		param := fmt.Sprintf("order_units[%d]", i)
		if ou.UnitID == "" {
			return apierror.NewFieldError(param+".unit_id", apierror.CodeMissingField, "unit_id is required.")
		}
		if seen[ou.UnitID] {
			return apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "Each unit can be listed once.")
		}
		seen[ou.UnitID] = true
		if ou.PortalVisibility != "" && !ou.PortalVisibility.IsValid() {
			return apierror.NewFieldError(param+".portal_visibility", apierror.CodeInvalidFormat, "portal_visibility must be visible or hidden.")
		}
	}
	return nil
}

// resolveSetOrderUnits checks every named unit and item exists and matches the items. A unit that doesn't suit one of
// the items (another dimension, or its stocking unit) fails that item's row when the job runs.
func resolveSetOrderUnits(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.SetOrderUnitsParams) ([]domain.ResolvedSetOrderUnitsRow, *apierror.APIError) {
	p := rows[0]
	unitIDs := make([]string, 0, len(p.OrderUnits)+1)
	for _, ou := range p.OrderUnits {
		unitIDs = append(unitIDs, ou.UnitID)
	}
	if p.DefaultOrderUnitID != nil {
		unitIDs = append(unitIDs, *p.DefaultOrderUnitID)
	}
	factors, apiErr := repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, unitIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	for i, ou := range p.OrderUnits {
		if _, ok := factors[ou.UnitID]; !ok {
			return nil, apierror.NewFieldError(fmt.Sprintf("order_units[%d].unit_id", i), apierror.CodeInvalidFormat, "The unit does not exist.")
		}
	}
	if p.DefaultOrderUnitID != nil {
		if _, ok := factors[*p.DefaultOrderUnitID]; !ok {
			return nil, apierror.NewFieldError("default_order_unit_id", apierror.CodeInvalidFormat, "The unit does not exist.")
		}
	}

	matched, apiErr := repos.NewItemUnitsRepo().MatchItems(ctx, accountID, p.Filter, maxSetOrderUnitsItems+1)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(matched) > maxSetOrderUnitsItems {
		return nil, apierror.NewFieldError("filter", apierror.CodeInvalidFormat,
			fmt.Sprintf("The filter matches more than %d items; narrow it and send several requests.", maxSetOrderUnitsItems))
	}
	if len(p.Filter.ItemIDs) > 0 {
		found := make(map[string]bool, len(matched))
		all, apiErr := repos.NewItemUnitsRepo().MatchItems(ctx, accountID, domain.SetOrderUnitsFilter{ItemIDs: p.Filter.ItemIDs}, len(p.Filter.ItemIDs))
		if apiErr != nil {
			return nil, apiErr
		}
		for _, m := range all {
			found[m.ID] = true
		}
		for i, id := range p.Filter.ItemIDs {
			if !found[id] {
				return nil, apierror.NewFieldError(fmt.Sprintf("filter.item_ids[%d]", i), apierror.CodeInvalidFormat, fmt.Sprintf("Item %q was not found.", id))
			}
		}
	}

	out := make([]domain.ResolvedSetOrderUnitsRow, len(matched))
	for i, m := range matched {
		out[i] = domain.ResolvedSetOrderUnitsRow{
			ItemID:             m.ID,
			ItemType:           m.Type,
			OrderUnits:         p.OrderUnits,
			DefaultOrderUnitID: p.DefaultOrderUnitID,
		}
	}
	if len(out) == 0 {
		return nil, apierror.NewFieldError("filter", apierror.CodeInvalidFormat, "No item matches the filter.")
	}
	return out, nil
}

// writeSetOrderUnits replaces each matched item's order units in its own savepoint, so an item they don't suit fails
// alone. An item that already has the requested units is reported without being written again, so a job re-run after
// failing part-way skips the chunks it already committed.
func writeSetOrderUnits(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedSetOrderUnitsRow) (BulkWriteResult, *apierror.APIError) {
	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i, row := range rows {
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			before, apiErr := readItem(spCtx, txRepos, accountID, row.ItemID)
			if apiErr != nil {
				return apiErr
			}
			orderUnits := append([]domain.ItemOrderUnit{}, row.OrderUnits...)
			in := domain.ItemUnitsInput{OrderUnits: &orderUnits, DefaultOrderUnitID: row.DefaultOrderUnitID}
			if in.DefaultOrderUnitID == nil && row.ItemType != constants.ItemTypeCodeService && before.Units != nil {
				stocking := before.Units.StockingUnitID
				in.DefaultOrderUnitID = &stocking
			}
			plan, apiErr := planItemUnits(spCtx, txRepos, accountID, row.ItemType, before.Units, in, "")
			if apiErr != nil {
				return apiErr
			}
			if itemUnitsUnchanged(before.Units, plan) {
				return nil
			}
			if apiErr := storeItemUnits(spCtx, txRepos, accountID, row.ItemID, plan); apiErr != nil {
				return apiErr
			}
			return publishItemUnitsAudit(spCtx, txRepos, accountID, before, nil)
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.RowError{Index: i, Error: rowErr.Object()})
			continue
		}
		results = append(results, newRowResult(i, row.ItemID, false))
	}
	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}

// itemUnitsUnchanged reports whether storing plan would leave the item's stored units as they are. Units still
// derived from unit groups are never unchanged: storing them is the write.
func itemUnitsUnchanged(current, plan *domain.ItemUnits) bool {
	return current != nil && current.Stored && len(audit.ComputeChanges(current, plan)) == 0
}

// publishItemUnitsAudit records an update event for the item's changes since before.
func publishItemUnitsAudit(ctx context.Context, repos domain.RepoFactory, accountID string, before *domain.Item, metadata map[string]any) *apierror.APIError {
	after, apiErr := readItem(ctx, repos, accountID, before.ID)
	if apiErr != nil {
		return apiErr
	}
	changes := audit.ComputeChanges(before, after)
	if len(changes) == 0 {
		return nil
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeItem,
		ResourceID:   before.ID,
		Changes:      changes,
		Metadata:     metadata,
	})
}

// ChangeItemStockingUnit validates the change and records it on a job to poll.
func (s *itemSvcImpl) ChangeItemStockingUnit(ctx context.Context, params domain.ChangeStockingUnitParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.changeStockingUnitSpec(), []domain.ChangeStockingUnitParams{params})
}

// ExecuteChangeItemStockingUnit restates the item of an accepted Change Stocking Unit job.
func (s *itemSvcImpl) ExecuteChangeItemStockingUnit(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.changeStockingUnitSpec(), event)
}

func (s *itemSvcImpl) changeStockingUnitSpec() bulkOperationSpec[domain.ChangeStockingUnitParams, domain.ResolvedChangeStockingUnitRow] {
	return bulkOperationSpec[domain.ChangeStockingUnitParams, domain.ResolvedChangeStockingUnitRow]{
		JobType:          constants.JobTypeChangeStockingUnit,
		ResourceType:     constants.ObjectTypeItem,
		RoutingKey:       messaging.ChangeItemStockingUnit.RoutingKey(),
		PermissionDomain: types.PermissionDomainItems,
		Actions:          []types.Action{types.ActionUpdate},
		EntityName:       "items",
		Validate:         validateChangeStockingUnit,
		Resolve:          resolveChangeStockingUnit,
		Write:            writeChangeStockingUnit,
	}
}

func validateChangeStockingUnit(rows []domain.ChangeStockingUnitParams) *apierror.APIError {
	p := rows[0]
	if p.StockingUnitID == "" {
		return apierror.NewFieldError("stocking_unit_id", apierror.CodeMissingField, "stocking_unit_id is required.")
	}
	if !p.Mode.IsValid() {
		return apierror.NewFieldError("mode", apierror.CodeInvalidFormat, "mode must be convert or relabel.")
	}
	return nil
}

// resolveChangeStockingUnit checks the item is stocked, the new unit exists and differs from the current one, and
// that a conversion stays in the same dimension.
func resolveChangeStockingUnit(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.ChangeStockingUnitParams) ([]domain.ResolvedChangeStockingUnitRow, *apierror.APIError) {
	p := rows[0]
	item, apiErr := readItem(ctx, repos, accountID, p.ItemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if constants.ItemTypeCode(item.ItemTypeCode) == constants.ItemTypeCodeService || item.Units == nil || item.Units.StockingUnitID == "" {
		return nil, apierror.NewFieldError("stocking_unit_id", apierror.CodeInvalidFormat, "A service is never stocked, so it has no stocking unit to change.")
	}
	current := item.Units.StockingUnitID
	if current == p.StockingUnitID {
		return nil, apierror.NewFieldError("stocking_unit_id", apierror.CodeInvalidFormat, "The item is already stocked in this unit.")
	}
	factors, apiErr := repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, []string{current, p.StockingUnitID})
	if apiErr != nil {
		return nil, apiErr
	}
	to, ok := factors[p.StockingUnitID]
	if !ok {
		return nil, apierror.NewFieldError("stocking_unit_id", apierror.CodeInvalidFormat, "The unit does not exist.")
	}
	if to.DimensionCode == string(constants.UnitTypeCurrency) {
		return nil, apierror.NewFieldError("stocking_unit_id", apierror.CodeInvalidFormat, "An item can't be stocked in a currency.")
	}
	if p.Mode == domain.ChangeStockingUnitModeConvert && factors[current].DimensionCode != to.DimensionCode {
		return nil, apierror.NewFieldError("stocking_unit_id", apierror.CodeInvalidFormat,
			"convert keeps the dimension: the new unit must measure "+factors[current].DimensionCode+". Use relabel to fix a unit entered in error.")
	}
	return []domain.ResolvedChangeStockingUnitRow{{ItemID: item.ID, StockingUnitID: p.StockingUnitID, Mode: p.Mode}}, nil
}

// writeChangeStockingUnit restates the item. convert scales what was recorded in the old unit by the units' ratio and
// keeps the old unit orderable; relabel keeps the numbers, and drops the order units that measure another dimension
// than the new unit, reporting them on the result.
func writeChangeStockingUnit(txCtx context.Context, txRepos domain.RepoFactory, _ db.SavepointRunner, accountID string, rows []domain.ResolvedChangeStockingUnitRow) (BulkWriteResult, *apierror.APIError) {
	row := rows[0]
	before, apiErr := readItem(txCtx, txRepos, accountID, row.ItemID)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	units := before.Units
	from := units.StockingUnitID
	if from == row.StockingUnitID {
		return BulkWriteResult{Results: []domain.RowResult{newRowResult(0, row.ItemID, false)}, WrittenIDs: []string{row.ItemID}}, nil
	}

	ids := []string{from, row.StockingUnitID}
	for _, ou := range units.OrderUnits {
		ids = append(ids, ou.UnitID)
	}
	factors, apiErr := txRepos.NewUnitConversionRepo().GetUnitFactors(txCtx, accountID, ids)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	factor := decimal.NewFromInt(1)
	if row.Mode == domain.ChangeStockingUnitModeConvert {
		fromBase := factors[from].ToBase(decimal.NewFromInt(1))
		toBase := factors[row.StockingUnitID].ToBase(decimal.NewFromInt(1))
		if toBase.IsZero() {
			return BulkWriteResult{}, apierror.NewFieldError("stocking_unit_id", apierror.CodeInvalidFormat, "The unit has no ratio to convert by.")
		}
		factor = fromBase.Div(toBase)
	}
	restated, apiErr := txRepos.NewStockingUnitRestateRepo().Restate(txCtx, accountID, row.ItemID, from, row.StockingUnitID, factor)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	dimension := factors[row.StockingUnitID].DimensionCode
	orderUnits := make([]domain.ItemOrderUnit, 0, len(units.OrderUnits)+1)
	var removed []string
	for _, ou := range units.OrderUnits {
		switch {
		case ou.UnitID == row.StockingUnitID:
		case factors[ou.UnitID].DimensionCode != dimension:
			removed = append(removed, ou.UnitID)
		default:
			orderUnits = append(orderUnits, ou)
		}
	}
	if row.Mode == domain.ChangeStockingUnitModeConvert {
		orderUnits = append(orderUnits, domain.ItemOrderUnit{UnitID: from, PortalVisibility: constants.CustomerPortalVisibilityVisible})
	}
	defaultUnit := units.DefaultOrderUnitID
	if defaultUnit == from {
		defaultUnit = row.StockingUnitID
	}
	plan := &domain.ItemUnits{StockingUnitID: row.StockingUnitID, DefaultOrderUnitID: defaultUnit, OrderUnits: orderUnits}
	if !plan.Allows(plan.DefaultOrderUnitID) {
		plan.DefaultOrderUnitID = row.StockingUnitID
	}
	if apiErr := storeItemUnits(txCtx, txRepos, accountID, row.ItemID, plan); apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	if apiErr := publishItemUnitsAudit(txCtx, txRepos, accountID, before, map[string]any{
		"mode": string(row.Mode), "from_unit_id": from, "rows_restated": restated,
	}); apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	result := newRowResult(0, row.ItemID, false)
	result.SubResources = domain.NewSubResourceRefs(constants.ObjectTypeUnit, removed)
	return BulkWriteResult{Results: []domain.RowResult{result}, WrittenIDs: []string{row.ItemID}}, nil
}
