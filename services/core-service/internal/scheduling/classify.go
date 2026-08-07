package scheduling

import (
	"math"
	"sort"
)

// Recommendation reasons. Each names the rule that decided, so a verdict is never a bare answer.
const (
	// ReasonLeadTimeInfeasible means the plant cannot produce inside the window customers were promised, so the stock has to exist before the order does.
	ReasonLeadTimeInfeasible = "lead_time_infeasible"
	// ReasonNoRecentDemand means nothing has sold for long enough that holding a buffer is holding dead stock.
	ReasonNoRecentDemand = "no_recent_demand"
	// ReasonSingleCustomer means effectively one customer buys this, and that customer is served to order.
	ReasonSingleCustomer = "single_customer"
	// ReasonLumpyDemand means demand arrives rarely and in wildly different sizes, which is the shape a safety stock sizes worst.
	ReasonLumpyDemand = "lumpy_demand"
	// ReasonSlowMovingHighValue means each unit is expensive and few sell, so the buffer ties up more money than the service it buys.
	ReasonSlowMovingHighValue = "slow_moving_high_value"
	// ReasonSteadyDemand means demand is regular enough to forecast, which is what a buffer is for.
	ReasonSteadyDemand = "steady_demand"
)

// RecommendationThresholds are the merchant-editable cut points the classifier draws against.
type RecommendationThresholds struct {
	// DormantMonths is how long without a sale before an item counts as dead.
	DormantMonths int
	// ConcentrationPct is the share of demand one customer must hold to count as the only customer.
	ConcentrationPct float64
	// ADIThreshold and CV2Threshold are the Syntetos-Boylan cut points separating smooth demand from lumpy.
	ADIThreshold float64
	CV2Threshold float64
	// SlowMoverCOGS is the annual cost of goods below which an item is a slow mover.
	SlowMoverCOGS float64
	// HighValueUnitCost is the unit cost above which holding stock is expensive.
	HighValueUnitCost float64
}

// DefaultRecommendationThresholds mirrors the schema defaults so a caller that has never configured them still classifies.
func DefaultRecommendationThresholds() RecommendationThresholds {
	return RecommendationThresholds{
		DormantMonths:     12,
		ConcentrationPct:  0.8,
		ADIThreshold:      1.32,
		CV2Threshold:      0.49,
		SlowMoverCOGS:     5000,
		HighValueUnitCost: 50,
	}
}

// CustomerDemand is one customer's share of an item's demand, with the lead time that customer is promised.
type CustomerDemand struct {
	CustomerAccountID string
	CustomerName      string
	Units             float64
	// LeadTimeDays is what this customer is committed to, resolved through the same chain an order is stamped from.
	LeadTimeDays int
	// FulfillmentPolicy is how this customer buys, when they say. Empty means they express no preference.
	FulfillmentPolicy string
}

// ClassificationInput is everything the classifier needs about one item.
type ClassificationInput struct {
	ItemID string
	SKU    string

	// Monthly is the item's demand history, one entry per month with demand. Months with none may be absent; MonthsObserved is the window they were drawn from.
	Monthly        []float64
	MonthsObserved int
	// MonthsSinceLastSale is how long since anything sold. Negative means nothing ever has.
	MonthsSinceLastSale int

	AnnualDemand float64
	UnitCost     float64

	// TotalProductionLeadTimeWeeks is constraint plus finishing: how long from deciding to build to having sellable stock.
	TotalProductionLeadTimeWeeks float64

	Customers []CustomerDemand

	// CurrentPolicy is what the item is planned as today, so the recommendation can say whether anything would change.
	CurrentPolicy string
}

// Recommendation is what the engine thinks an item should be, and why.
type Recommendation struct {
	ItemID string
	SKU    string

	CurrentPolicy     string
	RecommendedPolicy string
	// Reason is the rule that decided. Exactly one, because the rules are ordered and the first match wins — listing every rule that happened to agree would obscure which one actually drove it.
	Reason string

	// The measurements behind the verdict, reported so a planner can disagree with the rule rather than only with the answer.
	AverageDemandInterval  float64
	CoefficientOfVariation float64
	TopCustomerSharePct    float64
	TopCustomerName        string
	// DemandWeightedLeadTimeDays is what customers are promised on average, weighted by how much they buy.
	DemandWeightedLeadTimeDays float64
	AnnualCOGS                 float64
	MonthsSinceLastSale        int
}

// Changes reports whether adopting this recommendation would actually change how the item is planned.
func (r Recommendation) Changes() bool {
	return r.CurrentPolicy != r.RecommendedPolicy
}

