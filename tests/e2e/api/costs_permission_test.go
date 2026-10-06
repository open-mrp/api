//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// What goods cost the seller is shown only to its own staff holding costs:read, and to admins. Without it the fields read null and the record still loads; a report that is nothing but cost answers 403. Customer and supplier portal actors never see cost, whatever their own role grants.

const (
	costsRead = "costs:read"

	// The supplier portal key, owned by SeedSupplierAccountID (shared/db/seed/e2e/0014_e2e_extras.sql).
	SeedSupplierAPIKey = "mrp_sk_prod_SupPortalE2eTestKey01_SupplierPortalE2eTestSecretValueForCostVisibility1ea3nDd"

	manufacturingPath      = "/v1/core/analytics/manufacturing"
	manufacturingBatchPath = "/v1/core/analytics/manufacturing-batch"
	analyzeSalesPath       = "/v1/core/analytics/sales"
)

var supplierPortalKeyOnce sync.Once

// getSupplierPortalClient is the seed supplier's own API key aimed at the seller: a supplier relation actor. Its role is the global admin role, so it carries costs:read for its own account.
func getSupplierPortalClient(t *testing.T) *Client {
	t.Helper()
	// Inserted here as well as seeded, so the key exists on a stack seeded before it was added.
	supplierPortalKeyOnce.Do(func() {
		_, err := authDB(t).Exec("INSERT IGNORE INTO api_key (type_id, key_id, name, secret_hash, redacted_value, owner_account_id, role_id, created_at, updated_at) "+
			"VALUES ('apky_e2esupportal000000000000', 'SupPortalE2eTestKey01', 'Supplier Portal E2E Key', "+
			"UNHEX('d6ad5a2b73f1102e17ea4c3891f6c5fde1a365d165fdbf860b0fbc146c61427c'), 'mrp_sk_prod_****3nDd', ?, 'rl_mtg88e6u6fbu', NOW(3), NOW(3))",
			SeedSupplierAccountID)
		require.NoError(t, err)
	})
	return NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedSupplierAPIKey, SeedAccountID)
}

// isCostField mirrors the gateway's reading of a field name as the seller's cost or margin data.
func isCostField(name string) bool {
	switch name {
	case "labor_rate", "overhead_rate", "changeover_labor_rate", "inventory_value":
		return true
	}
	for _, token := range strings.Split(name, "_") {
		switch token {
		case "cost", "costs", "cogs", "margin", "margins", "profit", "profits", "valuation", "markup":
			return true
		}
	}
	return false
}

// costLeaks lists the JSON paths in v where a cost field holds a value.
func costLeaks(v any, path string) []string {
	var out []string
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			p := k
			if path != "" {
				p = path + "." + k
			}
			if isCostField(k) && child != nil {
				out = append(out, p)
				continue
			}
			out = append(out, costLeaks(child, p)...)
		}
	case []any:
		for _, elem := range x {
			out = append(out, costLeaks(elem, path+"[]")...)
		}
	}
	return out
}

func requireNoCost(t *testing.T, who, what string, body []byte) {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal(body, &v), "%s %s: %s", who, what, string(body))
	leaks := costLeaks(v, "")
	sort.Strings(leaks)
	assert.Empty(t, leaks, "%s received cost data from %s", who, what)
}

func getAs(t *testing.T, client *Client, path string, includes ...string) (int, []byte) {
	t.Helper()
	var params url.Values
	if len(includes) > 0 {
		params = url.Values{"include": includes}
	}
	status, body, err := client.GetListRaw(path, params)
	require.NoError(t, err)
	require.Less(t, status, 500, "GET %s must not 5xx: %s", path, string(body))
	return status, body
}

