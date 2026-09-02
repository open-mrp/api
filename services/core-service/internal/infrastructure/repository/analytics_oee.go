package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// GetOeeDepartmentData returns the unit counts and the standard time earned per department in the window.
func (r *analyticsRepoImpl) GetOeeDepartmentData(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeDepartmentDataRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_department_data")
	defer span.End()

	rows, err := r.queries.GetOeeDepartmentData(ctx, sqlc.GetOeeDepartmentDataParams{
		OwnerAccountID: params.AccountID,
		StartDate:      toRequiredNullTime(params.StartDate),
		EndDate:        toRequiredNullTime(params.EndDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeDepartmentDataRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OeeDepartmentDataRow{
			DepartmentID:          row.DepartmentID,
			DepartmentName:        row.DepartmentName,
			GoodUnits:             decimalToFloat64(row.GoodUnits),
			WasteUnits:            decimalToFloat64(row.WasteUnits),
			SecondsUnits:          decimalToFloat64(row.SecondsUnits),
			StandardSecondsEarned: decimalToFloat64(row.StandardSecondsEarned),
		}
	}
	return out, nil
}

// GetOeeEstimatedRuntime returns the estimated runtime seconds per department in the window.
func (r *analyticsRepoImpl) GetOeeEstimatedRuntime(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeEstimatedRuntimeRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_estimated_runtime")
	defer span.End()

	rows, err := r.queries.GetOeeEstimatedRuntime(ctx, sqlc.GetOeeEstimatedRuntimeParams{
		OwnerAccountID: params.AccountID,
		StartDate:      toRequiredNullTime(params.StartDate),
		EndDate:        toRequiredNullTime(params.EndDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeEstimatedRuntimeRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OeeEstimatedRuntimeRow{
			DepartmentID:   row.DepartmentID,
			RuntimeSeconds: decimalToFloat64(row.RuntimeSeconds),
		}
	}
	return out, nil
}

// GetOeeDowntimeByDepartment returns logged downtime per department and reason, clipped to the window.
func (r *analyticsRepoImpl) GetOeeDowntimeByDepartment(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeDowntimeRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_downtime_by_department")
	defer span.End()

	rows, err := r.queries.GetOeeDowntimeByDepartment(ctx, sqlc.GetOeeDowntimeByDepartmentParams{
		AccountID: params.AccountID,
		// sqlc infers these two asymmetrically from their use in GREATEST/LEAST.
		StartDate: toRequiredNullTime(params.StartDate),
		EndDate:   params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeDowntimeRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OeeDowntimeRow{
			DepartmentID:    row.DepartmentID,
			ReasonCode:      row.ReasonCode,
			OeeBucket:       row.OeeBucket,
			DowntimeSeconds: row.DowntimeSeconds,
			EventCount:      row.EventCount,
		}
	}
	return out, nil
}

// GetOeeTrendDepartmentDataByWeek returns unit counts and standard time earned per department per production week in the window.
func (r *analyticsRepoImpl) GetOeeTrendDepartmentDataByWeek(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeTrendDepartmentWeekRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_trend_department_data_by_week")
	defer span.End()

	rows, err := r.queries.GetOeeTrendDepartmentDataByWeek(ctx, sqlc.GetOeeTrendDepartmentDataByWeekParams{
		OwnerAccountID: params.AccountID,
		WeekStartDay:   int64(params.WeekStartDay),
		StartDate:      toRequiredNullTime(params.StartDate),
		EndDate:        toRequiredNullTime(params.EndDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeTrendDepartmentWeekRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OeeTrendDepartmentWeekRow{
			WeekStart:             row.WeekStartDate,
			DepartmentID:          row.DepartmentID,
			DepartmentName:        row.DepartmentName,
			GoodUnits:             decimalToFloat64(row.GoodUnits),
			WasteUnits:            decimalToFloat64(row.WasteUnits),
			SecondsUnits:          decimalToFloat64(row.SecondsUnits),
			StandardSecondsEarned: decimalToFloat64(row.StandardSecondsEarned),
		}
	}
	return out, nil
}

// GetOeeTrendDowntimeIntervals returns logged downtime per department as raw intervals so the caller can split them across week buckets.
func (r *analyticsRepoImpl) GetOeeTrendDowntimeIntervals(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeDowntimeIntervalRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_trend_downtime_intervals")
	defer span.End()

	rows, err := r.queries.GetOeeTrendDowntimeIntervals(ctx, sqlc.GetOeeTrendDowntimeIntervalsParams{
		AccountID: params.AccountID,
		StartDate: toRequiredNullTime(params.StartDate),
		EndDate:   params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeDowntimeIntervalRow, 0, len(rows))
	for _, row := range rows {
		// COALESCE guarantees an end, but a NULL here would otherwise become the zero time and turn one event into a window that swallows the whole trend.
		if !row.EndedAt.Valid {
			continue
		}
		out = append(out, domain.OeeDowntimeIntervalRow{
			DepartmentID: row.DepartmentID,
			OeeBucket:    row.OeeBucket,
			StartedAt:    row.StartedAt,
			EndedAt:      row.EndedAt.Time,
		})
	}
	return out, nil
}

// CountMachinesByDepartment returns the number of machines per department for the account.
func (r *analyticsRepoImpl) CountMachinesByDepartment(ctx context.Context, accountID string) ([]domain.DepartmentMachineCountRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.count_machines_by_department")
	defer span.End()

	rows, err := r.queries.CountMachinesByDepartment(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.DepartmentMachineCountRow, len(rows))
	for i, row := range rows {
		out[i] = domain.DepartmentMachineCountRow{
			DepartmentID: row.DepartmentID,
			MachineCount: row.MachineCount,
		}
	}
	return out, nil
}