// RecommendPolicy decides whether an item should be built to stock or to order, and names the rule that decided.
//
// The rules are ordered and the first match wins:
//
//  1. lead_time_infeasible — customers are promised less time than the plant needs. Checked first because it is a *necessary* condition rather than a preference: if you cannot produce inside the window you promised, stocking it is not a choice you get to make.
//  2. no_recent_demand — nothing has sold in the dormant window. A buffer here is dead stock.
//  3. single_customer — effectively one customer buys it and that customer is served to order.
//  4. lumpy_demand — demand is both intermittent and highly variable, which is the shape a statistical safety stock sizes worst. Uses the Syntetos-Boylan cut points.
//  5. slow_moving_high_value — expensive units, few sold. The buffer costs more than the service it buys.
//  6. otherwise steady_demand — regular enough to forecast, which is what stocking is for.
//
// A recommendation is advice, never applied on its own. Reclassifying a SKU silently would change what the plant builds without anyone deciding to.
func RecommendPolicy(in ClassificationInput, t RecommendationThresholds) Recommendation {
	rec := Recommendation{
		ItemID:              in.ItemID,
		SKU:                 in.SKU,
		CurrentPolicy:       policyOrDefault(in.CurrentPolicy),
		AnnualCOGS:          in.AnnualDemand * in.UnitCost,
		MonthsSinceLastSale: in.MonthsSinceLastSale,
	}

	rec.AverageDemandInterval = averageDemandInterval(in.Monthly, in.MonthsObserved)
	rec.CoefficientOfVariation = demandCV2(in.Monthly)
	rec.DemandWeightedLeadTimeDays = demandWeightedLeadTime(in.Customers)

	topShare, topName, topPolicy := topCustomer(in.Customers)
	rec.TopCustomerSharePct = topShare
	rec.TopCustomerName = topName

	// The production lead time in days, to compare against what customers were promised. Seven days to a week; a plant that needs eight weeks needs 56 days whatever the calendar does.
	productionDays := in.TotalProductionLeadTimeWeeks * 7

	switch {
	case len(in.Customers) > 0 && rec.DemandWeightedLeadTimeDays < productionDays:
		rec.RecommendedPolicy, rec.Reason = PolicyMakeToStock, ReasonLeadTimeInfeasible

	case t.DormantMonths > 0 && in.MonthsSinceLastSale >= t.DormantMonths:
		rec.RecommendedPolicy, rec.Reason = PolicyMakeToOrder, ReasonNoRecentDemand

	case topShare >= t.ConcentrationPct*100 && topPolicy == PolicyMakeToOrder:
		rec.RecommendedPolicy, rec.Reason = PolicyMakeToOrder, ReasonSingleCustomer

	case rec.AverageDemandInterval >= t.ADIThreshold && rec.CoefficientOfVariation >= t.CV2Threshold:
		rec.RecommendedPolicy, rec.Reason = PolicyMakeToOrder, ReasonLumpyDemand

	case rec.AnnualCOGS > 0 && rec.AnnualCOGS < t.SlowMoverCOGS && in.UnitCost > t.HighValueUnitCost:
		rec.RecommendedPolicy, rec.Reason = PolicyMakeToOrder, ReasonSlowMovingHighValue

	default:
		rec.RecommendedPolicy, rec.Reason = PolicyMakeToStock, ReasonSteadyDemand
	}

	return rec
}

// averageDemandInterval is months observed divided by months that had demand.
//
// One means demand every month; two means every other month on average. Measured on monthly buckets, which cannot tell two orders in one month from one — coarse, and reported as such rather than implied to be finer than it is.
func averageDemandInterval(monthly []float64, monthsObserved int) float64 {
	if monthsObserved <= 0 {
		return 0
	}
	withDemand := 0
	for _, units := range monthly {
		if units > 0 {
			withDemand++
		}
	}
	if withDemand == 0 {
		return 0
	}
	return float64(monthsObserved) / float64(withDemand)
}

// demandCV2 is the squared coefficient of variation over the months that had demand.
//
// Squared because that is the form the Syntetos-Boylan cut point is expressed in, and measured over months with demand only: including the empty months would count intermittency twice, once here and once in the interval.
func demandCV2(monthly []float64) float64 {
	values := make([]float64, 0, len(monthly))
	for _, units := range monthly {
		if units > 0 {
			values = append(values, units)
		}
	}
	if len(values) < 2 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	if mean == 0 {
		return 0
	}

	cv := StdDevFloat(values) / mean
	return cv * cv
}

// demandWeightedLeadTime is what customers are promised on average, weighted by how much each buys.
//
// Weighted rather than averaged flat: a plant serving one large 90-day customer and nine tiny 7-day ones can make to order, and a flat mean would say it cannot.
func demandWeightedLeadTime(customers []CustomerDemand) float64 {
	var weighted, total float64
	for _, c := range customers {
		if c.Units <= 0 {
			continue
		}
		weighted += float64(c.LeadTimeDays) * c.Units
		total += c.Units
	}
	if total == 0 {
		return 0
	}
	return weighted / total
}

// topCustomer returns the largest customer's share as a percentage, their name, and how they buy.
func topCustomer(customers []CustomerDemand) (sharePct float64, name string, policy string) {
	var total float64
	for _, c := range customers {
		if c.Units > 0 {
			total += c.Units
		}
	}
	if total == 0 {
		return 0, "", ""
	}

	// Sorted before the scan so two customers with identical volume resolve the same way every run.
	sorted := make([]CustomerDemand, len(customers))
	copy(sorted, customers)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Units != sorted[j].Units {
			return sorted[i].Units > sorted[j].Units
		}
		return sorted[i].CustomerAccountID < sorted[j].CustomerAccountID
	})

	top := sorted[0]
	return top.Units / total * 100, top.CustomerName, top.FulfillmentPolicy
}

// MixedStreamShare is how much of an item's demand comes from customers whose own policy disagrees with how the item is planned.
//
// Reported rather than acted on. Policy is resolved per SKU, so a SKU sold to both a stocking distributor and a contract customer has one policy either way; this is the number that says the choice is uncomfortable, and it is what would justify splitting demand streams later.
func MixedStreamShare(customers []CustomerDemand, itemPolicy string) float64 {
	var total, disagreeing float64
	for _, c := range customers {
		if c.Units <= 0 {
			continue
		}
		total += c.Units
		if c.FulfillmentPolicy != "" && c.FulfillmentPolicy != policyOrDefault(itemPolicy) {
			disagreeing += c.Units
		}
	}
	if total == 0 {
		return 0
	}
	return math.Round(disagreeing/total*10000) / 100
}
