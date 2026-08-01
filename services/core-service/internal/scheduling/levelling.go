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

// LevellingDiagnostics records what the solver could not do. These are the numbers a planner needs in order to trust or challenge the plan, so they are part of the output rather than log lines.
type LevellingDiagnostics struct {
	// EOQCappedSKUs had their economic lot size reduced to fit one machine-week, meaning shorter and more frequent campaigns than the policy would prefer.
	EOQCappedSKUs []string `json:"eoq_capped_skus"`
	// UnschedulableSKUs cannot fit even a single minimum lot into a machine-week. They are never scheduled, so expect stockouts.
	UnschedulableSKUs []string `json:"unschedulable_skus"`
	// CapacityStarvedSKUs are below their reorder point but never won a slot in the horizon. This is the honest signal that the plant is short of capacity.
	CapacityStarvedSKUs []string `json:"capacity_starved_skus"`
}

// LevellingResult is the plan plus the per-item projected stock position.
type LevellingResult struct {
	Campaigns   []Campaign
	Diagnostics LevellingDiagnostics
	// ProjectedOnHand[itemID][weekIndex] is the position at the END of that week, after that week's campaigns land and that week's demand is drawn down.
	ProjectedOnHand map[string][]float64
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
type LevellingItem struct {
	Policy            ItemPolicy
	EligibleMachineID map[string]bool
	// LotUnits is the rounding granularity for this item (a doff, a pallet).
	LotUnits float64
	// LotUnitID is what that granularity is counted in.
	LotUnitID string
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

// Level runs the capacity-levelled (s,S) sweep across the horizon.
//
// Each week, every item whose projected position has fallen below its trigger is a candidate. Candidates are served most-depleted-first and placed on the least-loaded eligible machine that still has room. Anything that does not fit waits for the next week.
//
// Pinned campaigns are applied to each week before its candidates are chosen: their inflow and capacity use are facts of the plan, not proposals, so everything the sweep derives already accounts for them. Pins are not re-emitted as campaigns — they exist as lines already.
//
// Determinism: items are sorted by SKU and machines by name before any iteration, and the due-set sort breaks ties by SKU. Iterating the maps directly would produce a different plan on every run.
func Level(items []LevellingItem, machines []Machine, s Settings, pinned []PinnedCampaign) LevellingResult {
	result := LevellingResult{ProjectedOnHand: make(map[string][]float64, len(items))}

	// Stable ordering up front. Machine names sort numerically so "9" precedes "10" and "51" precedes "52" — plain lexical order would interleave them wrongly.
	sortedItems := make([]LevellingItem, len(items))
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

	// Only items with real demand participate; a dead SKU should not consume capacity.
	active := make([]LevellingItem, 0, len(sortedItems))
	for _, item := range sortedItems {
		result.ProjectedOnHand[item.Policy.ItemID] = make([]float64, s.HorizonWeeks)
		if item.Policy.WeeklyDemand > 0 {
			active = append(active, item)
		}
	}

	position := make(map[string]float64, len(sortedItems))
	trigger := make(map[string]float64, len(active))
	campaignUnits := make(map[string]float64, len(active))
	campaignHours := make(map[string]float64, len(active))

	for _, item := range sortedItems {
		position[item.Policy.ItemID] = item.Policy.OnHandEchelon
	}

	for _, item := range active {
		id := item.Policy.ItemID

		// Trigger at the lower of the reorder point and the order-up-to ceiling, so a slow mover with a huge statistical ROP is not built past its cap.
		trig := math.Min(item.Policy.ReorderPoint, item.Policy.OrderUpTo)
		trigger[id] = trig

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
		if item.Policy.OnHandEchelon < trigger[item.Policy.ItemID] {
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
			machineHours[pin.MachineID] += pin.Units * secondsPerUnit[pin.ItemID] / 3600
			delete(starved, skuByItem[pin.ItemID])
		}

		due := make([]LevellingItem, 0, len(active))
		for _, item := range active {
			if position[item.Policy.ItemID] < trigger[item.Policy.ItemID] {
				due = append(due, item)
			}
		}

		// Most depleted relative to its reorder point goes first. Measuring the gap rather than the raw position keeps a high-volume item from always winning over a low-volume one that is closer to stocking out.
		sort.SliceStable(due, func(i, j int) bool {
			gapI := position[due[i].Policy.ItemID] - due[i].Policy.ReorderPoint
			gapJ := position[due[j].Policy.ItemID] - due[j].Policy.ReorderPoint
			if gapI != gapJ {
				return gapI < gapJ
			}
			return due[i].Policy.SKU < due[j].Policy.SKU
		})

		for _, item := range due {
			id := item.Policy.ItemID
			hours := campaignHours[id]

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

			units := campaignUnits[id]
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
			machineHours[best.ID] += hours
			delete(starved, item.Policy.SKU)
		}

		// Demand is drawn down AFTER the week's campaigns land. Doing it before would let an item dip below its trigger and be rebuilt in the same week, which double-counts a week of consumption across the horizon.
		for _, item := range sortedItems {
			id := item.Policy.ItemID
			position[id] -= item.Policy.WeeklyDemand
			result.ProjectedOnHand[id][week] = position[id]
		}
	}

	for sku := range starved {
		result.Diagnostics.CapacityStarvedSKUs = append(result.Diagnostics.CapacityStarvedSKUs, sku)
	}
	sort.Strings(result.Diagnostics.CapacityStarvedSKUs)

	return result
}
