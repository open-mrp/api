package mediator

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryAuditTracer = tracing.GetTracer("core-service.inventory_audit")

// RecordInventoryAuditTrail writes inventory_log and inventory_change_log after a mutation and recalculates burn rate when the change represents consumption.
func RecordInventoryAuditTrail(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID string,
	delta decimal.Decimal,
	unitID, actionType string,
	scanningStationID *string,
	responsibleUserID *string,
) *apierror.APIError {
	ctx, span := inventoryAuditTracer.Start(ctx, "mediator.inventory_audit.record_trail")
	defer span.End()

	if delta.IsZero() {
		return nil
	}

	invQueryRepo := repos.NewInventoryQueryRepo()
	// Logged in the unit the movement was recorded in, so the level and the change agree.
	currentPhysical, apiErr := invQueryRepo.FetchPhysicalInventory(ctx, itemID, accountID, unitID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return RecordInventoryAuditTrailWithLevel(ctx, repos, accountID, itemID, delta, currentPhysical, unitID, actionType, scanningStationID, responsibleUserID)
}

// RecordInventoryAuditTrailWithLevel is RecordInventoryAuditTrail with the post-mutation level handed
// in rather than fetched. The batch-scan consumer computes every audited item's level in one batched
// read after all of a scan's mutations are written and passes it here, so a scan that audits N items
// no longer runs N ledger aggregations. RecordInventoryAuditTrail fetches the level itself and calls
// through to this, so its callers are unchanged.
func RecordInventoryAuditTrailWithLevel(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID string,
	delta decimal.Decimal,
	level decimal.Decimal,
	unitID, actionType string,
	scanningStationID *string,
	responsibleUserID *string,
) *apierror.APIError {
	ctx, span := inventoryAuditTracer.Start(ctx, "mediator.inventory_audit.record_trail_with_level")
	defer span.End()

	if delta.IsZero() {
		return nil
	}

	invMutRepo := repos.NewInventoryMutationRepo()

	if apiErr := invMutRepo.CreateInventoryLog(ctx, domain.CreateInventoryLogParams{
		AccountID: accountID,
		ItemID:    itemID,
		Measure:   level,
		UnitID:    unitID,
	}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := invMutRepo.CreateInventoryChangeLog(ctx, domain.CreateInventoryChangeLogParams{
		AccountID:         accountID,
		ItemID:            itemID,
		Measure:           delta,
		UnitID:            unitID,
		ActionType:        actionType,
		ScanningStationID: scanningStationID,
		ResponsibleUserID: responsibleUserID,
	}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	MaybeRecalculateAfterConsumption(ctx, repos, accountID, itemID, delta, actionType)

	return nil
}

// RecordInventoryAuditTrailOrLog traces errors from RecordInventoryAuditTrail without failing the caller.
func RecordInventoryAuditTrailOrLog(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID string,
	delta decimal.Decimal,
	unitID, actionType string,
	scanningStationID *string,
	responsibleUserID *string,
) {
	if apiErr := RecordInventoryAuditTrail(ctx, repos, accountID, itemID, delta, unitID, actionType, scanningStationID, responsibleUserID); apiErr != nil {
		_, span := inventoryAuditTracer.Start(ctx, "mediator.inventory_audit.record_trail_failed")
		tracing.Trace(span, apiErr)
		span.End()
	}
}
