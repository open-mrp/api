package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// The scans a production cost report reads are the account's in the window. Selecting departments narrows that to their stations' scans, which the station key ranges directly; items and categories are checked on the rows either key yields, since no account-led key orders a batch by its item.
const (
	productionCostWindowKey  = "batch_account_id_scanned_at_idx"
	productionCostStationKey = "batch_account_id_scanning_station_id_scanned_at_id_idx"
)

// productionCostQuery totals the window's batches per production step, scanning station department and item category, in each quantity's base unit.
//
// Batches count as the dashboard counted them: every batch scanned in the window at a production step, open or closed, its productive quantity and whatever seconds and waste it recorded. A batch at no station has no department.
func productionCostQuery(params domain.AnalyzeProductionCostsParams, stationIDs, itemIDs []string) (string, []any) {
	key := productionCostWindowKey
	preds := []string{"b.account_id = ?", "b.scanned_at >= ?", "b.scanned_at <= ?", "b.production_step_id IS NOT NULL"}
	args := []any{params.AccountID, params.StartDate, params.EndDate}
	if len(stationIDs) > 0 {
		key = productionCostStationKey
		preds = append(preds, "b.scanning_station_id IN ("+placeholders(len(stationIDs))+")")
		args = append(args, stringsToAny(stationIDs)...)
	}
	if len(itemIDs) > 0 {
		preds = append(preds, "b.item_id IN ("+placeholders(len(itemIDs))+")")
		args = append(args, stringsToAny(itemIDs)...)
	}
	if len(params.CategoryIDs) > 0 {
		preds = append(preds, "i.item_category_id IN ("+placeholders(len(params.CategoryIDs))+")")
		args = append(args, stringsToAny(params.CategoryIDs)...)
	}

	query := `SELECT g.production_step_id, d.id, d.name, c.id, c.name,
    g.productive_base, g.productive_count, g.seconds_base, g.seconds_count, g.waste_base, g.waste_count
FROM (
    SELECT
        b.production_step_id,
        ss.department_id,
        i.item_category_id,
        CAST(SUM(bq.value * (bqu.ratio_numerator / bqu.ratio_denominator) + bqu.offset_numerator / bqu.offset_denominator) AS DECIMAL(65,30)) AS productive_base,
        COUNT(*) AS productive_count,
        CAST(COALESCE(SUM(bs.value * (bsu.ratio_numerator / bsu.ratio_denominator) + bsu.offset_numerator / bsu.offset_denominator), 0) AS DECIMAL(65,30)) AS seconds_base,
        COUNT(bsu.id) AS seconds_count,
        CAST(COALESCE(SUM(bw.value * (bwu.ratio_numerator / bwu.ratio_denominator) + bwu.offset_numerator / bwu.offset_denominator), 0) AS DECIMAL(65,30)) AS waste_base,
        COUNT(bwu.id) AS waste_count
    FROM batch b FORCE INDEX (` + key + `)
    JOIN quantity bq ON bq.id = b.quantity_id
    JOIN unit bqu ON bqu.id = bq.unit_id
    LEFT JOIN quantity bs ON bs.id = b.seconds_quantity_id
    LEFT JOIN unit bsu ON bsu.id = bs.unit_id
    LEFT JOIN quantity bw ON bw.id = b.waste_quantity_id
    LEFT JOIN unit bwu ON bwu.id = bw.unit_id
    LEFT JOIN scanning_station ss ON ss.id = b.scanning_station_id
    JOIN item i ON i.id = b.item_id
    WHERE ` + strings.Join(preds, " AND ") + `
    GROUP BY b.production_step_id, ss.department_id, i.item_category_id
) g
JOIN item_category c ON c.id = g.item_category_id
LEFT JOIN department d ON d.id = g.department_id`
	return query, args
}

