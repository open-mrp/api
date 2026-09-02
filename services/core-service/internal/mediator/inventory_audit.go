package mediator

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryAuditTracer = tracing.GetTracer("core-service.inventory_audit")

// RecordInventoryAuditTrail records an inventory change in the audit logs after stock is mutated. It first gets the item’s current physical inventory level, then passes that along with the change amount and metadata to the detailed audit-writing function.
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

// RecordInventoryAuditTrailWithLevel writes the inventory snapshot and change log using a level that was already calculated by the caller. This avoids doing another inventory lookup for each item and can recalculate burn rate when the change represents consumption.
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

	if apiErr := MaybeRecalculateAfterConsumption(ctx, repos, accountID, itemID, delta, actionType); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
