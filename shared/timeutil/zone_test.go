package timeutil

import (
	"testing"
	"time"
)

func TestLookupZone_USSubdivisions(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"NY": "America/New_York",
		"TX": "America/Chicago",
		"CO": "America/Denver",
		"CA": "America/Los_Angeles",
		"AK": "America/Anchorage",
		"HI": "Pacific/Honolulu",
		// Arizona keeps no daylight saving, so it cannot share Denver's zone.
		"AZ": "America/Phoenix",
		"PR": "America/Puerto_Rico",
	}

	for state, want := range cases {
		got, ok := LookupZone("US", state)
		if !ok || got != want {
			t.Errorf("US/%s = (%q, %v), want %q", state, got, ok, want)
		}
	}
}

// The state column is free text, so the same place arrives spelled out as often as abbreviated, in any case.
func TestLookupZone_AcceptsSpelledOutAndMixedCaseSubdivisions(t *testing.T) {
	t.Parallel()

	for _, state := range []string{"CA", "ca", "California", "CALIFORNIA", " california "} {
		got, ok := LookupZone("us", state)
		if !ok || got != "America/Los_Angeles" {
			t.Errorf("US/%q = (%q, %v), want America/Los_Angeles", state, got, ok)
		}
	}
}

func TestLookupZone_CanadaAndAustralia(t *testing.T) {
	t.Parallel()

	cases := []struct{ country, state, want string }{
		{"CA", "ON", "America/Toronto"},
		{"CA", "BC", "America/Vancouver"},
		{"CA", "Newfoundland and Labrador", "America/St_Johns"},
		// Saskatchewan keeps no daylight saving.
		{"CA", "SK", "America/Regina"},
		{"AU", "QLD", "Australia/Brisbane"},
		{"AU", "Western Australia", "Australia/Perth"},
	}

	for _, c := range cases {
		got, ok := LookupZone(c.country, c.state)
		if !ok || got != c.want {
			t.Errorf("%s/%s = (%q, %v), want %q", c.country, c.state, got, ok, c.want)
		}
	}
}

// Most countries keep one zone, which is why the bulk of the table needs no subdivision at all.
func TestLookupZone_SingleZoneCountriesNeedNoState(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"GB": "Europe/London",
		"DE": "Europe/Berlin",
		"JP": "Asia/Tokyo",
		"IN": "Asia/Kolkata",
		"SG": "Asia/Singapore",
		"ZA": "Africa/Johannesburg",
	}

	for country, want := range cases {
		got, ok := LookupZone(country, "")
		if !ok || got != want {
			t.Errorf("%s = (%q, %v), want %q", country, got, ok, want)
		}
	}
}

// An unrecognised state must not sink the whole lookup: the country answer is still far better than nothing.
func TestLookupZone_UnknownSubdivisionFallsBackToTheCountry(t *testing.T) {
	t.Parallel()

	got, ok := LookupZone("US", "Westeros")
	if !ok || got != "America/New_York" {
		t.Fatalf("got (%q, %v), want America/New_York", got, ok)
	}
}

func TestLookupZone_UnknownCountryIsNotGuessed(t *testing.T) {
	t.Parallel()

	if got, ok := LookupZone("ZZ", "XX"); ok {
		t.Fatalf("got %q, want no answer for an unknown country", got)
	}
	if _, ok := LookupZone("", ""); ok {
		t.Fatal("want no answer for an empty country")
	}
}

// Every zone in the table has to exist in the embedded database, or an address resolves silently to UTC.
func TestZoneTable_EveryZoneLoads(t *testing.T) {
	t.Parallel()

	seen := map[string]struct{}{}
	for _, zone := range countryZones {
		seen[zone] = struct{}{}
	}
	for _, subdivisions := range subdivisionZones {
		for _, zone := range subdivisions {
			seen[zone] = struct{}{}
		}
	}

	for zone := range seen {
		if _, err := time.LoadLocation(zone); err != nil {
			t.Errorf("zone %q does not load: %v", zone, err)
		}
	}
}

// Every alias has to land on a code the zone table actually holds, or spelling a state out silently loses its zone.
func TestZoneTable_EveryAliasResolves(t *testing.T) {
	t.Parallel()

	for country, aliases := range subdivisionNames {
		for name, code := range aliases {
			if _, ok := subdivisionZones[country][code]; !ok {
				t.Errorf("%s alias %q maps to %q, which has no zone", country, name, code)
			}
		}
	}
}

func TestZoneFor_PrefersTheStoredZone(t *testing.T) {
	t.Parallel()

	// The stored value is the only one a person can correct, so it has to beat the derivation even when the derivation would answer.
	stored := "America/Indiana/Indianapolis"
	got := ZoneFor(&stored, "US", "IN", "America/Chicago")

	if got.String() != stored {
		t.Fatalf("got %q, want %q", got, stored)
	}
}

func TestZoneFor_FallsThroughToTheAccountZoneThenUTC(t *testing.T) {
	t.Parallel()

	// An unrecognised country lands on the account's own zone: its addresses cluster near it, so Denver beats UTC.
	if got := ZoneFor(nil, "ZZ", "", "America/Denver"); got.String() != "America/Denver" {
		t.Fatalf("got %q, want America/Denver", got)
	}
	if got := ZoneFor(nil, "ZZ", "", ""); got != time.UTC {
		t.Fatalf("got %q, want UTC", got)
	}
	// A stored zone that does not load must not strand the address either.
	bad := "Mars/Olympus_Mons"
	if got := ZoneFor(&bad, "US", "CO", ""); got.String() != "America/Denver" {
		t.Fatalf("got %q, want the derived America/Denver", got)
	}
}

// The whole reason the destination zone matters: an instant just past midnight UTC is the previous day in Hawaii, and a receiving calendar tested against the wrong weekday is the bug this prevents.
func TestZoneFor_ShiftsTheCalendarDayAcrossTheDateLine(t *testing.T) {
	t.Parallel()

	instant := time.Date(2026, time.August, 22, 2, 0, 0, 0, time.UTC)

	hawaii := ZoneFor(nil, "US", "HI", "")
	if got := instant.In(hawaii).Day(); got != 21 {
		t.Fatalf("Hawaii day = %d, want 21", got)
	}
	newYork := ZoneFor(nil, "US", "NY", "")
	if got := instant.In(newYork).Day(); got != 21 {
		t.Fatalf("New York day = %d, want 21", got)
	}
}
