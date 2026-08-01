package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var machineStatusRepoTracer = tracing.GetTracer("core-service.machine_status_repository")

type machineStatusRepoImpl struct {
	queries *sqlc.Queries
}

func NewMachineStatusRepo(queries *sqlc.Queries) domain.MachineStatusRepo {
	return &machineStatusRepoImpl{queries: queries}
}

// ListMachinesForStatus returns every machine that can carry work, whether or not the plan has given it any, ordered by name then id.
func (r *machineStatusRepoImpl) ListMachinesForStatus(ctx context.Context, accountID string) ([]domain.MachineForStatusRow, *apierror.APIError) {
	ctx, span := machineStatusRepoTracer.Start(ctx, "repository.machine_status.list_machines_for_status")
	defer span.End()

	rows, err := r.queries.ListMachinesForStatus(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.MachineForStatusRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.MachineForStatusRow{
			ID:             row.ID,
			Name:           row.Name,
			DepartmentID:   row.DepartmentID,
			DepartmentName: nullStringPtr(row.DepartmentName),
		})
	}
	return out, nil
}

// ListOpenDowntimeForStatus returns the machines that are down right now — one row per machine, as the open-event guard enforces on write.
func (r *machineStatusRepoImpl) ListOpenDowntimeForStatus(ctx context.Context, accountID string) ([]domain.OpenDowntimeForStatusRow, *apierror.APIError) {
	ctx, span := machineStatusRepoTracer.Start(ctx, "repository.machine_status.list_open_downtime_for_status")
	defer span.End()

	rows, err := r.queries.ListOpenDowntimeForStatus(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OpenDowntimeForStatusRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.OpenDowntimeForStatusRow{
			ID:         row.ID,
			MachineID:  row.MachineID,
			ReasonCode: row.ReasonCode,
			ReasonName: nullStringPtr(row.ReasonName),
			OEEBucket:  nullStringPtr(row.OeeBucket),
			StartedAt:  row.StartedAt,
			Note:       nullStringPtr(row.Note),
		})
	}
	return out, nil
}

// ListScheduleLinesForStatus returns a published schedule's lines from the given week forward, with per-campaign scan progress, ordered by machine, week, sequence, id.
func (r *machineStatusRepoImpl) ListScheduleLinesForStatus(ctx context.Context, params domain.ListScheduleLinesForStatusParams) ([]domain.ScheduleLineForStatusRow, *apierror.APIError) {
	ctx, span := machineStatusRepoTracer.Start(ctx, "repository.machine_status.list_schedule_lines_for_status")
	defer span.End()

	rows, err := r.queries.ListScheduleLinesForStatus(ctx, sqlc.ListScheduleLinesForStatusParams{
		AccountID:            params.AccountID,
		ProductionScheduleID: params.ProductionScheduleID,
		FromWeek:             params.FromWeek,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.ScheduleLineForStatusRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ScheduleLineForStatusRow{
			ID:                      row.ID,
			MachineID:               row.MachineID,
			ItemID:                  row.ItemID,
			WeekIndex:               row.WeekIndex,
			WeekStartDate:           row.WeekStartDate,
			PlannedQuantity:         decimalToFloat64(row.PlannedQuantity),
			PlannedRunHours:         decimalToFloat64(row.PlannedRunHours),
			StatusCode:              row.StatusCode,
			ProductionRunID:         nullStringPtr(row.ProductionRunID),
			PlannedUnitAbbreviation: nullStringPtr(row.PlannedUnitAbbreviation),
			SKU:                     nullStringPtr(row.Sku),
			ReleasedBatchCount:      int64(decimalToFloat64(row.ReleasedBatchCount)),
			ScannedBatchCount:       int64(decimalToFloat64(row.ScannedBatchCount)),
			ScannedQuantity:         decimalToFloat64(row.ScannedQuantity),
		})
	}
	return out, nil
}
