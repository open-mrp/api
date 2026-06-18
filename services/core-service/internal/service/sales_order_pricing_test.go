package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/augno/api/services/core-service/internal/domain"
)

// These tests pin the server-side sales-order line pricing algorithm (a faithful port
// of the dormant Dashboard OrderLine pricing logic). The algorithm, in increasing
// precedence: list price -> unit-conversion discount (fixed then percent) ->
// base->ordered unit conversion -> multiplicative volume-tier discount (quantity summed
// across all lines sharing the discount) -> absolute account_price override. Rounding to
// cents is JS Math.round (half toward +infinity), applied once before the override.

// --- round2 (JS Math.round half-up semantics) -------------------------------

func TestRound2(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"10", "10"},
		{"10.1", "10.1"},
		{"10.124", "10.12"},   // rounds down
		{"10.125", "10.13"},   // exact half -> up (1012.5 is representable)
		{"10.126", "10.13"},   // rounds up
		{"-10.125", "-10.12"}, // half toward +infinity (not away from zero)
		{"1.005", "1"},        // float repr of 1.005 is < 1.005 -> floors to 1.00 (matches JS)
		{"0", "0"},
	}
	for _, c := range cases {
		got := round2(decimal.RequireFromString(c.in))
		assert.Truef(t, got.Equal(decimal.RequireFromString(c.want)),
			"round2(%s) = %s, want %s", c.in, got.String(), c.want)
	}
}

// --- normalizeQuantity / convertValue (unit ratio math) ---------------------

