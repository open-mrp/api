package mediator

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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
	currentPhysical, apiErr := invQueryRepo.FetchPhysicalInventory(ctx, itemID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	finalQty := decimal.NewFromFloat(currentPhysical)
	invMutRepo := repos.NewInventoryMutationRepo()

	if apiErr := invMutRepo.CreateInventoryLog(ctx, domain.CreateInventoryLogParams{
		AccountID: accountID,
		ItemID:    itemID,
		Measure:   finalQty,
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

	meds := NewMediatorFactory().Build(repos)
	MaybeRecalculateAfterConsumption(ctx, meds, accountID, itemID, delta, actionType)

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
