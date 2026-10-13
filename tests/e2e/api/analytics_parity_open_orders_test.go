//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The open-orders reports replace the dashboard's products-on-order views. An open order is an issued sales
// order not yet completed; only its sale lines count. Quantities are in the item's base unit (pairs for the
// seed socks); money is each line's quantity in the dimension's base unit (each) times its price per each.

// openOrderBook is one customer's order book:
//
//	A  issued      5 pr SCK-001 @ $4.25/pr   -> 10 ea x $2.125 = $21.25
//	               2 dz SCK-002 @ $3.00/pr   -> 24 ea x $1.50  = $36.00
//	B  issued     10 pr SCK-001 @ $2.00/pr   -> 20 ea x $1.00  = $20.00, 4 pr shipped and invoiced ($8.00)
//	done shipped   3 pr SCK-003 @ $1.00/pr   -> completed, so not open
//	est estimate   7 pr SCK-004 @ $1.00/pr   -> never issued, so not open
type openOrderBook struct {
	customerID     string
	customerNumber string
	a, b, done     map[string]any
	estimate       map[string]any
}

func seedOpenOrderBook(t *testing.T) openOrderBook {
	t.Helper()
	customerID := parityCustomer(t, "")
	book := openOrderBook{customerID: customerID}
	book.customerNumber = jsonField(parseJSON(mustGet(t, customersPath+"/"+customerID)), "number")

	book.a = issueParityOrder(t, customerID, "",
		parityLine(SeedProductID, "5", SeedUnitID, "4.25"),
		parityLine(parityProductSCK002, "2", seedDozenUnitID, "3.00"))
	book.b = issueParityOrder(t, customerID, "", parityLine(SeedProductID, "10", SeedUnitID, "2.00"))
	shipPartOfOrder(t, jsonField(book.b, "id"), "4")
	require.Empty(t, jsonField(readOrder(t, jsonField(book.b, "id")), "completed_at"), "a partly shipped order stays open")

	book.done = issueParityOrder(t, customerID, "", parityLine(parityProductSCK003, "3", SeedUnitID, "1.00"))
	shipWholeOrder(t, jsonField(book.done, "id"))
	require.NotEmpty(t, jsonField(readOrder(t, jsonField(book.done, "id")), "completed_at"), "a fully shipped order is completed")

	book.estimate = createParityEstimate(t, customerID, parityLine(parityProductSCK004, "7", SeedUnitID, "1.00"))
	return book
}

func (b openOrderBook) filter(extra map[string]any) map[string]any {
	body := map[string]any{"customer_ids": []string{b.customerID}}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func assertQuantity(t *testing.T, obj map[string]any, field, value, display, unitAbbr string) {
	t.Helper()
	q := jsonObject(obj, field)
	require.NotNil(t, q, "%s: %v", field, obj)
	assert.Equal(t, "computed_quantity", jsonField(q, "object"), field)
	assert.Equal(t, value, jsonField(q, "value"), field)
	assert.Equal(t, display, jsonField(q, "display_value"), field)
	if unitAbbr == "" {
		assertNilField(t, q, "unit")
		return
	}
	unit := jsonObject(q, "unit")
	require.NotNil(t, unit, "%s carries its unit", field)
	assert.Equal(t, unitAbbr, jsonField(unit, "abbreviation"), field)
}

func assertItem(t *testing.T, row map[string]any, itemID, sku string) {
	t.Helper()
	item := jsonObject(row, "item")
	require.NotNil(t, item)
	assert.Equal(t, itemID, jsonField(item, "id"))
	assert.Equal(t, "item", jsonField(item, "object"))
	assert.Equal(t, sku, jsonField(item, "sku"))
	assert.NotEmpty(t, jsonField(item, "description"))
}

// --- Summary ---

func TestAnalyticsParityOpenOrders_SummaryCountsOnlyOpenSaleLines(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)

	summary := mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, book.filter(nil))
	assert.Equal(t, "open_orders_summary", jsonField(summary, "object"))
	// 21.25 + 36 + 20; the shipped order, the estimate and the freight lines are not counted.
	assertQuantity(t, summary, "ordered", "77.25", "77.25", "")
	// A is wholly back-ordered (57.25); B has 6 of its 10 pr left (12).
	assertQuantity(t, summary, "back_ordered", "69.25", "69.25", "")
	assertQuantity(t, summary, "invoiced", "8", "8.00", "")

	// The line filters narrow what is counted.
	summary = mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, book.filter(map[string]any{"item_ids": []string{parityItemSCK002}}))
	assertQuantity(t, summary, "ordered", "36", "36.00", "")
	assertQuantity(t, summary, "invoiced", "0", "0.00", "")
	summary = mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, book.filter(map[string]any{"product_line_ids": []string{SeedProductLineID}}))
	assertQuantity(t, summary, "ordered", "77.25", "77.25", "")
}

