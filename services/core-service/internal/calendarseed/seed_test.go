package calendarseed

import (
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/constants"
)

// Both seeded calendars must reproduce the pre-calendar behaviour exactly. A newly seeded account should see its holidays respected and nothing else change.
func TestSeedDefaults_AreMondayToFriday(t *testing.T) {
	t.Parallel()

	for _, seed := range []struct {
		name string
		days string
		kind string
	}{
		{"ship", defaultShipCalendarSeed.DaysOfWeek, defaultShipCalendarSeed.KindCode},
		{"receive", defaultReceiveCalendarSeed.DaysOfWeek, defaultReceiveCalendarSeed.KindCode},
	} {
		cal, err := scheduling.NewCalendar(seed.days, nil)
		if err != nil {
			t.Fatalf("%s seed mask is invalid: %v", seed.name, err)
		}
		if cal.OpenDays != scheduling.DefaultCalendar().OpenDays {
			t.Errorf("%s seed opens %v, want the Monday-to-Friday default", seed.name, cal.OpenDays)
		}
	}

	if defaultShipCalendarSeed.KindCode != string(constants.OperatingCalendarKindShip) {
		t.Error("the ship seed is not a ship calendar")
	}
	if defaultReceiveCalendarSeed.KindCode != string(constants.OperatingCalendarKindReceive) {
		t.Error("the receive seed is not a receive calendar")
	}
	// Exactly one default per kind is what the resolution chain falls back to.
	if !defaultShipCalendarSeed.IsDefault || !defaultReceiveCalendarSeed.IsDefault {
		t.Error("both seeds must be their kind's default, or resolution has nothing to fall back to")
	}
}

// The cutoff has to parse, or every commitment silently loses its time of day.
func TestDefaultPickupCutoff_Parses(t *testing.T) {
	t.Parallel()

	parsed, err := time.Parse("15:04", defaultPickupCutoff)
	if err != nil {
		t.Fatalf("default cutoff %q does not parse: %v", defaultPickupCutoff, err)
	}
	if parsed.Hour() != 15 || parsed.Minute() != 0 {
		t.Fatalf("default cutoff is %02d:%02d, want 15:00", parsed.Hour(), parsed.Minute())
	}
}

// A seed has to cover the span an order can actually be promised across, or a long lead time resolves against a year with no holidays in it.
func TestClosureSeedHorizon_CoversTheYearsAnOrderCanReach(t *testing.T) {
	t.Parallel()

	asOf := time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC)

	var latest time.Time
	for offset := range closureSeedYears {
		for _, h := range scheduling.USFederalHolidays(asOf.Year() + offset) {
			if h.Date.After(latest) {
				latest = h.Date
			}
		}
	}

	// Seeding late in a year must still leave more than a year of runway ahead.
	if minimum := asOf.AddDate(1, 0, 0); latest.Before(minimum) {
		t.Fatalf("seeding on %s only reaches %s, want at least %s", asOf.Format(time.DateOnly), latest.Format(time.DateOnly), minimum.Format(time.DateOnly))
	}
}
