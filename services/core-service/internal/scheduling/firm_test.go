package scheduling

import (
	"testing"
	"time"
)

func firmTestSettings() Settings {
	s := DefaultSettings()
	s.HorizonWeeks = 13
	s.FinishLeadTimeWeeks = 6
	return s
}

var firmHorizonStart = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

func shipBy(daysFromStart int) *time.Time {
	d := firmHorizonStart.AddDate(0, 0, daysFromStart)
	return &d
}

// The constraint stage has to finish a finishing lead time before the ship date, so a promise 10 weeks out is a campaign due in week 4.
func TestBuildFirmSchedule_DatesBackwardsFromThePromise(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	got := BuildFirmSchedule([]OpenOrderLine{{
		SalesOrderID:     "so_1",
		SalesOrderNumber: "1001",
		FinishedItemID:   "itm_fg",
		ConstraintItemID: "itm_greige",
		Units:            500,
		ShipByDate:       shipBy(70), // week 10
	}}, firmHorizonStart, s)

	if len(got.Requirements) != 1 {
		t.Fatalf("got %d requirements, want 1", len(got.Requirements))
	}
	req := got.Requirements[0]
	if req.ShipByWeek != 10 {
		t.Fatalf("ship-by week = %d, want 10", req.ShipByWeek)
	}
	if req.DueWeek != 4 {
		t.Fatalf("due week = %d, want 4 (10 - 6 weeks of finishing)", req.DueWeek)
	}
	if req.IsPastDue {
		t.Fatal("a promise 10 weeks out is not past due")
	}
	if got.RequirementForWeek("itm_greige", 4) != 500 {
		t.Fatalf("week 4 requirement = %v, want 500", got.RequirementForWeek("itm_greige", 4))
	}
}

// An order whose constraint week lands before the horizon is clamped to week 0 and flagged. Dropping it would hide work that is already late; moving it forward silently would make the plan look achievable.
func TestBuildFirmSchedule_PastDueClampsToWeekZeroAndIsFlagged(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	got := BuildFirmSchedule([]OpenOrderLine{{
		SalesOrderID:     "so_late",
		ConstraintItemID: "itm_greige",
		Units:            100,
		ShipByDate:       shipBy(7), // week 1, needs the constraint 5 weeks before the horizon
	}}, firmHorizonStart, s)

	if len(got.Requirements) != 1 {
		t.Fatalf("got %d requirements, want 1", len(got.Requirements))
	}
	req := got.Requirements[0]
	if req.DueWeek != 0 {
		t.Fatalf("due week = %d, want 0", req.DueWeek)
	}
	if !req.IsPastDue {
		t.Fatal("expected the requirement to be flagged past due")
	}
	if got.PastDueCount != 1 {
		t.Fatalf("past-due count = %d, want 1", got.PastDueCount)
	}
	if got.RequirementForWeek("itm_greige", 0) != 100 {
		t.Fatal("a past-due requirement still has to be built, so its units belong in week 0")
	}
}

// A commitment past the horizon belongs to a future plan; counting it now would build stock this plan has no reason to hold.
func TestBuildFirmSchedule_BeyondHorizonIsExcluded(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	got := BuildFirmSchedule([]OpenOrderLine{{
		SalesOrderID:     "so_far",
		ConstraintItemID: "itm_greige",
		Units:            100,
		ShipByDate:       shipBy(7 * 30), // week 30, due week 24, horizon is 13
	}}, firmHorizonStart, s)

	if len(got.Requirements) != 0 {
		t.Fatalf("got %d requirements, want 0", len(got.Requirements))
	}
	if got.TotalUnits != 0 {
		t.Fatalf("total units = %v, want 0", got.TotalUnits)
	}
}

// An order issued before commitments were tracked is still work owed. It is dated at the front and reported as undated so a planner can tell it from a real promise.
func TestBuildFirmSchedule_UndatedOrderIsOwedNowAndSaysSo(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	got := BuildFirmSchedule([]OpenOrderLine{{
		SalesOrderID:     "so_undated",
		ConstraintItemID: "itm_greige",
		Units:            250,
	}}, firmHorizonStart, s)

	if len(got.Requirements) != 1 {
		t.Fatalf("got %d requirements, want 1", len(got.Requirements))
	}
	req := got.Requirements[0]
	if !req.IsUndated || !req.IsPastDue || req.DueWeek != 0 {
		t.Fatalf("got undated=%v pastDue=%v due=%d, want true/true/0", req.IsUndated, req.IsPastDue, req.DueWeek)
	}
	if got.UndatedCount != 1 {
		t.Fatalf("undated count = %d, want 1", got.UndatedCount)
	}
}

