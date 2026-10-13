package scheduling

import (
	"math"
	"reflect"
	"testing"
	"time"
)

// sockFamilyCommitted is one greige sock that is dyed black or blue downstream. Black has 300 pairs
// dyed and waiting (WIP) and 300 packed; blue has nothing anywhere.
func sockFamilyCommitted() []CommittedStock {
	return []CommittedStock{
		{ItemID: "it_black_dyed", OnHand: 300, FinishedItemIDs: []string{"it_black"}},
		{ItemID: "it_black", OnHand: 300, FinishedItemIDs: []string{"it_black"}},
	}
}

// sockSettings shortens the pipeline to one week so a week of family demand is the whole reorder
// point: the pooled echelon of 601 then sits just above it and, read as fungible, knits nothing.
func sockSettings() Settings {
	s := DefaultSettings()
	s.HorizonWeeks = 4
	s.DefaultConstraintLeadTimeWeeks = 0.5
	s.FinishLeadTimeWeeks = 0.5
	return s
}

func sockSolverInput(committed map[string][]CommittedStock) SolverInput {
	// 100 pairs a week of black and 500 of blue, flat, so the forecast is exactly that split and the safety stock is zero.
	black := series(2026, time.May, 12, 100*52.0/12)
	blue := series(2026, time.May, 12, 500*52.0/12)
	pooled := series(2026, time.May, 12, 600*52.0/12)

	return SolverInput{
		AccountID:    "ac_1",
		PlanningAsOf: time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC),
		Settings:     sockSettings(),
		Machines:     []Machine{{ID: "mc_1", Name: "1"}},

		ConfiguredEligibility:    map[string]map[string]bool{"it_greige": {"mc_1": true}},
		ConfiguredItemSKU:        map[string]string{"it_greige": "SOCK-GREIGE"},
		ConfiguredSecondsPerUnit: map[string]float64{"it_greige": 10},

		MonthlyByItem: map[string][]MonthlyDemand{"it_greige": pooled},
		DownstreamByItem: map[string][]FinishedGood{"it_greige": {
			{ItemID: "it_black", SKU: "SOCK-BLACK", Monthly: black, OnHand: 300},
			{ItemID: "it_blue", SKU: "SOCK-BLUE", Monthly: blue, OnHand: 0},
		}},
		// One loose greige pair keeps the pooled control clear of float noise around the 600-pair reorder point.
		OnHandByItem:         map[string]float64{"it_greige": 601},
		GreigeOnHandByItem:   map[string]float64{"it_greige": 1},
		CommittedStockByItem: committed,
		DemandBasisCode:      DemandBasisTrailing12,
	}
}

func campaignUnitsInWeek(campaigns []Campaign, itemID string, week int) float64 {
	var total float64
	for _, c := range campaigns {
		if c.ItemID == itemID && c.WeekIndex == week {
			total += c.Units
		}
	}
	return total
}

// The bug this exists for: black WIP and black finished goods were netted against the whole family's
// demand, so 600 black pairs read as a week of cover for 500 pairs of blue nobody has. Blue needs
// knitting in week 0.
func TestSolve_CommittedStockDoesNotCoverOtherColorways(t *testing.T) {
	t.Parallel()

	// Without the branch information the family is one pool, and 601 pairs clears a 600-pair reorder point. This is the behavior being fixed, kept as the control.
	pooled := Solve(sockSolverInput(nil))
	if got := campaignUnitsInWeek(pooled.Campaigns, "it_greige", 0); got != 0 {
		t.Fatalf("control: a fungible echelon at its reorder point knits nothing in week 0, got %v", got)
	}

	got := Solve(sockSolverInput(map[string][]CommittedStock{"it_greige": sockFamilyCommitted()}))
	if units := campaignUnitsInWeek(got.Campaigns, "it_greige", 0); units < 500 {
		t.Fatalf("week 0 knits %v pairs; blue has no stock anywhere and needs at least 500", units)
	}

	// The echelon is still reported whole: the black stock exists, it just cannot be blue.
	if got.Policies[0].OnHandEchelon != 601 {
		t.Errorf("OnHandEchelon = %v, want 601 (committed stock still counts in the total)", got.Policies[0].OnHandEchelon)
	}
}

// The same family with stock in every colorway in proportion to its demand is genuinely covered, so
// splitting out committed stock must not change the plan. The fix targets the wrong-mix case only.
func TestSolve_BalancedCommittedStockLeavesThePlanUnchanged(t *testing.T) {
	t.Parallel()

	balanced := func(committed map[string][]CommittedStock) SolverInput {
		in := sockSolverInput(committed)
		in.OnHandByItem["it_greige"] = 1201
		return in
	}
	committed := map[string][]CommittedStock{"it_greige": {
		{ItemID: "it_black", OnHand: 200, FinishedItemIDs: []string{"it_black"}},
		{ItemID: "it_blue", OnHand: 1000, FinishedItemIDs: []string{"it_blue"}},
	}}

	want := Solve(balanced(nil))
	got := Solve(balanced(committed))

	if !reflect.DeepEqual(want.Campaigns, got.Campaigns) {
		t.Errorf("balanced committed stock changed the campaigns:\nwithout: %+v\nwith:    %+v", want.Campaigns, got.Campaigns)
	}
	if !reflect.DeepEqual(want.ProjectedOnHand, got.ProjectedOnHand) {
		t.Errorf("balanced committed stock changed the projection:\nwithout: %v\nwith:    %v", want.ProjectedOnHand, got.ProjectedOnHand)
	}
}

