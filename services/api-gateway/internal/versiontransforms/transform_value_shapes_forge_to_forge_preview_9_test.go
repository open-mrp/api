package versiontransforms

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

const (
	kgJSON   = `{"id":"un_kg","object":"unit","name":"Kilogram","abbreviation":"kg","dimension":"mass"}`
	hrJSON   = `{"id":"un_hr","object":"unit","name":"Hour","abbreviation":"hr","dimension":"time"}`
	pairJSON = `{"id":"un_pr","object":"unit","name":"Pair","abbreviation":"pr","dimension":"quantity"}`

	// kgPreview9JSON is kgJSON as preview.9 callers see it.
	kgPreview9JSON = `{"id":"un_kg","object":"unit","name":"Kilogram","abbreviation":"kg","type":"mass","is_base_unit":false}`
)

func field(m map[string]any, path ...string) map[string]any {
	for _, k := range path {
		m = m[k].(map[string]any)
	}
	return m
}

func TestPreview9_MoneyIsAQuantityInTheCurrency(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeCustomer, `{"id":"cus_1","object":"customer","credit_limit":{"amount":"10000.00","currency":"usd"}}`)
	assert.Equal(t, map[string]any{"object": "quantity", "value": "10000.00", "display_value": "$10,000.00", "unit": nil}, got["credit_limit"])
}

func TestPreview9_UnitPriceIsARateFromTheCurrency(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeSalesOrderLine, `{"id":"sol_1","object":"sales_order_line","unit_price":{"amount":"18.45","currency":"usd","per_unit":`+pairJSON+`},"unit_cost":null}`)
	price := field(got, "unit_price")
	assert.Equal(t, "rate", price["object"])
	assert.Equal(t, "18.45", price["value"])
	assert.Nil(t, price["numerator_unit"])
	assert.Equal(t, "un_pr", price["denominator_unit"].(map[string]any)["id"])
	assert.Equal(t, "$18.45 / pr", price["display_value"])
	assert.NotContains(t, price, "amount")
	assert.NotContains(t, price, "per_unit")
	assert.Nil(t, got["unit_cost"])
}

func TestPreview9_RateHasNumeratorAndDenominatorUnits(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItem, `{"id":"itm_1","object":"item","burn_rate":{"value":"100","unit":`+kgJSON+`,"per_unit":`+hrJSON+`}}`)
	rate := field(got, "burn_rate")
	assert.Equal(t, "rate", rate["object"])
	assert.Equal(t, "un_kg", rate["numerator_unit"].(map[string]any)["id"])
	assert.Equal(t, "un_hr", rate["denominator_unit"].(map[string]any)["id"])
	assert.Equal(t, "100 kg / hr", rate["display_value"])
	assert.NotContains(t, rate, "unit")
	assert.NotContains(t, rate, "per_unit")
}

func TestPreview9_QuantityGetsItsObjectAndDisplay(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeBatch, `{"id":"bt_1","object":"batch","quantity":{"value":"1200","unit":`+kgJSON+`}}`)
	assert.Equal(t, map[string]any{"object": "quantity", "value": "1200", "display_value": "1,200 kg", "unit": decode(t, kgPreview9JSON)}, got["quantity"])

	got = downgrade(t, constants.ObjectTypeTransaction, `{"id":"tx_1","object":"transaction","amount":{"value":"500.00","unit":{"id":"dollar","object":"unit","abbreviation":"$","dimension":"currency"}}}`)
	assert.Equal(t, "$500.00", field(got, "amount")["display_value"], "a quantity in a currency unit reads as money")
}

