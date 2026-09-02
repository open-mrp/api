package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var scheduleAttainmentRepoTracer = tracing.GetTracer("core-service.schedule_attainment_repository")

type scheduleAttainmentRepoImpl struct {
	queries *sqlc.Queries
}

func NewScheduleAttainmentRepo(queries *sqlc.Queries) domain.ScheduleAttainmentRepo {
	return &scheduleAttainmentRepoImpl{queries: queries}
}

func (r *scheduleAttainmentRepoImpl) SelectAttainmentBaselines(ctx context.Context, params domain.SelectAttainmentBaselinesParams) ([]domain.AttainmentBaselineRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.select_baselines")
	defer span.End()

	rows, err := r.queries.SelectAttainmentBaselines(ctx, sqlc.SelectAttainmentBaselinesParams{
		AccountID:   params.AccountID,
		WindowEnd:   params.WindowEnd,
		WindowStart: params.WindowStart,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentBaselineRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentBaselineRow{
			ScheduleID:            row.ScheduleID,
			Version:               row.Version,
			HorizonStartDate:      row.HorizonStartDate,
			HorizonEndDate:        row.HorizonEndDate,
			PublishedAt:           nullTimePtr(row.PublishedAt),
			FrozenThroughDate:     nullTimePtr(row.FrozenThroughDate),
			FrozenLineCount:       row.FrozenLineCount,
			FrozenPlannedQuantity: decimalToFloat64(row.FrozenPlannedQuantity),
		}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) SumPlannedByWeek(ctx context.Context, params domain.SumPlannedByWeekParams) ([]domain.AttainmentPlannedRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.sum_planned_by_week")
	defer span.End()

	rows, err := r.queries.SumPlannedByWeek(ctx, sqlc.SumPlannedByWeekParams{
		AccountID:            params.AccountID,
		ProductionScheduleID: params.ProductionScheduleID,
		WindowStart:          params.WindowStart,
		WindowEnd:            params.WindowEnd,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentPlannedRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentPlannedRow{
			WeekStartDate:   row.WeekStartDate,
			MachineID:       row.MachineID,
			ItemID:          row.ItemID,
			DepartmentID:    nullStringPtr(row.DepartmentID),
			PlannedQuantity: decimalToFloat64(row.PlannedQuantity),
			PlannedRunHours: decimalToFloat64(row.PlannedRunHours),
			LineCount:       row.LineCount,
		}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) SumScheduledHoursByDepartmentWeek(ctx context.Context, params domain.SumPlannedByWeekParams) ([]domain.ScheduledHoursRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.sum_scheduled_hours_by_department_week")
	defer span.End()

	rows, err := r.queries.SumScheduledHoursByDepartmentWeek(ctx, sqlc.SumScheduledHoursByDepartmentWeekParams{
		AccountID:            params.AccountID,
		ProductionScheduleID: params.ProductionScheduleID,
		WindowStart:          params.WindowStart,
		WindowEnd:            params.WindowEnd,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.ScheduledHoursRow, len(rows))
	for i, row := range rows {
		out[i] = domain.ScheduledHoursRow{
			WeekStartDate:            row.WeekStartDate,
			DepartmentID:             row.DepartmentID.String,
			PlannedRunHours:          decimalToFloat64(row.PlannedRunHours),
			PlannedChangeoverMinutes: decimalToFloat64(row.PlannedChangeoverMinutes),
		}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) SumActualsByWeek(ctx context.Context, params domain.SumActualsByWeekParams) ([]domain.AttainmentActualRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.sum_actuals_by_week")
	defer span.End()

	rows, err := r.queries.SumActualsByWeek(ctx, sqlc.SumActualsByWeekParams{
		AccountID:    params.AccountID,
		WeekStartDay: int64(params.WeekStartDay),
		// scanned_at is nullable, so sqlc types the window as NullTime. An unscanned batch was never produced, so excluding it is correct.
		WindowStart: gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:   gosql.NullTime{Time: params.WindowEnd, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentActualRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentActualRow{
			WeekStartDate:  row.WeekStartDate,
			MachineID:      nullStringPtr(row.MachineID),
			ItemID:         row.ItemID,
			DepartmentID:   nullStringPtr(row.DepartmentID),
			ActualQuantity: decimalToFloat64(row.ActualQuantity),
			WasteQuantity:  decimalToFloat64(row.WasteQuantity),
			BatchCount:     row.BatchCount,
		}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) CountDeviationsForBaselines(ctx context.Context, accountID string, scheduleIDs []string) ([]domain.AttainmentDeviationRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.count_deviations_for_baselines")
	defer span.End()

	rows, err := r.queries.CountDeviationsForBaselines(ctx, sqlc.CountDeviationsForBaselinesParams{
		AccountID:   accountID,
		ScheduleIds: scheduleIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentDeviationRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentDeviationRow{
			ProductionScheduleID: row.ProductionScheduleID,
			DeviationCount:       row.DeviationCount,
			AddedCount:           decimalToFloat64(row.AddedCount),
			AbsDeltaQuantity:     decimalToFloat64(row.AbsDeltaQuantity),
		}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) GetMachineLabels(ctx context.Context, accountID string, ids []string) ([]domain.AttainmentLabelRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.get_machine_labels")
	defer span.End()

	rows, err := r.queries.GetMachinesByIDs(ctx, sqlc.GetMachinesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentLabelRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentLabelRow{ID: row.ID, Label: row.Name}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) GetDepartmentLabels(ctx context.Context, accountID string, ids []string) ([]domain.AttainmentLabelRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.get_department_labels")
	defer span.End()

	rows, err := r.queries.GetDepartmentsFullByIDs(ctx, sqlc.GetDepartmentsFullByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentLabelRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentLabelRow{ID: row.ID, Label: row.Name}
	}
	return out, nil
}

func (r *scheduleAttainmentRepoImpl) GetItemLabels(ctx context.Context, accountID string, ids []string) ([]domain.AttainmentLabelRow, *apierror.APIError) {
	ctx, span := scheduleAttainmentRepoTracer.Start(ctx, "repository.schedule_attainment.get_item_labels")
	defer span.End()

	rows, err := r.queries.GetItemsByIDs(ctx, sqlc.GetItemsByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.AttainmentLabelRow, len(rows))
	for i, row := range rows {
		out[i] = domain.AttainmentLabelRow{ID: row.ID, Label: row.Sku}
	}
	return out, nil
}
