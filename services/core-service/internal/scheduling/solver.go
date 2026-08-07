package scheduling

import (
	"sort"
	"time"
)

// SolverInput is everything the solver needs. It is loaded up front by the repository so this package stays pure: same input, same plan, no database, no clock.
type SolverInput struct {
	AccountID    string
	PlanningAsOf time.Time
	Settings     Settings

	Machines []Machine
	Batches  []BatchMeasurement

	// StepInputs maps a production step to the input items it consumes, which drives the changeover model.
	StepInputs map[string]map[string]bool

	// MonthlyByItem is demand pooled onto each constraint item from the finished goods it becomes; DownstreamByItem keeps the per-finished-good detail so the two safety-stock echelons can be computed separately and so each finished SKU's own target can be reported rather than only its contribution to the pool.
	MonthlyByItem    map[string][]MonthlyDemand
	DownstreamByItem map[string][]FinishedGood

	// OnHandByItem is echelon stock — the constraint item plus everything downstream of it — which is what the build decision is made against. GreigeOnHandByItem is the constraint stage on its own, which the echelon total cannot be decomposed back into once summed.
	OnHandByItem       map[string]float64
	GreigeOnHandByItem map[string]float64

	// MachinesWithoutStep is how many machines in the constraint department have no production step, and therefore derive no downstream work.
	MachinesWithoutStep int

	Overrides          []DemandOverride
	ItemsByProductLine map[string][]string

	// Fulfillment policy resolution. All empty means every item is make-to-stock, which is how the plan behaved before policies existed.
	ItemPolicyOverrides      map[string]string
	ProductLineByItem        map[string]string
	PolicyByProductLine      map[string]string
	DefaultFulfillmentPolicy string

	// OpenOrders is the order book the plan owes: outstanding quantities already pooled onto the constraint items that produce them. Empty means the plan is driven purely by forecast, which is how it behaved before the order book was read.
	OpenOrders []OpenOrderLine
	// HorizonStart is the first day of week 0, used to date the order book against the horizon.
	HorizonStart time.Time

	// ItemLotUnits overrides the default lot size for specific items.
	ItemLotUnits map[string]float64
	// LotDefaultByItem is the resolved lot each item is made in, size and unit together. Populated by the input load, which runs the whole precedence chain once.
	LotDefaultByItem map[string]LotDefault
	// ExcludedItemIDs are items the merchant has taken out of planning.
	ExcludedItemIDs map[string]bool

	// PinnedCampaigns are hand-edited campaigns already on the plan; the sweep plans around them rather than re-deriving them.
	PinnedCampaigns []PinnedCampaign

	DemandBasisCode string
	ForecastZ       float64
	ForecastMonths  int
}

// SolverOutput is the plan plus everything needed to explain it.
type SolverOutput struct {
	SolverVersion string
	PlanningAsOf  time.Time

	Policies []ItemPolicy
	// FinishedPolicies is the per-finished-SKU decomposition of the pooled greige buffers in Policies. The two stages together are the whole inventory picture and do not overlap: greige holds its own buffer, finished goods hold theirs.
	FinishedPolicies []FinishedPolicy
	Campaigns        []Campaign
	// ProjectedOnHand[itemID][weekIndex] is the position at the end of that week.
	ProjectedOnHand map[string][]float64

	// Allocations say which campaign is building which order. Written alongside the plan so "what is this campaign for" and "is my order covered" are two readings of one answer.
	Allocations []OrderAllocation

	Diagnostics Diagnostics
}