// --- Breakdown ---

func TestAnalyticsParityOpenOrders_BreakdownTotalsEachItemInItsBaseUnit(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)

	rows := listRows(t, mustPutAnalytics(t, apiClient, openOrdersBreakdownPath, nil, book.filter(nil)))
	require.Len(t, rows, 2, "only items on open lines: %v", rows)

	// Most back-ordered first: SCK-002 has 12 pr (2 dz) back-ordered, SCK-001 has 5 + 6 = 11.
	sck002, sck001 := rows[0], rows[1]
	assert.Equal(t, "open_order_product", jsonField(sck002, "object"))
	assertItem(t, sck002, parityItemSCK002, "SCK-002")
	assert.Equal(t, "pr", jsonField(jsonObject(sck002, "unit"), "abbreviation"))
	assert.Equal(t, SeedUnitID, jsonField(jsonObject(sck002, "unit"), "id"))
	assertQuantity(t, sck002, "quantity_ordered", "12", "12 pr", "pr")
	assertQuantity(t, sck002, "quantity_back_ordered", "12", "12 pr", "pr")
	assertQuantity(t, sck002, "quantity_invoiced", "0", "0 pr", "pr")

	assertItem(t, sck001, SeedItemID, "SCK-001")
	assertQuantity(t, sck001, "quantity_ordered", "15", "15 pr", "pr")
	assertQuantity(t, sck001, "quantity_back_ordered", "11", "11 pr", "pr")
	assertQuantity(t, sck001, "quantity_invoiced", "4", "4 pr", "pr")

	rows = listRows(t, mustPutAnalytics(t, apiClient, openOrdersBreakdownPath, nil, book.filter(map[string]any{"item_ids": []string{SeedItemID}})))
	require.Len(t, rows, 1)
	assertItem(t, rows[0], SeedItemID, "SCK-001")
	assertQuantity(t, rows[0], "quantity_ordered", "15", "15 pr", "pr")
}

func TestAnalyticsParityOpenOrders_BreakdownPagesByCursor(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)
	body := book.filter(nil)

	first := mustPutAnalytics(t, apiClient, openOrdersBreakdownPath, url.Values{"limit": {"1"}}, body)
	rows := listRows(t, first)
	require.Len(t, rows, 1)
	assertItem(t, rows[0], parityItemSCK002, "SCK-002")
	info := jsonObject(first, "page_info")
	assert.Equal(t, "true", jsonField(info, "has_next_page"))
	assert.Equal(t, "false", jsonField(info, "has_prev_page"))

	second := mustPutAnalytics(t, apiClient, openOrdersBreakdownPath, url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "next_page_url"))}}, body)
	rows = listRows(t, second)
	require.Len(t, rows, 1)
	assertItem(t, rows[0], SeedItemID, "SCK-001")
	info = jsonObject(second, "page_info")
	assert.Equal(t, "false", jsonField(info, "has_next_page"))
	assert.Equal(t, "true", jsonField(info, "has_prev_page"))

	back := mustPutAnalytics(t, apiClient, openOrdersBreakdownPath, url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "previous_page_url"))}}, body)
	rows = listRows(t, back)
	require.Len(t, rows, 1)
	assertItem(t, rows[0], parityItemSCK002, "SCK-002")
	info = jsonObject(back, "page_info")
	assert.Equal(t, "true", jsonField(info, "has_next_page"))
	assert.Equal(t, "false", jsonField(info, "has_prev_page"))
}

// --- List ---

