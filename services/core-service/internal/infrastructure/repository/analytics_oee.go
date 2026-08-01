package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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

// GetOeeScanIntervals returns one row per scanned batch ticket with the machine it came off, ordered so consecutive scans on the same machine can be diffed.
func (r *analyticsRepoImpl) GetOeeScanIntervals(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeScanIntervalRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_scan_intervals")
	defer span.End()

	rows, err := r.queries.GetOeeScanIntervals(ctx, sqlc.GetOeeScanIntervalsParams{
		OwnerAccountID: params.AccountID,
		StartDate:      toRequiredNullTime(params.StartDate),
		EndDate:        toRequiredNullTime(params.EndDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeScanIntervalRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OeeScanIntervalRow{
			MachineID:    row.MachineID,
			DepartmentID: row.DepartmentID,
			IdealSeconds: decimalToFloat64(row.IdealSeconds),
		}
		if row.ScannedAt.Valid {
			scannedAt := row.ScannedAt.Time
			out[i].ScannedAt = &scannedAt
		}
	}
	return out, nil
}

// GetOeeMachineDowntimeIntervals lists raw downtime intervals per machine, unclipped (open events coalesce to now), so the caller can clip them against each scan gap.
func (r *analyticsRepoImpl) GetOeeMachineDowntimeIntervals(ctx context.Context, params domain.GetOeeWindowParams) ([]domain.OeeMachineDowntimeIntervalRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_machine_downtime_intervals")
	defer span.End()

	rows, err := r.queries.GetOeeMachineDowntimeIntervals(ctx, sqlc.GetOeeMachineDowntimeIntervalsParams{
		AccountID: params.AccountID,
		StartDate: toRequiredNullTime(params.StartDate),
		EndDate:   params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OeeMachineDowntimeIntervalRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OeeMachineDowntimeIntervalRow{
			MachineID: row.MachineID,
			OeeBucket: row.OeeBucket,
			StartedAt: row.StartedAt,
		}
		if row.EndedAt.Valid {
			endedAt := row.EndedAt.Time
			out[i].EndedAt = &endedAt
		}
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