// costProduct creates a sale product whose item costs costValue per pair, returning the product and its item.
func costProduct(t *testing.T, costValue string) (productID, itemID string) {
	t.Helper()
	sku := uniqueName("e2e-costs-sku")
	created := createAndCleanup(t, productsPath, map[string]any{
		"sku":         sku,
		"type":        "sale",
		"category_id": SeedItemCategoryID,
		"unit_cost":   map[string]any{"value": costValue, "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
	})
	productID = jsonField(created, "id")
	itemID = jsonField(jsonObject(parseJSON(mustGetAs(t, apiClient, productsPath+"/"+productID, url.Values{"include": {"item"}})), "item"), "id")
	require.NotEmpty(t, itemID)
	return productID, itemID
}

func rateValue(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	rate := jsonObject(m, key)
	require.NotNil(t, rate, "%s is expected to carry a value: %v", key, m)
	return jsonField(rate, "value")
}

// --- Internal roles ---

func TestCosts_ItemUnitCostNeedsCostsRead(t *testing.T) {
	t.Parallel()
	productID, itemID := costProduct(t, "3.25")
	sku := jsonField(parseJSON(mustGetAs(t, apiClient, itemsPath+"/"+itemID, nil)), "sku")

	itemsOnly := customRoleClient(t, "items:read")
	withCosts := customRoleClient(t, "items:read", costsRead)
	include := url.Values{"include": {"unit_cost", "unit_value"}}

	adminItem := parseJSON(mustGetAs(t, apiClient, itemsPath+"/"+itemID, include))
	adminCost := rateValue(t, adminItem, "unit_cost")
	require.True(t, strings.HasPrefix(adminCost, "3.25"), "the admin reads the item's cost: %s", adminCost)

	got := parseJSON(mustGetAs(t, itemsOnly, itemsPath+"/"+itemID, include))
	assertNilField(t, got, "unit_cost")
	assert.NotNil(t, jsonObject(got, "unit_value"), "the selling value is not cost data")
	assert.Equal(t, itemID, jsonField(got, "id"), "the item still loads")

	list := parseJSON(mustGetAs(t, itemsOnly, itemsPath, url.Values{"q": {sku}, "include": include["include"]}))
	rows := listRows(t, list)
	require.Len(t, rows, 1)
	assertNilField(t, rows[0], "unit_cost")

	assert.Equal(t, adminCost, rateValue(t, parseJSON(mustGetAs(t, withCosts, itemsPath+"/"+itemID, include)), "unit_cost"))
	withCostsRows := listRows(t, parseJSON(mustGetAs(t, withCosts, itemsPath, url.Values{"q": {sku}, "include": include["include"]})))
	require.Len(t, withCostsRows, 1)
	assert.Equal(t, adminCost, rateValue(t, withCostsRows[0], "unit_cost"))

	// The same item embedded in a product follows the same rule.
	productInclude := url.Values{"include": {"item", "item.unit_cost"}}
	embedded := jsonObject(parseJSON(mustGetAs(t, itemsOnly, productsPath+"/"+productID, productInclude)), "item")
	require.NotNil(t, embedded)
	assertNilField(t, embedded, "unit_cost")
	embedded = jsonObject(parseJSON(mustGetAs(t, withCosts, productsPath+"/"+productID, productInclude)), "item")
	assert.Equal(t, adminCost, rateValue(t, embedded, "unit_cost"))
}

func TestCosts_SalesOrderLineUnitCostNeedsCostsRead(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)

	path := salesOrdersPath + "/" + orderID
	include := url.Values{"include": {"lines", "lines.unit_cost", "lines.unit_price"}}
	linesOf := func(client *Client) []map[string]any {
		var out []map[string]any
		for _, l := range jsonListData(parseJSON(mustGetAs(t, client, path, include)), "lines") {
			out = append(out, l.(map[string]any))
		}
		require.NotEmpty(t, out)
		return out
	}

	admin := linesOf(apiClient)
	for _, l := range linesOf(customRoleClient(t, "sales_orders:read")) {
		assertNilField(t, l, "unit_cost")
		assert.NotNil(t, jsonObject(l, "unit_price"), "the price charged is not cost data")
	}
	// A freight line carries no cost for anyone, so the comparison is line by line against what the admin sees.
	withCosts := linesOf(customRoleClient(t, "sales_orders:read", costsRead))
	require.Len(t, withCosts, len(admin))
	costed := 0
	for i := range admin {
		if jsonObject(admin[i], "unit_cost") == nil {
			assertNilField(t, withCosts[i], "unit_cost")
			continue
		}
		costed++
		assert.Equal(t, rateValue(t, admin[i], "unit_cost"), rateValue(t, withCosts[i], "unit_cost"))
	}
	require.Positive(t, costed, "the order has a product line whose cost the admin reads")
}

