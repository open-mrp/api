package scheduling

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func at(hoursAgo int) time.Time {
	return time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC).Add(-time.Duration(hoursAgo) * time.Hour)
}

// The unit abbreviation on the labor-time rate decides its scale. Getting this wrong
// silently rescales every run hour in the plan by 60x.
func TestSecondsPerUnitFromLaborTime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value float64
		unit  string
		want  float64
	}{
		{2, "min", 120},
		{2, "MIN", 120},
		{2, "minutes", 120},
		{1.5, "hr", 5400},
		{1.5, "h", 5400},
		{1.5, "hour", 5400},
		{30, "sec", 30},
		{30, "", 30},
		{30, "unrecognized", 30},
	}

	for _, c := range cases {
		if got := SecondsPerUnitFromLaborTime(c.value, c.unit); got != c.want {
			t.Errorf("SecondsPerUnitFromLaborTime(%v, %q) = %v, want %v", c.value, c.unit, got, c.want)
		}
	}
}

func TestMeasureItems_OneBatchIsOneLot(t *testing.T) {
	t.Parallel()

	// Wildly different quantities: the lot count must still be the batch count.
	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: at(3), Quantity: 60, MachineID: "mc_1"},
		{BatchID: "bt_2", ItemID: "it_A", SKU: "A", ScannedAt: at(2), Quantity: 6000, MachineID: "mc_1"},
	}

	got := MeasureItems(batches)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].LotCount != 2 {
		t.Errorf("lot count = %d, want 2 (one per batch, regardless of quantity)", got[0].LotCount)
	}
	if got[0].Quantity != 6060 {
		t.Errorf("quantity = %v, want 6060", got[0].Quantity)
	}
}

func TestMeasureItems_CollectsMachineAffinity(t *testing.T) {
	t.Parallel()

	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: at(3), MachineID: "mc_1"},
		{BatchID: "bt_2", ItemID: "it_A", SKU: "A", ScannedAt: at(2), MachineID: "mc_2"},
		{BatchID: "bt_3", ItemID: "it_A", SKU: "A", ScannedAt: at(1), MachineID: "mc_1"},
	}

	got := MeasureItems(batches)
	want := map[string]bool{"mc_1": true, "mc_2": true}
	if !reflect.DeepEqual(got[0].EligibleMachineID, want) {
		t.Errorf("eligible machines = %v, want %v", got[0].EligibleMachineID, want)
	}
}

func TestMeasureItems_MeanLeadTimeInWeeks(t *testing.T) {
	t.Parallel()

	opened := at(24 * 14) // 14 days before the reference point
	scanned := at(24 * 7) // 7 days before, so a 7-day lead time

	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: scanned, RunCreatedAt: &opened},
	}

	got := MeasureItems(batches)
	if math.Abs(got[0].MeasuredLeadTimeWeeks-1) > 1e-9 {
		t.Errorf("lead time = %v weeks, want 1", got[0].MeasuredLeadTimeWeeks)
	}
}

// A run left open for months is a data-entry artifact, and a handful of them would
// dominate the mean and inflate every safety stock.
func TestMeasureItems_DiscardsImplausibleLeadTimes(t *testing.T) {
	t.Parallel()

	sane := at(24 * 14)
	insane := at(24 * 400)
	scanned := at(24 * 7)

	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: scanned, RunCreatedAt: &sane},
		{BatchID: "bt_2", ItemID: "it_A", SKU: "A", ScannedAt: scanned, RunCreatedAt: &insane},
	}

	got := MeasureItems(batches)
	if got[0].LeadTimeSampleCount != 1 {
		t.Errorf("samples = %d, want 1; the 393-day sample must be discarded", got[0].LeadTimeSampleCount)
	}
	if math.Abs(got[0].MeasuredLeadTimeWeeks-1) > 1e-9 {
		t.Errorf("lead time = %v weeks, want 1", got[0].MeasuredLeadTimeWeeks)
	}
}

func TestMeasureItems_SortedBySKUForDeterminism(t *testing.T) {
	t.Parallel()

	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_C", SKU: "C", ScannedAt: at(3)},
		{BatchID: "bt_2", ItemID: "it_A", SKU: "A", ScannedAt: at(2)},
		{BatchID: "bt_3", ItemID: "it_B", SKU: "B", ScannedAt: at(1)},
	}

	for range 20 {
		got := MeasureItems(batches)
		if got[0].SKU != "A" || got[1].SKU != "B" || got[2].SKU != "C" {
			t.Fatalf("order = %s/%s/%s, want A/B/C on every run", got[0].SKU, got[1].SKU, got[2].SKU)
		}
	}
}

