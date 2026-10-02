//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The sales order, invoice, and sales line lists choose a page from their own table and hydrate it
// after, with filters on the order or its lines read as semijoins. These walk a filtered list a row at
// a time in both directions, which is where a page chosen apart from its rows would go wrong.

// oneSockLine is a single order line of the seed product.
func oneSockLine() []map[string]any {
	return []map[string]any{{
		"product_id": SeedProductID,
		"quantity":   map[string]any{"value": "1", "unit_id": SeedUnitID},
		"unit_price": map[string]any{"value": "2.00", "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
	}}
}

// pageBothWays lists path one row at a time with params: the first page, the next, and the previous
// one again. It returns the ids of the first and second pages.
func pageBothWays(t *testing.T, path string, params url.Values) (first, second string) {
	t.Helper()
	params.Set("limit", "1")
	rows, info := listPayments(t, path, params)
	require.Len(t, rows, 1)
	require.Equal(t, true, info["has_next_page"], "page_info: %v", info)
	first = rowIDs(rows)[0]

	next := jsonField(info, "next_page_url")
	status, body, err := apiClient.GetListRawFromPageURL(&next)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	page := parseJSON(body)
	data := jsonArray(page, "data")
	require.Len(t, data, 1)
	second = jsonField(data[0].(map[string]any), "id")
	require.NotEqual(t, first, second)

	info = jsonObject(page, "page_info")
	require.Equal(t, true, info["has_prev_page"], "page_info: %v", info)
	prev := jsonField(info, "previous_page_url")
	status, body, err = apiClient.GetListRawFromPageURL(&prev)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data = jsonArray(parseJSON(body), "data")
	require.Len(t, data, 1)
	assert.Equal(t, first, jsonField(data[0].(map[string]any), "id"), "the previous page is the first page again")
	return first, second
}

func TestListSalesOrders_ProductLineFilterPagesBothWays(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	older := jsonField(issueOrderForCustomer(t, customerID, map[string]any{"lines": oneSockLine()}), "id")
	newer := jsonField(issueOrderForCustomer(t, customerID, map[string]any{"lines": oneSockLine()}), "id")

	first, second := pageBothWays(t, salesOrdersPath, url.Values{
		"customer_ids":     {customerID},
		"product_line_ids": {SeedProductLineID},
	})
	assert.Equal(t, []string{newer, older}, []string{first, second}, "newest first")
}

func TestListInvoices_CustomerFilterPagesBothWays(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	shipOrder(t, customerID, oneSockLine())
	shipOrder(t, customerID, oneSockLine())

	all := listIDs(t, financeInvoicesPath, url.Values{"customer_ids": {customerID}})
	require.Len(t, all, 2, "one invoice per shipped order")
	first, second := pageBothWays(t, financeInvoicesPath, url.Values{"customer_ids": {customerID}, "item_ids": {SeedItemID}})
	assert.Equal(t, all, []string{first, second}, "paging returns the list's order")
}

func TestCustomerInvoices_PagesBothWays(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)
	shipOrder(t, customerID, oneSockLine())
	shipOrder(t, customerID, oneSockLine())

	path := "/v1/finance/accounts/" + customerID + "/invoices"
	all := listIDs(t, path, nil)
	require.Len(t, all, 2, "both invoices are unpaid")
	first, second := pageBothWays(t, path, url.Values{})
	assert.Equal(t, all, []string{first, second}, "paging returns the list's order")
}

// Two customers' lines are read buyer by buyer and merged; paging must still walk them in one order.
func TestSalesAnalytics_LinesOfSeveralCustomersPageBothWays(t *testing.T) {
	t.Parallel()
	a, b := shipSaleToNewCustomer(t), shipSaleToNewCustomer(t)
	awaitSalesSummary(t, a.customerID, 1)
	awaitSalesSummary(t, b.customerID, 1)

	filter := saleFilter(a.customerID)
	filter["customer_ids"] = []string{a.customerID, b.customerID}
	status, all, raw := putSales(t, salesLinesPath, url.Values{"limit": {"10"}}, filter)
	requireStatus(t, 200, status, raw)
	var want []string
	for _, row := range jsonArray(all, "data") {
		want = append(want, jsonField(row.(map[string]any), "id"))
	}
	require.Len(t, want, 4, "two lines per sale")

	status, first, raw := putSales(t, salesLinesPath, url.Values{"limit": {"3"}}, filter)
	requireStatus(t, 200, status, raw)
	next := cursorFromURL(t, jsonField(jsonObject(first, "page_info"), "next_page_url"))
	status, second, raw := putSales(t, salesLinesPath, url.Values{"limit": {"3"}, "cursor": {next}}, filter)
	requireStatus(t, 200, status, raw)
	var got []string
	for _, page := range []map[string]any{first, second} {
		for _, row := range jsonArray(page, "data") {
			got = append(got, jsonField(row.(map[string]any), "id"))
		}
	}
	assert.Equal(t, want, got, "two pages hold the list in its order")

	prev := cursorFromURL(t, jsonField(jsonObject(second, "page_info"), "previous_page_url"))
	status, back, raw := putSales(t, salesLinesPath, url.Values{"limit": {"3"}, "cursor": {prev}}, filter)
	requireStatus(t, 200, status, raw)
	var backIDs []string
	for _, row := range jsonArray(back, "data") {
		backIDs = append(backIDs, jsonField(row.(map[string]any), "id"))
	}
	assert.Equal(t, want[:3], backIDs, "going back returns the first page")
}
