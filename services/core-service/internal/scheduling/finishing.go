package scheduling

import (
	"maps"
	"math"
	"sort"
)

// The second stage of the plant.
//
// Stage one is the constraint — the knitting room — and it is planned machine by machine, because
// which machine runs which yarn is the decision that sets the pace of everything else. Stage two is
// the rest of the factory, and it is planned as one pool: dyeing, finishing, boarding and packing
// are steps a unit passes through rather than choices between, so modeling each machine would put a
// precision on the plan that the plan does not have.
//
// What stage two decides is the thing stage one deliberately does not: **which finished goods to
// make from the knitted parts**. A greige item becomes several SKUs — the same sock body in four
// colorways — and the knit plan pools their demand into one campaign precisely so the buffer can
// sit at the undifferentiated stage. That pooling is what makes the buffer cheap, and it is also why
// the mix has to be decided again, later, against each finished SKU's own position.

// FinishingItem is one finished SKU the second stage can make.
type FinishingItem struct {
	ItemID string
	SKU    string
	// GreigeItemID names the stage-one item this is made from. It is what ties the two schedules together: a finished SKU cannot be built in a week the greige it comes from has not arrived in.
	GreigeItemID  string
	GreigeSKU     string
	ProductLineID string

	// WeeklyDemand and OnHand are this SKU's own, not the family's. The pooled figures behind the knit plan cannot answer "is this colorway short", which is the whole question stage two exists to answer.
	WeeklyDemand float64
	OnHand       float64
	ReorderPoint float64
	SafetyStock  float64

	// SecondsPerUnit is the measured finishing run rate. Without one the SKU cannot be leveled, because there is no way to know what an hour of the department buys.
	SecondsPerUnit float64
	LotUnits       float64
	LotUnitID      string
	UnitID         string

	// GreigePerUnit is how much stage-one output one finished unit consumes. One unless a yield ratio says otherwise; a finishing loss makes it greater than one.
	GreigePerUnit float64

	// ProductionStepID and DepartmentID are where the SKU is made. The sweep does not decide with them — the second stage is one pool — but the plan line carries them so a department rollup needs no join.
	ProductionStepID string
	DepartmentID     string

	// FirmByWeek is the order book at this SKU, by the week it is owed.
	FirmByWeek    []float64
	IsMakeToOrder bool
}

// FinishingSupply is stage-one output becoming available to stage two.
//
// WeekIndex is the week the greige can be worked, not the week it was knitted — the caller applies
// the lag between the two, because how long greige sits before finishing can start is a property of
// the plant rather than of the plan.
type FinishingSupply struct {
	GreigeItemID string
	WeekIndex    int
	Quantity     float64
}

// FinishingInput is everything the second-stage sweep needs, already loaded.
type FinishingInput struct {
	Items  []FinishingItem
	Supply []FinishingSupply
	// GreigeOnHand is what is already knitted and waiting when the horizon opens. Without it the first weeks of every plan would read as starved while the greige store sat full.
	GreigeOnHand map[string]float64
	// WeeklyCapacityHours is the whole second stage's capacity in one week.
	WeeklyCapacityHours float64
	Settings            Settings
}

// FinishingLine is one finished SKU's build in one week.
type FinishingLine struct {
	ItemID       string
	SKU          string
	GreigeItemID string
	GreigeSKU    string
	WeekIndex    int

	Quantity  float64
	Lots      int
	LotUnits  float64
	LotUnitID string
	RunHours  float64
	// GreigeConsumed is what this line takes out of the stage-one buffer, which is what makes the two schedules reconcilable.
	GreigeConsumed float64
	// FirmUnits is how much of the week's draw was an order rather than a forecast, so a planner can see which lines are promises.
	FirmUnits float64

	// ProductionStepID and DepartmentID name where the work runs.
	ProductionStepID string
	DepartmentID     string

	ProjectedOnHandBefore float64
	ProjectedOnHandAfter  float64
}

