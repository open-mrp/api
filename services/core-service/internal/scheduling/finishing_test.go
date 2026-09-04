package scheduling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// finishingSettings is a plain horizon with a generous supply ceiling, so a test that means to
// exercise capacity is not accidentally exercising the weeks-of-supply cap.
func finishingSettings(weeks int) Settings {
	s := DefaultSettings()
	s.HorizonWeeks = weeks
	s.MaxWeeksSupply = 52
	s.FinishLeadTimeWeeks = 1
	return s
}

// finishingItem is a make-to-stock SKU that needs building from week zero unless the test says otherwise.
func finishingItem(sku, greige string, weekly, onHand, rop float64) FinishingItem {
	return FinishingItem{
		ItemID:         "it_" + sku,
		SKU:            sku,
		GreigeItemID:   greige,
		GreigeSKU:      greige,
		WeeklyDemand:   weekly,
		OnHand:         onHand,
		ReorderPoint:   rop,
		SecondsPerUnit: 60, // one minute a unit: 60 units an hour
		LotUnits:       60,
	}
}

func TestLevelFinishing_BuildsTheShortSkuFromItsOwnGreige(t *testing.T) {
	in := FinishingInput{
		Settings:            finishingSettings(4),
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{"greige_a": 600},
		Items:               []FinishingItem{finishingItem("SCK-001", "greige_a", 60, 0, 120)},
	}

	result := LevelFinishing(in)

	require.NotEmpty(t, result.Lines, "a SKU below its reorder point with greige and hours must be built")
	assert.Equal(t, "SCK-001", result.Lines[0].SKU)
	assert.Equal(t, 0, result.Lines[0].WeekIndex)
	// The build takes greige out of the stage-one buffer, which is what makes the two plans reconcilable.
	assert.InDelta(t, result.Lines[0].Quantity, result.Lines[0].GreigeConsumed, 1e-9)
}

// The whole reason stage two exists: one greige becomes several SKUs, and something has to decide the mix.
func TestLevelFinishing_SplitsOneGreigeAcrossItsFamilyByUrgency(t *testing.T) {
	// Each SKU is short by exactly one lot, so the greige is what decides the mix rather than any one SKU's appetite.
	in := FinishingInput{
		Settings:            finishingSettings(1),
		WeeklyCapacityHours: 100,
		// Only enough greige for two lots; three SKUs want one each.
		GreigeOnHand: map[string]float64{"greige_a": 120},
		Items: []FinishingItem{
			// Least depleted relative to its reorder point.
			finishingItem("SCK-003", "greige_a", 60, 55, 60),
			// Most depleted.
			finishingItem("SCK-001", "greige_a", 60, 5, 60),
			finishingItem("SCK-002", "greige_a", 60, 30, 60),
		},
	}

	result := LevelFinishing(in)

	require.Len(t, result.Lines, 2, "only two lots of greige exist: %v", result.Lines)
	assert.Equal(t, "SCK-001", result.Lines[0].SKU, "the emptiest SKU is finished first")
	assert.Equal(t, "SCK-002", result.Lines[1].SKU)

	// The one that missed out is named, and named as a greige problem rather than a capacity one.
	assert.Equal(t, []string{"SCK-003"}, result.Diagnostics.GreigeStarvedSKUs)
	assert.Empty(t, result.Diagnostics.CapacityStarvedSKUs)
}

// Greige and hours are different problems with opposite answers — knit more, or find another shift — so they are never collapsed into one list.
func TestLevelFinishing_CapacityStarvationIsDistinctFromGreigeStarvation(t *testing.T) {
	in := FinishingInput{
		Settings: finishingSettings(1),
		// One hour: 60 units, exactly one lot.
		WeeklyCapacityHours: 1,
		GreigeOnHand:        map[string]float64{"greige_a": 100_000},
		Items: []FinishingItem{
			finishingItem("SCK-001", "greige_a", 60, 0, 120),
			finishingItem("SCK-002", "greige_a", 60, 50, 120),
		},
	}

	result := LevelFinishing(in)

	require.Len(t, result.Lines, 1)
	assert.Equal(t, "SCK-001", result.Lines[0].SKU)
	assert.Equal(t, []string{"SCK-002"}, result.Diagnostics.CapacityStarvedSKUs,
		"greige was plentiful, so the reason is hours")
	assert.Empty(t, result.Diagnostics.GreigeStarvedSKUs)
}

