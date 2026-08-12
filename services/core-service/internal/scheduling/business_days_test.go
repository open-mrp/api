package scheduling

import (
	"testing"
	"time"
)

func TestSubtractBusinessDays_SkipsTheWeekend(t *testing.T) {
	t.Parallel()

	// Monday 2026-09-07 less three business days is Wednesday the 2nd: the 5th and 6th are a weekend and do not count.
	monday := time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC)

	got := SubtractBusinessDays(monday, 3)

	if want := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

// The calendar-day answer here would be Friday the 4th, two days late. This is the error the whole business-day rule exists to remove.
func TestSubtractBusinessDays_BeatsCalendarArithmeticAcrossAWeekend(t *testing.T) {
	t.Parallel()

	monday := time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC)

	if got := SubtractBusinessDays(monday, 3); got.Equal(monday.AddDate(0, 0, -3)) {
		t.Fatal("business-day subtraction matched calendar subtraction across a weekend")
	}
}

func TestSubtractBusinessDays_ZeroIsTheSameDay(t *testing.T) {
	t.Parallel()

	// A lane with no transit means ship-by is the delivery date itself.
	day := time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC)

	if got := SubtractBusinessDays(day, 0); !got.Equal(day) {
		t.Fatalf("got %s, want the date unchanged", got.Format(time.DateOnly))
	}
}

func TestSubtractBusinessDays_NeverLandsOnAWeekend(t *testing.T) {
	t.Parallel()

	// Every start date across a full week, every transit up to a fortnight: any positive transit must resolve to a weekday, since a ship-by nobody can ship on is not a deadline.
	start := time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC)
	for day := range 7 {
		from := start.AddDate(0, 0, day)
		for n := 1; n <= 14; n++ {
			if got := SubtractBusinessDays(from, n); isWeekend(got) {
				t.Fatalf("from %s less %d business days = %s, a %s", from.Format(time.DateOnly), n, got.Format(time.DateOnly), got.Weekday())
			}
		}
	}
}

// Ten business days is exactly two calendar weeks, which pins the loop against off-by-one drift.
func TestSubtractBusinessDays_TenBusinessDaysIsAFortnight(t *testing.T) {
	t.Parallel()

	monday := time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC)

	got := SubtractBusinessDays(monday, 10)

	if want := monday.AddDate(0, 0, -14); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}