// Diagnostics is the honest account of what the solver could not do and why the plan differs from raw history. A plan that cannot explain itself will not be trusted.
type Diagnostics struct {
	LevellingDiagnostics
	AppliedOverrides []AppliedOverride `json:"applied_overrides"`
	// ChangeoverSlopeMinutes is the calibrated minutes per additional input.
	ChangeoverSlopeMinutes float64 `json:"changeover_slope_minutes"`
	AverageInputsAdded     float64 `json:"average_inputs_added"`
	// ItemsWithoutRunRate never got a measured seconds-per-unit and cannot be scheduled; they are reported rather than silently dropped.
	ItemsWithoutRunRate []string `json:"items_without_run_rate"`
	ExcludedItemCount   int      `json:"excluded_item_count"`

	// FirmDemandUnits is the whole open order book expressed at the constraint. Zero means the plan is forecast-only.
	FirmDemandUnits float64 `json:"firm_demand_units"`
	// UndatedFirmOrderCount is how many open orders carry no ship-by commitment and were dated at the front of the horizon. A non-zero count means orders predating commitment tracking still need backfilling.
	UndatedFirmOrderCount int `json:"undated_firm_order_count"`
	// MakeToOrderItemCount is how many planned items are built only against the order book.
	MakeToOrderItemCount int `json:"make_to_order_item_count"`
	// AtRiskOrders are commitments this plan does not cover in time. This is the most actionable thing the solver produces: everything else describes what the plan does, and this says which promises it breaks.
	AtRiskOrders []AtRiskOrder `json:"at_risk_orders"`

	// ConstraintMachineCount is how many machines the constraint department contributed, and MeasuredBatchCount how much production history was found on them. A plan with machines but no history is empty for a reason a planner can act on — nothing has been scanned in the demand window — and saying so beats an empty grid.
	ConstraintMachineCount int `json:"constraint_machine_count"`
	MeasuredBatchCount     int `json:"measured_batch_count"`
	// MachinesWithoutStep have no production step, so their campaigns derive no downstream department work.
	MachinesWithoutStep int `json:"machines_without_step"`
}

// AtRiskOrder is a commitment the plan does not meet, with the reason it does not.
type AtRiskOrder struct {
	SalesOrderID     string  `json:"sales_order_id"`
	SalesOrderNumber string  `json:"sales_order_number"`
	ItemID           string  `json:"item_id"`
	SKU              string  `json:"sku"`
	Units            float64 `json:"units"`
	DueWeek          int     `json:"due_week"`
	// Reason is why the promise is at risk: `past_due` (the constraint stage needed to start before the horizon), `undated` (no commitment was ever recorded), or `short` (the plan projects less stock than the order needs in its week).
	Reason string `json:"reason"`
}

// At-risk reasons.
const (
	AtRiskReasonPastDue = "past_due"
	AtRiskReasonUndated = "undated"
	AtRiskReasonShort   = "short"
)