// What does not fit this week waits for the next one. That is what makes this a leveling rather than a one-shot allocation.
func TestLevelFinishing_OverflowIsPushedToTheNextWeek(t *testing.T) {
	in := FinishingInput{
		Settings: finishingSettings(3),
		// One lot a week.
		WeeklyCapacityHours: 1,
		GreigeOnHand:        map[string]float64{"greige_a": 100_000},
		Items: []FinishingItem{
			finishingItem("SCK-001", "greige_a", 10, 0, 200),
			finishingItem("SCK-002", "greige_a", 10, 0, 200),
		},
	}

	result := LevelFinishing(in)

	weeks := map[int]int{}
	for _, line := range result.Lines {
		weeks[line.WeekIndex]++
	}
	require.NotEmpty(t, result.Lines)
	for week, count := range weeks {
		assert.LessOrEqual(t, count, 1, "week %d holds more lots than an hour can run", week)
	}
	assert.Greater(t, len(weeks), 1, "work the first week could not hold must appear in a later one")

	// And nothing was quietly reported as starved: it was queued, not dropped.
	assert.Empty(t, result.Diagnostics.CapacityStarvedSKUs)
}

// Greige knitted in a later week cannot be finished in an earlier one. The lag is the reason the two stages are separate plans at all.
func TestLevelFinishing_SupplyArrivesInItsOwnWeek(t *testing.T) {
	in := FinishingInput{
		Settings:            finishingSettings(3),
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{},
		Supply:              []FinishingSupply{{GreigeItemID: "greige_a", WeekIndex: 2, Quantity: 600}},
		Items:               []FinishingItem{finishingItem("SCK-001", "greige_a", 60, 0, 120)},
	}

	result := LevelFinishing(in)

	require.NotEmpty(t, result.Lines)
	for _, line := range result.Lines {
		assert.GreaterOrEqual(t, line.WeekIndex, 2,
			"nothing can be finished before its greige is knitted")
	}
	// Waiting two weeks for greige is being queued, not being starved. A SKU the horizon does build is never reported as short, or every leveled plan would read as a shortage.
	assert.Empty(t, result.Diagnostics.GreigeStarvedSKUs)
}

// A promise outranks a statistical buffer when the two contend for the same hour.
func TestLevelFinishing_MakeToOrderIsServedBeforeMakeToStock(t *testing.T) {
	stocked := finishingItem("SCK-001", "greige_a", 60, 0, 120)

	ordered := finishingItem("SCK-999", "greige_a", 0, 0, 0)
	ordered.IsMakeToOrder = true
	ordered.FirmByWeek = []float64{60}

	in := FinishingInput{
		Settings:            finishingSettings(1),
		WeeklyCapacityHours: 1, // room for exactly one lot
		GreigeOnHand:        map[string]float64{"greige_a": 100_000},
		Items:               []FinishingItem{stocked, ordered},
	}

	result := LevelFinishing(in)

	require.Len(t, result.Lines, 1)
	assert.Equal(t, "SCK-999", result.Lines[0].SKU, "the order goes first even though it sorts last by SKU")
}

// An order inside the forecast is served BY the forecast, not added to it. Summing them would build twice for the same demand.
func TestLevelFinishing_FirmDemandDoesNotStackOnTopOfForecast(t *testing.T) {
	item := finishingItem("SCK-001", "greige_a", 100, 1000, 0)
	item.FirmByWeek = []float64{40}

	in := FinishingInput{
		Settings:            finishingSettings(1),
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{"greige_a": 100_000},
		Items:               []FinishingItem{item},
	}

	result := LevelFinishing(in)

	// 1000 on hand, nothing built, and the week draws the greater of 100 forecast and 40 firm.
	assert.InDelta(t, 900, result.ProjectedOnHand["it_SCK-001"][0], 1e-9)
}

