package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

// A pair is the stocking unit; a carton holds 12 pairs.
func listBundle(rules ...*domain.PricingRule) *domain.PricingBundle {
	line := "pl_boots"
	return &domain.PricingBundle{
		UsesPriceLists: true,
		Products: map[string]*domain.PricingProduct{
			"pr_boot": {
				ProductID: "pr_boot", ItemID: "it_boot", ProductLineID: &line, ItemCategoryID: "ic_footwear",
				AttributeIDs: []string{"at_black"}, UnitValue: "20", UnitValueNumeratorUnitID: "un_usd", UnitValueDenominatorUnitID: "un_pair",
			},
			"pr_sock": {
				ProductID: "pr_sock", ItemID: "it_sock", ItemCategoryID: "ic_footwear",
				UnitValue: "4", UnitValueNumeratorUnitID: "un_usd", UnitValueDenominatorUnitID: "un_pair",
			},
		},
		Units: map[string]*domain.PricingUnit{
			"un_pair":   {ID: "un_pair", RatioNumerator: "1", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1", IsBaseUnit: true},
			"un_carton": {ID: "un_carton", RatioNumerator: "12", RatioDenominator: "1", OffsetNumerator: "0", OffsetDenominator: "1"},
		},
		ItemUnits: map[string]*domain.ItemUnits{
			"it_boot": {StockingUnitID: "un_pair", OrderUnits: []domain.ItemOrderUnit{{UnitID: "un_carton"}}},
			"it_sock": {StockingUnitID: "un_pair"},
		},
		Rules: rules,
	}
}

func pct(v string) *string { return &v }

func rule(id string, source domain.PriceListSource, created time.Time, scope domain.PriceListRuleScope, price domain.PriceListPrice) *domain.PricingRule {
	return &domain.PricingRule{
		PriceListRule: &domain.PriceListRule{ID: id, Scope: scope, Price: price, CreatedAt: created},
		Source:        source,
	}
}

func priceLine(qty, unit string) domain.SalesOrderPriceLineInput {
	return domain.SalesOrderPriceLineInput{ProductID: "pr_boot", QuantityValue: qty, QuantityUnitID: unit}
}

func TestListPricing_NoRuleIsTheBasePriceConverted(t *testing.T) {
	b := listBundle()
	line := priceLine("2", "un_carton")
	got := computeUnitPrice(b, line, []domain.SalesOrderPriceLineInput{line})
	assert.Equal(t, "240", got.Value)
	assert.Equal(t, "un_carton", got.DenominatorUnitID)
}

func TestListPricing_MostSpecificRuleWins(t *testing.T) {
	now := time.Now()
	b := listBundle(
		rule("r_any", domain.PriceListSourceOwn, now, domain.PriceListRuleScope{}, domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("50")}),
		rule("r_line", domain.PriceListSourceGroup, now.Add(-time.Hour), domain.PriceListRuleScope{ProductLineIDs: []string{"pl_boots"}},
			domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("90")}),
		rule("r_attr", domain.PriceListSourceEveryone, now.Add(-2*time.Hour), domain.PriceListRuleScope{AttributeIDs: []string{"at_black"}},
			domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("80")}),
	)
	line := priceLine("1", "un_pair")
	got := computeUnitPrice(b, line, []domain.SalesOrderPriceLineInput{line})
	assert.Equal(t, "16", got.Value, "the attribute rule is more specific than the line rule and the any-item rule")
}

func TestListPricing_TiesGoToTheCustomersOwnListThenTheNewerRule(t *testing.T) {
	now := time.Now()
	scope := domain.PriceListRuleScope{ProductLineIDs: []string{"pl_boots"}}
	b := listBundle(
		rule("r_group", domain.PriceListSourceGroup, now, scope, domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("50")}),
		rule("r_direct_old", domain.PriceListSourceDirect, now.Add(-time.Hour), scope, domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("70")}),
		rule("r_direct_new", domain.PriceListSourceDirect, now, scope, domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("60")}),
	)
	line := priceLine("1", "un_pair")
	got := computeUnitPrice(b, line, []domain.SalesOrderPriceLineInput{line})
	assert.Equal(t, "12", got.Value)
}

func TestListPricing_FixedPriceIsConvertedToTheOrderedUnitAndDiscounted(t *testing.T) {
	b := listBundle(rule("r_fixed", domain.PriceListSourceOwn, time.Now(), domain.PriceListRuleScope{ProductLineIDs: []string{"pl_boots"}},
		domain.PriceListPrice{Type: constants.PriceListPriceTypeFixed, UnitPrice: &domain.UnitPrice{Amount: "18.455", Currency: constants.CurrencyUSD, PerUnitID: "un_pair"}}))

	pair := priceLine("1", "un_pair")
	got := computeUnitPrice(b, pair, []domain.SalesOrderPriceLineInput{pair})
	assert.Equal(t, "18.455", got.Value, "a fixed price needing no conversion is kept exactly")

	b.ItemUnits["it_boot"].OrderUnits[0].Discount = &domain.Discount{Type: constants.DiscountTypePercentOff, PercentOff: pct("10")}
	carton := priceLine("1", "un_carton")
	got = computeUnitPrice(b, carton, []domain.SalesOrderPriceLineInput{carton})
	assert.Equal(t, "199.31", got.Value, "18.455 x 12 less 10%, to the cent")
	assert.Equal(t, "un_carton", got.DenominatorUnitID)
}