// An order for blue cannot be shipped out of black stock, so the at-risk report has to say the blue
// order is short until a campaign covers it, while the black order is covered from stock.
func TestAllocate_CommittedStockOnlyServesItsOwnFinishedGoods(t *testing.T) {
	t.Parallel()

	firm := firmFor(
		FirmRequirement{ItemID: "it_greige", FinishedItemID: "it_black", SalesOrderID: "so_black", SalesOrderNumber: "1001", DueWeek: 0, Units: 100},
		FirmRequirement{ItemID: "it_greige", FinishedItemID: "it_blue", SalesOrderID: "so_blue", SalesOrderNumber: "1002", DueWeek: 0, Units: 500},
	)
	onHand := map[string]float64{"it_greige": 600}
	committed := map[string][]CommittedStock{"it_greige": sockFamilyCommitted()}

	got := AllocateCampaignsToOrders(nil, firm, onHand, committed)

	if len(got.Uncovered) != 1 {
		t.Fatalf("uncovered = %+v, want only the blue order", got.Uncovered)
	}
	if u := got.Uncovered[0]; u.SalesOrderID != "so_blue" || u.ShortUnits != 500 {
		t.Errorf("uncovered = %+v, want so_blue short 500", u)
	}

	// Control: read as one pool, the same stock covers both orders, which is the false all-clear.
	if pooled := AllocateCampaignsToOrders(nil, firm, onHand, nil); len(pooled.Uncovered) != 0 {
		t.Errorf("control: a fungible echelon covers both orders, got uncovered %+v", pooled.Uncovered)
	}
}

// Uncommitted stock (greige, or a stage before the branch) still serves any colorway, and is only
// reached after the committed stock that can serve the order is used.
func TestAllocate_UncommittedStockServesAnyFinishedGood(t *testing.T) {
	t.Parallel()

	firm := firmFor(
		FirmRequirement{ItemID: "it_greige", FinishedItemID: "it_black", SalesOrderID: "so_black", SalesOrderNumber: "1001", DueWeek: 0, Units: 100},
		FirmRequirement{ItemID: "it_greige", FinishedItemID: "it_blue", SalesOrderID: "so_blue", SalesOrderNumber: "1002", DueWeek: 0, Units: 500},
	)
	// 600 black plus 500 greige: black ships from black, and the greige is left whole for blue.
	onHand := map[string]float64{"it_greige": 1100}
	committed := map[string][]CommittedStock{"it_greige": sockFamilyCommitted()}

	got := AllocateCampaignsToOrders(nil, firm, onHand, committed)
	if len(got.Uncovered) != 0 {
		t.Errorf("uncovered = %+v, want none", got.Uncovered)
	}
}

// Make-to-order: the order book is the trigger, and committed stock can only net against orders for
// its own finished goods. 100 black is covered by black stock; 500 blue has to be knit.
func TestLevel_MakeToOrderBuildsTheColorwayCommittedStockCannotCover(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 2

	mto := func(committed []CommittedStock, demand []FinishedDemand) LevelingItem {
		return LevelingItem{
			Policy: ItemPolicy{
				ItemID:                  "it_greige",
				SKU:                     "SOCK-GREIGE",
				SecondsPerUnit:          10,
				EOQUnits:                600,
				OnHandEchelon:           600,
				FulfillmentPolicy:       PolicyMakeToOrder,
				ConstraintLeadTimeWeeks: 0.5,
				FinishLeadTimeWeeks:     0.5,
			},
			LotUnits:       60,
			FirmByWeek:     []float64{600, 0},
			Committed:      committed,
			FinishedDemand: demand,
		}
	}
	demand := []FinishedDemand{
		{ItemID: "it_black", FirmByWeek: []float64{100, 0}},
		{ItemID: "it_blue", FirmByWeek: []float64{500, 0}},
	}
	machines := []Machine{{ID: "mc_1", Name: "1"}}

	if control := Level([]LevelingItem{mto(nil, nil)}, machines, s, nil); len(control.Campaigns) != 0 {
		t.Fatalf("control: 600 fungible units cover a 600-unit order book, got %+v", control.Campaigns)
	}

	got := Level([]LevelingItem{mto(sockFamilyCommitted(), demand)}, machines, s, nil)
	if len(got.Campaigns) != 1 || got.Campaigns[0].WeekIndex != 0 {
		t.Fatalf("campaigns = %+v, want one in week 0", got.Campaigns)
	}
	// 500 short, rounded up to whole 60-unit lots.
	if got.Campaigns[0].Units != 540 {
		t.Errorf("campaign units = %v, want 540 (500 blue rounded up to lots)", got.Campaigns[0].Units)
	}
	// 600 black + 540 knit - 600 shipped: the 500 surplus black is still on hand, plus the 40 lot overrun.
	if pos := got.ProjectedOnHand["it_greige"][0]; math.Abs(pos-540) > 1e-9 {
		t.Errorf("projected echelon after week 0 = %v, want 540", pos)
	}
}

