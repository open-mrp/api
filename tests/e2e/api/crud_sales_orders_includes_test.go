//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// SalesOrder — Additional Include Tests
// ──────────────────────────────────────────────
//
// This file covers include fields not already tested in included_fields_test.go
// (which checks carrier, service_level, payment_term, shipping_term).

func TestSalesOrders_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["customer"], "customer should be null without ?include=customer")
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null without ?include=bill_to_address")
	assert.Nil(t, got["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, got["carrier"], "carrier should be null without ?include=carrier")
	assert.Nil(t, got["service_level"], "service_level should be null without ?include=service_level")
	assert.Nil(t, got["payment_term"], "payment_term should be null without ?include=payment_term")
	assert.Nil(t, got["shipping_term"], "shipping_term should be null without ?include=shipping_term")
	assert.Nil(t, got["order_discount"], "order_discount should be null without ?include=order_discount")
	assert.Nil(t, got["lines"], "lines should be null without ?include=lines")
}

func TestSalesOrders_IncludeCustomer(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cust := jsonObject(got, "customer")
	require.NotNil(t, cust, "customer should be present with ?include=customer")
	assert.Equal(t, "customer", jsonField(cust, "object"))
	assert.NotEmpty(t, jsonField(cust, "id"))
}

func TestSalesOrders_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["bill_to_address"]
	assert.True(t, ok, "bill_to_address key should be present with ?include=bill_to_address")
	if addr := jsonObject(got, "bill_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

func TestSalesOrders_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["ship_to_address"]
	assert.True(t, ok, "ship_to_address key should be present with ?include=ship_to_address")
	if addr := jsonObject(got, "ship_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

func TestSalesOrders_IncludeOrderDiscount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"order_discount"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	discount := jsonObject(got, "order_discount")
	require.NotNil(t, discount, "order_discount should be populated with ?include=order_discount (seed sets order_discount_id on ORD-001)")
	assert.Equal(t, "order_discount", jsonField(discount, "object"))
	assert.NotEmpty(t, jsonField(discount, "id"))
}

func TestSalesOrders_IncludeLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

// ──────────────────────────────────────────────
// SalesOrder — List parity (no summary object)
// ──────────────────────────────────────────────
//
// The list endpoint returns the full SalesOrder resource (there is no
// SalesOrderSummary). These tests pin that a list row can expand the same
// includes as detail, while inline scalars like line_count are always present.

// salesOrderListRow fetches the sales-order list with the given query params and
// returns the row for SeedSalesOrderID, failing if it is not on the page.
func salesOrderListRow(t *testing.T, params url.Values) map[string]any {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("limit", "100")

	status, body, err := apiClient.GetListRaw(salesOrdersPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	for _, item := range jsonArray(got, "data") {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(row, "id") == SeedSalesOrderID {
			return row
		}
	}
	require.FailNowf(t, "seed sales order not found in list", "id %s not in list response", SeedSalesOrderID)
	return nil
}

func TestSalesOrders_List_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, nil)

	// Inline scalars are always present on every row.
	assert.Equal(t, "sales_order", jsonField(row, "object"))
	assert.NotEmpty(t, jsonField(row, "number"))
	_, hasLineCount := row["line_count"]
	assert.True(t, hasLineCount, "line_count should always be present on a list row")

	// Expandable sub-resources are null until requested — same as detail.
	assert.Nil(t, row["customer"], "customer should be null without ?include=customer")
	assert.Nil(t, row["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, row["payment_term"], "payment_term should be null without ?include=payment_term")
	assert.Nil(t, row["lines"], "lines should be null without ?include=lines")
}

func TestSalesOrders_List_IncludeShipToAddress(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"ship_to_address"}})
	addr := jsonObject(row, "ship_to_address")
	require.NotNil(t, addr, "ship_to_address should be populated on the list row with ?include=ship_to_address")
	assert.Equal(t, "address", jsonField(addr, "object"))
}

func TestSalesOrders_List_IncludePaymentTerm(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"payment_term"}})
	term := jsonObject(row, "payment_term")
	require.NotNil(t, term, "payment_term should be populated on the list row with ?include=payment_term")
	assert.Equal(t, "payment_term", jsonField(term, "object"))
}

func TestSalesOrders_List_IncludeLines(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"lines"}})
	lines := jsonObject(row, "lines")
	require.NotNil(t, lines, "lines should be populated on the list row with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}