func costDepartment(t *testing.T) string {
	t.Helper()
	return jsonField(createAndCleanup(t, departmentsPath, map[string]any{
		"name":       uniqueName("e2e-costs-dept"),
		"labor_rate": map[string]any{"value": "31.50", "numerator_unit_id": dollarUnitID, "denominator_unit_id": "hour"},
	}), "id")
}

func TestCosts_DepartmentLaborRateNeedsCostsRead(t *testing.T) {
	t.Parallel()
	path := departmentsPath + "/" + costDepartment(t)

	adminRate := rateValue(t, parseJSON(mustGetAs(t, apiClient, path, nil)), "labor_rate")
	got := parseJSON(mustGetAs(t, customRoleClient(t, "departments:read"), path, nil))
	assertNilField(t, got, "labor_rate")
	assert.NotEmpty(t, jsonField(got, "name"), "the department still loads")
	assert.Equal(t, adminRate, rateValue(t, parseJSON(mustGetAs(t, customRoleClient(t, "departments:read", costsRead), path, nil)), "labor_rate"))
}

func TestCosts_ProductionStepRatesNeedCostsRead(t *testing.T) {
	t.Parallel()
	params := url.Values{"limit": {"50"}}
	byID := func(client *Client) map[string]map[string]any {
		out := map[string]map[string]any{}
		for _, r := range listRows(t, parseJSON(mustGetAs(t, client, productionStepsPath, params))) {
			out[jsonField(r, "id")] = r
		}
		return out
	}
	admin := byID(apiClient)
	without := byID(customRoleClient(t, "production_steps:read"))
	with := byID(customRoleClient(t, "production_steps:read", costsRead))

	compared := 0
	for id, row := range without {
		for _, f := range []string{"labor_rate", "overhead_rate"} {
			assertNilField(t, row, f)
			if a, ok := admin[id]; ok && a[f] != nil {
				if w, ok := with[id]; ok {
					compared++
					assert.Equal(t, a[f], w[f], "%s %s with costs:read", id, f)
				}
			}
		}
	}
	assert.Positive(t, compared, "the admin must see some step rate to compare against")
}

func TestCosts_ScheduleSettingsChangeoverLaborRateNeedsCostsRead(t *testing.T) {
	t.Parallel()
	admin := parseJSON(mustGetAs(t, apiClient, scheduleSettingsPath, nil))
	require.NotNil(t, admin["changeover_labor_rate"], "the admin reads the changeover labor rate")

	without := parseJSON(mustGetAs(t, customRoleClient(t, "production_schedules:read"), scheduleSettingsPath, nil))
	assertNilField(t, without, "changeover_labor_rate")
	assert.Equal(t, admin["holding_rate_pct"], without["holding_rate_pct"], "the rest of the settings still load")

	with := parseJSON(mustGetAs(t, customRoleClient(t, "production_schedules:read", costsRead), scheduleSettingsPath, nil))
	assert.Equal(t, admin["changeover_labor_rate"], with["changeover_labor_rate"])
}

