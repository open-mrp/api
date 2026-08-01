package scheduling

import (
	"math"
	"sort"
)

// ItemPolicy is the computed inventory policy for one constraint item.
type ItemPolicy struct {
	ItemID string
	SKU    string
	// UnitID is what every quantity in this policy is counted in. A reorder point of 2,508 is uninterpretable without it, and the policy exists whether or not the item won a slot this horizon — so it carries its own unit rather than borrowing one from a campaign that may not exist.
	UnitID string

	AnnualDemand   float64
	WeeklyDemand   float64
	SecondsPerUnit float64
	UnitCost       float64

	SetupCost   float64
	HoldingCost float64
	EOQUnits    float64

	// Lead times, in weeks.
	ConstraintLeadTimeWeeks float64
	FinishLeadTimeWeeks     float64

	// Two-echelon safety stock. The buffer is pooled at the constraint item because one greige feeds many finished SKUs, so their variability partially cancels (sigma_pooled = sqrt(sum of squares), not the sum). Finished goods keep a smaller per-SKU buffer for service.
	SigmaWeeklyPooled     float64
	SigmaDownstreamSum    float64
	SafetyStockPrimary    float64
	SafetyStockDownstream float64

	ReorderPoint float64
	OrderUpTo    float64

	OnHandEchelon float64
	// OnHandGreige is the constraint stage on its own. The echelon figure is what the build decision is made against; this is what is actually in the greige store, which a pooled total cannot be decomposed back into.
	OnHandGreige float64
	// Greige-stage holding: the buffer plus half a campaign on average, and a whole campaign at the peak. Finished safety stock is held as finished goods and counted on the finished policies, so the two stages sum to network storage without double-counting.
	AverageGreigeInventory float64
	MaxGreigeInventory     float64
	WeeksOfCover           float64
	ABCClass               string
}

// AnnualRunHours is the machine time this item's yearly demand consumes. ABC is classified on this rather than on revenue: the constraint is machine time, so the A items are the ones that eat the schedule.
func (p ItemPolicy) AnnualRunHours() float64 {
	return p.AnnualDemand * p.SecondsPerUnit / 3600
}

// FinishedGood is one finished SKU a constraint item becomes, carried with its identity so the pooled buffer can be decomposed back into per-SKU targets.
type FinishedGood struct {
	ItemID        string
	SKU           string
	ProductLineID string
	Monthly       []MonthlyDemand
	OnHand        float64
}

// FinishedGoodDemand is one finished SKU's measured demand, carried out of the pooling step so its own target can be computed.
type FinishedGoodDemand struct {
	ItemID        string
	SKU           string
	ProductLineID string
	AnnualDemand  float64
	SigmaWeekly   float64
	OnHand        float64
}

// FinishedPolicy is one finished SKU's own inventory target.
//
// The greige buffer is pooled across the family because one greige feeds many finished SKUs and their variability partly cancels. That pooling is what makes the buffer cheap, and it is also what makes these rows necessary: once pooled, the echelon total can no longer answer "is this SKU short".
type FinishedPolicy struct {
	ItemID        string
	SKU           string
	GreigeItemID  string
	GreigeSKU     string
	ProductLineID string

	AnnualDemand float64
	WeeklyDemand float64
	SigmaWeekly  float64

	// Sized against the finishing lead time, not the knit lead time: this stock is replenished by finishing, not by the constraint.
	SafetyStock  float64
	ReorderPoint float64
	OnHand       float64
	WeeksOfCover float64
}

// PolicyInput is one item's measured facts, before policy is applied.
type PolicyInput struct {
	ItemID string
	SKU    string

	AnnualDemand   float64
	SecondsPerUnit float64
	UnitCost       float64

	// OverheadRate is the step's machine burden. Setup cost adds the dedicated changeover labor rate from settings on top; production labor is excluded.
	OverheadRate float64

	// MeasuredLeadTimeWeeks is the observed constraint-step lead time; zero means unmeasured, in which case the settings default applies.
	MeasuredLeadTimeWeeks float64

	SigmaWeeklyPooled  float64
	SigmaDownstreamSum float64
	OnHandEchelon      float64
	OnHandGreige       float64
}