func TestNormalizeQuantity(t *testing.T) {
	t.Parallel()
	base := &domain.PricingUnit{ID: "ea", IsBaseUnit: true}
	assert.True(t, normalizeQuantity(decimal.NewFromInt(5), base).Equal(decimal.NewFromInt(5)),
		"base unit normalizes to itself")

	// dozen = 12 each, no offset.
	dozen := &domain.PricingUnit{ID: "dz", RatioNumerator: "12", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"}
	assert.True(t, normalizeQuantity(decimal.NewFromInt(1), dozen).Equal(decimal.NewFromInt(12)),
		"1 dozen normalizes to 12 base units")

	// unit with an offset (e.g. a temperature-like scale): ratio 2/1, offset 3/1.
	withOffset := &domain.PricingUnit{ID: "off", RatioNumerator: "2", RatioDenominator: "1", OffsetNumerator: "3", OffsetDenominator: "1"}
	assert.True(t, normalizeQuantity(decimal.NewFromInt(4), withOffset).Equal(decimal.NewFromInt(11)),
		"ratio*v + offset = 2*4 + 3 = 11")
}

func TestConvertValue_BetweenUnits(t *testing.T) {
	t.Parallel()
	base := &domain.PricingUnit{ID: "ea", IsBaseUnit: true}
	dozen := &domain.PricingUnit{ID: "dz", RatioNumerator: "12", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"}

	// 2 dozen -> 24 each.
	assert.True(t, convertValue(decimal.NewFromInt(2), dozen, base).Equal(decimal.NewFromInt(24)))
	// 24 each -> 2 dozen.
	assert.True(t, convertValue(decimal.NewFromInt(24), base, dozen).Equal(decimal.NewFromInt(2)))
}

// --- computeUnitPrice: the full algorithm -----------------------------------

// pricingTestBundle builds a bundle for a single product "p" priced in USD per "ea",
// with an "ea" base unit and a "dz" (dozen) unit, in unit group "ug".
func pricingTestBundle(product *domain.PricingProduct, groupUnits map[string]*domain.PricingUnitGroupUnit, accountPrices []*domain.PricingAccountPrice, discounts []*domain.PricingVolumeDiscount) *domain.PricingBundle {
	return &domain.PricingBundle{
		Products: map[string]*domain.PricingProduct{"p": product},
		Units: map[string]*domain.PricingUnit{
			"ea": {ID: "ea", IsBaseUnit: true},
			"dz": {ID: "dz", RatioNumerator: "12", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"},
		},
		UnitGroupUnits:  map[string]map[string]*domain.PricingUnitGroupUnit{"ug": groupUnits},
		AccountPrices:   accountPrices,
		VolumeDiscounts: discounts,
	}
}

func basePricingProduct(unitValue string, productLineID *string, attrs []string) *domain.PricingProduct {
	return &domain.PricingProduct{
		ProductID:                  "p",
		ItemID:                     "it",
		SKU:                        "SKU",
		UnitCost:                   "1",
		UnitCostNumeratorUnitID:    "usd",
		UnitCostDenominatorUnitID:  "ea",
		UnitValue:                  unitValue,
		UnitValueNumeratorUnitID:   "usd",
		UnitValueDenominatorUnitID: "ea",
		ProductLineID:              productLineID,
		CategoryUnitGroupID:        "ug",
		AttributeIDs:               attrs,
	}
}

func line(qty, unitID string) domain.SalesOrderPriceLineInput {
	return domain.SalesOrderPriceLineInput{ProductID: "p", QuantityValue: qty, QuantityUnitID: unitID}
}

func assertPrice(t *testing.T, got domain.SalesOrderLinePrice, wantValue, wantNum, wantDen string) {
	t.Helper()
	assert.Truef(t, decimal.RequireFromString(got.Value).Equal(decimal.RequireFromString(wantValue)),
		"unit price value = %s, want %s", got.Value, wantValue)
	assert.Equal(t, wantNum, got.NumeratorUnitID, "numerator unit")
	assert.Equal(t, wantDen, got.DenominatorUnitID, "denominator unit")
}

func TestComputeUnitPrice_ListPriceNoDiscounts(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	product := basePricingProduct("10", nil, nil)
	bundle := pricingTestBundle(product, map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}, nil, nil)

	got := svc.computeUnitPrice(bundle, line("1", "ea"), []domain.SalesOrderPriceLineInput{line("1", "ea")})
	assertPrice(t, got, "10", "usd", "ea")
}

func TestComputeUnitPrice_UnitConversionDiscount(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	product := basePricingProduct("10", nil, nil)
	// (10 - 1) * (1 - 0.1) = 8.1
	groupUnits := map[string]*domain.PricingUnitGroupUnit{
		"ea": {UnitGroupID: "ug", UnitID: "ea", DiscountFixed: "1", DiscountPercentage: "0.1"},
	}
	bundle := pricingTestBundle(product, groupUnits, nil, nil)

	got := svc.computeUnitPrice(bundle, line("1", "ea"), []domain.SalesOrderPriceLineInput{line("1", "ea")})
	assertPrice(t, got, "8.1", "usd", "ea")
}

func TestComputeUnitPrice_BaseToOrderedUnitConversion(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	product := basePricingProduct("10", nil, nil) // $10 per each
	// Ordered in dozens: price = 10 * 12 = 120 per dozen. No unit-conversion discount for "dz".
	groupUnits := map[string]*domain.PricingUnitGroupUnit{
		"ea": {UnitGroupID: "ug", UnitID: "ea"},
		"dz": {UnitGroupID: "ug", UnitID: "dz"},
	}
	bundle := pricingTestBundle(product, groupUnits, nil, nil)

	got := svc.computeUnitPrice(bundle, line("1", "dz"), []domain.SalesOrderPriceLineInput{line("1", "dz")})
	assertPrice(t, got, "120", "usd", "dz")
}

func TestComputeUnitPrice_VolumeDiscount_SingleAndMultipleTiers(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	product := basePricingProduct("10", nil, nil)
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}

	// Single tier met: qty 10 >= 5 -> 1 - 0.2 = 0.8 -> 10*0.8 = 8.
	single := []*domain.PricingVolumeDiscount{{
		ID: "vd", AcceptableUnitIDs: []string{"ea"},
		Tiers: []domain.PricingVolumeDiscountTier{{Threshold: "5", DiscountPercentage: "0.2"}},
	}}
	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, nil, single), line("10", "ea"), []domain.SalesOrderPriceLineInput{line("10", "ea")})
	assertPrice(t, got, "8", "usd", "ea")

	// Multiple tiers met: qty 10 >= both 5 and 10 -> 0.9 * 0.8 = 0.72 -> 10*0.72 = 7.2.
	multi := []*domain.PricingVolumeDiscount{{
		ID: "vd", AcceptableUnitIDs: []string{"ea"},
		Tiers: []domain.PricingVolumeDiscountTier{
			{Threshold: "5", DiscountPercentage: "0.1"},
			{Threshold: "10", DiscountPercentage: "0.2"},
		},
	}}
	got = svc.computeUnitPrice(pricingTestBundle(product, groupUnits, nil, multi), line("10", "ea"), []domain.SalesOrderPriceLineInput{line("10", "ea")})
	assertPrice(t, got, "7.2", "usd", "ea")
}

