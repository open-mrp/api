package repository

import (
	"context"
	gosql "database/sql"
	"errors"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var operatingCalendarRepoTracer = tracing.GetTracer("core-service.operating_calendar_repository")

type operatingCalendarRepoImpl struct {
	queries *sqlc.Queries
}

func NewOperatingCalendarRepo(queries *sqlc.Queries) domain.OperatingCalendarRepo {
	return &operatingCalendarRepoImpl{queries: queries}
}

func operatingCalendarFromRow(row sqlc.OperatingCalendar) domain.OperatingCalendar {
	cal := domain.OperatingCalendar{
		ID:         row.ID,
		AccountID:  row.AccountID,
		Code:       row.Code,
		Name:       row.Name,
		KindCode:   row.OperatingCalendarKindCode,
		DaysOfWeek: row.DaysOfWeek,
		IsDefault:  row.IsDefault,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
	if row.CutoffAt.Valid {
		cal.CutoffAt = &row.CutoffAt.String
	}
	if row.Timezone.Valid {
		cal.Timezone = &row.Timezone.String
	}
	return cal
}

func (r *operatingCalendarRepoImpl) Get(ctx context.Context, accountID, calendarID string) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.get")
	defer span.End()

	row, err := r.queries.GetOperatingCalendar(ctx, sqlc.GetOperatingCalendarParams{ID: calendarID, AccountID: accountID})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Operating calendar not found."))
		}
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to retrieve operating calendar."))
	}

	cal := operatingCalendarFromRow(row)
	return &cal, nil
}

func (r *operatingCalendarRepoImpl) GetByCode(ctx context.Context, accountID, code string) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.get_by_code")
	defer span.End()

	row, err := r.queries.GetOperatingCalendarByCode(ctx, sqlc.GetOperatingCalendarByCodeParams{AccountID: accountID, Code: code})
	if err != nil {
		// A missing code is how callers test whether a seed has already run, so it is nil rather than an error.
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to retrieve operating calendar."))
	}

	cal := operatingCalendarFromRow(row)
	return &cal, nil
}

func (r *operatingCalendarRepoImpl) List(ctx context.Context, params domain.ListOperatingCalendarsParams) ([]domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.list")
	defer span.End()

	rows, err := r.queries.ListOperatingCalendars(ctx, sqlc.ListOperatingCalendarsParams{
		AccountID: params.AccountID,
		KindCode:  db.NullStringPtr(params.KindCode),
	})
	if err != nil {
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to list operating calendars."))
	}

	out := make([]domain.OperatingCalendar, 0, len(rows))
	for _, row := range rows {
		out = append(out, operatingCalendarFromRow(row))
	}
	return out, nil
}

// ResolveShip and ResolveReceive both treat no row as nil rather than an error. An account that has configured no calendars is the ordinary state, and the caller's fallback — Monday to Friday with nothing closed — is exactly how every date resolved before calendars existed.
func (r *operatingCalendarRepoImpl) ResolveShip(ctx context.Context, accountID string) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.resolve_ship")
	defer span.End()

	row, err := r.queries.ResolveShipCalendar(ctx, sqlc.ResolveShipCalendarParams{AccountID: accountID})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to resolve shipping calendar."))
	}

	cal := operatingCalendarFromRow(row.OperatingCalendar)
	return &cal, nil
}

func (r *operatingCalendarRepoImpl) ResolveReceive(ctx context.Context, query domain.ReceiveCalendarQuery) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.resolve_receive")
	defer span.End()

	row, err := r.queries.ResolveReceiveCalendar(ctx, sqlc.ResolveReceiveCalendarParams{
		AccountID:      query.AccountID,
		AddressID:      db.NullStringPtr(query.AddressID),
		BuyerAccountID: db.NullStringPtr(query.BuyerAccountID),
	})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, nil
		}
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to resolve receiving calendar."))
	}

	cal := operatingCalendarFromRow(row.OperatingCalendar)
	return &cal, nil
}

