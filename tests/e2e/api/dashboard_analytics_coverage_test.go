//go:build e2e

package api_test

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Who may run each dashboard report, what each filter the pages send does, and that another tenant sees none of the seller's data.

const (
	dashAnalyticsPastStart = "2021-03-01T00:00:00Z"
	dashAnalyticsPastEnd   = "2021-03-31T23:59:59Z"

	// Another seeded product line, one the seed socks are not on.
	dashAnalyticsOtherProductLineID = "pdln_01k0a735ypfjva933tg57wfx0t"
)

// dashAnalyticsPastWindow is a window long past, so no test's fixtures fall in it.
func dashAnalyticsPastWindow(extra map[string]any) map[string]any {
	body := map[string]any{"starts_at": dashAnalyticsPastStart, "ends_at": dashAnalyticsPastEnd}
	maps.Copy(body, extra)
	return body
}

func dashAnalyticsWith(base map[string]any, extra map[string]any) map[string]any {
	body := maps.Clone(base)
	maps.Copy(body, extra)
	return body
}

type dashAnalyticsReport struct {
	name   string
	method string
	path   string
	body   map[string]any
}

// dashAnalyticsInternalReports is every report the dashboard calls that only the seller's own staff may run, each with a valid request.
func dashAnalyticsInternalReports() []dashAnalyticsReport {
	put := func(name, path string, body map[string]any) dashAnalyticsReport {
		return dashAnalyticsReport{name: name, method: "PUT", path: path, body: body}
	}
	return []dashAnalyticsReport{
		put("customer-pricing", customerPricingPath, map[string]any{}),
		put("delivery-performance", deliveryPerformancePath, dashAnalyticsPastWindow(nil)),
		put("demand-forecast", demandForecastPath, map[string]any{}),
		put("materials", analyticsMaterialsPath, map[string]any{}),
		put("new-customers-table", newCustomersPath, dashAnalyticsPastWindow(nil)),
		put("oee", analyticsOeePath, dashAnalyticsPastWindow(nil)),
		put("oee-trend", analyticsOeeTrendPath, dashAnalyticsPastWindow(nil)),
		put("open-batches", openBatchesPath, map[string]any{}),
		put("open-orders", openOrdersPath, map[string]any{}),
		put("open-orders-breakdown", openOrdersBreakdownPath, map[string]any{}),
		put("open-orders-summary", openOrdersSummaryPath, map[string]any{}),
		put("production-costs", productionCostsPath, dashAnalyticsPastWindow(nil)),
		put("quarterly-orders", quarterlyOrdersPath, map[string]any{}),
		put("realized-margins", realizedMarginsPath, dashAnalyticsPastWindow(nil)),
		put("sales-breakdown", salesBreakdownPath, dashAnalyticsPastWindow(map[string]any{"group_by": "customer"})),
		put("sales-invoices", salesInvoicesPath, dashAnalyticsPastWindow(nil)),
		put("sales-lines", salesLinesPath, dashAnalyticsPastWindow(nil)),
		put("sales-summary", salesSummaryPath, dashAnalyticsPastWindow(nil)),
		put("schedule-attainment", scheduleAttainmentPath, dashAnalyticsPastWindow(nil)),
		{name: "weeks-of-sales", method: "GET", path: weeksOfSalesPath},
		{name: "open-order-lines", method: "GET", path: openOrdersPath + "/" + SeedSalesOrderID + "/lines"},
	}
}

func dashAnalyticsCall(t *testing.T, client *Client, r dashAnalyticsReport) (int, []byte) {
	t.Helper()
	var (
		status int
		body   []byte
		err    error
	)
	if r.method == "GET" {
		status, body, err = client.GetListRaw(r.path, nil)
	} else {
		status, body, err = client.PutRaw(r.path, nil, r.body)
	}
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", r.path, string(body))
	return status, body
}

// --- Who may run a report ---

// A customer's portal key, and the seller's own key aimed at a customer account, are not the seller's staff.
func TestDashAnalytics_ReportsRefuseCustomerPortalAndCrossAccountActors(t *testing.T) {
	t.Parallel()
	actors := map[string]*Client{
		"customer portal":             getCustomerPortalClient(),
		"seller targeting a customer": apiClient.WithAccountID(SeedCustomerAccountID),
	}
	for _, r := range dashAnalyticsInternalReports() {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			for actor, client := range actors {
				status, raw := dashAnalyticsCall(t, client, r)
				require.Equal(t, 403, status, "%s: %s", actor, string(raw))
				requireErrorResponse(t, raw, "insufficient_permissions", "invalid_request_error")
			}
		})
	}
}

// Inventory receipts answer a customer with the stock it owns or holds, never the seller's own.
func TestDashAnalytics_InventoryReceiptsShowACustomerOnlyItsOwnStock(t *testing.T) {
	t.Parallel()
	_, item := parityYarn(t, nil)
	receiveInto(t, item, "2", poundUnitID, "", "")
	awaitRemaining(t, item, "2")

	for actor, client := range map[string]*Client{
		"customer portal":             getCustomerPortalClient(),
		"seller targeting a customer": apiClient.WithAccountID(SeedCustomerAccountID),
	} {
		assert.Empty(t, receiptSummaries(t, client, map[string]any{"item_ids": []string{item}}), "%s sees the seller's stock", actor)
		for _, row := range receiptSummaries(t, client, nil) {
			owner, holder := entityID(row, "owner_account"), entityID(row, "holder_account")
			assert.True(t, owner == SeedCustomerAccountID || holder == SeedCustomerAccountID,
				"%s: a receipt the customer neither owns nor holds (owner %s, holder %s)", actor, owner, holder)
		}
	}
}

