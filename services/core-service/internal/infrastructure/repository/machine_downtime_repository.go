package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var machineDowntimeRepoTracer = tracing.GetTracer("core-service.machine_downtime_repository")

type machineDowntimeRepoImpl struct {
	queries *sqlc.Queries
}

func NewMachineDowntimeRepo(queries *sqlc.Queries) domain.MachineDowntimeRepo {
	return &machineDowntimeRepoImpl{queries: queries}
}

func downtimeStartedAt(e *domain.MachineDowntimeEvent) time.Time { return e.StartedAt }
func downtimeID(e *domain.MachineDowntimeEvent) string           { return e.ID }

// dtNullString is deliberately not db.NullStringPtr: that helper coerces a pointer to empty string to SQL NULL, whereas the downtime writes must preserve &"" as a valid ” value (department_id is NOT NULL on machine, with ” meaning unassigned).
func dtNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

// dtNullTime is kept only for demand_override_repository.go; machine-downtime code uses db.NullTimePtr. Delete once the demand-override call sites migrate too.
func dtNullTime(t *time.Time) gosql.NullTime {
	if t == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *t, Valid: true}
}

func dtNullInt32(i *int32) gosql.NullInt32 {
	if i == nil {
		return gosql.NullInt32{}
	}
	return gosql.NullInt32{Int32: *i, Valid: true}
}

// dtNullStrings converts an id filter to the nullable slice sqlc generates for nullable columns (department_id is nullable, so its IN-list is []sql.NullString).
func dtNullStrings(values []string) []gosql.NullString {
	out := make([]gosql.NullString, len(values))
	for i, v := range values {
		out[i] = gosql.NullString{String: v, Valid: true}
	}
	return out
}

