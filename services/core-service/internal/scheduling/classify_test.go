package scheduling

import (
	"math"
	"testing"
)

// steadyItem is a well-behaved make-to-stock item: sells every month, in similar quantities, to customers who allow enough time.
func steadyItem() ClassificationInput {
	monthly := make([]float64, 12)
	for i := range monthly {
		monthly[i] = 100 + float64(i%3)
	}
	return ClassificationInput{
		ItemID:                       "itm_a",
		SKU:                          "SKU-A",
		Monthly:                      monthly,
		MonthsObserved:               12,
		MonthsSinceLastSale:          0,
		AnnualDemand:                 1200,
		UnitCost:                     10,
		TotalProductionLeadTimeWeeks: 4,
		Customers: []CustomerDemand{
			{CustomerAccountID: "ac_1", CustomerName: "Acme", Units: 700, LeadTimeDays: 60},
			{CustomerAccountID: "ac_2", CustomerName: "Globex", Units: 500, LeadTimeDays: 60},
		},
		CurrentPolicy: PolicyMakeToStock,
	}
}

func TestRecommendPolicy_SteadyDemandStaysMakeToStock(t *testing.T) {
	t.Parallel()

	got := RecommendPolicy(steadyItem(), DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToStock || got.Reason != ReasonSteadyDemand {
		t.Fatalf("got %s/%s, want make_to_stock/steady_demand", got.RecommendedPolicy, got.Reason)
	}
	if got.Changes() {
		t.Fatal("an item already planned to stock should not be reported as changing")
	}
}

// The necessary condition, checked before every preference: a plant that needs eight weeks cannot make to order for customers promised two.
func TestRecommendPolicy_LeadTimeInfeasibilityOverridesEverything(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	// Lumpy, dormant, expensive and single-customer — every rule that would say make-to-order.
	in.Monthly = []float64{0, 0, 0, 5000, 0, 0, 0, 0, 0, 0, 0, 20}
	in.MonthsSinceLastSale = 24
	in.UnitCost = 500
	in.AnnualDemand = 10
	in.TotalProductionLeadTimeWeeks = 8
	in.Customers = []CustomerDemand{{
		CustomerAccountID: "ac_1", CustomerName: "Acme", Units: 1000,
		LeadTimeDays: 14, FulfillmentPolicy: PolicyMakeToOrder,
	}}

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToStock || got.Reason != ReasonLeadTimeInfeasible {
		t.Fatalf("got %s/%s, want make_to_stock/lead_time_infeasible — a promise you cannot keep by producing is not a choice",
			got.RecommendedPolicy, got.Reason)
	}
}

func TestRecommendPolicy_DormantItemIsNotWorthStocking(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	in.MonthsSinceLastSale = 18
	in.Customers[0].LeadTimeDays = 120
	in.Customers[1].LeadTimeDays = 120

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToOrder || got.Reason != ReasonNoRecentDemand {
		t.Fatalf("got %s/%s, want make_to_order/no_recent_demand", got.RecommendedPolicy, got.Reason)
	}
	if !got.Changes() {
		t.Fatal("moving a stocked item to make-to-order is a change")
	}
}

// One customer taking nearly all of it, and buying to order, is the clearest make-to-order signal there is.
func TestRecommendPolicy_SingleCustomerWhoBuysToOrder(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	in.Customers = []CustomerDemand{
		{CustomerAccountID: "ac_1", CustomerName: "Contract Co", Units: 950, LeadTimeDays: 90, FulfillmentPolicy: PolicyMakeToOrder},
		{CustomerAccountID: "ac_2", CustomerName: "Occasional", Units: 50, LeadTimeDays: 90},
	}

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToOrder || got.Reason != ReasonSingleCustomer {
		t.Fatalf("got %s/%s, want make_to_order/single_customer", got.RecommendedPolicy, got.Reason)
	}
	if got.TopCustomerName != "Contract Co" {
		t.Fatalf("top customer = %q, want Contract Co", got.TopCustomerName)
	}
	if math.Abs(got.TopCustomerSharePct-95) > 0.001 {
		t.Fatalf("top share = %v, want 95", got.TopCustomerSharePct)
	}
}

// The same dominant customer who stocks rather than orders is not a reason to stop stocking.
func TestRecommendPolicy_SingleCustomerWhoStocksIsNotASignal(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	in.Customers = []CustomerDemand{
		{CustomerAccountID: "ac_1", CustomerName: "Distributor", Units: 950, LeadTimeDays: 90, FulfillmentPolicy: PolicyMakeToStock},
		{CustomerAccountID: "ac_2", CustomerName: "Occasional", Units: 50, LeadTimeDays: 90},
	}

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToStock {
		t.Fatalf("got %s/%s, want make_to_stock — a dominant customer who wants stock is a reason to hold it",
			got.RecommendedPolicy, got.Reason)
	}
}

// Rare, wildly varying demand is exactly the shape a statistical safety stock sizes worst.
func TestRecommendPolicy_LumpyDemand(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	in.Monthly = []float64{0, 0, 5, 0, 0, 0, 900, 0, 0, 0, 0, 40}
	in.Customers[0].LeadTimeDays = 120
	in.Customers[1].LeadTimeDays = 120

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToOrder || got.Reason != ReasonLumpyDemand {
		t.Fatalf("got %s/%s (adi=%v cv2=%v), want make_to_order/lumpy_demand",
			got.RecommendedPolicy, got.Reason, got.AverageDemandInterval, got.CoefficientOfVariation)
	}
	if got.AverageDemandInterval < DefaultRecommendationThresholds().ADIThreshold {
		t.Fatalf("ADI = %v, expected it above the threshold for this series", got.AverageDemandInterval)
	}
}

func TestRecommendPolicy_SlowMovingHighValue(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	// Steady but tiny, and expensive: 20 units a year at $200 is $4,000 of COGS.
	in.Monthly = []float64{2, 2, 1, 2, 2, 1, 2, 2, 1, 2, 2, 1}
	in.AnnualDemand = 20
	in.UnitCost = 200
	in.Customers[0].LeadTimeDays = 120
	in.Customers[1].LeadTimeDays = 120

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.RecommendedPolicy != PolicyMakeToOrder || got.Reason != ReasonSlowMovingHighValue {
		t.Fatalf("got %s/%s (cogs=%v), want make_to_order/slow_moving_high_value",
			got.RecommendedPolicy, got.Reason, got.AnnualCOGS)
	}
	if got.AnnualCOGS != 4000 {
		t.Fatalf("annual COGS = %v, want 4000", got.AnnualCOGS)
	}
}

// The thresholds are merchant-editable, so the same item has to classify differently under different ones.
func TestRecommendPolicy_ThresholdsDecide(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	in.Monthly = []float64{0, 0, 5, 0, 0, 0, 900, 0, 0, 0, 0, 40}
	in.Customers[0].LeadTimeDays = 120
	in.Customers[1].LeadTimeDays = 120

	lenient := DefaultRecommendationThresholds()
	lenient.ADIThreshold = 99
	lenient.CV2Threshold = 99

	if got := RecommendPolicy(in, lenient); got.RecommendedPolicy != PolicyMakeToStock {
		t.Fatalf("with the lumpiness cut points raised out of reach, got %s/%s, want make_to_stock",
			got.RecommendedPolicy, got.Reason)
	}
}

// Weighted by volume, not averaged flat: one large patient customer is what makes producing to order possible.
func TestDemandWeightedLeadTime_WeightsByVolume(t *testing.T) {
	t.Parallel()

	customers := []CustomerDemand{
		{CustomerAccountID: "ac_big", Units: 900, LeadTimeDays: 90},
		{CustomerAccountID: "ac_small", Units: 100, LeadTimeDays: 10},
	}
	// (900*90 + 100*10) / 1000 = 82
	if got := demandWeightedLeadTime(customers); math.Abs(got-82) > 0.001 {
		t.Fatalf("got %v, want 82 — a flat mean would say 50 and misjudge the plant", got)
	}
}

func TestAverageDemandInterval(t *testing.T) {
	t.Parallel()

	// Demand in 4 of 12 months: one order every three months on average.
	if got := averageDemandInterval([]float64{10, 0, 0, 5, 0, 0, 8, 0, 0, 3, 0, 0}, 12); math.Abs(got-3) > 0.001 {
		t.Fatalf("got %v, want 3", got)
	}
	// Demand every month.
	if got := averageDemandInterval([]float64{1, 1, 1, 1}, 4); math.Abs(got-1) > 0.001 {
		t.Fatalf("got %v, want 1", got)
	}
	// Nothing ever sold: no interval to report, and reporting zero as "sells constantly" would invert the meaning.
	if got := averageDemandInterval([]float64{0, 0, 0}, 3); got != 0 {
		t.Fatalf("got %v, want 0", got)
	}
}

// Measured over months with demand only: counting the empty ones would charge intermittency twice, once here and once in the interval.
func TestDemandCV2_IgnoresEmptyMonths(t *testing.T) {
	t.Parallel()

	steady := demandCV2([]float64{100, 100, 100, 0, 0, 0})
	if steady != 0 {
		t.Fatalf("identical demand should have no variation, got %v", steady)
	}

	varied := demandCV2([]float64{10, 1000, 0, 0})
	if varied <= DefaultRecommendationThresholds().CV2Threshold {
		t.Fatalf("10 vs 1000 should read as highly variable, got %v", varied)
	}
}

func TestMixedStreamShare(t *testing.T) {
	t.Parallel()

	customers := []CustomerDemand{
		{CustomerAccountID: "ac_1", Units: 780, FulfillmentPolicy: PolicyMakeToOrder},
		{CustomerAccountID: "ac_2", Units: 220, FulfillmentPolicy: PolicyMakeToStock},
	}

	// Planned to stock, so the make-to-order customers are the ones who disagree.
	if got := MixedStreamShare(customers, PolicyMakeToStock); math.Abs(got-78) > 0.001 {
		t.Fatalf("got %v, want 78", got)
	}
	// Planned to order, so the disagreement flips.
	if got := MixedStreamShare(customers, PolicyMakeToOrder); math.Abs(got-22) > 0.001 {
		t.Fatalf("got %v, want 22", got)
	}
	// A customer expressing no preference disagrees with nothing.
	silent := []CustomerDemand{{CustomerAccountID: "ac_1", Units: 100}}
	if got := MixedStreamShare(silent, PolicyMakeToStock); got != 0 {
		t.Fatalf("got %v, want 0", got)
	}
}

// Two customers with identical volume must not swap between runs; the plan is deterministic and a recommendation shown beside it has to be too.
func TestTopCustomer_TiesBreakDeterministically(t *testing.T) {
	t.Parallel()

	customers := []CustomerDemand{
		{CustomerAccountID: "ac_zzz", CustomerName: "Zeta", Units: 500},
		{CustomerAccountID: "ac_aaa", CustomerName: "Alpha", Units: 500},
	}
	for range 50 {
		_, name, _ := topCustomer(customers)
		if name != "Alpha" {
			t.Fatalf("tie resolved to %q; the lowest id must win every time", name)
		}
	}
}

// An item nobody has ever bought has no customers to weigh, and must not be judged infeasible on a lead time of zero.
func TestRecommendPolicy_NoCustomersSkipsTheFeasibilityRule(t *testing.T) {
	t.Parallel()

	in := steadyItem()
	in.Customers = nil

	got := RecommendPolicy(in, DefaultRecommendationThresholds())

	if got.Reason == ReasonLeadTimeInfeasible {
		t.Fatal("with no customers there is no promised lead time to be infeasible against")
	}
}
