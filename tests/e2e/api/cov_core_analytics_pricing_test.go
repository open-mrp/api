//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/augno/api/shared/id"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	customerPricingPath = "/v1/core/analytics/customer-pricing"
	realizedMarginsPath = "/v1/core/analytics/realized-margins"
	priceListPath       = "/v1/sales/account-prices/actions/export-price-list"

	// The seeded dollar unit is a global system unit; pairs are the seeded per-unit basis.
	seedCurrencyUnitID = "dollar"
)

// analyzeCustomerPricing calls the configured-pricing audit and fails on anything other than 200. A 5xx here is a backend defect, never something to skip past.
func analyzeCustomerPricing(t *testing.T, params url.Values, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := apiClient.PutRaw(customerPricingPath, params, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "customer pricing analytics must not 5xx: %s", string(respBody))
	requireStatus(t, http.StatusOK, status, respBody)
	return parseJSON(respBody)
}

func analyzeRealizedMargins(t *testing.T, params url.Values, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := apiClient.PutRaw(realizedMarginsPath, params, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "realized margin analytics must not 5xx: %s", string(respBody))
	requireStatus(t, http.StatusOK, status, respBody)
	return parseJSON(respBody)
}

// createSeededAccountPrice records a contracted price on the seeded product line and returns its id, removing it when the test ends so repeated runs do not pile up fixtures.
func createSeededAccountPrice(t *testing.T, rateValue string) string {
	t.Helper()
	status, body, err := apiClient.Post(accountPricesPath, map[string]any{
		"recipient_account_id": SeedCustomerAccountID,
		"product_line_id":      SeedProductLineID,
		"rate": map[string]any{
			"value":               rateValue,
			"numerator_unit_id":   seedCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)

	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() {
		_, _, _ = apiClient.Delete(accountPricesPath + "/" + id)
	})
	return id
}

// summaryCount reads a whole-number counter out of an analysis summary.
func summaryCount(t *testing.T, summary map[string]any, key string) int {
	t.Helper()
	raw, ok := summary[key]
	require.True(t, ok, "summary is missing %q", key)
	n, ok := raw.(float64)
	require.True(t, ok, "summary %q is not a number: %#v", key, raw)
	return int(n)
}

// findingsForAccountPrice returns every finding raised for a contracted price. One price yields one finding per customer it reaches, so a price on a parent account produces a row for the parent and one for each child.
func findingsForAccountPrice(resp map[string]any, accountPriceID string) []map[string]any {
	var out []map[string]any
	for _, raw := range jsonListData(resp, "findings") {
		finding, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(finding, "account_price_id") == accountPriceID {
			out = append(out, finding)
		}
	}
	return out
}

// findFindingByAccountPrice returns the finding for the customer the price is actually recorded against.
func findFindingByAccountPrice(resp map[string]any, accountPriceID string) map[string]any {
	for _, finding := range findingsForAccountPrice(resp, accountPriceID) {
		if jsonField(finding, "origin") == "direct" {
			return finding
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// Customer pricing — shape
// ──────────────────────────────────────────────

func TestAnalyticsCustomerPricing_ResponseShape(t *testing.T) {
	t.Parallel()

	resp := analyzeCustomerPricing(t, nil, map[string]any{})

	assert.Equal(t, "analyze_customer_pricing_response", jsonField(resp, "object"))

	findings := jsonObject(resp, "findings")
	require.NotNil(t, findings, "findings must be a list envelope, not a bare array")
	assert.Equal(t, "list", jsonField(findings, "object"))

	summary := jsonObject(resp, "summary")
	require.NotNil(t, summary)
	assert.Equal(t, "customer_pricing_summary", jsonField(summary, "object"))
}

// The account-wide analysis has no scope filter, so counts move with whatever else the suite has created. Only the invariants between the numbers are safe to assert.
func TestAnalyticsCustomerPricing_SummaryIsSelfConsistent(t *testing.T) {
	t.Parallel()

	resp := analyzeCustomerPricing(t, nil, map[string]any{})
	summary := jsonObject(resp, "summary")

	analyzed := summaryCount(t, summary, "prices_analyzed")
	belowPeer := summaryCount(t, summary, "below_peer_median_count")
	belowMargin := summaryCount(t, summary, "below_target_margin_count")
	notAssessed := summaryCount(t, summary, "margin_not_assessed_count")

	assert.LessOrEqual(t, belowPeer, analyzed, "cannot flag more prices than were examined")
	assert.LessOrEqual(t, belowMargin, analyzed)
	assert.LessOrEqual(t, notAssessed, analyzed)

	findings := jsonListData(resp, "findings")
	assert.LessOrEqual(t, len(findings), belowPeer+belowMargin, "every finding carries at least one flag")
}

// ──────────────────────────────────────────────
// Customer pricing — the outlier check actually fires
// ──────────────────────────────────────────────

// A price far below its peers on the same product line and per-unit basis must be flagged. Two prices are created so the peer group has a median at all — a lone contracted price is its own median and can never be an outlier.
func TestAnalyticsCustomerPricing_FlagsPriceFarBelowPeers(t *testing.T) {
	createSeededAccountPrice(t, "500.00")
	outlierID := createSeededAccountPrice(t, "0.01")

	resp := analyzeCustomerPricing(t, nil, map[string]any{
		"outlier_tolerance":   "0.15",
		"target_gross_margin": "0",
	})

	finding := findFindingByAccountPrice(resp, outlierID)
	require.NotNil(t, finding, "a price of 0.01 against a peer at 500.00 must be flagged")

	assert.Equal(t, "customer_pricing_finding", jsonField(finding, "object"))
	assert.Contains(t, []string{"below_peer_median", "below_peer_median_and_target_margin"}, jsonField(finding, "reason"))
	assert.Equal(t, "direct", jsonField(finding, "origin"), "the price is recorded against the customer, not inherited")
	assert.NotEmpty(t, jsonField(finding, "id"))

	unitPrice := jsonObject(finding, "unit_price")
	require.NotNil(t, unitPrice, "unit price is always present")
	assert.Equal(t, "computed_rate", jsonField(unitPrice, "object"), "a computed rate carries no id because nothing was stored")
	assert.NotEmpty(t, jsonField(unitPrice, "display_value"))
	assert.Empty(t, jsonField(unitPrice, "id"), "computed rates must not invent an id")

	peerMedian := jsonObject(finding, "peer_median_price")
	require.NotNil(t, peerMedian, "an outlier is only an outlier against a median")
	assert.Equal(t, "computed_rate", jsonField(peerMedian, "object"))
}

// The response models flags as one enum rather than a pair of booleans, so the wire format must never carry the old boolean fields.
func TestAnalyticsCustomerPricing_UsesEnumNotBooleans(t *testing.T) {
	createSeededAccountPrice(t, "500.00")
	outlierID := createSeededAccountPrice(t, "0.01")

	resp := analyzeCustomerPricing(t, nil, map[string]any{"outlier_tolerance": "0.15", "target_gross_margin": "0"})
	finding := findFindingByAccountPrice(resp, outlierID)
	require.NotNil(t, finding)

	for _, gone := range []string{"is_below_peer_median", "is_below_target_margin", "is_inherited"} {
		_, present := finding[gone]
		assert.False(t, present, "%s should have been replaced by an enum", gone)
	}
	assert.Contains(t,
		[]string{"below_peer_median", "below_target_margin", "below_peer_median_and_target_margin"},
		jsonField(finding, "reason"))
	assert.Contains(t, []string{"direct", "inherited"}, jsonField(finding, "origin"))
}

// A price recorded against a parent account also prices its children's orders, so auditing customer by customer would miss them. Each reached customer gets its own finding, marked by where the price actually lives.
func TestAnalyticsCustomerPricing_FansOutToChildAccounts(t *testing.T) {
	createSeededAccountPrice(t, "500.00")
	outlierID := createSeededAccountPrice(t, "0.01")

	resp := analyzeCustomerPricing(t,
		url.Values{"include": {"customer"}},
		map[string]any{"outlier_tolerance": "0.15", "target_gross_margin": "0"},
	)

	findings := findingsForAccountPrice(resp, outlierID)
	require.NotEmpty(t, findings)

	var direct, inherited int
	seen := map[string]bool{}
	for _, finding := range findings {
		switch jsonField(finding, "origin") {
		case "direct":
			direct++
			assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(finding, "customer"), "id"))
		case "inherited":
			inherited++
			assert.NotEqual(t, SeedCustomerAccountID, jsonField(jsonObject(finding, "customer"), "id"),
				"an inherited row belongs to a child account, not the account the price is recorded against")
		}
		id := jsonField(finding, "id")
		assert.False(t, seen[id], "each reached customer must get its own finding id")
		seen[id] = true
	}
	assert.Equal(t, 1, direct, "the price is recorded against exactly one customer")
	assert.Positive(t, inherited, "the seeded customer has child accounts the price also reaches")
}

// ──────────────────────────────────────────────
// Customer pricing — expandable relations
// ──────────────────────────────────────────────

// Relations are ids on the wire only when asked for; unexpanded they are null, and the customer never leaks as an inlined name.
func TestAnalyticsCustomerPricing_RelationsAreNullWithoutInclude(t *testing.T) {
	createSeededAccountPrice(t, "500.00")
	outlierID := createSeededAccountPrice(t, "0.01")

	resp := analyzeCustomerPricing(t, nil, map[string]any{"outlier_tolerance": "0.15", "target_gross_margin": "0"})
	finding := findFindingByAccountPrice(resp, outlierID)
	require.NotNil(t, finding)

	assert.Nil(t, finding["customer"], "customer must be null until expanded")
	assert.Nil(t, finding["product_line"], "product line must be null until expanded")
	for _, inlined := range []string{"customer_name", "customer_number", "product_line_name"} {
		_, present := finding[inlined]
		assert.False(t, present, "%s should have been replaced by an expandable relation", inlined)
	}
}

func TestAnalyticsCustomerPricing_ExpandsCustomerAndProductLine(t *testing.T) {
	createSeededAccountPrice(t, "500.00")
	outlierID := createSeededAccountPrice(t, "0.01")

	resp := analyzeCustomerPricing(t,
		url.Values{"include": {"customer", "product_line"}},
		map[string]any{"outlier_tolerance": "0.15", "target_gross_margin": "0"},
	)
	finding := findFindingByAccountPrice(resp, outlierID)
	require.NotNil(t, finding)

	customer := jsonObject(finding, "customer")
	require.NotNil(t, customer, "?include=customer must populate the relation")
	assert.Equal(t, SeedCustomerAccountID, jsonField(customer, "id"))
	assert.Equal(t, "customer", jsonField(customer, "object"))

	productLine := jsonObject(finding, "product_line")
	require.NotNil(t, productLine)
	assert.Equal(t, SeedProductLineID, jsonField(productLine, "id"))
}

func TestAnalyticsCustomerPricing_RejectsUnknownInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.PutRaw(customerPricingPath, url.Values{"include": {"not_a_relation"}}, map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown include is a client error: %s", string(body))
	assert.Equal(t, http.StatusBadRequest, status, "body: %s", string(body))
}

// ──────────────────────────────────────────────
// Customer pricing — validation
// ──────────────────────────────────────────────

func TestAnalyticsCustomerPricing_RejectsOutOfRangeFractions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ field, value string }{
		{"target_gross_margin", "1.5"},
		{"target_gross_margin", "-0.1"},
		{"outlier_tolerance", "2"},
		{"outlier_tolerance", "not-a-number"},
	} {
		t.Run(tc.field+"="+tc.value, func(t *testing.T) {
			status, body, err := apiClient.Put(customerPricingPath, map[string]any{tc.field: tc.value})
			require.NoError(t, err)
			require.Less(t, status, 500, "bad input must not 5xx: %s", string(body))
			require.Equal(t, http.StatusBadRequest, status, "body: %s", string(body))
			errObj := requireErrorResponse(t, body, "", "")
			assertErrorParam(t, errObj, tc.field)
		})
	}
}

