package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
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

// GetFinishingMachines returns every machine outside the constraint department.
//
// The second stage is the complement of the constraint, not a list of its own: a department added to the plant belongs to stage two the day it exists, without anyone revisiting settings.
func (r *productionScheduleInputRepoImpl) GetFinishingMachines(
	ctx context.Context,
	accountID, constraintDepartmentID string,
) ([]scheduling.Machine, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_finishing_machines")
	defer span.End()

	rows, err := r.queries.GetFinishingMachines(ctx, sqlc.GetFinishingMachinesParams{
		AccountID:              accountID,
		ConstraintDepartmentID: constraintDepartmentID,
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

// GetFinishingBatchMeasurements returns the second stage's production history for the given finished goods.
func (r *productionScheduleInputRepoImpl) GetFinishingBatchMeasurements(
	ctx context.Context,
	params domain.GetFinishingBatchMeasurementsParams,
) ([]domain.FinishingBatchRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_finishing_batch_measurements")
	defer span.End()

	if len(params.ItemIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetFinishingBatchMeasurements(ctx, sqlc.GetFinishingBatchMeasurementsParams{
		AccountID:              params.AccountID,
		WindowStart:            gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:              gosql.NullTime{Time: params.WindowEnd, Valid: true},
		ItemIds:                params.ItemIDs,
		ConstraintDepartmentID: gosql.NullString{String: params.ConstraintDepartmentID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.FinishingBatchRow, 0, len(rows))
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

		batchRow := domain.FinishingBatchRow{Measurement: measurement}
		if row.QuantityUnitID.Valid {
			quantityUnitID := row.QuantityUnitID.String
			batchRow.QuantityUnitID = &quantityUnitID
			batchRow.QuantityUnitRatio = scheduleUnitRatio(row.RatioNumerator, row.RatioDenominator)
		}
		if row.ProductionStepID.Valid {
			productionStepID := row.ProductionStepID.String
			batchRow.ProductionStepID = &productionStepID
		}
		// COALESCE makes this a plain string, so an empty one is the absence rather than a NULL.
		if row.StepDepartmentID != "" {
			departmentID := row.StepDepartmentID
			batchRow.StepDepartmentID = &departmentID
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

// GetSeedBatchesForItems returns every scanned batch for the given items inside the window, most recent first per item, to start the genealogy walk from.
func (r *productionScheduleInputRepoImpl) GetSeedBatchesForItems(
	ctx context.Context,
	params domain.GetSeedBatchesParams,
) ([]domain.SeedBatchRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_seed_batches")
	defer span.End()

	rows, err := r.queries.GetSeedBatchesForItems(ctx, sqlc.GetSeedBatchesForItemsParams{
		AccountID:   params.AccountID,
		ItemIds:     params.ItemIDs,
		WindowStart: gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:   gosql.NullTime{Time: params.WindowEnd, Valid: true},
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

	// Pure read on the promise-quote hot path: a dropped connection (Vitess tablet failover) retries instead of surfacing a 500.
	var rows []sqlc.GetEchelonOnHandRow
	err := db.WithConnRetry(ctx, nil, "production_schedule_input.get_echelon_on_hand", func() error {
		var queryErr error
		rows, queryErr = r.queries.GetEchelonOnHand(ctx, sqlc.GetEchelonOnHandParams{
			AccountID: accountID,
			ItemIds:   itemIDs,
		})
		return queryErr
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
func (r *productionScheduleInputRepoImpl) GetOpenOrderRequirements(
	ctx context.Context,
	accountID string,
	productIDs []string,
) ([]domain.OpenOrderRequirementRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_open_order_requirements")
	defer span.End()

	if len(productIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetOpenOrderRequirements(ctx, sqlc.GetOpenOrderRequirementsParams{
		AccountID:  accountID,
		ProductIds: scheduleNullStrings(productIDs),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OpenOrderRequirementRow, 0, len(rows))
	for _, row := range rows {
		if !row.ProductID.Valid {
			continue
		}
		// A fully packed line owes nothing; an over-packed one owes less than nothing, which is not demand either.
		outstanding := decimalToFloat64(row.OutstandingQuantity)
		if outstanding <= 0 {
			continue
		}
		req := domain.OpenOrderRequirementRow{
			SalesOrderID:     row.SalesOrderID,
			SalesOrderNumber: row.SalesOrderNumber,
			SalesOrderLineID: row.SalesOrderLineID,
			ProductID:        row.ProductID.String,
			OutstandingQty:   outstanding,
		}
		if row.ShipByDate.Valid {
			req.ShipByDate = &row.ShipByDate.Time
		}
		out = append(out, req)
	}
	return out, nil
}

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
		PlanningHorizonWeeks:         int(row.PlanningHorizonWeeks),
		FrozenWeeks:                  int(row.FrozenWeeks),
		WeekStartDay:                 int(row.WeekStartDay),
		ShiftsPerDay:                 int(row.ShiftsPerDay),
		HoursPerShift:                decimalToFloat64(row.HoursPerShift),
		WorkDaysPerWeek:              int(row.WorkDaysPerWeek),
		WeeksPerYear:                 int(row.WeeksPerYear),
		CapacityHeadroomPct:          decimalToFloat64(row.CapacityHeadroomPct),
		DefaultLotUnits:              decimalToFloat64(row.DefaultLotUnits),
		DefaultCustomerLeadTimeDays:  int(row.DefaultCustomerLeadTimeDays),
		DefaultFulfillmentPolicyCode: row.DefaultFulfillmentPolicyCode,
		RecommendationThresholds: scheduling.RecommendationThresholds{
			DormantMonths:     int(row.RecommendationDormantMonths),
			ConcentrationPct:  decimalToFloat64(row.RecommendationConcentrationPct),
			ADIThreshold:      decimalToFloat64(row.RecommendationAdiThreshold),
			CV2Threshold:      decimalToFloat64(row.RecommendationCv2Threshold),
			SlowMoverCOGS:     decimalToFloat64(row.RecommendationSlowMoverCogs),
			HighValueUnitCost: decimalToFloat64(row.RecommendationHighValueUnitCost),
		},
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

// GetConstraintDepartmentLaborRate returns the hourly labor rate configured on the constraint department, or nil when the department has none.
func (r *productionScheduleInputRepoImpl) GetConstraintDepartmentLaborRate(
	ctx context.Context,
	accountID, departmentID string,
) (*float64, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_constraint_department_labor_rate")
	defer span.End()

	value, err := r.queries.GetConstraintDepartmentLaborRate(ctx, sqlc.GetConstraintDepartmentLaborRateParams{
		AccountID:    accountID,
		DepartmentID: departmentID,
	})
	if err == gosql.ErrNoRows {
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rate := decimalToFloat64(value)
	return &rate, nil
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
		if item.FulfillmentPolicyCode.Valid {
			setting.FulfillmentPolicyCode = item.FulfillmentPolicyCode.String
		}
		out = append(out, setting)
	}
	return out, nil
}

func (r *productionScheduleInputRepoImpl) ListProductLineFulfillmentPolicies(
	ctx context.Context,
	accountID string,
) (map[string]string, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.list_product_line_policies")
	defer span.End()

	rows, err := r.queries.ListProductLineFulfillmentPolicies(ctx, gosql.NullString{String: accountID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.FulfillmentPolicyCode.Valid && row.FulfillmentPolicyCode.String != "" {
			out[row.ID] = row.FulfillmentPolicyCode.String
		}
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

func (r *productionScheduleInputRepoImpl) GetProductDemandByCustomer(
	ctx context.Context,
	params domain.GetPooledOrderDemandParams,
) ([]domain.CustomerDemandRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_demand_by_customer")
	defer span.End()

	if len(params.ProductIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetProductDemandByCustomer(ctx, sqlc.GetProductDemandByCustomerParams{
		AccountID:   params.AccountID,
		WindowStart: gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:   gosql.NullTime{Time: params.WindowEnd, Valid: true},
		ProductIds:  scheduleNullStrings(params.ProductIDs),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.CustomerDemandRow, 0, len(rows))
	for _, row := range rows {
		if !row.ProductID.Valid {
			continue
		}
		out = append(out, domain.CustomerDemandRow{
			ProductID:      row.ProductID.String,
			BuyerAccountID: row.BuyerAccountID,
			Year:           int(row.DemandYear),
			Month:          int(row.DemandMonth),
			Quantity:       decimalToFloat64(row.Quantity),
		})
	}
	return out, nil
}

// GetCustomerFulfillmentProfiles resolves each customer's lead time and policy through the same chains an order and an item use, so a recommendation and a stamped commitment can never disagree about the same customer.
func (r *productionScheduleInputRepoImpl) GetCustomerFulfillmentProfiles(
	ctx context.Context,
	accountID string,
	accountDefaultLeadTimeDays int,
) ([]domain.CustomerFulfillmentProfile, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_customer_profiles")
	defer span.End()

	rows, err := r.queries.GetCustomerFulfillmentProfiles(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.CustomerFulfillmentProfile, 0, len(rows))
	for _, row := range rows {
		profile := domain.CustomerFulfillmentProfile{
			CustomerAccountID: row.CustomerAccountID,
			CustomerName:      row.CustomerName,
			LeadTimeDays:      accountDefaultLeadTimeDays,
		}

		in := scheduling.LeadTimeInput{AccountLeadTimeDays: &accountDefaultLeadTimeDays}
		if row.CustomerLeadTimeDays.Valid {
			days := int(row.CustomerLeadTimeDays.Int32)
			in.CustomerLeadTimeDays = &days
		}
		if row.ParentCustomerLeadTimeDays.Valid {
			days := int(row.ParentCustomerLeadTimeDays.Int32)
			in.ParentCustomerLeadTimeDays = &days
		}
		if row.AccountGroupLeadTimeDays.Valid {
			days := int(row.AccountGroupLeadTimeDays.Int32)
			in.AccountGroupLeadTimeDays = &days
		}
		if days, _, ok := scheduling.ResolveLeadTime(in); ok {
			profile.LeadTimeDays = days
		}

		// The policy chain is the customer's own, then its group's. There is no account-wide customer policy: the account default is about items, not about how people buy.
		if row.CustomerPolicy.Valid && row.CustomerPolicy.String != "" {
			profile.FulfillmentPolicyCode = row.CustomerPolicy.String
		} else if row.AccountGroupPolicy.Valid && row.AccountGroupPolicy.String != "" {
			profile.FulfillmentPolicyCode = row.AccountGroupPolicy.String
		}

		out = append(out, profile)
	}
	return out, nil
}

func (r *productionScheduleInputRepoImpl) GetAllSellableProducts(ctx context.Context, accountID string) ([]domain.SellableProductRow, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_all_sellable_products")
	defer span.End()

	rows, err := r.queries.GetAllSellableProducts(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.SellableProductRow, 0, len(rows))
	for _, row := range rows {
		item := domain.SellableProductRow{ProductID: row.ProductID, ItemID: row.ItemID, SKU: row.Sku}
		if row.Description.Valid {
			item.Description = &row.Description.String
		}
		if row.ProductLineID.Valid {
			item.ProductLineID = &row.ProductLineID.String
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *productionScheduleInputRepoImpl) GetItemUnitCosts(ctx context.Context, accountID string, itemIDs []string) (map[string]float64, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.get_item_unit_costs")
	defer span.End()

	if len(itemIDs) == 0 {
		return map[string]float64{}, nil
	}

	rows, err := r.queries.GetItemUnitCosts(ctx, sqlc.GetItemUnitCostsParams{
		AccountID: accountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make(map[string]float64, len(rows))
	for _, row := range rows {
		out[row.ItemID] = decimalToFloat64(row.UnitCost)
	}
	return out, nil
}

// deliveryFilterArgs turns the empty-means-all domain filters into the include-flag plus slice pairs the queries take.
//
// An empty slice is safe: sqlc rewrites it to `IN (NULL)`, and the `include_x = false` half of the OR short-circuits before that is ever evaluated.
type deliveryFilterArgs struct {
	includeCustomer      bool
	customerIDs          []string
	includeCustomerGroup bool
	customerGroupIDs     []gosql.NullString
	includeProductLine   bool
	productLineIDs       []gosql.NullString
	includeSalesRep      bool
	salesRepIDs          []gosql.NullString
}

func buildDeliveryFilterArgs(filters domain.DeliveryFilters) deliveryFilterArgs {
	// The columns behind these three are nullable, so sqlc types their slices as NullString.
	nullable := func(ids []string) []gosql.NullString {
		out := make([]gosql.NullString, 0, len(ids))
		for _, id := range ids {
			out = append(out, gosql.NullString{String: id, Valid: true})
		}
		return out
	}
	return deliveryFilterArgs{
		includeCustomer:      len(filters.CustomerIDs) > 0,
		customerIDs:          filters.CustomerIDs,
		includeCustomerGroup: len(filters.CustomerGroupIDs) > 0,
		customerGroupIDs:     nullable(filters.CustomerGroupIDs),
		includeProductLine:   len(filters.ProductLineIDs) > 0,
		productLineIDs:       nullable(filters.ProductLineIDs),
		includeSalesRep:      len(filters.SalesRepIDs) > 0,
		salesRepIDs:          nullable(filters.SalesRepIDs),
	}
}

func (r *productionScheduleInputRepoImpl) ListDeliveryOutcomes(ctx context.Context, accountID string, start, end time.Time, filters domain.DeliveryFilters) ([]scheduling.DeliveryOutcome, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.list_delivery_outcomes")
	defer span.End()

	args := buildDeliveryFilterArgs(filters)

	rows, err := r.queries.ListDeliveryPerformanceOrders(ctx, sqlc.ListDeliveryPerformanceOrdersParams{
		AccountID:                  accountID,
		WindowStart:                gosql.NullTime{Time: start, Valid: true},
		WindowEnd:                  gosql.NullTime{Time: end, Valid: true},
		IncludeSalesRepFilter:      args.includeSalesRep,
		SalesRepIds:                args.salesRepIDs,
		IncludeProductLineFilter:   args.includeProductLine,
		ProductLineIds:             args.productLineIDs,
		IncludeCustomerGroupFilter: args.includeCustomerGroup,
		CustomerGroupIds:           args.customerGroupIDs,
		IncludeCustomerFilter:      args.includeCustomer,
		CustomerIds:                args.customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lineRows, err := r.queries.ListDeliveryOrderProductLines(ctx, sqlc.ListDeliveryOrderProductLinesParams{
		AccountID:                  accountID,
		WindowStart:                gosql.NullTime{Time: start, Valid: true},
		WindowEnd:                  gosql.NullTime{Time: end, Valid: true},
		IncludeSalesRepFilter:      args.includeSalesRep,
		SalesRepIds:                args.salesRepIDs,
		IncludeProductLineFilter:   args.includeProductLine,
		ProductLineIds:             args.productLineIDs,
		IncludeCustomerGroupFilter: args.includeCustomerGroup,
		CustomerGroupIds:           args.customerGroupIDs,
		IncludeCustomerFilter:      args.includeCustomer,
		CustomerIds:                args.customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productLines := map[string][]scheduling.ProductLineRef{}
	for _, row := range lineRows {
		productLines[row.SalesOrderID] = append(productLines[row.SalesOrderID], scheduling.ProductLineRef{
			ID:   nullSQLString(row.ProductLineID),
			Name: row.ProductLineName,
		})
	}

	out := make([]scheduling.DeliveryOutcome, 0, len(rows))
	for _, row := range rows {
		if !row.ShipByDate.Valid {
			continue
		}
		outcome := scheduling.DeliveryOutcome{
			SalesOrderID:      row.SalesOrderID,
			SalesOrderNumber:  row.SalesOrderNumber,
			BuyerAccountID:    row.BuyerAccountID,
			CustomerName:      nullSQLString(row.CustomerName),
			CustomerGroupID:   nullSQLString(row.CustomerGroupID),
			CustomerGroupName: nullSQLString(row.CustomerGroupName),
			SalesRepID:        nullSQLString(row.SalesRepID),
			ProductLines:      productLines[row.SalesOrderID],
			ShipByDate:        row.ShipByDate.Time,
			QuantityOrdered:   decimalToFloat64(row.QuantityOrdered),
			QuantityPacked:    decimalToFloat64(row.QuantityPacked),
			CommitmentSource:  nullSQLString(row.LeadTimeSourceCode),
		}
		if row.IssuedAt.Valid {
			outcome.IssuedAt = &row.IssuedAt.Time
		}
		if row.FirstShipAt.Valid {
			outcome.FirstShipAt = &row.FirstShipAt.Time
		}
		if row.LeadTimeDays.Valid {
			outcome.CommittedLeadTimeDays = int(row.LeadTimeDays.Int32)
		}
		out = append(out, outcome)
	}
	return out, nil
}

func (r *productionScheduleInputRepoImpl) CountUncommittedOrders(ctx context.Context, accountID string, start, end time.Time, filters domain.DeliveryFilters) (int, *apierror.APIError) {
	ctx, span := scheduleInputRepoTracer.Start(ctx, "repository.production_schedule_input.count_uncommitted_orders")
	defer span.End()

	args := buildDeliveryFilterArgs(filters)

	count, err := r.queries.CountUncommittedOrders(ctx, sqlc.CountUncommittedOrdersParams{
		AccountID:                  accountID,
		WindowStart:                gosql.NullTime{Time: start, Valid: true},
		WindowEnd:                  gosql.NullTime{Time: end, Valid: true},
		IncludeSalesRepFilter:      args.includeSalesRep,
		SalesRepIds:                args.salesRepIDs,
		IncludeProductLineFilter:   args.includeProductLine,
		ProductLineIds:             args.productLineIDs,
		IncludeCustomerGroupFilter: args.includeCustomerGroup,
		CustomerGroupIds:           args.customerGroupIDs,
		IncludeCustomerFilter:      args.includeCustomer,
		CustomerIds:                args.customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return int(count), nil
}
