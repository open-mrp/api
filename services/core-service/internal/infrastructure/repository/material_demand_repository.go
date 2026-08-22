package repository

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var materialDemandRepoTracer = tracing.GetTracer("core-service.material_demand_repository")

type materialDemandRepo struct {
	queries *sqlc.Queries
}

func NewMaterialDemandRepo(queries *sqlc.Queries) domain.MaterialDemandRepo {
	return &materialDemandRepo{queries: queries}
}

// demandUnit holds a unit's conversion factors so quantities can be normalized to their
// dimension's base unit before the demand math (matching Dashboard's BaseQuantityUtils).
type demandUnit struct {
	ratioNum   decimal.Decimal
	ratioDen   decimal.Decimal
	offsetNum  decimal.Decimal
	offsetDen  decimal.Decimal
	isBaseUnit bool
}

// GetMaterialDemand calculates the raw-material demand for producing `measure` (in `unitID`)
// of a single item, exploding its BOM. A leaf item is a raw material; a target item with no
// production flow yields no demand (matching Dashboard, which reserves nothing for a finished
// good that isn't produced by any step).
func (r *materialDemandRepo) GetMaterialDemand(ctx context.Context, accountID string, productItemID string, measure decimal.Decimal, unitID string) ([]domain.MaterialDemandItem, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, materialDemandRepoTracer, "repository.material_demand.get")
	defer span.End()

	demands := make(map[string]*domain.MaterialDemandItem)
	units := make(map[string]*demandUnit)
	if apiErr := r.explode(ctx, accountID, productItemID, measure, unitID, true, make(map[string]bool), demands, units); apiErr != nil {
		return nil, apiErr
	}
	return demandSlice(demands), nil
}

// GetMaterialDemandForOrder calculates the aggregated raw-material demand across a set of
// order lines. Demand for the same material arriving via different lines (or different BOM
// paths) is summed into a single entry, matching Dashboard's per-material reservation.
func (r *materialDemandRepo) GetMaterialDemandForOrder(ctx context.Context, accountID string, lines []domain.MaterialDemandLineInput) ([]domain.MaterialDemandItem, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, materialDemandRepoTracer, "repository.material_demand.get_for_order")
	defer span.End()

	demands := make(map[string]*domain.MaterialDemandItem)
	units := make(map[string]*demandUnit)
	for _, line := range lines {
		if apiErr := r.explode(ctx, accountID, line.ItemID, line.Measure, line.UnitID, true, make(map[string]bool), demands, units); apiErr != nil {
			return nil, apiErr
		}
	}
	return demandSlice(demands), nil
}

