package repository

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var productionFlowRepoTracer = tracing.GetTracer("core-service.production_flow_repository")

type productionFlowRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionFlowRepo(queries *sqlc.Queries) domain.ProductionFlowRepo {
	return &productionFlowRepoImpl{queries: queries}
}

func (r *productionFlowRepoImpl) LinkFlow(ctx context.Context, productionStepID, accountID string) *apierror.APIError {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.link_flow")
	defer span.End()

	// Get consumed part item IDs for this step.
	consumedItemIDs, err := r.queries.GetConsumptionPartItemIDs(ctx, productionStepID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// For each consumed part item, find steps that produce it.
	var inIDs []string
	for _, itemID := range consumedItemIDs {
		stepIDs, err := r.queries.FindStepsByProducedItem(ctx, sqlc.FindStepsByProducedItemParams{
			AccountID: accountID,
			ItemID:    itemID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		inIDs = append(inIDs, stepIDs...)
	}

	// Get produced item ID for this step.
	producedItemID, err := r.queries.FlowGetProducedItemIDByStep(ctx, sql.NullString{String: productionStepID, Valid: true})
	if err != nil && err != sql.ErrNoRows {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Find steps that consume the produced item.
	var outIDs []string
	if producedItemID != "" {
		stepIDs, err := r.queries.FindStepsThatConsumeItem(ctx, sqlc.FindStepsThatConsumeItemParams{
			AccountID: accountID,
			ItemID:    producedItemID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		outIDs = stepIDs
	}

	// Clear all existing links for this step.
	err = r.queries.ClearStepLinks(ctx, sqlc.ClearStepLinksParams{
		StepID: productionStepID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Connect incoming steps (steps that produce items this step consumes).
	for _, inID := range inIDs {
		err = r.queries.ConnectSteps(ctx, sqlc.ConnectStepsParams{
			SourceID: inID,
			TargetID: productionStepID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	// Connect outgoing steps (steps that consume items this step produces).
	for _, outID := range outIDs {
		err = r.queries.ConnectSteps(ctx, sqlc.ConnectStepsParams{
			SourceID: productionStepID,
			TargetID: outID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *productionFlowRepoImpl) DisconnectSteps(ctx context.Context, sourceID, targetID string) *apierror.APIError {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.disconnect_steps")
	defer span.End()

	err := r.queries.FlowDisconnectSteps(ctx, sqlc.FlowDisconnectStepsParams{
		SourceID: sourceID,
		TargetID: targetID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionFlowRepoImpl) FindSourceStepsByConsumption(ctx context.Context, productionStepID, consumptionID, accountID string) ([]string, *apierror.APIError) {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.find_source_steps_by_consumption")
	defer span.End()

	stepIDs, err := r.queries.FindSourceStepsByConsumption(ctx, sqlc.FindSourceStepsByConsumptionParams{
		TargetStepID:  productionStepID,
		AccountID:     accountID,
		ConsumptionID: consumptionID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return stepIDs, nil
}

func (r *productionFlowRepoImpl) FindDownstreamStepByItem(ctx context.Context, productionStepID, itemID, accountID string) (*string, *apierror.APIError) {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.find_downstream_step_by_item")
	defer span.End()

	id, err := r.queries.FindDownstreamStepByItem(ctx, sqlc.FindDownstreamStepByItemParams{
		SourceStepID: productionStepID,
		AccountID:    accountID,
		ItemID:       itemID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return &id, nil
}

func (r *productionFlowRepoImpl) GetAllStepEdgesForAccount(ctx context.Context, accountID string) ([]domain.StepEdge, *apierror.APIError) {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.get_all_step_edges_for_account")
	defer span.End()

	rows, err := r.queries.GetAllStepEdgesForAccount(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	edges := make([]domain.StepEdge, 0, len(rows))
	for _, row := range rows {
		edges = append(edges, domain.StepEdge{
			ParentStepID: row.ParentStepID,
			ChildStepID:  row.ChildStepID,
		})
	}

	return edges, nil
}

func (r *productionFlowRepoImpl) ConnectStepsIdempotent(ctx context.Context, sourceID, targetID string) *apierror.APIError {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.connect_steps_idempotent")
	defer span.End()

	err := r.queries.ConnectStepsIdempotent(ctx, sqlc.ConnectStepsIdempotentParams{
		SourceID: sourceID,
		TargetID: targetID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionFlowRepoImpl) GetFlowStep(ctx context.Context, accountID, stepID string) (*domain.ProductionFlowStep, *apierror.APIError) {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.get_flow_step")
	defer span.End()

	row, err := r.queries.GetProductionFlowStep(ctx, sqlc.GetProductionFlowStepParams{
		ID:        stepID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	step := &domain.ProductionFlowStep{
		ID:             row.ID,
		Name:           row.Name,
		LevelingFactor: row.LevelingFactor,
		Allowances:     row.Allowances,
		Production: domain.StepProduction{
			ID: row.ProductionID,
			ProducedItem: domain.LightItem{
				ID:  row.ProducedItemID,
				SKU: row.ProducedItemSku,
			},
			Quantity: domain.BatchQuantity{
				ID: row.ProducedQuantityID,
				Unit: domain.LightUnit{
					ID:           row.ProducedUnitID,
					Abbreviation: row.ProducedUnitAbbreviation,
					Type:         row.ProducedUnitType,
				},
			},
		},
	}

	// Parse produced quantity value.
	prodVal, parseErr := decimal.NewFromString(row.ProducedQuantityValue)
	if parseErr != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(parseErr, "Failed to parse produced quantity value."))
	}
	step.Production.Quantity.Measure = prodVal

	if row.ScanningStationID.Valid {
		step.ScanningStationID = &row.ScanningStationID.String
	}

	if row.LaborRateID.Valid {
		step.LaborRate = &domain.FlowRate{
			ID:                row.LaborRateID.String,
			Value:             row.LaborRateValue.String,
			NumeratorUnitID:   row.LaborRateNumUnitID.String,
			DenominatorUnitID: row.LaborRateDenUnitID.String,
		}
	}

	if row.LaborTimeID.Valid {
		step.LaborTime = &domain.FlowRate{
			ID:                row.LaborTimeID.String,
			Value:             row.LaborTimeValue.String,
			NumeratorUnitID:   row.LaborTimeNumUnitID.String,
			DenominatorUnitID: row.LaborTimeDenUnitID.String,
		}
	}

	if row.OverheadRateID.Valid {
		step.OverheadRate = &domain.FlowRate{
			ID:                row.OverheadRateID.String,
			Value:             row.OverheadRateValue.String,
			NumeratorUnitID:   row.OverheadRateNumUnitID.String,
			DenominatorUnitID: row.OverheadRateDenUnitID.String,
		}
	}

	return step, nil
}

func (r *productionFlowRepoImpl) FindStepsByProducedItem(ctx context.Context, accountID, itemID string) ([]string, *apierror.APIError) {
	ctx, span := productionFlowRepoTracer.Start(ctx, "repository.production_flow.find_steps_by_produced_item")
	defer span.End()

	stepIDs, err := r.queries.FindStepsByProducedItem(ctx, sqlc.FindStepsByProducedItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return stepIDs, nil
}