// --- Validation ---

// An end before the start is a client mistake, as the sibling reports already answer it.
func TestDashAnalytics_InvertedWindowsAreRejected(t *testing.T) {
	t.Parallel()
	inverted := map[string]any{"starts_at": dashAnalyticsPastEnd, "ends_at": dashAnalyticsPastStart}
	cases := []struct {
		name, path, param string
		body              map[string]any
	}{
		{"sales-summary", salesSummaryPath, "ends_at", inverted},
		{"sales-summary comparison", salesSummaryPath, "comparison_ends_at", dashAnalyticsPastWindow(map[string]any{
			"comparison_starts_at": "2020-03-31T00:00:00Z", "comparison_ends_at": "2020-03-01T00:00:00Z",
		})},
		{"sales-breakdown", salesBreakdownPath, "ends_at", dashAnalyticsWith(inverted, map[string]any{"group_by": "customer"})},
		{"sales-invoices", salesInvoicesPath, "ends_at", inverted},
		{"sales-lines", salesLinesPath, "ends_at", inverted},
		{"oee", analyticsOeePath, "ends_at", inverted},
		{"oee-trend", analyticsOeeTrendPath, "ends_at", inverted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, _, raw := putAnalytics(t, apiClient, tc.path, nil, tc.body)
			require.Equal(t, 400, status, string(raw))
			assertErrorParam(t, requireErrorResponse(t, raw, "", "invalid_request_error"), tc.param)
		})
	}
}

