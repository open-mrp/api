package scheduling

import (
	"math"
	"testing"
)

func firmFor(reqs ...FirmRequirement) FirmSchedule {
	out := FirmSchedule{ByItemWeek: map[string][]float64{}, Requirements: reqs}
	for _, r := range reqs {
		if out.ByItemWeek[r.ItemID] == nil {
			out.ByItemWeek[r.ItemID] = make([]float64, 13)
		}
		if r.DueWeek >= 0 && r.DueWeek < 13 {
			out.ByItemWeek[r.ItemID][r.DueWeek] += r.Units
		}
	}
	return out
}

func TestAllocate_EarliestPromiseIsServedFirst(t *testing.T) {
	t.Parallel()

	got := AllocateCampaignsToOrders(
		[]Campaign{{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 0, Units: 100}},
		firmFor(
			FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_late", SalesOrderNumber: "1002", DueWeek: 5, Units: 100},
			FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_early", SalesOrderNumber: "1001", DueWeek: 1, Units: 100},
		),
		nil,
	)

	if len(got.Allocations) != 1 {
		t.Fatalf("got %d allocations, want 1", len(got.Allocations))
	}
	if got.Allocations[0].SalesOrderID != "so_early" {
		t.Fatalf("the 100 units went to %s; the earlier promise must be served first", got.Allocations[0].SalesOrderID)
	}
	if len(got.Uncovered) != 1 || got.Uncovered[0].SalesOrderID != "so_late" {
		t.Fatalf("the later order should be reported uncovered, got %+v", got.Uncovered)
	}
}

