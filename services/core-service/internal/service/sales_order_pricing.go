package service

import (
	"context"
	"math"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// computeSalesOrderLinePrices prices every line of a sales order at once. The volume-discount stage sums quantities across all lines that share a chosen discount, so the whole line set must be priced together.
//
// When the caller is an internal actor and provides an explicit OverrideUnitPrice for a line, that override is used verbatim. Otherwise the unit price is computed from the list price through the unit-conversion discount, base->ordered-unit conversion, volume discount, and account-price override stages (in that order of increasing precedence).
// The pricing engine is a set of free functions (not methods) so that both the
// sales-order service and the sales-order-line service can price lines the same way —
// a single line added to an existing order is priced identically to a line created
// with the order. They take the RepoFactory directly since the only state they need is
// the pricing repository.
func computeSalesOrderLinePrices(
	ctx context.Context,
	repos domain.RepoFactory,
	ownerAccountID, buyerAccountID string,
	isInternalActor bool,
	lines []domain.SalesOrderPriceLineInput,
) ([]domain.SalesOrderLinePrice, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.compute_line_prices")
	defer span.End()

	out := make([]domain.SalesOrderLinePrice, len(lines))

	// Fast path: nothing to compute.
	if len(lines) == 0 {
		return out, nil
	}

	// Load all data needed for the whole order in one repository call.
	productIDs := make([]string, 0, len(lines))
	orderedUnitIDs := make([]string, 0, len(lines))
	for _, l := range lines {
		productIDs = append(productIDs, l.ProductID)
		orderedUnitIDs = append(orderedUnitIDs, l.QuantityUnitID)
	}

	bundle, apiErr := repos.NewPricingRepo().LoadPricingBundle(ctx, domain.LoadPricingBundleParams{
		OwnerAccountID: ownerAccountID,
		BuyerAccountID: buyerAccountID,
		ProductIDs:     productIDs,
		OrderedUnitIDs: orderedUnitIDs,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for i, line := range lines {
		// Internal-actor explicit override wins outright.
		if isInternalActor && line.OverrideUnitPrice != nil {
			out[i] = domain.SalesOrderLinePrice{
				Value:             line.OverrideUnitPrice.Value,
				NumeratorUnitID:   line.OverrideUnitPrice.NumeratorUnitID,
				DenominatorUnitID: line.OverrideUnitPrice.DenominatorUnitID,
			}
			continue
		}

		price := computeUnitPrice(bundle, line, lines)
		out[i] = price
	}

	return out, nil
}

// resolveSalesOrderCreateLines resolves a set of create-line inputs against the pricing bundle in one pass: validates that each line's quantity unit belongs to the product's unit group, defaults the SKU/description from the product, derives the item and unit cost, and computes the unit price (honoring an internal actor's override when provided). All lines are resolved together so the volume-discount stage can sum quantities across the order.
func resolveSalesOrderCreateLines(
	ctx context.Context,
	repos domain.RepoFactory,
	ownerAccountID, buyerAccountID string,
	isInternalActor bool,
	lines []domain.CreateSalesOrderLineInput,
) ([]domain.ResolvedSalesOrderLine, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.resolve_create_lines")
	defer span.End()

	out := make([]domain.ResolvedSalesOrderLine, len(lines))
	if len(lines) == 0 {
		return out, nil
	}

	productIDs := make([]string, 0, len(lines))
	orderedUnitIDs := make([]string, 0, len(lines))
	priceInputs := make([]domain.SalesOrderPriceLineInput, len(lines))
	for i, l := range lines {
		productIDs = append(productIDs, l.ProductID)
		orderedUnitIDs = append(orderedUnitIDs, l.QuantityUnitID)
		priceInputs[i] = domain.SalesOrderPriceLineInput{
			ProductID:         l.ProductID,
			QuantityValue:     l.QuantityValue,
			QuantityUnitID:    l.QuantityUnitID,
			OverrideUnitPrice: l.UnitPrice,
		}
	}

	bundle, apiErr := repos.NewPricingRepo().LoadPricingBundle(ctx, domain.LoadPricingBundleParams{
		OwnerAccountID: ownerAccountID,
		BuyerAccountID: buyerAccountID,
		ProductIDs:     productIDs,
		OrderedUnitIDs: orderedUnitIDs,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for i, line := range lines {
		product := bundle.Products[line.ProductID]
		if product == nil {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Product not found.", "product_id"))
		}

		// The quantity unit must belong to the product's unit group (the product line's group when present, else the item category's).
		unitGroupID := product.CategoryUnitGroupID
		if product.ProductLineUnitGroupID != nil && *product.ProductLineUnitGroupID != "" {
			unitGroupID = *product.ProductLineUnitGroupID
		}
		groupUnits, ok := bundle.UnitGroupUnits[unitGroupID]
		if !ok {
			return nil, tracing.Trace(span, apierror.NewValidationError("The product's unit group is not configured."))
		}
		if _, ok := groupUnits[line.QuantityUnitID]; !ok {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The unit is not valid for this product.", "quantity_unit_id"))
		}

		// Unit price: internal override, else computed via the pricing algorithm.
		var price domain.SalesOrderLinePrice
		if isInternalActor && line.UnitPrice != nil {
			price = domain.SalesOrderLinePrice(*line.UnitPrice)
		} else {
			price = computeUnitPrice(bundle, priceInputs[i], priceInputs)
		}

		sku := product.SKU
		if line.ProductSKU != nil && *line.ProductSKU != "" {
			sku = *line.ProductSKU
		}
		desc := product.Description
		if line.ProductDescription != nil {
			desc = line.ProductDescription
		}

		out[i] = domain.ResolvedSalesOrderLine{
			ProductID:          line.ProductID,
			ItemID:             product.ItemID,
			ProductSKU:         sku,
			ProductDescription: desc,
			QuantityValue:      line.QuantityValue,
			QuantityUnitID:     line.QuantityUnitID,
			UnitPrice:          domain.RateValue(price),
			UnitCost: domain.RateValue{
				Value:             product.UnitCost,
				NumeratorUnitID:   product.UnitCostNumeratorUnitID,
				DenominatorUnitID: product.UnitCostDenominatorUnitID,
			},
		}
	}

	return out, nil
}

// computeUnitPrice ports the pricing algorithm for a single line.
func computeUnitPrice(
	bundle *domain.PricingBundle,
	line domain.SalesOrderPriceLineInput,
	allLines []domain.SalesOrderPriceLineInput,
) domain.SalesOrderLinePrice {
	product := bundle.Products[line.ProductID]
	if product == nil {
		// Unknown / unscoped product: nothing to price against. Return a zero price carrying the ordered unit as denominator (numerator unknown).
		return domain.SalesOrderLinePrice{
			Value:             "0",
			NumeratorUnitID:   "",
			DenominatorUnitID: line.QuantityUnitID,
		}
	}

	currencyUnitID := product.UnitValueNumeratorUnitID
	orderedUnitID := line.QuantityUnitID

	baseValue := parseDecimal(product.UnitValue)

	// --- A. Unit-conversion discount for the ordered unit -------------------
	// Group = product line's unit group when present, else the item category's.
	unitGroupID := product.CategoryUnitGroupID
	if product.ProductLineUnitGroupID != nil && *product.ProductLineUnitGroupID != "" {
		unitGroupID = *product.ProductLineUnitGroupID
	}

	discounted := baseValue
	if groupUnits, ok := bundle.UnitGroupUnits[unitGroupID]; ok {
		if ugu, ok := groupUnits[orderedUnitID]; ok {
			discountFixed := parseDecimal(ugu.DiscountFixed)
			discountPct := parseDecimal(ugu.DiscountPercentage)
			// (baseValue - discountFixed) * (1 - discountPercentage)
			discounted = baseValue.Sub(discountFixed).Mul(decimal.NewFromInt(1).Sub(discountPct))
		}
	}

	// --- B. Convert base -> ordered unit ------------------------------------
	// factor = normalizeQuantity(1, orderedUnit) / normalizeQuantity(1, baseDenominatorUnit)
	baseDenomUnit := bundle.Units[product.UnitValueDenominatorUnitID]
	orderedUnit := bundle.Units[orderedUnitID]

	converted := discounted
	denominatorUnitID := orderedUnitID
	if orderedUnit != nil && baseDenomUnit != nil {
		one := decimal.NewFromInt(1)
		normOrdered := normalizeQuantity(one, orderedUnit)
		normBase := normalizeQuantity(one, baseDenomUnit)
		if !normBase.IsZero() {
			factor := normOrdered.Div(normBase)
			converted = discounted.Mul(factor)
		}
	} else if orderedUnit == nil {
		// Ordered unit unknown: cannot convert; keep base denominator unit.
		denominatorUnitID = product.UnitValueDenominatorUnitID
	}

	// --- C. Volume discount multiplier --------------------------------------
	final := converted
	if discount := selectVolumeDiscount(bundle, product, line, allLines); discount != nil {
		multiplier := decimal.NewFromInt(1)
		summedQty := sumQuantityForDiscount(bundle, discount, allLines)
		for _, tier := range discount.Tiers {
			threshold := parseDecimal(tier.Threshold)
			if summedQty.GreaterThanOrEqual(threshold) {
				pct := parseDecimal(tier.DiscountPercentage)
				multiplier = multiplier.Mul(decimal.NewFromInt(1).Sub(pct))
			}
		}
		final = converted.Mul(multiplier)
	}

	value := round2(final)
	numeratorUnitID := currencyUnitID

	// --- D. account_price absolute override (beats everything; not rounded) --
	if override := selectAccountPrice(bundle, product); override != nil {
		value = parseDecimal(override.UnitValue)
		numeratorUnitID = override.NumeratorUnitID
		denominatorUnitID = override.DenominatorUnitID
	}

	return domain.SalesOrderLinePrice{
		Value:             value.String(),
		NumeratorUnitID:   numeratorUnitID,
		DenominatorUnitID: denominatorUnitID,
	}
}

// selectAccountPrice returns the applicable account-price override, last match wins (the bundle is ordered created_at ASC). A product with no product line never matches an account price.
func selectAccountPrice(bundle *domain.PricingBundle, product *domain.PricingProduct) *domain.PricingAccountPrice {
	if product.ProductLineID == nil || *product.ProductLineID == "" {
		return nil
	}
	productAttrs := make(map[string]struct{}, len(product.AttributeIDs))
	for _, a := range product.AttributeIDs {
		productAttrs[a] = struct{}{}
	}

	var match *domain.PricingAccountPrice
	for _, ap := range bundle.AccountPrices {
		if ap.ProductLineID != *product.ProductLineID {
			continue
		}
		// Every price attribute id must be in the product's attribute ids. (Recipient buyer-or-parent already filtered at load time; categories are ignored.)
		ok := true
		for _, attr := range ap.AttributeIDs {
			if _, found := productAttrs[attr]; !found {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		match = ap // last match wins
	}
	return match
}

// selectVolumeDiscount chooses the applicable discount for a line: customer-group matching discounts first, then the highest total multiplier among the met tiers. Returns nil if none apply.
func selectVolumeDiscount(
	bundle *domain.PricingBundle,
	product *domain.PricingProduct,
	line domain.SalesOrderPriceLineInput,
	allLines []domain.SalesOrderPriceLineInput,
) *domain.PricingVolumeDiscount {
	var best *domain.PricingVolumeDiscount
	var bestMultiplier decimal.Decimal
	var bestMatchesGroup bool

	for _, d := range bundle.VolumeDiscounts {
		if !discountMatchesProduct(d, product) {
			continue // this product is outside the discount's scope
		}
		summed := sumQuantityForDiscount(bundle, d, allLines)
		multiplier, anyTierMet := discountMultiplier(d, summed)
		if !anyTierMet {
			continue
		}
		if best == nil {
			best, bestMultiplier, bestMatchesGroup = d, multiplier, d.MatchesCustomerGroup
			continue
		}
		// Customer-group-matching discounts take precedence.
		if d.MatchesCustomerGroup != bestMatchesGroup {
			if d.MatchesCustomerGroup {
				best, bestMultiplier, bestMatchesGroup = d, multiplier, d.MatchesCustomerGroup
			}
			continue
		}
		// Within the same group-match class, the highest total multiplier wins (multiplier m = product of (1 - pct) over met tiers).
		if multiplier.GreaterThan(bestMultiplier) {
			best, bestMultiplier = d, multiplier
		}
	}
	return best
}

// discountMatchesProduct reports whether a product falls within a volume discount's scope. Each dimension (product line, item category, attributes) filters only when non-empty; an empty dimension is a wildcard. Customer-group scope is applied at load. Mirrors the legacy VolumeDiscountUtils.findApplicableDiscounts (AND across dimensions).
func discountMatchesProduct(d *domain.PricingVolumeDiscount, product *domain.PricingProduct) bool {
	if len(d.ProductLineIDs) > 0 {
		if product.ProductLineID == nil || !slices.Contains(d.ProductLineIDs, *product.ProductLineID) {
			return false
		}
	}
	if len(d.CategoryIDs) > 0 && !slices.Contains(d.CategoryIDs, product.ItemCategoryID) {
		return false
	}
	// Every required attribute must be present on the product.
	for _, a := range d.AttributeIDs {
		if !slices.Contains(product.AttributeIDs, a) {
			return false
		}
	}
	return true
}

// discountMultiplier returns the multiplicative product of (1 - pct) over all met tiers, and whether any tier was met.
func discountMultiplier(d *domain.PricingVolumeDiscount, summedQty decimal.Decimal) (decimal.Decimal, bool) {
	multiplier := decimal.NewFromInt(1)
	any := false
	for _, tier := range d.Tiers {
		threshold := parseDecimal(tier.Threshold)
		if summedQty.GreaterThanOrEqual(threshold) {
			pct := parseDecimal(tier.DiscountPercentage)
			multiplier = multiplier.Mul(decimal.NewFromInt(1).Sub(pct))
			any = true
		}
	}
	return multiplier, any
}

// sumQuantityForDiscount sums the ordered quantity across all lines, each normalized into one of the discount's acceptable units. On conversion failure the line contributes 0.
func sumQuantityForDiscount(
	bundle *domain.PricingBundle,
	discount *domain.PricingVolumeDiscount,
	allLines []domain.SalesOrderPriceLineInput,
) decimal.Decimal {
	sum := decimal.Zero
	if len(discount.AcceptableUnitIDs) == 0 {
		return sum
	}
	// Sum only lines whose product is within the discount's scope (so an LTD discount sums LTD lines across products, not the whole cart).
	for _, line := range allLines {
		product := bundle.Products[line.ProductID]
		if product == nil || !discountMatchesProduct(discount, product) {
			continue
		}
		qty := parseDecimal(line.QuantityValue)
		fromUnit := bundle.Units[line.QuantityUnitID]
		if fromUnit == nil {
			continue // conversion failure -> add 0
		}
		// Try to convert into any acceptable unit; use the first that succeeds.
		converted, ok := convertToAnyAcceptable(bundle, qty, fromUnit, discount.AcceptableUnitIDs)
		if !ok {
			continue // add 0
		}
		sum = sum.Add(converted)
	}
	return sum
}

// convertToAnyAcceptable converts value from fromUnit into the first acceptable unit that can be resolved. Returns false if none can be resolved.
func convertToAnyAcceptable(
	bundle *domain.PricingBundle,
	value decimal.Decimal,
	fromUnit *domain.PricingUnit,
	acceptableUnitIDs []string,
) (decimal.Decimal, bool) {
	for _, toID := range acceptableUnitIDs {
		toUnit := bundle.Units[toID]
		if toUnit == nil {
			continue
		}
		return convertValue(value, fromUnit, toUnit), true
	}
	return decimal.Zero, false
}

// convertValue converts a measure from one unit to another within a unit group, via the shared base measure (UnitGroupUtils-style):
//
//	base = normalizeQuantity(value, from)
//	result = inverse-normalize(base, to)
func convertValue(value decimal.Decimal, from, to *domain.PricingUnit) decimal.Decimal {
	base := normalizeQuantity(value, from)
	if to.IsBaseUnit {
		return base
	}
	ratio := ratioOf(to)
	offset := offsetOf(to)
	if ratio.IsZero() {
		return base
	}
	// base = ratio*result + offset  =>  result = (base - offset) / ratio
	return base.Sub(offset).Div(ratio)
}

// normalizeQuantity converts a quantity in unit u into the group's base measure:
//
//	u.is_base_unit ? v : (ratio_num/ratio_den)*v + (offset_num/offset_den)
func normalizeQuantity(v decimal.Decimal, u *domain.PricingUnit) decimal.Decimal {
	if u.IsBaseUnit {
		return v
	}
	return ratioOf(u).Mul(v).Add(offsetOf(u))
}

func ratioOf(u *domain.PricingUnit) decimal.Decimal {
	num := parseDecimal(u.RatioNumerator)
	den := parseDecimal(u.RatioDenominator)
	if den.IsZero() {
		return decimal.NewFromInt(1)
	}
	return num.Div(den)
}

func offsetOf(u *domain.PricingUnit) decimal.Decimal {
	num := parseDecimal(u.OffsetNumerator)
	den := parseDecimal(u.OffsetDenominator)
	if den.IsZero() {
		return decimal.Zero
	}
	return num.Div(den)
}

// parseDecimal parses a decimal string, treating empty / invalid input as zero.
func parseDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// round2 rounds to 2 decimal places using JS Math.round semantics (half toward
// +infinity): math.Floor(x*100 + 0.5) / 100. Applied only at the final step.
//
// The rounding decision is performed in float64 to match the upstream JS engine bit-for-bit (including float64 representation effects on exact half-way values). The float64 result is then re-parsed through decimal at scale 2 so the emitted string is a clean 2-decimal value with no float artifacts.
func round2(x decimal.Decimal) decimal.Decimal {
	f, _ := x.Float64()
	rounded := math.Floor(f*100+0.5) / 100
	return decimal.NewFromFloat(rounded).Round(2)
}
