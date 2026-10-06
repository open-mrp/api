//go:build e2e

package api_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
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

// --- Jobs ---

// A job the seller's staff started reads exactly as a missing one does; a portal still follows the jobs it starts itself.
func TestPortalRecordScope_PortalsReadOnlyTheJobsTheyStarted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productsPath+"/actions/export", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusAccepted, status, body)
	staffJobID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, staffJobID)

	missing := parseJSON(requireStatusAs(t, http.StatusNotFound, "customer portal", getCustomerPortalClient(), jobsPath+"/jb_e2eportalscopemissing0", nil))
	for who, portal := range portalClients(t) {
		got := parseJSON(requireStatusAs(t, http.StatusNotFound, who, portal, jobsPath+"/"+staffJobID, nil))
		assert.Equal(t, jsonField(jsonObject(missing, "error"), "message"), jsonField(jsonObject(got, "error"), "message"), "%s cannot tell the job exists", who)

		own := completedExportJobAs(t, portal, productsPath+"/actions/export", nil)
		assert.Equal(t, "completed", jsonField(own, "status"), "%s follows its own export to completion", who)
	}
}

// --- Sweeps ---

var accountIDPattern = regexp.MustCompile(`"(ac_[0-9a-z_]+)"`)

// portalOwnAccounts is the accounts a portal may meet in what it reads: the seller, its own, and its own parent and child accounts, which an include reaches along its relation.
func portalOwnAccounts(t *testing.T, own string) map[string]bool {
	t.Helper()
	accounts := map[string]bool{SeedAccountID: true, own: true}
	status, body, err := apiClient.GetListRaw(customersPath+"/"+own, url.Values{"include": {"parent_account", "child_accounts"}})
	require.NoError(t, err)
	if status != http.StatusOK {
		return accounts
	}
	customer := parseJSON(body)
	if parent := jsonField(jsonObject(customer, "parent_account"), "id"); parent != "" {
		accounts[parent] = true
	}
	for _, child := range jsonListData(customer, "child_accounts") {
		accounts[jsonField(child.(map[string]any), "id")] = true
	}
	return accounts
}

func otherAccountsIn(body []byte, own map[string]bool) []string {
	seen := map[string]bool{}
	for _, m := range accountIDPattern.FindAllSubmatch(body, -1) {
		if id := string(m[1]); !own[id] {
			seen[id] = true
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func TestPortalRecordScope_CustomerPortalSweepFindsNoOtherBuyersRecords(t *testing.T) {
	t.Parallel()
	newOtherBuyer(t)
	portalRecordSweep(t, "customer portal", getCustomerPortalClient(), portalOwnAccounts(t, SeedCustomerAccountID))
}

func TestPortalRecordScope_SupplierPortalSweepFindsNoOtherBuyersRecords(t *testing.T) {
	t.Parallel()
	newOtherBuyer(t)
	portalRecordSweep(t, "supplier portal", getSupplierPortalClient(t), portalOwnAccounts(t, SeedSupplierAccountID))
}

// portalRecordSweep calls every GET the spec documents as a portal, bare and with its includes, and fails on any account in an answer that is not the portal's own or the seller's. A record path is read at the first record the portal's own list returned (or a seed record), and again at the first record the seller's staff list, which belongs to whoever the seller dealt with last.
func portalRecordSweep(t *testing.T, who string, client *Client, own map[string]bool) {
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

	check := func(what string, body []byte) {
		if others := otherAccountsIn(body, own); len(others) > 0 {
			t.Errorf("%s received records of other accounts from %s: %s", who, what, strings.Join(others, ", "))
		}
	}
	read := func(specPath, path string) (bare []byte, ok bool) {
		status, bare := getAs(t, client, path)
		if status != http.StatusOK || !json.Valid(bare) {
			return nil, false
		}
		check("GET "+path, bare)
		includes := operationIncludes(spec.Paths[specPath]["get"])
		if len(includes) == 0 {
			return bare, true
		}
		if status, body := getAs(t, client, path, includes...); status == http.StatusOK {
			check("GET "+path+" with every include", body)
			return bare, true
		}
		for _, include := range includes {
			if status, body := getAs(t, client, path, include); status == http.StatusOK {
				check("GET "+path+"?include="+include, body)
			}
		}
		return bare, true
	}

	firstIDs := map[string]string{}
	reached := 0
	for _, specPath := range paths {
		if portalSweepSkipped(specPath) {
			continue
		}
		path, ok := portalSweepPath(specPath, firstIDs)
		if !ok {
			continue
		}
		if body, ok := read(specPath, path); ok {
			reached++
			if id := firstListID(body); id != "" {
				firstIDs[specPath] = id
			}
		}

		params := pathParamsOf(specPath)
		if i := strings.LastIndex(specPath, "/"); len(params) == 1 && strings.HasSuffix(specPath, "}") {
			if status, body, err := apiClient.GetListRaw(specPath[:i], nil); err == nil && status == http.StatusOK {
				if staffID := firstListID(body); staffID != "" && specPath[:i]+"/"+staffID != path {
					read(specPath, specPath[:i]+"/"+staffID)
				}
			}
		}
	}
	assert.Positive(t, reached, "%s reached no endpoint at all, so the sweep checked nothing", who)
}
