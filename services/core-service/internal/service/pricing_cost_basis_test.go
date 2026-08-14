package service

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
)

// countUnits is a unit group where a pair is two each and a dozen is twelve each, with "each" as the base.
func countUnits() map[string]*domain.Unit {
	return map[string]*domain.Unit{
		"un_each": {ID: "un_each", UnitDimensionCode: "count", IsBaseUnit: true, RatioNumerator: "1", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"},
		"un_pair": {ID: "un_pair", UnitDimensionCode: "count", RatioNumerator: "2", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"},
		"un_doz":  {ID: "un_doz", UnitDimensionCode: "count", RatioNumerator: "12", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"},
		"un_kg":   {ID: "un_kg", UnitDimensionCode: "mass", IsBaseUnit: true, RatioNumerator: "1", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"},
		"un_degc": {ID: "un_degc", UnitDimensionCode: "temp", RatioNumerator: "1", RatioDenominator: "1", OffsetNumerator: "273", OffsetDenominator: "1"},
	}
}

// A rate's denominator scales inversely to a quantity: costing $3 per each is $6 per pair, not $1.50.
func TestCostOnBasis_ScalesInverselyToTheDenominator(t *testing.T) {
	units := countUnits()

	got, ok := costOnBasis(decimal.RequireFromString("3"), "un_each", "un_pair", units)
	if !ok {
		t.Fatal("a cost per each must be expressible per pair")
	}
	if !got.Equal(decimal.RequireFromString("6")) {
		t.Errorf("cost per pair = %s, want 6", got)
	}

	got, ok = costOnBasis(decimal.RequireFromString("3"), "un_each", "un_doz", units)
	if !ok {
		t.Fatal("a cost per each must be expressible per dozen")
	}
	if !got.Equal(decimal.RequireFromString("36")) {
		t.Errorf("cost per dozen = %s, want 36", got)
	}

	// And back down again: $36 per dozen is $3 per each.
	got, ok = costOnBasis(decimal.RequireFromString("36"), "un_doz", "un_each", units)
	if !ok {
		t.Fatal("a cost per dozen must be expressible per each")
	}
	if !got.Equal(decimal.RequireFromString("3")) {
		t.Errorf("cost per each = %s, want 3", got)
	}
}

func TestCostOnBasis_SameUnitIsUnchanged(t *testing.T) {
	got, ok := costOnBasis(decimal.RequireFromString("4.25"), "un_pair", "un_pair", countUnits())
	if !ok || !got.Equal(decimal.RequireFromString("4.25")) {
		t.Errorf("costOnBasis(same unit) = %s, %v; want 4.25, true", got, ok)
	}
}

// Units of different dimensions share no base measure, so there is nothing to convert through.
func TestCostOnBasis_RefusesAcrossDimensions(t *testing.T) {
	if _, ok := costOnBasis(decimal.RequireFromString("3"), "un_kg", "un_pair", countUnits()); ok {
		t.Error("a cost per kilogram must not be restated per pair")
	}
}

// A rate against an affine unit has no single multiplier, so converting it would produce a confidently wrong margin.
func TestCostOnBasis_RefusesAffineUnits(t *testing.T) {
	if _, ok := costOnBasis(decimal.RequireFromString("3"), "un_degc", "un_pair", countUnits()); ok {
		t.Error("a cost against an offset unit must not be rescaled by a single factor")
	}
}

func TestCostOnBasis_RefusesUnknownUnits(t *testing.T) {
	if _, ok := costOnBasis(decimal.RequireFromString("3"), "un_missing", "un_pair", countUnits()); ok {
		t.Error("an unknown cost unit must not convert")
	}
	if _, ok := costOnBasis(decimal.RequireFromString("3"), "un_each", "un_missing", countUnits()); ok {
		t.Error("an unknown price unit must not convert")
	}
}

// This is the bug the analysis shipped with: costs are kept per each while prices are contracted per pair, and dropping the mismatches left prices unassessed even though every product had a cost.
func TestRepresentativeCost_ConvertsCostsRecordedOnAnotherBasis(t *testing.T) {
	entries := []pricingCostEntry{
		{productLineID: "pl_1", attributeIDs: []string{"at_sz6"}, unitCost: decimal.RequireFromString("3"), denominatorUnit: "un_each"},
		{productLineID: "pl_1", attributeIDs: []string{"at_sz8"}, unitCost: decimal.RequireFromString("4"), denominatorUnit: "un_each"},
	}

	got, ok := representativeCost(entries, "pl_1", nil, "un_pair", countUnits())
	if !ok {
		t.Fatal("costs recorded per each must still assess a price contracted per pair")
	}
	// Median of $3 and $4 per each is $3.50 per each, which is $7 per pair.
	if !got.Equal(decimal.RequireFromString("7")) {
		t.Errorf("representative cost = %s, want 7", got)
	}
}

// A cost that genuinely cannot be restated is still excluded, so an unassessable price is reported as unassessed rather than assessed wrongly.
func TestRepresentativeCost_StillExcludesUnconvertibleCosts(t *testing.T) {
	entries := []pricingCostEntry{
		{productLineID: "pl_1", unitCost: decimal.RequireFromString("3"), denominatorUnit: "un_kg"},
	}

	if _, ok := representativeCost(entries, "pl_1", nil, "un_pair", countUnits()); ok {
		t.Error("a cost per kilogram must not assess a price per pair")
	}
}

func TestCostCoverageNote_ReportsTheShareOfTheCatalog(t *testing.T) {
	got := costCoverageNote("no unit cost recorded", 20, 2000)
	want := "20 of 2000 products (1.0%) have no unit cost recorded, so prices covering only those products were not margin-checked."
	if got != want {
		t.Errorf("note = %q, want %q", got, want)
	}
}
