package scheduling

import (
	"math"
	"testing"
	"time"
)

func mo(year int, month time.Month) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}

// series builds n months of constant demand ending at the given month.
func series(endYear int, endMonth time.Month, n int, value float64) []MonthlyDemand {
	out := make([]MonthlyDemand, n)
	for i := range n {
		out[i] = MonthlyDemand{
			MonthStart: time.Date(endYear, endMonth-time.Month(n-1-i), 1, 0, 0, 0, 0, time.UTC),
			Quantity:   value,
		}
	}
	return out
}

func TestResolveDemand_TrailingTwelveExcludesPartialMonth(t *testing.T) {
	t.Parallel()

	// 24 months of 100/month ending May 2026; planning as of mid-June 2026.
	in := DemandInput{
		AsOf:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:     DemandBasisTrailing12,
		WeeksPerYear:  52,
		MonthlyByItem: map[string][]MonthlyDemand{"it_A": series(2026, time.May, 24, 100)},
	}

	got, _ := ResolveDemand(in)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TrailingAnnual != 1200 {
		t.Errorf("trailing annual = %v, want 1200 (twelve complete months)", got[0].TrailingAnnual)
	}
}

// The current month is always partial. Counting it would read as a collapse in demand
// and shrink every campaign.
func TestResolveDemand_IgnoresCurrentPartialMonth(t *testing.T) {
	t.Parallel()

	withPartial := append(series(2026, time.May, 12, 100),
		MonthlyDemand{MonthStart: mo(2026, time.June), Quantity: 3})

	got, _ := ResolveDemand(DemandInput{
		AsOf:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:     DemandBasisTrailing12,
		WeeksPerYear:  52,
		MonthlyByItem: map[string][]MonthlyDemand{"it_A": withPartial},
	})

	if got[0].TrailingAnnual != 1200 {
		t.Errorf("trailing annual = %v, want 1200; the partial month must be excluded", got[0].TrailingAnnual)
	}
}

func TestResolveDemand_SeasonalEMABasisUsesForecast(t *testing.T) {
	t.Parallel()

	in := DemandInput{
		AsOf:           time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:      DemandBasisSeasonalEMA,
		WeeksPerYear:   52,
		ForecastMonths: 12,
		ForecastZ:      1,
		MonthlyByItem:  map[string][]MonthlyDemand{"it_A": series(2026, time.May, 24, 100)},
	}

	got, _ := ResolveDemand(in)
	// A flat history forecasts flat, so the forward year matches the trailing year.
	if math.Abs(got[0].AnnualDemand-1200) > 1 {
		t.Errorf("forecast annual = %v, want ~1200 for a flat history", got[0].AnnualDemand)
	}
}

func TestResolveDemand_AbsoluteOverrideReplacesMonth(t *testing.T) {
	t.Parallel()

	got, applied := ResolveDemand(DemandInput{
		AsOf:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:     DemandBasisTrailing12,
		WeeksPerYear:  52,
		MonthlyByItem: map[string][]MonthlyDemand{"it_A": series(2026, time.May, 12, 100)},
		Overrides: []DemandOverride{{
			ID: "ov_1", ScopeCode: OverrideScopeItem, ScopeRefID: "it_A",
			PeriodStart: mo(2026, time.May), PeriodEnd: mo(2026, time.May),
			TypeCode: OverrideTypeAbsolute, Value: 500, ReasonCode: "new_customer",
		}},
	})

	// 11 months at 100 plus the overridden month at 500.
	if got[0].TrailingAnnual != 1600 {
		t.Errorf("trailing annual = %v, want 1600 (1100 + 500)", got[0].TrailingAnnual)
	}
	if len(applied) != 1 {
		t.Fatalf("applied = %d, want 1; an override that moves a number must be recorded", len(applied))
	}
	if applied[0].Before != 100 || applied[0].After != 500 {
		t.Errorf("applied = %v -> %v, want 100 -> 500", applied[0].Before, applied[0].After)
	}
	if applied[0].ReasonCode != "new_customer" {
		t.Errorf("reason = %q, want new_customer; the plan must be able to explain itself", applied[0].ReasonCode)
	}
}

func TestResolveDemand_DeltaOverridesStack(t *testing.T) {
	t.Parallel()

	got, _ := ResolveDemand(DemandInput{
		AsOf:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:     DemandBasisTrailing12,
		WeeksPerYear:  52,
		MonthlyByItem: map[string][]MonthlyDemand{"it_A": series(2026, time.May, 12, 100)},
		Overrides: []DemandOverride{
			// Deliberately supplied percent-first to prove ordering is enforced.
			{ID: "ov_pct", ScopeCode: OverrideScopeItem, ScopeRefID: "it_A",
				PeriodStart: mo(2026, time.May), PeriodEnd: mo(2026, time.May),
				TypeCode: OverrideTypeDeltaPercent, Value: 100},
			{ID: "ov_units", ScopeCode: OverrideScopeItem, ScopeRefID: "it_A",
				PeriodStart: mo(2026, time.May), PeriodEnd: mo(2026, time.May),
				TypeCode: OverrideTypeDeltaUnits, Value: 50},
		},
	})

	// units first (100 + 50 = 150), then percent (150 * 2 = 300).
	// Applying percent first would give (100*2)+50 = 250.
	if got[0].TrailingAnnual != 1100+300 {
		t.Errorf("trailing annual = %v, want %v; units must apply before percent",
			got[0].TrailingAnnual, 1100+300)
	}
}

