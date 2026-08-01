package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var scheduleInputRepoTracer = tracing.GetTracer("core-service.production_schedule_input_repository")

type productionScheduleInputRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionScheduleInputRepo(queries *sqlc.Queries) domain.ProductionScheduleInputRepo {
	return &productionScheduleInputRepoImpl{queries: queries}
}

// GetConstraintMachines returns every planned machine in the constraint department, in name order.
func (r *productionScheduleInputRepoImpl) GetConstraintMachines(
	ctx context.Context,
	accountID, departmentID string,
) ([]scheduling.Machine, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_constraint_machines")
	defer span.End()

	rows, err := r.queries.GetConstraintMachines(ctx, sqlc.GetConstraintMachinesParams{
		AccountID:    accountID,
		DepartmentID: departmentID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var machines []scheduling.Machine
	for _, m := range rows {
		machines = append(machines, scheduling.Machine{ID: m.ID, Name: m.Name})
	}
	return machines, nil
}

// CountConstraintMachinesWithoutStep reports how many constraint machines cannot carry a plan downstream because they have no production step.
func (r *productionScheduleInputRepoImpl) CountConstraintMachinesWithoutStep(
	ctx context.Context,
	accountID, departmentID string,
) (int, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.count_machines_without_step")
	defer span.End()

	coverage, err := r.queries.GetConstraintDepartmentStepCoverage(ctx, sqlc.GetConstraintDepartmentStepCoverageParams{
		AccountID:    accountID,
		DepartmentID: departmentID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return int(decimalToFloat64(coverage.MachinesWithoutStep)), nil
}

// GetConstraintBatchMeasurements returns one row per historical batch produced on the given machines inside the window: run rates, costs, affinity, lead times.
func (r *productionScheduleInputRepoImpl) GetConstraintBatchMeasurements(
	ctx context.Context,
	params domain.GetConstraintBatchMeasurementsParams,
) ([]domain.ConstraintBatchRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_constraint_batch_measurements")
	defer span.End()

	rows, err := r.queries.GetConstraintBatchMeasurements(ctx, sqlc.GetConstraintBatchMeasurementsParams{
		AccountID:              params.AccountID,
		WindowStart:            gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:              gosql.NullTime{Time: params.WindowEnd, Valid: true},
		MachineIds:             params.MachineIDs,
		ConstraintDepartmentID: gosql.NullString{String: params.ConstraintDepartmentID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.ConstraintBatchRow
	for _, row := range rows {
		measurement := scheduling.BatchMeasurement{
			BatchID:     row.BatchID,
			ItemID:      row.ItemID,
			SKU:         row.Sku,
			MachineID:   row.MachineID,
			MachineName: row.MachineName,
		}
		if row.ScannedAt.Valid {
			measurement.ScannedAt = row.ScannedAt.Time
		}
		if row.QuantityValue.Valid {
			// Normalized through the unit ratio so items stocked in different units are comparable.
			measurement.Quantity = decimalToFloat64(row.QuantityValue.String) * scheduleUnitRatio(row.RatioNumerator, row.RatioDenominator)
		}
		if row.ProductionStepID.Valid {
			measurement.ProductionStepID = row.ProductionStepID.String
		}
		if row.UnitCost.Valid {
			measurement.UnitCost = decimalToFloat64(row.UnitCost.String)
		}
		if row.LaborTimeValue.Valid {
			measurement.LaborTimeValue = decimalToFloat64(row.LaborTimeValue.String)
		}
		if row.LaborTimeUnit.Valid {
			measurement.LaborTimeUnit = row.LaborTimeUnit.String
		}
		if row.LaborRate.Valid {
			measurement.LaborRate = decimalToFloat64(row.LaborRate.String)
		}
		if row.OverheadRate.Valid {
			measurement.OverheadRate = decimalToFloat64(row.OverheadRate.String)
		}
		if row.RunCreatedAt.Valid {
			runCreatedAt := row.RunCreatedAt.Time
			measurement.RunCreatedAt = &runCreatedAt
		}

		batchRow := domain.ConstraintBatchRow{Measurement: measurement}
		if row.QuantityUnitID.Valid {
			quantityUnitID := row.QuantityUnitID.String
			batchRow.QuantityUnitID = &quantityUnitID
			batchRow.QuantityUnitRatio = scheduleUnitRatio(row.RatioNumerator, row.RatioDenominator)
		}
		if row.ProductionStepID.Valid {
			productionStepID := row.ProductionStepID.String
			batchRow.ProductionStepID = &productionStepID
		}
		out = append(out, batchRow)
	}
	return out, nil
}

// GetStepConsumptionItems returns the input items each production step consumes. Rows carrying no step cannot be attributed and are dropped.
func (r *productionScheduleInputRepoImpl) GetStepConsumptionItems(
	ctx context.Context,
	stepIDs []string,
) ([]domain.StepConsumptionRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_step_consumption_items")
	defer span.End()

	rows, err := r.queries.GetStepConsumptionItems(ctx, scheduleNullStrings(stepIDs))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.StepConsumptionRow
	for _, row := range rows {
		if !row.ProductionStepID.Valid {
			continue
		}
		out = append(out, domain.StepConsumptionRow{
			ProductionStepID: row.ProductionStepID.String,
			ItemID:           row.ItemID,
		})
	}
	return out, nil
}

// GetSeedBatchesForItems returns scanned batches for the given items, most recent first per item, to start the genealogy walk from.
func (r *productionScheduleInputRepoImpl) GetSeedBatchesForItems(
	ctx context.Context,
	accountID string,
	itemIDs []string,
) ([]domain.SeedBatchRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_seed_batches")
	defer span.End()

	rows, err := r.queries.GetSeedBatchesForItems(ctx, sqlc.GetSeedBatchesForItemsParams{
		AccountID: accountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.SeedBatchRow
	for _, row := range rows {
		out = append(out, domain.SeedBatchRow{BatchID: row.BatchID, ItemID: row.ItemID})
	}
	return out, nil
}

// GetBatchFlowChildren returns the immediate downstream batches of the given parent batches, one genealogy level in one query.
func (r *productionScheduleInputRepoImpl) GetBatchFlowChildren(
	ctx context.Context,
	accountID string,
	parentBatchIDs []string,
) ([]domain.BatchFlowChildRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_batch_flow_children")
	defer span.End()

	rows, err := r.queries.GetBatchFlowChildren(ctx, sqlc.GetBatchFlowChildrenParams{
		AccountID:      accountID,
		ParentBatchIds: parentBatchIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.BatchFlowChildRow
	for _, row := range rows {
		out = append(out, domain.BatchFlowChildRow{
			ParentBatchID: row.ParentBatchID,
			BatchID:       row.BatchID,
			ItemID:        row.ItemID,
		})
	}
	return out, nil
}

// GetEchelonOnHand returns available inventory per item, net of allocations, normalized through the unit ratio.
func (r *productionScheduleInputRepoImpl) GetEchelonOnHand(
	ctx context.Context,
	accountID string,
	itemIDs []string,
) (map[string]float64, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_echelon_on_hand")
	defer span.End()

	rows, err := r.queries.GetEchelonOnHand(ctx, sqlc.GetEchelonOnHandParams{
		AccountID: accountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	onHandByItem := map[string]float64{}
	for _, row := range rows {
		onHandByItem[row.ItemID] = decimalToFloat64(row.OnHand)
	}
	return onHandByItem, nil
}

// GetProductsForItems returns the sellable products carried by the given items, with the SKU and product line each one carries.
func (r *productionScheduleInputRepoImpl) GetProductsForItems(
	ctx context.Context,
	accountID string,
	itemIDs []string,
) ([]domain.SellableProductRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_products_for_items")
	defer span.End()

	rows, err := r.queries.GetProductsForItems(ctx, sqlc.GetProductsForItemsParams{
		AccountID: accountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.SellableProductRow
	for _, row := range rows {
		productRow := domain.SellableProductRow{
			ProductID: row.ProductID,
			ItemID:    row.ItemID,
			SKU:       row.Sku,
		}
		if row.ProductLineID.Valid {
			productLineID := row.ProductLineID.String
			productRow.ProductLineID = &productLineID
		}
		out = append(out, productRow)
	}
	return out, nil
}

// GetPooledOrderDemandByProduct returns monthly sold quantity per product inside the window. Rows carrying no product cannot be attributed and are dropped.
func (r *productionScheduleInputRepoImpl) GetPooledOrderDemandByProduct(
	ctx context.Context,
	params domain.GetPooledOrderDemandParams,
) ([]domain.PooledMonthlyDemandRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_pooled_order_demand")
	defer span.End()

	rows, err := r.queries.GetPooledOrderDemandByProduct(ctx, sqlc.GetPooledOrderDemandByProductParams{
		AccountID:   params.AccountID,
		WindowStart: gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:   gosql.NullTime{Time: params.WindowEnd, Valid: true},
		ProductIds:  scheduleNullStrings(params.ProductIDs),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.PooledMonthlyDemandRow
	for _, row := range rows {
		if !row.ProductID.Valid {
			continue
		}
		out = append(out, domain.PooledMonthlyDemandRow{
			ProductID: row.ProductID.String,
			Year:      int(row.DemandYear),
			Month:     int(row.DemandMonth),
			Quantity:  decimalToFloat64(row.Quantity),
		})
	}
	return out, nil
}

// GetActiveDemandOverrides returns the demand overrides in force at the planning date.
func (r *productionScheduleInputRepoImpl) GetActiveDemandOverrides(
	ctx context.Context,
	accountID string,
	asOf time.Time,
) ([]scheduling.DemandOverride, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_active_demand_overrides")
	defer span.End()

	rows, err := r.queries.GetActiveDemandOverrides(ctx, sqlc.GetActiveDemandOverridesParams{
		AccountID:  accountID,
		AsOf:       asOf,
		AsOfExpiry: gosql.NullTime{Time: asOf, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []scheduling.DemandOverride
	for _, row := range rows {
		override := scheduling.DemandOverride{
			ID:          row.ID,
			ScopeCode:   row.ScopeCode,
			ScopeRefID:  row.ScopeRefID,
			PeriodStart: row.PeriodStartDate,
			PeriodEnd:   row.PeriodEndDate,
			TypeCode:    row.OverrideTypeCode,
			Value:       decimalToFloat64(row.Value),
			CreatedAt:   row.CreatedAt,
		}
		if row.ReasonCode.Valid {
			override.ReasonCode = row.ReasonCode.String
		}
		out = append(out, override)
	}
	return out, nil
}

// GetItemsForProductLines maps product lines to the items sold under them. Rows carrying no line cannot be attributed and are dropped.
func (r *productionScheduleInputRepoImpl) GetItemsForProductLines(
	ctx context.Context,
	accountID string,
	productLineIDs []string,
) ([]domain.ProductLineItemRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_items_for_product_lines")
	defer span.End()

	rows, err := r.queries.GetItemsForProductLines(ctx, sqlc.GetItemsForProductLinesParams{
		AccountID:      accountID,
		ProductLineIds: scheduleNullStrings(productLineIDs),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.ProductLineItemRow
	for _, row := range rows {
		if !row.ProductLineID.Valid {
			continue
		}
		out = append(out, domain.ProductLineItemRow{
			ProductLineID: row.ProductLineID.String,
			ItemID:        row.ItemID,
		})
	}
	return out, nil
}

// ListProductLineLotDefaults returns every product line in the account that has a lot convention. The query inner-joins the lot quantity, so every row carries a full one.
func (r *productionScheduleInputRepoImpl) ListProductLineLotDefaults(
	ctx context.Context,
	accountID string,
) ([]scheduling.ProductLineLot, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.list_product_line_lot_defaults")
	defer span.End()

	rows, err := r.queries.ListProductLineLotDefaults(ctx, gosql.NullString{String: accountID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []scheduling.ProductLineLot
	for _, line := range rows {
		out = append(out, scheduling.ProductLineLot{
			ProductLineID: line.ID,
			Quantity:      decimalToFloat64(line.DefaultLotValue),
			UnitID:        line.DefaultLotUnitID,
		})
	}
	return out, nil
}

// ListItemProductLines maps items to the product line they sell under. Rows carrying no line are dropped.
func (r *productionScheduleInputRepoImpl) ListItemProductLines(
	ctx context.Context,
	accountID string,
	itemIDs []string,
) ([]domain.ItemProductLineRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.list_item_product_lines")
	defer span.End()

	rows, err := r.queries.ListItemProductLines(ctx, sqlc.ListItemProductLinesParams{
		AccountID: accountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.ItemProductLineRow
	for _, row := range rows {
		if !row.ProductLineID.Valid {
			continue
		}
		out = append(out, domain.ItemProductLineRow{
			ItemID:        row.ItemID,
			ProductLineID: row.ProductLineID.String,
		})
	}
	return out, nil
}

// GetAccountScheduleSettings returns the account's stored planning assumptions as one raw row, or nil when the account has never configured scheduling. Merging code defaults over the gaps is the service's job.
func (r *productionScheduleInputRepoImpl) GetAccountScheduleSettings(
	ctx context.Context,
	accountID string,
) (*domain.ProductionScheduleSettingsRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_account_settings")
	defer span.End()

	row, err := r.queries.GetAccountProductionScheduleSetting(ctx, accountID)
	if err == gosql.ErrNoRows {
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	settings := &domain.ProductionScheduleSettingsRow{
		PlanningHorizonWeeks:           int(row.PlanningHorizonWeeks),
		FrozenWeeks:                    int(row.FrozenWeeks),
		WeekStartDay:                   int(row.WeekStartDay),
		ShiftsPerDay:                   int(row.ShiftsPerDay),
		HoursPerShift:                  decimalToFloat64(row.HoursPerShift),
		WorkDaysPerWeek:                int(row.WorkDaysPerWeek),
		WeeksPerYear:                   int(row.WeeksPerYear),
		CapacityHeadroomPct:            decimalToFloat64(row.CapacityHeadroomPct),
		DefaultLotUnits:                decimalToFloat64(row.DefaultLotUnits),
		ChangeoverAvgMinutes:           decimalToFloat64(row.ChangeoverAvgMinutes),
		ChangeoverMinMinutes:           decimalToFloat64(row.ChangeoverMinMinutes),
		ChangeoverMaxMinutes:           decimalToFloat64(row.ChangeoverMaxMinutes),
		ChangeoverLaborRate:            decimalToFloat64(row.ChangeoverLaborRate),
		HoldingRatePct:                 decimalToFloat64(row.HoldingRatePct),
		ServiceLevelZ:                  decimalToFloat64(row.ServiceLevelZ),
		FinishLeadTimeWeeks:            decimalToFloat64(row.FinishLeadTimeWeeks),
		DefaultConstraintLeadTimeWeeks: decimalToFloat64(row.DefaultConstraintLeadTimeWeeks),
		MaxWeeksSupply:                 decimalToFloat64(row.MaxWeeksSupply),
		MaxFlowDepth:                   int(row.MaxFlowDepth),
		DemandWindowMonths:             int(row.DemandWindowMonths),
		ForecastHistoryMonths:          int(row.ForecastHistoryMonths),
		ForecastMonths:                 int(row.ForecastMonths),
		DemandBasisCode:                row.DemandBasisCode,
		ForecastZ:                      decimalToFloat64(row.ForecastZ),
	}
	if row.ConstraintDepartmentID.Valid {
		settings.ConstraintDepartmentID = row.ConstraintDepartmentID.String
	}
	return settings, nil
}

// ListScheduleItemSettings returns the account's per-item planning overrides.
func (r *productionScheduleInputRepoImpl) ListScheduleItemSettings(
	ctx context.Context,
	accountID string,
) ([]domain.ProductionScheduleItemSetting, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.list_item_settings")
	defer span.End()

	rows, err := r.queries.GetProductionScheduleItemSettings(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var out []domain.ProductionScheduleItemSetting
	for _, item := range rows {
		setting := domain.ProductionScheduleItemSetting{
			ItemID:     item.ItemID,
			IsExcluded: item.IsExcluded,
		}
		if item.LotMultipleUnits.Valid {
			setting.LotMultipleUnits = decimalToFloat64(item.LotMultipleUnits.String)
		}
		out = append(out, setting)
	}
	return out, nil
}

// unitRatio converts a quantity to the unit group's base unit. Missing ratios mean the quantity is already in base units.
func scheduleUnitRatio(numerator, denominator gosql.NullString) float64 {
	if !numerator.Valid || !denominator.Valid {
		return 1
	}
	num := decimalToFloat64(numerator.String)
	den := decimalToFloat64(denominator.String)
	if num == 0 || den == 0 {
		return 1
	}
	return num / den
}

func scheduleNullStrings(values []string) []gosql.NullString {
	out := make([]gosql.NullString, len(values))
	for i, v := range values {
		out[i] = gosql.NullString{String: v, Valid: true}
	}
	return out
}
