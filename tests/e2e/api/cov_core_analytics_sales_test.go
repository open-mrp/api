//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sales analytics read sales_line_fact, which core-service fills from invoices after they are written.
// Each test ships its own order to a customer created for it and filters to that customer (and the
// seed product line, which keeps any freight line out), so totals are exact and parallel tests never
// see each other's sales.

const (
	salesSummaryPath   = "/v1/core/analytics/sales-summary"
	salesBreakdownPath = "/v1/core/analytics/sales-breakdown"
	salesInvoicesPath  = "/v1/core/analytics/sales-invoices"
	salesLinesPath     = "/v1/core/analytics/sales-lines"

	dollarUnitID = "dollar"
)

// shippedSale is one invoiced order to a fresh customer: 5 pr at $4.25/pr, and 2 dz priced at $3.00/pr.
// A dozen is 12 each, which is 6 pr, so the second line invoices 12 pr for $36.00.
type shippedSale struct {
	customerID string
	shipmentID string
	invoicedAt time.Time
}

const (
	shippedSaleRevenue  = "57.25"
	shippedSaleQuantity = "17"
)

func shipSaleToNewCustomer(t *testing.T) shippedSale {
	t.Helper()

	return shipOrder(t, setupOrderCustomer(t), []map[string]any{
		{
			"product_id": SeedProductID,
			"quantity":   map[string]any{"value": "5", "unit_id": SeedUnitID},
			"unit_price": map[string]any{"value": "4.25", "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
		},
		{
			"product_id": SeedProductID,
			"quantity":   map[string]any{"value": "2", "unit_id": seedDozenUnitID},
			"unit_price": map[string]any{"value": "3.00", "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
		},
	})
}

// shipOrder issues an order of the given lines to the customer, then picks, packs and ships it, which invoices it.
func shipOrder(t *testing.T, customerID string, lines []map[string]any) shippedSale {
	t.Helper()

	order := issueOrderForCustomer(t, customerID, map[string]any{"lines": lines})
	orderID := jsonField(order, "id")

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"related.pick"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	pickID := jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	require.NotEmpty(t, pickID, "issuing must create a pick: %s", string(body))

	pickAllLines(t, pickID)
	packPick(t, pickID)
	numbers := pickShipmentNumbers(t, pickID)
	require.NotEmpty(t, numbers, "packing must produce a shipment")
	listStatus, listBody, err := apiClient.GetListRaw(shipmentsPath, url.Values{"q": {numbers[0]}})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	shipments := jsonArray(parseJSON(listBody), "data")
	require.NotEmpty(t, shipments)
	shipmentID := jsonField(shipments[0].(map[string]any), "id")

	shipStatus, shipBody, err := apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/ship", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, shipStatus, shipBody)
	t.Cleanup(func() {
		_, _, _ = apiClient.Post(shipmentsPath+"/"+shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	})

	return shippedSale{customerID: customerID, shipmentID: shipmentID, invoicedAt: time.Now().UTC()}
}

// saleFilter scopes a report to one customer's socks over a window around now.
func saleFilter(customerID string) map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"starts_at":        rfc3339(now.Add(-24 * time.Hour)),
		"ends_at":          rfc3339(now.Add(24 * time.Hour)),
		"customer_ids":     []string{customerID},
		"product_line_ids": []string{SeedProductLineID},
	}
}

func putSales(t *testing.T, path string, params url.Values, body map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	status, respBody, err := apiClient.PutRaw(path, params, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", path, string(respBody))
	if status != 200 {
		return status, nil, respBody
	}
	return status, parseJSON(respBody), respBody
}

// awaitSalesSummary polls until the customer's summary counts want invoices: facts are written a few seconds after the invoice.
func awaitSalesSummary(t *testing.T, customerID string, want int) map[string]any {
	t.Helper()
	var last []byte
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		status, summary, body := putSales(t, salesSummaryPath, nil, saleFilter(customerID))
		last = body
		// 503 means the facts are still being backfilled on a fresh stack.
		if status == 200 && jsonField(jsonObject(summary, "overall"), "invoice_count") == itoa(want) {
			return summary
		}
		require.Contains(t, []int{200, 503}, status, "unexpected status: %s", string(body))
		time.Sleep(time.Second)
	}
	t.Fatalf("sales summary never reached %d invoice(s); last response: %s", want, string(last))
	return nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// cursorFromURL pulls the cursor out of a page_info URL; these endpoints take their filters in the body, so a page is fetched by resending the body with that cursor.
func cursorFromURL(t *testing.T, pageURL string) string {
	t.Helper()
	require.NotEmpty(t, pageURL, "page_info must link the adjacent page")
	u, err := url.Parse(pageURL)
	require.NoError(t, err)
	cursor := u.Query().Get("cursor")
	require.NotEmpty(t, cursor, "page URL carries no cursor: %s", pageURL)
	return cursor
}

// awaitSalesReports waits out a fresh stack's backfill, during which sales reports answer 503.
func awaitSalesReports(t *testing.T) {
	t.Helper()
	now := time.Now().UTC()
	eventually(t, 45*time.Second, time.Second, func() error {
		status, _, body := putSales(t, salesSummaryPath, nil, map[string]any{"starts_at": rfc3339(now.Add(-time.Hour)), "ends_at": rfc3339(now)})
		if status != 200 {
			return fmt.Errorf("sales summary answered %d: %s", status, string(body))
		}
		return nil
	})
}

func computedValue(t *testing.T, obj map[string]any, field string) string {
	t.Helper()
	q := jsonObject(obj, field)
	require.NotNil(t, q, "%s must be present: %v", field, obj)
	assert.Equal(t, "computed_quantity", jsonField(q, "object"))
	return jsonField(q, "value")
}

// --- Summary ---

func TestSalesAnalytics_ShippedSaleIsReportedExactly(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)

	summary := awaitSalesSummary(t, sale.customerID, 1)
	assert.Equal(t, "analyze_sales_summary_response", jsonField(summary, "object"))

	overall := jsonObject(summary, "overall")
	assert.Equal(t, "sales_totals", jsonField(overall, "object"))
	assertNilField(t, overall, "period_start")
	assert.Equal(t, shippedSaleRevenue, computedValue(t, overall, "revenue"), "5 pr x $4.25 + 2 dz (12 pr) x $3.00")
	assert.Equal(t, shippedSaleQuantity, computedValue(t, overall, "quantity_invoiced"), "quantities are summed in the base unit, pairs")
	assert.NotEmpty(t, computedValue(t, overall, "cost"), "an admin sees cost")
	assert.Equal(t, "1", jsonField(overall, "invoice_count"))
	assert.Equal(t, "2", jsonField(overall, "line_count"))
	assert.Equal(t, "57.25", jsonField(jsonObject(overall, "revenue"), "display_value"))

	days := jsonListData(summary, "periods")
	require.Len(t, days, 1, "one day had sales")
	day := days[0].(map[string]any)
	assert.Equal(t, shippedSaleRevenue, computedValue(t, day, "revenue"))
	assertValidTimestamp(t, jsonField(day, "period_start"), "period_start")

	assertNilField(t, summary, "comparison")
	assertNilField(t, summary, "comparison_periods")
}

func TestSalesAnalytics_ComparisonPeriodIsReportedSeparately(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	body := saleFilter(sale.customerID)
	body["comparison_starts_at"] = rfc3339(time.Now().UTC().AddDate(-1, 0, -1))
	body["comparison_ends_at"] = rfc3339(time.Now().UTC().AddDate(-1, 0, 1))
	status, summary, raw := putSales(t, salesSummaryPath, nil, body)
	requireStatus(t, 200, status, raw)

	comparison := jsonObject(summary, "comparison")
	require.NotNil(t, comparison, "a comparison period was asked for")
	assert.Equal(t, "0", computedValue(t, comparison, "revenue"), "the customer bought nothing a year ago")
	assert.Equal(t, "0", jsonField(comparison, "invoice_count"))
	assert.Empty(t, jsonListData(summary, "comparison_periods"))
	assert.Equal(t, shippedSaleRevenue, computedValue(t, jsonObject(summary, "overall"), "revenue"))
}

func TestSalesAnalytics_LocalDayBucketsFollowTheCallersOffset(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	for _, offset := range []int{-600, 0, 600} {
		body := saleFilter(sale.customerID)
		body["tz_offset_minutes"] = offset
		status, summary, raw := putSales(t, salesSummaryPath, nil, body)
		requireStatus(t, 200, status, raw)
		days := jsonListData(summary, "periods")
		require.Len(t, days, 1)
		got, err := time.Parse(time.RFC3339, jsonField(days[0].(map[string]any), "period_start"))
		require.NoError(t, err)
		local := sale.invoicedAt.Add(time.Duration(offset) * time.Minute)
		// The shipment was invoiced moments before sale.invoicedAt was read; allow the day to straddle midnight.
		wantDay := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
		assert.True(t, got.Equal(wantDay) || got.Equal(wantDay.AddDate(0, 0, -1)),
			"offset %d: bucket %s, want the caller's local day %s", offset, got, wantDay)
	}
}

// --- Breakdown ---

func TestSalesAnalytics_BreakdownByProductCarriesItemDetail(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	body := saleFilter(sale.customerID)
	body["group_by"] = "product"
	status, list, raw := putSales(t, salesBreakdownPath, nil, body)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "list", jsonField(list, "object"))

	groups := jsonArray(list, "data")
	require.Len(t, groups, 1, "both lines are the same item")
	g := groups[0].(map[string]any)
	assert.Equal(t, "sales_breakdown", jsonField(g, "object"))
	assert.Equal(t, SeedItemID, jsonField(g, "key"))
	assert.Equal(t, "SCK-001", jsonField(g, "label"))
	assert.NotEmpty(t, jsonField(g, "description"))
	assert.Equal(t, "pr", jsonField(g, "unit_abbreviation"))
	totals := jsonObject(g, "totals")
	assert.Equal(t, shippedSaleRevenue, computedValue(t, totals, "revenue"))
	assert.Equal(t, shippedSaleQuantity, computedValue(t, totals, "quantity_invoiced"))
	assert.Equal(t, "17 pr", jsonField(jsonObject(totals, "quantity_invoiced"), "display_value"))
	assertNilField(t, g, "comparison_totals")
}

