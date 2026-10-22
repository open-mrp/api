package scheduling

import (
	"testing"
	"time"
)

// tightsSolverInput is a pair of tights knitted as two items, a left leg and a right leg, joined by one
// sewing step. The load attributes the sewn stage and the finished tights to both legs, so each leg
// carries the whole family's demand and the finished good appears under both.
func tightsSolverInput() SolverInput {
	tights := series(2026, time.May, 12, 100*52.0/12)
	family := []FinishedGood{{ItemID: "it_tights", SKU: "TIGHTS", Monthly: tights}}

	return SolverInput{
		AccountID:    "ac_1",
		PlanningAsOf: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		Settings:     sockSettings(),
		Machines:     []Machine{{ID: "mc_1", Name: "1"}, {ID: "mc_2", Name: "2"}},

		ConfiguredEligibility:    map[string]map[string]bool{"it_left": {"mc_1": true}, "it_right": {"mc_2": true}},
		ConfiguredItemSKU:        map[string]string{"it_left": "LEG-L", "it_right": "LEG-R"},
		ConfiguredSecondsPerUnit: map[string]float64{"it_left": 10, "it_right": 10},

		MonthlyByItem:    map[string][]MonthlyDemand{"it_left": tights, "it_right": tights},
		DownstreamByItem: map[string][]FinishedGood{"it_left": family, "it_right": family},
		OnHandByItem:     map[string]float64{},
		DemandBasisCode:  DemandBasisTrailing12,
	}
}

// Both legs of a pair are knitted, in equal numbers.
func TestSolve_AssembledFamilyKnitsEveryLeg(t *testing.T) {
	t.Parallel()

	got := Solve(tightsSolverInput())

	left, right := campaignUnitsInWeek(got.Campaigns, "it_left", 0), campaignUnitsInWeek(got.Campaigns, "it_right", 0)
	if left <= 0 || right <= 0 {
		t.Fatalf("week 0 knits %v left and %v right legs; a pair needs both", left, right)
	}
	if left != right {
		t.Errorf("week 0 knits %v left and %v right legs; the legs of a pair are owed in equal numbers", left, right)
	}
}

// The finished good sits in both legs' families but is one SKU, finished once: reporting it under each
// leg would plan its finishing twice and write two policies for one SKU.
func TestSolve_AssembledFinishedGoodIsReportedOnce(t *testing.T) {
	t.Parallel()

	got := Solve(tightsSolverInput())

	var count int
	for _, p := range got.FinishedPolicies {
		if p.ItemID == "it_tights" {
			count++
			if p.GreigeItemID != "it_left" {
				t.Errorf("tights reported under %q, want the first leg in SKU order", p.GreigeItemID)
			}
		}
	}
	if count != 1 {
		t.Fatalf("tights reported %d times, want once", count)
	}
}

// An order for tights is owed by both legs, so it arrives once per leg. The finishing stage makes the
// tights themselves and owes the order once.
func TestFirmDemandByFinishedItem_CountsAnOrderOwedByTwoLegsOnce(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC)
	shipBy := start.AddDate(0, 0, 8)
	in := SolverInput{
		Settings:     sockSettings(),
		HorizonStart: start,
		OpenOrders: []OpenOrderLine{
			{SalesOrderID: "so_1", SalesOrderLineID: "sol_1", FinishedItemID: "it_tights", ConstraintItemID: "it_left", Units: 12, ShipByDate: &shipBy},
			{SalesOrderID: "so_1", SalesOrderLineID: "sol_1", FinishedItemID: "it_tights", ConstraintItemID: "it_right", Units: 12, ShipByDate: &shipBy},
			{SalesOrderID: "so_2", SalesOrderLineID: "sol_2", FinishedItemID: "it_tights", ConstraintItemID: "it_left", Units: 5, ShipByDate: &shipBy},
		},
	}

	got := firmDemandByFinishedItem(in)

	if got["it_tights"][1] != 17 {
		t.Errorf("week 1 owes %v tights, want 17 (12 + 5, each order once)", got["it_tights"][1])
	}
}