func TestDashAnalytics_MalformedReportRequestsAreRejected(t *testing.T) {
	t.Parallel()
	badDate := map[string]any{"starts_at": "last tuesday", "ends_at": dashAnalyticsPastEnd}
	cases := []struct {
		name, method, path, param string
		params                    url.Values
		body                      map[string]any
	}{
		// Unparseable dates.
		{"sales-summary bad date", "PUT", salesSummaryPath, "starts_at", nil, badDate},
		{"sales-breakdown bad date", "PUT", salesBreakdownPath, "starts_at", nil, dashAnalyticsWith(badDate, map[string]any{"group_by": "customer"})},
		{"sales-invoices bad date", "PUT", salesInvoicesPath, "starts_at", nil, badDate},
		{"sales-lines bad date", "PUT", salesLinesPath, "starts_at", nil, badDate},
		{"new-customers-table bad date", "PUT", newCustomersPath, "starts_at", nil, badDate},
		{"oee bad date", "PUT", analyticsOeePath, "starts_at", nil, badDate},
		{"oee-trend bad date", "PUT", analyticsOeeTrendPath, "starts_at", nil, badDate},
		{"production-costs bad date", "PUT", productionCostsPath, "starts_at", nil, badDate},
		{"realized-margins bad date", "PUT", realizedMarginsPath, "starts_at", nil, badDate},
		{"schedule-attainment bad date", "PUT", scheduleAttainmentPath, "starts_at", nil, badDate},
		{"sales-summary impossible calendar date", "PUT", salesSummaryPath, "ends_at", nil, map[string]any{"starts_at": dashAnalyticsPastStart, "ends_at": "2021-13-45T00:00:00Z"}},

		// Missing window bounds.
		{"oee-trend without a start", "PUT", analyticsOeeTrendPath, "starts_at", nil, map[string]any{"ends_at": dashAnalyticsPastEnd}},
		{"oee-trend without an end", "PUT", analyticsOeeTrendPath, "ends_at", nil, map[string]any{"starts_at": dashAnalyticsPastStart}},
		{"oee without an end", "PUT", analyticsOeePath, "ends_at", nil, map[string]any{"starts_at": dashAnalyticsPastStart}},
		{"sales-breakdown without an end", "PUT", salesBreakdownPath, "ends_at", nil, map[string]any{"starts_at": dashAnalyticsPastStart, "group_by": "customer"}},
		{"schedule-attainment without a start", "PUT", scheduleAttainmentPath, "starts_at", nil, map[string]any{"ends_at": dashAnalyticsPastEnd}},

		// A filter that is not a list of ids.
		{"customer-pricing customer_ids", "PUT", customerPricingPath, "customer_ids", nil, map[string]any{"customer_ids": SeedCustomerAccountID}},
		{"demand-forecast item_ids", "PUT", demandForecastPath, "item_ids", nil, map[string]any{"item_ids": SeedItemID}},
		{"inventory-receipts location_ids", "PUT", inventoryReceiptsPath, "location_ids", nil, map[string]any{"location_ids": SeedLocationID}},
		{"materials supplier_ids", "PUT", analyticsMaterialsPath, "supplier_ids", nil, map[string]any{"supplier_ids": SeedSupplierAccountID}},
		{"open-batches item_ids of numbers", "PUT", openBatchesPath, "item_ids[0]", nil, map[string]any{"item_ids": []int{1, 2}}},
		{"open-orders-summary customer_ids", "PUT", openOrdersSummaryPath, "customer_ids", nil, map[string]any{"customer_ids": SeedCustomerAccountID}},
		{"quarterly-orders item_ids", "PUT", quarterlyOrdersPath, "item_ids", nil, map[string]any{"item_ids": SeedItemID}},
		{"sales-summary sales_rep_ids", "PUT", salesSummaryPath, "sales_rep_ids", nil, dashAnalyticsPastWindow(map[string]any{"sales_rep_ids": SeedAccountUserID})},
		{"new-customers-table customer_group_ids", "PUT", newCustomersPath, "customer_group_ids", nil, dashAnalyticsPastWindow(map[string]any{"customer_group_ids": SeedCustomerGroupID})},
		{"oee department_ids", "PUT", analyticsOeePath, "department_ids", nil, dashAnalyticsPastWindow(map[string]any{"department_ids": SeedDepartmentID})},
		{"schedule-attainment machine_ids", "PUT", scheduleAttainmentPath, "machine_ids", nil, dashAnalyticsPastWindow(map[string]any{"machine_ids": SeedMachineID})},
		{"realized-margins product_line_ids", "PUT", realizedMarginsPath, "product_line_ids", nil, dashAnalyticsPastWindow(map[string]any{"product_line_ids": SeedProductLineID})},
		{"delivery-performance customer_ids", "PUT", deliveryPerformancePath, "customer_ids", nil, dashAnalyticsPastWindow(map[string]any{"customer_ids": SeedCustomerAccountID})},
		{"production-costs category_ids", "PUT", productionCostsPath, "category_ids", nil, dashAnalyticsPastWindow(map[string]any{"category_ids": SeedItemCategoryID})},

		// Optional enums and windows refuse an explicit null.
		{"schedule-attainment null group_by", "PUT", scheduleAttainmentPath, "group_by", nil, dashAnalyticsPastWindow(map[string]any{"group_by": nil})},
		{"delivery-performance null granularity", "PUT", deliveryPerformancePath, "granularity", nil, dashAnalyticsPastWindow(map[string]any{"granularity": nil})},
		{"sales-summary null comparison", "PUT", salesSummaryPath, "comparison_starts_at", nil, dashAnalyticsPastWindow(map[string]any{"comparison_starts_at": nil, "comparison_ends_at": nil})},
		{"sales-summary offset below range", "PUT", salesSummaryPath, "tz_offset_minutes", nil, dashAnalyticsPastWindow(map[string]any{"tz_offset_minutes": -841})},

		// Page sizes and periods out of range.
		{"new-customers-table zero limit", "PUT", newCustomersPath, "limit", url.Values{"limit": {"0"}}, dashAnalyticsPastWindow(nil)},
		{"new-customers-table limit over 1000", "PUT", newCustomersPath, "limit", url.Values{"limit": {"1001"}}, dashAnalyticsPastWindow(nil)},
		{"new-customers-table forged cursor", "PUT", newCustomersPath, "cursor", url.Values{"cursor": {"not-a-cursor"}}, dashAnalyticsPastWindow(nil)},
		{"sales-invoices limit over 100", "PUT", salesInvoicesPath, "limit", url.Values{"limit": {"101"}}, dashAnalyticsPastWindow(nil)},
		{"sales-invoices forged cursor", "PUT", salesInvoicesPath, "cursor", url.Values{"cursor": {"not-a-cursor"}}, dashAnalyticsPastWindow(nil)},
		{"sales-lines limit over 500", "PUT", salesLinesPath, "limit", url.Values{"limit": {"501"}}, dashAnalyticsPastWindow(nil)},
		{"open-orders limit over 100", "PUT", openOrdersPath, "limit", url.Values{"limit": {"101"}}, map[string]any{}},
		{"weeks-of-sales period over 520", "GET", weeksOfSalesPath, "period_in_weeks", url.Values{"period_in_weeks": {"521"}}, nil},
		{"weeks-of-sales unknown query parameter", "GET", weeksOfSalesPath, bogusE2EQueryParam, url.Values{bogusE2EQueryParam: {"1"}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var (
				status int
				raw    []byte
				err    error
			)
			if tc.method == "GET" {
				status, raw, err = apiClient.GetListRaw(tc.path, tc.params)
			} else {
				status, raw, err = apiClient.PutRaw(tc.path, tc.params, tc.body)
			}
			require.NoError(t, err)
			require.Equal(t, 400, status, string(raw))
			assertErrorParam(t, requireErrorResponse(t, raw, "", "invalid_request_error"), tc.param)
		})
	}
}

// --- Sales reports ---

// dashAnalyticsSale is one invoiced order of 5 pr at $4.25/pr, sold by the seed account user, to a customer alone in a group of its own.
type dashAnalyticsSale struct {
	customerID, groupID string
}

const dashAnalyticsSaleRevenue = "21.25"

func dashAnalyticsShipSale(t *testing.T) dashAnalyticsSale {
	t.Helper()
	groupID := newAccountGroup(t)
	customerID := parityCustomer(t, groupID)
	order := issueParityOrder(t, customerID, SeedAccountUserID, parityLine(SeedProductID, "5", SeedUnitID, "4.25"))
	shipWholeOrder(t, jsonField(order, "id"))
	awaitSalesSummary(t, customerID, 1)
	return dashAnalyticsSale{customerID: customerID, groupID: groupID}
}

// dashAnalyticsSalesView is what each of the four sales reports returns for one filter.
type dashAnalyticsSalesView struct {
	revenue      string
	invoiceCount string
	groupKeys    []string
	invoiceIDs   []string
	lineCustomer []string
}

