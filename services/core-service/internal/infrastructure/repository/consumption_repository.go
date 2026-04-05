package repository

import (
	"context"
	"database/sql"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var consumptionRepoTracer = tracing.GetTracer("core-service.consumption_repository")

type consumptionRepoImpl struct {
	queries *sqlc.Queries
}

func NewConsumptionRepo(queries *sqlc.Queries) domain.ConsumptionRepo {
	return &consumptionRepoImpl{queries: queries}
}

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func ptrToNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{}
}

func mapGetConsumptionRow(row sqlc.GetConsumptionRow, productionStepID string) *domain.Consumption {
	return &domain.Consumption{
		ID:              row.ID,
		ItemID:          row.ItemID,
		ItemSKU:         row.ItemSku,
		ItemDescription: nullStringToPtr(row.ItemDescription),
		ItemTypeCode:    row.ItemTypeCode,
		Quantity: domain.Quantity{
			ID:               row.QuantityID,
			Value:            row.QuantityValue,
			UnitID:           row.QuantityUnitID,
			UnitAbbreviation: row.QuantityUnitAbbreviation,
			UnitType:         row.QuantityUnitType,
		},
		WasteQuantity: domain.Quantity{
			ID:               row.WasteQuantityID,
			Value:            row.WasteQuantityValue,
			UnitID:           row.WasteUnitID,
			UnitAbbreviation: row.WasteUnitAbbreviation,
			UnitType:         row.WasteUnitType,
		},
		Instructions:     nullStringToPtr(row.Instructions),
		ProductionStepID: productionStepID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (r *consumptionRepoImpl) Get(ctx context.Context, accountID, productionStepID, consumptionID string) (*domain.Consumption, *apierror.APIError) {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.get")
	defer span.End()

	row, err := r.queries.GetConsumption(ctx, sqlc.GetConsumptionParams{
		ConsumptionID: consumptionID,
		StepID:        productionStepID,
		AccountID:     accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetConsumptionRow(row, productionStepID), nil
}

func (r *consumptionRepoImpl) Create(ctx context.Context, consumptionID, quantityID, wasteQuantityID string, params domain.CreateConsumptionParams) (*domain.Consumption, *apierror.APIError) {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.create")
	defer span.End()

	err := r.queries.InsertConsumptionQuantity(ctx, sqlc.InsertConsumptionQuantityParams{
		ID:     quantityID,
		Value:  params.QuantityValue,
		UnitID: params.QuantityUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	err = r.queries.InsertConsumptionQuantity(ctx, sqlc.InsertConsumptionQuantityParams{
		ID:     wasteQuantityID,
		Value:  params.WasteQuantityValue,
		UnitID: params.WasteQuantityUnitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	err = r.queries.InsertConsumption(ctx, sqlc.InsertConsumptionParams{
		ID:               consumptionID,
		ItemID:           params.ItemID,
		QuantityID:       quantityID,
		WasteQuantityID:  wasteQuantityID,
		ProductionStepID: sql.NullString{String: params.ProductionStepID, Valid: true},
		Instructions:     ptrToNullString(params.Instructions),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, params.ProductionStepID, consumptionID)
}

func (r *consumptionRepoImpl) UpdateItem(ctx context.Context, accountID, consumptionID, itemID string, instructions *string) *apierror.APIError {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.update_item")
	defer span.End()

	err := r.queries.UpdateConsumptionItem(ctx, sqlc.UpdateConsumptionItemParams{
		ItemID:       itemID,
		Instructions: ptrToNullString(instructions),
		ID:           consumptionID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *consumptionRepoImpl) UpdateQuantity(ctx context.Context, quantityID, value, unitID string) *apierror.APIError {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.update_quantity")
	defer span.End()

	err := r.queries.UpdateConsumptionQuantity(ctx, sqlc.UpdateConsumptionQuantityParams{
		Value:  value,
		UnitID: unitID,
		ID:     quantityID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *consumptionRepoImpl) Delete(ctx context.Context, accountID, consumptionID string) *apierror.APIError {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.delete")
	defer span.End()

	err := r.queries.DeleteConsumptionRow(ctx, consumptionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *consumptionRepoImpl) IsInAccount(ctx context.Context, accountID, consumptionID string) (bool, *apierror.APIError) {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.is_in_account")
	defer span.End()

	count, err := r.queries.IsConsumptionInAccount(ctx, sqlc.IsConsumptionInAccountParams{
		ConsumptionID: consumptionID,
		AccountID:     accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *consumptionRepoImpl) GetQuantityIDs(ctx context.Context, consumptionID string) (string, string, *apierror.APIError) {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.get_quantity_ids")
	defer span.End()

	row, err := r.queries.GetConsumptionQuantityIDs(ctx, consumptionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", "", tracing.Trace(span, apiErr)
	}

	return row.QuantityID, row.WasteQuantityID, nil
}

func (r *consumptionRepoImpl) InsertQuantity(ctx context.Context, id, value, unitID string) *apierror.APIError {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.insert_quantity")
	defer span.End()

	err := r.queries.InsertConsumptionQuantity(ctx, sqlc.InsertConsumptionQuantityParams{
		ID:     id,
		Value:  value,
		UnitID: unitID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *consumptionRepoImpl) DeleteQuantity(ctx context.Context, id string) *apierror.APIError {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.delete_quantity")
	defer span.End()

	err := r.queries.DeleteConsumptionQuantity(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *consumptionRepoImpl) GetItemID(ctx context.Context, consumptionID string) (string, *apierror.APIError) {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.get_item_id")
	defer span.End()

	itemID, err := r.queries.GetConsumptionItemID(ctx, consumptionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return itemID, nil
}

func (r *consumptionRepoImpl) GetInstructions(ctx context.Context, consumptionID string) (*string, *apierror.APIError) {
	ctx, span := consumptionRepoTracer.Start(ctx, "repository.consumption.get_instructions")
	defer span.End()

	ns, err := r.queries.GetConsumptionInstructions(ctx, consumptionID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return nullStringToPtr(ns), nil
}