// "A large new customer is about to order" has to work for an item with no sales
// history, which means an override must be able to create a month.
func TestResolveDemand_OverrideCreatesMonthWithNoHistory(t *testing.T) {
	t.Parallel()

	got, applied := ResolveDemand(DemandInput{
		AsOf:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:     DemandBasisTrailing12,
		WeeksPerYear:  52,
		MonthlyByItem: map[string][]MonthlyDemand{"it_A": {}},
		Overrides: []DemandOverride{{
			ID: "ov_1", ScopeCode: OverrideScopeItem, ScopeRefID: "it_A",
			PeriodStart: mo(2026, time.April), PeriodEnd: mo(2026, time.May),
			TypeCode: OverrideTypeAbsolute, Value: 400,
		}},
	})

	if got[0].TrailingAnnual != 800 {
		t.Errorf("trailing annual = %v, want 800 (two created months at 400)", got[0].TrailingAnnual)
	}
	if len(applied) != 2 {
		t.Errorf("applied = %d, want 2 (one per month in the period)", len(applied))
	}
}

func TestResolveDemand_ProductLineOverrideReachesItems(t *testing.T) {
	t.Parallel()

	got, _ := ResolveDemand(DemandInput{
		AsOf:         time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:    DemandBasisTrailing12,
		WeeksPerYear: 52,
		MonthlyByItem: map[string][]MonthlyDemand{
			"it_A": series(2026, time.May, 12, 100),
			"it_B": series(2026, time.May, 12, 100),
		},
		ItemsByProductLine: map[string][]string{"pdln_1": {"it_A"}},
		Overrides: []DemandOverride{{
			ID: "ov_1", ScopeCode: OverrideScopeProductLine, ScopeRefID: "pdln_1",
			PeriodStart: mo(2026, time.May), PeriodEnd: mo(2026, time.May),
			TypeCode: OverrideTypeDeltaPercent, Value: 100,
		}},
	})

	byItem := map[string]ItemDemand{}
	for _, d := range got {
		byItem[d.ItemID] = d
	}
	if byItem["it_A"].TrailingAnnual != 1300 {
		t.Errorf("it_A = %v, want 1300 (its May doubled)", byItem["it_A"].TrailingAnnual)
	}
	if byItem["it_B"].TrailingAnnual != 1200 {
		t.Errorf("it_B = %v, want 1200; an item outside the line must be untouched", byItem["it_B"].TrailingAnnual)
	}
}

func TestResolveDemand_OverrideCannotDriveDemandNegative(t *testing.T) {
	t.Parallel()

	got, _ := ResolveDemand(DemandInput{
		AsOf:          time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:     DemandBasisTrailing12,
		WeeksPerYear:  52,
		MonthlyByItem: map[string][]MonthlyDemand{"it_A": series(2026, time.May, 12, 100)},
		Overrides: []DemandOverride{{
			ID: "ov_1", ScopeCode: OverrideScopeItem, ScopeRefID: "it_A",
			PeriodStart: mo(2026, time.May), PeriodEnd: mo(2026, time.May),
			TypeCode: OverrideTypeDeltaUnits, Value: -5000,
		}},
	})

	if got[0].TrailingAnnual != 1100 {
		t.Errorf("trailing annual = %v, want 1100; the month must floor at zero", got[0].TrailingAnnual)
	}
}

func TestResolveDemand_PoolsDownstreamSigmaAsRootSumSquares(t *testing.T) {
	t.Parallel()

	// Two downstream SKUs with identical variability.
	downstream := []FinishedGood{
		{ItemID: "it_FG1", SKU: "FG-1", Monthly: []MonthlyDemand{{mo(2026, time.April), 100}, {mo(2026, time.May), 200}}},
		{ItemID: "it_FG2", SKU: "FG-2", Monthly: []MonthlyDemand{{mo(2026, time.April), 100}, {mo(2026, time.May), 200}}},
	}

	got, _ := ResolveDemand(DemandInput{
		AsOf:             time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		BasisCode:        DemandBasisTrailing12,
		WeeksPerYear:     52,
		MonthlyByItem:    map[string][]MonthlyDemand{"it_A": series(2026, time.May, 12, 100)},
		DownstreamByItem: map[string][]FinishedGood{"it_A": downstream},
	})

	// Pooling must be sqrt(2)*sigma, not 2*sigma: independent variability partially
	// cancels, which is the whole reason the buffer sits at the constraint.
	single := got[0].SigmaDownstreamSum / 2
	wantPooled := single * math.Sqrt2
	if math.Abs(got[0].SigmaWeeklyPooled-wantPooled) > 1e-9 {
		t.Errorf("pooled sigma = %v, want %v (root-sum-squares, not the plain sum %v)",
			got[0].SigmaWeeklyPooled, wantPooled, got[0].SigmaDownstreamSum)
	}
}