// FinishingDiagnostics is the honest account of what stage two could not do.
//
// The two starvation lists are the point of the whole model. A SKU held back for want of greige is a
// stage-one problem — knit more, or knit it sooner — and a SKU held back for want of hours is a
// stage-two problem — another shift, or a different mix. Collapsing them into one "short" list would
// throw away the only thing a two-stage plan knows that a one-stage plan does not.
type FinishingDiagnostics struct {
	WeeklyCapacityHours float64   `json:"weekly_capacity_hours"`
	PlannedHoursByWeek  []float64 `json:"planned_hours_by_week"`
	// UtilisationByWeek is planned hours over capacity. Nil entries are impossible here — capacity is constant across the horizon — but a zero-capacity input yields zeroes rather than infinities.
	UtilisationByWeek []float64 `json:"utilisation_by_week"`
	// GreigeStarvedSKUs wanted building and had no greige to build from.
	GreigeStarvedSKUs []string `json:"greige_starved_skus"`
	// CapacityStarvedSKUs had greige and no hours.
	CapacityStarvedSKUs []string `json:"capacity_starved_skus"`
	// ItemsWithoutRunRate never got a measured finishing rate and cannot be leveled.
	ItemsWithoutRunRate []string `json:"items_without_run_rate"`
	// UnusedGreigeUnits is stage-one output the horizon never converts. A large number means the two stages are planned against different demand, which is worth surfacing rather than leaving as an unexplained pile of greige.
	UnusedGreigeUnits float64 `json:"unused_greige_units"`
	TotalPlannedUnits float64 `json:"total_planned_units"`
	LineCount         int     `json:"line_count"`
}

// FinishingResult is the second-stage plan.
type FinishingResult struct {
	Lines []FinishingLine
	// ProjectedOnHand[itemID][weekIndex] is the finished SKU's position at the end of that week.
	ProjectedOnHand map[string][]float64
	Diagnostics     FinishingDiagnostics
}

// greigePerUnit is what one finished unit costs the stage-one buffer, defaulting to one for a plant that has never measured a finishing yield.
func (i FinishingItem) greigePerUnit() float64 {
	if i.GreigePerUnit > 0 {
		return i.GreigePerUnit
	}
	return 1
}

// hasFirmDemand reports whether anything is on order for this SKU inside the horizon.
func (i FinishingItem) hasFirmDemand() bool {
	for _, units := range i.FirmByWeek {
		if units > 0 {
			return true
		}
	}
	return false
}

// leadTimeWeeks is how far ahead a make-to-order finished SKU has to look. Only the finishing lead time: the knit lead time sits behind the greige buffer, and charging it here would build finished goods against a wait the buffer already absorbs.
func (i FinishingItem) leadTimeWeeks(s Settings) int {
	return max(int(math.Ceil(s.FinishLeadTimeWeeks)), 1)
}

// firmRequiredThrough is the order book this SKU owes from the given week through its lead time.
func (i FinishingItem) firmRequiredThrough(week int, s Settings) float64 {
	if len(i.FirmByWeek) == 0 {
		return 0
	}
	var total float64
	for w := week; w <= week+i.leadTimeWeeks(s) && w < len(i.FirmByWeek); w++ {
		if w < 0 {
			continue
		}
		total += i.FirmByWeek[w]
	}
	return total
}

// triggerForWeek is the position below which this SKU needs building.
//
// The same rule stage one uses, for the same reason: a make-to-stock SKU triggers at the lower of
// its reorder point and its order-up-to ceiling, so a slow mover with a large statistical buffer is
// not finished into months of stock; a make-to-order SKU has no average to reduce to and recomputes
// from the dated order book each week.
func (i FinishingItem) triggerForWeek(week int, s Settings) float64 {
	if i.IsMakeToOrder {
		return i.firmRequiredThrough(week, s)
	}
	ceiling := s.MaxWeeksSupply * i.WeeklyDemand
	if ceiling <= 0 {
		return i.ReorderPoint
	}
	return math.Min(i.ReorderPoint, ceiling)
}

// demandForWeek is what this SKU consumes in one week: the greater of its forecast rate and the orders already on the book for that week.
//
// Never the sum. An order inside the forecast is served by the forecast rather than added to it, and adding them counts the same demand twice — once as history repeating and once as the order history predicted.
func (i FinishingItem) demandForWeek(week int) float64 {
	forecast := i.WeeklyDemand
	if week < 0 || week >= len(i.FirmByWeek) {
		return forecast
	}
	if firm := i.FirmByWeek[week]; firm > forecast {
		return firm
	}
	return forecast
}

// wholeLotsWithin is the largest whole-lot quantity that fits in a ceiling.
//
// Deliberately not maxLotsInCapacity, which floors at one lot. That floor is right for a whole
// machine-week, which always holds at least one lot; it is wrong here, where the ceiling is what is
// left of a shared pool after earlier SKUs have taken their share and the honest answer is often
// nothing.
func wholeLotsWithin(ceiling, lotUnits float64) float64 {
	if ceiling <= 0 {
		return 0
	}
	if lotUnits <= 0 {
		return ceiling
	}
	return math.Floor(ceiling/lotUnits) * lotUnits
}

