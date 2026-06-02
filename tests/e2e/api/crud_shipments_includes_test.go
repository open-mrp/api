//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Shipment — Additional Include Tests
// ──────────────────────────────────────────────
//
// Extends coverage beyond included_fields_test.go (which only tests carrier +
// service_level).

func TestShipments_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["lines"], "lines should be null without ?include=lines")
	assert.Nil(t, got["shipping_cases"], "shipping_cases should be null without ?include=shipping_cases")
	assert.Nil(t, got["sales_order"], "sales_order should be null without ?include=sales_order")
	assert.Nil(t, got["customer"], "customer should be null without ?include=customer")
	assert.Nil(t, got["carrier"], "carrier should be null without ?include=carrier")
	assert.Nil(t, got["service_level"], "service_level should be null without ?include=service_level")
	assert.Nil(t, got["shipping_address"], "shipping_address should be null without ?include=shipping_address")
	assert.Nil(t, got["shipped_by"], "shipped_by should be null without ?include=shipped_by")
	assert.Nil(t, got["invoice"], "invoice should be null without ?include=invoice")
	assert.Nil(t, got["pick"], "pick should be null without ?include=pick")
}

func TestShipments_IncludeLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonObject(parseJSON(body), "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

func TestShipments_IncludeShippingCases(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"shipping_cases"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	cases := jsonObject(parseJSON(body), "shipping_cases")
	require.NotNil(t, cases, "shipping_cases should be present with ?include=shipping_cases")
	assert.Equal(t, "list", jsonField(cases, "object"))
}

func TestShipments_IncludeSalesOrder(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"sales_order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	so := jsonObject(parseJSON(body), "sales_order")
	require.NotNil(t, so, "sales_order should be present with ?include=sales_order")
	assert.Equal(t, "sales_order", jsonField(so, "object"))
	assert.NotEmpty(t, jsonField(so, "id"))
}

func TestShipments_IncludeCustomer(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	cust := jsonObject(parseJSON(body), "customer")
	require.NotNil(t, cust, "customer should be present with ?include=customer")
	assert.Equal(t, "customer", jsonField(cust, "object"))
}

func TestShipments_IncludeShippingAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"shipping_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["shipping_address"]
	assert.True(t, ok, "shipping_address key should be present with ?include=shipping_address")
	if addr := jsonObject(got, "shipping_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

func TestShipments_IncludeShippedBy(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"shipped_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["shipped_by"]
	assert.True(t, ok, "shipped_by key should be present with ?include=shipped_by")
	// shipped_by may be null if not yet shipped
	if sb := jsonObject(got, "shipped_by"); sb != nil {
		assert.Equal(t, "account_user", jsonField(sb, "object"))
	}
}

func TestShipments_IncludeInvoice(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"invoice"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["invoice"]
	assert.True(t, ok, "invoice key should be present with ?include=invoice")
	if inv := jsonObject(got, "invoice"); inv != nil {
		assert.Equal(t, "invoice", jsonField(inv, "object"))
	}
}

func TestShipments_IncludePick(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+SeedShipmentID, url.Values{"include": {"pick"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["pick"]
	assert.True(t, ok, "pick key should be present with ?include=pick")
	if pick := jsonObject(got, "pick"); pick != nil {
		assert.Equal(t, "pick", jsonField(pick, "object"))
	}
}