func TestSalesAnalytics_BreakdownByEveryDimensionResponds(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	keys := map[string]string{"customer": sale.customerID, "product_line": SeedProductLineID, "product": SeedItemID}
	for _, groupBy := range []string{"customer", "product", "product_line", "customer_group", "sales_rep", "discount"} {
		body := saleFilter(sale.customerID)
		body["group_by"] = groupBy
		body["comparison_starts_at"] = rfc3339(time.Now().UTC().AddDate(-1, 0, 0))
		body["comparison_ends_at"] = rfc3339(time.Now().UTC().AddDate(-1, 0, 1))
		status, list, raw := putSales(t, salesBreakdownPath, nil, body)
		requireStatus(t, 200, status, raw)
		groups := jsonArray(list, "data")
		if want, ok := keys[groupBy]; ok {
			require.Len(t, groups, 1, "%s: %s", groupBy, string(raw))
			g := groups[0].(map[string]any)
			assert.Equal(t, want, jsonField(g, "key"), groupBy)
			assert.Equal(t, shippedSaleRevenue, computedValue(t, jsonObject(g, "totals"), "revenue"), groupBy)
			assert.Equal(t, "0", computedValue(t, jsonObject(g, "comparison_totals"), "revenue"), groupBy)
		}
	}
}

func TestSalesAnalytics_BreakdownPagesByCursor(t *testing.T) {
	t.Parallel()
	// Three customers of its own, so parallel tests' sales cannot reorder the groups mid-walk. Their
	// sales are equal, so the walk also crosses the keyset's revenue ties.
	var customers []string
	for range 3 {
		sale := shipSaleToNewCustomer(t)
		awaitSalesSummary(t, sale.customerID, 1)
		customers = append(customers, sale.customerID)
	}
	body := saleFilter(customers[0])
	body["customer_ids"], body["group_by"] = customers, "customer"

	type row struct {
		key     string
		revenue float64
	}
	page := func(params url.Values) ([]row, map[string]any) {
		status, list, raw := putSales(t, salesBreakdownPath, params, body)
		requireStatus(t, 200, status, raw)
		var out []row
		for _, g := range jsonArray(list, "data") {
			m := g.(map[string]any)
			rev, err := strconv.ParseFloat(computedValue(t, jsonObject(m, "totals"), "revenue"), 64)
			require.NoError(t, err)
			out = append(out, row{jsonField(m, "key"), rev})
		}
		return out, jsonObject(list, "page_info")
	}

	var forward []row
	params := url.Values{"limit": {"1"}}
	var info map[string]any
	for i := 0; i < 50; i++ {
		var rows []row
		rows, info = page(params)
		forward = append(forward, rows...)
		if jsonField(info, "has_next_page") != "true" {
			break
		}
		params = url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "next_page_url"))}}
	}
	require.Len(t, forward, len(customers), "one page per customer")
	seen := map[string]bool{}
	for i, r := range forward {
		assert.False(t, seen[r.key], "customer %s listed twice", r.key)
		seen[r.key] = true
		if i > 0 {
			assert.LessOrEqual(t, r.revenue, forward[i-1].revenue, "groups are ranked by revenue")
		}
	}

	// Walk back from the last page to the first: the same groups, in reverse.
	var backward []row
	for i := 0; i < 50 && jsonField(info, "has_prev_page") == "true"; i++ {
		var rows []row
		rows, info = page(url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "previous_page_url"))}})
		backward = append(rows, backward...)
	}
	require.Equal(t, forward[:len(forward)-1], backward, "paging back retraces the pages before the last")
}