func TestComputeUnitPrice_VolumeDiscount_ThresholdNotMet(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	product := basePricingProduct("10", nil, nil)
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}
	discounts := []*domain.PricingVolumeDiscount{{
		ID: "vd", AcceptableUnitIDs: []string{"ea"},
		Tiers: []domain.PricingVolumeDiscountTier{{Threshold: "100", DiscountPercentage: "0.5"}},
	}}

	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, nil, discounts), line("10", "ea"), []domain.SalesOrderPriceLineInput{line("10", "ea")})
	assertPrice(t, got, "10", "usd", "ea") // no tier met -> list price
}

func TestComputeUnitPrice_VolumeDiscount_SumsAcrossLines(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	product := basePricingProduct("10", nil, nil)
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}
	// Threshold 10; each line is 6 (< 10) but the two lines sum to 12 (>= 10) -> discount applies.
	discounts := []*domain.PricingVolumeDiscount{{
		ID: "vd", AcceptableUnitIDs: []string{"ea"},
		Tiers: []domain.PricingVolumeDiscountTier{{Threshold: "10", DiscountPercentage: "0.25"}},
	}}
	allLines := []domain.SalesOrderPriceLineInput{line("6", "ea"), line("6", "ea")}

	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, nil, discounts), allLines[0], allLines)
	assertPrice(t, got, "7.5", "usd", "ea") // 10 * (1 - 0.25)
}

// Mirrors the production case: a volume discount scoped to a product line applies to (and
// sums quantity across) only the matching products, ignoring out-of-scope lines.
func TestComputeUnitPrice_VolumeDiscount_ScopedAndSummedAcrossMatchingProducts(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}

	mk := func(id, lineID string) *domain.PricingProduct {
		pl := lineID
		return &domain.PricingProduct{
			ProductID: id, ItemID: "it_" + id, UnitValue: "100",
			UnitValueNumeratorUnitID: "usd", UnitValueDenominatorUnitID: "ea",
			UnitCost: "1", UnitCostNumeratorUnitID: "usd", UnitCostDenominatorUnitID: "ea",
			ProductLineID: &pl, ProductLineUnitGroupID: strptr("ug"), CategoryUnitGroupID: "ug",
		}
	}
	bundle := &domain.PricingBundle{
		Products: map[string]*domain.PricingProduct{
			"p_ltd1": mk("p_ltd1", "ltd"), "p_ltd2": mk("p_ltd2", "ltd"), "p_other": mk("p_other", "other"),
		},
		Units:          map[string]*domain.PricingUnit{"ea": {ID: "ea", IsBaseUnit: true}},
		UnitGroupUnits: map[string]map[string]*domain.PricingUnitGroupUnit{"ug": {"ea": {UnitGroupID: "ug", UnitID: "ea"}}},
		VolumeDiscounts: []*domain.PricingVolumeDiscount{{
			ID: "vd", AcceptableUnitIDs: []string{"ea"}, ProductLineIDs: []string{"ltd"},
			Tiers: []domain.PricingVolumeDiscountTier{{Threshold: "10", DiscountPercentage: "0.2"}},
		}},
	}
	// 6 of one LTD product + 5 of another (sum 11 ≥ 10) + a huge out-of-scope line that
	// must NOT inflate the LTD sum.
	allLines := []domain.SalesOrderPriceLineInput{
		{ProductID: "p_ltd1", QuantityValue: "6", QuantityUnitID: "ea"},
		{ProductID: "p_ltd2", QuantityValue: "5", QuantityUnitID: "ea"},
		{ProductID: "p_other", QuantityValue: "100", QuantityUnitID: "ea"},
	}
	// Both LTD products get 20% off (100 → 80) from the summed 11; the out-of-scope product
	// is not in any discount → list price 100.
	assertPrice(t, svc.computeUnitPrice(bundle, allLines[0], allLines), "80", "usd", "ea")
	assertPrice(t, svc.computeUnitPrice(bundle, allLines[1], allLines), "80", "usd", "ea")
	assertPrice(t, svc.computeUnitPrice(bundle, allLines[2], allLines), "100", "usd", "ea")
}