// ──────────────────────────────────────────────
// Realized margins
// ──────────────────────────────────────────────

func TestAnalyticsRealizedMargins_ResponseShape(t *testing.T) {
	t.Parallel()

	end := time.Now().UTC()
	resp := analyzeRealizedMargins(t, nil, map[string]any{
		"starts_at": rfc3339(end.AddDate(-2, 0, 0)),
		"ends_at":   rfc3339(end),
	})

	assert.Equal(t, "analyze_realized_margins_response", jsonField(resp, "object"))

	findings := jsonObject(resp, "findings")
	require.NotNil(t, findings)
	assert.Equal(t, "list", jsonField(findings, "object"))

	summary := jsonObject(resp, "summary")
	require.NotNil(t, summary)
	assert.Equal(t, "realized_margin_summary", jsonField(summary, "object"))

	// Any finding present must carry computed amounts rather than bare decimals.
	for _, raw := range jsonListData(resp, "findings") {
		finding, ok := raw.(map[string]any)
		require.True(t, ok)
		revenue := jsonObject(finding, "revenue")
		require.NotNil(t, revenue, "revenue must be a computed quantity")
		assert.Equal(t, "computed_quantity", jsonField(revenue, "object"))
		assert.Empty(t, jsonField(revenue, "id"), "computed quantities must not invent an id")
		assert.Equal(t, "computed_rate", jsonField(jsonObject(finding, "average_unit_price"), "object"))
		assert.Contains(t,
			[]string{"below_peer_median", "below_target_margin", "below_peer_median_and_target_margin"},
			jsonField(finding, "reason"))
		break
	}
}