func assertOpenOrderRow(t *testing.T, row map[string]any, book openOrderBook, order map[string]any, lineCount, total, display string) {
	t.Helper()
	assert.Equal(t, "open_order", jsonField(row, "object"))
	o := jsonObject(row, "order")
	require.NotNil(t, o)
	assert.Equal(t, jsonField(order, "id"), jsonField(o, "id"))
	assert.Equal(t, "sales_order", jsonField(o, "object"))
	assert.Equal(t, paddedNumber(jsonField(order, "number")), jsonField(o, "number"), "the number is zero-padded as the dashboard printed it")
	assert.Equal(t, "issued", jsonField(row, "status"))
	assert.Equal(t, jsonField(order, "issued_at"), jsonField(row, "issued_at"))
	customer := jsonObject(row, "customer")
	require.NotNil(t, customer)
	assert.Equal(t, book.customerID, jsonField(customer, "id"))
	assert.Equal(t, "customer", jsonField(customer, "object"))
	assert.Contains(t, jsonField(customer, "name"), "e2e-parity-cust")
	assert.Equal(t, book.customerNumber, jsonField(customer, "number"))
	shipTo := jsonObject(row, "ship_to")
	require.NotNil(t, shipTo)
	assert.Equal(t, "CA", jsonField(shipTo, "state"))
	assert.Equal(t, "US", jsonField(shipTo, "country"))
	assert.Equal(t, lineCount, jsonField(row, "line_count"))
	assertQuantity(t, row, "total_ordered", total, display, "")
}

func TestAnalyticsParityOpenOrders_ListShowsOpenOrdersNewestFirst(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)

	rows := listRows(t, mustPutAnalytics(t, apiClient, openOrdersPath, nil, book.filter(nil)))
	require.Len(t, rows, 2, "the shipped order and the estimate are not open: %v", rows)
	assertOpenOrderRow(t, rows[0], book, book.b, "1", "20", "20.00")
	assertOpenOrderRow(t, rows[1], book, book.a, "2", "57.25", "57.25")

	// A line filter counts only matching lines, and an order with none drops out.
	rows = listRows(t, mustPutAnalytics(t, apiClient, openOrdersPath, nil, book.filter(map[string]any{"item_ids": []string{parityItemSCK002}})))
	require.Len(t, rows, 1)
	assertOpenOrderRow(t, rows[0], book, book.a, "1", "36", "36.00")

	rows = listRows(t, mustPutAnalytics(t, apiClient, openOrdersPath, nil, book.filter(map[string]any{"product_line_ids": []string{SeedProductLineID}})))
	require.Len(t, rows, 2)
}

func TestAnalyticsParityOpenOrders_ListPagesByCursor(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)
	body := book.filter(nil)

	first := mustPutAnalytics(t, apiClient, openOrdersPath, url.Values{"limit": {"1"}}, body)
	rows := listRows(t, first)
	require.Len(t, rows, 1)
	assert.Equal(t, jsonField(book.b, "id"), jsonField(jsonObject(rows[0], "order"), "id"))
	info := jsonObject(first, "page_info")
	assert.Equal(t, "true", jsonField(info, "has_next_page"))
	assert.Equal(t, "false", jsonField(info, "has_prev_page"))

	second := mustPutAnalytics(t, apiClient, openOrdersPath, url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "next_page_url"))}}, body)
	rows = listRows(t, second)
	require.Len(t, rows, 1)
	assert.Equal(t, jsonField(book.a, "id"), jsonField(jsonObject(rows[0], "order"), "id"))
	info = jsonObject(second, "page_info")
	assert.Equal(t, "false", jsonField(info, "has_next_page"))
	assert.Equal(t, "true", jsonField(info, "has_prev_page"))

	back := mustPutAnalytics(t, apiClient, openOrdersPath, url.Values{"limit": {"1"}, "cursor": {cursorFromURL(t, jsonField(info, "previous_page_url"))}}, body)
	rows = listRows(t, back)
	require.Len(t, rows, 1)
	assert.Equal(t, jsonField(book.b, "id"), jsonField(jsonObject(rows[0], "order"), "id"))
}

// --- Lines ---

func openOrderLines(t *testing.T, client *Client, orderID string) (int, []map[string]any, []byte) {
	t.Helper()
	status, body, err := client.GetListRaw(openOrdersPath+"/"+orderID+"/lines", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "lines must not 5xx: %s", string(body))
	if status != 200 {
		return status, nil, body
	}
	return status, listRows(t, parseJSON(body)), body
}

