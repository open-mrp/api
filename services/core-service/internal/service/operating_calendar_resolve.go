package service

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/timeutil"
	"github.com/augno/api/shared/tracing"
)

var operatingCalendarTracer = tracing.GetTracer("core-service.operating_calendar")

// closureWindowBefore and closureWindowAfter bound how much of an account's closure history is read to resolve one commitment.
//
// Asymmetric because the walk is: a ship-by date is worked backwards from a delivery date, so almost the whole window needed is in the past relative to it. Half a year covers a transit of any plausible length across a shutdown, and reading the whole table instead would make every issue slower as an account accumulates years of holidays.
const (
	closureWindowBefore = 180 * 24 * time.Hour
	closureWindowAfter  = 30 * 24 * time.Hour
)

// resolveCommitmentCalendars gathers the three day-sets and two zones a commitment is resolved against.
//
// Every lookup degrades to the Monday-to-Friday default rather than failing. An account that has configured no calendars is the ordinary state and must keep getting exactly the dates it got before calendars existed; a commitment refused because nobody had filled in a settings page would be a far worse outcome than one resolved against sensible defaults.
//
// Runs inside the issue transaction, so the cost is deliberately fixed: two indexed seeks for the calendars and one bounded range read for their closures, regardless of how much history the account has.
func (s *salesOrderSvcImpl) resolveCommitmentCalendars(ctx context.Context, accountID string, order *domain.SalesOrder, anchor time.Time) (scheduling.Calendars, *apierror.APIError) {
	ctx, span := operatingCalendarTracer.Start(ctx, "service.operating_calendar.resolve_commitment_calendars")
	defer span.End()

	cals := scheduling.DefaultCalendars()
	repo := s.repos.NewOperatingCalendarRepo()

	shipCal, apiErr := repo.ResolveShip(ctx, accountID)
	if apiErr != nil {
		return cals, tracing.Trace(span, apiErr)
	}

	receiveCal, apiErr := repo.ResolveReceive(ctx, domain.ReceiveCalendarQuery{
		AccountID:      accountID,
		BuyerAccountID: stringPtrOrNil(order.BuyerAccountID),
		AddressID:      stringPtrOrNil(order.ShippingAddressID),
	})
	if apiErr != nil {
		return cals, tracing.Trace(span, apiErr)
	}

	closures, apiErr := s.loadClosures(ctx, accountID, anchor, shipCal, receiveCal)
	if apiErr != nil {
		return cals, tracing.Trace(span, apiErr)
	}

	if shipCal != nil {
		if built, err := scheduling.NewCalendar(shipCal.DaysOfWeek, closureDates(closures[shipCal.ID])); err == nil {
			cals.Ship = built
			// The carrier does not keep the plant's weekday pattern, but it does stop for the same holidays: a US federal holiday is precisely a day the network is not moving. Its own weekdays stay Monday to Friday.
			if carrier, err := scheduling.NewCalendar("1111100", closureDates(closures[shipCal.ID])); err == nil {
				cals.Carrier = carrier
			}
		}
		if shipCal.CutoffAt != nil {
			cals.ShipCutoff = *shipCal.CutoffAt
		}
		cals.ShipLocation = timeutil.ZoneFor(shipCal.Timezone, "", "", s.accountFallbackZone(ctx, accountID))
	}

	if receiveCal != nil {
		if built, err := scheduling.NewCalendar(receiveCal.DaysOfWeek, closureDates(closures[receiveCal.ID])); err == nil {
			cals.Receive = built
		}
	}

	return cals, nil
}

// loadClosures reads one bounded window covering both calendars in a single query.
func (s *salesOrderSvcImpl) loadClosures(ctx context.Context, accountID string, anchor time.Time, calendars ...*domain.OperatingCalendar) (map[string][]domain.OperatingCalendarClosure, *apierror.APIError) {
	ids := make([]string, 0, len(calendars))
	for _, cal := range calendars {
		if cal != nil {
			ids = append(ids, cal.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	return s.repos.NewOperatingCalendarRepo().ListClosures(ctx, domain.ClosureWindowQuery{
		AccountID:   accountID,
		CalendarIDs: ids,
		From:        anchor.Add(-closureWindowBefore),
		To:          anchor.Add(closureWindowAfter),
	})
}

// accountFallbackZone is the account's own zone, the last stop before UTC. An account's addresses cluster near it, so a Denver plant shipping somewhere unrecognised is far better served by Denver time.
func (s *salesOrderSvcImpl) accountFallbackZone(ctx context.Context, accountID string) string {
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil || settings == nil {
		return ""
	}
	return settings.GenerationTimezone
}

func closureDates(closures []domain.OperatingCalendarClosure) []time.Time {
	if len(closures) == 0 {
		return nil
	}
	out := make([]time.Time, 0, len(closures))
	for _, c := range closures {
		out = append(out, c.ClosedOn)
	}
	return out
}

// stringPtrOrNil keeps an unset identifier out of the resolution chain, where an empty string would match nothing but still cost a comparison.
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