func dashAnalyticsReadSales(t *testing.T, client *Client, body map[string]any, groupBy string) dashAnalyticsSalesView {
	t.Helper()
	var v dashAnalyticsSalesView
	summary := mustPutAnalytics(t, client, salesSummaryPath, nil, body)
	v.revenue = computedValue(t, jsonObject(summary, "overall"), "revenue")
	v.invoiceCount = jsonField(jsonObject(summary, "overall"), "invoice_count")
	for _, g := range listRows(t, mustPutAnalytics(t, client, salesBreakdownPath, url.Values{"limit": {"100"}}, dashAnalyticsWith(body, map[string]any{"group_by": groupBy}))) {
		v.groupKeys = append(v.groupKeys, jsonField(g, "key"))
	}
	for _, inv := range listRows(t, mustPutAnalytics(t, client, salesInvoicesPath, url.Values{"limit": {"100"}}, body)) {
		v.invoiceIDs = append(v.invoiceIDs, jsonField(inv, "id"))
	}
	for _, line := range listRows(t, mustPutAnalytics(t, client, salesLinesPath, url.Values{"limit": {"500"}}, body)) {
		v.lineCustomer = append(v.lineCustomer, jsonField(line, "customer_id"))
	}
	return v
}

func dashAnalyticsRealizedFindingIDs(t *testing.T, client *Client, body map[string]any) []string {
	t.Helper()
	var ids []string
	for _, f := range listRows(t, jsonObject(mustPutAnalytics(t, client, realizedMarginsPath, nil, body), "findings")) {
		ids = append(ids, jsonField(f, "id"))
	}
	return ids
}

// dashAnalyticsLiveWindow is a day either side of now, to the second: realized margins cache by window, so a fresh one is never another test's stale answer.
func dashAnalyticsLiveWindow() map[string]any {
	now := time.Now().UTC()
	return map[string]any{"starts_at": rfc3339(now.Add(-24 * time.Hour)), "ends_at": rfc3339(now.Add(24 * time.Hour))}
}

func TestDashAnalytics_SalesReportsHonorEveryFilterTheDashboardSends(t *testing.T) {
	t.Parallel()
	sale := dashAnalyticsShipSale(t)
	otherGroup := newAccountGroup(t)
	window := dashAnalyticsLiveWindow()
	scoped := func(extra map[string]any) map[string]any {
		return dashAnalyticsWith(window, dashAnalyticsWith(map[string]any{"product_line_ids": []string{SeedProductLineID}}, extra))
	}
	mine := []string{sale.customerID}

	for name, tc := range map[string]struct {
		body  map[string]any
		match bool
	}{
		"its customer group":          {scoped(map[string]any{"customer_group_ids": []string{sale.groupID}}), true},
		"another customer group":      {scoped(map[string]any{"customer_ids": mine, "customer_group_ids": []string{otherGroup}}), false},
		"its sales rep":               {scoped(map[string]any{"customer_ids": mine, "sales_rep_ids": []string{SeedAccountUserID}}), true},
		"another sales rep":           {scoped(map[string]any{"customer_ids": mine, "sales_rep_ids": []string{"acus_dashanalyticsnone"}}), false},
		"its item":                    {scoped(map[string]any{"customer_ids": mine, "item_ids": []string{SeedItemID}}), true},
		"another item":                {scoped(map[string]any{"customer_ids": mine, "item_ids": []string{parityItemSCK002}}), false},
		"another product line":        {dashAnalyticsWith(window, map[string]any{"customer_ids": mine, "product_line_ids": []string{dashAnalyticsOtherProductLineID}}), false},
		"a window before it was sold": {dashAnalyticsWith(scoped(map[string]any{"customer_ids": mine}), map[string]any{"starts_at": dashAnalyticsPastStart, "ends_at": dashAnalyticsPastEnd}), false},
	} {
		t.Run(name, func(t *testing.T) {
			got := dashAnalyticsReadSales(t, apiClient, tc.body, "customer")
			if tc.match {
				assert.True(t, decimal.RequireFromString(dashAnalyticsSaleRevenue).Equal(decimal.RequireFromString(got.revenue)), "revenue %s", got.revenue)
				assert.Equal(t, "1", got.invoiceCount)
				assert.Equal(t, mine, got.groupKeys)
				assert.Len(t, got.invoiceIDs, 1)
				assert.Equal(t, mine, got.lineCustomer)
				return
			}
			assert.Equal(t, "0", got.revenue)
			assert.Equal(t, "0", got.invoiceCount)
			assert.Empty(t, got.groupKeys)
			assert.Empty(t, got.invoiceIDs)
			assert.Empty(t, got.lineCustomer)
		})
	}

	t.Run("breakdown keys name the rep and the group", func(t *testing.T) {
		body := scoped(map[string]any{"customer_ids": mine})
		assert.Equal(t, []string{SeedAccountUserID}, dashAnalyticsReadSales(t, apiClient, body, "sales_rep").groupKeys)
		assert.Equal(t, []string{sale.groupID}, dashAnalyticsReadSales(t, apiClient, body, "customer_group").groupKeys)
	})

	t.Run("lines without a window list every invoiced line", func(t *testing.T) {
		rows := listRows(t, mustPutAnalytics(t, apiClient, salesLinesPath, nil, map[string]any{"customer_ids": mine, "product_line_ids": []string{SeedProductLineID}}))
		require.Len(t, rows, 1)
		assert.Equal(t, "5", jsonField(rows[0], "quantity_invoiced"))
	})

	t.Run("realized margins", func(t *testing.T) {
		want := sale.customerID + ":" + SeedItemID
		// $4.25 a pair is under the sock's captured cost, so the sale is a finding.
		var live map[string]any
		eventually(t, 45*time.Second, time.Second, func() error {
			live = dashAnalyticsLiveWindow()
			if ids := dashAnalyticsRealizedFindingIDs(t, apiClient, dashAnalyticsWith(live, map[string]any{"customer_ids": mine})); !slices.Equal(ids, []string{want}) {
				return fmt.Errorf("findings %v, want only %s", ids, want)
			}
			return nil
		})
		assert.Equal(t, []string{want}, dashAnalyticsRealizedFindingIDs(t, apiClient, dashAnalyticsWith(live, map[string]any{"customer_group_ids": []string{sale.groupID}})))
		assert.Empty(t, dashAnalyticsRealizedFindingIDs(t, apiClient, dashAnalyticsWith(live, map[string]any{"customer_group_ids": []string{otherGroup}})))
		assert.Empty(t, dashAnalyticsRealizedFindingIDs(t, apiClient, dashAnalyticsWith(live, map[string]any{
			"customer_ids": mine, "product_line_ids": []string{dashAnalyticsOtherProductLineID},
		})))
	})
}