func TestAnalyticsRealizedMargins_RequiresAWindow(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(realizedMarginsPath, map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "a missing window is a client error: %s", string(body))
	assert.Equal(t, http.StatusBadRequest, status, "body: %s", string(body))
}

// An inverted window is a client mistake, not an empty result.
func TestAnalyticsRealizedMargins_RejectsInvertedWindow(t *testing.T) {
	t.Parallel()

	end := time.Now().UTC()
	status, body, err := apiClient.Put(realizedMarginsPath, map[string]any{
		"starts_at": rfc3339(end),
		"ends_at":   rfc3339(end.AddDate(-1, 0, 0)),
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "an inverted window must not 5xx: %s", string(body))
	require.Equal(t, http.StatusBadRequest, status, "body: %s", string(body))
	errObj := requireErrorResponse(t, body, "", "")
	assertErrorParam(t, errObj, "ends_at")
}

func TestAnalyticsRealizedMargins_RejectsUnknownInclude(t *testing.T) {
	t.Parallel()

	end := time.Now().UTC()
	status, body, err := apiClient.PutRaw(realizedMarginsPath, url.Values{"include": {"not_a_relation"}}, map[string]any{
		"starts_at": rfc3339(end.AddDate(-1, 0, 0)),
		"ends_at":   rfc3339(end),
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown include is a client error: %s", string(body))
	assert.Equal(t, http.StatusBadRequest, status, "body: %s", string(body))
}

// ──────────────────────────────────────────────
// Price list export
// ──────────────────────────────────────────────

// Pricing a whole catalog runs in the background, so the export accepts a job and the file arrives when it finishes. The object is downloaded straight from storage, which is why its key — not a response header — has to carry the .pdf name.
func TestPriceListExport_ProducesAPDF(t *testing.T) {
	t.Parallel()

	job := completedExportJob(t, priceListPath, map[string]any{"customer_id": SeedCustomerAccountID})

	// Reaching "completed" is the assertion: the job only settles once the worker has
	// loaded the customer's catalog, priced it against one pricing bundle, rendered the
	// PDF and durably stored it. The file itself is out of reach — test mode's object
	// store discards the bytes and hands back a fixed URL — so the .pdf naming is
	// pinned by TestExportObjectKey_ExtensionFollowsTheFormat instead.
	export := jsonObject(job, "export")
	require.NotNil(t, export, "a completed export job names its file")
	assert.NotEmpty(t, jsonField(export, "url"), "the finished job must carry a download URL")
}

func TestPriceListExport_RequiresACustomer(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(priceListPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a missing customer is a client error: %s", string(body))
	assert.Equal(t, http.StatusBadRequest, status, "body: %s", string(body))
}

// The accept only records which customer to price, so an unknown one surfaces when the worker runs rather than rejecting the request.
func TestPriceListExport_UnknownCustomerIsAccepted(t *testing.T) {
	t.Parallel()

	missing := mustGenID(t, id.AccountIDPrefix)
	status, body, err := apiClient.Post(priceListPath, map[string]any{"customer_id": missing}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown customer must not 5xx: %s", string(body))
	assert.Equal(t, http.StatusAccepted, status, "body: %s", string(body))
}
