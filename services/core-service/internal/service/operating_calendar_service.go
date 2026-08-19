package service

import (
	"context"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

var operatingCalendarSvcTracer = tracing.GetTracer("core-service.operating_calendar_service")

// closureListWindowYears bounds a closure listing when the caller names no dates. A calendar accumulates closures forever, and a settings page wants the year either side of today, not every holiday since the account opened.
const closureListWindowYears = 1

type operatingCalendarSvcImpl struct {
	repos domain.RepoFactory
}

type OperatingCalendarSvcConfig struct {
	Repos domain.RepoFactory
}

func NewOperatingCalendarSvc(config *OperatingCalendarSvcConfig) domain.OperatingCalendarSvc {
	return &operatingCalendarSvcImpl{repos: config.Repos}
}

// actingAccount resolves the acting account and rejects a non-internal caller. Calendars are account configuration, so every path here is internal-only: a customer has no business seeing which days their supplier's plant runs.
//
// The permission check is deliberately left to each method rather than folded in here. The drift guard reads for a literal check per handler, and an endpoint whose permissions are only visible one indirection away is exactly what that guard exists to catch.
func (s *operatingCalendarSvcImpl) actingAccount(ctx context.Context) (*types.Identity, string, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, "", apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, "", apiErr
	}
	return identity, identity.Target.AccountID, nil
}

func (s *operatingCalendarSvcImpl) ListOperatingCalendars(ctx context.Context, kindCode *string) ([]domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.list")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := validateCalendarKind(kindCode); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	calendars, apiErr := s.repos.NewOperatingCalendarRepo().List(ctx, domain.ListOperatingCalendarsParams{AccountID: accountID, KindCode: kindCode})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return calendars, nil
}

func (s *operatingCalendarSvcImpl) GetOperatingCalendar(ctx context.Context, calendarID string) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.get")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	calendar, apiErr := s.repos.NewOperatingCalendarRepo().Get(ctx, accountID, calendarID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return calendar, nil
}