func TestDashAnalytics_SalesReportsShowTenantBNoneOfTheSellersSales(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)
	awaitNewCustomer(t, sale.customerID)
	tenantB := getTenantBClient()
	scoped := saleFilter(sale.customerID)
	window := map[string]any{"starts_at": scoped["starts_at"], "ends_at": scoped["ends_at"]}

	seller := dashAnalyticsReadSales(t, apiClient, scoped, "customer")
	require.Len(t, seller.invoiceIDs, 1, "precondition: the seller sees its sale")

	got := dashAnalyticsReadSales(t, tenantB, scoped, "customer")
	assert.Equal(t, "0", got.revenue)
	assert.Equal(t, "0", got.invoiceCount)
	assert.Empty(t, got.groupKeys)
	assert.Empty(t, got.invoiceIDs)
	assert.Empty(t, got.lineCustomer)

	unfiltered := dashAnalyticsReadSales(t, tenantB, window, "customer")
	assert.NotContains(t, unfiltered.invoiceIDs, seller.invoiceIDs[0])
	assert.NotContains(t, unfiltered.groupKeys, sale.customerID)
	assert.NotContains(t, unfiltered.lineCustomer, sale.customerID)

	t.Run("new customers", func(t *testing.T) {
		now := time.Now().UTC()
		list := mustPutAnalytics(t, tenantB, newCustomersPath, url.Values{"limit": {"1000"}}, map[string]any{
			"starts_at": rfc3339(now.Add(-time.Hour)), "ends_at": rfc3339(now.Add(time.Hour)),
		})
		assert.Nil(t, newCustomerRow(list, sale.customerID))
	})

	t.Run("realized margins", func(t *testing.T) {
		want := sale.customerID + ":" + SeedItemID
		var live map[string]any
		eventually(t, 45*time.Second, time.Second, func() error {
			live = dashAnalyticsLiveWindow()
			if ids := dashAnalyticsRealizedFindingIDs(t, apiClient, dashAnalyticsWith(live, map[string]any{"customer_ids": []string{sale.customerID}})); !slices.Contains(ids, want) {
				return fmt.Errorf("the seller's finding %s is not reported yet: %v", want, ids)
			}
			return nil
		})
		assert.Empty(t, dashAnalyticsRealizedFindingIDs(t, tenantB, dashAnalyticsWith(live, map[string]any{"customer_ids": []string{sale.customerID}})))
		for _, id := range dashAnalyticsRealizedFindingIDs(t, tenantB, live) {
			assert.False(t, strings.HasPrefix(id, sale.customerID), "tenant B sees the seller's customer: %s", id)
		}
	})
}

// --- Open orders and quarterly orders ---

func TestDashAnalytics_OpenOrderReportsShowTenantBNoneOfTheSellersBook(t *testing.T) {
	t.Parallel()
	customer := parityCustomer(t, "")
	orderID := jsonField(issueParityOrder(t, customer, "", parityLine(SeedProductID, "3", SeedUnitID, "2.00")), "id") // 6 ea x $1
	body := map[string]any{"customer_ids": []string{customer}}
	tenantB := getTenantBClient()

	assert.Equal(t, "6", computedValue(t, mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, body), "ordered"), "precondition: the seller sees its order")
	status, _, raw := openOrderLines(t, apiClient, orderID)
	requireStatus(t, 200, status, raw)

	assert.Equal(t, "0", computedValue(t, mustPutAnalytics(t, tenantB, openOrdersSummaryPath, nil, body), "ordered"))
	assert.Empty(t, listRows(t, mustPutAnalytics(t, tenantB, openOrdersBreakdownPath, nil, body)))
	assert.Empty(t, listRows(t, mustPutAnalytics(t, tenantB, openOrdersPath, nil, body)))
	for _, row := range listRows(t, mustPutAnalytics(t, tenantB, openOrdersPath, url.Values{"limit": {"100"}}, nil)) {
		assert.NotEqual(t, orderID, jsonField(jsonObject(row, "order"), "id"))
	}

	status, _, raw = openOrderLines(t, tenantB, orderID)
	require.Equal(t, 404, status, string(raw))
	requireErrorResponse(t, raw, "resource_not_found", "invalid_request_error")
}

