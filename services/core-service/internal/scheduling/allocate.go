package scheduling

import "sort"

// OrderAllocation is one campaign's contribution to one order.
type OrderAllocation struct {
	ItemID string
	// CampaignWeek and CampaignMachineID identify the campaign; the caller maps them back onto the line it wrote.
	CampaignWeek      int
	CampaignMachineID string

	SalesOrderID     string
	SalesOrderNumber string
	SalesOrderLineID string
	Units            float64
}

// UncoveredRequirement is an order the plan does not build in time, and by how much.
type UncoveredRequirement struct {
	ItemID           string
	SalesOrderID     string
	SalesOrderNumber string
	DueWeek          int
	ShortUnits       float64
	// CoveredUnits is how much of it the plan does build, so a partly-covered order reads as partly covered rather than as a total miss.
	CoveredUnits float64
}

// AllocationResult is the whole answer: what each campaign is for, and what is not covered.
type AllocationResult struct {
	Allocations []OrderAllocation
	Uncovered   []UncoveredRequirement
}

// AllocateCampaignsToOrders decides which campaign is building which order.
//
// Earliest promise first, and supply consumed in the order it becomes available: stock on hand, then each campaign as it lands. A requirement can only be served by supply that exists by the week it is due — a campaign in week 6 cannot fill an order the constraint owed in week 2, and pretending otherwise is exactly how a plan reports itself achievable while the floor misses dates.
//
// What is left after the walk is what the plan does not build in time, with the amount it does build recorded alongside. A partly-covered order is not a total miss and should not read as one.
//
// Deterministic: both sides are sorted before the walk, so the same plan and order book always produce the same links.
func AllocateCampaignsToOrders(campaigns []Campaign, firm FirmSchedule, onHandByItem map[string]float64) AllocationResult {
	out := AllocationResult{}

	requirementsByItem := map[string][]FirmRequirement{}
	for _, req := range firm.Requirements {
		requirementsByItem[req.ItemID] = append(requirementsByItem[req.ItemID], req)
	}

	campaignsByItem := map[string][]Campaign{}
	for _, c := range campaigns {
		campaignsByItem[c.ItemID] = append(campaignsByItem[c.ItemID], c)
	}

	itemIDs := make([]string, 0, len(requirementsByItem))
	for itemID := range requirementsByItem {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)

	for _, itemID := range itemIDs {
		requirements := append([]FirmRequirement(nil), requirementsByItem[itemID]...)
		// Earliest promise first; ties broken so two orders due the same week always resolve the same way.
		sort.SliceStable(requirements, func(i, j int) bool {
			if requirements[i].DueWeek != requirements[j].DueWeek {
				return requirements[i].DueWeek < requirements[j].DueWeek
			}
			if requirements[i].SalesOrderNumber != requirements[j].SalesOrderNumber {
				return requirements[i].SalesOrderNumber < requirements[j].SalesOrderNumber
			}
			return requirements[i].SalesOrderID < requirements[j].SalesOrderID
		})

		supply := append([]Campaign(nil), campaignsByItem[itemID]...)
		sort.SliceStable(supply, func(i, j int) bool {
			if supply[i].WeekIndex != supply[j].WeekIndex {
				return supply[i].WeekIndex < supply[j].WeekIndex
			}
			return supply[i].MachineID < supply[j].MachineID
		})

		remaining := make([]float64, len(supply))
		for i, c := range supply {
			remaining[i] = c.Units
		}
		onHand := onHandByItem[itemID]

		for _, req := range requirements {
			needed := req.Units

			// Stock already on the floor covers the earliest promises, and carries no campaign to point at.
			if onHand > 0 {
				used := min(onHand, needed)
				onHand -= used
				needed -= used
			}

			for i := range supply {
				if needed <= 0 {
					break
				}
				// Supply that lands after the promise is due cannot serve it.
				if supply[i].WeekIndex > req.DueWeek {
					break
				}
				if remaining[i] <= 0 {
					continue
				}
				used := min(remaining[i], needed)
				remaining[i] -= used
				needed -= used

				out.Allocations = append(out.Allocations, OrderAllocation{
					ItemID:            itemID,
					CampaignWeek:      supply[i].WeekIndex,
					CampaignMachineID: supply[i].MachineID,
					SalesOrderID:      req.SalesOrderID,
					SalesOrderNumber:  req.SalesOrderNumber,
					SalesOrderLineID:  req.SalesOrderLineID,
					Units:             used,
				})
			}

			// Floating point noise from the subtractions above should not read as a real shortfall.
			if needed > 1e-6 {
				out.Uncovered = append(out.Uncovered, UncoveredRequirement{
					ItemID:           itemID,
					SalesOrderID:     req.SalesOrderID,
					SalesOrderNumber: req.SalesOrderNumber,
					DueWeek:          req.DueWeek,
					ShortUnits:       needed,
					CoveredUnits:     req.Units - needed,
				})
			}
		}
	}

	return out
}

// EarliestPromiseWeek is the first horizon week by which the plan could supply a quantity of an item that is not already committed to something else.
//
// Capable-to-promise: what could still be promised, as opposed to what has been. Existing commitments are consumed first, because a date offered out of stock somebody else is already owed is not a date at all.
//
// Returns false when the horizon cannot supply it, which is the honest answer — a plan that runs thirteen weeks cannot speak for the fourteenth.
func EarliestPromiseWeek(itemID string, quantity float64, campaigns []Campaign, firm FirmSchedule, onHandByItem map[string]float64, horizonWeeks int) (int, bool) {
	if quantity <= 0 {
		return 0, true
	}

	supply := make([]float64, horizonWeeks)
	for _, c := range campaigns {
		if c.ItemID != itemID || c.WeekIndex < 0 || c.WeekIndex >= horizonWeeks {
			continue
		}
		supply[c.WeekIndex] += c.Units
	}

	committed := firm.ByItemWeek[itemID]

	available := onHandByItem[itemID]
	for week := range horizonWeeks {
		available += supply[week]
		if week < len(committed) {
			available -= committed[week]
		}
		if available >= quantity {
			return week, true
		}
	}
	return 0, false
}
