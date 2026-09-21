package service

import (
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// oeeShiftWindow places an account's shift capacity on the clock, so a downtime event can be charged only the part of it the plant was actually open.
//
// OEE capacity is shifts_per_day x hours_per_shift x work_days_per_week — how MUCH time a machine has, never WHEN. Downtime is stored as UTC instants. Without a calendar to intersect them, a stop was charged its whole wall-clock span, so a breakdown logged 15:00 Thursday to 07:00 Friday took 16 hours out of a day that only holds 16, though just 8 of them were inside the shift. Run time is the Performance denominator, so over-charging it is one of the two things that pushed P above 100%.
//
// A shift belongs to the local day it STARTS. A plant running 18:00 to 02:00 works Friday night on Friday's capacity, and a Saturday that the day mask closes takes none of it.
type oeeShiftWindow struct {
	loc *time.Location
	// startHour and startMinute are the local wall-clock time the first shift begins. Anchored to the wall clock rather than added as a duration so a shift still starts at 06:00 on the day the clocks change.
	startHour, startMinute int
	// length is shifts_per_day x hours_per_shift, capped at 24h: a plant running more shift-hours than a day holds is running continuously, not overlapping itself.
	length time.Duration
	// openDays is Monday-first, matching operating_calendar.days_of_week.
	openDays [7]bool
}

// maxShiftDay caps a day's shift length; beyond this the plant is running around the clock.
const maxShiftDay = 24 * time.Hour

// newOeeShiftWindow reads the window off the account's settings, or returns nil when it has never configured one.
//
// Nil means no clipping — every logged second counts, which is the behaviour before this existed. That is deliberate: defaulting an unset window to some assumed start hour would silently re-scale the OEE of every account that never told us when its day begins, and a wrong window is worse than none.
func newOeeShiftWindow(settings *domain.ProductionScheduleSettings) *oeeShiftWindow {
	if settings == nil || settings.ShiftStartTime == nil || settings.ShiftTimezone == nil {
		return nil
	}
	loc, err := time.LoadLocation(*settings.ShiftTimezone)
	if err != nil {
		// An unresolvable zone is a misconfiguration, not a reason to invent a window: fall back to counting every second, as an unconfigured account does.
		return nil
	}
	start, err := time.Parse("15:04", *settings.ShiftStartTime)
	if err != nil {
		// Stored as "HH:MM"; tolerate a seconds-bearing value rather than dropping the window.
		start, err = time.Parse("15:04:05", *settings.ShiftStartTime)
		if err != nil {
			return nil
		}
	}

	length := time.Duration(float64(settings.ShiftsPerDay) * settings.HoursPerShift * float64(time.Hour))
	if length <= 0 {
		return nil
	}
	if length > maxShiftDay {
		length = maxShiftDay
	}

	w := &oeeShiftWindow{loc: loc, startHour: start.Hour(), startMinute: start.Minute(), length: length}
	for i := range w.openDays {
		// A mask shorter than seven characters leaves the rest closed rather than defaulting them open.
		w.openDays[i] = i < len(settings.ShiftDaysOfWeek) && settings.ShiftDaysOfWeek[i] == '1'
	}
	return w
}

// OverlapSeconds is how much of [start, end) falls inside open shift time.
//
// A nil window counts the whole span, so callers need no branch of their own.
func (w *oeeShiftWindow) OverlapSeconds(start, end time.Time) float64 {
	if !end.After(start) {
		return 0
	}
	if w == nil {
		return end.Sub(start).Seconds()
	}

	localStart := start.In(w.loc)
	localEnd := end.In(w.loc)
	// Begin a day early: a shift that opened the previous evening can still be running when the span starts.
	day := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, w.loc).AddDate(0, 0, -1)

	var total float64
	for !day.After(localEnd) {
		if w.openDays[(int(day.Weekday())+6)%7] {
			shiftStart := time.Date(day.Year(), day.Month(), day.Day(), w.startHour, w.startMinute, 0, 0, w.loc)
			total += overlapSeconds(start, end, shiftStart, shiftStart.Add(w.length))
		}
		day = day.AddDate(0, 0, 1)
	}
	return total
}

// overlapSeconds is the length of the intersection of two half-open intervals, in seconds, or zero when they do not meet.
func overlapSeconds(aStart, aEnd, bStart, bEnd time.Time) float64 {
	from := aStart
	if bStart.After(from) {
		from = bStart
	}
	to := aEnd
	if bEnd.Before(to) {
		to = bEnd
	}
	if !to.After(from) {
		return 0
	}
	return to.Sub(from).Seconds()
}