// Sales figures carry their cost and profit; a reader without costs:read gets the sales with those left null.
func TestCosts_SalesReportsHideCostWithoutCostsRead(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)
	filter := saleFilter(sale.customerID)
	invoicesOnly := customRoleClient(t, "invoices:read")
	withCosts := customRoleClient(t, "invoices:read", costsRead)

	entries := func(client *Client) []map[string]any {
		got := mustPutAnalytics(t, client, analyzeSalesPath, nil, filter)
		rows := listRows(t, got)
		require.NotEmpty(t, rows)
		return rows
	}
	for _, row := range entries(invoicesOnly) {
		for _, f := range []string{"unit_cost", "unit_profit", "total_cost", "total_profit"} {
			assertNilField(t, row, f)
		}
		assert.NotEmpty(t, jsonField(row, "total_invoiced"), "revenue is not cost data")
	}
	for _, row := range entries(withCosts) {
		for _, f := range []string{"unit_cost", "unit_profit", "total_cost", "total_profit"} {
			assert.NotNil(t, row[f], "%s with costs:read", f)
		}
	}

	summary := mustPutAnalytics(t, invoicesOnly, salesSummaryPath, nil, filter)
	assertNilField(t, jsonObject(summary, "overall"), "cost")
	assert.NotNil(t, jsonObject(jsonObject(summary, "overall"), "revenue"))
	summary = mustPutAnalytics(t, withCosts, salesSummaryPath, nil, filter)
	assert.NotNil(t, jsonObject(jsonObject(summary, "overall"), "cost"), "a costs:read holder sees the cost of what was sold")

	now := time.Now().UTC()
	batchBody := map[string]any{
		"starts_at": rfc3339(now.Add(-24 * time.Hour)), "ends_at": rfc3339(now.Add(24 * time.Hour)),
		"comparison_starts_at": rfc3339(now.Add(-48 * time.Hour)), "comparison_ends_at": rfc3339(now.Add(-24 * time.Hour)),
	}
	batch := mustPutAnalytics(t, invoicesOnly, manufacturingBatchPath, nil, batchBody)
	for _, period := range []string{"current", "comparison"} {
		assertNilField(t, jsonObject(batch, period), "costs_per_unit")
		assertNilField(t, jsonObject(batch, period), "margin")
		assert.NotNil(t, jsonObject(batch, period)["production"], "the other metrics are not cost data")
	}
	batch = mustPutAnalytics(t, withCosts, manufacturingBatchPath, nil, batchBody)
	assert.NotNil(t, jsonObject(batch, "current")["costs_per_unit"])
}

// A report that is nothing but cost or margin is refused outright to a role without costs:read, including one holding the permission it used to need.
func TestCosts_CostReportsNeedCostsRead(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	window := map[string]any{"starts_at": rfc3339(now.Add(-30 * 24 * time.Hour)), "ends_at": rfc3339(now)}

	reports := []struct {
		name       string
		method     string
		path       string
		body       map[string]any
		formerPerm string
		extraPerm  []string
	}{
		{"item costs", "GET", itemsPath + "/" + SeedItemID + "/costs", nil, "items:read", nil},
		{"production costs", "PUT", productionCostsPath, window, "batches:read", nil},
		{"realized margins", "PUT", realizedMarginsPath, window, "invoices:read", nil},
		{"customer pricing", "PUT", customerPricingPath, map[string]any{}, "discounts:read", nil},
		{"manufacturing cost per unit", "PUT", manufacturingPath, withType(window, "costsPerUnit"), "invoices:read", []string{"invoices:read"}},
		{"manufacturing margin", "PUT", manufacturingPath, withType(window, "margin"), "invoices:read", []string{"invoices:read"}},
	}
	call := func(client *Client, method, path string, body map[string]any) (int, []byte) {
		if method == "GET" {
			return getAs(t, client, path)
		}
		status, _, raw := putAnalytics(t, client, path, nil, body)
		return status, raw
	}
	for _, r := range reports {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			status, raw := call(customRoleClient(t, r.formerPerm), r.method, r.path, r.body)
			requireStatus(t, http.StatusForbidden, status, raw)
			requireErrorResponse(t, raw, "insufficient_permissions", "invalid_request_error")

			status, raw = call(customRoleClient(t, append([]string{costsRead}, r.extraPerm...)...), r.method, r.path, r.body)
			requireStatus(t, http.StatusOK, status, raw)

			status, raw = call(apiClient, r.method, r.path, r.body)
			requireStatus(t, http.StatusOK, status, raw)

			for who, client := range map[string]*Client{"customer portal": getCustomerPortalClient(), "supplier portal": getSupplierPortalClient(t)} {
				status, raw = call(client, r.method, r.path, r.body)
				assert.Equal(t, http.StatusForbidden, status, "%s: %s", who, string(raw))
			}
		})
	}

	// A manufacturing metric that is not cost data keeps the report's own permission.
	status, _, raw := putAnalytics(t, customRoleClient(t, "invoices:read"), manufacturingPath, nil, withType(window, "production"))
	requireStatus(t, http.StatusOK, status, raw)
}