func (r *analyticsRepoImpl) GetProductionCostRows(ctx context.Context, params domain.AnalyzeProductionCostsParams) ([]domain.ProductionCostRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_production_cost_rows")
	defer span.End()

	itemIDs, apiErr := selectedProductionParts(ctx, r.queries.DB(), params.AccountID, params.ItemIDs, params.ProductLineIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var stationIDs []string
	if len(params.DepartmentIDs) > 0 {
		var err error
		stationIDs, err = r.queries.ListDepartmentStationIDs(ctx, sqlc.ListDepartmentStationIDsParams{AccountID: params.AccountID, DepartmentIds: params.DepartmentIDs})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(stationIDs) == 0 {
			return []domain.ProductionCostRow{}, nil
		}
	}

	query, args := productionCostQuery(params, stationIDs, itemIDs)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()

	out := []domain.ProductionCostRow{}
	for rows.Next() {
		var (
			row                                       domain.ProductionCostRow
			departmentID, departmentName              sql.NullString
			productiveBase, secondsBase, wasteBase    string
			productiveCount, secondsCount, wasteCount int64
		)
		if err := rows.Scan(&row.ProductionStepID, &departmentID, &departmentName, &row.Category.ID, &row.Category.Name,
			&productiveBase, &productiveCount, &secondsBase, &secondsCount, &wasteBase, &wasteCount); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		if departmentID.Valid {
			row.Department = &domain.ProductionCostRef{ID: departmentID.String, Name: departmentName.String}
		}
		var apiErr *apierror.APIError
		if row.Productive, apiErr = productionCostBatches(productiveBase, productiveCount); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if row.Seconds, apiErr = productionCostBatches(secondsBase, secondsCount); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if row.Waste, apiErr = productionCostBatches(wasteBase, wasteCount); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return out, nil
}

func productionCostBatches(base string, count int64) (domain.ProductionCostBatches, *apierror.APIError) {
	value, err := decimal.NewFromString(base)
	if err != nil {
		return domain.ProductionCostBatches{}, apierror.NewInternalError(err, "Invalid batch quantity total.")
	}
	return domain.ProductionCostBatches{BaseQuantity: value, Count: count}, nil
}

func (r *analyticsRepoImpl) GetProductionCostSteps(ctx context.Context, accountID string, stepIDs []string) (map[string]domain.ProductionCostStep, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_production_cost_steps")
	defer span.End()

	out := map[string]domain.ProductionCostStep{}
	if len(stepIDs) == 0 {
		return out, nil
	}
	nullIDs := toNullStringSlice(stepIDs)

	productions, err := r.queries.ListProductionCostProductions(ctx, sqlc.ListProductionCostProductionsParams{AccountID: accountID, StepIds: nullIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	earliest := make(map[string]domain.StepProduction, len(stepIDs))
	for _, p := range productions {
		if _, seen := earliest[p.ProductionStepID.String]; seen {
			continue
		}
		measure, err := decimal.NewFromString(p.QuantityValue)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Invalid produced quantity."))
		}
		earliest[p.ProductionStepID.String] = domain.StepProduction{Quantity: domain.BatchQuantity{
			Measure: measure,
			Unit: domain.LightUnit{
				ID:                p.UnitID,
				Type:              p.UnitType,
				RatioNumerator:    p.RatioNumerator,
				RatioDenominator:  p.RatioDenominator,
				OffsetNumerator:   p.OffsetNumerator,
				OffsetDenominator: p.OffsetDenominator,
			},
		}}
	}

	steps, err := r.queries.ListProductionCostSteps(ctx, sqlc.ListProductionCostStepsParams{AccountID: accountID, StepIds: stepIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, s := range steps {
		production, ok := earliest[s.ID]
		if !ok {
			continue
		}
		step := domain.ProductionFlowStep{
			ID:             s.ID,
			Production:     production,
			LevelingFactor: s.LevelingFactor,
			Allowances:     s.Allowances,
		}
		if s.LaborTimeValue.Valid {
			step.LaborTime = &domain.FlowRate{
				Value:               s.LaborTimeValue.String,
				NumeratorRatio:      s.LaborTimeNumRatio,
				DenominatorRatio:    s.LaborTimeDenRatio,
				DenominatorUnitType: s.LaborTimeDenUnitType.String,
			}
		}
		if s.LaborRateValue.Valid {
			step.LaborRate = &domain.FlowRate{Value: s.LaborRateValue.String, DenominatorRatio: s.LaborRateDenRatio}
		}
		if s.OverheadRateValue.Valid {
			step.OverheadRate = &domain.FlowRate{Value: s.OverheadRateValue.String, DenominatorRatio: s.OverheadRateDenRatio}
		}
		out[s.ID] = domain.ProductionCostStep{Step: step, Consumptions: []domain.CostFlowConsumption{}}
	}

	consumptions, err := r.queries.ListProductionCostConsumptions(ctx, sqlc.ListProductionCostConsumptionsParams{AccountID: accountID, StepIds: nullIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, c := range consumptions {
		step, ok := out[c.ProductionStepID.String]
		if !ok {
			continue
		}
		consumption, apiErr := productionCostConsumption(c)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		step.Consumptions = append(step.Consumptions, consumption)
		out[c.ProductionStepID.String] = step
	}
	return out, nil
}

func productionCostConsumption(c sqlc.ListProductionCostConsumptionsRow) (domain.CostFlowConsumption, *apierror.APIError) {
	values := make([]decimal.Decimal, 0, 5)
	for _, raw := range []string{c.ConsumptionQuantityValue, c.ConsumptionUnitRatio, c.WasteQuantityValue, c.WasteUnitRatio, c.ConsumedItemUnitCostRatio} {
		v, err := decimal.NewFromString(raw)
		if err != nil {
			return domain.CostFlowConsumption{}, apierror.NewInternalError(err, "Invalid consumption quantity.")
		}
		values = append(values, v)
	}
	unitCost, err := decimal.NewFromString(c.ConsumedItemUnitCost)
	if err != nil {
		return domain.CostFlowConsumption{}, apierror.NewInternalError(err, "Invalid item unit cost.")
	}
	return domain.CostFlowConsumption{
		ConsumedItemID:           c.ConsumedItemID,
		ConsumedItemType:         c.ConsumedItemType,
		ConsumptionQuantity:      values[0],
		ConsumptionUnitRatio:     values[1],
		WasteQuantity:            values[2],
		WasteUnitRatio:           values[3],
		UnitCost:                 unitCost,
		UnitCostDenominatorRatio: values[4],
	}, nil
}

func (r *analyticsRepoImpl) GetBaseUnitIDsByDimension(ctx context.Context) (map[string]string, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_base_unit_ids_by_dimension")
	defer span.End()

	rows, err := r.queries.ListBaseUnitsByDimension(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if _, ok := out[row.UnitDimensionCode]; !ok {
			out[row.UnitDimensionCode] = row.ID
		}
	}
	return out, nil
}
