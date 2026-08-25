package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryQueryRepoTracer = tracing.GetTracer("core-service.inventory_query_repository")

type inventoryQueryRepoImpl struct {
	queries *sqlc.Queries
}

func NewInventoryQueryRepo(queries *sqlc.Queries) domain.InventoryQueryRepo {
	return &inventoryQueryRepoImpl{queries: queries}
}

func (r *inventoryQueryRepoImpl) FetchCurrentInventory(ctx context.Context, itemID, ownerAccountID string) (*domain.InventorySnapshot, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_current_inventory")
	defer span.End()

	row, err := r.queries.FetchCurrentInventoryForItem(ctx, sqlc.FetchCurrentInventoryForItemParams{
		ItemID:         itemID,
		OwnerAccountID: ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	atp := decimal.NewFromInt(int64(row.AvailableToPromise))

	var unitAbbreviation string
	if row.UnitAbbreviation != nil {
		unitAbbreviation = fmt.Sprintf("%v", row.UnitAbbreviation)
	}

	return &domain.InventorySnapshot{
		AvailableToPromiseMeasure:          atp,
		AvailableToPromiseUnitAbbreviation: unitAbbreviation,
	}, nil
}

func (r *inventoryQueryRepoImpl) FetchOnHandInventoryBulk(ctx context.Context, itemIDs []string, ownerAccountID string) ([]*domain.BulkOnHandInventory, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_on_hand_inventory_bulk")
	defer span.End()

	rows, err := r.queries.FetchOnHandInventoryBulk(ctx, sqlc.FetchOnHandInventoryBulkParams{
		ItemIds:        itemIDs,
		OwnerAccountID: ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]*domain.BulkOnHandInventory, len(rows))
	for i, row := range rows {
		result[i] = &domain.BulkOnHandInventory{
			ItemID:           row.ItemID,
			OnHandQuantity:   float64(row.OnHandQuantity),
			UnitID:           row.UnitID,
			UnitAbbreviation: row.UnitAbbreviation,
			UnitType:         row.UnitType,
		}
	}

	return result, nil
}

// FetchPhysicalInventory returns the level as a decimal, never a float.
//
// The ledger is decimal from the column to the arithmetic, and a round trip through float64 in the
// middle of it does not survive contact with a reconcile: 9959.03214 comes back 9959.032140000001,
// and a correction to the figure already on screen writes a receipt of 0.000000000003 instead of
// recognising there is nothing to do.
func (r *inventoryQueryRepoImpl) FetchPhysicalInventory(ctx context.Context, itemID, ownerAccountID, unitID string) (decimal.Decimal, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_physical_inventory")
	defer span.End()

	physicalInv, err := r.queries.FetchPhysicalInventoryForItem(ctx, sqlc.FetchPhysicalInventoryForItemParams{
		ItemID:         itemID,
		OwnerAccountID: ownerAccountID,
		UnitID:         unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return decimal.Zero, tracing.Trace(span, apiErr)
	}

	measure, parseErr := decimal.NewFromString(physicalInv)
	if parseErr != nil {
		return decimal.Zero, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid physical inventory value."))
	}
	return measure, nil
}

// FetchPhysicalInventoryBaseForItems returns each item's physical inventory in base units, netting
// allocations and normalising every row through its own unit's ratio exactly as FetchPhysicalInventory
// does, but for many items in one query. The target-unit divide is left to the caller, which applies
// it per event. Items with no receipts or issues are absent from the map.
func (r *inventoryQueryRepoImpl) FetchPhysicalInventoryBaseForItems(ctx context.Context, accountID string, itemIDs []string) (map[string]decimal.Decimal, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_physical_inventory_base_for_items")
	defer span.End()

	out := make(map[string]decimal.Decimal, len(itemIDs))
	if len(itemIDs) == 0 {
		return out, nil
	}

	rows, err := r.queries.FetchPhysicalInventoryBaseForItems(ctx, sqlc.FetchPhysicalInventoryBaseForItemsParams{
		AccountID: accountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, row := range rows {
		measure, parseErr := decimal.NewFromString(row.PhysicalBase)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid physical inventory value."))
		}
		out[row.ItemID] = measure
	}
	return out, nil
}