// Committed stock is drawn down by its own SKU's demand only, so a surplus colorway keeps counting
// toward its own future weeks instead of vanishing into the family draw.
func TestLevel_CommittedStockIsConsumedByItsOwnDemand(t *testing.T) {
	t.Parallel()

	s := testSettings()
	s.HorizonWeeks = 3

	li := LevelingItem{
		Policy: ItemPolicy{
			ItemID:         "it_greige",
			SKU:            "SOCK-GREIGE",
			WeeklyDemand:   600,
			SecondsPerUnit: 10,
			EOQUnits:       600,
			ReorderPoint:   0, // never triggers: this test is about the projection
			OrderUpTo:      0,
			OnHandEchelon:  1600,
		},
		LotUnits:  60,
		Committed: sockFamilyCommitted(),
		FinishedDemand: []FinishedDemand{
			{ItemID: "it_black", WeeklyForecast: 100},
			{ItemID: "it_blue", WeeklyForecast: 500},
		},
	}

	got := Level([]LevelingItem{li}, []Machine{{ID: "mc_1", Name: "1"}}, s, nil)
	if len(got.Campaigns) != 0 {
		t.Fatalf("campaigns = %+v, want none", got.Campaigns)
	}
	// The echelon still falls by the whole family's 600 a week: black from black stock, blue from the 1,000 uncommitted.
	want := []float64{1000, 400, -200}
	for week, w := range want {
		if pos := got.ProjectedOnHand["it_greige"][week]; math.Abs(pos-w) > 1e-9 {
			t.Errorf("week %d projected echelon = %v, want %v", week, pos, w)
		}
	}
}

func TestNewCommittedPools_NarrowestFirstAndMergedBySet(t *testing.T) {
	t.Parallel()

	got := newCommittedPools([]CommittedStock{
		{ItemID: "dyed_black", OnHand: 50, FinishedItemIDs: []string{"black_l", "black_m"}},
		{ItemID: "black_m", OnHand: 10, FinishedItemIDs: []string{"black_m"}},
		{ItemID: "dyed_black_2", OnHand: 5, FinishedItemIDs: []string{"black_m", "black_l"}},
	})

	want := []committedPool{
		{finished: []string{"black_m"}, onHand: 10},
		{finished: []string{"black_l", "black_m"}, onHand: 55},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pools = %+v, want %+v", got, want)
	}
}

// Shared WIP that can still become either size is used only after the size-specific stock, so it is
// left for the size that has nothing of its own.
func TestCoverable_UsesNarrowStockBeforeSharedStock(t *testing.T) {
	t.Parallel()

	pools := newCommittedPools([]CommittedStock{
		{ItemID: "black_m", OnHand: 100, FinishedItemIDs: []string{"black_m"}},
		{ItemID: "dyed_black", OnHand: 100, FinishedItemIDs: []string{"black_l", "black_m"}},
	})

	got := coverable(pools, map[string]float64{"black_m": 100, "black_l": 100})
	if got != 200 {
		t.Errorf("coverable = %v, want 200 (M from M stock, L from the shared dyed WIP)", got)
	}
}

func TestBuildFinishedDemand_SplitsTheForecastAndOrderBookBySKU(t *testing.T) {
	t.Parallel()

	policy := ItemPolicy{WeeklyDemand: 1200} // an override doubled the pooled 600 a week
	pooled := series(2026, time.May, 12, 600)
	downstream := []FinishedGood{
		{ItemID: "it_black", Monthly: series(2026, time.May, 12, 100)},
		{ItemID: "it_blue", Monthly: series(2026, time.May, 12, 500)},
	}
	firm := firmFor(
		FirmRequirement{ItemID: "it_greige", FinishedItemID: "it_black", DueWeek: 1, Units: 40},
		FirmRequirement{ItemID: "it_other", FinishedItemID: "it_black", DueWeek: 1, Units: 999},
	)

	got := buildFinishedDemand("it_greige", policy, sockFamilyCommitted(), pooled, downstream, firm, 3)

	want := []FinishedDemand{{ItemID: "it_black", WeeklyForecast: 200, FirmByWeek: []float64{0, 40, 0}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("finished demand = %+v, want %+v (only SKUs committed stock can become, at their share of the pooled rate)", got, want)
	}
}