// Consecutive batches of the same item are one campaign, so they cost one changeover
// between them, not one per batch.
func TestAverageInputsAdded_RunLengthEncodesCampaigns(t *testing.T) {
	t.Parallel()

	stepInputs := map[string]map[string]bool{
		"prs_A": {"yarn_1": true, "yarn_2": true},
		"prs_B": {"yarn_1": true, "yarn_3": true, "yarn_4": true},
	}

	// A, A, A, B — one transition, introducing yarn_3 and yarn_4.
	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", ScannedAt: at(4), MachineID: "mc_1", ProductionStepID: "prs_A"},
		{BatchID: "bt_2", ItemID: "it_A", ScannedAt: at(3), MachineID: "mc_1", ProductionStepID: "prs_A"},
		{BatchID: "bt_3", ItemID: "it_A", ScannedAt: at(2), MachineID: "mc_1", ProductionStepID: "prs_A"},
		{BatchID: "bt_4", ItemID: "it_B", ScannedAt: at(1), MachineID: "mc_1", ProductionStepID: "prs_B"},
	}

	if got := AverageInputsAdded(batches, stepInputs); got != 2 {
		t.Errorf("average added = %v, want 2 (one transition adding yarn_3 and yarn_4)", got)
	}
}

// Only additions cost time: removing a yarn is free, threading a new one is not.
func TestAverageInputsAdded_RemovalsAreFree(t *testing.T) {
	t.Parallel()

	stepInputs := map[string]map[string]bool{
		"prs_A": {"yarn_1": true, "yarn_2": true, "yarn_3": true},
		"prs_B": {"yarn_1": true},
	}

	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", ScannedAt: at(2), MachineID: "mc_1", ProductionStepID: "prs_A"},
		{BatchID: "bt_2", ItemID: "it_B", ScannedAt: at(1), MachineID: "mc_1", ProductionStepID: "prs_B"},
	}

	if got := AverageInputsAdded(batches, stepInputs); got != 0 {
		t.Errorf("average added = %v, want 0; dropping yarns costs nothing", got)
	}
}

func TestAverageInputsAdded_NoTransitionsReturnsZero(t *testing.T) {
	t.Parallel()

	batches := []BatchMeasurement{
		{BatchID: "bt_1", ItemID: "it_A", ScannedAt: at(1), MachineID: "mc_1", ProductionStepID: "prs_A"},
	}
	if got := AverageInputsAdded(batches, map[string]map[string]bool{}); got != 0 {
		t.Errorf("average added = %v, want 0 with a single campaign", got)
	}
}

// ──────────────────────────────────────────────
// End-to-end
// ──────────────────────────────────────────────

func TestSolve_ProducesAPlanFromRawHistory(t *testing.T) {
	t.Parallel()

	settings := DefaultSettings()
	settings.HorizonWeeks = 4

	opened := at(24 * 14)
	scanned := at(24 * 7)

	in := SolverInput{
		AccountID:    "ac_1",
		PlanningAsOf: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		Settings:     settings,
		Machines:     []Machine{{ID: "mc_1", Name: "1"}, {ID: "mc_2", Name: "2"}},
		Batches: []BatchMeasurement{
			{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: scanned, Quantity: 60,
				MachineID: "mc_1", ProductionStepID: "prs_A", UnitCost: 4,
				LaborTimeValue: 30, LaborTimeUnit: "min", LaborRate: 18, OverheadRate: 2,
				RunCreatedAt: &opened},
			{BatchID: "bt_2", ItemID: "it_B", SKU: "B", ScannedAt: scanned, Quantity: 60,
				MachineID: "mc_2", ProductionStepID: "prs_B", UnitCost: 6,
				LaborTimeValue: 20, LaborTimeUnit: "min", LaborRate: 18, OverheadRate: 2,
				RunCreatedAt: &opened},
		},
		StepInputs: map[string]map[string]bool{
			"prs_A": {"yarn_1": true},
			"prs_B": {"yarn_1": true, "yarn_2": true},
		},
		MonthlyByItem: map[string][]MonthlyDemand{
			"it_A": series(2026, time.May, 24, 500),
			"it_B": series(2026, time.May, 24, 300),
		},
		OnHandByItem:    map[string]float64{"it_A": 0, "it_B": 0},
		DemandBasisCode: DemandBasisTrailing12,
	}

	got := Solve(in)

	if got.SolverVersion != SolverVersion {
		t.Errorf("solver version = %q, want %q", got.SolverVersion, SolverVersion)
	}
	if len(got.Policies) != 2 {
		t.Fatalf("policies = %d, want 2", len(got.Policies))
	}
	if len(got.Campaigns) == 0 {
		t.Fatal("no campaigns planned for two items with demand and no stock")
	}
	for _, c := range got.Campaigns {
		if c.RunHours <= 0 {
			t.Errorf("campaign %s has no run hours", c.SKU)
		}
		if c.MachineID == "" {
			t.Errorf("campaign %s was not assigned a machine", c.SKU)
		}
	}
	if got.ProjectedOnHand["it_A"] == nil {
		t.Error("no projected on-hand series for it_A")
	}
}