func mapDowntimeReason(row sqlc.MachineDowntimeReason) *domain.MachineDowntimeReason {
	return &domain.MachineDowntimeReason{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		OeeBucket: row.OeeBucket,
		IsPlanned: row.IsPlanned,
		SortOrder: row.SortOrder,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// downtimeEventFields is the shared column set every downtime query selects. The per-query row structs sqlc generates are distinct types with identical shapes, so each mapper funnels into this to keep the field wiring in one place.
type downtimeEventFields struct {
	ID               string
	AccountID        string
	MachineID        string
	DepartmentID     gosql.NullString
	ProductionStepID gosql.NullString
	ReasonCode       string
	ReasonName       gosql.NullString
	ReasonOeeBucket  gosql.NullString
	ReasonIsPlanned  gosql.NullBool
	StartedAt        time.Time
	EndedAt          gosql.NullTime
	DurationSeconds  gosql.NullInt32
	ShiftDate        time.Time
	ShiftCode        gosql.NullString
	ItemID           gosql.NullString
	ProductionRunID  gosql.NullString
	BatchID          gosql.NullString
	ScheduleLineID   gosql.NullString
	Note             gosql.NullString
	ReportedByID     string
	SourceCode       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func mapDowntimeEvent(f downtimeEventFields) *domain.MachineDowntimeEvent {
	e := &domain.MachineDowntimeEvent{
		ID:           f.ID,
		AccountID:    f.AccountID,
		MachineID:    f.MachineID,
		ReasonCode:   f.ReasonCode,
		StartedAt:    f.StartedAt,
		ShiftDate:    f.ShiftDate,
		ReportedByID: f.ReportedByID,
		SourceCode:   f.SourceCode,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
	e.DepartmentID = db.StringFromNullString(f.DepartmentID)
	e.ProductionStepID = db.StringFromNullString(f.ProductionStepID)
	e.ReasonName = db.StringFromNullString(f.ReasonName)
	e.ReasonOeeBucket = db.StringFromNullString(f.ReasonOeeBucket)
	if f.ReasonIsPlanned.Valid {
		e.ReasonIsPlanned = &f.ReasonIsPlanned.Bool
	}
	e.EndedAt = db.TimeFromNullTime(f.EndedAt)
	if f.DurationSeconds.Valid {
		e.DurationSeconds = &f.DurationSeconds.Int32
	}
	e.ShiftCode = db.StringFromNullString(f.ShiftCode)
	e.ItemID = db.StringFromNullString(f.ItemID)
	e.ProductionRunID = db.StringFromNullString(f.ProductionRunID)
	e.BatchID = db.StringFromNullString(f.BatchID)
	e.ScheduleLineID = db.StringFromNullString(f.ScheduleLineID)
	e.Note = db.StringFromNullString(f.Note)
	return e
}

func (r *machineDowntimeRepoImpl) ListReasons(ctx context.Context) ([]*domain.MachineDowntimeReason, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.list_reasons")
	defer span.End()

	rows, err := r.queries.ListMachineDowntimeReasons(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	reasons := make([]*domain.MachineDowntimeReason, len(rows))
	for i, row := range rows {
		reasons[i] = mapDowntimeReason(row)
	}
	return reasons, nil
}

func (r *machineDowntimeRepoImpl) GetReason(ctx context.Context, code string) (*domain.MachineDowntimeReason, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.get_reason")
	defer span.End()

	row, err := r.queries.GetMachineDowntimeReason(ctx, code)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapDowntimeReason(row), nil
}

// dtSearchParam wraps a search term for a substring match against the note.
func dtSearchParam(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + *query + "%", Valid: true}
}

func (r *machineDowntimeRepoImpl) List(ctx context.Context, params domain.ListMachineDowntimeEventsParams) (*domain.ListMachineDowntimeEventsResult, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.list")
	defer span.End()

	searchQuery := dtSearchParam(params.Query)
	startDate := db.NullTimePtr(params.StartDate)
	endDate := db.NullTimePtr(params.EndDate)
	includeMachineFilter := len(params.MachineIDs) > 0
	includeDepartmentFilter := len(params.DepartmentIDs) > 0
	includeReasonFilter := len(params.ReasonCodes) > 0

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListMachineDowntimeEventsBackward(ctx, sqlc.ListMachineDowntimeEventsBackwardParams{
				AccountID:               params.AccountID,
				IncludeMachineFilter:    includeMachineFilter,
				MachineIds:              params.MachineIDs,
				IncludeDepartmentFilter: includeDepartmentFilter,
				DepartmentIds:           dtNullStrings(params.DepartmentIDs),
				IncludeReasonFilter:     includeReasonFilter,
				ReasonCodes:             params.ReasonCodes,
				OpenOnly:                params.OpenOnly,
				SearchQuery:             searchQuery,
				StartDate:               startDate,
				EndDate:                 endDate,
				CursorStartedAt:         cur.OccurredAt,
				CursorID:                cur.ID,
				Limit:                   params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			events := make([]*domain.MachineDowntimeEvent, len(rows))
			for i, row := range rows {
				events[i] = mapDowntimeEvent(downtimeEventFields(row))
			}
			result, pageInfo := pagination.BuildPageString(events, params.Limit, cursorDir, downtimeStartedAt, downtimeID)
			return &domain.ListMachineDowntimeEventsResult{Events: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListMachineDowntimeEventsForward(ctx, sqlc.ListMachineDowntimeEventsForwardParams{
			AccountID:               params.AccountID,
			IncludeMachineFilter:    includeMachineFilter,
			MachineIds:              params.MachineIDs,
			IncludeDepartmentFilter: includeDepartmentFilter,
			DepartmentIds:           dtNullStrings(params.DepartmentIDs),
			IncludeReasonFilter:     includeReasonFilter,
			ReasonCodes:             params.ReasonCodes,
			OpenOnly:                params.OpenOnly,
			SearchQuery:             searchQuery,
			StartDate:               startDate,
			EndDate:                 endDate,
			CursorStartedAt:         gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:                gosql.NullString{String: cur.ID, Valid: true},
			Limit:                   params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		events := make([]*domain.MachineDowntimeEvent, len(rows))
		for i, row := range rows {
			events[i] = mapDowntimeEvent(downtimeEventFields(row))
		}
		result, pageInfo := pagination.BuildPageString(events, params.Limit, cursorDir, downtimeStartedAt, downtimeID)
		return &domain.ListMachineDowntimeEventsResult{Events: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListMachineDowntimeEventsForward(ctx, sqlc.ListMachineDowntimeEventsForwardParams{
		AccountID:               params.AccountID,
		IncludeMachineFilter:    includeMachineFilter,
		MachineIds:              params.MachineIDs,
		IncludeDepartmentFilter: includeDepartmentFilter,
		DepartmentIds:           dtNullStrings(params.DepartmentIDs),
		IncludeReasonFilter:     includeReasonFilter,
		ReasonCodes:             params.ReasonCodes,
		OpenOnly:                params.OpenOnly,
		SearchQuery:             searchQuery,
		StartDate:               startDate,
		EndDate:                 endDate,
		Limit:                   params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	events := make([]*domain.MachineDowntimeEvent, len(rows))
	for i, row := range rows {
		events[i] = mapDowntimeEvent(downtimeEventFields(row))
	}
	result, pageInfo := pagination.BuildPageString(events, params.Limit, cursorDir, downtimeStartedAt, downtimeID)
	return &domain.ListMachineDowntimeEventsResult{Events: result, PageInfo: pageInfo}, nil
}

func (r *machineDowntimeRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetMachineDowntimeEventsByIDs(ctx, sqlc.GetMachineDowntimeEventsByIDsParams{
		AccountID: accountID,
		Ids:       ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	events := make([]*domain.MachineDowntimeEvent, len(rows))
	for i, row := range rows {
		events[i] = mapDowntimeEvent(downtimeEventFields(row))
	}
	return events, nil
}

func (r *machineDowntimeRepoImpl) Get(ctx context.Context, params domain.GetMachineDowntimeEventParams) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.get")
	defer span.End()

	row, err := r.queries.GetMachineDowntimeEvent(ctx, sqlc.GetMachineDowntimeEventParams{
		AccountID: params.AccountID,
		ID:        params.EventID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapDowntimeEvent(downtimeEventFields(row)), nil
}

func (r *machineDowntimeRepoImpl) GetOpenForMachine(ctx context.Context, accountID, machineID string) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.get_open_for_machine")
	defer span.End()

	row, err := r.queries.GetOpenMachineDowntimeEventForMachine(ctx, sqlc.GetOpenMachineDowntimeEventForMachineParams{
		AccountID: accountID,
		MachineID: machineID,
	})
	if err == gosql.ErrNoRows {
		// Not an error: the machine is simply running.
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapDowntimeEvent(downtimeEventFields(row)), nil
}

func (r *machineDowntimeRepoImpl) Create(ctx context.Context, id string, event *domain.MachineDowntimeEvent) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.create")
	defer span.End()

	err := r.queries.CreateMachineDowntimeEvent(ctx, sqlc.CreateMachineDowntimeEventParams{
		ID:               id,
		AccountID:        event.AccountID,
		MachineID:        event.MachineID,
		DepartmentID:     dtNullString(event.DepartmentID),
		ProductionStepID: dtNullString(event.ProductionStepID),
		ReasonCode:       event.ReasonCode,
		StartedAt:        event.StartedAt,
		EndedAt:          db.NullTimePtr(event.EndedAt),
		DurationSeconds:  dtNullInt32(event.DurationSeconds),
		ShiftDate:        event.ShiftDate,
		ShiftCode:        dtNullString(event.ShiftCode),
		ItemID:           dtNullString(event.ItemID),
		ProductionRunID:  dtNullString(event.ProductionRunID),
		BatchID:          dtNullString(event.BatchID),
		ScheduleLineID:   dtNullString(event.ScheduleLineID),
		Note:             dtNullString(event.Note),
		ReportedByID:     event.ReportedByID,
		SourceCode:       event.SourceCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetMachineDowntimeEventParams{AccountID: event.AccountID, EventID: id})
}

func (r *machineDowntimeRepoImpl) Update(ctx context.Context, event *domain.MachineDowntimeEvent) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.update")
	defer span.End()

	err := r.queries.UpdateMachineDowntimeEvent(ctx, sqlc.UpdateMachineDowntimeEventParams{
		AccountID:        event.AccountID,
		ID:               event.ID,
		MachineID:        event.MachineID,
		DepartmentID:     dtNullString(event.DepartmentID),
		ProductionStepID: dtNullString(event.ProductionStepID),
		ReasonCode:       event.ReasonCode,
		StartedAt:        event.StartedAt,
		EndedAt:          db.NullTimePtr(event.EndedAt),
		DurationSeconds:  dtNullInt32(event.DurationSeconds),
		ShiftDate:        event.ShiftDate,
		ShiftCode:        dtNullString(event.ShiftCode),
		ItemID:           dtNullString(event.ItemID),
		ProductionRunID:  dtNullString(event.ProductionRunID),
		BatchID:          dtNullString(event.BatchID),
		ScheduleLineID:   dtNullString(event.ScheduleLineID),
		Note:             dtNullString(event.Note),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetMachineDowntimeEventParams{AccountID: event.AccountID, EventID: event.ID})
}

func (r *machineDowntimeRepoImpl) Delete(ctx context.Context, params domain.DeleteMachineDowntimeEventParams) *apierror.APIError {
	ctx, span := machineDowntimeRepoTracer.Start(ctx, "repository.machine_downtime.delete")
	defer span.End()

	err := r.queries.DeleteMachineDowntimeEvent(ctx, sqlc.DeleteMachineDowntimeEventParams{
		AccountID: params.AccountID,
		ID:        params.EventID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
