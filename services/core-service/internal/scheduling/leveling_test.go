package scheduling

import (
	"math"
	"reflect"
	"testing"
)

func testSettings() Settings {
	s := DefaultSettings()
	s.HorizonWeeks = 4
	return s
}

// item builds a levelling item with a policy already computed from simple inputs.
func item(sku string, weeklyDemand, secondsPerUnit, onHand, eoq float64) LevellingItem {
	return LevellingItem{
		Policy: ItemPolicy{
			ItemID:         "it_" + sku,
			SKU:            sku,
			WeeklyDemand:   weeklyDemand,
			AnnualDemand:   weeklyDemand * 52,
			SecondsPerUnit: secondsPerUnit,
			EOQUnits:       eoq,
			ReorderPoint:   weeklyDemand * 4,
			OrderUpTo:      weeklyDemand * 12,
			OnHandEchelon:  onHand,
		},
		LotUnits: 60,
	}
}

func TestRoundUpToLot_FloorsAtOneLot(t *testing.T) {
	t.Parallel()

	if got := roundUpToLot(10, 60); got != 60 {
		t.Errorf("roundUpToLot(10, 60) = %v, want 60 (never less than one lot)", got)
	}
	if got := roundUpToLot(61, 60); got != 120 {
		t.Errorf("roundUpToLot(61, 60) = %v, want 120 (rounds up)", got)
	}
	if got := roundUpToLot(120, 60); got != 120 {
		t.Errorf("roundUpToLot(120, 60) = %v, want 120 (exact stays put)", got)
	}
}

// The economic lot rounds UP and the capacity ceiling rounds DOWN. Rounding the
// ceiling up would emit campaigns that cannot physically run in the week.
func TestMaxLotsInCapacity_RoundsDownUnlikeEconomicLot(t *testing.T) {
	t.Parallel()

	// 63 hours of capacity at 3600 s/unit = 63 units -> one whole 60-unit lot.
	if got := maxLotsInCapacity(63, 3600, 60); got != 60 {
		t.Errorf("maxLotsInCapacity = %v, want 60 (rounds down to whole lots)", got)
	}
	// Even when nothing fits, one lot is returned; the caller flags it unschedulable
	// by comparing its run hours to capacity rather than getting a zero here.
	if got := maxLotsInCapacity(0.1, 3600, 60); got != 60 {
		t.Errorf("maxLotsInCapacity = %v, want 60 floor", got)
	}
}

// Demand must be subtracted AFTER the week's campaigns land. Subtracting first lets
// an item dip below its trigger and be rebuilt in the same week, double-counting a
// week of consumption across the horizon.
func TestLevel_DemandDrawnDownAfterPlacement(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	items := []LevellingItem{item("A", 100, 10, 0, 600)}
	machines := []Machine{{ID: "mc_1", Name: "1"}}

	got := Level(items, machines, s, nil)
	if len(got.Campaigns) != 1 {
		t.Fatalf("campaigns = %d, want 1", len(got.Campaigns))
	}

	// Position = 0 (on hand) + 600 (campaign) - 100 (demand) = 500.
	if want := 500.0; got.ProjectedOnHand["it_A"][0] != want {
		t.Errorf("projected on hand = %v, want %v", got.ProjectedOnHand["it_A"][0], want)
	}
}

func TestLevel_RespectsMachineCapacity(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	capacity := s.MachineWeeklyCapacityHours() // 2 * 7 * 5 * 0.9 = 63h

	// Each campaign is 60 units at 1800 s/unit = 30h, so only two fit in 63h.
	items := []LevellingItem{
		item("A", 100, 1800, 0, 60),
		item("B", 100, 1800, 0, 60),
		item("C", 100, 1800, 0, 60),
	}
	machines := []Machine{{ID: "mc_1", Name: "1"}}

	got := Level(items, machines, s, nil)
	if len(got.Campaigns) != 2 {
		t.Fatalf("campaigns = %d, want 2 (third exceeds %vh capacity)", len(got.Campaigns), capacity)
	}

	var totalHours float64
	for _, c := range got.Campaigns {
		totalHours += c.RunHours
	}
	if totalHours > capacity {
		t.Errorf("planned %vh on one machine, capacity is %vh", totalHours, capacity)
	}
}

