package service

import (
	"testing"

	"github.com/augno/api/shared/constants"
)

func TestValidateCalendarShape_RejectsAMaskNothingCanShipOn(t *testing.T) {
	t.Parallel()

	ship := string(constants.OperatingCalendarKindShip)

	for _, mask := range []string{"0000000", "111110", "11111000", "1111x00", ""} {
		if apiErr := validateCalendarShape(ship, mask, nil, nil); apiErr == nil {
			t.Errorf("expected mask %q to be rejected", mask)
		}
	}
	// The all-closed case matters most: it would send every commitment resolved against it to the snap-back limit before failing.
	if apiErr := validateCalendarShape(ship, "0000000", nil, nil); apiErr == nil {
		t.Fatal("an all-closed mask must be rejected at the write path")
	}
}

func TestValidateCalendarShape_AcceptsRealMasks(t *testing.T) {
	t.Parallel()

	ship := string(constants.OperatingCalendarKindShip)

	for _, mask := range []string{"1111100", "1111000", "1111110", "0000001"} {
		if apiErr := validateCalendarShape(ship, mask, nil, nil); apiErr != nil {
			t.Errorf("mask %q rejected: %s", mask, apiErr.PublicMessage)
		}
	}
}

// A cutoff is a shipping concept: it is when the plant hands freight over, and a customer's dock has no equivalent.
func TestValidateCalendarShape_CutoffOnlyOnShippingCalendars(t *testing.T) {
	t.Parallel()

	cutoff := "15:00"

	if apiErr := validateCalendarShape(string(constants.OperatingCalendarKindShip), "1111100", &cutoff, nil); apiErr != nil {
		t.Fatalf("a shipping calendar must accept a cutoff: %s", apiErr.PublicMessage)
	}
	if apiErr := validateCalendarShape(string(constants.OperatingCalendarKindReceive), "1111100", &cutoff, nil); apiErr == nil {
		t.Fatal("a receiving calendar must reject a cutoff")
	}
}

func TestValidateCalendarShape_RejectsUnparseableCutoff(t *testing.T) {
	t.Parallel()

	ship := string(constants.OperatingCalendarKindShip)

	for _, cutoff := range []string{"half past three", "25:00", "3pm", "15:00:00"} {
		c := cutoff
		if apiErr := validateCalendarShape(ship, "1111100", &c, nil); apiErr == nil {
			t.Errorf("expected cutoff %q to be rejected", cutoff)
		}
	}

	// Empty is not a value, it is the absence of one, and must pass.
	empty := ""
	if apiErr := validateCalendarShape(ship, "1111100", &empty, nil); apiErr != nil {
		t.Fatalf("an empty cutoff must be treated as unset: %s", apiErr.PublicMessage)
	}
}

// A zone that does not load would silently resolve every date against UTC, which is the failure the stored zone exists to prevent.
func TestValidateCalendarShape_RejectsUnknownTimezone(t *testing.T) {
	t.Parallel()

	ship := string(constants.OperatingCalendarKindShip)

	bad := "Mars/Olympus_Mons"
	if apiErr := validateCalendarShape(ship, "1111100", nil, &bad); apiErr == nil {
		t.Fatal("expected an unknown zone to be rejected")
	}

	for _, zone := range []string{"America/Chicago", "UTC", "Europe/London"} {
		z := zone
		if apiErr := validateCalendarShape(ship, "1111100", nil, &z); apiErr != nil {
			t.Errorf("zone %q rejected: %s", zone, apiErr.PublicMessage)
		}
	}
}

func TestValidateCalendarKind_RejectsUnknownKinds(t *testing.T) {
	t.Parallel()

	unknown := "warehouse"
	if apiErr := validateCalendarKind(&unknown); apiErr == nil {
		t.Fatal("expected an unknown kind to be rejected")
	}
	// An absent filter means both kinds, not an invalid one.
	if apiErr := validateCalendarKind(nil); apiErr != nil {
		t.Fatalf("a nil kind filter must be allowed: %s", apiErr.PublicMessage)
	}
	empty := ""
	if apiErr := validateCalendarKind(&empty); apiErr != nil {
		t.Fatalf("an empty kind filter must be allowed: %s", apiErr.PublicMessage)
	}
	for _, kind := range []string{"ship", "receive"} {
		k := kind
		if apiErr := validateCalendarKind(&k); apiErr != nil {
			t.Errorf("kind %q rejected: %s", kind, apiErr.PublicMessage)
		}
	}
}

func TestValueOr_TreatsBlankAsUnset(t *testing.T) {
	t.Parallel()

	set := "1111000"
	blank := "   "

	if got := valueOr(&set, "1111100"); got != set {
		t.Fatalf("got %q, want the provided value", got)
	}
	if got := valueOr(nil, "1111100"); got != "1111100" {
		t.Fatalf("got %q, want the fallback", got)
	}
	// Whitespace is not a mask; falling back beats validating a blank string the caller never meant to send.
	if got := valueOr(&blank, "1111100"); got != "1111100" {
		t.Fatalf("got %q, want the fallback", got)
	}
}
