//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A customer or supplier portal reads the seller's account only for what is its own, and a supplier relation opens nothing a customer relation does not. Another buyer's record reads as not found.

// otherBuyer is a second customer of the seller, with an issued order packed into a shipment and a price of its own.
type otherBuyer struct {
	accountID   string
	orderID     string
	orderNumber string
	shipmentID  string
	lineID      string
	priceID     string
}

func newOtherBuyer(t *testing.T) otherBuyer {
	t.Helper()
	b := otherBuyer{accountID: leadTimeCustomer(t, "e2e-portal-scope-buyer", nil, "")}
	b.orderID, _, b.shipmentID = packedOrder(t, minimalSalesOrderCreateBody(t, b.accountID))
	b.orderNumber = jsonField(getSalesOrder(t, b.orderID, nil), "number")
	lines := jsonListData(readShipment(t, b.shipmentID, "lines"), "lines")
	require.NotEmpty(t, lines, "the other buyer's shipment has lines")
	b.lineID = jsonField(lines[0].(map[string]any), "id")
	b.priceID = jsonField(createAccountPrice(t, b.accountID, "12.34"), "id")
	return b
}

func requireStatusAs(t *testing.T, want int, who string, client *Client, path string, params url.Values) []byte {
	t.Helper()
	status, body, err := client.GetListRaw(path, params)
	require.NoError(t, err)
	require.Equal(t, want, status, "%s GET %s: %s", who, path, string(body))
	return body
}

// --- Sales orders ---

func TestPortalRecordScope_AnotherBuyersSalesOrderIsHiddenFromEveryPortal(t *testing.T) {
	t.Parallel()
	other := newOtherBuyer(t)

	found := parseJSON(requireStatusAs(t, http.StatusOK, "staff", apiClient, salesOrdersPath, url.Values{"q": {other.orderNumber}}))
	require.Len(t, jsonArray(found, "data"), 1, "the seller's staff find the other buyer's order by its number")

	for who, portal := range portalClients(t) {
		list := parseJSON(requireStatusAs(t, http.StatusOK, who, portal, salesOrdersPath, url.Values{"q": {other.orderNumber}}))
		assert.Empty(t, jsonArray(list, "data"), "%s does not list another buyer's order", who)
		requireStatusAs(t, http.StatusNotFound, who, portal, salesOrdersPath+"/"+other.orderID, nil)
	}
}

// Every order a portal lists is one its own account bought, whichever way it relates to the seller.
func TestPortalRecordScope_PortalsListOnlyTheOrdersTheirAccountBought(t *testing.T) {
	t.Parallel()
	for who, c := range map[string]struct {
		client *Client
		own    string
	}{
		"customer portal": {getCustomerPortalClient(), SeedCustomerAccountID},
		"supplier portal": {getSupplierPortalClient(t), SeedSupplierAccountID},
	} {
		list := parseJSON(requireStatusAs(t, http.StatusOK, who, c.client, salesOrdersPath, url.Values{"include": {"customer"}, "limit": {"100"}}))
		for _, row := range jsonArray(list, "data") {
			order := row.(map[string]any)
			assert.Equal(t, c.own, jsonField(jsonObject(order, "customer"), "id"), "%s listed order %s", who, jsonField(order, "id"))
		}
	}

	own := parseJSON(requireStatusAs(t, http.StatusOK, "customer portal", getCustomerPortalClient(), salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"customer"}}))
	assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(own, "customer"), "id"), "the customer still reads its own order")
	requireStatusAs(t, http.StatusNotFound, "supplier portal", getSupplierPortalClient(t), salesOrdersPath+"/"+SeedSalesOrderID, nil)
}

// --- Customers ---

func TestPortalRecordScope_CustomerRecordsArePortalsOwn(t *testing.T) {
	t.Parallel()
	otherID := leadTimeCustomer(t, "e2e-portal-scope-cust", nil, "")
	supplier := getSupplierPortalClient(t)

	for _, path := range []string{customersPath + "/" + SeedCustomerAccountID, customersPath + "/" + SeedCustomerAccountID + "/frequently-ordered-products"} {
		requireStatusAs(t, http.StatusNotFound, "supplier portal", supplier, path, nil)
	}
	for who, portal := range portalClients(t) {
		for _, path := range []string{customersPath + "/" + otherID, customersPath + "/" + otherID + "/frequently-ordered-products"} {
			requireStatusAs(t, http.StatusNotFound, who, portal, path, nil)
		}
	}

	own := parseJSON(requireStatusAs(t, http.StatusOK, "customer portal", getCustomerPortalClient(), customersPath+"/"+SeedCustomerAccountID, nil))
	assert.Equal(t, SeedCustomerAccountID, jsonField(own, "id"))
	requireStatusAs(t, http.StatusOK, "customer portal", getCustomerPortalClient(), customersPath+"/"+SeedCustomerAccountID+"/frequently-ordered-products", nil)
	assert.Equal(t, SeedSupplierAccountID, jsonField(parseJSON(requireStatusAs(t, http.StatusOK, "supplier portal", supplier, suppliersPath+"/"+SeedSupplierAccountID, nil)), "id"),
		"the supplier still reads its own supplier record")
}

