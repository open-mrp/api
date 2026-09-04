package scheduling

import (
	"math"
	"sort"
)

// Machine is one unit of constraint capacity.
type Machine struct {
	ID   string
	Name string
}

// Campaign is one planned production block: make this item, on this machine, in this week.
type Campaign struct {
	ItemID    string
	SKU       string
	MachineID string
	WeekIndex int
	Units     float64
	Lots      int
	// LotUnits is the granularity the campaign was sized at, carried onto the plan so releasing the week to the floor splits into exactly the lots that were planned.
	LotUnits float64
	// LotUnitID is what the lot is counted in — pairs for sock greige, eaches for armsleeve greige. Without it a 60 on the plan cannot be reconciled with a 60 on the floor.
	LotUnitID string
	RunHours  float64
}

// LevelingDiagnostics records what the solver could not do. These are the numbers a planner needs in order to trust or challenge the plan, so they are part of the output rather than log lines.
type LevelingDiagnostics struct {
	// EOQCappedSKUs had their economic lot size reduced to fit one machine-week, meaning shorter and more frequent campaigns than the policy would prefer.
	EOQCappedSKUs []string `json:"eoq_capped_skus"`
	// UnschedulableSKUs cannot fit even a single minimum lot into a machine-week. They are never scheduled, so expect stockouts.
	UnschedulableSKUs []string `json:"unschedulable_skus"`
	// CapacityStarvedSKUs are below their reorder point but never won a slot in the horizon. This is the honest signal that the plant is short of capacity.
	CapacityStarvedSKUs []string `json:"capacity_starved_skus"`
}

// LevelingResult is the plan plus the per-item projected stock position.
type LevelingResult struct {
	Campaigns   []Campaign
	Diagnostics LevelingDiagnostics
	// ProjectedOnHand[itemID][weekIndex] is the echelon position at the END of that week, after that week's campaigns land and that week's demand is drawn down.
	ProjectedOnHand map[string][]float64
	// ProjectedGreigeOnHand[itemID][weekIndex] is the physical greige store — the constraint stage on its own, not the echelon — at the end of that week. It starts from what is knitted and waiting, rises with each campaign, and is drawn down by demand as a proxy for finishing pull. It is what the greige-buffer trigger watches, and it is always populated so the store can be shown even where the trigger is off.
	ProjectedGreigeOnHand map[string][]float64
}

// PinnedCampaign is a hand-edited campaign the sweep must plan around rather than re-derive. Its units raise the item's projected position in its week and its run time consumes that machine's capacity, so the rest of the plan responds to the hand edit: build something sooner and the solver builds less of it later; trim a campaign and the solver replenishes earlier.
type PinnedCampaign struct {
	ItemID    string
	MachineID string
	WeekIndex int
	Units     float64
}

// pinnedSlotKey identifies the machine-week slot a pinned campaign occupies, so the sweep does not double-book it with a solver campaign for the same item.
type pinnedSlotKey struct {
	ItemID    string
	MachineID string
	WeekIndex int
}

// itemEligibility restricts an item to the machines that have historically run it. Empty means any machine.
type LevelingItem struct {
	Policy            ItemPolicy
	EligibleMachineID map[string]bool
	// LotUnits is the rounding granularity for this item (a doff, a pallet).
	LotUnits float64
	// LotUnitID is what that granularity is counted in.
	LotUnitID string
	// FirmByWeek is the dated order book for this item, indexed by week. Nil when nothing is on order, which is the case the plan behaved as before this existed.
	FirmByWeek []float64
}

// hasFirmDemand reports whether anything is on order for this item inside the horizon.
func (i LevelingItem) hasFirmDemand() bool {
	for _, units := range i.FirmByWeek {
		if units > 0 {
			return true
		}
	}
	return false
}

