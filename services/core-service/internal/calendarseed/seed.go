// Package calendarseed gives a new account the operating calendars a ship-by commitment is resolved against.
//
// Its own package because both account registration and sandbox provisioning need it, and the sandbox mediator cannot import the service layer that owns registration without a cycle.
package calendarseed

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
)

// closureSeedYears is how far ahead a seed writes holidays.
//
// Three years rather than one so a plant quoting long lead times still resolves against real holidays, and so the horizon does not have to be topped up the moment a year turns. It is refreshed on every seed, which is idempotent, so a long-lived account keeps drifting forward rather than running out.
const closureSeedYears = 3

// Seed gives an account the two calendars a commitment needs, seeded with US federal holidays.
//
// Seeded rather than left empty so the feature does something on day one: an account that never opens the settings page still stops committing orders to Thanksgiving. Everything written here is an ordinary editable row — a plant that runs through Columbus Day or ships Saturdays changes it, and nothing about the federal list binds a private factory.
//
// Idempotent by code, so re-running it on an account that already has calendars tops up the closure horizon and touches nothing else. That is what makes it safe to call from registration, from a sandbox, and from a backfill over existing accounts.
func Seed(ctx context.Context, repos domain.RepoFactory, accountID string, asOf time.Time) *apierror.APIError {
	repo := repos.NewOperatingCalendarRepo()

	for _, seed := range []domain.CreateOperatingCalendarParams{defaultShipCalendarSeed, defaultReceiveCalendarSeed} {
		calendarID, apiErr := ensureSeedCalendar(ctx, repo, accountID, seed)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := seedFederalClosures(ctx, repo, accountID, calendarID, asOf); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// ensureSeedCalendar returns the ID of the account's seeded calendar, creating it only if its code is not already taken. An operator who has renamed or re-dayed it keeps their version.
func ensureSeedCalendar(ctx context.Context, repo domain.OperatingCalendarRepo, accountID string, seed domain.CreateOperatingCalendarParams) (string, *apierror.APIError) {
	existing, apiErr := repo.GetByCode(ctx, accountID, seed.Code)
	if apiErr != nil {
		return "", apiErr
	}
	if existing != nil {
		return existing.ID, nil
	}

	calendarID, apiErr := id.GenID(id.OperatingCalendarIDPrefix, nil)
	if apiErr != nil {
		return "", apiErr
	}

	params := seed
	params.ID = calendarID
	params.AccountID = accountID
	if params.KindCode == string(constants.OperatingCalendarKindShip) {
		cutoff := defaultPickupCutoff
		params.CutoffAt = &cutoff
	}
	if apiErr := repo.Create(ctx, params); apiErr != nil {
		return "", apiErr
	}
	return calendarID, nil
}

// seedFederalClosures writes this year's and the next two years' federal holidays onto a calendar. The write is an upsert keyed on the date, so a closure an operator has already relabelled or deliberately deleted-and-re-added is not disturbed.
func seedFederalClosures(ctx context.Context, repo domain.OperatingCalendarRepo, accountID, calendarID string, asOf time.Time) *apierror.APIError {
	closures := make([]domain.UpsertClosureParams, 0, closureSeedYears*11)

	for offset := range closureSeedYears {
		for _, holiday := range scheduling.USFederalHolidays(asOf.Year() + offset) {
			closureID, apiErr := id.GenID(id.OperatingCalendarClosureIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			closures = append(closures, domain.UpsertClosureParams{
				ID:         closureID,
				AccountID:  accountID,
				CalendarID: calendarID,
				ClosedOn:   holiday.Date,
				Name:       holiday.Name,
			})
		}
	}

	return repo.UpsertClosures(ctx, closures)
}

// defaultPickupCutoff is the local time a seeded plant is assumed to tender freight by. Late enough that a normal working day still makes it, early enough to be a real deadline rather than a formality.
const defaultPickupCutoff = "15:00"

// defaultShipCalendarSeed and defaultReceiveCalendarSeed are the calendars an account starts with: a plant tendering freight Monday to Friday with a 3pm pickup, and a dock receiving Monday to Friday.
//
// Monday to Friday rather than anything cleverer because it is what every date resolved to before calendars existed, so a newly seeded account sees no change to its dates until it says otherwise. The holidays are the one thing that does change, which is the point.
var (
	defaultShipCalendarSeed = domain.CreateOperatingCalendarParams{
		Code:       "default_ship",
		Name:       "Shipping days",
		KindCode:   string(constants.OperatingCalendarKindShip),
		DaysOfWeek: "1111100",
		IsDefault:  true,
	}
	defaultReceiveCalendarSeed = domain.CreateOperatingCalendarParams{
		Code:       "default_receive",
		Name:       "Customer receiving days",
		KindCode:   string(constants.OperatingCalendarKindReceive),
		DaysOfWeek: "1111100",
		IsDefault:  true,
	}
)