func TestDashAnalytics_OpenOrderLinesOfANonOrderIDAre404(t *testing.T) {
	t.Parallel()
	for name, id := range map[string]string{
		"a customer": SeedCustomerAccountID,
		"an item":    SeedItemID,
		"a product":  SeedProductID,
		"an invoice": SeedInvoiceID,
	} {
		status, _, raw := openOrderLines(t, apiClient, id)
		require.Equal(t, 404, status, "%s: %s", name, string(raw))
		requireErrorResponse(t, raw, "resource_not_found", "invalid_request_error")
	}
}

func TestDashAnalytics_QuarterlyOrdersShowTenantBNoneOfTheSellersOrders(t *testing.T) {
	t.Parallel()
	customer := parityCustomer(t, "")
	issueParityOrder(t, customer, "", parityLine(SeedProductID, "1", SeedUnitID, "10.00")) // 2 ea x $5
	body := map[string]any{"customer_ids": []string{customer}}
	tenantB := getTenantBClient()

	assertOnlyThisQuarter(t, quarterlyOrders(t, apiClient, body), "10")
	assert.Empty(t, quarterlyOrders(t, tenantB, body))
}

// --- Inventory and catalog reports ---

func TestDashAnalytics_InventoryReportsShowTenantBNoneOfTheSellersStock(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	t.Run("inventory receipts and materials", func(t *testing.T) {
		t.Parallel()
		material, item := parityYarn(t, nil)
		receiveInto(t, item, "2", poundUnitID, "", "")
		awaitRemaining(t, item, "2")
		materialRow(t, material)

		assert.Empty(t, receiptSummaries(t, tenantB, map[string]any{"item_ids": []string{item}}))
		for _, row := range receiptSummaries(t, tenantB, nil) {
			assert.NotEqual(t, item, entityID(row, "item"))
			assert.NotEqual(t, SeedAccountID, entityID(row, "owner_account"))
			assert.NotEqual(t, SeedAccountID, entityID(row, "holder_account"))
		}
		for _, row := range listRows(t, mustPutAnalytics(t, tenantB, analyticsMaterialsPath, nil, map[string]any{"supplier_ids": []string{SeedSupplierAccountID}})) {
			assert.NotContains(t, []string{material, SeedMaterialID}, jsonField(row, "id"))
		}
	})

	t.Run("weeks of sales", func(t *testing.T) {
		t.Parallel()
		lines := func(client *Client) []string {
			status, raw, err := client.GetListRaw(weeksOfSalesPath, url.Values{"period_in_weeks": {"13"}})
			require.NoError(t, err)
			requireStatus(t, 200, status, raw)
			var ids []string
			for _, r := range jsonArray(parseJSON(raw), "data") {
				ids = append(ids, entityID(r.(map[string]any), "product_line"))
			}
			return ids
		}
		require.Contains(t, lines(apiClient), SeedProductLineID, "precondition: the seller's socks are reported")
		assert.NotContains(t, lines(tenantB), SeedProductLineID)
	})

	t.Run("open batches", func(t *testing.T) {
		t.Parallel()
		seller := listRows(t, mustPutAnalytics(t, apiClient, openBatchesPath, nil, nil))
		require.NotEmpty(t, seller, "precondition: the seller has open batches")
		stations, items := map[string]bool{}, []string{}
		for _, r := range seller {
			stations[entityID(r, "scanning_station")] = true
			items = append(items, entityID(r, "item"))
		}
		assert.Empty(t, listRows(t, mustPutAnalytics(t, tenantB, openBatchesPath, nil, map[string]any{"item_ids": items})))
		for _, r := range listRows(t, mustPutAnalytics(t, tenantB, openBatchesPath, nil, nil)) {
			assert.False(t, stations[entityID(r, "scanning_station")], "tenant B sees the seller's station %s", entityID(r, "scanning_station"))
		}
	})

	t.Run("demand forecast", func(t *testing.T) {
		t.Parallel()
		body := map[string]any{"item_ids": []string{SeedItemID}}
		require.NotNil(t, forecastRow(t, SeedItemID, body), "precondition: the seller's socks are forecast")
		got := mustPutAnalytics(t, tenantB, demandForecastPath, nil, body)
		assert.Empty(t, listRows(t, jsonObject(got, "data")))
		for _, r := range listRows(t, jsonObject(mustPutAnalytics(t, tenantB, demandForecastPath, nil, nil), "data")) {
			assert.NotEqual(t, SeedItemID, entityID(r, "item"))
		}
	})
}

