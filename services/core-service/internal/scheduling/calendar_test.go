package scheduling

import (
	"testing"
	"time"
)

func onDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func mustCalendar(t *testing.T, mask string, closures ...time.Time) Calendar {
	t.Helper()
	cal, err := NewCalendar(mask, closures)
	if err != nil {
		t.Fatalf("NewCalendar(%q): %v", mask, err)
	}
	return cal
}

func TestParseDayMask_RejectsACalendarThatNeverRuns(t *testing.T) {
	t.Parallel()

	// An operation with no open day would send every date resolved against it to the snap-back limit. Naming the mask here beats surfacing later as an order that cannot be committed.
	if _, err := ParseDayMask("0000000"); err == nil {
		t.Fatal("expected an all-closed mask to be rejected")
	}
}

func TestParseDayMask_RejectsMalformedInput(t *testing.T) {
	t.Parallel()

	for _, mask := range []string{"", "111110", "11111000", "111110x", "1111 00"} {
		if _, err := ParseDayMask(mask); err == nil {
			t.Fatalf("expected %q to be rejected", mask)
		}
	}
}

func TestParseDayMask_ReadsMondayFirst(t *testing.T) {
	t.Parallel()

	// "1111000" is a Monday-to-Thursday plant, the case that motivated the feature.
	got, err := ParseDayMask("1111000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := [7]bool{true, true, true, true, false, false, false}; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCalendar_IsOpenHonoursBothTheMaskAndTheClosures(t *testing.T) {
	t.Parallel()

	thanksgiving := onDay(2026, time.November, 26)
	cal := mustCalendar(t, "1111100", thanksgiving)

	if cal.IsOpen(thanksgiving) {
		t.Fatal("a closure date must read as closed even though Thursday is an open weekday")
	}
	if !cal.IsOpen(onDay(2026, time.November, 25)) {
		t.Fatal("the Wednesday before is open")
	}
	if cal.IsOpen(onDay(2026, time.November, 28)) {
		t.Fatal("Saturday is closed by the mask")
	}
}

func TestCalendar_SnapBackMovesOffAClosedWeekday(t *testing.T) {
	t.Parallel()

	// A Monday-to-Thursday plant cannot tender freight on Friday the 21st, so the deadline is Thursday the 20th.
	cal := mustCalendar(t, "1111000")

	got, moved, ok := cal.SnapBack(onDay(2026, time.August, 21))
	if !ok {
		t.Fatal("expected a snap")
	}
	if want := onDay(2026, time.August, 20); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	if moved != 1 {
		t.Fatalf("got %d days moved, want 1", moved)
	}
}

func TestCalendar_SnapBackWalksPastAShutdownWeek(t *testing.T) {
	t.Parallel()

	// The whole week of Christmas 2026 is shut, so a date inside it resolves back to Thursday the 24th.
	shutdown := []time.Time{
		onDay(2026, time.December, 28), onDay(2026, time.December, 29), onDay(2026, time.December, 30), onDay(2026, time.December, 31),
		onDay(2026, time.December, 25),
	}
	cal := mustCalendar(t, "1111100", shutdown...)

	got, moved, ok := cal.SnapBack(onDay(2026, time.December, 31))
	if !ok {
		t.Fatal("expected a snap")
	}
	if want := onDay(2026, time.December, 24); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	if moved != 7 {
		t.Fatalf("got %d days moved, want 7", moved)
	}
}

func TestCalendar_SnapBackLeavesAnOpenDayAlone(t *testing.T) {
	t.Parallel()

	cal := DefaultCalendar()
	wednesday := onDay(2026, time.August, 19)

	got, moved, ok := cal.SnapBack(wednesday)
	if !ok || !got.Equal(wednesday) || moved != 0 {
		t.Fatalf("got (%s, %d, %v), want the date unchanged", got.Format(time.DateOnly), moved, ok)
	}
}

// Snapping must only ever move a date earlier. Resolving forward would push a shipment past the day it was promised for, which is the failure the whole rule exists to prevent.
func TestCalendar_SnapBackNeverMovesForward(t *testing.T) {
	t.Parallel()

	cal := mustCalendar(t, "1111000", onDay(2026, time.August, 19))

	start := onDay(2026, time.August, 1)
	for i := range 90 {
		from := start.AddDate(0, 0, i)
		got, _, ok := cal.SnapBack(from)
		if !ok {
			t.Fatalf("no open day at or before %s", from.Format(time.DateOnly))
		}
		if got.After(from) {
			t.Fatalf("snapping %s moved forward to %s", from.Format(time.DateOnly), got.Format(time.DateOnly))
		}
	}
}

func TestCalendar_SnapBackFailsWhenNothingIsEverOpen(t *testing.T) {
	t.Parallel()

	// Reachable only through closures, since the mask itself cannot be all-closed. An indefinitely shut calendar must surface as an unresolvable commitment rather than a hung walk.
	closures := make([]time.Time, 0, snapBackLimit)
	start := onDay(2026, time.August, 20)
	for i := range snapBackLimit {
		closures = append(closures, start.AddDate(0, 0, -i))
	}
	cal := mustCalendar(t, "1111111", closures...)

	if _, _, ok := cal.SnapBack(start); ok {
		t.Fatal("expected the walk to give up rather than return a closed day")
	}
}

func TestCalendar_SubtractDaysSkipsClosuresAsWellAsWeekends(t *testing.T) {
	t.Parallel()

	// Three carrier days back from Monday 2026-11-30, with Thanksgiving on the 26th shut: the 27th, 25th and 24th are the three moving days, so freight has to leave on the 24th.
	cal := mustCalendar(t, "1111100", onDay(2026, time.November, 26))

	got, ok := cal.SubtractDays(onDay(2026, time.November, 30), 3)
	if !ok {
		t.Fatal("expected a result")
	}
	if want := onDay(2026, time.November, 24); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

func TestCalendar_SubtractDaysLeavesTheStartingDateUnsnapped(t *testing.T) {
	t.Parallel()

	// Zero transit returns the date as given, even a closed one. Callers snap the start themselves so the explanation can say which rule moved the date.
	cal := DefaultCalendar()
	saturday := onDay(2026, time.August, 22)

	got, ok := cal.SubtractDays(saturday, 0)
	if !ok || !got.Equal(saturday) {
		t.Fatalf("got (%s, %v), want the date unchanged", got.Format(time.DateOnly), ok)
	}
}

func TestDefaultCalendar_MatchesTheWeekendOnlyRuleItReplaced(t *testing.T) {
	t.Parallel()

	// The pre-calendar behaviour, pinned: an account that configures nothing must not see a single date move.
	cal := DefaultCalendar()
	start := onDay(2026, time.September, 7)

	for offset := range 30 {
		from := start.AddDate(0, 0, offset)
		for n := range 15 {
			got, ok := cal.SubtractDays(from, n)
			if !ok {
				t.Fatalf("no result for %s less %d", from.Format(time.DateOnly), n)
			}
			if want := SubtractBusinessDays(from, n); !got.Equal(want) {
				t.Fatalf("from %s less %d: got %s, want %s", from.Format(time.DateOnly), n, got.Format(time.DateOnly), want.Format(time.DateOnly))
			}
		}
	}
}