func TestLevel_BalancesAcrossMachinesLeastLoadedFirst(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	items := []LevellingItem{
		item("A", 100, 1800, 0, 60),
		item("B", 100, 1800, 0, 60),
	}
	machines := []Machine{{ID: "mc_1", Name: "1"}, {ID: "mc_2", Name: "2"}}

	got := Level(items, machines, s, nil)
	if len(got.Campaigns) != 2 {
		t.Fatalf("campaigns = %d, want 2", len(got.Campaigns))
	}
	if got.Campaigns[0].MachineID == got.Campaigns[1].MachineID {
		t.Error("both campaigns landed on one machine; load must spread to the least-loaded")
	}
}

func TestLevel_HonoursMachineEligibility(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	restricted := item("A", 100, 1800, 0, 60)
	restricted.EligibleMachineID = map[string]bool{"mc_2": true}

	got := Level([]LevellingItem{restricted}, []Machine{{ID: "mc_1", Name: "1"}, {ID: "mc_2", Name: "2"}}, s, nil)
	if len(got.Campaigns) != 1 {
		t.Fatalf("campaigns = %d, want 1", len(got.Campaigns))
	}
	if got.Campaigns[0].MachineID != "mc_2" {
		t.Errorf("machine = %q, want mc_2; an item must not run on an ineligible machine", got.Campaigns[0].MachineID)
	}
}

// A well-stocked item should not be built. This is the whole point of netting against
// inventory rather than scheduling to gross demand.
func TestLevel_SkipsItemsAboveTrigger(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	// On hand 10,000 against weekly demand of 100 is far above any trigger.
	got := Level([]LevellingItem{item("A", 100, 1800, 10_000, 60)}, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)

	if len(got.Campaigns) != 0 {
		t.Errorf("campaigns = %d, want 0; a fully stocked item must not consume capacity", len(got.Campaigns))
	}
}

func TestLevel_ReportsCapacityStarvation(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	// Two items both below trigger, but each campaign eats the whole machine-week.
	items := []LevellingItem{
		item("A", 100, 3600, 0, 60),
		item("B", 100, 3600, 0, 60),
	}
	got := Level(items, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)

	if len(got.Diagnostics.CapacityStarvedSKUs) == 0 {
		t.Error("expected a capacity-starved SKU to be reported rather than silently dropped")
	}
}

func TestLevel_FlagsUnschedulableWhenOneLotExceedsCapacity(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	// 60 units at 7200 s/unit = 120h, well past the 63h machine-week.
	got := Level([]LevellingItem{item("A", 100, 7200, 0, 60)}, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)

	if len(got.Diagnostics.UnschedulableSKUs) != 1 {
		t.Errorf("unschedulable = %v, want [A]; a lot that cannot fit must be surfaced", got.Diagnostics.UnschedulableSKUs)
	}
}

// Go randomizes map iteration, and the script relied on JS insertion order. Without
// explicit sorting the plan would differ run to run for identical input.
func TestLevel_Deterministic(t *testing.T) {
	t.Parallel()

	s := testSettings()
	items := []LevellingItem{
		item("D", 120, 900, 0, 180),
		item("A", 100, 1800, 0, 60),
		item("C", 80, 1200, 50, 120),
		item("B", 90, 1500, 10, 90),
	}
	machines := []Machine{{ID: "mc_2", Name: "2"}, {ID: "mc_1", Name: "1"}, {ID: "mc_10", Name: "10"}}

	first := Level(items, machines, s, nil)
	for run := range 50 {
		got := Level(items, machines, s, nil)
		if !reflect.DeepEqual(first.Campaigns, got.Campaigns) {
			t.Fatalf("run %d produced a different plan; the solver must be deterministic", run)
		}
	}
}