// explode recursively traverses the BOM, accumulating raw-material demand into `demands`.
// For each produced item it finds the producing step, computes the normalization factor
// (requested quantity ÷ step output, both normalized to base units) and scales every
// consumption (its quantity plus waste) by it before recursing. `isTarget` distinguishes the
// top-level finished good (no step ⇒ no demand) from a deeper leaf (no step ⇒ raw material).
func (r *materialDemandRepo) explode(
	ctx context.Context,
	accountID, itemID string,
	measure decimal.Decimal,
	unitID string,
	isTarget bool,
	visited map[string]bool,
	demands map[string]*domain.MaterialDemandItem,
	units map[string]*demandUnit,
) *apierror.APIError {
	// Prevent infinite loops on circular BOMs. Unmark on exit so the same item can still be
	// reached via a sibling path (a diamond in the BOM graph).
	if visited[itemID] {
		return nil
	}
	visited[itemID] = true
	defer func() { visited[itemID] = false }()

	stepIDs, err := r.queries.FindStepsByProducedItem(ctx, sqlc.FindStepsByProducedItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	if len(stepIDs) == 0 {
		// The top-level target having no production flow means there is nothing to produce,
		// so no materials are reserved (and the finished good is never reserved as its own
		// material). A deeper leaf with no step is a raw material.
		if isTarget {
			return nil
		}
		r.addDemand(demands, itemID, measure, unitID, units)
		return nil
	}

	// Use the first matching step.
	step, err := r.queries.GetProductionStep(ctx, sqlc.GetProductionStepParams{
		ID:        stepIDs[0],
		AccountID: accountID,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	stepOutputQty, parseErr := decimal.NewFromString(step.ProducedQuantityValue)
	if parseErr != nil || stepOutputQty.IsZero() {
		return apierror.NewInternalError(parseErr, "Invalid production step output quantity.")
	}

	consumptions, err := r.queries.GetProductionStepConsumptions(ctx, sql.NullString{
		String: stepIDs[0],
		Valid:  true,
	})
	if err != nil {
		return db.MapSQLError(err)
	}

	// Load conversion factors for every unit involved in this step's normalization.
	unitIDs := make([]string, 0, 2+2*len(consumptions))
	unitIDs = append(unitIDs, unitID, step.ProducedUnitID)
	for _, c := range consumptions {
		unitIDs = append(unitIDs, c.ConsumptionUnitID, c.WasteUnitID)
	}
	if apiErr := r.loadUnits(ctx, accountID, unitIDs, units); apiErr != nil {
		return apiErr
	}

	// Normalization factor = requested ÷ step output, both normalized to their base unit so a
	// request expressed in a different unit than the step output (e.g. an order in dozens for a
	// step that outputs pairs) scales the whole BOM correctly.
	outputBase := normalizeToBase(stepOutputQty, units[step.ProducedUnitID])
	if outputBase.IsZero() {
		return apierror.NewInternalError(nil, "Production step output normalizes to zero.")
	}
	normFactor := normalizeToBase(measure, units[unitID]).Div(outputBase)

	for _, c := range consumptions {
		consumptionQty, parseErr := decimal.NewFromString(c.ConsumptionQuantityValue)
		if parseErr != nil {
			return apierror.NewInternalError(parseErr, "Invalid consumption quantity value.")
		}
		wasteQty, wasteErr := decimal.NewFromString(c.WasteQuantityValue)
		if wasteErr != nil {
			wasteQty = decimal.Zero
		}
		// Waste is folded into demand (matching Dashboard), converted into the consumption's
		// unit when the two differ.
		effective := consumptionQty.Add(convertUnit(wasteQty, units[c.WasteUnitID], units[c.ConsumptionUnitID]))
		scaled := effective.Mul(normFactor)

		if apiErr := r.explode(ctx, accountID, c.ConsumedItemID, scaled, c.ConsumptionUnitID, false, visited, demands, units); apiErr != nil {
			return apiErr
		}
	}

	return nil
}

// addDemand accumulates raw-material demand for an item. When the same material is added again
// in a different unit, the incoming measure is converted into the first-seen unit before
// summing (matching Dashboard, which sums in base units and reports in the first unit).
func (r *materialDemandRepo) addDemand(demands map[string]*domain.MaterialDemandItem, itemID string, measure decimal.Decimal, unitID string, units map[string]*demandUnit) {
	if existing, ok := demands[itemID]; ok {
		add := measure
		if existing.UnitID != unitID {
			add = convertUnit(measure, units[unitID], units[existing.UnitID])
		}
		existing.Measure = existing.Measure.Add(add)
		return
	}
	demands[itemID] = &domain.MaterialDemandItem{
		ItemID:  itemID,
		Measure: measure,
		UnitID:  unitID,
	}
}

// loadUnits batch-loads any not-yet-cached unit conversions. A unit that cannot be found is
// cached as an identity (base) unit so the math degrades to the raw values rather than panicking.
func (r *materialDemandRepo) loadUnits(ctx context.Context, accountID string, ids []string, units map[string]*demandUnit) *apierror.APIError {
	missing := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := units[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	rows, err := r.queries.GetUnitsByIDsScoped(ctx, sqlc.GetUnitsByIDsScopedParams{
		Ids:       missing,
		AccountID: sql.NullString{String: accountID, Valid: true},
	})
	if err != nil {
		return db.MapSQLError(err)
	}
	for _, row := range rows {
		units[row.ID] = &demandUnit{
			ratioNum:   parseDecimalOrZero(row.RatioNumerator),
			ratioDen:   parseDecimalOrZero(row.RatioDenominator),
			offsetNum:  parseDecimalOrZero(row.OffsetNumerator),
			offsetDen:  parseDecimalOrZero(row.OffsetDenominator),
			isBaseUnit: row.IsBaseUnit,
		}
	}
	// Any id still missing (unknown unit) degrades to an identity unit.
	for _, id := range missing {
		if _, ok := units[id]; !ok {
			units[id] = &demandUnit{isBaseUnit: true}
		}
	}
	return nil
}

// normalizeToBase converts a measure in unit `u` to its dimension's base measure:
// isBaseUnit ⇒ identity; otherwise (ratio × measure) + offset.
func normalizeToBase(v decimal.Decimal, u *demandUnit) decimal.Decimal {
	if u == nil || u.isBaseUnit {
		return v
	}
	return unitRatio(u).Mul(v).Add(unitOffset(u))
}

// convertUnit converts a measure from unit `from` to unit `to` (via their shared base unit).
func convertUnit(v decimal.Decimal, from, to *demandUnit) decimal.Decimal {
	base := normalizeToBase(v, from)
	if to == nil || to.isBaseUnit {
		return base
	}
	ratio := unitRatio(to)
	if ratio.IsZero() {
		return base
	}
	// base = ratio*result + offset  =>  result = (base - offset) / ratio
	return base.Sub(unitOffset(to)).Div(ratio)
}

func unitRatio(u *demandUnit) decimal.Decimal {
	if u.ratioDen.IsZero() {
		return decimal.NewFromInt(1)
	}
	return u.ratioNum.Div(u.ratioDen)
}

func unitOffset(u *demandUnit) decimal.Decimal {
	if u.offsetDen.IsZero() {
		return decimal.Zero
	}
	return u.offsetNum.Div(u.offsetDen)
}

func parseDecimalOrZero(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func demandSlice(demands map[string]*domain.MaterialDemandItem) []domain.MaterialDemandItem {
	result := make([]domain.MaterialDemandItem, 0, len(demands))
	for _, d := range demands {
		result = append(result, *d)
	}
	return result
}