// --- Invoices ---

func TestSalesAnalytics_InvoicesListTheShippedInvoice(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	status, list, raw := putSales(t, salesInvoicesPath, nil, saleFilter(sale.customerID))
	requireStatus(t, 200, status, raw)
	invoices := jsonArray(list, "data")
	require.Len(t, invoices, 1)
	inv := invoices[0].(map[string]any)
	assert.Equal(t, "sales_invoice", jsonField(inv, "object"))
	assertIDFormat(t, jsonField(inv, "id"), "iv")
	assert.NotEmpty(t, jsonField(inv, "number"))
	assert.Equal(t, sale.customerID, jsonField(inv, "customer_id"))
	assert.NotEmpty(t, jsonField(inv, "customer_name"))
	assertValidTimestamp(t, jsonField(inv, "invoiced_at"), "invoiced_at")
	assert.Equal(t, "1", jsonField(inv, "item_count"))
	assert.Equal(t, shippedSaleRevenue, computedValue(t, inv, "revenue"))
}

// A filtered invoice page is chosen from the filter's lines, a page per customer merged newest first;
// paging forward and back across several customers must list each invoice once, in order, both ways.
func TestSalesAnalytics_InvoicesPageAcrossCustomersForwardAndBack(t *testing.T) {
	t.Parallel()
	var customers []string
	for range 3 {
		sale := shipSaleToNewCustomer(t)
		awaitSalesSummary(t, sale.customerID, 1)
		customers = append(customers, sale.customerID)
	}
	body := saleFilter(customers[0])
	body["customer_ids"] = customers

	type row struct {
		id         string
		invoicedAt string
	}
	page := func(params url.Values) ([]row, map[string]any) {
		status, list, raw := putSales(t, salesInvoicesPath, params, body)
		requireStatus(t, 200, status, raw)
		var out []row
		for _, inv := range jsonArray(list, "data") {
			m := inv.(map[string]any)
			assert.Contains(t, customers, jsonField(m, "customer_id"))
			out = append(out, row{jsonField(m, "id"), jsonField(m, "invoiced_at")})
		}
		return out, jsonObject(list, "page_info")
	}

	var forward []row
	params := url.Values{"limit": {"1"}}
	var info map[string]any
	for i := 0; i < 10; i++ {
		var rows []row
		rows, info = page(params)
		forward = append(forward, rows...)
		if jsonField(info, "has_next_page") != "true" {
			break
		}
		params = url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "next_page_url"))}}
	}
	require.Len(t, forward, len(customers), "one invoice per customer")
	seen := map[string]bool{}
	for i, r := range forward {
		assert.False(t, seen[r.id], "invoice %s listed twice", r.id)
		seen[r.id] = true
		if i > 0 {
			assert.LessOrEqual(t, r.invoicedAt, forward[i-1].invoicedAt, "invoices are listed newest first")
		}
	}

	var backward []row
	for i := 0; i < 10 && jsonField(info, "has_prev_page") == "true"; i++ {
		var rows []row
		rows, info = page(url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "previous_page_url"))}})
		backward = append(rows, backward...)
	}
	require.Equal(t, forward[:len(forward)-1], backward, "paging back retraces the pages before the last")
}