func withType(body map[string]any, metric string) map[string]any {
	out := map[string]any{"type": metric}
	for k, v := range body {
		out[k] = v
	}
	return out
}

// An audited change to a labor rate keeps its values from an audit reader without costs:read.
func TestCosts_AuditedCostChangesNeedCostsRead(t *testing.T) {
	t.Parallel()
	deptID := costDepartment(t)
	params := url.Values{"resource_ids": {deptID}, "include": {"changes"}}

	laborChange := func(client *Client) map[string]any {
		var change map[string]any
		eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
			for _, ev := range listRows(t, parseJSON(mustGetAs(t, client, auditEventsPath, params))) {
				for _, c := range jsonListData(ev, "changes") {
					if cm := c.(map[string]any); jsonField(cm, "field") == "labor_rate" {
						change = cm
						return nil
					}
				}
			}
			return fmt.Errorf("no labor_rate change recorded for %s yet", deptID)
		})
		return change
	}

	assert.NotNil(t, laborChange(apiClient)["new_value"], "the admin reads the rate the department was created with")
	assert.Nil(t, laborChange(customRoleClient(t, "audit_events:read"))["new_value"])
	assert.NotNil(t, laborChange(customRoleClient(t, "audit_events:read", costsRead))["new_value"])
}

// A request log is readable with request_logs:read alone, so the costs a request carried are masked there for everyone.
func TestCosts_RequestLogsKeepCostOut(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	status, body, err := apiClient.Post(productsPath+"?include=item,item.unit_cost", map[string]any{
		"sku":         uniqueName("e2e-costs-rqlog"),
		"type":        "sale",
		"category_id": SeedItemCategoryID,
		"unit_cost":   map[string]any{"value": "6.75", "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	created := parseJSON(body)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productsPath + "/" + jsonField(created, "id")) })
	require.NotNil(t, jsonObject(jsonObject(created, "item"), "unit_cost"), "the admin who set the cost reads it back")

	var logID string
	eventually(t, e2eRequestLogWaitTimeout, e2eRequestLogPollInterval, func() error {
		list, _, err := apiClient.GetList(requestLogsPath, url.Values{"idempotency_key": {idemKey}, "limit": {"1"}})
		if err != nil {
			return err
		}
		if len(list.Data) == 0 {
			return fmt.Errorf("no request log yet for idempotency key %s", idemKey)
		}
		logID = jsonField(parseJSON(list.Data[0]), "id")
		return nil
	})

	got := parseJSON(mustGetAs(t, apiClient, requestLogsPath+"/"+logID, url.Values{"include": {"request_body", "response_body"}}))
	assert.Equal(t, "****", jsonField(jsonObject(got, "request_body"), "unit_cost"), "the cost the request set")
	assert.Equal(t, "****", jsonField(jsonObject(jsonObject(got, "response_body"), "item"), "unit_cost"), "the cost the response carried")
}

// The file itself is out of reach in e2e (the test object store discards it); this proves an export without cost columns still renders. The columns left out are asserted in core-service's export tests.
func TestCosts_ExportsRenderWithoutCostColumns(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path  string
		perms []string
	}{
		{productsPath, []string{"items:read", "jobs:read"}},
		{materialsPath, []string{"materials:read", "jobs:read"}},
		{partsPath, []string{"parts:read", "jobs:read"}},
		{productionStepsPath, []string{"production_steps:read", "jobs:read"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			job := completedExportJobAs(t, customRoleClient(t, tc.perms...), tc.path+"/actions/export", nil)
			assert.NotNil(t, job["export"])
		})
	}
	job := completedExportJobAs(t, getCustomerPortalClient(), productsPath+"/actions/export", nil)
	assert.NotNil(t, job["export"], "a customer's product export renders, without the seller's unit cost")
}

// --- Portals ---

// A customer reads its own order with every line expansion the endpoint offers and still sees no cost.
func TestCosts_CustomerPortalOwnSalesOrderCarriesNoCost(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()
	var lineIncludes []string
	for _, include := range specIncludes(t, salesOrdersPath+"/{id}") {
		if strings.HasPrefix(include, "lines") {
			lineIncludes = append(lineIncludes, include)
		}
	}
	require.Contains(t, lineIncludes, "lines.unit_cost")

	status, body := getAs(t, portal, salesOrdersPath+"/"+SeedSalesOrderID, lineIncludes...)
	requireStatus(t, http.StatusOK, status, body)
	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines, "the customer's order has lines")
	for _, l := range lines {
		line := l.(map[string]any)
		assertNilField(t, line, "unit_cost")
		assert.NotNil(t, jsonObject(line, "unit_price"), "the customer sees what it is charged")
	}
	requireNoCost(t, "customer portal", "its own sales order", body)

	status, body = getAs(t, portal, salesOrdersPath, lineIncludes...)
	requireStatus(t, http.StatusOK, status, body)
	requireNoCost(t, "customer portal", "its sales order list", body)

	// Every include at once, as a portal can ask once every include rides on the order's own read.
	if status, body = getAs(t, portal, salesOrdersPath+"/"+SeedSalesOrderID, specIncludes(t, salesOrdersPath+"/{id}")...); status == http.StatusOK {
		requireNoCost(t, "customer portal", "its own sales order with every include", body)
	}
}

// The catalog a customer orders from, invoices, shipments and picks, with every expansion: no cost anywhere.
func TestCosts_CustomerPortalDocumentsCarryNoCost(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()
	for _, tc := range []struct {
		path     string
		specPath string
	}{
		{productsPath, productsPath},
		{productsPath + "/" + SeedProductID, productsPath + "/{id}"},
		{invoicesPath, invoicesPath},
		{invoicesPath + "/" + SeedInvoiceID, invoicesPath + "/{id}"},
		{shipmentsPath, shipmentsPath},
		{shipmentsPath + "/" + SeedShipmentID, shipmentsPath + "/{id}"},
		{picksPath, picksPath},
		{picksPath + "/" + SeedPickID, picksPath + "/{id}"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			status, body := getAs(t, portal, tc.path, specIncludes(t, tc.specPath)...)
			if strings.HasPrefix(tc.path, productsPath) {
				requireStatus(t, http.StatusOK, status, body)
			}
			if status != http.StatusOK {
				return
			}
			requireNoCost(t, "customer portal", tc.path, body)
		})
	}

	// The item a product carries is the one place the item's own cost is offered to a portal.
	status, body := getAs(t, portal, productsPath+"/"+SeedProductID, "item", "item.unit_cost")
	requireStatus(t, http.StatusOK, status, body)
	if item := jsonObject(parseJSON(body), "item"); item != nil {
		assertNilField(t, item, "unit_cost")
	}

	receipts := mustPutAnalytics(t, portal, inventoryReceiptsPath, nil, map[string]any{})
	requireNoCost(t, "customer portal", "inventory receipts", mustJSON(t, receipts))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestCosts_CustomerPortalSweepFindsNoCost(t *testing.T) {
	t.Parallel()
	portalSweep(t, "customer portal", getCustomerPortalClient())
}

func TestCosts_SupplierPortalSweepFindsNoCost(t *testing.T) {
	t.Parallel()
	portalSweep(t, "supplier portal", getSupplierPortalClient(t))
}

// portalSweep calls every GET the spec documents, as a portal actor, and fails on any cost field holding a value. Each endpoint is read bare, then with every include it offers at once, then (when one of them is refused) with each include on its own. A path id is the first record the portal's own list of that resource returned, or a seed record when it has no list.
func portalSweep(t *testing.T, who string, client *Client) {
	t.Helper()
	spec, err := LoadFullSpec()
	require.NoError(t, err)

	paths := make([]string, 0, len(spec.Paths))
	for p, methods := range spec.Paths {
		if _, ok := methods["get"]; ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)

	firstIDs := map[string]string{}
	reached, expanded := 0, 0
	for _, specPath := range paths {
		if portalSweepSkipped(specPath) {
			continue
		}
		path, ok := portalSweepPath(specPath, firstIDs)
		if !ok {
			continue
		}
		status, body := getAs(t, client, path)
		if status != http.StatusOK || !json.Valid(body) {
			continue
		}
		reached++
		requireNoCost(t, who, "GET "+specPath, body)
		if id := firstListID(body); id != "" {
			firstIDs[specPath] = id
		}

		includes := operationIncludes(spec.Paths[specPath]["get"])
		if len(includes) == 0 {
			continue
		}
		if status, body = getAs(t, client, path, includes...); status == http.StatusOK {
			expanded++
			requireNoCost(t, who, "GET "+specPath+" with every include", body)
			continue
		}
		for _, include := range includes {
			if status, body = getAs(t, client, path, include); status == http.StatusOK {
				expanded++
				requireNoCost(t, who, "GET "+specPath+"?include="+include, body)
			}
		}
	}
	assert.Positive(t, reached, "%s reached no endpoint at all, so the sweep checked nothing", who)
	t.Logf("%s: %d GET endpoints answered, %d of them with includes expanded", who, reached, expanded)
}

func portalSweepPath(specPath string, firstIDs map[string]string) (string, bool) {
	params := pathParamsOf(specPath)
	if i := strings.LastIndex(specPath, "/"); len(params) == 1 && strings.HasSuffix(specPath, "}") {
		if id := firstIDs[specPath[:i]]; id != "" {
			return specPath[:i] + "/" + id, true
		}
	}
	return (&ListEndpointSpec{Path: specPath, PathParams: params}).ResolvePath()
}

func firstListID(body []byte) string {
	parsed := parseJSON(body)
	if jsonField(parsed, "object") != "list" {
		return ""
	}
	data := jsonArray(parsed, "data")
	if len(data) == 0 {
		return ""
	}
	row, ok := data[0].(map[string]any)
	if !ok {
		return ""
	}
	return jsonField(row, "id")
}

// portalSweepSkipped names GETs that answer with a redirect to an image rather than a resource.
func portalSweepSkipped(path string) bool {
	for _, marker := range []string{"/favicon", "/logo"} {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func pathParamsOf(path string) []string {
	var params []string
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			params = append(params, strings.TrimSuffix(strings.TrimPrefix(seg, "{"), "}"))
		}
	}
	return params
}

func operationIncludes(op openAPIOperation) []string {
	for _, p := range op.Parameters {
		if (p.Name != "include[]" && p.Name != "include") || p.Schema == nil {
			continue
		}
		enum := p.Schema.Enum
		if p.Schema.Items != nil {
			enum = p.Schema.Items.Enum
		}
		out := make([]string, 0, len(enum))
		for _, v := range enum {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func specIncludes(t *testing.T, specPath string) []string {
	t.Helper()
	spec, err := LoadFullSpec()
	require.NoError(t, err)
	op, ok := spec.Paths[specPath]["get"]
	require.True(t, ok, "the spec documents GET %s", specPath)
	return operationIncludes(op)
}
