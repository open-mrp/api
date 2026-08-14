package apiresource

import "testing"

// A quoted price has to equal the price the order will charge, so the value is carried through untouched however many places it has.
func TestNewComputedRate_PreservesTheCallersPrecision(t *testing.T) {
	unit := &Unit{ID: "un_pair", Abbreviation: "pr"}
	dollars := &Unit{ID: "un_usd", Abbreviation: "$"}

	rate := NewComputedRate("18.456789", dollars, unit)
	if rate.Value != "18.456789" {
		t.Errorf("value = %q, want it unrounded", rate.Value)
	}
	if rate.DisplayValue != "$18.46 / pr" {
		t.Errorf("display value = %q, want %q", rate.DisplayValue, "$18.46 / pr")
	}
	if rate.NumeratorUnit == nil || rate.DenominatorUnit == nil {
		t.Error("eagerly resolved units must be attached")
	}
}

// The quote endpoints resolve units from a map, so a lookup miss yields nil rather than a panic, and the display simply drops that half.
func TestNewComputedRate_ToleratesMissingUnits(t *testing.T) {
	rate := NewComputedRate("5", nil, nil)
	if rate.DisplayValue != "5.00" {
		t.Errorf("display value = %q, want %q", rate.DisplayValue, "5.00")
	}
}

func TestFormatRateDisplay_UnparseableValueReadsAsZero(t *testing.T) {
	if got := FormatRateDisplay("not-a-number", "$", "pr"); got != "$0.00 / pr" {
		t.Errorf("display = %q, want %q", got, "$0.00 / pr")
	}
}
