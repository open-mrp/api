package event

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryAuditCollectorTracer = tracing.GetTracer("core-service.inventory_audit_collector")

// updateInventoryWithAudit moves inventory and records the movement on the audit trail. Takes the
// repo factory rather than hanging off a consumer so the batch-scanned and execute-production-step
// consumers share one implementation while the latter is being retired.
func updateInventoryWithAudit(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID string,
	params domain.InventoryUpdateParams,
) *apierror.APIError {
	inventoryRepo := repos.NewInventoryMutationRepo()
	if apiErr := inventoryRepo.UpdateInventory(ctx, params); apiErr != nil {
		return apiErr
	}

	var scanningStationID *string
	if params.ScanningStationID != "" {
		id := params.ScanningStationID
		scanningStationID = &id
	}

	mediator.RecordInventoryAuditTrailOrLog(
		ctx,
		repos,
		accountID,
		params.ItemID,
		params.Measure,
		params.UnitID,
		params.ActionType,
		scanningStationID,
		params.ResponsibleUserID,
	)
	return nil
}

func (c *ExecuteProductionStepConsumer) updateInventoryWithAudit(
	ctx context.Context,
	accountID string,
	params domain.InventoryUpdateParams,
) *apierror.APIError {
	return updateInventoryWithAudit(ctx, c.repos, accountID, params)
}

// pendingInventoryAudit is one movement waiting to be logged. Its level is not known yet: it is the
// item's physical inventory after every mutation the scan makes, which the collector reads in one
// batched query at the end of the scan rather than re-aggregating the ledger per movement.
type pendingInventoryAudit struct {
	itemID            string
	delta             decimal.Decimal
	unitID            string
	actionType        string
	scanningStationID *string
	responsibleUserID *string
}

// inventoryAuditCollector defers the audit trail of a scan's inventory movements so their levels can
// be computed together. The old path fetched each movement's level inline, one full ledger
// aggregation per audited item, which on a large account was the bulk of a scan's transaction and
// pushed it past the 20s limit. The collector records the movements as they happen, then finalize
// levels them all from a single batched read taken once every mutation is written.
type inventoryAuditCollector struct {
	pending []pendingInventoryAudit
}

// mutate applies a movement and records it for later levelling. The mutation itself must not fail
// silently, so its error is returned; the audit trail is best-effort, matching updateInventoryWithAudit.
func (col *inventoryAuditCollector) mutate(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID string,
	params domain.InventoryUpdateParams,
) *apierror.APIError {
	if apiErr := repos.NewInventoryMutationRepo().UpdateInventory(ctx, params); apiErr != nil {
		return apiErr
	}

	if params.Measure.IsZero() {
		return nil
	}

	var scanningStationID *string
	if params.ScanningStationID != "" {
		id := params.ScanningStationID
		scanningStationID = &id
	}

	col.pending = append(col.pending, pendingInventoryAudit{
		itemID:            params.ItemID,
		delta:             params.Measure,
		unitID:            params.UnitID,
		actionType:        params.ActionType,
		scanningStationID: scanningStationID,
		responsibleUserID: params.ResponsibleUserID,
	})
	return nil
}

// finalize levels every recorded movement and writes the audit trail. It runs after all of a scan's
// mutations, so each item's level is its final physical inventory for the scan — endpoint-consistent
// with what the item's inventory query would return now. It is best-effort throughout, as the inline
// path was: a failure to level or log is traced, never surfaced to fail the scan.
//
// An item moved more than once in a single scan gets that same final level on each of its entries;
// the intermediate levels the inline path recorded are not reproduced, which no consumer depends on.
func (col *inventoryAuditCollector) finalize(ctx context.Context, repos domain.RepoFactory, accountID string) {
	if len(col.pending) == 0 {
		return
	}

	ctx, span := inventoryAuditCollectorTracer.Start(ctx, "event.inventory_audit_collector.finalize")
	defer span.End()

	itemSeen := make(map[string]struct{}, len(col.pending))
	itemIDs := make([]string, 0, len(col.pending))
	unitSeen := make(map[string]struct{}, len(col.pending))
	unitIDs := make([]string, 0, len(col.pending))
	for _, p := range col.pending {
		if _, ok := itemSeen[p.itemID]; !ok {
			itemSeen[p.itemID] = struct{}{}
			itemIDs = append(itemIDs, p.itemID)
		}
		if p.unitID == "" {
			continue
		}
		if _, ok := unitSeen[p.unitID]; !ok {
			unitSeen[p.unitID] = struct{}{}
			unitIDs = append(unitIDs, p.unitID)
		}
	}

	baseByItem, apiErr := repos.NewInventoryQueryRepo().FetchPhysicalInventoryBaseForItems(ctx, accountID, itemIDs)
	if apiErr != nil {
		tracing.Trace(span, apiErr)
		return
	}

	factorsByUnit, apiErr := repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, unitIDs)
	if apiErr != nil {
		tracing.Trace(span, apiErr)
		return
	}

	for _, p := range col.pending {
		base := baseByItem[p.itemID]
		level := levelInUnit(base, factorsByUnit, p.unitID)
		if apiErr := mediator.RecordInventoryAuditTrailWithLevel(
			ctx,
			repos,
			accountID,
			p.itemID,
			p.delta,
			level,
			p.unitID,
			p.actionType,
			p.scanningStationID,
			p.responsibleUserID,
		); apiErr != nil {
			tracing.Trace(span, apiErr)
		}
	}
}

// levelInUnit expresses a base-unit physical inventory in unitID, dividing by the unit's ratio exactly
// as FetchPhysicalInventoryForItem does. An unknown unit, a missing ratio denominator or a zero ratio
// all leave the figure in base units, matching that query's COALESCE(NULLIF(ratio, 0), 1).
func levelInUnit(base decimal.Decimal, factorsByUnit map[string]domain.UnitFactors, unitID string) decimal.Decimal {
	factors, ok := factorsByUnit[unitID]
	if !ok {
		return base
	}
	if factors.RatioDen.IsZero() {
		return base
	}
	ratio := factors.RatioNum.Div(factors.RatioDen)
	if ratio.IsZero() {
		return base
	}
	return base.DivRound(ratio, 30)
}
