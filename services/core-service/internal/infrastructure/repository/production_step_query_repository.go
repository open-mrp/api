package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var productionStepQueryRepoTracer = tracing.GetTracer("core-service.production_step_query_repository")

type productionStepQueryRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionStepQueryRepo(queries *sqlc.Queries) domain.ProductionStepQueryRepo {
	return &productionStepQueryRepoImpl{queries: queries}
}

func mapProductionStepRow(row sqlc.GetProductionStepRow) (*domain.ProductionStepDetail, *apierror.APIError) {
	producedQuantityValue, err := decimal.NewFromString(row.ProducedQuantityValue)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse produced quantity value.")
	}

	return &domain.ProductionStepDetail{
		ID:   row.ID,
		Name: row.Name,
		Production: domain.StepProduction{
			ID: row.ProductionID,
			ProducedItem: domain.LightItem{
				ID:  row.ProducedItemID,
				SKU: row.ProducedItemSku,
			},
			Quantity: domain.BatchQuantity{
				ID:      row.ProducedQuantityID,
				Measure: producedQuantityValue,
				Unit: domain.LightUnit{
					ID:           row.ProducedUnitID,
					Abbreviation: row.ProducedUnitAbbreviation,
					Type:         row.ProducedUnitType,
				},
			},
		},
	}, nil
}

func mapFindStepRow(row sqlc.FindStepByScanningStationAndItemRow) (*domain.ProductionStepDetail, *apierror.APIError) {
	producedQuantityValue, err := decimal.NewFromString(row.ProducedQuantityValue)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse produced quantity value.")
	}

	return &domain.ProductionStepDetail{
		ID:   row.ID,
		Name: row.Name,
		Production: domain.StepProduction{
			ID: row.ProductionID,
			ProducedItem: domain.LightItem{
				ID:  row.ProducedItemID,
				SKU: row.ProducedItemSku,
			},
			Quantity: domain.BatchQuantity{
				ID:      row.ProducedQuantityID,
				Measure: producedQuantityValue,
				Unit: domain.LightUnit{
					ID:           row.ProducedUnitID,
					Abbreviation: row.ProducedUnitAbbreviation,
					Type:         row.ProducedUnitType,
				},
			},
		},
	}, nil
}