// The line-level sales and open-order analytics resolve a customer filter to that customer's buyers:
// each lists exactly the customer's lines, and a customer group it is not in excludes them.
func TestSalesAnalytics_LegacyEntriesFilterByCustomer(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)
	openCustomer := setupOrderCustomer(t)
	issueOrderForCustomer(t, openCustomer, nil)

	entries := func(path string, body map[string]any) []any {
		t.Helper()
		status, list, raw := putSales(t, path, nil, body)
		requireStatus(t, 200, status, raw)
		return jsonArray(list, "data")
	}
	now := time.Now().UTC()
	window := func(extra map[string]any) map[string]any {
		body := map[string]any{"starts_at": rfc3339(now.Add(-24 * time.Hour)), "ends_at": rfc3339(now.Add(24 * time.Hour))}
		for k, v := range extra {
			body[k] = v
		}
		return body
	}

	lines := entries("/v1/core/analytics/sales", window(map[string]any{"customer_ids": []string{sale.customerID}}))
	require.NotEmpty(t, lines, "the customer's invoiced lines, its shipping line among them")
	for _, l := range lines {
		assert.Equal(t, sale.customerID, jsonField(l.(map[string]any), "customer_id"))
	}
	assert.Empty(t, entries("/v1/core/analytics/sales", window(map[string]any{
		"customer_ids": []string{sale.customerID}, "customer_group_ids": []string{"ag_definitely_not_a_real_group"}})),
		"a customer filter and a group the customer is not in admit no buyer")

	orders := entries("/v1/core/analytics/orders", map[string]any{"customer_ids": []string{openCustomer}})
	require.NotEmpty(t, orders, "the customer's open order is listed")
	for _, o := range orders {
		assert.Equal(t, openCustomer, jsonField(o.(map[string]any), "customer_id"))
	}
}