func TestPreview9_ComputedValuesKeepTheirObjectTypes(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeItemInventory, `{"object":"item_inventory","on_hand":{"value":"12","unit":`+pairJSON+`}}`)
	assert.Equal(t, "computed_quantity", field(got, "on_hand")["object"])

	got = downgrade(t, "", `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"object":"open_order_line","id":"sol_1","unit_price":{"amount":"4.25","currency":"usd","per_unit":`+pairJSON+`},
		 "total_ordered":{"amount":"5100","currency":"usd"},"quantity_invoiced":{"value":"360","unit":`+pairJSON+`}}]}`)
	line := got["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "computed_rate", field(line, "unit_price")["object"])
	assert.Equal(t, map[string]any{"object": "computed_quantity", "value": "5100", "display_value": "5,100.00", "unit": nil}, line["total_ordered"],
		"computed money reads without a symbol, as the reports showed it")
	assert.Equal(t, "computed_quantity", field(line, "quantity_invoiced")["object"])

	got = downgrade(t, "", `{"object":"production_cost","produced":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[{"value":"4800","unit":`+pairJSON+`}]}}`)
	produced := field(got, "produced")["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "computed_quantity", produced["object"], "a list's items take the field that holds the list")
}

func TestPreview9_SalesTotalQuantityIsAComputedQuantity(t *testing.T) {
	t.Parallel()

	got := downgrade(t, "", `{"object":"sales_totals","revenue":{"amount":"16200.5","currency":"usd"},"quantity_invoiced":"1200"}`)
	assert.Equal(t, map[string]any{"object": "computed_quantity", "value": "1200", "display_value": "1,200", "unit": nil}, got["quantity_invoiced"])
	assert.Equal(t, "computed_quantity", field(got, "revenue")["object"])
}

func TestPreview9_WeightedAverageCostIsANumeratorOverADenominator(t *testing.T) {
	t.Parallel()

	got := downgrade(t, "", `{"object":"list","data":[{"weighted_average_unit_cost":{"amount":"2.50","currency":"usd","per_unit":`+kgJSON+`}}]}`)
	rate := got["data"].([]any)[0].(map[string]any)["weighted_average_unit_cost"].(map[string]any)
	assert.Equal(t, "2.50", field(rate, "numerator")["value"])
	assert.Equal(t, "1", field(rate, "denominator")["value"])
	assert.Equal(t, "un_kg", field(rate, "denominator", "unit")["id"])
}

func TestPreview9_OnlyValueShapesAreRewritten(t *testing.T) {
	t.Parallel()

	body := `{"id":"x_1","object":"thing","metadata":{"value":"a","unit":"b"},"totals":{"object":"sales_order_stage_total","amount":"1.00","completion":0.5},
		"unit":%s,"amount_and_more":{"amount":"1","currency":"usd","extra":true}}`
	assert.Equal(t, decode(t, fmt.Sprintf(body, kgPreview9JSON)), downgrade(t, "", fmt.Sprintf(body, kgJSON)))
}

func upgrade(t *testing.T, body string) map[string]any {
	t.Helper()
	return version.TransformRequest(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeSalesOrder, decode(t, body))
}

func TestPreview9_RequestRatesAndMoneyUpgrade(t *testing.T) {
	t.Parallel()

	got := upgrade(t, `{"lines":[{"unit_price":{"value":"18.45","numerator_unit_id":"dollar","denominator_unit_id":"un_pr"}}],
		"labor_time":{"value":"1.5","numerator_unit_id":"un_hr","denominator_unit_id":"un_ea"},
		"credit_limit":{"value":"10000","unit_id":"dollar"},
		"order_point":{"value":"5","unit_id":"un_kg"}}`)
	assert.Equal(t, map[string]any{"amount": "18.45", "currency": "usd", "per_unit_id": "un_pr"}, got["lines"].([]any)[0].(map[string]any)["unit_price"])
	assert.Equal(t, map[string]any{"value": "1.5", "unit_id": "un_hr", "per_unit_id": "un_ea"}, got["labor_time"])
	assert.Equal(t, map[string]any{"amount": "10000", "currency": "usd"}, got["credit_limit"])
	assert.Equal(t, map[string]any{"value": "5", "unit_id": "un_kg"}, got["order_point"], "a quantity that is not money is unchanged")

	assert.Nil(t, upgrade(t, `{"credit_limit":null}`)["credit_limit"], "a cleared amount stays cleared")
}

func TestPreview9_FlatValueFieldsUpgrade(t *testing.T) {
	t.Parallel()

	got := upgrade(t, `{"unit_price_value":"15.00","unit_price_numerator_unit_id":"dollar","unit_price_denominator_unit_id":"un_ea",
		"freight_amount_value":"12.50","freight_amount_unit_id":"dollar","freight_weight_value":"5","amount_value":"50000","amount_unit_id":"dollar"}`)
	assert.Equal(t, map[string]any{"amount": "15.00", "currency": "usd", "per_unit_id": "un_ea"}, got["unit_price"])
	assert.Equal(t, map[string]any{"amount": "12.50", "currency": "usd"}, got["freight_amount"])
	assert.Equal(t, map[string]any{"amount": "50000", "currency": "usd"}, got["amount"])
	assert.Equal(t, "5", got["freight_weight_value"], "a non-money flat field is left alone")
	for _, k := range []string{"unit_price_value", "unit_price_numerator_unit_id", "unit_price_denominator_unit_id", "freight_amount_value", "freight_amount_unit_id", "amount_value", "amount_unit_id"} {
		assert.NotContains(t, got, k)
	}
}

func TestPreview9_IncludeKeysIntoUnitsAreCutBack(t *testing.T) {
	t.Parallel()

	route := "/v1/test/value-shape-includes/{id}"
	RegisterRouteIncludes(route, []string{"lines", "lines.unit_price", "customer"})
	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeSalesOrder, route, url.Values{
		"include[]": {"lines", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit", "lines.quantity.unit", "customer", "lines.quantity"},
	})
	assert.Equal(t, []string{"lines", "lines.unit_price", "customer"}, q["include[]"])

	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeBatch, "/v1/test/no-includes", url.Values{"include": {"quantity.unit,seconds.unit"}})
	assert.NotContains(t, q, "include", "an include left with no keys is dropped")
}
