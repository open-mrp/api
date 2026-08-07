package constants

// FulfillmentRecommendationReason is the rule that decided how a SKU should be produced.
type FulfillmentRecommendationReason string

const (
	// FulfillmentRecommendationReasonLeadTimeInfeasible means customers are promised less time than production needs, so the stock has to exist before the order does.
	FulfillmentRecommendationReasonLeadTimeInfeasible FulfillmentRecommendationReason = "lead_time_infeasible"
	// FulfillmentRecommendationReasonNoRecentDemand means nothing has sold for long enough that a buffer is dead stock.
	FulfillmentRecommendationReasonNoRecentDemand FulfillmentRecommendationReason = "no_recent_demand"
	// FulfillmentRecommendationReasonSingleCustomer means effectively one customer buys it, and that customer is served to order.
	FulfillmentRecommendationReasonSingleCustomer FulfillmentRecommendationReason = "single_customer"
	// FulfillmentRecommendationReasonLumpyDemand means demand arrives rarely and in wildly different sizes.
	FulfillmentRecommendationReasonLumpyDemand FulfillmentRecommendationReason = "lumpy_demand"
	// FulfillmentRecommendationReasonSlowMovingHighValue means expensive units and few sold.
	FulfillmentRecommendationReasonSlowMovingHighValue FulfillmentRecommendationReason = "slow_moving_high_value"
	// FulfillmentRecommendationReasonSteadyDemand means demand is regular enough to forecast.
	FulfillmentRecommendationReasonSteadyDemand FulfillmentRecommendationReason = "steady_demand"
)

func (m FulfillmentRecommendationReason) IsValid() bool {
	switch m {
	case FulfillmentRecommendationReasonLeadTimeInfeasible,
		FulfillmentRecommendationReasonNoRecentDemand,
		FulfillmentRecommendationReasonSingleCustomer,
		FulfillmentRecommendationReasonLumpyDemand,
		FulfillmentRecommendationReasonSlowMovingHighValue,
		FulfillmentRecommendationReasonSteadyDemand:
		return true
	default:
		return false
	}
}

func (m FulfillmentRecommendationReason) EnumValues() []string {
	return []string{
		string(FulfillmentRecommendationReasonLeadTimeInfeasible),
		string(FulfillmentRecommendationReasonNoRecentDemand),
		string(FulfillmentRecommendationReasonSingleCustomer),
		string(FulfillmentRecommendationReasonLumpyDemand),
		string(FulfillmentRecommendationReasonSlowMovingHighValue),
		string(FulfillmentRecommendationReasonSteadyDemand),
	}
}

func (m *FulfillmentRecommendationReason) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