func assertUnitPrice(t *testing.T, row map[string]any, value, display string) {
	t.Helper()
	price := jsonObject(row, "unit_price")
	require.NotNil(t, price)
	assert.Equal(t, "computed_rate", jsonField(price, "object"))
	assert.Equal(t, value, jsonField(price, "value"))
	assert.Equal(t, display, jsonField(price, "display_value"))
	assertNilField(t, price, "numerator_unit")
	assertNilField(t, price, "denominator_unit")
}

func TestAnalyticsParityOpenOrders_LinesPriceEachLinePerBaseUnit(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)

	status, lines, raw := openOrderLines(t, apiClient, jsonField(book.a, "id"))
	requireStatus(t, 200, status, raw)
	require.Len(t, lines, 2, "sale lines only, by SKU: %s", string(raw))

	l := lines[0]
	assert.Equal(t, "open_order_line", jsonField(l, "object"))
	assert.NotEmpty(t, jsonField(l, "id"))
	assertItem(t, l, SeedItemID, "SCK-001")
	assert.Equal(t, "pr", jsonField(jsonObject(l, "unit"), "abbreviation"))
	// $4.25 per pair is $2.125 per each.
	assertUnitPrice(t, l, "2.125", "$2.13 / ea")
	assertQuantity(t, l, "quantity_back_ordered", "5", "5 pr", "pr")
	assertQuantity(t, l, "quantity_invoiced", "0", "0 pr", "pr")
	assertQuantity(t, l, "total_ordered", "21.25", "21.25", "")

	l = lines[1]
	assertItem(t, l, parityItemSCK002, "SCK-002")
	assertUnitPrice(t, l, "1.5", "$1.50 / ea")
	assertQuantity(t, l, "quantity_back_ordered", "12", "12 pr", "pr")
	assertQuantity(t, l, "total_ordered", "36", "36.00", "")

	status, lines, raw = openOrderLines(t, apiClient, jsonField(book.b, "id"))
	requireStatus(t, 200, status, raw)
	require.Len(t, lines, 1)
	assertUnitPrice(t, lines[0], "1", "$1.00 / ea")
	assertQuantity(t, lines[0], "quantity_back_ordered", "6", "6 pr", "pr")
	assertQuantity(t, lines[0], "quantity_invoiced", "4", "4 pr", "pr")
	assertQuantity(t, lines[0], "total_ordered", "20", "20.00", "")

	// The detail reads any of the account's sales orders, open or not, as the dashboard's did.
	status, lines, raw = openOrderLines(t, apiClient, jsonField(book.done, "id"))
	requireStatus(t, 200, status, raw)
	require.Len(t, lines, 1)
	assertQuantity(t, lines[0], "quantity_back_ordered", "0", "0 pr", "pr")
	assertQuantity(t, lines[0], "quantity_invoiced", "3", "3 pr", "pr")
	assertQuantity(t, lines[0], "total_ordered", "3", "3.00", "")
	status, lines, raw = openOrderLines(t, apiClient, jsonField(book.estimate, "id"))
	requireStatus(t, 200, status, raw)
	require.Len(t, lines, 1)
	assertQuantity(t, lines[0], "quantity_back_ordered", "7", "7 pr", "pr")
}

func TestAnalyticsParityOpenOrders_LinesOfAnythingButTheAccountsSalesOrderAre404(t *testing.T) {
	t.Parallel()
	for name, orderID := range map[string]string{
		"unknown order":          "or_doesnotexist000",
		"purchase order":         SeedPurchaseOrderID,
		"another tenant's order": otherTenantSalesOrderID(t),
	} {
		status, _, raw := openOrderLines(t, apiClient, orderID)
		assert.Equal(t, 404, status, "%s: %s", name, string(raw))
		if status == 404 {
			requireErrorResponse(t, raw, "resource_not_found", "invalid_request_error")
		}
	}
}

// --- Customers, groups, and what an order is ---

