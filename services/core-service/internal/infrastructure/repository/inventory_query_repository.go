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

	// Decimal from the column through to the caller. It used to arrive as an int64 because the query
	// cast the total to SIGNED, which rounded away the half-units an item stocked in pairs and drawn
	// on in each ends up holding.
	atp, parseErr := decimal.NewFromString(row.AvailableToPromise)
	if parseErr != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid available-to-promise value."))
	}

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
		// The query returns DECIMAL rather than the SIGNED it used to, so a level of 60.5 pairs
		// survives the trip instead of arriving as 60. Parsed as a decimal and narrowed once here
		// rather than scanned straight into a float: the list column is a display figure, but the
		// rounding that produced it should happen where it can be seen.
		onHand, parseErr := decimal.NewFromString(row.OnHandQuantity)
		if parseErr != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Invalid on-hand quantity value."))
		}
		measure, _ := onHand.Float64()

		result[i] = &domain.BulkOnHandInventory{
			ItemID:           row.ItemID,
			OnHandQuantity:   measure,
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
// recognizing there is nothing to do.
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
// allocations and normalizing every row through its own unit's ratio exactly as FetchPhysicalInventory
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