// ComputePolicy derives the inventory policy for one item.
//
//	EOQ = sqrt(2 * D * S / H)                              classic economic order quantity
//	SS_primary    = z * sigma_pooled * sqrt(constraint LT)
//	SS_downstream = z * sum(sigma_fg) * sqrt(finish LT)
//	ROP = weekly * (constraint LT + finish LT) + SS_primary + SS_downstream
//
// The reorder point covers demand over the whole pipeline — constraint plus finishing — because that is how long it takes for a decision made today to become sellable stock.
func ComputePolicy(in PolicyInput, s Settings) ItemPolicy {
	weeksPerYear := float64(s.WeeksPerYear)
	if weeksPerYear <= 0 {
		weeksPerYear = 52
	}

	p := ItemPolicy{
		ItemID:              in.ItemID,
		SKU:                 in.SKU,
		AnnualDemand:        in.AnnualDemand,
		WeeklyDemand:        in.AnnualDemand / weeksPerYear,
		SecondsPerUnit:      in.SecondsPerUnit,
		UnitCost:            in.UnitCost,
		FinishLeadTimeWeeks: s.FinishLeadTimeWeeks,
		SigmaWeeklyPooled:   in.SigmaWeeklyPooled,
		SigmaDownstreamSum:  in.SigmaDownstreamSum,
		OnHandEchelon:       in.OnHandEchelon,
		OnHandGreige:        in.OnHandGreige,
	}

	p.SetupCost = SetupCost(s.ChangeoverAvgMinutes, s.ChangeoverLaborRate, in.OverheadRate)

	// Holding cost floors at a cent so a zero-costed item cannot divide EOQ to infinity. A missing standard cost is a data problem, not a reason to plan an unbounded campaign.
	p.HoldingCost = math.Max(in.UnitCost*s.HoldingRatePct, 0.01)

	if in.AnnualDemand > 0 {
		p.EOQUnits = math.Max(1, math.Sqrt(2*in.AnnualDemand*p.SetupCost/p.HoldingCost))
	}

	p.ConstraintLeadTimeWeeks = in.MeasuredLeadTimeWeeks
	if p.ConstraintLeadTimeWeeks <= 0 {
		p.ConstraintLeadTimeWeeks = s.DefaultConstraintLeadTimeWeeks
	}

	p.SafetyStockPrimary = s.ServiceLevelZ * in.SigmaWeeklyPooled * math.Sqrt(p.ConstraintLeadTimeWeeks)
	p.SafetyStockDownstream = s.ServiceLevelZ * in.SigmaDownstreamSum * math.Sqrt(s.FinishLeadTimeWeeks)

	p.ReorderPoint = p.WeeklyDemand*(p.ConstraintLeadTimeWeeks+s.FinishLeadTimeWeeks) +
		p.SafetyStockPrimary + p.SafetyStockDownstream

	// The "S" of the (s,S) policy: never build past this many weeks of supply, so a cheap-to-make item cannot crowd out one that is actually short.
	p.OrderUpTo = s.MaxWeeksSupply * p.WeeklyDemand

	// Greige-stage holding: the buffer is always there, and a campaign lands on top of it and drains, so the stage averages half a campaign above the buffer and peaks a whole one above it. This is the greige store's size, distinct from the echelon total the build decision uses.
	p.AverageGreigeInventory = p.SafetyStockPrimary + p.EOQUnits/2
	p.MaxGreigeInventory = p.SafetyStockPrimary + p.EOQUnits

	if p.WeeklyDemand > 0 {
		p.WeeksOfCover = in.OnHandEchelon / p.WeeklyDemand
	}

	return p
}

// ClassifyABC assigns A/B/C by cumulative share of annual run hours: A up to 80%, B to 95%, C the tail. Mutates in place and returns the slice sorted by run hours descending, which is also the order the caller wants for display.
//
// Ties are broken by SKU so the classification is stable — without it, two items with identical run hours could swap classes between runs.
//
// Known edge case, preserved from the script for parity: the share is cumulative INCLUSIVE of the current item, so a portfolio dominated by one item classifies that item as C rather than A (its own share already exceeds 95%). With a realistic spread of SKUs this never triggers. Do not "fix" it without re-running the parity gate — the classification feeds display and reporting, not the plan itself.
func ClassifyABC(policies []ItemPolicy) []ItemPolicy {
	sort.SliceStable(policies, func(i, j int) bool {
		hi, hj := policies[i].AnnualRunHours(), policies[j].AnnualRunHours()
		if hi != hj {
			return hi > hj
		}
		return policies[i].SKU < policies[j].SKU
	})

	var total float64
	for _, p := range policies {
		total += p.AnnualRunHours()
	}

	var cumulative float64
	for i := range policies {
		cumulative += policies[i].AnnualRunHours()

		share := 1.0
		if total > 0 {
			share = cumulative / total
		}

		switch {
		case share <= 0.8:
			policies[i].ABCClass = "A"
		case share <= 0.95:
			policies[i].ABCClass = "B"
		default:
			policies[i].ABCClass = "C"
		}
	}

	return policies
}

// ComputeFinishedPolicies decomposes one greige family's pooled buffer into a target per finished SKU.
//
// The greige buffer is sized against the knit lead time and pooled across the family; these are sized against the finishing lead time and stand alone, because that is the stage that actually replenishes them. The two are complementary, not alternatives: the greige buffer decouples knitting from finishing, and these cover the finishing lead time in front of the customer.
func ComputeFinishedPolicies(greige ItemPolicy, finished []FinishedGoodDemand, s Settings) []FinishedPolicy {
	if len(finished) == 0 {
		return nil
	}

	out := make([]FinishedPolicy, 0, len(finished))
	for _, fg := range finished {
		weekly := fg.AnnualDemand / float64(s.WeeksPerYear)

		p := FinishedPolicy{
			ItemID:        fg.ItemID,
			SKU:           fg.SKU,
			GreigeItemID:  greige.ItemID,
			GreigeSKU:     greige.SKU,
			ProductLineID: fg.ProductLineID,
			AnnualDemand:  fg.AnnualDemand,
			WeeklyDemand:  weekly,
			SigmaWeekly:   fg.SigmaWeekly,
			OnHand:        fg.OnHand,
		}

		p.SafetyStock = s.ServiceLevelZ * fg.SigmaWeekly * math.Sqrt(s.FinishLeadTimeWeeks)
		// Only the finishing lead time: the knit lead time is already covered by the greige buffer sitting behind this stock, and charging it twice would inflate finished goods, which is the expensive echelon.
		p.ReorderPoint = weekly*s.FinishLeadTimeWeeks + p.SafetyStock

		if weekly > 0 {
			p.WeeksOfCover = fg.OnHand / weekly
		}

		out = append(out, p)
	}

	// Go randomizes map iteration upstream of this, so the order is pinned here rather than left to whatever the caller happened to accumulate.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SKU != out[j].SKU {
			return out[i].SKU < out[j].SKU
		}
		return out[i].ItemID < out[j].ItemID
	})
	return out
}
