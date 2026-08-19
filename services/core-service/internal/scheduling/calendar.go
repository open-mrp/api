package scheduling

import (
	"fmt"
	"strings"
	"time"
)

// dayMaskLength is the width of a days-of-week mask, one character per ISO weekday starting Monday. Matches the `days_of_week` column shared with production_shift, so a plant's shipping days and a shift's working days are written the same way.
const dayMaskLength = 7

// snapBackLimit bounds how far SnapBack will walk looking for an open day. A calendar with a live weekday should never need more than a week; the limit is generous because a shutdown fortnight is legitimate, and finite because a calendar closed indefinitely must surface as an unresolvable commitment rather than as a hung request.
const snapBackLimit = 400

// Calendar is the set of days an operation runs: which weekdays it is open, less any dated closures.
//
// One type covers all three parties to a shipment because they differ only in their days, never in how the days are counted — the plant tenders freight Monday to Thursday, the carrier moves it Monday to Friday, the customer's dock accepts it on its own days, and each of them shuts for its own holidays.
//
// The zero value is closed every day and is not usable. Build one with NewCalendar or DefaultCalendar.
type Calendar struct {
	// OpenDays is indexed by ISO weekday, Monday at 0 through Sunday at 6.
	OpenDays [dayMaskLength]bool
	// Closures are dated shutdowns — holidays, shutdown weeks — keyed on the UTC-truncated date.
	Closures map[time.Time]struct{}
}

// DefaultCalendar is Monday to Friday with nothing closed.
//
// This is what the system did before calendars existed, and it is deliberately the fallback for an account that has configured nothing: adding the feature must not move a single date until somebody says which days they actually work.
func DefaultCalendar() Calendar {
	return Calendar{OpenDays: [dayMaskLength]bool{true, true, true, true, true, false, false}}
}

// NewCalendar builds a calendar from a stored day mask and closure dates.
//
// A mask with no open day is rejected rather than stored: it describes an operation that never runs, and every date resolved against it would walk the full snap-back limit before failing. Catching it here means the failure names the calendar instead of surfacing later as an order that cannot be committed.
func NewCalendar(mask string, closures []time.Time) (Calendar, error) {
	openDays, err := ParseDayMask(mask)
	if err != nil {
		return Calendar{}, err
	}

	cal := Calendar{OpenDays: openDays}
	if len(closures) > 0 {
		cal.Closures = make(map[time.Time]struct{}, len(closures))
		for _, c := range closures {
			cal.Closures[dateOnlyUTC(c)] = struct{}{}
		}
	}
	return cal, nil
}

// ParseDayMask reads a seven-character days-of-week mask, Monday first, where '1' is an open day.
func ParseDayMask(mask string) ([dayMaskLength]bool, error) {
	var out [dayMaskLength]bool

	trimmed := strings.TrimSpace(mask)
	if len(trimmed) != dayMaskLength {
		return out, fmt.Errorf("days-of-week mask %q must be %d characters", mask, dayMaskLength)
	}

	any := false
	for i, c := range trimmed {
		switch c {
		case '1':
			out[i] = true
			any = true
		case '0':
		default:
			return out, fmt.Errorf("days-of-week mask %q contains %q, expected only '0' or '1'", mask, c)
		}
	}
	if !any {
		return out, fmt.Errorf("days-of-week mask %q closes every day", mask)
	}
	return out, nil
}

// IsOpen reports whether the operation runs on a date.
func (c Calendar) IsOpen(d time.Time) bool {
	if !c.OpenDays[isoWeekdayIndex(d)] {
		return false
	}
	_, closed := c.Closures[dateOnlyUTC(d)]
	return !closed
}

// SnapBack moves a date to the nearest open day at or before it, and reports how many days that took.
//
// Always backward, never forward. A date the operation cannot act on has to resolve to one it can, and resolving forward would push a shipment past the day it was promised for — the whole failure this exists to prevent. Snapping back can only make an order leave early, which is safe.
//
// ok is false when no open day exists within the snap-back limit, which means the calendar is closed indefinitely. Callers treat that as an unresolvable commitment rather than substituting a date.
func (c Calendar) SnapBack(d time.Time) (snapped time.Time, moved int, ok bool) {
	day := dateOnlyUTC(d)
	for walked := range snapBackLimit {
		if c.IsOpen(day) {
			return day, walked, true
		}
		day = day.AddDate(0, 0, -1)
	}
	return time.Time{}, 0, false
}

func (c Calendar) SnapForward(d time.Time) (snapped time.Time, moved int, ok bool) {
	day := dateOnlyUTC(d)
	for walked := range snapBackLimit {
		if c.IsOpen(day) {
			return day, walked, true
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}, 0, false
}

func (c Calendar) AddDays(from time.Time, n int) (time.Time, bool) {
	day := dateOnlyUTC(from)
	for range n {
		day = day.AddDate(0, 0, 1)
		snappedDay, _, ok := c.SnapForward(day)
		if !ok {
			return time.Time{}, false
		}
		day = snappedDay
	}
	return day, true
}

// SubtractDays walks back n open days from a date.
//
// Carriers quote transit in the days they actually move freight: a service quoted at three days and handed over on a Thursday delivers the following Tuesday, not Sunday. Counting calendar days instead would put every ship-by date up to two days late in exactly the cases that matter, since a week has more weekday-crossing lanes than not — and the same holds for a holiday the carrier's network is down.
//
// n <= 0 returns the starting date, so a lane with no transit means ship-by is the delivery date. The starting date is not itself snapped: callers that need it on an open day snap it first, and conflating the two would hide which rule moved the date.
func (c Calendar) SubtractDays(from time.Time, n int) (time.Time, bool) {
	day := dateOnlyUTC(from)
	for range n {
		day = day.AddDate(0, 0, -1)
		snappedDay, _, ok := c.SnapBack(day)
		if !ok {
			return time.Time{}, false
		}
		day = snappedDay
	}
	return day, true
}

// SubtractBusinessDays walks back n weekdays from a date, skipping weekends and nothing else.
//
// Retained as the un-configured case of Calendar.SubtractDays: it is what every account got before calendars existed, and keeping it named separately keeps that baseline behaviour pinned by its own tests.
func SubtractBusinessDays(from time.Time, n int) time.Time {
	// DefaultCalendar has open weekdays and no closures, so the walk can never exhaust the limit.
	out, _ := DefaultCalendar().SubtractDays(from, n)
	return out
}

// isoWeekdayIndex maps a date onto a day mask, Monday at 0 through Sunday at 6. Go counts weekdays from Sunday, the mask counts from Monday.
func isoWeekdayIndex(t time.Time) int {
	return (int(t.Weekday()) + 6) % dayMaskLength
}

func isWeekend(t time.Time) bool {
	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return true
	default:
		return false
	}
}
