//go:build e2e

package api_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for POST /v1/sales/sales-orders/price-quote, pinning each layer of
// the server-side pricing engine against seeded data: list price, base->ordered unit
// conversion, attribute-gated account-price override (the path that regressed when the
// _item_attributes A/B columns were swapped), attribute-gate discrimination, an ungated
// account price for a different recipient, batched positional results, and customer-self
// authorization.
//
// Seed facts (verified against the e2e DB):
//   - SCK-002 pd_01k0a65nx5e3haz2fgfm34hmcz: list 10/pair, attrs {large} — NOT beige, so no
//     account price applies for SeedCustomerAccountID; it pays list price.
//   - SCK-005 pd_01k0a65nx5fwmt17sqp317ekyr: list 10/pair, attrs {beige, small}.
//   - account_price acpr_01seedaccprice0000: 8.5/pair to SeedCustomerAccountID, Socks line, gated by beige.
//   - account_price acpr_01seedaccprice0001: 7.5/pair to customer2 (ungated).
//   - Socks unit group: pair is the pricing denominator; 1 dozen = 6 pairs.
const (
	salesOrderPriceQuotePath = "/v1/sales/sales-orders/price-quote"
	seedNonContractProductID = "pd_01k0a65nx5e3haz2fgfm34hmcz" // SCK-002, no beige → list price
	seedBeigeProductID       = "pd_01k0a65nx5fwmt17sqp317ekyr" // SCK-005, beige → gated account price
	seedCustomer2AccountID   = "ac_01seedcustomer2_acct0"
	seedDozenUnitID          = "un_01seeddozen00000000"

	// Volume-discount fixtures (E2E Volume LTD line, list 29.95/pair = 359.40/carton; tier
	// ladder 0→5.8096828%, then 4% at thresholds 4/7/10 cartons). Carton = 12 pairs.
	seedVolumeProductA = "pd_e2evol1000000000"
	seedVolumeProductB = "pd_e2evol2000000000"
	seedCartonUnitID   = "un_e2ecarton12pr000"
)

// quoteLine builds one price-quote line input.
func quoteLine(productID, value, unitID string) map[string]any {
	return map[string]any{
		"product_id": productID,
		"quantity":   map[string]any{"value": value, "unit_id": unitID},
	}
}

// quoteLineAt returns the quote line at idx from the List[QuotedSalesOrderLine] response.
func quoteLineAt(t *testing.T, respBody []byte, idx int) map[string]any {
	t.Helper()
	lines := jsonListData(parseJSON(respBody), "lines")
	require.Greater(t, len(lines), idx, "quote should have line %d", idx)
	line, _ := lines[idx].(map[string]any)
	return line
}

// quoteUnitPriceAt posts a quote and returns the unit_price value of line idx as a float.
func quoteUnitPriceAt(t *testing.T, respBody []byte, idx int) float64 {
	t.Helper()
	line := quoteLineAt(t, respBody, idx)
	unitPrice, _ := line["unit_price"].(map[string]any)
	require.NotNil(t, unitPrice, "line %d should have a unit_price", idx)
	got, err := strconv.ParseFloat(fmt.Sprint(unitPrice["value"]), 64)
	require.NoError(t, err, "unit_price.value should parse: %v", unitPrice["value"])
	return got
}

func TestQuoteSalesOrderPrices_FullyPresentsUnits(t *testing.T) {
	t.Parallel()
	// A line item's unit_price shares the sales_order_quote_rate shape with quote-freight: its
	// numerator/denominator units must come back fully presented, not as bare {id, object}.
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines":            []map[string]any{quoteLine(seedNonContractProductID, "1", SeedUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	unitPrice := jsonObject(quoteLineAt(t, respBody, 0), "unit_price")
	require.NotNil(t, unitPrice, "line has a unit_price")
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "numerator_unit"), "numerator_unit")
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "denominator_unit"), "denominator_unit")
}

