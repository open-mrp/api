package mediator

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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

	return RecordInventoryMovementWithLevel(ctx, repos, accountID, itemID, delta, currentPhysical, unitID, actionType, scanningStationID, responsibleUserID, domain.MovementFields{})
}

// RecordInventoryMovement is RecordInventoryAuditTrail for a caller that knows what the movement was: its type, actor,
// location, lot, reason, source document or cost. Fields left zero default as RecordInventoryAuditTrail's do.
func RecordInventoryMovement(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID string,
	delta decimal.Decimal,
	unitID, actionType string,
	scanningStationID *string,
	responsibleUserID *string,
	movement domain.MovementFields,
) *apierror.APIError {
	ctx, span := inventoryAuditTracer.Start(ctx, "mediator.inventory_audit.record_movement")
	defer span.End()

	if delta.IsZero() {
		return nil
	}
	currentPhysical, apiErr := repos.NewInventoryQueryRepo().FetchPhysicalInventory(ctx, itemID, accountID, unitID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return RecordInventoryMovementWithLevel(ctx, repos, accountID, itemID, delta, currentPhysical, unitID, actionType, scanningStationID, responsibleUserID, movement)
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
	return RecordInventoryMovementWithLevel(ctx, repos, accountID, itemID, delta, level, unitID, actionType, scanningStationID, responsibleUserID, domain.MovementFields{})
}

// RecordInventoryMovementWithLevel is RecordInventoryAuditTrailWithLevel with the movement's own fields.
func RecordInventoryMovementWithLevel(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID string,
	delta decimal.Decimal,
	level decimal.Decimal,
	unitID, actionType string,
	scanningStationID *string,
	responsibleUserID *string,
	movement domain.MovementFields,
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
		Movement:          movement,
	}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := MaybeRecalculateAfterConsumption(ctx, repos, accountID, itemID, delta, actionType); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