// supplier_ids does not narrow the materials; it adds those suppliers' names and part numbers to the materials they supply.
func TestDashAnalytics_MaterialsSupplierIDsAddSupplierDetail(t *testing.T) {
	t.Parallel()
	supplierName := jsonField(parseJSON(mustGet(t, suppliersPath+"/"+SeedSupplierAccountID)), "name")
	require.NotEmpty(t, supplierName)
	row := func(body map[string]any) map[string]any {
		for _, r := range listRows(t, mustPutAnalytics(t, apiClient, analyticsMaterialsPath, nil, body)) {
			if jsonField(r, "id") == SeedMaterialID {
				return r
			}
		}
		t.Fatalf("seed material %s is not in the analytics", SeedMaterialID)
		return nil
	}

	withSupplier := row(map[string]any{"supplier_ids": []string{SeedSupplierAccountID}})
	assert.Equal(t, []any{supplierName}, withSupplier["supplier_names"])
	parts, _ := withSupplier["supplier_part_numbers"].([]any)
	assert.Len(t, parts, 1)

	without := row(nil)
	assert.Empty(t, without["supplier_names"])
	assert.Empty(t, without["supplier_part_numbers"])

	unknown := row(map[string]any{"supplier_ids": []string{"ac_dashanalyticsnone"}})
	assert.Empty(t, unknown["supplier_names"], "a supplier that supplies nothing adds nothing")
}

func TestDashAnalytics_DemandForecastHistoryAndHorizonFollowTheRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ history, forecast int }{{1, 2}, {3, 6}} {
		r := forecastRow(t, SeedItemID, map[string]any{"item_ids": []string{SeedItemID}, "history_months": tc.history, "forecast_months": tc.forecast})
		require.NotNil(t, r, "the seed socks have demand")
		assert.LessOrEqual(t, len(jsonArray(r, "history")), tc.history, "history is bounded by history_months")
		assert.Len(t, jsonArray(r, "forecast"), tc.forecast)
		assert.Len(t, jsonArray(r, "revenue_forecast"), tc.forecast)
	}
}

// --- Production reports ---

func TestDashAnalytics_ProductionReportsShowTenantBNoneOfTheSellersPlant(t *testing.T) {
	t.Parallel()
	start, end := oeeSeededWindow(t, 24*time.Hour)
	window := map[string]any{"starts_at": rfc3339(start), "ends_at": rfc3339(end)}
	scoped := dashAnalyticsWith(window, map[string]any{"department_ids": []string{SeedDepartmentID}})
	tenantB := getTenantBClient()

	t.Run("oee", func(t *testing.T) {
		require.NotNil(t, findOeeDepartment(analyzeOee(t, scoped), SeedDepartmentID), "precondition: the seed department produced in the window")
		assert.Empty(t, jsonListData(mustPutAnalytics(t, tenantB, analyticsOeePath, nil, scoped), "departments"))
		for _, raw := range jsonListData(mustPutAnalytics(t, tenantB, analyticsOeePath, nil, window), "departments") {
			assert.NotEqual(t, SeedDepartmentID, entityID(raw.(map[string]any), "department"))
		}
	})

	t.Run("oee trend", func(t *testing.T) {
		goodUnits := func(resp map[string]any) float64 {
			var sum float64
			for _, raw := range jsonListData(resp, "periods") {
				n, _ := raw.(map[string]any)["good_units"].(float64)
				sum += n
			}
			return sum
		}
		require.Positive(t, goodUnits(analyzeOeeTrend(t, scoped)), "precondition: the seed department produced in the window")
		got := mustPutAnalytics(t, tenantB, analyticsOeeTrendPath, nil, scoped)
		assert.Zero(t, goodUnits(got))
		for _, raw := range jsonListData(got, "periods") {
			assert.Zero(t, raw.(map[string]any)["scheduled_seconds"])
			assert.Nil(t, raw.(map[string]any)["oee_pct"])
		}
	})
}

// A version published now is the baseline for the weeks that start after it, so week 1 of its horizon is measured against it.
func TestDashAnalytics_ScheduleAttainmentFiltersAndTenantIsolation(t *testing.T) {
	t.Parallel()
	lockPublishing(t)
	schedule := ownedSchedule(t, uniqueName("e2e-dashan-attain"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 1, "quantity": 700})
	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	horizon := mustTime(t, jsonField(schedule, "horizon_starts_at"))
	week := map[string]any{
		"starts_at": rfc3339(horizon.AddDate(0, 0, 7)),
		"ends_at":   rfc3339(horizon.AddDate(0, 0, 14).Add(-time.Second)),
	}
	attainment := func(client *Client, extra map[string]any) map[string]any {
		got := mustPutAnalytics(t, client, scheduleAttainmentPath, nil, dashAnalyticsWith(week, extra))
		assert.Equal(t, "analyze_schedule_attainment_response", jsonField(got, "object"))
		return got
	}
	planned := func(got map[string]any) float64 {
		n, _ := jsonObject(got, "totals")["planned_quantity"].(float64)
		return n
	}
	bucketKeys := func(got map[string]any) []string {
		var keys []string
		for _, b := range jsonListData(got, "buckets") {
			keys = append(keys, jsonField(b.(map[string]any), "key"))
		}
		return keys
	}

	byMachine := attainment(apiClient, map[string]any{"group_by": "machine", "machine_ids": []string{SeedMachineID}})
	assert.Equal(t, "measured", jsonField(byMachine, "baseline_status"))
	var baselines []string
	for _, b := range jsonListData(byMachine, "baseline_schedules") {
		baselines = append(baselines, jsonField(b.(map[string]any), "id"))
	}
	assert.Contains(t, baselines, scheduleID, "the version just published plans the week")
	assert.Equal(t, []string{SeedMachineID}, bucketKeys(byMachine))
	assert.GreaterOrEqual(t, planned(byMachine), float64(700), "the added line is planned on the machine")

	byDepartment := attainment(apiClient, map[string]any{"group_by": "department", "department_ids": []string{SeedDepartmentID}})
	assert.Equal(t, []string{SeedDepartmentID}, bucketKeys(byDepartment), "the seed machine sits in the seed department")
	assert.Equal(t, planned(byMachine), planned(byDepartment))

	emptyDepartment := jsonField(createAndCleanup(t, departmentsPath, map[string]any{"name": uniqueName("e2e-dashan-dept")}), "id")
	for name, extra := range map[string]map[string]any{
		"unknown machine":         {"group_by": "machine", "machine_ids": []string{"mc_dashanalyticsnone"}},
		"department with nothing": {"group_by": "department", "department_ids": []string{emptyDepartment}},
	} {
		got := attainment(apiClient, extra)
		assert.Zero(t, planned(got), name)
		assert.Empty(t, bucketKeys(got), name)
		assertNilField(t, jsonObject(got, "totals"), "attainment_pct")
	}

	tenantB := getTenantBClient()
	for _, extra := range []map[string]any{{"group_by": "machine"}, {"group_by": "machine", "machine_ids": []string{SeedMachineID}}} {
		got := attainment(tenantB, extra)
		assert.Equal(t, "no_baseline", jsonField(got, "baseline_status"))
		assert.Empty(t, jsonListData(got, "baseline_schedules"))
		assert.Empty(t, bucketKeys(got))
		assert.Zero(t, planned(got))
	}
}