// Several orders landing in the same week for the same item are one requirement on the machine, so they accumulate rather than overwrite.
func TestBuildFirmSchedule_AccumulatesWithinAWeek(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	got := BuildFirmSchedule([]OpenOrderLine{
		{SalesOrderID: "so_1", ConstraintItemID: "itm_greige", Units: 100, ShipByDate: shipBy(70)},
		{SalesOrderID: "so_2", ConstraintItemID: "itm_greige", Units: 250, ShipByDate: shipBy(72)},
	}, firmHorizonStart, s)

	if got.RequirementForWeek("itm_greige", 4) != 350 {
		t.Fatalf("week 4 = %v, want 350", got.RequirementForWeek("itm_greige", 4))
	}
	if got.TotalUnits != 350 {
		t.Fatalf("total = %v, want 350", got.TotalUnits)
	}
}

// A line nothing in the plan produces cannot become a campaign, and inventing a constraint item for it would put work on a machine for a product it does not make.
func TestBuildFirmSchedule_LinesWithNoConstraintItemAreDropped(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	got := BuildFirmSchedule([]OpenOrderLine{
		{SalesOrderID: "so_1", ConstraintItemID: "", Units: 100, ShipByDate: shipBy(70)},
		{SalesOrderID: "so_2", ConstraintItemID: "itm_greige", Units: 0, ShipByDate: shipBy(70)},
	}, firmHorizonStart, s)

	if len(got.Requirements) != 0 {
		t.Fatalf("got %d requirements, want 0", len(got.Requirements))
	}
}

// A ship-by date inside the current week is week 0, and one three days before the horizon must be week -1 rather than rounding toward zero into week 0 alongside it.
func TestWeeksBetween_FloorsNegativesAwayFromZero(t *testing.T) {
	t.Parallel()

	cases := []struct {
		days int
		want int
	}{
		{0, 0}, {3, 0}, {6, 0}, {7, 1}, {13, 1}, {14, 2},
		{-1, -1}, {-3, -1}, {-7, -1}, {-8, -2},
	}
	for _, c := range cases {
		got := weeksBetween(firmHorizonStart, firmHorizonStart.AddDate(0, 0, c.days))
		if got != c.want {
			t.Fatalf("%+d days = week %d, want %d", c.days, got, c.want)
		}
	}
}

func TestBuildFirmSchedule_Deterministic(t *testing.T) {
	t.Parallel()

	s := firmTestSettings()
	lines := []OpenOrderLine{
		{SalesOrderID: "so_c", SalesOrderNumber: "1003", ConstraintItemID: "itm_b", Units: 10, ShipByDate: shipBy(70)},
		{SalesOrderID: "so_a", SalesOrderNumber: "1001", ConstraintItemID: "itm_a", Units: 20, ShipByDate: shipBy(70)},
		{SalesOrderID: "so_b", SalesOrderNumber: "1002", ConstraintItemID: "itm_a", Units: 30, ShipByDate: shipBy(49)},
	}

	first := BuildFirmSchedule(lines, firmHorizonStart, s)
	for range 50 {
		got := BuildFirmSchedule(lines, firmHorizonStart, s)
		if len(got.Requirements) != len(first.Requirements) {
			t.Fatal("requirement count changed between runs")
		}
		for i := range got.Requirements {
			if got.Requirements[i] != first.Requirements[i] {
				t.Fatalf("requirement %d differs between runs: %+v vs %+v", i, got.Requirements[i], first.Requirements[i])
			}
		}
	}
}

// Forecast consumption: an order inside the forecast is served BY it, not added to it.
func TestLevellingItem_DemandForWeek_TakesTheGreaterOfForecastAndOrders(t *testing.T) {
	t.Parallel()

	item := LevellingItem{
		Policy:     ItemPolicy{WeeklyDemand: 100},
		FirmByWeek: []float64{0, 50, 100, 250},
	}

	cases := []struct {
		week int
		want float64
	}{
		{0, 100},  // no orders: the forecast stands
		{1, 100},  // orders below the forecast are absorbed by it
		{2, 100},  // orders exactly at the forecast are the same number
		{3, 250},  // orders above the forecast replace it
		{9, 100},  // past the end of the order book: the forecast stands
		{-1, 100}, // defensive
	}
	for _, c := range cases {
		if got := item.demandForWeek(c.week); got != c.want {
			t.Fatalf("week %d = %v, want %v", c.week, got, c.want)
		}
	}
}

// The guarantee the whole phase rests on: with no order book, every item consumes exactly its forecast, which is what the plan did before firm demand existed.
func TestLevellingItem_DemandForWeek_NoOrderBookIsTheForecast(t *testing.T) {
	t.Parallel()

	item := LevellingItem{Policy: ItemPolicy{WeeklyDemand: 137.5}}
	for week := range 13 {
		if got := item.demandForWeek(week); got != 137.5 {
			t.Fatalf("week %d = %v, want the plain forecast 137.5", week, got)
		}
	}
}
