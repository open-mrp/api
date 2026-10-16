package service

import (
	"slices"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

var hundred = decimal.NewFromInt(100)

// ruleSpecificity ranks how narrowly a rule's scope names items: an item beats attributes or a category, which beat a
// product line, which beats any item.
func ruleSpecificity(scope domain.PriceListRuleScope) int {
	switch {
	case len(scope.ItemIDs) > 0:
		return 3
	case len(scope.AttributeIDs) > 0 || len(scope.CategoryIDs) > 0:
		return 2
	case len(scope.ProductLineIDs) > 0:
		return 1
	default:
		return 0
	}
}

// ruleMatchesProduct reports whether a product is in a rule's scope: every non-empty dimension must match, and the
// product must have every listed attribute.
func ruleMatchesProduct(scope domain.PriceListRuleScope, product *domain.PricingProduct) bool {
	if len(scope.ItemIDs) > 0 && !slices.Contains(scope.ItemIDs, product.ItemID) {
		return false
	}
	if len(scope.ProductLineIDs) > 0 && (product.ProductLineID == nil || !slices.Contains(scope.ProductLineIDs, *product.ProductLineID)) {
		return false
	}
	if len(scope.CategoryIDs) > 0 && !slices.Contains(scope.CategoryIDs, product.ItemCategoryID) {
		return false
	}
	for _, a := range scope.AttributeIDs {
		if !slices.Contains(product.AttributeIDs, a) {
			return false
		}
	}
	return true
}

// rulePrecedes reports whether rule a wins over b for one line: more specific scope, then the list's source (the
// customer's own, then assigned to it, its parent's, its group's, everyone's), then the newer rule.
func rulePrecedes(a, b *domain.PricingRule) bool {
	sa, sb := ruleSpecificity(a.Scope), ruleSpecificity(b.Scope)
	if sa != sb {
		return sa > sb
	}
	if a.Source != b.Source {
		return a.Source < b.Source
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return a.ID > b.ID
}

// tierFor is the highest tier of a tiered rule the order reaches: the quantity across every line in the rule's scope,
// counted in the rule's unit. Nil when no tier is reached.
func tierFor(bundle *domain.PricingBundle, rule *domain.PricingRule, allLines []domain.SalesOrderPriceLineInput) *domain.PriceListTier {
	toUnit := bundle.Units[rule.Price.UnitID]
	if toUnit == nil {
		return nil
	}
	total := decimal.Zero
	for _, line := range allLines {
		product := bundle.Products[line.ProductID]
		if product == nil || !ruleMatchesProduct(rule.Scope, product) {
			continue
		}
		fromUnit := bundle.Units[line.QuantityUnitID]
		if fromUnit == nil {
			continue
		}
		total = total.Add(convertValue(parseDecimal(line.QuantityValue), fromUnit, toUnit))
	}
	var reached *domain.PriceListTier
	for i := range rule.Price.Tiers {
		tier := &rule.Price.Tiers[i]
		if total.GreaterThanOrEqual(parseDecimal(tier.MinQuantity)) {
			reached = tier
		}
	}
	return reached
}

// selectPricingRule is the one rule that prices a line, or nil when none applies. A tiered rule whose lowest tier the
// order doesn't reach doesn't apply, so the next rule does.
func selectPricingRule(bundle *domain.PricingBundle, product *domain.PricingProduct, allLines []domain.SalesOrderPriceLineInput) (*domain.PricingRule, *domain.PriceListTier) {
	var best *domain.PricingRule
	var bestTier *domain.PriceListTier
	for _, rule := range bundle.Rules {
		if !ruleMatchesProduct(rule.Scope, product) {
			continue
		}
		if best != nil && !rulePrecedes(rule, best) {
			continue
		}
		var tier *domain.PriceListTier
		if rule.Price.Type == constants.PriceListPriceTypeTiered {
			if tier = tierFor(bundle, rule, allLines); tier == nil {
				continue
			}
		}
		best, bestTier = rule, tier
	}
	return best, bestTier
}

// computeListUnitPrice prices a line from price lists: the item's base sales price, replaced or scaled by the one
// winning rule, converted to the ordered unit, less the order unit's discount. A fixed price that needs no conversion
// or discount is kept exactly; anything computed rounds to the cent.
func computeListUnitPrice(bundle *domain.PricingBundle, product *domain.PricingProduct, line domain.SalesOrderPriceLineInput,
	allLines []domain.SalesOrderPriceLineInput) domain.SalesOrderLinePrice {
	base := parseDecimal(product.UnitValue)
	price, perUnitID := base, product.UnitValueDenominatorUnitID
	exact := false

	rule, tier := selectPricingRule(bundle, product, allLines)
	set := func(unitPrice *domain.UnitPrice, percentOfBase *string) {
		switch {
		case unitPrice != nil:
			price, perUnitID, exact = parseDecimal(unitPrice.Amount), unitPrice.PerUnitID, true
		case percentOfBase != nil:
			price = base.Mul(parseDecimal(*percentOfBase)).Div(hundred)
		}
	}
	if rule != nil {
		switch rule.Price.Type {
		case constants.PriceListPriceTypeFixed:
			set(rule.Price.UnitPrice, nil)
		case constants.PriceListPriceTypePercentOfBase:
			set(nil, rule.Price.PercentOfBase)
		case constants.PriceListPriceTypeTiered:
			set(tier.UnitPrice, tier.PercentOfBase)
		}
	}

	units := bundle.ItemUnits[product.ItemID]
	orderUnit := units.OrderUnit(line.QuantityUnitID)
	converted, denominatorUnitID := convertPriceToOrderedUnit(bundle, units, orderUnit, price, perUnitID, line.QuantityUnitID)
	final := orderUnit.ApplyDiscount(converted)

	value := final
	if !exact || !final.Equal(price) || denominatorUnitID != perUnitID {
		value = round2(final)
	}
	return domain.SalesOrderLinePrice{
		Value:             value.String(),
		NumeratorUnitID:   product.UnitValueNumeratorUnitID,
		DenominatorUnitID: denominatorUnitID,
	}
}

// convertPriceToOrderedUnit restates a price per perUnitID as a price per the ordered unit. An order unit that holds
// its own quantity of stocking units for this item measures that many stocking units instead of its ratio. When a unit
// is unknown the price stays per perUnitID.
func convertPriceToOrderedUnit(bundle *domain.PricingBundle, units *domain.ItemUnits, orderUnit *domain.ItemOrderUnit,
	price decimal.Decimal, perUnitID, orderedUnitID string) (decimal.Decimal, string) {
	perUnit := bundle.Units[perUnitID]
	orderedUnit := bundle.Units[orderedUnitID]
	if orderedUnit == nil || perUnit == nil {
		return price, perUnitID
	}
	if orderedUnitID == perUnitID && (orderUnit == nil || orderUnit.StockingQuantity == nil) {
		return price, orderedUnitID
	}
	one := decimal.NewFromInt(1)
	normOrdered := normalizeQuantity(one, orderedUnit)
	if orderUnit != nil && orderUnit.StockingQuantity != nil && units != nil {
		if stocking := bundle.Units[units.StockingUnitID]; stocking != nil {
			normOrdered = normalizeQuantity(parseDecimal(*orderUnit.StockingQuantity), stocking)
		}
	}
	normPer := normalizeQuantity(one, perUnit)
	if normPer.IsZero() {
		return price, orderedUnitID
	}
	return price.Mul(normOrdered.Div(normPer)), orderedUnitID
}