func TestQuoteSalesOrderPrices_FullyPresentsUnitsWhenDiscountChangesUnit(t *testing.T) {
	t.Parallel()
	// The unit a line's price is quoted in depends on the discount that wins: customer2's
	// per-pair account price (18.45/pair) beats the volume discount and is returned in its
	// native per-pair unit — NOT the ordered carton unit. That discount-driven denominator
	// unit (which does not come from the request) is exactly the one that used to come back
	// as a bare {id, object}. It must be fully presented.
	body := map[string]any{
		"buyer_account_id": seedCustomer2AccountID,
		"lines":            []map[string]any{quoteLine(seedVolumeProductA, "11", seedCartonUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	unitPrice := jsonObject(quoteLineAt(t, respBody, 0), "unit_price")
	require.NotNil(t, unitPrice, "line has a unit_price")
	assert.InDelta(t, 18.45, quoteUnitPriceAt(t, respBody, 0), 0.001, "account price (per pair) beats the volume discount")

	denominator := jsonObject(unitPrice, "denominator_unit")
	// The discount changed the unit away from the ordered carton to the account price's pair.
	assert.Equal(t, "un_01seedpair000000000", jsonField(denominator, "id"),
		"the discount drives a per-pair denominator, not the ordered carton unit")
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "numerator_unit"), "numerator_unit")
	assertFullyPresentedUnit(t, denominator, "denominator_unit")
}

func TestQuoteSalesOrderPrices_ListPriceWhenNoContract(t *testing.T) {
	t.Parallel()
	// SCK-002 has no beige attribute, so the beige-gated account price does not apply →
	// the customer pays the product's list price (10/pair).
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines":            []map[string]any{quoteLine(seedNonContractProductID, "3", SeedUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 10.0, quoteUnitPriceAt(t, respBody, 0), 0.001, "list price per pair")
}

func TestQuoteSalesOrderPrices_ConvertsListPriceToOrderedUnit(t *testing.T) {
	t.Parallel()
	// Ordered in dozens: the per-pair list price (10) converts to the ordered unit.
	// 1 dozen = 6 pairs, so 10/pair → 60/dozen.
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines":            []map[string]any{quoteLine(seedNonContractProductID, "1", seedDozenUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 60.0, quoteUnitPriceAt(t, respBody, 0), 0.001, "list price converted to per dozen")
}

func TestQuoteSalesOrderPrices_AppliesAttributeGatedAccountPrice(t *testing.T) {
	t.Parallel()
	// The beige sock carries the beige attribute, so the beige-gated account price
	// (8.5/pair) applies — beating list price. Fails if product attributes aren't loaded
	// (regression guard for the _item_attributes A/B column orientation).
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines":            []map[string]any{quoteLine(seedBeigeProductID, "1", SeedUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 8.5, quoteUnitPriceAt(t, respBody, 0), 0.001, "attribute-gated account price")
}

func TestQuoteSalesOrderPrices_AttributeGateDiscriminatesInOneBatch(t *testing.T) {
	t.Parallel()
	// One request, two products for the same customer: the beige one gets the gated
	// account price (8.5), the non-beige one falls through to list price (10). This pins both
	// the attribute gating AND the positional (request-order) response mapping.
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines": []map[string]any{
			quoteLine(seedBeigeProductID, "1", SeedUnitID),
			quoteLine(seedNonContractProductID, "1", SeedUnitID),
		},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	lines := jsonListData(parseJSON(respBody), "lines")
	require.Len(t, lines, 2)
	assert.Equal(t, seedBeigeProductID, jsonField(jsonObject(lines[0].(map[string]any), "product"), "id"))
	assert.Equal(t, seedNonContractProductID, jsonField(jsonObject(lines[1].(map[string]any), "product"), "id"))
	assert.InDelta(t, 8.5, quoteUnitPriceAt(t, respBody, 0), 0.001, "beige → account price")
	assert.InDelta(t, 10.0, quoteUnitPriceAt(t, respBody, 1), 0.001, "non-beige → list price")
}