func (s *operatingCalendarSvcImpl) CreateOperatingCalendar(ctx context.Context, params domain.CreateOperatingCalendarParams) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.create")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = accountID

	if apiErr := validateCalendarShape(params.KindCode, params.DaysOfWeek, params.CutoffAt, params.Timezone); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	calendarID, apiErr := id.GenID(id.OperatingCalendarIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.ID = calendarID

	repo := s.repos.NewOperatingCalendarRepo()
	if apiErr := repo.Create(ctx, params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Exactly one default per kind, or resolution has two answers and picks arbitrarily.
	if params.IsDefault {
		if apiErr := repo.ClearDefault(ctx, accountID, params.KindCode, calendarID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return repo.Get(ctx, accountID, calendarID)
}

func (s *operatingCalendarSvcImpl) UpdateOperatingCalendar(ctx context.Context, params domain.UpdateOperatingCalendarParams) (*domain.OperatingCalendar, *apierror.APIError) {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.update")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = accountID

	repo := s.repos.NewOperatingCalendarRepo()
	existing, apiErr := repo.Get(ctx, accountID, params.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Validated against the calendar's own kind, since the kind cannot be changed: a ship calendar that became a receive calendar would silently drop the cutoff every commitment on it depends on.
	if apiErr := validateCalendarShape(existing.KindCode, valueOr(params.DaysOfWeek, existing.DaysOfWeek), params.CutoffAt, params.Timezone); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := repo.Update(ctx, params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.IsDefault != nil && *params.IsDefault {
		if apiErr := repo.ClearDefault(ctx, accountID, existing.KindCode, params.ID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return repo.Get(ctx, accountID, params.ID)
}

// DeleteOperatingCalendar refuses while anything still points at the calendar.
//
// The alternative — deleting and letting the links dangle — would silently return every affected customer to Monday-to-Friday, which reads as the feature quietly breaking rather than as a decision anybody made.
func (s *operatingCalendarSvcImpl) DeleteOperatingCalendar(ctx context.Context, calendarID string) *apierror.APIError {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.delete")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewOperatingCalendarRepo()
	if _, apiErr := repo.Get(ctx, accountID, calendarID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	refs, apiErr := repo.CountReferences(ctx, accountID, calendarID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if refs.Total() > 0 {
		return tracing.Trace(span, apierror.NewResourceConflictError("This calendar is still in use. Point the addresses, customers, groups, or account settings that reference it somewhere else first."))
	}

	return tracing.Trace(span, repo.Delete(ctx, accountID, calendarID))
}

func (s *operatingCalendarSvcImpl) ListOperatingCalendarClosures(ctx context.Context, calendarID string, from, to *time.Time) ([]domain.OperatingCalendarClosure, *apierror.APIError) {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.list_closures")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewOperatingCalendarRepo()
	if _, apiErr := repo.Get(ctx, accountID, calendarID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	now := time.Now()
	window := domain.ClosureWindowQuery{
		AccountID:   accountID,
		CalendarIDs: []string{calendarID},
		From:        now.AddDate(-closureListWindowYears, 0, 0),
		To:          now.AddDate(closureListWindowYears, 0, 0),
	}
	if from != nil {
		window.From = *from
	}
	if to != nil {
		window.To = *to
	}
	if window.To.Before(window.From) {
		return nil, tracing.Trace(span, apierror.NewValidationError("to_date must not fall before from_date."))
	}

	byCalendar, apiErr := repo.ListClosures(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return byCalendar[calendarID], nil
}

func (s *operatingCalendarSvcImpl) CreateOperatingCalendarClosure(ctx context.Context, calendarID string, closedOn time.Time, name string) (*domain.OperatingCalendarClosure, *apierror.APIError) {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.create_closure")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewOperatingCalendarRepo()
	if _, apiErr := repo.Get(ctx, accountID, calendarID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	closureID, apiErr := id.GenID(id.OperatingCalendarClosureIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Truncated to a day because a closure is a date, and an instant carrying a time would compare unequal to the dates every calendar walk tests against.
	closedOn = time.Date(closedOn.Year(), closedOn.Month(), closedOn.Day(), 0, 0, 0, 0, time.UTC)

	if apiErr := repo.UpsertClosures(ctx, []domain.UpsertClosureParams{{
		ID:         closureID,
		AccountID:  accountID,
		CalendarID: calendarID,
		ClosedOn:   closedOn,
		Name:       name,
	}}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Read back rather than returned from the inputs: the write is an upsert, so re-closing a date that was already closed keeps the original row, and its ID and timestamps are the ones the caller should see.
	return repo.GetClosureByDate(ctx, accountID, calendarID, closedOn)
}

func (s *operatingCalendarSvcImpl) DeleteOperatingCalendarClosure(ctx context.Context, closureID string) *apierror.APIError {
	ctx, span := operatingCalendarSvcTracer.Start(ctx, "service.operating_calendar.delete_closure")
	defer span.End()

	identity, accountID, apiErr := s.actingAccount(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	repo := s.repos.NewOperatingCalendarRepo()
	if _, apiErr := repo.GetClosure(ctx, accountID, closureID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return tracing.Trace(span, repo.DeleteClosure(ctx, accountID, closureID))
}

func validateCalendarKind(kindCode *string) *apierror.APIError {
	if kindCode == nil || *kindCode == "" {
		return nil
	}
	if !constants.OperatingCalendarKind(*kindCode).IsValid() {
		return apierror.NewValidationErrorWithParam("Unknown operating calendar kind.", "kind_code")
	}
	return nil
}

// validateCalendarShape rejects a calendar that cannot resolve a date.
//
// The day mask is parsed rather than pattern-matched so the write path and the engine agree exactly on what a valid mask is — including the all-closed case, which would send every commitment to the snap-back limit.
func validateCalendarShape(kindCode, daysOfWeek string, cutoffAt, timezone *string) *apierror.APIError {
	if !constants.OperatingCalendarKind(kindCode).IsValid() {
		return apierror.NewValidationErrorWithParam("Unknown operating calendar kind.", "kind_code")
	}
	if _, err := scheduling.ParseDayMask(daysOfWeek); err != nil {
		return apierror.NewValidationErrorWithParam("days_of_week must be seven characters of '0' or '1', Monday first, with at least one open day.", "days_of_week")
	}
	if cutoffAt != nil && *cutoffAt != "" {
		if kindCode != string(constants.OperatingCalendarKindShip) {
			return apierror.NewValidationErrorWithParam("Only a shipping calendar carries a pickup cutoff.", "cutoff_at")
		}
		if _, err := time.Parse("15:04", *cutoffAt); err != nil {
			return apierror.NewValidationErrorWithParam("cutoff_at must be a 24-hour local time, such as \"15:00\".", "cutoff_at")
		}
	}
	if timezone != nil && *timezone != "" {
		if _, err := time.LoadLocation(*timezone); err != nil {
			return apierror.NewValidationErrorWithParam("timezone must be an IANA zone name, such as \"America/Chicago\".", "timezone")
		}
	}
	return nil
}

func valueOr(v *string, fallback string) string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return fallback
	}
	return *v
}