func TestComputeUnitPrice_AccountPriceOverride_BeatsEverything(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	pl := "pl"
	product := basePricingProduct("10", &pl, nil)
	product.ProductLineUnitGroupID = strptr("ug")
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}
	// A volume discount AND an account price both apply; the account price (absolute) wins.
	discounts := []*domain.PricingVolumeDiscount{{
		ID: "vd", AcceptableUnitIDs: []string{"ea"},
		Tiers: []domain.PricingVolumeDiscountTier{{Threshold: "1", DiscountPercentage: "0.5"}},
	}}
	accountPrices := []*domain.PricingAccountPrice{{
		ID: "ap", ProductLineID: "pl", UnitValue: "3.5", NumeratorUnitID: "usd", DenominatorUnitID: "ea",
	}}

	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, accountPrices, discounts), line("10", "ea"), []domain.SalesOrderPriceLineInput{line("10", "ea")})
	assertPrice(t, got, "3.5", "usd", "ea") // override, not 10*0.5
}

func TestComputeUnitPrice_AccountPrice_MatchingRules(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}
	l := line("1", "ea")
	all := []domain.SalesOrderPriceLineInput{l}

	// (a) Product line mismatch -> no override (list price 10).
	pl := "pl"
	product := basePricingProduct("10", &pl, nil)
	apWrongLine := []*domain.PricingAccountPrice{{ID: "ap", ProductLineID: "other", UnitValue: "3", NumeratorUnitID: "usd", DenominatorUnitID: "ea"}}
	assertPrice(t, svc.computeUnitPrice(pricingTestBundle(product, groupUnits, apWrongLine, nil), l, all), "10", "usd", "ea")

	// (b) Attribute not a subset of the product's -> no override.
	productNoAttrs := basePricingProduct("10", &pl, []string{"a1"})
	apNeedsAttrs := []*domain.PricingAccountPrice{{ID: "ap", ProductLineID: "pl", UnitValue: "3", NumeratorUnitID: "usd", DenominatorUnitID: "ea", AttributeIDs: []string{"a1", "a2"}}}
	assertPrice(t, svc.computeUnitPrice(pricingTestBundle(productNoAttrs, groupUnits, apNeedsAttrs, nil), l, all), "10", "usd", "ea")

	// (c) Attributes are a subset -> override applies.
	productWithAttrs := basePricingProduct("10", &pl, []string{"a1", "a2", "a3"})
	apSubset := []*domain.PricingAccountPrice{{ID: "ap", ProductLineID: "pl", UnitValue: "3", NumeratorUnitID: "usd", DenominatorUnitID: "ea", AttributeIDs: []string{"a1", "a2"}}}
	assertPrice(t, svc.computeUnitPrice(pricingTestBundle(productWithAttrs, groupUnits, apSubset, nil), l, all), "3", "usd", "ea")

	// (d) Product with NO product line never matches an account price.
	productNoLine := basePricingProduct("10", nil, nil)
	apAnyLine := []*domain.PricingAccountPrice{{ID: "ap", ProductLineID: "pl", UnitValue: "3", NumeratorUnitID: "usd", DenominatorUnitID: "ea"}}
	assertPrice(t, svc.computeUnitPrice(pricingTestBundle(productNoLine, groupUnits, apAnyLine, nil), l, all), "10", "usd", "ea")

	// (e) Product with NO attributes must NOT match an attribute-gated price -> list price.
	// This is exactly the state the _item_attributes A/B column bug produced for every
	// product (attributes silently failed to load), which made gated contracts vanish and
	// orders fall back to list price. The matching logic is correct; the e2e
	// TestQuoteSalesOrderPrices_AttributeGateDiscriminatesInOneBatch guards that attributes
	// actually load from the DB.
	productNoAttr := basePricingProduct("10", &pl, nil)
	apGated := []*domain.PricingAccountPrice{{ID: "ap", ProductLineID: "pl", UnitValue: "3", NumeratorUnitID: "usd", DenominatorUnitID: "ea", AttributeIDs: []string{"beige"}}}
	assertPrice(t, svc.computeUnitPrice(pricingTestBundle(productNoAttr, groupUnits, apGated, nil), l, all), "10", "usd", "ea")
}