// leadTimeWeeks is how far ahead a make-to-order item has to look: a decision made now becomes sellable stock only after the constraint stage and finishing have both run.
func (i LevelingItem) leadTimeWeeks() int {
	weeks := int(math.Ceil(i.Policy.ConstraintLeadTimeWeeks + i.Policy.FinishLeadTimeWeeks))
	return max(weeks, 1)
}

// firmRequiredThrough is the order book this item owes from the given week through its lead time.
//
// This is the make-to-order reorder point, and it is the same idea as the statistical one: stock has to cover demand over the lead time. The difference is where the demand comes from — a forecast averaged over a year, or the dated orders already on the book.
func (i LevelingItem) firmRequiredThrough(week int) float64 {
	if len(i.FirmByWeek) == 0 {
		return 0
	}
	var total float64
	for w := week; w <= week+i.leadTimeWeeks() && w < len(i.FirmByWeek); w++ {
		if w < 0 {
			continue
		}
		total += i.FirmByWeek[w]
	}
	return total
}

// triggerForWeek is the position below which this item needs building in the given week.
//
// Make-to-stock uses the (s,S) trigger it always has: a constant, the lower of the reorder point and the order-up-to ceiling. Make-to-order recomputes each week from the dated order book, because there is no average to reduce it to — an item with nothing on order this week needs nothing, and the same item needs a full campaign the week an order lands inside its lead time.
func (i LevelingItem) triggerForWeek(week int) float64 {
	if i.Policy.IsMakeToOrder() {
		return i.firmRequiredThrough(week)
	}
	return math.Min(i.Policy.ReorderPoint, i.Policy.OrderUpTo)
}

// demandForWeek is what the item actually consumes in one week: the greater of its forecast rate and the orders already on the book for that week.
//
// This is forecast consumption. An order inside the forecast is served BY the forecast rather than added to it — adding them would double-count the same demand, once as history repeating and once as the order that history predicted. Taking the greater means a week with no orders still plans for the average, and a week holding a large order plans for the order.
//
// With no order book this returns the weekly forecast unchanged, which is what keeps the plan byte-identical to the one produced before the order book existed.
func (i LevelingItem) demandForWeek(week int) float64 {
	forecast := i.Policy.WeeklyDemand
	if week < 0 || week >= len(i.FirmByWeek) {
		return forecast
	}
	if firm := i.FirmByWeek[week]; firm > forecast {
		return firm
	}
	return forecast
}

// roundUpToLot rounds a quantity up to a whole number of lots, with a floor of one lot. Script: toDoffs = max(AVG_DOFF, ceil(q / AVG_DOFF) * AVG_DOFF).
func roundUpToLot(quantity, lotUnits float64) float64 {
	if lotUnits <= 0 {
		return quantity
	}
	return math.Max(lotUnits, math.Ceil(quantity/lotUnits)*lotUnits)
}

// maxLotsInCapacity is the largest whole-lot quantity that fits in the given hours. Note this rounds DOWN where roundUpToLot rounds up: one is the economic lot size, the other is a physical ceiling, and conflating them produces campaigns that cannot run. Script: fitDoffs = max(AVG_DOFF, floor(maxUnits / AVG_DOFF) * AVG_DOFF).
func maxLotsInCapacity(capacityHours, secondsPerUnit, lotUnits float64) float64 {
	if secondsPerUnit <= 0 || lotUnits <= 0 {
		return lotUnits
	}
	maxUnits := capacityHours * 3600 / secondsPerUnit
	return math.Max(lotUnits, math.Floor(maxUnits/lotUnits)*lotUnits)
}

