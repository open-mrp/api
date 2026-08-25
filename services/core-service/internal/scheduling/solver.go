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

	// Stage two: the rest of the factory. Empty means the plan stops at the constraint, which is how it behaved before the finishing stage existed.
	//
	// FinishingMachines is every machine outside the constraint department; their count is what the second stage's weekly capacity is derived from, so hiring a shift onto a new machine changes the plan the way it changes the plant.
	FinishingMachines []Machine
	// FinishingBatches is production history for the finished goods, measured anywhere outside the constraint department. It is where the second stage's run rates come from.
	FinishingBatches []BatchMeasurement
	// FinishingRateScaleByItem converts a finished good's measured seconds-per-unit into the unit the plan is denominated in — the greige's. A sock scanned in eaches and knitted in pairs finishes at twice its per-each rate per planned unit.
	FinishingRateScaleByItem map[string]float64
	// FinishingStepByItem is where each finished good is made, denormalized onto its plan line so a department rollup needs no join.
	FinishingStepByItem map[string]FinishingStep
	// FinishingLotByItem is the lot each finished good is made in. Absent means the SKU is planned unlotted.
	FinishingLotByItem map[string]LotDefault
}

// FinishingStep is where a finished good is made, as the second stage's history records it.
type FinishingStep struct {
	ProductionStepID string
	DepartmentID     string
}

// SolverOutput is the plan plus everything needed to explain it.
type SolverOutput struct {
	SolverVersion string
	PlanningAsOf  time.Time

	Policies []ItemPolicy
	// FinishedPolicies is the per-finished-SKU decomposition of the pooled greige buffers in Policies. The two stages together are the whole inventory picture and do not overlap: greige holds its own buffer, finished goods hold theirs.
	FinishedPolicies []FinishedPolicy
	Campaigns        []Campaign
	// ProjectedOnHand[itemID][weekIndex] is the echelon position at the end of that week.
	ProjectedOnHand map[string][]float64
	// ProjectedGreigeOnHand[itemID][weekIndex] is the physical greige store at the end of that week — the constraint stage on its own, which the echelon projection cannot be decomposed back into. It is what the greige buffer is measured against.
	ProjectedGreigeOnHand map[string][]float64

	// Allocations say which campaign is building which order. Written alongside the plan so "what is this campaign for" and "is my order covered" are two readings of one answer.
	Allocations []OrderAllocation

	// FinishingLines are stage two: how many of which finished good to make from the knitted parts, week by week. Empty when the plant has no second stage configured.
	FinishingLines []FinishingLine
	// FinishingProjectedOnHand[itemID][weekIndex] is a finished SKU's own position at the end of that week, which the pooled greige projection cannot answer.
	FinishingProjectedOnHand map[string][]float64

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

	// Finishing is stage two's own account of itself: what it could not make, and why.
	Finishing FinishingDiagnostics `json:"finishing"`
	// FinishingMachineCount is how many machines the second stage was sized from. Zero means its capacity was estimated from the shift pattern alone.
	FinishingMachineCount int `json:"finishing_machine_count"`
	// FinishingCapacityIsEstimated says the plant has no machines outside the constraint department, so stage two was sized as a single notional resource rather than counted. A levelled plan against a guessed capacity is worth flagging as such.
	FinishingCapacityIsEstimated bool `json:"finishing_capacity_is_estimated"`
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
	out.ProjectedGreigeOnHand = levelled.ProjectedGreigeOnHand
	out.Diagnostics.LevellingDiagnostics = levelled.Diagnostics

	skuByItem := make(map[string]string, len(measurements))
	for _, m := range measurements {
		skuByItem[m.ItemID] = m.SKU
	}

	// Which campaign serves which order, and what nothing serves. The at-risk report is read off the same walk rather than derived separately: two calculations of "is this order covered" would eventually disagree, and the one on the plan grid is the one a planner would believe.
	allocation := AllocateCampaignsToOrders(levelled.Campaigns, firm, in.OnHandByItem)
	out.Allocations = allocation.Allocations
	out.Diagnostics.AtRiskOrders = findAtRiskOrders(firm, allocation, skuByItem)

	// Stage two runs against the plan stage one just produced, not against a forecast of it. Solving the two independently would let the finishing plan draw greige the knitting plan never makes.
	finishing := LevelFinishing(buildFinishingInput(in, out))
	out.FinishingLines = finishing.Lines
	out.FinishingProjectedOnHand = finishing.ProjectedOnHand
	out.Diagnostics.Finishing = finishing.Diagnostics
	out.Diagnostics.FinishingMachineCount = len(in.FinishingMachines)
	out.Diagnostics.FinishingCapacityIsEstimated = len(in.FinishingMachines) == 0

	return out
}

// FinishingSupplyLagWeeks is how long stage-one output waits before stage two can work it.
//
// One week, and the reason is arithmetic rather than physical: a campaign planned in week w is
// produced across that week, so treating it as available to finishing in the same week would let the
// plan finish greige that is still on the needles. A configurable lag was considered and dropped —
// it would be a second knob over a lag the week granularity already fixes.
const FinishingSupplyLagWeeks = 1