func (r *operatingCalendarRepoImpl) ListClosures(ctx context.Context, query domain.ClosureWindowQuery) (map[string][]domain.OperatingCalendarClosure, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.list_closures")
	defer span.End()

	// An empty calendar set would generate `IN ()`, which is a syntax error rather than an empty result.
	if len(query.CalendarIDs) == 0 {
		return map[string][]domain.OperatingCalendarClosure{}, nil
	}

	rows, err := r.queries.ListOperatingCalendarClosures(ctx, sqlc.ListOperatingCalendarClosuresParams{
		AccountID:   query.AccountID,
		CalendarIds: query.CalendarIDs,
		FromDate:    query.From,
		ToDate:      query.To,
	})
	if err != nil {
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to list calendar closures."))
	}

	out := make(map[string][]domain.OperatingCalendarClosure, len(query.CalendarIDs))
	for _, row := range rows {
		out[row.OperatingCalendarID] = append(out[row.OperatingCalendarID], domain.OperatingCalendarClosure{
			ID:         row.ID,
			AccountID:  row.AccountID,
			CalendarID: row.OperatingCalendarID,
			ClosedOn:   row.ClosedOn,
			Name:       row.Name,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *operatingCalendarRepoImpl) Create(ctx context.Context, params domain.CreateOperatingCalendarParams) *apierror.APIError {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.create")
	defer span.End()

	err := r.queries.CreateOperatingCalendar(ctx, sqlc.CreateOperatingCalendarParams{
		ID:        params.ID,
		AccountID: params.AccountID,
		Code:      params.Code,
		Name:      params.Name,
		KindCode:  params.KindCode,
		// The column default only applies to an omitted column, and sqlc always sends every one.
		DaysOfWeek: params.DaysOfWeek,
		CutoffAt:   db.NullStringPtr(params.CutoffAt),
		Timezone:   db.NullStringPtr(params.Timezone),
		IsDefault:  params.IsDefault,
	})
	if err != nil {
		if db.IsDuplicateEntry(err) {
			return tracing.Trace(span, apierror.NewResourceConflictError("An operating calendar with this code already exists."))
		}
		return tracing.Trace(span, mapCalendarError(err, "Failed to create operating calendar."))
	}
	return nil
}

func (r *operatingCalendarRepoImpl) Update(ctx context.Context, params domain.UpdateOperatingCalendarParams) *apierror.APIError {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.update")
	defer span.End()

	err := r.queries.UpdateOperatingCalendar(ctx, sqlc.UpdateOperatingCalendarParams{
		ID:         params.ID,
		AccountID:  params.AccountID,
		Name:       db.NullStringPtr(params.Name),
		DaysOfWeek: db.NullStringPtr(params.DaysOfWeek),
		CutoffAt:   db.NullStringPtr(params.CutoffAt),
		Timezone:   db.NullStringPtr(params.Timezone),
		IsDefault:  boolPtrToNull(params.IsDefault),
	})
	if err != nil {
		return tracing.Trace(span, mapCalendarError(err, "Failed to update operating calendar."))
	}

	// Clearing runs as its own statement because a NULL argument above means "leave alone", so one parameter cannot express both.
	if params.ClearCutoffAt {
		if err := r.queries.ClearOperatingCalendarCutoff(ctx, sqlc.ClearOperatingCalendarCutoffParams{ID: params.ID, AccountID: params.AccountID}); err != nil {
			return tracing.Trace(span, mapCalendarError(err, "Failed to clear the pickup cutoff."))
		}
	}
	if params.ClearTimezone {
		if err := r.queries.ClearOperatingCalendarTimezone(ctx, sqlc.ClearOperatingCalendarTimezoneParams{ID: params.ID, AccountID: params.AccountID}); err != nil {
			return tracing.Trace(span, mapCalendarError(err, "Failed to clear the calendar timezone."))
		}
	}
	return nil
}

func (r *operatingCalendarRepoImpl) ClearDefault(ctx context.Context, accountID, kindCode, keepID string) *apierror.APIError {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.clear_default")
	defer span.End()

	err := r.queries.ClearOperatingCalendarDefault(ctx, sqlc.ClearOperatingCalendarDefaultParams{
		AccountID: accountID,
		KindCode:  kindCode,
		KeepID:    keepID,
	})
	if err != nil {
		return tracing.Trace(span, mapCalendarError(err, "Failed to update the default operating calendar."))
	}
	return nil
}

func (r *operatingCalendarRepoImpl) Delete(ctx context.Context, accountID, calendarID string) *apierror.APIError {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.delete")
	defer span.End()

	if err := r.queries.DeleteOperatingCalendar(ctx, sqlc.DeleteOperatingCalendarParams{ID: calendarID, AccountID: accountID}); err != nil {
		return tracing.Trace(span, mapCalendarError(err, "Failed to delete operating calendar."))
	}
	return nil
}

func (r *operatingCalendarRepoImpl) CountReferences(ctx context.Context, accountID, calendarID string) (*domain.OperatingCalendarReferences, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.count_references")
	defer span.End()

	row, err := r.queries.CountOperatingCalendarReferences(ctx, sqlc.CountOperatingCalendarReferencesParams{ID: db.NullString(calendarID)})
	if err != nil {
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to count operating calendar references."))
	}

	return &domain.OperatingCalendarReferences{
		Addresses: row.AddressCount,
		Customers: row.CustomerCount,
		Groups:    row.GroupCount,
		Settings:  row.SettingCount,
	}, nil
}

func (r *operatingCalendarRepoImpl) GetClosure(ctx context.Context, accountID, closureID string) (*domain.OperatingCalendarClosure, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.get_closure")
	defer span.End()

	row, err := r.queries.GetOperatingCalendarClosure(ctx, sqlc.GetOperatingCalendarClosureParams{ID: closureID, AccountID: accountID})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Calendar closure not found."))
		}
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to retrieve calendar closure."))
	}

	return &domain.OperatingCalendarClosure{
		ID:         row.ID,
		AccountID:  row.AccountID,
		CalendarID: row.OperatingCalendarID,
		ClosedOn:   row.ClosedOn,
		Name:       row.Name,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func (r *operatingCalendarRepoImpl) GetClosureByDate(ctx context.Context, accountID, calendarID string, closedOn time.Time) (*domain.OperatingCalendarClosure, *apierror.APIError) {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.get_closure_by_date")
	defer span.End()

	row, err := r.queries.GetOperatingCalendarClosureByDate(ctx, sqlc.GetOperatingCalendarClosureByDateParams{
		AccountID:  accountID,
		CalendarID: calendarID,
		ClosedOn:   closedOn,
	})
	if err != nil {
		if errors.Is(err, gosql.ErrNoRows) {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Calendar closure not found."))
		}
		return nil, tracing.Trace(span, mapCalendarError(err, "Failed to retrieve calendar closure."))
	}

	return &domain.OperatingCalendarClosure{
		ID:         row.ID,
		AccountID:  row.AccountID,
		CalendarID: row.OperatingCalendarID,
		ClosedOn:   row.ClosedOn,
		Name:       row.Name,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func (r *operatingCalendarRepoImpl) UpsertClosures(ctx context.Context, closures []domain.UpsertClosureParams) *apierror.APIError {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.upsert_closures")
	defer span.End()

	for _, c := range closures {
		err := r.queries.UpsertOperatingCalendarClosure(ctx, sqlc.UpsertOperatingCalendarClosureParams{
			ID:         c.ID,
			AccountID:  c.AccountID,
			CalendarID: c.CalendarID,
			ClosedOn:   c.ClosedOn,
			Name:       c.Name,
		})
		if err != nil {
			return tracing.Trace(span, mapCalendarError(err, "Failed to write calendar closures."))
		}
	}
	return nil
}

func (r *operatingCalendarRepoImpl) DeleteClosure(ctx context.Context, accountID, closureID string) *apierror.APIError {
	ctx, span := operatingCalendarRepoTracer.Start(ctx, "repository.operating_calendar.delete_closure")
	defer span.End()

	if err := r.queries.DeleteOperatingCalendarClosure(ctx, sqlc.DeleteOperatingCalendarClosureParams{ID: closureID, AccountID: accountID}); err != nil {
		return tracing.Trace(span, mapCalendarError(err, "Failed to delete calendar closure."))
	}
	return nil
}

func mapCalendarError(err error, message string) *apierror.APIError {
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	return apierror.NewInternalError(err, message)
}

// boolPtrToNull distinguishes an omitted flag from an explicit false, which COALESCE in the update statement reads as "leave alone".
func boolPtrToNull(b *bool) gosql.NullBool {
	if b == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: *b, Valid: true}
}
