package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

// downtimeListColumns is the event and its reason, in the order scanDowntimeEvents reads them.
const downtimeListColumns = `e.id, e.account_id, e.machine_id, e.department_id, e.production_step_id, e.reason_code,
	r.name, r.oee_bucket, r.is_planned, e.started_at, e.ended_at, e.duration_seconds, e.shift_date, e.shift_code,
	e.item_id, e.production_run_id, e.batch_id, e.schedule_line_id, e.note, e.reported_by_id, e.source_code,
	e.created_at, e.updated_at`

// List pages an account's downtime events, newest started first. The page is chosen from
// machine_downtime_event alone (keysetPage) and the reason joined after: joined first, the planner
// hashes the reason table and sorts every event.
func (r *machineDowntimeRepoImpl) List(ctx context.Context, params domain.ListMachineDowntimeEventsParams) (*domain.ListMachineDowntimeEventsResult, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.list")
	defer span.End()

	page := keysetPage{
		table: "machine_downtime_event", alias: "e", sortColumn: "started_at",
		createdIndex: "machine_downtime_account_started_idx",
		filters: []keysetFilter{
			{column: "e.machine_id", index: "machine_downtime_account_machine_started_idx", values: params.MachineIDs},
			{column: "e.department_id", index: "machine_downtime_account_dept_started_idx", values: params.DepartmentIDs},
			{column: "e.reason_code", index: "machine_downtime_account_reason_started_idx", values: params.ReasonCodes},
			{column: "e.ended_at", index: "machine_downtime_account_ended_started_idx", isNull: params.OpenOnly},
		},
		where: []string{"e.account_id = ?"},
		args:  []any{params.AccountID},
		desc:  true,
		limit: params.Limit + 1,
	}
	// The note is the only prose an event carries; a substring match on it is residual to the keys.
	if search := dtSearchParam(params.Query); search.Valid {
		page.where, page.args = append(page.where, "e.note LIKE ?"), append(page.args, search.String)
	}
	if params.StartDate != nil {
		page.where, page.args = append(page.where, "e.started_at >= ?"), append(page.args, *params.StartDate)
	}
	if params.EndDate != nil {
		page.where, page.args = append(page.where, "e.started_at <= ?"), append(page.args, *params.EndDate)
	}

	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction
		if cur.Direction == pagination.DirectionBackward {
			page.where = append(page.where, "(e.started_at > ? OR (e.started_at = ? AND e.id > ?))")
			page.desc = false
		} else {
			page.where = append(page.where, "(e.started_at < ? OR (e.started_at = ? AND e.id < ?))")
		}
		page.args = append(page.args, cur.OccurredAt, cur.OccurredAt, cur.ID)
	}

	orderBy := " ORDER BY e.started_at DESC, e.id DESC"
	if !page.desc {
		orderBy = " ORDER BY e.started_at ASC, e.id ASC"
	}
	pageSQL, args := page.sql()
	rows, err := r.queries.DB().QueryContext(ctx, "SELECT "+downtimeListColumns+" FROM ("+pageSQL+") page"+
		" JOIN machine_downtime_event e ON e.id = page.id"+
		" LEFT JOIN machine_downtime_reason r ON r.code = e.reason_code"+orderBy, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()

	events := []*domain.MachineDowntimeEvent{}
	for rows.Next() {
		var f downtimeEventFields
		if err := rows.Scan(&f.ID, &f.AccountID, &f.MachineID, &f.DepartmentID, &f.ProductionStepID, &f.ReasonCode,
			&f.ReasonName, &f.ReasonOeeBucket, &f.ReasonIsPlanned, &f.StartedAt, &f.EndedAt, &f.DurationSeconds,
			&f.ShiftDate, &f.ShiftCode, &f.ItemID, &f.ProductionRunID, &f.BatchID, &f.ScheduleLineID, &f.Note,
			&f.ReportedByID, &f.SourceCode, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		events = append(events, mapDowntimeEvent(f))
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, pageInfo := pagination.BuildPageString(events, params.Limit, cursorDir, downtimeStartedAt, downtimeID)
	return &domain.ListMachineDowntimeEventsResult{Events: result, PageInfo: pageInfo}, nil
}