// TestComputeUnitPrice_AccountPrice_DisambiguatesByAttribute mirrors the real-world data
// that exposed the attribute-load bug: a customer has several contracted prices on one
// product line, distinguished ONLY by attribute. Exactly one must apply.
func TestComputeUnitPrice_AccountPrice_DisambiguatesByAttribute(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	pl := "pl"
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}
	l := line("1", "ea")
	all := []domain.SalesOrderPriceLineInput{l}

	// Product carries {large}; three prices gated by {small}, {beige}, {large} respectively.
	product := basePricingProduct("99", &pl, []string{"large"})
	accountPrices := []*domain.PricingAccountPrice{
		{ID: "ap_small", ProductLineID: "pl", UnitValue: "304.92", NumeratorUnitID: "usd", DenominatorUnitID: "ea", AttributeIDs: []string{"small"}},
		{ID: "ap_beige", ProductLineID: "pl", UnitValue: "144", NumeratorUnitID: "usd", DenominatorUnitID: "ea", AttributeIDs: []string{"beige"}},
		{ID: "ap_large", ProductLineID: "pl", UnitValue: "170.40", NumeratorUnitID: "usd", DenominatorUnitID: "ea", AttributeIDs: []string{"large"}},
	}
	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, accountPrices, nil), l, all)
	assertPrice(t, got, "170.40", "usd", "ea") // only the {large}-gated price matches
}

func TestComputeUnitPrice_LastAccountPriceWins(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	pl := "pl"
	product := basePricingProduct("10", &pl, nil)
	groupUnits := map[string]*domain.PricingUnitGroupUnit{"ea": {UnitGroupID: "ug", UnitID: "ea"}}
	// Bundle is created_at ASC; the last applicable match wins.
	accountPrices := []*domain.PricingAccountPrice{
		{ID: "ap1", ProductLineID: "pl", UnitValue: "4", NumeratorUnitID: "usd", DenominatorUnitID: "ea"},
		{ID: "ap2", ProductLineID: "pl", UnitValue: "5", NumeratorUnitID: "usd", DenominatorUnitID: "ea"},
	}
	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, accountPrices, nil), line("1", "ea"), []domain.SalesOrderPriceLineInput{line("1", "ea")})
	assertPrice(t, got, "5", "usd", "ea")
}

func TestComputeUnitPrice_UnknownProduct(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	bundle := &domain.PricingBundle{Products: map[string]*domain.PricingProduct{}}
	got := svc.computeUnitPrice(bundle, line("1", "ea"), []domain.SalesOrderPriceLineInput{line("1", "ea")})
	assertPrice(t, got, "0", "", "ea")
}

func TestComputeUnitPrice_FullPrecedence(t *testing.T) {
	t.Parallel()
	svc := &salesOrderSvcImpl{}
	pl := "pl"
	// list 20, unit-conversion discount (fixed 2, 10%), ordered in dozens (x12), volume 0.5.
	product := basePricingProduct("20", &pl, nil)
	product.ProductLineUnitGroupID = strptr("ug")
	groupUnits := map[string]*domain.PricingUnitGroupUnit{
		"ea": {UnitGroupID: "ug", UnitID: "ea"},
		"dz": {UnitGroupID: "ug", UnitID: "dz", DiscountFixed: "2", DiscountPercentage: "0.1"},
	}
	discounts := []*domain.PricingVolumeDiscount{{
		ID: "vd", AcceptableUnitIDs: []string{"dz"},
		Tiers: []domain.PricingVolumeDiscountTier{{Threshold: "1", DiscountPercentage: "0.5"}},
	}}

	// Without an account price: (20-2)*0.9 = 16.2; convert to dozen x12 = 194.4; volume *0.5 = 97.2.
	got := svc.computeUnitPrice(pricingTestBundle(product, groupUnits, nil, discounts), line("1", "dz"), []domain.SalesOrderPriceLineInput{line("1", "dz")})
	assertPrice(t, got, "97.2", "usd", "dz")

	// With an account price: it overrides the whole chain.
	accountPrices := []*domain.PricingAccountPrice{{ID: "ap", ProductLineID: "pl", UnitValue: "50", NumeratorUnitID: "usd", DenominatorUnitID: "dz"}}
	got = svc.computeUnitPrice(pricingTestBundle(product, groupUnits, accountPrices, discounts), line("1", "dz"), []domain.SalesOrderPriceLineInput{line("1", "dz")})
	assertPrice(t, got, "50", "usd", "dz")
}

func strptr(s string) *string { return &s }