// Solve produces the production plan.
//
// Deterministic by construction: every collection is sorted before iteration, so the same input yields byte-identical output. See TestSolve_Deterministic.
func Solve(in SolverInput) SolverOutput {
	out := SolverOutput{
		SolverVersion:   SolverVersion,
		PlanningAsOf:    in.PlanningAsOf,
		ProjectedOnHand: map[string][]float64{},
	}

	out.Diagnostics.ConstraintMachineCount = len(in.Machines)
	out.Diagnostics.MeasuredBatchCount = len(in.Batches)
	out.Diagnostics.MachinesWithoutStep = in.MachinesWithoutStep

	// The order book is dated and pooled before anything is measured, so both the policy pass and the sweep see the same requirements.
	firm := BuildFirmSchedule(in.OpenOrders, in.HorizonStart, in.Settings)
	out.Diagnostics.FirmDemandUnits = firm.TotalUnits
	out.Diagnostics.UndatedFirmOrderCount = firm.UndatedCount

	measurements := MeasureItems(in.Batches)

	averageAdded := AverageInputsAdded(in.Batches, in.StepInputs)
	changeover := CalibrateChangeover(
		in.Settings.ChangeoverMinMinutes,
		in.Settings.ChangeoverAvgMinutes,
		in.Settings.ChangeoverMaxMinutes,
		averageAdded,
	)
	out.Diagnostics.AverageInputsAdded = averageAdded
	out.Diagnostics.ChangeoverSlopeMinutes = changeover.Slope()

	demands, appliedOverrides := ResolveDemand(DemandInput{
		AsOf:               in.PlanningAsOf,
		BasisCode:          in.DemandBasisCode,
		ForecastZ:          in.ForecastZ,
		WeeksPerYear:       in.Settings.WeeksPerYear,
		ForecastMonths:     in.ForecastMonths,
		MonthlyByItem:      in.MonthlyByItem,
		DownstreamByItem:   in.DownstreamByItem,
		Overrides:          in.Overrides,
		ItemsByProductLine: in.ItemsByProductLine,
	})
	out.Diagnostics.AppliedOverrides = appliedOverrides

	demandByItem := make(map[string]ItemDemand, len(demands))
	for _, d := range demands {
		demandByItem[d.ItemID] = d
	}

	policies := make([]ItemPolicy, 0, len(measurements))
	levellingItems := make([]LevellingItem, 0, len(measurements))

	for _, m := range measurements {
		if in.ExcludedItemIDs[m.ItemID] {
			out.Diagnostics.ExcludedItemCount++
			continue
		}
		// Without a run rate there is no way to know how much machine time a campaign consumes, so the item cannot be levelled. Report it rather than plan a campaign of unknown duration.
		if m.SecondsPerUnit <= 0 {
			out.Diagnostics.ItemsWithoutRunRate = append(out.Diagnostics.ItemsWithoutRunRate, m.SKU)
			continue
		}

		d := demandByItem[m.ItemID]

		resolution := ResolveFulfillmentPolicy(m.ItemID, PolicyResolutionInput{
			ItemOverrides:       in.ItemPolicyOverrides,
			ProductLineByItem:   in.ProductLineByItem,
			PolicyByProductLine: in.PolicyByProductLine,
			AccountDefault:      in.DefaultFulfillmentPolicy,
			DownstreamByItem:    in.DownstreamByItem,
		})
		if resolution.Policy == PolicyMakeToOrder {
			out.Diagnostics.MakeToOrderItemCount++
		}

		var firmUnits float64
		for _, units := range firm.ByItemWeek[m.ItemID] {
			firmUnits += units
		}

		policy := ComputePolicy(PolicyInput{
			ItemID:                m.ItemID,
			SKU:                   m.SKU,
			AnnualDemand:          d.AnnualDemand,
			SecondsPerUnit:        m.SecondsPerUnit,
			UnitCost:              m.UnitCost,
			OverheadRate:          m.OverheadRate,
			MeasuredLeadTimeWeeks: m.MeasuredLeadTimeWeeks,
			SigmaWeeklyPooled:     d.SigmaWeeklyPooled,
			SigmaDownstreamSum:    d.SigmaDownstreamSum,
			OnHandEchelon:         in.OnHandByItem[m.ItemID],
			OnHandGreige:          in.GreigeOnHandByItem[m.ItemID],
			FulfillmentPolicy:     resolution.Policy,
			PolicySource:          resolution.Source,
			FirmDemandUnits:       firmUnits,
			ForecastDemandUnits:   d.AnnualDemand,
		}, in.Settings)

		// The pooled buffer decomposed back into per-SKU finished targets. Without these the echelon total can say whether to knit but not whether any given finished SKU is short.
		out.FinishedPolicies = append(out.FinishedPolicies,
			ComputeFinishedPolicies(policy, d.FinishedGoods, in.Settings)...)

		// The resolved lot carries its unit as well as its size: a doff of sock greige is 60 pairs and a doff of armsleeve greige 60 eaches, and the plan has to say which.
		lotUnits := in.Settings.DefaultLotUnits
		lotUnitID := ""
		if resolved, ok := in.LotDefaultByItem[m.ItemID]; ok && resolved.Quantity > 0 {
			lotUnits = resolved.Quantity
			lotUnitID = resolved.UnitID
		} else if override, ok := in.ItemLotUnits[m.ItemID]; ok && override > 0 {
			lotUnits = override
		}

		policy.UnitID = lotUnitID
		policies = append(policies, policy)
		levellingItems = append(levellingItems, LevellingItem{
			Policy:            policy,
			EligibleMachineID: m.EligibleMachineID,
			LotUnits:          lotUnits,
			LotUnitID:         lotUnitID,
			FirmByWeek:        firm.ByItemWeek[m.ItemID],
		})
	}

	sort.Strings(out.Diagnostics.ItemsWithoutRunRate)

	// ClassifyABC sorts by run hours, so classify a copy and map the classes back onto the levelling items rather than letting the reorder leak into the plan input.
	classified := ClassifyABC(policies)
	classByItem := make(map[string]string, len(classified))
	for _, p := range classified {
		classByItem[p.ItemID] = p.ABCClass
	}
	for i := range levellingItems {
		levellingItems[i].Policy.ABCClass = classByItem[levellingItems[i].Policy.ItemID]
	}
	out.Policies = classified

	levelled := Level(levellingItems, in.Machines, in.Settings, in.PinnedCampaigns)
	out.Campaigns = levelled.Campaigns
	out.ProjectedOnHand = levelled.ProjectedOnHand
	out.Diagnostics.LevellingDiagnostics = levelled.Diagnostics

	skuByItem := make(map[string]string, len(measurements))
	for _, m := range measurements {
		skuByItem[m.ItemID] = m.SKU
	}

	// Which campaign serves which order, and what nothing serves. The at-risk report is read off the same walk rather than derived separately: two calculations of "is this order covered" would eventually disagree, and the one on the plan grid is the one a planner would believe.
	allocation := AllocateCampaignsToOrders(levelled.Campaigns, firm, in.OnHandByItem)
	out.Allocations = allocation.Allocations
	out.Diagnostics.AtRiskOrders = findAtRiskOrders(firm, allocation, skuByItem)

	return out
}