func TestAnalyticsParityOpenOrders_CustomerFilterIncludesChildAccountsAndGroupsNarrowIt(t *testing.T) {
	t.Parallel()
	group := newAccountGroup(t)
	parent := parityCustomer(t, "")
	child := parityCustomer(t, group)
	makeChildCustomer(t, parent, child)
	issueParityOrder(t, child, "", parityLine(SeedProductID, "3", SeedUnitID, "2.00")) // 6 ea x $1 = $6

	for name, body := range map[string]map[string]any{
		"the parent":           {"customer_ids": []string{parent}},
		"the child":            {"customer_ids": []string{child}},
		"the child's group":    {"customer_group_ids": []string{group}},
		"parent and the group": {"customer_ids": []string{parent}, "customer_group_ids": []string{group}},
	} {
		summary := mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, body)
		assert.Equal(t, "6", computedValue(t, summary, "ordered"), name)
	}

	other := newAccountGroup(t)
	summary := mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, map[string]any{"customer_ids": []string{parent}, "customer_group_ids": []string{other}})
	assert.Equal(t, "0", computedValue(t, summary, "ordered"), "the customer and group filters combine with AND")
	rows := listRows(t, mustPutAnalytics(t, apiClient, openOrdersPath, nil, map[string]any{"customer_group_ids": []string{other}}))
	assert.Empty(t, rows)
}

func TestAnalyticsParityOpenOrders_PurchaseOrdersAreNeverOpenSalesOrders(t *testing.T) {
	t.Parallel()
	po := createPurchaseOrder(t, nil)
	poID := jsonField(po, "id")
	status, body := changePurchaseOrderStatus(t, poID, "issue")
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { changePurchaseOrderStatus(t, poID, "unissue") })

	// A purchase order is bought by the account itself; a fresh customer keeps the filter's cache entry this run's own.
	fresh := parityCustomer(t, "")
	rows := listRows(t, mustPutAnalytics(t, apiClient, openOrdersPath, url.Values{"limit": {"100"}}, map[string]any{"customer_ids": []string{SeedAccountID, fresh}}))
	for _, row := range rows {
		assert.NotEqual(t, poID, jsonField(jsonObject(row, "order"), "id"), "an issued purchase order is not an open sales order")
	}
	linesStatus, _, raw := openOrderLines(t, apiClient, poID)
	assert.Equal(t, 404, linesStatus, string(raw))
}

func TestAnalyticsParityOpenOrders_AnotherTenantsCustomerYieldsNothing(t *testing.T) {
	t.Parallel()
	var foreignBuyer string
	require.NoError(t, authDB(t).QueryRow(`SELECT buyer_account_id FROM sales_order WHERE owner_account_id <> ? AND sales_order_type_code = 'sales_order' LIMIT 1`, SeedAccountID).Scan(&foreignBuyer))
	body := map[string]any{"customer_ids": []string{foreignBuyer, parityCustomer(t, "")}}
	summary := mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, body)
	assert.Equal(t, "0", computedValue(t, summary, "ordered"))
	assert.Empty(t, listRows(t, mustPutAnalytics(t, apiClient, openOrdersPath, nil, body)))
	assert.Empty(t, listRows(t, mustPutAnalytics(t, apiClient, openOrdersBreakdownPath, nil, body)))
}

// --- Sales reps and permissions ---

func TestAnalyticsParityOpenOrders_SalesRepsSeeOnlyTheirOwnOrders(t *testing.T) {
	t.Parallel()
	rep, repID := paritySalesRep(t, "invoices", "sales_orders", "jobs")
	customer := parityCustomer(t, "")
	mine := issueParityOrder(t, customer, repID, parityLine(SeedProductID, "2", SeedUnitID, "2.00"))      // $4
	theirs := issueParityOrder(t, customer, "", parityLine(parityProductSCK002, "3", SeedUnitID, "2.00")) // $6
	body := map[string]any{"customer_ids": []string{customer}}

	admin := mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, body)
	assert.Equal(t, "10", computedValue(t, admin, "ordered"))
	byRep := mustPutAnalytics(t, apiClient, openOrdersSummaryPath, nil, map[string]any{"customer_ids": []string{customer}, "sales_rep_ids": []string{repID}})
	assert.Equal(t, "4", computedValue(t, byRep, "ordered"), "an admin can filter to one rep")

	repSummary := mustPutAnalytics(t, rep, openOrdersSummaryPath, nil, map[string]any{"customer_ids": []string{customer}, "sales_rep_ids": []string{SeedAccountUserID}})
	assert.Equal(t, "4", computedValue(t, repSummary, "ordered"), "a rep sees only their own orders, whatever they ask for")

	rows := listRows(t, mustPutAnalytics(t, rep, openOrdersPath, nil, body))
	require.Len(t, rows, 1)
	assert.Equal(t, jsonField(mine, "id"), jsonField(jsonObject(rows[0], "order"), "id"))
	products := listRows(t, mustPutAnalytics(t, rep, openOrdersBreakdownPath, nil, body))
	require.Len(t, products, 1)
	assertItem(t, products[0], SeedItemID, "SCK-001")

	status, _, raw := openOrderLines(t, rep, jsonField(mine, "id"))
	requireStatus(t, 200, status, raw)
	status, _, raw = openOrderLines(t, rep, jsonField(theirs, "id"))
	assert.Equal(t, 404, status, "another rep's order is not the rep's to read: %s", string(raw))

	job := completedExportJobAs(t, rep, openOrderLinesExport, body)
	assert.Equal(t, "sales_order_line", jsonField(job, "resource_type"))
}