func TestQuoteSalesOrderPrices_UngatedAccountPriceForOtherRecipient(t *testing.T) {
	t.Parallel()
	// A different customer (customer2) has an ungated account price (7.5/pair) on the same
	// product line, so even SCK-002 gets it. Confirms account prices are scoped to
	// the buyer.
	body := map[string]any{
		"buyer_account_id": seedCustomer2AccountID,
		"lines":            []map[string]any{quoteLine(seedNonContractProductID, "1", SeedUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 7.5, quoteUnitPriceAt(t, respBody, 0), 0.001, "ungated account price for customer2")
}

func TestQuoteSalesOrderPrices_CustomerMayQuoteOwnAccount(t *testing.T) {
	t.Parallel()
	// A customer portal actor may quote prices for its own account.
	portal := getCustomerPortalClient()
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines":            []map[string]any{quoteLine(seedBeigeProductID, "1", SeedUnitID)},
	}
	status, respBody, err := portal.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 8.5, quoteUnitPriceAt(t, respBody, 0), 0.001, "customer self-quote returns the gated price")
}

func TestQuoteSalesOrderPrices_VolumeDiscountTiers(t *testing.T) {
	t.Parallel()
	// Multiplicative tier ladder off the 359.40/carton base: each example carton quantity
	// meets a different set of thresholds (0 / 4 / 7 / 10).
	cases := []struct {
		cartons string
		want    float64
	}{
		{"1", 338.52},
		{"4", 324.98},
		{"8", 311.98},
		{"11", 299.50},
	}
	for _, c := range cases {
		c := c
		t.Run(c.cartons+"ct", func(t *testing.T) {
			t.Parallel()
			body := map[string]any{
				"buyer_account_id": SeedCustomerAccountID,
				"lines":            []map[string]any{quoteLine(seedVolumeProductA, c.cartons, seedCartonUnitID)},
			}
			status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
			require.NoError(t, err)
			requireStatus(t, 200, status, respBody)
			assert.InDelta(t, c.want, quoteUnitPriceAt(t, respBody, 0), 0.001, "%s cartons", c.cartons)
		})
	}
}

func TestQuoteSalesOrderPrices_VolumeDiscountSumsAcrossMatchingProducts(t *testing.T) {
	t.Parallel()
	// 6 cartons of product A + 5 of product B (both on the volume line) sum to 11 cartons,
	// so the top tier (299.50/carton) applies to BOTH lines.
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines": []map[string]any{
			quoteLine(seedVolumeProductA, "6", seedCartonUnitID),
			quoteLine(seedVolumeProductB, "5", seedCartonUnitID),
		},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 299.50, quoteUnitPriceAt(t, respBody, 0), 0.001, "product A at the summed top tier")
	assert.InDelta(t, 299.50, quoteUnitPriceAt(t, respBody, 1), 0.001, "product B at the summed top tier")
}

func TestQuoteSalesOrderPrices_AccountPriceBeatsVolumeDiscount(t *testing.T) {
	t.Parallel()
	// customer2 has a per-pair account price (18.45/pair) on the volume line; it overrides
	// the volume discount regardless of quantity, and is returned in its native per-pair unit.
	body := map[string]any{
		"buyer_account_id": seedCustomer2AccountID,
		"lines":            []map[string]any{quoteLine(seedVolumeProductA, "11", seedCartonUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 18.45, quoteUnitPriceAt(t, respBody, 0), 0.001, "account price (per pair) beats the volume discount")

	// The price is returned in its native per-pair unit (not converted to the ordered
	// carton), which the frontend resolves to display "18.45 / pair".
	unitPrice := jsonObject(quoteLineAt(t, respBody, 0), "unit_price")
	assert.Equal(t, "un_01seedpair000000000", jsonField(jsonObject(unitPrice, "denominator_unit"), "id"))
}