// findAtRiskOrders reports the commitments this plan does not meet.
//
// Three ways a promise is at risk, in order of how certain they are: no commitment was ever recorded, the constraint stage needed to start before the horizon, or the allocation walk could not cover it from stock and campaigns landing in time.
//
// The first two are facts about the order and hold whatever the plan does. The third is a fact about the plan, and it is the one that moves when capacity or settings change — it comes from the same allocation that links campaigns to orders, so the shortfall reported here is exactly the quantity no campaign was earmarked for.
func findAtRiskOrders(firm FirmSchedule, allocation AllocationResult, skuByItem map[string]string) []AtRiskOrder {
	out := make([]AtRiskOrder, 0, len(firm.Requirements))

	shortfall := make(map[string]UncoveredRequirement, len(allocation.Uncovered))
	for _, u := range allocation.Uncovered {
		shortfall[u.SalesOrderID+"|"+u.ItemID] = u
	}

	for _, req := range firm.Requirements {
		reason := ""
		units := req.Units

		switch {
		case req.IsUndated:
			reason = AtRiskReasonUndated
		case req.IsPastDue:
			reason = AtRiskReasonPastDue
		default:
			short, ok := shortfall[req.SalesOrderID+"|"+req.ItemID]
			if !ok {
				continue
			}
			reason = AtRiskReasonShort
			// The quantity actually at risk, not the whole order: a mostly-built order is mostly built.
			units = short.ShortUnits
		}

		out = append(out, AtRiskOrder{
			SalesOrderID:     req.SalesOrderID,
			SalesOrderNumber: req.SalesOrderNumber,
			ItemID:           req.ItemID,
			SKU:              skuByItem[req.ItemID],
			Units:            units,
			DueWeek:          req.DueWeek,
			Reason:           reason,
		})
	}

	return out
}