func TestListPricing_TieredRuleCountsEveryLineInScope(t *testing.T) {
	tiered := domain.PriceListPrice{Type: constants.PriceListPriceTypeTiered, UnitID: "un_carton", Tiers: []domain.PriceListTier{
		{MinQuantity: "0", PercentOfBase: pct("100")},
		{MinQuantity: "2", PercentOfBase: pct("90")},
		{MinQuantity: "5", UnitPrice: &domain.UnitPrice{Amount: "15", Currency: constants.CurrencyUSD, PerUnitID: "un_pair"}},
	}}
	b := listBundle(rule("r_tier", domain.PriceListSourceDirect, time.Now(), domain.PriceListRuleScope{CategoryIDs: []string{"ic_footwear"}}, tiered))

	boot := priceLine("12", "un_pair")
	sock := domain.SalesOrderPriceLineInput{ProductID: "pr_sock", QuantityValue: "12", QuantityUnitID: "un_pair"}
	lines := []domain.SalesOrderPriceLineInput{boot, sock}
	assert.Equal(t, "18", computeUnitPrice(b, boot, lines).Value, "two cartons across both lines reach the 90% tier")
	assert.Equal(t, "3.6", computeUnitPrice(b, sock, lines).Value)

	big := priceLine("60", "un_pair")
	assert.Equal(t, "15", computeUnitPrice(b, big, []domain.SalesOrderPriceLineInput{big}).Value)
}

func TestListPricing_UnreachedTieredRuleFallsThrough(t *testing.T) {
	now := time.Now()
	b := listBundle(
		rule("r_tier", domain.PriceListSourceOwn, now, domain.PriceListRuleScope{ItemIDs: []string{"it_boot"}},
			domain.PriceListPrice{Type: constants.PriceListPriceTypeTiered, UnitID: "un_carton", Tiers: []domain.PriceListTier{{MinQuantity: "10", PercentOfBase: pct("50")}}}),
		rule("r_line", domain.PriceListSourceOwn, now, domain.PriceListRuleScope{ProductLineIDs: []string{"pl_boots"}},
			domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("75")}),
	)
	line := priceLine("1", "un_pair")
	assert.Equal(t, "15", computeUnitPrice(b, line, []domain.SalesOrderPriceLineInput{line}).Value)
}

func TestVolumeDiscountPrice(t *testing.T) {
	flat := volumeDiscountPrice(&domain.LegacyVolumeDiscount{Tiers: []domain.LegacyVolumeTier{{Threshold: "0", Percentage: "-1.353"}}})
	require.NotNil(t, flat)
	assert.Equal(t, constants.PriceListPriceTypePercentOfBase, flat.Type)
	assert.Equal(t, "235.3", *flat.PercentOfBase, "a markup becomes a percent of base above 100")

	tiered := volumeDiscountPrice(&domain.LegacyVolumeDiscount{UnitIDs: []string{"un_carton"}, Tiers: []domain.LegacyVolumeTier{
		{Threshold: "10", Percentage: "0.1"}, {Threshold: "0", Percentage: "0.05"},
	}})
	require.NotNil(t, tiered)
	assert.Equal(t, constants.PriceListPriceTypeTiered, tiered.Type)
	assert.Equal(t, "un_carton", tiered.UnitID)
	require.Len(t, tiered.Tiers, 2)
	assert.Equal(t, "0", tiered.Tiers[0].MinQuantity)
	assert.Equal(t, "95", *tiered.Tiers[0].PercentOfBase)
	assert.Equal(t, "10", tiered.Tiers[1].MinQuantity)
	assert.Equal(t, "85.5", *tiered.Tiers[1].PercentOfBase, "tiers compound as they did: 0.95 x 0.9")

	assert.Nil(t, volumeDiscountPrice(&domain.LegacyVolumeDiscount{Tiers: []domain.LegacyVolumeTier{{Threshold: "5", Percentage: "0.1"}}}),
		"with nothing to count in, a tier above zero is never reached")
}

func TestNormalizeRulePrice(t *testing.T) {
	_, apiErr := normalizeRulePrice(domain.PriceListPrice{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: pct("0")})
	require.NotNil(t, apiErr)
	assert.Equal(t, "price.percent_of_base", apiErr.Param)

	got, apiErr := normalizeRulePrice(domain.PriceListPrice{Type: constants.PriceListPriceTypeTiered, UnitID: "un_carton", Tiers: []domain.PriceListTier{
		{MinQuantity: "5", PercentOfBase: pct("80")}, {MinQuantity: "1", PercentOfBase: pct("90")},
	}})
	require.Nil(t, apiErr)
	assert.Equal(t, "1", got.Tiers[0].MinQuantity, "tiers are sorted")

	_, apiErr = normalizeRulePrice(domain.PriceListPrice{Type: constants.PriceListPriceTypeTiered, UnitID: "un_carton", Tiers: []domain.PriceListTier{
		{MinQuantity: "1", PercentOfBase: pct("80")}, {MinQuantity: "1.0", PercentOfBase: pct("90")},
	}})
	require.NotNil(t, apiErr, "two tiers can't share a min_quantity")

	_, apiErr = normalizeRulePrice(domain.PriceListPrice{Type: constants.PriceListPriceTypeFixed, PercentOfBase: pct("80")})
	require.NotNil(t, apiErr)
}