// A campaign that lands after an order was due cannot serve it. Allowing it would make the plan report itself achievable while the floor misses the date.
func TestAllocate_SupplyCannotServeAnEarlierPromise(t *testing.T) {
	t.Parallel()

	got := AllocateCampaignsToOrders(
		[]Campaign{{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 6, Units: 500}},
		firmFor(FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_1", SalesOrderNumber: "1001", DueWeek: 2, Units: 100}),
		nil,
	)

	if len(got.Allocations) != 0 {
		t.Fatalf("a week-6 campaign must not be allocated to a week-2 promise, got %+v", got.Allocations)
	}
	if len(got.Uncovered) != 1 || got.Uncovered[0].ShortUnits != 100 {
		t.Fatalf("the promise should be fully uncovered, got %+v", got.Uncovered)
	}
}

// Stock already on the floor covers the earliest promises and carries no campaign to point at.
func TestAllocate_OnHandCoversBeforeAnyCampaign(t *testing.T) {
	t.Parallel()

	got := AllocateCampaignsToOrders(
		[]Campaign{{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 0, Units: 100}},
		firmFor(FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_1", SalesOrderNumber: "1001", DueWeek: 3, Units: 60}),
		map[string]float64{"itm_a": 60},
	)

	if len(got.Allocations) != 0 {
		t.Fatalf("stock on hand covers this, so no campaign should be earmarked: %+v", got.Allocations)
	}
	if len(got.Uncovered) != 0 {
		t.Fatalf("a covered order must not be reported at risk: %+v", got.Uncovered)
	}
}

// One campaign serves several orders and one order draws on several campaigns; that is why the link is a table rather than a column.
func TestAllocate_ManyToMany(t *testing.T) {
	t.Parallel()

	got := AllocateCampaignsToOrders(
		[]Campaign{
			{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 0, Units: 100},
			{ItemID: "itm_a", MachineID: "mc_2", WeekIndex: 1, Units: 100},
		},
		firmFor(
			FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_1", SalesOrderNumber: "1001", DueWeek: 2, Units: 40},
			FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_2", SalesOrderNumber: "1002", DueWeek: 2, Units: 150},
		),
		nil,
	)

	if len(got.Uncovered) != 0 {
		t.Fatalf("200 units of supply covers 190 of demand: %+v", got.Uncovered)
	}

	byOrder := map[string]float64{}
	campaignsPerOrder := map[string]int{}
	for _, a := range got.Allocations {
		byOrder[a.SalesOrderID] += a.Units
		campaignsPerOrder[a.SalesOrderID]++
	}
	if math.Abs(byOrder["so_1"]-40) > 1e-9 || math.Abs(byOrder["so_2"]-150) > 1e-9 {
		t.Fatalf("allocated quantities wrong: %+v", byOrder)
	}
	if campaignsPerOrder["so_2"] != 2 {
		t.Fatalf("the 150-unit order should draw on both campaigns, got %d", campaignsPerOrder["so_2"])
	}
}

// A partly-built order is not a total miss and must not read as one.
func TestAllocate_PartialCoverageReportsBothSides(t *testing.T) {
	t.Parallel()

	got := AllocateCampaignsToOrders(
		[]Campaign{{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 0, Units: 30}},
		firmFor(FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_1", SalesOrderNumber: "1001", DueWeek: 2, Units: 100}),
		nil,
	)

	if len(got.Uncovered) != 1 {
		t.Fatalf("got %d uncovered, want 1", len(got.Uncovered))
	}
	u := got.Uncovered[0]
	if math.Abs(u.ShortUnits-70) > 1e-9 || math.Abs(u.CoveredUnits-30) > 1e-9 {
		t.Fatalf("got short=%v covered=%v, want 70/30", u.ShortUnits, u.CoveredUnits)
	}
}

func TestAllocate_Deterministic(t *testing.T) {
	t.Parallel()

	campaigns := []Campaign{
		{ItemID: "itm_b", MachineID: "mc_2", WeekIndex: 1, Units: 50},
		{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 0, Units: 80},
		{ItemID: "itm_a", MachineID: "mc_3", WeekIndex: 0, Units: 80},
	}
	firm := firmFor(
		FirmRequirement{ItemID: "itm_b", SalesOrderID: "so_3", SalesOrderNumber: "1003", DueWeek: 2, Units: 50},
		FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_1", SalesOrderNumber: "1001", DueWeek: 2, Units: 100},
		FirmRequirement{ItemID: "itm_a", SalesOrderID: "so_2", SalesOrderNumber: "1002", DueWeek: 2, Units: 40},
	)

	first := AllocateCampaignsToOrders(campaigns, firm, nil)
	for range 50 {
		got := AllocateCampaignsToOrders(campaigns, firm, nil)
		if len(got.Allocations) != len(first.Allocations) {
			t.Fatal("allocation count changed between runs")
		}
		for i := range got.Allocations {
			if got.Allocations[i] != first.Allocations[i] {
				t.Fatalf("allocation %d differs between runs: %+v vs %+v", i, got.Allocations[i], first.Allocations[i])
			}
		}
	}
}

// Capable-to-promise: the earliest week the plan could supply a quantity nobody is already owed.
func TestEarliestPromiseWeek(t *testing.T) {
	t.Parallel()

	campaigns := []Campaign{
		{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 1, Units: 100},
		{ItemID: "itm_a", MachineID: "mc_1", WeekIndex: 4, Units: 100},
	}

	// Nothing committed: 100 units are there in week 1.
	week, ok := EarliestPromiseWeek("itm_a", 100, campaigns, FirmSchedule{ByItemWeek: map[string][]float64{}}, nil, 13)
	if !ok || week != 1 {
		t.Fatalf("got week %d ok=%v, want week 1", week, ok)
	}

	// 150 units needs both campaigns, so week 4.
	week, ok = EarliestPromiseWeek("itm_a", 150, campaigns, FirmSchedule{ByItemWeek: map[string][]float64{}}, nil, 13)
	if !ok || week != 4 {
		t.Fatalf("got week %d ok=%v, want week 4", week, ok)
	}

	// With week 1's output already owed to somebody, the same 100 units are not promisable until week 4.
	committed := FirmSchedule{ByItemWeek: map[string][]float64{"itm_a": make([]float64, 13)}}
	committed.ByItemWeek["itm_a"][1] = 100
	week, ok = EarliestPromiseWeek("itm_a", 100, campaigns, committed, nil, 13)
	if !ok || week != 4 {
		t.Fatalf("got week %d ok=%v, want week 4 — supply somebody is already owed is not promisable", week, ok)
	}

	// Beyond what the horizon can supply, the honest answer is that it cannot say.
	if _, ok := EarliestPromiseWeek("itm_a", 10000, campaigns, FirmSchedule{ByItemWeek: map[string][]float64{}}, nil, 13); ok {
		t.Fatal("a quantity the horizon cannot supply must not be given a date")
	}
}

// Stock on hand can promise immediately.
func TestEarliestPromiseWeek_OnHandPromisesNow(t *testing.T) {
	t.Parallel()

	week, ok := EarliestPromiseWeek("itm_a", 50, nil, FirmSchedule{ByItemWeek: map[string][]float64{}}, map[string]float64{"itm_a": 500}, 13)
	if !ok || week != 0 {
		t.Fatalf("got week %d ok=%v, want week 0", week, ok)
	}
}
