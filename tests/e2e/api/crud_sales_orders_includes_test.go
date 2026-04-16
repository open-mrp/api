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
	_, ok := got["order_discount"]
	assert.True(t, ok, "order_discount key should be present with ?include=order_discount")
	// order_discount may legitimately be null if no discount applied
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
