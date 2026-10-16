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

// quoteUnitPriceAt posts a quote and returns the unit_price amount of line idx as a float.
func quoteUnitPriceAt(t *testing.T, respBody []byte, idx int) float64 {
	t.Helper()
	line := quoteLineAt(t, respBody, idx)
	unitPrice, _ := line["unit_price"].(map[string]any)
	require.NotNil(t, unitPrice, "line %d should have a unit_price", idx)
	assert.Equal(t, "usd", jsonField(unitPrice, "currency"), "line %d unit_price.currency", idx)
	got, err := strconv.ParseFloat(fmt.Sprint(unitPrice["amount"]), 64)
	require.NoError(t, err, "unit_price.amount should parse: %v", unitPrice["amount"])
	return got
}

func TestQuoteSalesOrderPrices_FullyPresentsUnits(t *testing.T) {
	t.Parallel()
	lockPricingRead(t)
	// The unit a line's price is per must come back fully presented, not as a bare {id, object}.
	body := map[string]any{
		"buyer_account_id": SeedCustomerAccountID,
		"lines":            []map[string]any{quoteLine(seedNonContractProductID, "1", SeedUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	unitPrice := jsonObject(quoteLineAt(t, respBody, 0), "unit_price")
	require.NotNil(t, unitPrice, "line has a unit_price")
	assert.NotEmpty(t, jsonField(unitPrice, "amount"), "unit_price.amount")
	assert.Equal(t, "usd", jsonField(unitPrice, "currency"), "unit_price.currency")
	assertFullyPresentedUnit(t, jsonObject(unitPrice, "per_unit"), "per_unit")
}

func TestQuoteSalesOrderPrices_FullyPresentsUnitsWhenDiscountChangesUnit(t *testing.T) {
	t.Parallel()
	lockPricingRead(t)
	// customer2's own per-pair price (18.45/pair) beats the volume tiers and is converted to the
	// ordered carton (12 pairs). The per unit must come back fully presented.
	body := map[string]any{
		"buyer_account_id": seedCustomer2AccountID,
		"lines":            []map[string]any{quoteLine(seedVolumeProductA, "11", seedCartonUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	unitPrice := jsonObject(quoteLineAt(t, respBody, 0), "unit_price")
	require.NotNil(t, unitPrice, "line has a unit_price")
	assert.InDelta(t, 221.40, quoteUnitPriceAt(t, respBody, 0), 0.001, "the customer's own price, per carton")

	perUnit := jsonObject(unitPrice, "per_unit")
	assert.Equal(t, seedCartonUnitID, jsonField(perUnit, "id"), "the price is per the ordered carton")
	assertFullyPresentedUnit(t, perUnit, "per_unit")
}

func TestQuoteSalesOrderPrices_ListPriceWhenNoContract(t *testing.T) {
	t.Parallel()
	lockPricingRead(t)
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
	lockPricingRead(t)
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
	lockPricingRead(t)
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
	lockPricingRead(t)
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
	lockPricingRead(t)
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
	lockPricingRead(t)
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
	lockPricingRead(t)
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
			// Serial, and no lock of its own: the parent already holds the shared side for the
			// whole table, and RLock is not reentrant once a writer is waiting. Parallel
			// subtests would deadlock the package — the parent cannot release the read lock
			// until they finish, they cannot start until a parallel slot frees, and the slots
			// fill with the account-price tests blocked on the exclusive side. Four quotes run
			// serially in a few hundred milliseconds, which is not worth a lock cycle to shave.
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
	lockPricingRead(t)
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
	lockPricingRead(t)
	// customer2 has its own per-pair price (18.45/pair) on the volume line; its list beats the
	// account-wide volume tiers on an equally specific scope, whatever the quantity.
	body := map[string]any{
		"buyer_account_id": seedCustomer2AccountID,
		"lines":            []map[string]any{quoteLine(seedVolumeProductA, "11", seedCartonUnitID)},
	}
	status, respBody, err := apiClient.Post(salesOrderPriceQuotePath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.InDelta(t, 221.40, quoteUnitPriceAt(t, respBody, 0), 0.001, "18.45 per pair, converted to the ordered carton")

	unitPrice := jsonObject(quoteLineAt(t, respBody, 0), "unit_price")
	assert.Equal(t, seedCartonUnitID, jsonField(jsonObject(unitPrice, "per_unit"), "id"))
}