// An item with no measured run rate cannot be leveled — there is no way to know how
// much machine time it needs. It must be reported, not silently dropped.
func TestSolve_ReportsItemsWithoutRunRate(t *testing.T) {
	t.Parallel()

	in := SolverInput{
		PlanningAsOf: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		Settings:     DefaultSettings(),
		Machines:     []Machine{{ID: "mc_1", Name: "1"}},
		Batches: []BatchMeasurement{
			{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: at(1), Quantity: 60, MachineID: "mc_1"},
		},
		MonthlyByItem:   map[string][]MonthlyDemand{"it_A": series(2026, time.May, 12, 100)},
		DemandBasisCode: DemandBasisTrailing12,
	}

	got := Solve(in)
	if len(got.Diagnostics.ItemsWithoutRunRate) != 1 {
		t.Errorf("itemsWithoutRunRate = %v, want [A]", got.Diagnostics.ItemsWithoutRunRate)
	}
	if len(got.Campaigns) != 0 {
		t.Error("an item with no run rate must not be scheduled")
	}
}

func TestSolve_HonoursExcludedItems(t *testing.T) {
	t.Parallel()

	in := SolverInput{
		PlanningAsOf: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		Settings:     DefaultSettings(),
		Machines:     []Machine{{ID: "mc_1", Name: "1"}},
		Batches: []BatchMeasurement{
			{BatchID: "bt_1", ItemID: "it_A", SKU: "A", ScannedAt: at(1), Quantity: 60,
				MachineID: "mc_1", ProductionStepID: "prs_A", UnitCost: 4,
				LaborTimeValue: 10, LaborTimeUnit: "min"},
		},
		MonthlyByItem:   map[string][]MonthlyDemand{"it_A": series(2026, time.May, 12, 100)},
		ExcludedItemIDs: map[string]bool{"it_A": true},
		DemandBasisCode: DemandBasisTrailing12,
	}

	got := Solve(in)
	if got.Diagnostics.ExcludedItemCount != 1 {
		t.Errorf("excluded count = %d, want 1", got.Diagnostics.ExcludedItemCount)
	}
	if len(got.Campaigns) != 0 {
		t.Error("an excluded item must not be scheduled")
	}
}

func TestSolve_Deterministic(t *testing.T) {
	t.Parallel()

	settings := DefaultSettings()
	settings.HorizonWeeks = 6

	in := SolverInput{
		PlanningAsOf: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		Settings:     settings,
		Machines:     []Machine{{ID: "mc_2", Name: "2"}, {ID: "mc_10", Name: "10"}, {ID: "mc_1", Name: "1"}},
		Batches: []BatchMeasurement{
			{BatchID: "bt_1", ItemID: "it_C", SKU: "C", ScannedAt: at(4), Quantity: 60, MachineID: "mc_1",
				ProductionStepID: "prs_C", UnitCost: 3, LaborTimeValue: 25, LaborTimeUnit: "min"},
			{BatchID: "bt_2", ItemID: "it_A", SKU: "A", ScannedAt: at(3), Quantity: 60, MachineID: "mc_2",
				ProductionStepID: "prs_A", UnitCost: 4, LaborTimeValue: 30, LaborTimeUnit: "min"},
			{BatchID: "bt_3", ItemID: "it_B", SKU: "B", ScannedAt: at(2), Quantity: 60, MachineID: "mc_10",
				ProductionStepID: "prs_B", UnitCost: 5, LaborTimeValue: 20, LaborTimeUnit: "min"},
		},
		MonthlyByItem: map[string][]MonthlyDemand{
			"it_A": series(2026, time.May, 24, 400),
			"it_B": series(2026, time.May, 24, 350),
			"it_C": series(2026, time.May, 24, 300),
		},
		OnHandByItem:    map[string]float64{"it_A": 100, "it_B": 0, "it_C": 50},
		DemandBasisCode: DemandBasisTrailing12,
	}

	first := Solve(in)
	for run := range 50 {
		got := Solve(in)
		if !reflect.DeepEqual(first.Campaigns, got.Campaigns) {
			t.Fatalf("run %d produced different campaigns; Solve must be deterministic", run)
		}
		if !reflect.DeepEqual(first.Policies, got.Policies) {
			t.Fatalf("run %d produced different policies; Solve must be deterministic", run)
		}
	}
}