// --- Lines ---

func TestSalesAnalytics_LinesArePricedInTheBaseUnit(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	status, list, raw := putSales(t, salesLinesPath, nil, saleFilter(sale.customerID))
	requireStatus(t, 200, status, raw)
	lines := jsonArray(list, "data")
	require.Len(t, lines, 2)

	byQuantity := map[string]map[string]any{}
	for _, l := range lines {
		line := l.(map[string]any)
		byQuantity[jsonField(line, "quantity_invoiced")] = line
		assert.Equal(t, sale.customerID, jsonField(line, "customer_id"))
		assert.Equal(t, SeedItemID, jsonField(line, "item_id"))
		assert.Equal(t, "pr", jsonField(line, "unit"))
	}
	dozens := byQuantity["12"]
	require.NotNil(t, dozens, "2 dz must be reported as 12 pr: %s", string(raw))
	assert.Equal(t, "36", jsonField(dozens, "total_invoiced"))
	assert.Equal(t, "3", jsonField(dozens, "unit_price"), "the unit price is per base unit")
	pairs := byQuantity["5"]
	require.NotNil(t, pairs)
	assert.Equal(t, "21.25", jsonField(pairs, "total_invoiced"))
}

func TestSalesAnalytics_LinesPageForwardAndBack(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	status, first, raw := putSales(t, salesLinesPath, url.Values{"limit": {"1"}}, saleFilter(sale.customerID))
	requireStatus(t, 200, status, raw)
	firstInfo := jsonObject(first, "page_info")
	assert.Equal(t, "true", jsonField(firstInfo, "has_next_page"))
	assert.Equal(t, "false", jsonField(firstInfo, "has_prev_page"))
	firstID := jsonField(jsonArray(first, "data")[0].(map[string]any), "id")

	next := cursorFromURL(t, jsonField(firstInfo, "next_page_url"))
	status, second, raw := putSales(t, salesLinesPath, url.Values{"limit": {"1"}, "cursor": {next}}, saleFilter(sale.customerID))
	requireStatus(t, 200, status, raw)
	secondInfo := jsonObject(second, "page_info")
	assert.Equal(t, "false", jsonField(secondInfo, "has_next_page"))
	assert.Equal(t, "true", jsonField(secondInfo, "has_prev_page"))
	secondID := jsonField(jsonArray(second, "data")[0].(map[string]any), "id")
	assert.NotEqual(t, firstID, secondID)

	prev := cursorFromURL(t, jsonField(secondInfo, "previous_page_url"))
	status, back, raw := putSales(t, salesLinesPath, url.Values{"limit": {"1"}, "cursor": {prev}}, saleFilter(sale.customerID))
	requireStatus(t, 200, status, raw)
	assert.Equal(t, firstID, jsonField(jsonArray(back, "data")[0].(map[string]any), "id"), "going back returns the first page")
}

