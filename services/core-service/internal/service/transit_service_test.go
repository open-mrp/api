package service

import (
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
)

func TestNormalizePostal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		country string
		postal  string
		want    string
	}{
		// The +4 addresses a building, not a route, so keeping it would file one lane under a row per customer site while quoting identical transit for each.
		{"us zip+4 collapses to the base", "US", "43215-1234", "43215"},
		{"us five digit is unchanged", "US", "43215", "43215"},
		{"lowercase country still collapses", "us", "78701-0420", "78701"},
		// Outside the US a hyphen can be part of the code itself, so it is never cut.
		{"non-us keeps its full code", "CA", "K1A-0B1", "K1A-0B1"},
		{"internal spaces are stripped", "GB", "SW1A 1AA", "SW1A1AA"},
		{"case is normalized", "GB", "sw1a1aa", "SW1A1AA"},
		{"surrounding space is stripped", "US", "  43215  ", "43215"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizePostal(tc.country, tc.postal); got != tc.want {
				t.Fatalf("normalizePostal(%q, %q) = %q, want %q", tc.country, tc.postal, got, tc.want)
			}
		})
	}
}

// A warm and a read of the same journey have to produce the same key, or the cache never hits.
func TestBuildTransitLane_WarmAndReadAgreeOnTheKey(t *testing.T) {
	t.Parallel()

	origin := domain.ShippingAddress{Country: "us", Zip: "43215-1234"}
	dest := domain.ShippingAddress{Country: "US", Zip: "78701"}

	asWarmed := buildTransitLane("svcl_1", origin, dest)
	asRead := buildTransitLane("svcl_1", domain.ShippingAddress{Country: "US", Zip: "43215"}, dest)

	if asWarmed != asRead {
		t.Fatalf("same journey produced different lanes:\n warm = %+v\n read = %+v", asWarmed, asRead)
	}
	if !asWarmed.IsComplete() {
		t.Fatal("expected a complete lane")
	}
}

func TestBuildTransitLane_IncompleteWhenAPartIsMissing(t *testing.T) {
	t.Parallel()

	full := domain.ShippingAddress{Country: "US", Zip: "43215"}

	cases := map[string]domain.TransitLane{
		"no service level": buildTransitLane("", full, full),
		"no origin postal": buildTransitLane("svcl_1", domain.ShippingAddress{Country: "US"}, full),
		"no dest country":  buildTransitLane("svcl_1", full, domain.ShippingAddress{Zip: "78701"}),
	}

	for name, lane := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if lane.IsComplete() {
				t.Fatalf("lane %+v reported complete", lane)
			}
		})
	}
}

func TestSelectTransit(t *testing.T) {
	t.Parallel()

	stale := time.Now().Add(-2 * transitEstimateTTL)
	days := func(n int) *int { return &n }

	t.Run("the quoted lane wins over the service level default", func(t *testing.T) {
		t.Parallel()
		got := selectTransit(&domain.CarrierTransitCandidates{LaneDays: days(3), ServiceLevelDefaultDays: days(5)})
		if got == nil || got.Days != 3 || got.Source != string(constants.TransitSourceCarrierLane) {
			t.Fatalf("got %+v, want 3 days from carrier_lane", got)
		}
	})

	// Age governs whether a warm re-quotes, not whether a reader may trust the answer: a months-old measurement of this exact journey still beats one number standing in for every lane the account ships.
	t.Run("a stale lane still beats the default", func(t *testing.T) {
		t.Parallel()
		got := selectTransit(&domain.CarrierTransitCandidates{
			LaneDays: days(3), LaneRefreshedAt: &stale, ServiceLevelDefaultDays: days(5),
		})
		if got == nil || got.Days != 3 || got.Source != string(constants.TransitSourceCarrierLane) {
			t.Fatalf("got %+v, want the stale lane's 3 days", got)
		}
	})

	t.Run("falls back to the service level default", func(t *testing.T) {
		t.Parallel()
		got := selectTransit(&domain.CarrierTransitCandidates{ServiceLevelDefaultDays: days(5)})
		if got == nil || got.Days != 5 || got.Source != string(constants.TransitSourceServiceLevel) {
			t.Fatalf("got %+v, want 5 days from service_level", got)
		}
	})

	// Nothing configured means transit is unknown, which leaves ship-by equal to the promised date rather than guessing at zero.
	t.Run("nothing known resolves to no transit", func(t *testing.T) {
		t.Parallel()
		if got := selectTransit(&domain.CarrierTransitCandidates{}); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
		if got := selectTransit(nil); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
	})

	// Zero is a real lane (will-call, same-day), distinct from an absent one.
	t.Run("a zero-day lane is an answer", func(t *testing.T) {
		t.Parallel()
		got := selectTransit(&domain.CarrierTransitCandidates{LaneDays: days(0), ServiceLevelDefaultDays: days(5)})
		if got == nil || got.Days != 0 || got.Source != string(constants.TransitSourceCarrierLane) {
			t.Fatalf("got %+v, want 0 days from carrier_lane", got)
		}
	})
}
