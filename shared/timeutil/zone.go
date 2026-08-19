package timeutil

import (
	"strings"
	"time"

	// Zone data is embedded rather than read from the host, because the containers these services run in carry no tzdata and a missing zone would silently degrade every date resolved against it to UTC.
	_ "time/tzdata"
)

// ZoneFor resolves the zone a place keeps time in, most trustworthy source first.
//
// A stored zone wins outright. It is the only source a person can correct, and the derivation below is deliberately approximate — thirteen US states span two zones, and the majority answer is wrong for the minority of addresses in them. Persisting the derived value and letting it be edited is what makes those addresses fixable at all.
//
// The fallback chain ends at the account's own zone rather than UTC, because an account's addresses cluster near it: a Denver plant shipping to an unrecognised state is far better served by Denver time than by UTC.
func ZoneFor(stored *string, country, state, fallback string) *time.Location {
	for _, candidate := range []string{derefZone(stored), zoneNameFor(country, state), fallback} {
		if candidate == "" {
			continue
		}
		if loc, err := time.LoadLocation(candidate); err == nil {
			return loc
		}
	}
	return time.UTC
}

// LookupZone reports the zone a country and subdivision keep time in, and whether one is known.
//
// The subdivision is consulted only for the countries that genuinely span zones. Everywhere else the country answer is exact, which is why most of the table is keyed on country alone.
func LookupZone(country, state string) (string, bool) {
	name := zoneNameFor(country, state)
	return name, name != ""
}

func zoneNameFor(country, state string) string {
	c := strings.ToUpper(strings.TrimSpace(country))
	if c == "" {
		return ""
	}

	if subdivisions, ok := subdivisionZones[c]; ok {
		if zone, ok := subdivisions[normalizeSubdivision(c, state)]; ok {
			return zone
		}
	}
	return countryZones[c]
}

// normalizeSubdivision reduces a stored state to the code the table is keyed on. The column is free text, so the same place arrives as "CA", "ca", or "California" depending on who typed it.
func normalizeSubdivision(country, state string) string {
	s := strings.ToUpper(strings.Join(strings.Fields(state), " "))
	if s == "" {
		return ""
	}
	if code, ok := subdivisionNames[country][s]; ok {
		return code
	}
	return s
}

func derefZone(stored *string) string {
	if stored == nil {
		return ""
	}
	return strings.TrimSpace(*stored)
}
