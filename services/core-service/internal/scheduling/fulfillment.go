package scheduling

import "github.com/augno/api/shared/constants"

// Fulfillment policies, aliased from the shared enum so the engine cannot drift from the API contract.
const (
	PolicyMakeToStock = string(constants.FulfillmentPolicyMakeToStock)
	PolicyMakeToOrder = string(constants.FulfillmentPolicyMakeToOrder)
)

// Policy sources, aliased from the shared enum.
const (
	PolicySourceItem           = string(constants.FulfillmentPolicySourceItem)
	PolicySourceProductLine    = string(constants.FulfillmentPolicySourceProductLine)
	PolicySourceAccountDefault = string(constants.FulfillmentPolicySourceAccountDefault)
)

// FulfillmentResolution is a SKU's policy and the rule that produced it.
type FulfillmentResolution struct {
	Policy string
	Source string
}

// PolicyResolutionInput is everything the chain needs, gathered once.
//
// Customer and account-group policies are deliberately absent. Policy is resolved per SKU, and a SKU sold to both a stocking distributor and a contract customer cannot take its policy from whichever customer is being looked at — those settings drive the recommendation instead, which is a different question asked at a different time.
type PolicyResolutionInput struct {
	// ItemOverrides are explicit per-item policies.
	ItemOverrides map[string]string
	// ProductLineByItem maps an item to the line it sells under. Intermediate items are absent.
	ProductLineByItem map[string]string
	// PolicyByProductLine is the default configured on each line.
	PolicyByProductLine map[string]string
	// AccountDefault is the last resort; empty means make-to-stock.
	AccountDefault string
	// DownstreamByItem is what each planned item becomes, so an intermediate can inherit the policy of the finished goods it turns into.
	DownstreamByItem map[string][]FinishedGood
}

// ResolveFulfillmentPolicy decides how one item is produced.
//
// The chain, most specific first: an explicit item override, the item's own product line, the lines of what it becomes, then the account default.
//
// The downstream step exists for the same reason it exists for lot sizes: greige is not sold, carries no product line of its own, and has to inherit from the finished goods it becomes. It differs in one way, and the difference is the whole design. A greige item is built to order only when **every** finished good it feeds is built to order — one stocked sibling means the greige is still forecast-driven, because that sibling's buffer has to come from somewhere. This is what makes a two-value enum sufficient: the pooled greige buffer shrinks in proportion to how much of its family went make-to-order, with no third policy to name.
func ResolveFulfillmentPolicy(itemID string, in PolicyResolutionInput) FulfillmentResolution {
	accountDefault := in.AccountDefault
	if accountDefault == "" {
		accountDefault = PolicyMakeToStock
	}

	if policy, ok := in.ItemOverrides[itemID]; ok && policy != "" {
		return FulfillmentResolution{Policy: policy, Source: PolicySourceItem}
	}

	if lineID, ok := in.ProductLineByItem[itemID]; ok {
		if policy, ok := in.PolicyByProductLine[lineID]; ok && policy != "" {
			return FulfillmentResolution{Policy: policy, Source: PolicySourceProductLine}
		}
	}

	if policy, ok := inheritedPolicy(itemID, in, accountDefault); ok {
		return policy
	}

	return FulfillmentResolution{Policy: accountDefault, Source: PolicySourceAccountDefault}
}

// inheritedPolicy takes an intermediate item's policy from the finished goods it becomes.
//
// Unanimity is required, and deliberately so: a greige feeding one make-to-order SKU and three stocked ones still has to hold a buffer for the three. Requiring every descendant to agree means the answer degrades safely — the moment one sibling needs stock, the whole family is planned to stock.
func inheritedPolicy(itemID string, in PolicyResolutionInput, accountDefault string) (FulfillmentResolution, bool) {
	downstream := in.DownstreamByItem[itemID]
	if len(downstream) == 0 {
		return FulfillmentResolution{}, false
	}

	sawOne := false
	for _, finished := range downstream {
		policy := descendantPolicy(finished, in, accountDefault)
		if policy != PolicyMakeToOrder {
			return FulfillmentResolution{}, false
		}
		sawOne = true
	}
	if !sawOne {
		return FulfillmentResolution{}, false
	}
	return FulfillmentResolution{Policy: PolicyMakeToOrder, Source: PolicySourceProductLine}, true
}

// descendantPolicy resolves one finished good's own policy, without recursing back through the downstream step. A finished good is sold, so it always has an item override, a product line, or the account default to fall back on.
func descendantPolicy(finished FinishedGood, in PolicyResolutionInput, accountDefault string) string {
	if policy, ok := in.ItemOverrides[finished.ItemID]; ok && policy != "" {
		return policy
	}
	lineID := finished.ProductLineID
	if lineID == "" {
		lineID = in.ProductLineByItem[finished.ItemID]
	}
	if policy, ok := in.PolicyByProductLine[lineID]; ok && policy != "" {
		return policy
	}
	return accountDefault
}