func TestAnalyticsParityOpenOrders_TotalsNeedInvoicesAndTheDetailNeedsSalesOrders(t *testing.T) {
	t.Parallel()
	customer := parityCustomer(t, "")
	order := issueParityOrder(t, customer, "", parityLine(SeedProductID, "1", SeedUnitID, "2.00"))
	body := map[string]any{"customer_ids": []string{customer}}

	invoices := customRoleClient(t, "invoices:read")
	for _, path := range []string{openOrdersSummaryPath, openOrdersBreakdownPath, openOrdersPath} {
		status, _, raw := putAnalytics(t, invoices, path, nil, body)
		assert.Equal(t, 200, status, "%s: %s", path, string(raw))
	}
	status, _, raw := openOrderLines(t, invoices, jsonField(order, "id"))
	assert.Equal(t, 403, status, string(raw))
	exportStatus, exportBody, err := invoices.Post(openOrderLinesExport, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, exportStatus, string(exportBody))

	salesOrders := customRoleClient(t, "sales_orders:read", "jobs:read")
	for _, path := range []string{openOrdersSummaryPath, openOrdersBreakdownPath, openOrdersPath} {
		status, _, raw := putAnalytics(t, salesOrders, path, nil, body)
		assert.Equal(t, 403, status, "%s: %s", path, string(raw))
	}
	status, lines, raw := openOrderLines(t, salesOrders, jsonField(order, "id"))
	requireStatus(t, 200, status, raw)
	assert.Len(t, lines, 1)
	completedExportJobAs(t, salesOrders, openOrderLinesExport, body)
}

// --- Export ---

func TestAnalyticsParityOpenOrders_ExportRendersThroughAJob(t *testing.T) {
	t.Parallel()
	book := seedOpenOrderBook(t)

	job := completedExportJob(t, openOrderLinesExport, book.filter(map[string]any{"item_ids": []string{SeedItemID}}))
	assert.Equal(t, "sales_order_line", jsonField(job, "resource_type"), "the job says what the export lists")
	export := jsonObject(job, "export")
	require.NotNil(t, export, "a completed export links its file: %v", job)
	assert.Contains(t, jsonField(export, "url"), "open_order_lines_export_")
}

// --- Validation ---

func TestAnalyticsParityOpenOrders_RejectsBadRequests(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		path   string
		params url.Values
		body   map[string]any
	}{
		{"breakdown over the page limit", openOrdersBreakdownPath, url.Values{"limit": {"101"}}, nil},
		{"breakdown with a forged cursor", openOrdersBreakdownPath, url.Values{"cursor": {"not-a-cursor"}}, nil},
		{"list with a forged cursor", openOrdersPath, url.Values{"cursor": {"not-a-cursor"}}, nil},
		{"list with a zero limit", openOrdersPath, url.Values{"limit": {"0"}}, nil},
		{"summary with an unknown field", openOrdersSummaryPath, nil, map[string]any{"starts_at": "2026-01-01T00:00:00Z"}},
	}
	for _, tc := range cases {
		status, _, raw := putAnalytics(t, apiClient, tc.path, tc.params, tc.body)
		assert.Contains(t, []int{400, 422}, status, "%s: %s", tc.name, string(raw))
	}
}