// --- Shipments ---

func TestPortalRecordScope_AnotherBuyersShipmentIsNotFound(t *testing.T) {
	t.Parallel()
	other := newOtherBuyer(t)

	requireStatusAs(t, http.StatusOK, "staff", apiClient, shipmentsPath+"/"+other.shipmentID+"/lines", nil)
	for who, portal := range portalClients(t) {
		for _, path := range []string{
			shipmentsPath + "/" + other.shipmentID,
			shipmentsPath + "/" + other.shipmentID + "/lines",
			shipmentsPath + "/" + other.shipmentID + "/lines/" + other.lineID,
		} {
			requireStatusAs(t, http.StatusNotFound, who, portal, path, nil)
		}
	}
}

func TestPortalRecordScope_CustomerReadsItsOwnShipmentAndLines(t *testing.T) {
	t.Parallel()
	_, _, shipmentID := packedOrder(t, orderBodyForQuantity(t, "1"))
	portal := getCustomerPortalClient()

	shipment := parseJSON(requireStatusAs(t, http.StatusOK, "customer portal", portal, shipmentsPath+"/"+shipmentID, url.Values{"include": {"customer"}}))
	assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(shipment, "customer"), "id"))
	lines := jsonArray(parseJSON(requireStatusAs(t, http.StatusOK, "customer portal", portal, shipmentsPath+"/"+shipmentID+"/lines", nil)), "data")
	require.NotEmpty(t, lines, "the customer lists its own shipment's lines")
	requireStatusAs(t, http.StatusOK, "customer portal", portal, shipmentsPath+"/"+shipmentID+"/lines/"+jsonField(lines[0].(map[string]any), "id"), nil)

	requireStatusAs(t, http.StatusNotFound, "supplier portal", getSupplierPortalClient(t), shipmentsPath+"/"+shipmentID, nil)
}

// --- Prices and discounts ---

func TestPortalRecordScope_AnotherBuyersPriceIsHiddenFromEveryPortal(t *testing.T) {
	t.Parallel()
	other := newOtherBuyer(t)
	requireStatusAs(t, http.StatusOK, "staff", apiClient, accountPricesPath+"/"+other.priceID, nil)

	for who, portal := range portalClients(t) {
		requireStatusAs(t, http.StatusNotFound, who, portal, accountPricesPath+"/"+other.priceID, nil)
		list := parseJSON(requireStatusAs(t, http.StatusOK, who, portal, accountPricesPath, url.Values{"recipient_account_id": {other.accountID}, "limit": {"100"}}))
		for _, row := range jsonArray(list, "data") {
			assert.NotEqual(t, other.priceID, jsonField(row.(map[string]any), "id"), "%s lists another buyer's price", who)
		}
	}
}

func TestPortalRecordScope_DiscountForAnotherGroupIsHiddenFromEveryPortal(t *testing.T) {
	t.Parallel()
	groupID := leadTimeAccountGroup(t, "e2e-portal-scope-grp", nil)
	leadTimeCustomer(t, "e2e-portal-scope-grp-cust", nil, groupID)
	discountID := jsonField(createVolumeDiscount(t, map[string]any{"customer_group_ids": []string{groupID}}), "id")
	requireStatusAs(t, http.StatusOK, "staff", apiClient, volumeDiscountsPath+"/"+discountID, nil)

	for who, portal := range portalClients(t) {
		requireStatusAs(t, http.StatusNotFound, who, portal, volumeDiscountsPath+"/"+discountID, nil)

		// Whatever the portal's listing carries, it can open.
		list := parseJSON(requireStatusAs(t, http.StatusOK, who, portal, volumeDiscountsPath, url.Values{"limit": {"100"}}))
		for _, row := range jsonArray(list, "data") {
			id := jsonField(row.(map[string]any), "id")
			assert.NotEqual(t, discountID, id, "%s lists another group's discount", who)
			requireStatusAs(t, http.StatusOK, who, portal, volumeDiscountsPath+"/"+id, nil)
		}
	}
}

// --- Billing ---

func TestPortalRecordScope_BillingUsageAndSpendingCapAreRefusedToPortals(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"/v1/billing/accounts/usage", "/v1/billing/spending-cap"} {
		requireStatusAs(t, http.StatusOK, "staff", apiClient, path, nil)
		for who, portal := range portalClients(t) {
			requireStatusAs(t, http.StatusForbidden, who, portal, path, nil)
		}
	}
}