// Level runs the capacity-leveled (s,S) sweep across the horizon.
//
// Each week, every item whose projected position has fallen below its trigger is a candidate. Candidates are served most-depleted-first and placed on the least-loaded eligible machine that still has room. Anything that does not fit waits for the next week.
//
// Pinned campaigns are applied to each week before its candidates are chosen: their inflow and capacity use are facts of the plan, not proposals, so everything the sweep derives already accounts for them. Pins are not re-emitted as campaigns — they exist as lines already.
//
// Determinism: items are sorted by SKU and machines by name before any iteration, and the due-set sort breaks ties by SKU. Iterating the maps directly would produce a different plan on every run.
func Level(items []LevelingItem, machines []Machine, s Settings, pinned []PinnedCampaign) LevelingResult {
	result := LevelingResult{
		ProjectedOnHand:       make(map[string][]float64, len(items)),
		ProjectedGreigeOnHand: make(map[string][]float64, len(items)),
	}

	// Stable ordering up front. Machine names sort numerically so "9" precedes "10" and "51" precedes "52" — plain lexical order would interleave them wrongly.
	sortedItems := make([]LevelingItem, len(items))
	copy(sortedItems, items)
	sort.SliceStable(sortedItems, func(i, j int) bool {
		return sortedItems[i].Policy.SKU < sortedItems[j].Policy.SKU
	})

	sortedMachines := make([]Machine, len(machines))
	copy(sortedMachines, machines)
	sort.SliceStable(sortedMachines, func(i, j int) bool {
		return naturalLess(sortedMachines[i].Name, sortedMachines[j].Name)
	})

	capacityPerMachine := s.MachineWeeklyCapacityHours()

	// Only items with real demand participate; a dead SKU should not consume capacity. Demand means either a forecast or orders already on the book — a make-to-order item has no forecast by construction, and filtering on the forecast alone would drop exactly the items this policy exists to build.
	active := make([]LevelingItem, 0, len(sortedItems))
	for _, item := range sortedItems {
		result.ProjectedOnHand[item.Policy.ItemID] = make([]float64, s.HorizonWeeks)
		result.ProjectedGreigeOnHand[item.Policy.ItemID] = make([]float64, s.HorizonWeeks)
		if item.Policy.WeeklyDemand > 0 || item.hasFirmDemand() {
			active = append(active, item)
		}
	}

	position := make(map[string]float64, len(sortedItems))
	// greigePosition is the physical greige store, tracked in parallel to the echelon position. The echelon draws down on sale and the greige store on finishing conversion; the sweep models no lead delay, so both are drawn down by the same weekly demand, which conserves units over the horizon. The greige-buffer trigger watches this rather than the echelon so a family whose stock is locked up as finished goods still knits when its raw buffer runs dry.
	greigePosition := make(map[string]float64, len(sortedItems))
	// greigeFloor is the reorder point for that store: the pooled greige safety stock, which a campaign of one EOQ then lands on top of, so the store oscillates between the floor and floor+EOQ exactly as AverageGreigeInventory / MaxGreigeInventory describe. Make-to-order items hold no buffer, so their floor is zero and the trigger never fires for them.
	greigeFloor := make(map[string]float64, len(active))
	trigger := make(map[string]float64, len(active))
	campaignUnits := make(map[string]float64, len(active))
	campaignHours := make(map[string]float64, len(active))

	for _, item := range sortedItems {
		position[item.Policy.ItemID] = item.Policy.OnHandEchelon
		greigePosition[item.Policy.ItemID] = item.Policy.OnHandGreige
	}

	for _, item := range active {
		id := item.Policy.ItemID

		// Trigger at the lower of the reorder point and the order-up-to ceiling, so a slow mover with a huge statistical ROP is not built past its cap. A make-to-order item recomputes this per week instead; the value cached here is only its week-zero position.
		trig := item.triggerForWeek(0)
		trigger[id] = trig

		// A make-to-order item is not buffered, so its greige floor stays zero and only the echelon trigger governs it.
		if s.GreigeBufferEnabled && !item.Policy.IsMakeToOrder() {
			greigeFloor[id] = item.Policy.SafetyStockPrimary
		}

		economic := roundUpToLot(item.Policy.EOQUnits, item.LotUnits)
		fits := maxLotsInCapacity(capacityPerMachine, item.Policy.SecondsPerUnit, item.LotUnits)
		units := math.Min(economic, fits)
		campaignUnits[id] = units
		campaignHours[id] = units * item.Policy.SecondsPerUnit / 3600

		if units < economic {
			result.Diagnostics.EOQCappedSKUs = append(result.Diagnostics.EOQCappedSKUs, item.Policy.SKU)
		}
		if campaignHours[id] > capacityPerMachine {
			result.Diagnostics.UnschedulableSKUs = append(result.Diagnostics.UnschedulableSKUs, item.Policy.SKU)
		}
	}

	// Track which already-short items never get served, so capacity starvation is reported rather than silently absorbed.
	starved := make(map[string]bool)
	for _, item := range active {
		id := item.Policy.ItemID
		// Short on either count is a candidate for starvation: an item whose greige buffer is dry needs building even where its echelon reads full, and if it never wins a slot the plant could not afford that buffer — which is the same capacity signal.
		greigeDry := s.GreigeBufferEnabled && item.Policy.OnHandGreige < greigeFloor[id]
		if item.Policy.OnHandEchelon < trigger[id] || greigeDry {
			starved[item.Policy.SKU] = true
		}
	}

	// Pins indexed for the sweep: what lands each week, and which slots are already taken. An item the solver has no measurements for cannot be positioned or costed, so its pins are ignored rather than guessed at.
	secondsPerUnit := make(map[string]float64, len(sortedItems))
	skuByItem := make(map[string]string, len(sortedItems))
	for _, item := range sortedItems {
		secondsPerUnit[item.Policy.ItemID] = item.Policy.SecondsPerUnit
		skuByItem[item.Policy.ItemID] = item.Policy.SKU
	}
	pinsByWeek := make(map[int][]PinnedCampaign, len(pinned))
	pinnedSlot := make(map[pinnedSlotKey]bool, len(pinned))
	for _, pin := range pinned {
		if pin.WeekIndex < 0 || pin.WeekIndex >= s.HorizonWeeks {
			continue
		}
		// A campaign building nothing is not a campaign. Honouring it would hold its slot against the sweep, so the plan would leave a machine-week empty for work that was never going to happen.
		if pin.Units <= 0 {
			continue
		}
		if _, known := secondsPerUnit[pin.ItemID]; !known {
			continue
		}
		pinsByWeek[pin.WeekIndex] = append(pinsByWeek[pin.WeekIndex], pin)
		pinnedSlot[pinnedSlotKey{ItemID: pin.ItemID, MachineID: pin.MachineID, WeekIndex: pin.WeekIndex}] = true
	}

	for week := range s.HorizonWeeks {
		machineHours := make(map[string]float64, len(sortedMachines))

		// Hand-pinned campaigns land first: their stock arrives and their machine time is spent before the sweep decides what else the week can hold.
		for _, pin := range pinsByWeek[week] {
			position[pin.ItemID] += pin.Units
			greigePosition[pin.ItemID] += pin.Units
			machineHours[pin.MachineID] += pin.Units * secondsPerUnit[pin.ItemID] / 3600
			delete(starved, skuByItem[pin.ItemID])
		}

		// Make-to-order triggers move with the order book, so they are recomputed each week rather than read from the constant above.
		weekTrigger := make(map[string]float64, len(active))
		for _, item := range active {
			id := item.Policy.ItemID
			if item.Policy.IsMakeToOrder() {
				weekTrigger[id] = item.triggerForWeek(week)
			} else {
				weekTrigger[id] = trigger[id]
			}
		}

		due := make([]LevelingItem, 0, len(active))
		for _, item := range active {
			id := item.Policy.ItemID
			// Due on either count: the echelon has fallen below its reorder point, or the physical greige store has fallen below its own floor even though the family reads covered. The second is what keeps a buffer of undifferentiated greige in front of finishing so it can build the colorways that are actually short. The greige store is tracked even when the buffer is off (so it can be shown), and it drifts negative as demand is drawn down, so the floor check is guarded by the flag rather than by the floor being zero.
			greigeDry := s.GreigeBufferEnabled && greigePosition[id] < greigeFloor[id]
			if position[id] < weekTrigger[id] || greigeDry {
				due = append(due, item)
			}
		}

		// A contractual promise outranks a statistical buffer when the two contend for the same machine-hour, so make-to-order candidates are served first. Within each group the rule is unchanged: most depleted relative to its reorder point goes first, because measuring the gap rather than the raw position keeps a high-volume item from always winning over a low-volume one that is closer to stocking out.
		sort.SliceStable(due, func(i, j int) bool {
			mtoI, mtoJ := due[i].Policy.IsMakeToOrder(), due[j].Policy.IsMakeToOrder()
			if mtoI != mtoJ {
				return mtoI
			}
			gapI := position[due[i].Policy.ItemID] - due[i].Policy.ReorderPoint
			gapJ := position[due[j].Policy.ItemID] - due[j].Policy.ReorderPoint
			if gapI != gapJ {
				return gapI < gapJ
			}
			return due[i].Policy.SKU < due[j].Policy.SKU
		})

		for _, item := range due {
			id := item.Policy.ItemID

			// A make-to-order campaign is sized to what is actually short, not to an economic lot: the economic quantity exists to amortize a setup across future demand, and there is no future demand to amortize against — only the order in front of it.
			units := campaignUnits[id]
			if item.Policy.IsMakeToOrder() {
				shortfall := weekTrigger[id] - position[id]
				if shortfall <= 0 {
					continue
				}
				fits := maxLotsInCapacity(capacityPerMachine, item.Policy.SecondsPerUnit, item.LotUnits)
				units = math.Min(roundUpToLot(shortfall, item.LotUnits), fits)
			}
			if units <= 0 {
				continue
			}
			hours := units * item.Policy.SecondsPerUnit / 3600

			var best *Machine
			for i := range sortedMachines {
				machine := &sortedMachines[i]
				if len(item.EligibleMachineID) > 0 && !item.EligibleMachineID[machine.ID] {
					continue
				}
				// A slot a hand edit already occupies is not re-proposed; the pinned line IS this item's campaign there.
				if pinnedSlot[pinnedSlotKey{ItemID: id, MachineID: machine.ID, WeekIndex: week}] {
					continue
				}
				if machineHours[machine.ID]+hours > capacityPerMachine {
					continue
				}
				if best == nil || machineHours[machine.ID] < machineHours[best.ID] {
					best = machine
				}
			}
			if best == nil {
				continue
			}

			lots := 0
			if item.LotUnits > 0 {
				lots = int(math.Round(units / item.LotUnits))
			}

			result.Campaigns = append(result.Campaigns, Campaign{
				ItemID:    id,
				SKU:       item.Policy.SKU,
				MachineID: best.ID,
				WeekIndex: week,
				Units:     units,
				Lots:      lots,
				LotUnits:  item.LotUnits,
				LotUnitID: item.LotUnitID,
				RunHours:  hours,
			})

			position[id] += units
			greigePosition[id] += units
			machineHours[best.ID] += hours
			delete(starved, item.Policy.SKU)
		}

		// Demand is drawn down AFTER the week's campaigns land. Doing it before would let an item dip below its trigger and be rebuilt in the same week, which double-counts a week of consumption across the horizon.
		for _, item := range sortedItems {
			id := item.Policy.ItemID
			demand := item.demandForWeek(week)
			position[id] -= demand
			greigePosition[id] -= demand
			result.ProjectedOnHand[id][week] = position[id]
			result.ProjectedGreigeOnHand[id][week] = greigePosition[id]
		}
	}

	for sku := range starved {
		result.Diagnostics.CapacityStarvedSKUs = append(result.Diagnostics.CapacityStarvedSKUs, sku)
	}
	sort.Strings(result.Diagnostics.CapacityStarvedSKUs)

	return result
}
