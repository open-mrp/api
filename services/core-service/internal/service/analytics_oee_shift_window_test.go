package service

import (
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// carolonSettings is the shape that made this necessary: two eight-hour shifts, Monday to Friday,
// starting at 06:00 Eastern. The plant is open 06:00-22:00 local and shut the other eight hours.
func carolonSettings() *domain.ProductionScheduleSettings {
	start := "06:00"
	zone := "America/New_York"
	return &domain.ProductionScheduleSettings{
		ShiftsPerDay:    2,
		HoursPerShift:   8,
		ShiftStartTime:  &start,
		ShiftTimezone:   &zone,
		ShiftDaysOfWeek: "1111100",
	}
}

// eastern parses a local wall-clock time in the plant's zone, which is how a shift is actually specified.
func eastern(t *testing.T, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	require.NoError(t, err)
	return parsed
}

// An account that has never configured a window must behave exactly as it did before the window
// existed: every logged second counts. Anything else silently re-scales the OEE of every account
// that never told us when its day begins.
func TestOeeShiftWindow_UnconfiguredCountsEverything(t *testing.T) {
	t.Parallel()

	settings := carolonSettings()
	settings.ShiftStartTime = nil

	window := newOeeShiftWindow(settings)
	assert.Nil(t, window, "a settings row with no start time configures no window")

	start := eastern(t, "2026-09-10 15:00")
	assert.InDelta(t, 16*3600.0, window.OverlapSeconds(start, start.Add(16*time.Hour)), 0.001,
		"a nil window charges the whole span")
}

// The case from production: a stop logged 15:00 Thursday to 07:00 Friday spans sixteen wall-clock
// hours, but the plant was only open for eight of them — seven of Thursday's shift and one of
// Friday's. Charging all sixteen is what deflated run time and pushed Performance over 100%.
func TestOeeShiftWindow_ClipsOvernightStop(t *testing.T) {
	t.Parallel()

	window := newOeeShiftWindow(carolonSettings())
	require.NotNil(t, window)

	got := window.OverlapSeconds(eastern(t, "2026-09-10 15:00"), eastern(t, "2026-09-11 07:00"))
	assert.InDelta(t, 8*3600.0, got, 0.001, "15:00-22:00 Thursday plus 06:00-07:00 Friday is eight hours")
}

// A shift belongs to the day it starts, and a closed day takes none of it. A Friday-evening stop
// running into Saturday stops counting at Friday's 22:00.
func TestOeeShiftWindow_ClosedDayTakesNothing(t *testing.T) {
	t.Parallel()

	window := newOeeShiftWindow(carolonSettings())
	require.NotNil(t, window)

	got := window.OverlapSeconds(eastern(t, "2026-09-11 19:00"), eastern(t, "2026-09-12 11:00"))
	assert.InDelta(t, 3*3600.0, got, 0.001, "only 19:00-22:00 Friday counts; Saturday is closed")

	weekend := window.OverlapSeconds(eastern(t, "2026-09-12 08:00"), eastern(t, "2026-09-12 20:00"))
	assert.Zero(t, weekend, "a stop wholly on a closed day costs no capacity")
}

// A stop entirely inside the shift is charged in full — the clip must not shave time the plant was
// genuinely open for.
func TestOeeShiftWindow_InsideShiftCountsInFull(t *testing.T) {
	t.Parallel()

	window := newOeeShiftWindow(carolonSettings())
	require.NotNil(t, window)

	got := window.OverlapSeconds(eastern(t, "2026-09-10 09:00"), eastern(t, "2026-09-10 12:30"))
	assert.InDelta(t, 3.5*3600.0, got, 0.001)
}

// A multi-day stop accumulates one shift per open day it covers, and skips the weekend in between.
func TestOeeShiftWindow_AccumulatesAcrossDays(t *testing.T) {
	t.Parallel()

	window := newOeeShiftWindow(carolonSettings())
	require.NotNil(t, window)

	// Thursday 06:00 through the following Tuesday 22:00: Thu, Fri, Mon, Tue are open, Sat and Sun are not.
	got := window.OverlapSeconds(eastern(t, "2026-09-10 06:00"), eastern(t, "2026-09-15 22:00"))
	assert.InDelta(t, 4*16*3600.0, got, 0.001, "four open days at sixteen hours each")
}

// The shift starts at 06:00 local on the day the clocks go back, not 05:00 or 07:00: the start is a
// wall-clock anchor. The day itself is 25 hours long, but the plant still works two eight-hour shifts.
func TestOeeShiftWindow_DaylightSavingKeepsWallClockStart(t *testing.T) {
	t.Parallel()

	window := newOeeShiftWindow(carolonSettings())
	require.NotNil(t, window)

	// 2026-11-01 is the US fall-back Sunday, so Monday the 2nd is the first open day on standard time.
	got := window.OverlapSeconds(eastern(t, "2026-11-02 06:00"), eastern(t, "2026-11-02 22:00"))
	assert.InDelta(t, 16*3600.0, got, 0.001, "a normal open day is sixteen hours whatever the clocks did")
}

// A plant running more shift-hours than a day holds is running continuously; the window must not
// overlap itself into more than twenty-four hours of capacity in one day.
func TestOeeShiftWindow_ContinuousPlantCapsAtTheDay(t *testing.T) {
	t.Parallel()

	settings := carolonSettings()
	settings.ShiftsPerDay = 4
	settings.HoursPerShift = 8

	window := newOeeShiftWindow(settings)
	require.NotNil(t, window)

	got := window.OverlapSeconds(eastern(t, "2026-09-10 06:00"), eastern(t, "2026-09-11 06:00"))
	assert.InDelta(t, 24*3600.0, got, 0.001, "thirty-two shift-hours in a day is round-the-clock, not thirty-two hours")
}

// A zone the database holds but the runtime cannot resolve is a misconfiguration. Falling back to no
// window counts every second, which is wrong-but-unchanged; inventing a window would be wrong-and-new.
func TestOeeShiftWindow_UnresolvableZoneDisablesClipping(t *testing.T) {
	t.Parallel()

	settings := carolonSettings()
	zone := "Mars/Olympus_Mons"
	settings.ShiftTimezone = &zone

	assert.Nil(t, newOeeShiftWindow(settings), "an unresolvable zone configures no window")
}
