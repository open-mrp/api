package repository

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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

func (r *inventoryQueryRepoImpl) FetchPhysicalInventory(ctx context.Context, itemID, ownerAccountID string) (float64, *apierror.APIError) {
	ctx, span := inventoryQueryRepoTracer.Start(ctx, "repository.inventory_query.fetch_physical_inventory")
	defer span.End()

	physicalInv, err := r.queries.FetchPhysicalInventoryForItem(ctx, sqlc.FetchPhysicalInventoryForItemParams{
		ItemID:         itemID,
		OwnerAccountID: ownerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return float64(physicalInv), nil
}