func mapStepConsumptionRow(row sqlc.GetProductionStepConsumptionsRow) (*domain.StepConsumption, *apierror.APIError) {
	consumptionValue, err := decimal.NewFromString(row.ConsumptionQuantityValue)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse consumption quantity value.")
	}

	wasteValue, err := decimal.NewFromString(row.WasteQuantityValue)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to parse waste quantity value.")
	}

	var instructions *string
	if row.Instructions.Valid {
		instructions = &row.Instructions.String
	}

	return &domain.StepConsumption{
		ID: row.ID,
		ConsumedItem: domain.LightItem{
			ID:  row.ConsumedItemID,
			SKU: row.ConsumedItemSku,
		},
		Quantity: domain.BatchQuantity{
			ID:      row.ConsumptionQuantityID,
			Measure: consumptionValue,
			Unit: domain.LightUnit{
				ID:           row.ConsumptionUnitID,
				Abbreviation: row.ConsumptionUnitAbbreviation,
				Type:         row.ConsumptionUnitType,
			},
		},
		WasteQuantity: domain.BatchQuantity{
			ID:      row.WasteQuantityID,
			Measure: wasteValue,
			Unit: domain.LightUnit{
				ID:           row.WasteUnitID,
				Abbreviation: row.WasteUnitAbbreviation,
				Type:         row.WasteUnitType,
			},
		},
		Instructions: instructions,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *productionStepQueryRepoImpl) Find(ctx context.Context, accountID, id string) (*domain.ProductionStepDetail, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.find")
	defer span.End()

	row, err := r.queries.GetProductionStep(ctx, sqlc.GetProductionStepParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	step, apiErr := mapProductionStepRow(row)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	consumptionRows, err := r.queries.GetProductionStepConsumptions(ctx, sql.NullString{String: id, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	consumptions := make([]domain.StepConsumption, 0, len(consumptionRows))
	for _, cr := range consumptionRows {
		consumption, apiErr := mapStepConsumptionRow(cr)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		consumptions = append(consumptions, *consumption)
	}
	step.Consumptions = consumptions

	return step, nil
}

func (r *productionStepQueryRepoImpl) IsInAccount(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.is_in_account")
	defer span.End()

	count, err := r.queries.IsProductionStepInAccount(ctx, sqlc.IsProductionStepInAccountParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *productionStepQueryRepoImpl) IsMultiPart(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.is_multi_part")
	defer span.End()

	count, err := r.queries.CountProductionStepConsumptions(ctx, sql.NullString{String: id, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 1, nil
}

func (r *productionStepQueryRepoImpl) IsLastStep(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.is_last_step")
	defer span.End()

	isLast, err := r.queries.IsLastProductionStep(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return isLast == 1, nil
}

func (r *productionStepQueryRepoImpl) IsInputOfStep(ctx context.Context, accountID, currentStepID, inputStepID string) (bool, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.is_input_of_step")
	defer span.End()

	count, err := r.queries.IsInputOfProductionStep(ctx, sqlc.IsInputOfProductionStepParams{
		CurrentStepID: currentStepID,
		InputStepID:   inputStepID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *productionStepQueryRepoImpl) FindProducedItemID(ctx context.Context, accountID, id string) (string, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.find_produced_item_id")
	defer span.End()

	itemID, err := r.queries.FindProducedItemIDByStep(ctx, sql.NullString{String: id, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return itemID, nil
}

func (r *productionStepQueryRepoImpl) FindProducedUnit(ctx context.Context, accountID, id string) (*domain.LightUnit, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.find_produced_unit")
	defer span.End()

	row, err := r.queries.FindProducedUnitByStep(ctx, sql.NullString{String: id, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.LightUnit{
		ID:           row.ID,
		Abbreviation: row.Abbreviation,
		Type:         row.Type,
	}, nil
}

func (r *productionStepQueryRepoImpl) FindIDByScanningStationAndProducedBlock(ctx context.Context, accountID, scanningStationID, itemID string) (string, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.find_id_by_scanning_station_and_produced_block")
	defer span.End()

	id, err := r.queries.FindStepIDByScanningStationAndItem(ctx, sqlc.FindStepIDByScanningStationAndItemParams{
		ScanningStationID: sql.NullString{String: scanningStationID, Valid: true},
		AccountID:         accountID,
		ItemID:            itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return id, nil
}

func (r *productionStepQueryRepoImpl) FindOneByScanningStationAndProducedBlock(ctx context.Context, accountID, scanningStationID, itemID string) (*domain.ProductionStepDetail, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.find_one_by_scanning_station_and_produced_block")
	defer span.End()

	row, err := r.queries.FindStepByScanningStationAndItem(ctx, sqlc.FindStepByScanningStationAndItemParams{
		ScanningStationID: sql.NullString{String: scanningStationID, Valid: true},
		AccountID:         accountID,
		ItemID:            itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	step, apiErr := mapFindStepRow(row)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	consumptionRows, err := r.queries.GetProductionStepConsumptions(ctx, sql.NullString{String: row.ID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	consumptions := make([]domain.StepConsumption, 0, len(consumptionRows))
	for _, cr := range consumptionRows {
		consumption, apiErr := mapStepConsumptionRow(cr)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		consumptions = append(consumptions, *consumption)
	}
	step.Consumptions = consumptions

	return step, nil
}

func (r *productionStepQueryRepoImpl) CalculateNextStepQuantities(ctx context.Context, accountID, itemID string, batchQuantity domain.BatchQuantity, stepID string) (*domain.NextStepQuantitiesResult, *apierror.APIError) {
	ctx, span := productionStepQueryRepoTracer.Start(ctx, "repository.production_step_query.calculate_next_step_quantities")
	defer span.End()

	step, apiErr := r.Find(ctx, accountID, stepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Find the consumption that matches the input item.
	var matchedConsumption *domain.StepConsumption
	for i := range step.Consumptions {
		if step.Consumptions[i].ConsumedItem.ID == itemID {
			matchedConsumption = &step.Consumptions[i]
			break
		}
	}
	if matchedConsumption == nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(
			fmt.Errorf("item %s is not a consumption of step %s", itemID, stepID),
			"Item is not a consumption of the specified production step.",
		))
	}

	// multiplier = batchQuantity.Measure / consumption.Quantity.Measure
	if matchedConsumption.Quantity.Measure.IsZero() {
		return nil, tracing.Trace(span, apierror.NewInternalError(
			fmt.Errorf("consumption quantity measure is zero for step %s", stepID),
			"Consumption quantity measure is zero.",
		))
	}
	multiplier := batchQuantity.Measure.Div(matchedConsumption.Quantity.Measure)

	// outputQuantity = production.Quantity.Measure * multiplier
	outputQuantity := step.Production.Quantity.Measure.Mul(multiplier)

	return &domain.NextStepQuantitiesResult{
		Quantity:       outputQuantity,
		ItemID:         step.Production.ProducedItem.ID,
		ProducedUnitID: step.Production.Quantity.Unit.ID,
	}, nil
}