// buildFinishingInput turns the constraint plan into stage two's supply, and the finished-goods policies into its demand.
//
// This is the seam between the two schedules, and everything load-bearing about the two-stage model
// lives here: campaigns become dated greige arrivals, the pooled family policy becomes one target per
// finished SKU, and the order book is read at the SKU it was actually placed for rather than at the
// greige it was pooled onto.
func buildFinishingInput(in SolverInput, out SolverOutput) FinishingInput {
	capacityPerMachine := in.Settings.MachineWeeklyCapacityHours()
	capacity := float64(len(in.FinishingMachines)) * capacityPerMachine
	if len(in.FinishingMachines) == 0 {
		// A plant with no machines outside the constraint still has a second stage — it just has not been modelled machine by machine. Sizing it as one notional resource keeps the plan usable, and the diagnostic says the number was estimated rather than counted.
		capacity = capacityPerMachine
	}

	finishing := FinishingInput{
		Settings:            in.Settings,
		WeeklyCapacityHours: capacity,
		GreigeOnHand:        map[string]float64{},
	}

	// What is already knitted and waiting when the horizon opens. Without it the first weeks read as starved while the greige store sits full.
	for itemID, quantity := range in.GreigeOnHandByItem {
		if quantity > 0 {
			finishing.GreigeOnHand[itemID] = quantity
		}
	}

	for _, campaign := range out.Campaigns {
		finishing.Supply = append(finishing.Supply, FinishingSupply{
			GreigeItemID: campaign.ItemID,
			WeekIndex:    campaign.WeekIndex + FinishingSupplyLagWeeks,
			Quantity:     campaign.Units,
		})
	}

	// Deterministic supply order. The sweep sums arrivals into a map so order does not change the plan, but a stable slice keeps a diff between two versions readable.
	sort.SliceStable(finishing.Supply, func(i, j int) bool {
		if finishing.Supply[i].WeekIndex != finishing.Supply[j].WeekIndex {
			return finishing.Supply[i].WeekIndex < finishing.Supply[j].WeekIndex
		}
		return finishing.Supply[i].GreigeItemID < finishing.Supply[j].GreigeItemID
	})

	rates := map[string]float64{}
	for _, m := range MeasureItems(in.FinishingBatches) {
		rates[m.ItemID] = m.SecondsPerUnit
	}

	// The order book at the SKU it was placed for. Stage one pools these onto the greige to decide how much to knit; stage two needs them un-pooled, because which colourway was ordered is exactly what it has to decide.
	firmByFinishedItem := firmDemandByFinishedItem(in)

	for _, policy := range out.FinishedPolicies {
		item := FinishingItem{
			ItemID:        policy.ItemID,
			SKU:           policy.SKU,
			GreigeItemID:  policy.GreigeItemID,
			GreigeSKU:     policy.GreigeSKU,
			ProductLineID: policy.ProductLineID,
			WeeklyDemand:  policy.WeeklyDemand,
			OnHand:        policy.OnHand,
			ReorderPoint:  policy.ReorderPoint,
			SafetyStock:   policy.SafetyStock,
			FirmByWeek:    firmByFinishedItem[policy.ItemID],
		}

		// A rate measured per finished unit has to be expressed per planned unit, because the whole plan is denominated in the greige's unit. Without the scale, a SKU scanned in eaches and knitted in pairs would be costed at half the hours it really takes.
		item.SecondsPerUnit = rates[policy.ItemID]
		if scale, ok := in.FinishingRateScaleByItem[policy.ItemID]; ok && scale > 0 {
			item.SecondsPerUnit *= scale
		}

		// Where the SKU is made rides along on the line. It changes nothing about what the sweep decides — the second stage is one pool — but a plan that cannot say which room a job belongs to is not a plan a supervisor can work.
		if step, ok := in.FinishingStepByItem[policy.ItemID]; ok {
			item.ProductionStepID = step.ProductionStepID
			item.DepartmentID = step.DepartmentID
		}
		if lot, ok := in.FinishingLotByItem[policy.ItemID]; ok && lot.Quantity > 0 {
			item.LotUnits = lot.Quantity
			item.LotUnitID = lot.UnitID
		}

		finishing.Items = append(finishing.Items, item)
	}

	return finishing
}

// firmDemandByFinishedItem re-reads the order book at the finished SKU, dated the same way stage one dates it.
//
// BuildFirmSchedule pools onto the constraint item and offsets by the finishing lead time, because a
// greige campaign has to finish before the finishing stage can start. Stage two is that finishing
// stage, so its requirements sit at the promise week itself rather than a lead time earlier.
func firmDemandByFinishedItem(in SolverInput) map[string][]float64 {
	out := map[string][]float64{}
	if in.Settings.HorizonWeeks <= 0 {
		return out
	}

	for _, line := range in.OpenOrders {
		if line.FinishedItemID == "" || line.Units <= 0 {
			continue
		}
		week := 0
		if line.ShipByDate != nil {
			week = weeksBetween(in.HorizonStart, *line.ShipByDate)
		}
		// An order already due is owed now, not never; clamping to week zero is what makes it visible rather than silently dropped.
		week = max(week, 0)
		if week >= in.Settings.HorizonWeeks {
			continue
		}
		if out[line.FinishedItemID] == nil {
			out[line.FinishedItemID] = make([]float64, in.Settings.HorizonWeeks)
		}
		out[line.FinishedItemID][week] += line.Units
	}
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
