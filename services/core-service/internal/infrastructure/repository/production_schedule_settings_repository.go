package repository

import (
	"context"
	gosql "database/sql"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

// defaultSettings returns the settings an account gets before it saves any, taken from the solver's own defaults so the API can never advertise assumptions the solver would not actually apply.
func defaultSettings(accountID string) *domain.ProductionScheduleSettings {
	d := scheduling.DefaultSettings()
	return &domain.ProductionScheduleSettings{
		AccountID:                      accountID,
		PlanningHorizonWeeks:           safeconv.IntToInt32(d.HorizonWeeks),
		FrozenWeeks:                    safeconv.IntToInt32(d.FrozenWeeks),
		WeekStartDay:                   1,
		DemandWindowMonths:             12,
		ForecastHistoryMonths:          24,
		ForecastMonths:                 12,
		DemandBasisCode:                scheduling.DemandBasisTrailing12,
		ForecastZ:                      d.ServiceLevelZ,
		ChangeoverAvgMinutes:           d.ChangeoverAvgMinutes,
		ChangeoverMinMinutes:           d.ChangeoverMinMinutes,
		ChangeoverMaxMinutes:           d.ChangeoverMaxMinutes,
		ChangeoverLaborRate:            d.ChangeoverLaborRate,
		HoldingRatePct:                 d.HoldingRatePct,
		ServiceLevelZ:                  d.ServiceLevelZ,
		FinishLeadTimeWeeks:            d.FinishLeadTimeWeeks,
		DefaultConstraintLeadTimeWeeks: d.DefaultConstraintLeadTimeWeeks,
		MaxWeeksSupply:                 d.MaxWeeksSupply,
		MaxFlowDepth:                   safeconv.IntToInt32(d.MaxFlowDepth),
		ShiftsPerDay:                   safeconv.IntToInt32(d.ShiftsPerDay),
		HoursPerShift:                  d.HoursPerShift,
		WorkDaysPerWeek:                safeconv.IntToInt32(d.WorkDaysPerWeek),
		WeeksPerYear:                   safeconv.IntToInt32(d.WeeksPerYear),
		CapacityHeadroomPct:            d.CapacityHeadroomPct,
		DefaultLotUnits:                d.DefaultLotUnits,
		GenerationTimezone:             "UTC",
		HasStoredSettings:              false,
	}
}

func (r *productionScheduleRepoImpl) GetSettings(ctx context.Context, accountID string) (*domain.ProductionScheduleSettings, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.get_settings")
	defer span.End()

	row, err := r.queries.GetAccountProductionScheduleSetting(ctx, accountID)
	if err == gosql.ErrNoRows {
		// Not an error: an account that has never opened the settings page plans on defaults, and the caller should see those rather than an empty resource.
		return defaultSettings(accountID), nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	settings := &domain.ProductionScheduleSettings{
		AccountID:                      accountID,
		PlanningHorizonWeeks:           row.PlanningHorizonWeeks,
		FrozenWeeks:                    row.FrozenWeeks,
		WeekStartDay:                   row.WeekStartDay,
		DemandWindowMonths:             row.DemandWindowMonths,
		ForecastHistoryMonths:          row.ForecastHistoryMonths,
		ForecastMonths:                 row.ForecastMonths,
		DemandBasisCode:                row.DemandBasisCode,
		ForecastZ:                      decimalToFloat64(row.ForecastZ),
		ChangeoverAvgMinutes:           decimalToFloat64(row.ChangeoverAvgMinutes),
		ChangeoverMinMinutes:           decimalToFloat64(row.ChangeoverMinMinutes),
		ChangeoverMaxMinutes:           decimalToFloat64(row.ChangeoverMaxMinutes),
		ChangeoverLaborRate:            decimalToFloat64(row.ChangeoverLaborRate),
		HoldingRatePct:                 decimalToFloat64(row.HoldingRatePct),
		ServiceLevelZ:                  decimalToFloat64(row.ServiceLevelZ),
		FinishLeadTimeWeeks:            decimalToFloat64(row.FinishLeadTimeWeeks),
		DefaultConstraintLeadTimeWeeks: decimalToFloat64(row.DefaultConstraintLeadTimeWeeks),
		MaxWeeksSupply:                 decimalToFloat64(row.MaxWeeksSupply),
		MaxFlowDepth:                   row.MaxFlowDepth,
		ShiftsPerDay:                   row.ShiftsPerDay,
		HoursPerShift:                  decimalToFloat64(row.HoursPerShift),
		WorkDaysPerWeek:                row.WorkDaysPerWeek,
		WeeksPerYear:                   row.WeeksPerYear,
		CapacityHeadroomPct:            decimalToFloat64(row.CapacityHeadroomPct),
		DefaultLotUnits:                decimalToFloat64(row.DefaultLotUnits),
		IsEnabled:                      row.IsEnabled,
		GenerationTimezone:             row.GenerationTimezone,
		AutoPublish:                    row.AutoPublish,
		HasStoredSettings:              true,
		CreatedAt:                      row.CreatedAt,
		UpdatedAt:                      row.UpdatedAt,
	}
	if row.ConstraintDepartmentID.Valid {
		settings.ConstraintDepartmentID = &row.ConstraintDepartmentID.String
	}
	if row.GenerationCron.Valid {
		settings.GenerationCron = &row.GenerationCron.String
	}
	if row.LastGeneratedAt.Valid {
		settings.LastGeneratedAt = &row.LastGeneratedAt.Time
	}
	return settings, nil
}

func (r *productionScheduleRepoImpl) UpsertSettings(ctx context.Context, s *domain.ProductionScheduleSettings) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.upsert_settings")
	defer span.End()

	// Only used when the row does not exist; the upsert keeps the stored id otherwise.
	settingID, apiErr := id.GenID(id.AccountProductionScheduleSettingIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err := r.queries.UpsertAccountProductionScheduleSetting(ctx, sqlc.UpsertAccountProductionScheduleSettingParams{
		ID:                             settingID,
		AccountID:                      s.AccountID,
		ConstraintDepartmentID:         dtNullString(s.ConstraintDepartmentID),
		PlanningHorizonWeeks:           s.PlanningHorizonWeeks,
		FrozenWeeks:                    s.FrozenWeeks,
		WeekStartDay:                   s.WeekStartDay,
		DemandWindowMonths:             s.DemandWindowMonths,
		ForecastHistoryMonths:          s.ForecastHistoryMonths,
		ForecastMonths:                 s.ForecastMonths,
		DemandBasisCode:                s.DemandBasisCode,
		ForecastZ:                      floatToDecimalString(s.ForecastZ),
		ChangeoverAvgMinutes:           floatToDecimalString(s.ChangeoverAvgMinutes),
		ChangeoverMinMinutes:           floatToDecimalString(s.ChangeoverMinMinutes),
		ChangeoverMaxMinutes:           floatToDecimalString(s.ChangeoverMaxMinutes),
		ChangeoverLaborRate:            floatToDecimalString(s.ChangeoverLaborRate),
		HoldingRatePct:                 floatToDecimalString(s.HoldingRatePct),
		ServiceLevelZ:                  floatToDecimalString(s.ServiceLevelZ),
		FinishLeadTimeWeeks:            floatToDecimalString(s.FinishLeadTimeWeeks),
		DefaultConstraintLeadTimeWeeks: floatToDecimalString(s.DefaultConstraintLeadTimeWeeks),
		MaxWeeksSupply:                 floatToDecimalString(s.MaxWeeksSupply),
		MaxFlowDepth:                   s.MaxFlowDepth,
		ShiftsPerDay:                   s.ShiftsPerDay,
		HoursPerShift:                  floatToDecimalString(s.HoursPerShift),
		WorkDaysPerWeek:                s.WorkDaysPerWeek,
		WeeksPerYear:                   s.WeeksPerYear,
		CapacityHeadroomPct:            floatToDecimalString(s.CapacityHeadroomPct),
		DefaultLotUnits:                floatToDecimalString(s.DefaultLotUnits),
		IsEnabled:                      s.IsEnabled,
		GenerationCron:                 dtNullString(s.GenerationCron),
		GenerationTimezone:             s.GenerationTimezone,
		AutoPublish:                    s.AutoPublish,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) ListResourceSettings(ctx context.Context, accountID string) ([]*domain.ProductionScheduleResourceSetting, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_resource_settings")
	defer span.End()

	rows, err := r.queries.ListProductionScheduleResourceSettings(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.ProductionScheduleResourceSetting, len(rows))
	for i, row := range rows {
		setting := &domain.ProductionScheduleResourceSetting{
			ID:                  row.ID,
			AccountID:           accountID,
			ScopeCode:           row.ScopeCode,
			ScopeRefID:          row.ScopeRefID,
			IsExcluded:          row.IsExcluded,
			LeadTimeOffsetWeeks: decimalToFloat64(row.LeadTimeOffsetWeeks),
		}
		out[i] = setting
	}
	return out, nil
}

func (r *productionScheduleRepoImpl) UpsertResourceSetting(ctx context.Context, id string, params domain.UpsertResourceSettingParams) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.upsert_resource_setting")
	defer span.End()

	leadTime := gosql.NullString{}
	if params.LeadTimeWeeks != nil {
		leadTime = gosql.NullString{String: floatToDecimalString(*params.LeadTimeWeeks), Valid: true}
	}

	err := r.queries.UpsertProductionScheduleResourceSetting(ctx, sqlc.UpsertProductionScheduleResourceSettingParams{
		ID:                  id,
		AccountID:           params.AccountID,
		ScopeCode:           params.ScopeCode,
		ScopeRefID:          params.ScopeRefID,
		IsExcluded:          params.IsExcluded,
		LeadTimeWeeks:       leadTime,
		LeadTimeOffsetWeeks: floatToDecimalString(params.LeadTimeOffsetWeeks),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) DeleteResourceSetting(ctx context.Context, accountID, settingID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.delete_resource_setting")
	defer span.End()

	rows, err := r.queries.DeleteProductionScheduleResourceSetting(ctx, sqlc.DeleteProductionScheduleResourceSettingParams{
		AccountID: accountID,
		ID:        settingID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// A delete that matched nothing is reported rather than swallowed: a caller that mistypes an ID, or deletes an override a colleague already removed, would otherwise be told the resource is back on account defaults when nothing happened.
	if rows == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Production schedule resource setting not found."))
	}
	return nil
}
