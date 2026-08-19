package scheduling

import (
	"testing"
	"time"
)

func holidayByName(t *testing.T, year int, name string) time.Time {
	t.Helper()
	for _, h := range USFederalHolidays(year) {
		if h.Name == name {
			return h.Date
		}
	}
	t.Fatalf("no holiday named %q in %d", name, year)
	return time.Time{}
}

// The floating holidays are why this is computed rather than listed: a hardcoded table would be wrong from the year after it was written.
func TestUSFederalHolidays_FloatingDatesLandOnTheRightWeekday(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		year    int
		want    time.Time
		weekday time.Weekday
	}{
		{"Martin Luther King Jr. Day", 2026, onDay(2026, time.January, 19), time.Monday},
		{"Presidents' Day", 2026, onDay(2026, time.February, 16), time.Monday},
		{"Memorial Day", 2026, onDay(2026, time.May, 25), time.Monday},
		{"Labor Day", 2026, onDay(2026, time.September, 7), time.Monday},
		{"Columbus Day", 2026, onDay(2026, time.October, 12), time.Monday},
		{"Thanksgiving Day", 2026, onDay(2026, time.November, 26), time.Thursday},
		{"Thanksgiving Day", 2027, onDay(2027, time.November, 25), time.Thursday},
		{"Memorial Day", 2027, onDay(2027, time.May, 31), time.Monday},
	}

	for _, c := range cases {
		got := holidayByName(t, c.year, c.name)
		if !got.Equal(c.want) {
			t.Errorf("%s %d = %s, want %s", c.name, c.year, got.Format(time.DateOnly), c.want.Format(time.DateOnly))
		}
		if got.Weekday() != c.weekday {
			t.Errorf("%s %d fell on a %s, want %s", c.name, c.year, got.Weekday(), c.weekday)
		}
	}
}

// Memorial Day is the last Monday in May, not the fourth: in a year where May has five Mondays those are different dates.
func TestUSFederalHolidays_MemorialDayIsTheLastMondayNotTheFourth(t *testing.T) {
	t.Parallel()

	// May 2027 has Mondays on the 3rd, 10th, 17th, 24th and 31st.
	got := holidayByName(t, 2027, "Memorial Day")

	if want := onDay(2027, time.May, 31); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

// The observance rule is what carriers and banks actually follow: a Saturday holiday closes the Friday before, a Sunday one the Monday after.
func TestUSFederalHolidays_FixedDatesShiftOffTheWeekend(t *testing.T) {
	t.Parallel()

	// 2026-07-04 is a Saturday, so Independence Day is observed on Friday the 3rd.
	if got, want := holidayByName(t, 2026, "Independence Day"), onDay(2026, time.July, 3); !got.Equal(want) {
		t.Errorf("Independence Day 2026 = %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	// 2027-07-04 is a Sunday, so it moves to Monday the 5th.
	if got, want := holidayByName(t, 2027, "Independence Day"), onDay(2027, time.July, 5); !got.Equal(want) {
		t.Errorf("Independence Day 2027 = %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	// 2026-12-25 is a Friday and needs no shift.
	if got, want := holidayByName(t, 2026, "Christmas Day"), onDay(2026, time.December, 25); !got.Equal(want) {
		t.Errorf("Christmas 2026 = %s, want %s", got.Format(time.DateOnly), want.Format(time.DateOnly))
	}
}

func TestUSFederalHolidays_NeverLandOnAWeekend(t *testing.T) {
	t.Parallel()

	// Every holiday in the seed is a day somebody is closed, and nobody observes a closure on a day they were already shut.
	for year := 2026; year <= 2040; year++ {
		for _, h := range USFederalHolidays(year) {
			if isWeekend(h.Date) {
				t.Errorf("%s %d fell on a %s", h.Name, year, h.Date.Weekday())
			}
		}
	}
}

func TestUSFederalHolidays_ElevenPerYearAllInThatYear(t *testing.T) {
	t.Parallel()

	for year := 2026; year <= 2040; year++ {
		got := USFederalHolidays(year)
		if len(got) != 11 {
			t.Fatalf("%d returned %d holidays, want 11", year, len(got))
		}
		seen := map[time.Time]string{}
		for _, h := range got {
			// A New Year's Day observed on January 1st of a Saturday year shifts to the previous December, which would file a closure against the wrong year's seed.
			if h.Date.Year() != year && h.Name != "New Year's Day" {
				t.Errorf("%s %d resolved into %d", h.Name, year, h.Date.Year())
			}
			if other, dup := seen[h.Date]; dup {
				t.Errorf("%s and %s both resolved to %s", other, h.Name, h.Date.Format(time.DateOnly))
			}
			seen[h.Date] = h.Name
		}
	}
}
