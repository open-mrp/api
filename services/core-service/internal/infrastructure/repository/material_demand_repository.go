package repository

import (
	"context"
	"database/sql"
	"maps"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var materialDemandRepoTracer = tracing.GetTracer("core-service.material_demand_repository")

type materialDemandRepo struct {
	queries *sqlc.Queries
}

func NewMaterialDemandRepo(queries *sqlc.Queries) domain.MaterialDemandRepo {
	return &materialDemandRepo{queries: queries}
}

func (r *materialDemandRepo) GetMaterialDemand(ctx context.Context, accountID string, productItemID string, measure decimal.Decimal, unitID string) ([]domain.MaterialDemandItem, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, materialDemandRepoTracer, "repository.material_demand.get")
	defer span.End()

	visited := make(map[string]bool)
	demands := make(map[string]*domain.MaterialDemandItem)

	if apiErr := r.explode(ctx, accountID, productItemID, measure, unitID, visited, demands); apiErr != nil {
		return nil, apiErr
	}

	result := make([]domain.MaterialDemandItem, 0, len(demands))
	for _, d := range demands {
		result = append(result, *d)
	}
	return result, nil
}

// explode recursively traverses the BOM to calculate raw material demands.
// For each item, it finds the production step that produces it, calculates
// the normalization factor (requested qty / step output qty), and recurses
// into each consumption. Leaf items (no production step) are raw materials.
func (r *materialDemandRepo) explode(
	ctx context.Context,
	accountID, itemID string,
	measure decimal.Decimal,
	unitID string,
	visited map[string]bool,
	demands map[string]*domain.MaterialDemandItem,
) *apierror.APIError {
	// Prevent infinite loops on circular BOMs
	if visited[itemID] {
		return nil
	}
	visited[itemID] = true
	defer func() { visited[itemID] = false }()

	// Find the production step that produces this item
	stepIDs, err := r.queries.FindStepsByProducedItem(ctx, sqlc.FindStepsByProducedItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	// No production step means this is a raw material (leaf node)
	if len(stepIDs) == 0 {
		r.addDemand(demands, itemID, measure, unitID)
		return nil
	}

	// Use the first matching step
	stepID := stepIDs[0]

	// Get the step's production output
	step, err := r.queries.GetProductionStep(ctx, sqlc.GetProductionStepParams{
		ID:        stepID,
		AccountID: accountID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	// Calculate normalization factor: requested / step output
	stepOutputQty, parseErr := decimal.NewFromString(step.ProducedQuantityValue)
	if parseErr != nil || stepOutputQty.IsZero() {
		return apierror.NewInternalError(parseErr, "Invalid production step output quantity.")
	}

	// If units differ, we work with the quantities as-is since the production
	// step's consumption quantities are already expressed relative to its output.
	normFactor := measure.Div(stepOutputQty)

	// Get all consumptions for this step
	consumptions, err := r.queries.GetProductionStepConsumptions(ctx, sql.NullString{
		String: stepID,
		Valid:  true,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	// For each consumption, calculate the scaled demand and recurse
	for _, c := range consumptions {
		consumptionQty, parseErr := decimal.NewFromString(c.ConsumptionQuantityValue)
		if parseErr != nil {
			return apierror.NewInternalError(parseErr, "Invalid consumption quantity value.")
		}

		scaledMeasure := consumptionQty.Mul(normFactor)

		// Recurse to see if this consumed item is itself produced
		subVisited := make(map[string]bool)
		maps.Copy(subVisited, visited)
		subDemands := make(map[string]*domain.MaterialDemandItem)
		if apiErr := r.explode(ctx, accountID, c.ConsumedItemID, scaledMeasure, c.ConsumptionUnitID, subVisited, subDemands); apiErr != nil {
			return apiErr
		}

		// If no sub-demands were generated, this consumption IS a raw material
		if len(subDemands) == 0 {
			r.addDemand(demands, c.ConsumedItemID, scaledMeasure, c.ConsumptionUnitID)
		} else {
			// Aggregate sub-demands into the main demands map
			for subItemID, subDemand := range subDemands {
				r.addDemand(demands, subItemID, subDemand.Measure, subDemand.UnitID)
			}
		}
	}

	return nil
}

// addDemand adds or accumulates a material demand for a given item.
func (r *materialDemandRepo) addDemand(demands map[string]*domain.MaterialDemandItem, itemID string, measure decimal.Decimal, unitID string) {
	if existing, ok := demands[itemID]; ok {
		existing.Measure = existing.Measure.Add(measure)
	} else {
		demands[itemID] = &domain.MaterialDemandItem{
			ItemID:  itemID,
			Measure: measure,
			UnitID:  unitID,
		}
	}
}