// --- Pricing and delivery filters ---

func dashAnalyticsAccountPrice(t *testing.T, customerID, productLineID, value string) string {
	t.Helper()
	created := createAndCleanup(t, accountPricesPath, map[string]any{
		"recipient_account_id": customerID,
		"product_line_id":      productLineID,
		"rate":                 map[string]any{"value": value, "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
	})
	return jsonField(created, "id")
}

// Two customers price a product line of the test's own, one far below the other, so the low one is an outlier against the group's median.
func TestDashAnalytics_CustomerPricingFiltersByCustomerAndGroup(t *testing.T) {
	t.Parallel()
	line := parityProductLine(t)
	group := newAccountGroup(t)
	peer := parityCustomer(t, "", line)
	low := parityCustomer(t, group, line)
	dashAnalyticsAccountPrice(t, peer, line, "500.00")
	lowPrice := dashAnalyticsAccountPrice(t, low, line, "0.01")

	findings := func(client *Client, extra map[string]any) []map[string]any {
		body := dashAnalyticsWith(map[string]any{"outlier_tolerance": "0.15", "target_gross_margin": "0"}, extra)
		return listRows(t, jsonObject(mustPutAnalytics(t, client, customerPricingPath, url.Values{"include": {"customer"}}, body), "findings"))
	}

	byCustomer := findings(apiClient, map[string]any{"customer_ids": []string{low}})
	require.Len(t, byCustomer, 1, "%v", byCustomer)
	f := byCustomer[0]
	assert.Equal(t, lowPrice, jsonField(f, "account_price_id"))
	assert.Contains(t, []string{"below_peer_median", "below_peer_median_and_target_margin"}, jsonField(f, "reason"))
	assert.Equal(t, "direct", jsonField(f, "origin"))
	assert.Equal(t, low, entityID(f, "customer"))
	assert.True(t, decimal.RequireFromString("250.005").Equal(decimal.RequireFromString(jsonField(jsonObject(f, "peer_median_price"), "value"))),
		"the median of the group's two prices: %v", f["peer_median_price"])

	byGroup := findings(apiClient, map[string]any{"customer_group_ids": []string{group}})
	require.Len(t, byGroup, 1)
	assert.Equal(t, lowPrice, jsonField(byGroup[0], "account_price_id"))

	assert.Empty(t, findings(apiClient, map[string]any{"customer_ids": []string{peer}}), "the dear price is not below its peer")
	assert.Empty(t, findings(apiClient, map[string]any{"customer_group_ids": []string{newAccountGroup(t)}}))

	assert.Empty(t, findings(getTenantBClient(), map[string]any{"customer_ids": []string{low, peer}}))
}

func TestDashAnalytics_DeliveryPerformanceGroupAndProductLineFilters(t *testing.T) {
	t.Parallel()
	group := newAccountGroup(t)
	customer := leadTimeCustomer(t, "e2e-dashan-delivery", ptrInt(30), group)
	order := issueOrderForCustomer(t, customer, nil)
	require.NotEmpty(t, shipByDate(t, order), "the order must carry a commitment to be measured")
	mine := []string{customer}

	for name, tc := range map[string]struct {
		extra     map[string]any
		committed float64
	}{
		"its group":            {map[string]any{"customer_group_ids": []string{group}}, 1},
		"another group":        {map[string]any{"customer_ids": mine, "customer_group_ids": []string{newAccountGroup(t)}}, 0},
		"its product line":     {map[string]any{"customer_ids": mine, "product_line_ids": []string{SeedProductLineID}}, 1},
		"another product line": {map[string]any{"customer_ids": mine, "product_line_ids": []string{dashAnalyticsOtherProductLineID}}, 0},
	} {
		got := analyzeDelivery(t, dashAnalyticsWith(deliveryWindow(), tc.extra))
		committed, _ := jsonObject(got, "overall")["committed_order_count"].(float64)
		assert.Equal(t, tc.committed, committed, name)
	}
}