// unitsWithinHours is how many units of a SKU fit in a number of department hours.
func unitsWithinHours(hours, secondsPerUnit float64) float64 {
	if hours <= 0 || secondsPerUnit <= 0 {
		return 0
	}
	return hours * 3600 / secondsPerUnit
}

// LevelFinishing runs the capacity-leveled sweep over the second stage.
//
// Each week: greige arrives, every SKU whose projected position has fallen below its trigger becomes
// a candidate, and candidates are served most-depleted-first out of two shared pools — the greige
// their family knitted, and the department's hours. What does not fit waits for the next week, which
// is what makes this a leveling rather than an allocation.
//
// Determinism: SKUs are sorted before any iteration and every tie breaks on SKU, so the same plan
// solves to the same mix twice. Go randomizes map iteration, and a mix that wobbled between runs
// would make a version diff unreadable.
func LevelFinishing(in FinishingInput) FinishingResult {
	weeks := in.Settings.HorizonWeeks
	result := FinishingResult{
		ProjectedOnHand: make(map[string][]float64, len(in.Items)),
		Diagnostics: FinishingDiagnostics{
			WeeklyCapacityHours: in.WeeklyCapacityHours,
			PlannedHoursByWeek:  make([]float64, max(weeks, 0)),
			UtilisationByWeek:   make([]float64, max(weeks, 0)),
			GreigeStarvedSKUs:   []string{},
			CapacityStarvedSKUs: []string{},
			ItemsWithoutRunRate: []string{},
		},
	}
	if weeks <= 0 {
		return result
	}

	items := make([]FinishingItem, len(in.Items))
	copy(items, in.Items)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SKU != items[j].SKU {
			return items[i].SKU < items[j].SKU
		}
		return items[i].ItemID < items[j].ItemID
	})

	position := make(map[string]float64, len(items))
	active := make([]FinishingItem, 0, len(items))
	for _, item := range items {
		result.ProjectedOnHand[item.ItemID] = make([]float64, weeks)
		position[item.ItemID] = item.OnHand

		// A SKU with no measured finishing rate cannot be leveled: there is no way to know what it costs the department, and guessing would let one SKU quietly consume a week nobody planned for it.
		if item.SecondsPerUnit <= 0 {
			result.Diagnostics.ItemsWithoutRunRate = append(result.Diagnostics.ItemsWithoutRunRate, item.SKU)
			continue
		}
		// A dead SKU should not draw greige away from a live one. Demand means either a forecast or orders already on the book — a make-to-order SKU has no forecast by construction.
		if item.WeeklyDemand > 0 || item.hasFirmDemand() {
			active = append(active, item)
		}
	}
	sort.Strings(result.Diagnostics.ItemsWithoutRunRate)

	greige := make(map[string]float64, len(in.GreigeOnHand))
	maps.Copy(greige, in.GreigeOnHand)

	arrivals := make(map[int]map[string]float64, weeks)
	for _, supply := range in.Supply {
		if supply.Quantity <= 0 {
			continue
		}
		// Greige that lands after the horizon is not supply this plan can use, and greige that landed before it is already in GreigeOnHand.
		if supply.WeekIndex < 0 || supply.WeekIndex >= weeks {
			continue
		}
		if arrivals[supply.WeekIndex] == nil {
			arrivals[supply.WeekIndex] = map[string]float64{}
		}
		arrivals[supply.WeekIndex][supply.GreigeItemID] += supply.Quantity
	}

	// Tracked across the horizon rather than per week: a SKU short in week 1 that gets built in week 4 was not starved, it was queued. Only what never gets served at all is reported, which is why a SKU that has been served once can never be added back.
	served := map[string]bool{}
	greigeStarved := map[string]bool{}
	capacityStarved := map[string]bool{}

	for week := range weeks {
		for greigeItemID, quantity := range arrivals[week] {
			greige[greigeItemID] += quantity
		}

		capacityLeft := in.WeeklyCapacityHours

		trigger := make(map[string]float64, len(active))
		due := make([]FinishingItem, 0, len(active))
		for _, item := range active {
			trig := item.triggerForWeek(week, in.Settings)
			trigger[item.ItemID] = trig
			if position[item.ItemID] < trig {
				due = append(due, item)
			}
		}

		// A contractual promise outranks a statistical buffer when the two contend for the same hour, so make-to-order candidates go first. Within each group, most depleted relative to its reorder point wins — measuring the gap rather than the raw position keeps a high-volume SKU from always outranking a low-volume one that is closer to stocking out.
		sort.SliceStable(due, func(i, j int) bool {
			if due[i].IsMakeToOrder != due[j].IsMakeToOrder {
				return due[i].IsMakeToOrder
			}
			gapI := position[due[i].ItemID] - due[i].ReorderPoint
			gapJ := position[due[j].ItemID] - due[j].ReorderPoint
			if gapI != gapJ {
				return gapI < gapJ
			}
			return due[i].SKU < due[j].SKU
		})

		for _, item := range due {
			shortfall := trigger[item.ItemID] - position[item.ItemID]
			if shortfall <= 0 {
				continue
			}

			want := roundUpToLot(shortfall, item.LotUnits)

			perUnit := item.greigePerUnit()
			greigeCeiling := greige[item.GreigeItemID] / perUnit
			fitsGreige := wholeLotsWithin(greigeCeiling, item.LotUnits)
			fitsCapacity := wholeLotsWithin(unitsWithinHours(capacityLeft, item.SecondsPerUnit), item.LotUnits)

			units := math.Min(want, math.Min(fitsGreige, fitsCapacity))
			if units <= 0 {
				// A SKU that has already been built this horizon is queued behind a busier week, not starved. Reporting it would make every leveled plan look like a shortage, since being pushed out of one week is the normal working of a sweep.
				if served[item.SKU] {
					continue
				}
				// Which pool ran out is the diagnosis, and the two lead to opposite actions: knit more, or find more hours. Greige is checked first because it is the upstream cause — a SKU with no greige has no capacity problem to report.
				if fitsGreige <= 0 {
					greigeStarved[item.SKU] = true
				} else {
					capacityStarved[item.SKU] = true
				}
				continue
			}

			hours := units * item.SecondsPerUnit / 3600
			consumed := units * perUnit
			lots := 0
			if item.LotUnits > 0 {
				lots = int(math.Round(units / item.LotUnits))
			}

			before := position[item.ItemID]
			position[item.ItemID] += units
			greige[item.GreigeItemID] -= consumed
			capacityLeft -= hours

			result.Lines = append(result.Lines, FinishingLine{
				ItemID:                item.ItemID,
				SKU:                   item.SKU,
				GreigeItemID:          item.GreigeItemID,
				GreigeSKU:             item.GreigeSKU,
				WeekIndex:             week,
				Quantity:              units,
				Lots:                  lots,
				LotUnits:              item.LotUnits,
				LotUnitID:             item.LotUnitID,
				RunHours:              hours,
				GreigeConsumed:        consumed,
				FirmUnits:             firmUnitsFor(item, week),
				ProductionStepID:      item.ProductionStepID,
				DepartmentID:          item.DepartmentID,
				ProjectedOnHandBefore: before,
				ProjectedOnHandAfter:  position[item.ItemID],
			})

			result.Diagnostics.PlannedHoursByWeek[week] += hours
			result.Diagnostics.TotalPlannedUnits += units

			served[item.SKU] = true
			delete(greigeStarved, item.SKU)
			delete(capacityStarved, item.SKU)
		}

		// Demand is drawn down AFTER the week's builds land, exactly as stage one does it. Drawing first would let a SKU dip below its trigger and be rebuilt inside the same week, double-counting a week of consumption across the horizon.
		for _, item := range items {
			position[item.ItemID] -= item.demandForWeek(week)
			result.ProjectedOnHand[item.ItemID][week] = position[item.ItemID]
		}

		if in.WeeklyCapacityHours > 0 {
			result.Diagnostics.UtilisationByWeek[week] = result.Diagnostics.PlannedHoursByWeek[week] / in.WeeklyCapacityHours
		}
	}

	for sku := range greigeStarved {
		result.Diagnostics.GreigeStarvedSKUs = append(result.Diagnostics.GreigeStarvedSKUs, sku)
	}
	for sku := range capacityStarved {
		result.Diagnostics.CapacityStarvedSKUs = append(result.Diagnostics.CapacityStarvedSKUs, sku)
	}
	sort.Strings(result.Diagnostics.GreigeStarvedSKUs)
	sort.Strings(result.Diagnostics.CapacityStarvedSKUs)

	for _, left := range greige {
		if left > 0 {
			result.Diagnostics.UnusedGreigeUnits += left
		}
	}
	result.Diagnostics.LineCount = len(result.Lines)

	return result
}

// firmUnitsFor is how much of a week's draw on a SKU is an order rather than a forecast.
func firmUnitsFor(item FinishingItem, week int) float64 {
	if week < 0 || week >= len(item.FirmByWeek) {
		return 0
	}
	return item.FirmByWeek[week]
}