// --- Lifecycle ---

func TestSalesAnalytics_VoidingTheShipmentRemovesTheSale(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	status, body, err := apiClient.Post(shipmentsPath+"/"+sale.shipmentID+"/actions/void", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	summary := awaitSalesSummary(t, sale.customerID, 0)
	assert.Equal(t, "0", computedValue(t, jsonObject(summary, "overall"), "revenue"), "a voided shipment's invoice is no longer a sale")
}

// --- Validation ---

func TestSalesAnalytics_RejectsInvalidRequests(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	window := func(extra map[string]any) map[string]any {
		body := map[string]any{"starts_at": rfc3339(now.AddDate(0, -1, 0)), "ends_at": rfc3339(now)}
		for k, v := range extra {
			body[k] = v
		}
		return body
	}
	cases := []struct {
		name   string
		path   string
		params url.Values
		body   map[string]any
	}{
		{"summary without dates", salesSummaryPath, nil, map[string]any{}},
		{"summary with half a comparison", salesSummaryPath, nil, window(map[string]any{"comparison_starts_at": rfc3339(now.AddDate(-1, 0, 0))})},
		{"summary with an impossible offset", salesSummaryPath, nil, window(map[string]any{"tz_offset_minutes": 5000})},
		{"breakdown without group_by", salesBreakdownPath, nil, window(nil)},
		{"breakdown by an unknown dimension", salesBreakdownPath, nil, window(map[string]any{"group_by": "warehouse"})},
		{"breakdown with a forged cursor", salesBreakdownPath, url.Values{"cursor": {"not-a-cursor"}}, window(map[string]any{"group_by": "customer"})},
		{"breakdown over the page limit", salesBreakdownPath, url.Values{"limit": {"1000"}}, window(map[string]any{"group_by": "customer"})},
		{"lines with only a start", salesLinesPath, nil, map[string]any{"starts_at": rfc3339(now)}},
		{"lines with a forged cursor", salesLinesPath, url.Values{"cursor": {"not-a-cursor"}}, window(nil)},
		{"invoices without dates", salesInvoicesPath, nil, map[string]any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, raw := putSales(t, tc.path, tc.params, tc.body)
			assert.Contains(t, []int{400, 422}, status, "%s: %s", tc.name, string(raw))
		})
	}
}

func TestSalesAnalytics_UnknownCustomerYieldsNoSales(t *testing.T) {
	t.Parallel()
	awaitSalesReports(t)
	status, summary, raw := putSales(t, salesSummaryPath, nil, saleFilter("ac_doesnotexist0"))
	requireStatus(t, 200, status, raw)
	assert.Equal(t, "0", jsonField(jsonObject(summary, "overall"), "invoice_count"))
	assert.Equal(t, "0", computedValue(t, jsonObject(summary, "overall"), "revenue"))
}

// --- Export ---

const salesLinesExportPath = salesLinesPath + "/actions/export"

func TestSalesAnalytics_ExportRendersThroughAJob(t *testing.T) {
	t.Parallel()
	sale := shipSaleToNewCustomer(t)
	awaitSalesSummary(t, sale.customerID, 1)

	job := completedExportJob(t, salesLinesExportPath, saleFilter(sale.customerID))
	assert.Equal(t, "invoice_line", jsonField(job, "resource_type"), "the job says what the export lists")
	export := jsonObject(job, "export")
	require.NotNil(t, export, "a completed export links its file: %v", job)
	assert.Contains(t, jsonField(export, "url"), "sales_data_export_", "the file is named for the sales data export")
}

func TestSalesAnalytics_ExportOfEveryLineIsAccepted(t *testing.T) {
	t.Parallel()
	awaitSalesReports(t)
	// No window: every invoiced line in the account, as the order data page exports with no date range.
	completedExportJob(t, salesLinesExportPath, map[string]any{"product_line_ids": []string{SeedProductLineID}})
}

func TestSalesAnalytics_ExportRejectsHalfAWindow(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(salesLinesExportPath, map[string]any{"starts_at": rfc3339(time.Now().UTC())}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, string(body))
	assert.Contains(t, []int{400, 422}, status, string(body))
}