func TestNaturalLess_SortsEmbeddedNumbersNumerically(t *testing.T) {
	t.Parallel()

	cases := []struct {
		a, b string
		want bool
	}{
		{"9", "10", true},
		{"10", "9", false},
		{"Merz 9", "Merz 10", true},
		{"51", "52", true},
		{"Merz 2", "Merz 10", true},
		{"A", "B", true},
	}

	for _, c := range cases {
		if got := naturalLess(c.a, c.b); got != c.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestComputePolicy_EOQAndReorderPoint(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	p := ComputePolicy(PolicyInput{
		ItemID:                "it_A",
		SKU:                   "A",
		AnnualDemand:          5200, // 100/week
		SecondsPerUnit:        1800,
		UnitCost:              4,
		OverheadRate:          0,
		MeasuredLeadTimeWeeks: 2,
		SigmaWeeklyPooled:     10,
		SigmaDownstreamSum:    20,
		OnHandEchelon:         500,
	}, s)

	if math.Abs(p.WeeklyDemand-100) > 1e-9 {
		t.Errorf("weekly demand = %v, want 100", p.WeeklyDemand)
	}

	// setup = (30/60) * 20 = 10; holding = max(4*0.25, 0.01) = 1
	// EOQ = sqrt(2 * 5200 * 10 / 1) = sqrt(104000)
	wantEOQ := math.Sqrt(104_000)
	if math.Abs(p.EOQUnits-wantEOQ) > 1e-6 {
		t.Errorf("EOQ = %v, want %v", p.EOQUnits, wantEOQ)
	}

	// ROP = 100 * (2 + 6) + 1.645*10*sqrt(2) + 1.645*20*sqrt(6)
	wantROP := 100*(2+6) + 1.645*10*math.Sqrt(2) + 1.645*20*math.Sqrt(6)
	if math.Abs(p.ReorderPoint-wantROP) > 1e-6 {
		t.Errorf("ROP = %v, want %v", p.ReorderPoint, wantROP)
	}
}

func TestComputePolicy_FallsBackToDefaultLeadTime(t *testing.T) {
	t.Parallel()

	s := DefaultSettings()
	p := ComputePolicy(PolicyInput{SKU: "A", AnnualDemand: 520, UnitCost: 1}, s)

	if p.ConstraintLeadTimeWeeks != s.DefaultConstraintLeadTimeWeeks {
		t.Errorf("lead time = %v, want the %v default when unmeasured",
			p.ConstraintLeadTimeWeeks, s.DefaultConstraintLeadTimeWeeks)
	}
}

// A zero standard cost is a data problem; the holding floor stops it dividing EOQ to
// an unbounded campaign.
func TestComputePolicy_HoldingCostFloor(t *testing.T) {
	t.Parallel()

	p := ComputePolicy(PolicyInput{SKU: "A", AnnualDemand: 520, UnitCost: 0}, DefaultSettings())
	if p.HoldingCost != 0.01 {
		t.Errorf("holding = %v, want the 0.01 floor for a zero-cost item", p.HoldingCost)
	}
	if math.IsInf(p.EOQUnits, 0) || math.IsNaN(p.EOQUnits) {
		t.Errorf("EOQ = %v, want a finite value", p.EOQUnits)
	}
}

func TestClassifyABC_ByRunHoursDescending(t *testing.T) {
	t.Parallel()

	// A realistic spread: 20 items with a gentle decline, so no single item dominates.
	policies := make([]ItemPolicy, 20)
	for i := range policies {
		policies[i] = ItemPolicy{
			SKU:            string(rune('a' + i)),
			AnnualDemand:   float64(1000 - i*40),
			SecondsPerUnit: 100,
		}
	}

	got := ClassifyABC(policies)

	if got[0].ABCClass != "A" {
		t.Errorf("largest item class = %s, want A", got[0].ABCClass)
	}
	if got[len(got)-1].ABCClass != "C" {
		t.Errorf("smallest item class = %s, want C", got[len(got)-1].ABCClass)
	}
	// Sorted by run hours descending.
	for i := 1; i < len(got); i++ {
		if got[i-1].AnnualRunHours() < got[i].AnnualRunHours() {
			t.Fatalf("not sorted descending at index %d", i)
		}
	}
	// Classes must be non-decreasing down the list: no B above an A.
	rank := map[string]int{"A": 0, "B": 1, "C": 2}
	for i := 1; i < len(got); i++ {
		if rank[got[i].ABCClass] < rank[got[i-1].ABCClass] {
			t.Fatalf("class regressed from %s to %s at index %d", got[i-1].ABCClass, got[i].ABCClass, i)
		}
	}
}

// Documents a known quirk inherited from the script rather than hiding it: because the
// cumulative share is inclusive of the current item, a portfolio dominated by a single
// item classifies that item as C. Preserved deliberately for parity; ABC drives
// display and reporting, not the plan.
func TestClassifyABC_DominantItemFallsToC_KnownScriptParity(t *testing.T) {
	t.Parallel()

	policies := []ItemPolicy{
		{SKU: "small", AnnualDemand: 100, SecondsPerUnit: 1},
		{SKU: "huge", AnnualDemand: 100_000, SecondsPerUnit: 10},
	}

	got := ClassifyABC(policies)

	if got[0].SKU != "huge" {
		t.Fatalf("first = %s, want huge (sorted by run hours)", got[0].SKU)
	}
	if got[0].ABCClass != "C" {
		t.Errorf("dominant item class = %s, want C; if this changed, the parity gate "+
			"against the TS script must be re-run", got[0].ABCClass)
	}
}

func TestCalibrateChangeover_ReproducesTargetAverage(t *testing.T) {
	t.Parallel()

	// Average of 4 added inputs should land on the 30-minute target.
	c := CalibrateChangeover(15, 30, 90, 4)
	if got := c.Minutes(4); math.Abs(got-30) > 1e-9 {
		t.Errorf("Minutes(4) = %v, want 30 (the calibrated average)", got)
	}
	if got := c.Minutes(0); got != 15 {
		t.Errorf("Minutes(0) = %v, want the 15 minimum", got)
	}
	if got := c.Minutes(1000); got != 90 {
		t.Errorf("Minutes(1000) = %v, want the 90 maximum", got)
	}
}

func TestCalibrateChangeover_NoHistoryUsesMinimum(t *testing.T) {
	t.Parallel()

	c := CalibrateChangeover(15, 30, 90, 0)
	if c.Slope() != 0 {
		t.Errorf("slope = %v, want 0 when there is no history to calibrate against", c.Slope())
	}
	if got := c.Minutes(5); got != 15 {
		t.Errorf("Minutes(5) = %v, want 15", got)
	}
}

func TestSetupCost_FallsBackWhenRateMissing(t *testing.T) {
	t.Parallel()

	// Zero rates would make setup cost zero and collapse EOQ toward a lot size of one.
	if got := SetupCost(30, 0, 0); got != 4 {
		t.Errorf("SetupCost = %v, want 4 (0.5h at the 8/hr fallback)", got)
	}
}

// A pinned hand edit is a fact of the plan: its stock arrives, so the solver must not
// rebuild the item the sweep would otherwise find due.
func TestLevel_PinnedCampaignRaisesPositionSoSolverBuildsLess(t *testing.T) {
	t.Parallel()

	s := testSettings()
	// Weekly demand 10, ROP 40, on hand 0: due immediately, and without help the sweep
	// would build in week 0.
	unpinned := Level([]LevellingItem{item("A", 10, 600, 0, 600)}, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)
	if len(unpinned.Campaigns) == 0 || unpinned.Campaigns[0].WeekIndex != 0 {
		t.Fatalf("control run should build in week 0, got %+v", unpinned.Campaigns)
	}

	// A hand-pinned 600-unit build in week 0 covers the horizon; the solver adds nothing.
	pinned := Level([]LevellingItem{item("A", 10, 600, 0, 600)}, []Machine{{ID: "mc_1", Name: "1"}}, s,
		[]PinnedCampaign{{ItemID: "it_A", MachineID: "mc_1", WeekIndex: 0, Units: 600}})
	if len(pinned.Campaigns) != 0 {
		t.Errorf("solver re-built despite the pinned inflow: %+v", pinned.Campaigns)
	}
	// The projection includes the pinned inflow: 600 in, 10 demanded in week 0.
	if got := pinned.ProjectedOnHand["it_A"][0]; got != 590 {
		t.Errorf("projected on hand = %v, want 590 (pinned units minus one week of demand)", got)
	}
}

// A campaign building nothing is not a campaign. A zero-unit pin must not hold the slot
// it sits in, or the plan leaves the machine-week empty for work that will never run.
func TestLevel_ZeroUnitPinDoesNotHoldItsSlot(t *testing.T) {
	t.Parallel()

	s := testSettings()
	pins := []PinnedCampaign{{ItemID: "it_A", MachineID: "mc_1", WeekIndex: 0, Units: 0}}
	got := Level([]LevellingItem{item("A", 10, 600, 0, 600)}, []Machine{{ID: "mc_1", Name: "1"}}, s, pins)

	var placed bool
	for _, c := range got.Campaigns {
		if c.SKU == "A" && c.WeekIndex == 0 && c.MachineID == "mc_1" {
			placed = true
		}
	}
	if !placed {
		t.Errorf("the zero-unit pin blocked the slot: %+v", got.Campaigns)
	}
}

// A pinned campaign's run time occupies its machine, so another item that needs the
// same week must go elsewhere — or wait.
func TestLevel_PinnedCampaignConsumesCapacity(t *testing.T) {
	t.Parallel()

	s := testSettings()
	// One machine-week is 63 hours at default settings. The pin burns 60 of them
	// (360 units x 600 s), leaving too little for B's 30-hour campaign.
	itemB := item("B", 10, 600, 0, 180)
	pins := []PinnedCampaign{{ItemID: "it_A", MachineID: "mc_1", WeekIndex: 0, Units: 360}}
	got := Level([]LevellingItem{item("A", 10, 600, 900, 600), itemB}, []Machine{{ID: "mc_1", Name: "1"}}, s, pins)

	for _, c := range got.Campaigns {
		if c.SKU == "B" && c.WeekIndex == 0 {
			t.Errorf("B was placed in week 0 despite the pinned campaign filling the machine: %+v", c)
		}
	}
}

// The point of the greige buffer: a family can be flush with finished goods — so its
// echelon reads well above the reorder point — while the physical greige store it needs
// to build the short colourways is empty. With the buffer on, that empty store must
// still knit; with it off, the healthy echelon knits nothing, which is the pooled
// behaviour the parity gate preserves.
func TestLevel_GreigeBufferKnitsWhenStoreDryDespiteHealthyEchelon(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1

	// Echelon is far above its reorder point (stock is held downstream as finished goods),
	// but the greige store is empty and its safety stock is 200.
	dryGreige := func() LevellingItem {
		return LevellingItem{
			Policy: ItemPolicy{
				ItemID:             "it_A",
				SKU:                "A",
				WeeklyDemand:       100,
				AnnualDemand:       5200,
				SecondsPerUnit:     10,
				EOQUnits:           600,
				ReorderPoint:       400,
				OrderUpTo:          1200,
				OnHandEchelon:      5000,
				OnHandGreige:       0,
				SafetyStockPrimary: 200,
			},
			LotUnits: 60,
		}
	}
	machines := []Machine{{ID: "mc_1", Name: "1"}}

	s.GreigeBufferEnabled = true
	on := Level([]LevellingItem{dryGreige()}, machines, s, nil)
	if len(on.Campaigns) != 1 {
		t.Fatalf("buffer on: a dry greige store must knit despite a healthy echelon; campaigns = %d, want 1", len(on.Campaigns))
	}
	// 0 on hand + 600 built - 100 demand = 500 greige at week end.
	if got := on.ProjectedGreigeOnHand["it_A"][0]; got != 500 {
		t.Errorf("projected greige on hand = %v, want 500 (built EOQ minus one week of pull)", got)
	}

	s.GreigeBufferEnabled = false
	off := Level([]LevellingItem{dryGreige()}, machines, s, nil)
	if len(off.Campaigns) != 0 {
		t.Fatalf("buffer off: a healthy echelon knits nothing; campaigns = %d, want 0", len(off.Campaigns))
	}
	// The store is still projected even when the trigger is off, so it can be shown.
	if off.ProjectedGreigeOnHand["it_A"] == nil {
		t.Errorf("projected greige on hand must be populated even with the buffer off")
	}
}

// A make-to-order item is built against its order book, not to a buffer, so it holds no
// greige safety stock and the buffer trigger must never fire for it even when its store
// is dry.
func TestLevel_MakeToOrderHoldsNoGreigeBuffer(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 1
	s.GreigeBufferEnabled = true

	mto := LevellingItem{
		Policy: ItemPolicy{
			ItemID:             "it_A",
			SKU:                "A",
			WeeklyDemand:       0,
			SecondsPerUnit:     10,
			EOQUnits:           600,
			OnHandEchelon:      500,
			OnHandGreige:       0,
			SafetyStockPrimary: 200, // ignored: a make-to-order item is not buffered
			FulfillmentPolicy:  PolicyMakeToOrder,
		},
		LotUnits:   60,
		FirmByWeek: []float64{50}, // covered by the 500 on hand, so nothing is due
	}

	got := Level([]LevellingItem{mto}, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)
	if len(got.Campaigns) != 0 {
		t.Errorf("a make-to-order item with a dry greige store must not knit a buffer: %+v", got.Campaigns)
	}
}