// A SKU nobody has ever finished has no measured rate, so there is no way to know what it costs the department. Reported, not guessed at.
func TestLevelFinishing_SkusWithoutARunRateAreReportedNotPlanned(t *testing.T) {
	item := finishingItem("SCK-001", "greige_a", 60, 0, 120)
	item.SecondsPerUnit = 0

	in := FinishingInput{
		Settings:            finishingSettings(2),
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{"greige_a": 100_000},
		Items:               []FinishingItem{item},
	}

	result := LevelFinishing(in)

	assert.Empty(t, result.Lines)
	assert.Equal(t, []string{"SCK-001"}, result.Diagnostics.ItemsWithoutRunRate)
}

// A finishing loss means a finished unit costs more than one greige unit, and the buffer has to be drawn down by what was actually taken.
func TestLevelFinishing_YieldLossConsumesMoreGreigeThanItProduces(t *testing.T) {
	item := finishingItem("SCK-001", "greige_a", 60, 0, 120)
	item.GreigePerUnit = 1.25

	in := FinishingInput{
		Settings:            finishingSettings(1),
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{"greige_a": 75},
		Items:               []FinishingItem{item},
	}

	result := LevelFinishing(in)

	require.Len(t, result.Lines, 1)
	assert.InDelta(t, 60, result.Lines[0].Quantity, 1e-9)
	assert.InDelta(t, 75, result.Lines[0].GreigeConsumed, 1e-9, "60 units at 1.25 costs the whole 75")
}

// Stage-one output the horizon never converts is a signal that the two stages disagree about demand, so it is surfaced rather than left as an unexplained pile.
func TestLevelFinishing_UnusedGreigeIsReported(t *testing.T) {
	in := FinishingInput{
		Settings:            finishingSettings(1),
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{"greige_a": 1000},
		Items:               []FinishingItem{finishingItem("SCK-001", "greige_a", 60, 0, 120)},
	}

	result := LevelFinishing(in)

	require.Len(t, result.Lines, 1)
	assert.InDelta(t, 1000-result.Lines[0].GreigeConsumed, result.Diagnostics.UnusedGreigeUnits, 1e-9)
}

// A version diff is unreadable if the mix wobbles between runs, and Go randomizes map iteration.
func TestLevelFinishing_Deterministic(t *testing.T) {
	build := func() FinishingInput {
		return FinishingInput{
			Settings:            finishingSettings(6),
			WeeklyCapacityHours: 8,
			GreigeOnHand:        map[string]float64{"greige_a": 900, "greige_b": 900},
			Supply: []FinishingSupply{
				{GreigeItemID: "greige_a", WeekIndex: 2, Quantity: 300},
				{GreigeItemID: "greige_b", WeekIndex: 3, Quantity: 300},
			},
			Items: []FinishingItem{
				finishingItem("SCK-004", "greige_b", 45, 20, 150),
				finishingItem("SCK-001", "greige_a", 60, 0, 120),
				finishingItem("SCK-003", "greige_a", 30, 90, 200),
				finishingItem("SCK-002", "greige_b", 75, 10, 180),
			},
		}
	}

	first := LevelFinishing(build())
	for range 25 {
		assert.Equal(t, first.Lines, LevelFinishing(build()).Lines)
	}
}

// The weeks-of-supply ceiling is what stops a slow mover with a large statistical buffer being finished into months of stock.
func TestLevelFinishing_MaxWeeksSupplyCapsTheTrigger(t *testing.T) {
	settings := finishingSettings(1)
	settings.MaxWeeksSupply = 2

	// A reorder point of 600 against 10 units a week is 60 weeks of stock; the ceiling is 20.
	item := finishingItem("SCK-001", "greige_a", 10, 30, 600)

	in := FinishingInput{
		Settings:            settings,
		WeeklyCapacityHours: 100,
		GreigeOnHand:        map[string]float64{"greige_a": 100_000},
		Items:               []FinishingItem{item},
	}

	result := LevelFinishing(in)

	assert.Empty(t, result.Lines, "30 on hand already clears the two-week ceiling of 20")
}
