//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A write asks for the endpoint's own permission in the seller's own account. In a customer's or supplier's account it
// asks for the customers or suppliers permission the service has always used there, and the own permission does not
// open it; the customers and suppliers permissions no longer open the seller's own records.

type counterpartyWrite struct {
	name string
	own  string
	// customer and supplier are what a customer's and a supplier's account take.
	customer, supplier string
	method             string
	// refusedPath and refusedBody make a well-formed request for a record that does not exist, so a caller who is wrongly let through changes nothing.
	refusedPath string
	refusedBody map[string]any
	// ownWrite makes the write as c in the seller's own account, on a record it creates there first.
	ownWrite func(t *testing.T, c *Client) (int, []byte)
	wantOwn  int
	// counterpartyWrite makes it as c in account, which c acts in; nil where the service lets nobody write in a counterparty's account.
	counterpartyWrite func(t *testing.T, c *Client, account string) (int, []byte)
	// wantCounterparty is the status a permitted write gets there: a record made in that account is written, and one of the seller's own reads as not found.
	wantCounterparty int
}

// Records no account holds, for requests that must be refused before they are looked up.
const (
	counterpartyMissingMaterial    = "ml_01cpmissing00000000"
	counterpartyMissingPart        = "pt_01cpmissing00000000"
	counterpartyMissingAccountUser = "acus_01cpmissing000000"
	counterpartyMissingOrder       = "or_01cpmissing00000000"
	counterpartyMissingLine        = "orln_01cpmissing000000"
	counterpartyMissingAddress     = "ad_01cpmissing00000000"
	counterpartyMissingCustomer    = "ac_01cpmissing00000000"
)

var counterpartyDescription = map[string]any{"description": "e2e-cp-refused"}

func counterpartyPost(t *testing.T, c *Client, path string, body map[string]any) (int, []byte) {
	t.Helper()
	status, resp, err := c.Post(path, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, resp
}

func counterpartyPatch(t *testing.T, c *Client, path string, body map[string]any) (int, []byte) {
	t.Helper()
	status, resp, err := c.Patch(path, body, newIdempotencyKey())
	require.NoError(t, err)
	return status, resp
}

func counterpartyPut(t *testing.T, c *Client, path string) (int, []byte) {
	t.Helper()
	status, resp, err := c.Put(path, nil)
	require.NoError(t, err)
	return status, resp
}

func counterpartyDelete(t *testing.T, c *Client, path string) (int, []byte) {
	t.Helper()
	status, resp, err := c.Delete(path)
	require.NoError(t, err)
	return status, resp
}

func counterpartyRefusedCall(t *testing.T, c *Client, w counterpartyWrite) (int, []byte) {
	t.Helper()
	var status int
	var body []byte
	switch w.method {
	case http.MethodPost:
		status, body = counterpartyPost(t, c, w.refusedPath, w.refusedBody)
	case http.MethodPatch:
		status, body = counterpartyPatch(t, c, w.refusedPath, w.refusedBody)
	case http.MethodPut:
		status, body = counterpartyPut(t, c, w.refusedPath)
	default:
		status, body = counterpartyDelete(t, c, w.refusedPath)
	}
	if status == http.StatusCreated {
		id := jsonField(parseJSON(body), "id")
		t.Cleanup(func() { _, _, _ = c.Delete(w.refusedPath + "/" + id) })
	}
	return status, body
}

// counterpartyCatalogItem creates a material or part in the seller's own account for a test to write to.
func counterpartyCatalogItem(t *testing.T, path string, body map[string]any) string {
	t.Helper()
	return jsonField(createAndCleanup(t, path, body), "id")
}

// counterpartyContact adds a user to account, the seller's own team when account is SeedAccountID, and removes it afterwards.
func counterpartyContact(t *testing.T, account string) string {
	t.Helper()
	name, email := contactEmail("e2e-cp-write-user")
	admin := apiClient.WithAccountID(account)
	id := jsonField(createContact(t, admin, name, email, nil), "id")
	t.Cleanup(func() { _, _, _ = admin.Put(accountUsersPath+"/"+id+"/actions/remove", nil) })
	return id
}

func counterpartyNewContact(t *testing.T, c *Client, account string) (int, []byte) {
	t.Helper()
	name, email := contactEmail("e2e-cp-write-new")
	status, body := counterpartyPost(t, c, accountUsersPath, map[string]any{"name": name, "email": email})
	if status == http.StatusCreated {
		id := jsonField(parseJSON(body), "id")
		t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(account).Put(accountUsersPath+"/"+id+"/actions/remove", nil) })
	}
	return status, body
}

func counterpartyAddress(t *testing.T, account string) string {
	t.Helper()
	id := addressOn(t, apiClient, account, uniqueName("e2e-cp-write-addr"))
	t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(account).Delete(addressesPath + "/" + id) })
	return id
}

func counterpartyNewAddress(t *testing.T, c *Client, account string) (int, []byte) {
	t.Helper()
	status, body := counterpartyPost(t, c, addressesPath, map[string]any{"name": uniqueName("e2e-cp-write-addr-new"), "country": "US"})
	if status == http.StatusCreated {
		id := jsonField(parseJSON(body), "id")
		t.Cleanup(func() { _, _, _ = apiClient.WithAccountID(account).Delete(addressesPath + "/" + id) })
	}
	return status, body
}

// counterpartyPurchaseOrderLines makes an estimate with two lines and returns it and its lines.
func counterpartyPurchaseOrderLines(t *testing.T) (string, []string) {
	t.Helper()
	orderID := jsonField(createPurchaseOrder(t, nil), "id")
	status, body := counterpartyPost(t, apiClient, purchaseOrdersPath+"/"+orderID+"/lines", purchaseOrderLineBody(uniqueName("E2E-CP-PO")))
	requireStatus(t, http.StatusCreated, status, body)
	order := parseJSON(mustGetAs(t, apiClient, purchaseOrdersPath+"/"+orderID, url.Values{"include": {"lines"}}))
	var lines []string
	for _, line := range jsonListData(order, "lines") {
		lines = append(lines, jsonField(line.(map[string]any), "id"))
	}
	require.Len(t, lines, 2)
	return orderID, lines
}

// counterpartySalesOrderLines makes an order with two sale lines and returns it and those lines.
func counterpartySalesOrderLines(t *testing.T) (string, []string) {
	t.Helper()
	orderID := createLifecycleOrder(t)
	addSaleLine(t, orderID, uniqueName("E2E-CP-SO"))
	lines := orderLineRoles(t, orderID).saleIDs
	require.Len(t, lines, 2)
	return orderID, lines
}

func counterpartySaleLineBody() map[string]any {
	return map[string]any{"product_id": SeedProductID, "product_sku": uniqueName("E2E-CP-SO"), "quantity": map[string]any{"value": "1", "unit_id": SeedUnitID}}
}

var counterpartyWrites = []counterpartyWrite{
	{
		name: "create material", own: "materials:create", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPost, refusedPath: materialsPath, refusedBody: validMaterialBody("e2e-cp-refused"), wantOwn: http.StatusCreated,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			status, body := counterpartyPost(t, c, materialsPath, validMaterialBody(uniqueName("e2e-cp-mat")))
			if status == http.StatusCreated {
				id := jsonField(parseJSON(body), "id")
				t.Cleanup(func() { _, _, _ = apiClient.Delete(materialsPath + "/" + id) })
			}
			return status, body
		},
	},
	{
		name: "update material", own: "materials:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: materialsPath + "/" + counterpartyMissingMaterial, refusedBody: counterpartyDescription, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			id := counterpartyCatalogItem(t, materialsPath, validMaterialBody(uniqueName("e2e-cp-mat")))
			return counterpartyPatch(t, c, materialsPath+"/"+id, map[string]any{"description": uniqueName("e2e-cp")})
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPatch(t, c, materialsPath+"/"+SeedMaterialID, map[string]any{"description": uniqueName("e2e-cp")})
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "delete material", own: "materials:delete", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodDelete, refusedPath: materialsPath + "/" + counterpartyMissingMaterial, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			id := counterpartyCatalogItem(t, materialsPath, validMaterialBody(uniqueName("e2e-cp-mat")))
			return counterpartyDelete(t, c, materialsPath+"/"+id)
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyDelete(t, c, materialsPath+"/"+SeedMaterialID)
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "create part", own: "parts:create", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPost, refusedPath: partsPath, refusedBody: validPartBody("e2e-cp-refused"), wantOwn: http.StatusCreated,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			status, body := counterpartyPost(t, c, partsPath, validPartBody(uniqueName("e2e-cp-part")))
			if status == http.StatusCreated {
				id := jsonField(parseJSON(body), "id")
				t.Cleanup(func() { _, _, _ = apiClient.Delete(partsPath + "/" + id) })
			}
			return status, body
		},
	},
	{
		name: "update part", own: "parts:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: partsPath + "/" + counterpartyMissingPart, refusedBody: counterpartyDescription, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			id := counterpartyCatalogItem(t, partsPath, validPartBody(uniqueName("e2e-cp-part")))
			return counterpartyPatch(t, c, partsPath+"/"+id, map[string]any{"description": uniqueName("e2e-cp")})
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPatch(t, c, partsPath+"/"+SeedPartID, map[string]any{"description": uniqueName("e2e-cp")})
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "delete part", own: "parts:delete", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodDelete, refusedPath: partsPath + "/" + counterpartyMissingPart, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			id := counterpartyCatalogItem(t, partsPath, validPartBody(uniqueName("e2e-cp-part")))
			return counterpartyDelete(t, c, partsPath+"/"+id)
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyDelete(t, c, partsPath+"/"+SeedPartID)
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "create account user", own: "team:create", customer: "customers:create", supplier: "suppliers:create",
		method: http.MethodPost, refusedPath: accountUsersPath, refusedBody: map[string]any{"name": "e2e-cp-refused", "email": "e2e-cp-refused@e2e-test.openmrp.ai"}, wantOwn: http.StatusCreated,
		ownWrite:          func(t *testing.T, c *Client) (int, []byte) { return counterpartyNewContact(t, c, SeedAccountID) },
		counterpartyWrite: counterpartyNewContact, wantCounterparty: http.StatusCreated,
	},
	{
		name: "update account user", own: "team:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: accountUsersPath + "/" + counterpartyMissingAccountUser, refusedBody: map[string]any{"name": "e2e-cp-refused"}, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPatch(t, c, accountUsersPath+"/"+counterpartyContact(t, SeedAccountID), map[string]any{"name": uniqueName("e2e-cp")})
		},
		counterpartyWrite: func(t *testing.T, c *Client, account string) (int, []byte) {
			return counterpartyPatch(t, c, accountUsersPath+"/"+counterpartyContact(t, account), map[string]any{"name": uniqueName("e2e-cp")})
		},
		wantCounterparty: http.StatusOK,
	},
	{
		name: "disable account user", own: "team:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPut, refusedPath: accountUsersPath + "/" + counterpartyMissingAccountUser + "/actions/disable", wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPut(t, c, accountUsersPath+"/"+counterpartyContact(t, SeedAccountID)+"/actions/disable")
		},
		counterpartyWrite: func(t *testing.T, c *Client, account string) (int, []byte) {
			return counterpartyPut(t, c, accountUsersPath+"/"+counterpartyContact(t, account)+"/actions/disable")
		},
		wantCounterparty: http.StatusOK,
	},
	{
		name: "activate account user", own: "team:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPut, refusedPath: accountUsersPath + "/" + counterpartyMissingAccountUser + "/actions/activate", wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPut(t, c, accountUsersPath+"/"+counterpartyContact(t, SeedAccountID)+"/actions/activate")
		},
		counterpartyWrite: func(t *testing.T, c *Client, account string) (int, []byte) {
			return counterpartyPut(t, c, accountUsersPath+"/"+counterpartyContact(t, account)+"/actions/activate")
		},
		wantCounterparty: http.StatusOK,
	},
	{
		name: "remove account user", own: "team:delete", customer: "customers:delete", supplier: "suppliers:delete",
		method: http.MethodPut, refusedPath: accountUsersPath + "/" + counterpartyMissingAccountUser + "/actions/remove", wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPut(t, c, accountUsersPath+"/"+counterpartyContact(t, SeedAccountID)+"/actions/remove")
		},
		counterpartyWrite: func(t *testing.T, c *Client, account string) (int, []byte) {
			return counterpartyPut(t, c, accountUsersPath+"/"+counterpartyContact(t, account)+"/actions/remove")
		},
		wantCounterparty: http.StatusOK,
	},
	{
		name: "create purchase order line", own: "purchase_orders:update", customer: "purchase_orders:update", supplier: "suppliers:update",
		method: http.MethodPost, refusedPath: purchaseOrdersPath + "/" + counterpartyMissingOrder + "/lines", refusedBody: purchaseOrderLineBody("E2E-CP-REFUSED"), wantOwn: http.StatusCreated,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			orderID := jsonField(createPurchaseOrder(t, nil), "id")
			return counterpartyPost(t, c, purchaseOrdersPath+"/"+orderID+"/lines", purchaseOrderLineBody(uniqueName("E2E-CP-PO")))
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPost(t, c, purchaseOrdersPath+"/"+SeedPurchaseOrderID+"/lines", purchaseOrderLineBody(uniqueName("E2E-CP-PO")))
		},
		// The seller's product is not the counterparty's either, and the line is refused for it before the order is looked up.
		wantCounterparty: http.StatusBadRequest,
	},
	{
		name: "update purchase order line", own: "purchase_orders:update", customer: "purchase_orders:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: purchaseOrdersPath + "/" + counterpartyMissingOrder + "/lines/" + counterpartyMissingLine, refusedBody: map[string]any{"product_sku": "E2E-CP-REFUSED"}, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			orderID, lines := counterpartyPurchaseOrderLines(t)
			return counterpartyPatch(t, c, purchaseOrdersPath+"/"+orderID+"/lines/"+lines[0], map[string]any{"product_sku": uniqueName("E2E-CP-PO")})
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPatch(t, c, purchaseOrdersPath+"/"+SeedPurchaseOrderID+"/lines/poln_000000000000", map[string]any{"product_sku": uniqueName("E2E-CP-PO")})
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "delete purchase order line", own: "purchase_orders:update", customer: "purchase_orders:update", supplier: "suppliers:update",
		method: http.MethodDelete, refusedPath: purchaseOrdersPath + "/" + counterpartyMissingOrder + "/lines/" + counterpartyMissingLine, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			orderID, lines := counterpartyPurchaseOrderLines(t)
			return counterpartyDelete(t, c, purchaseOrdersPath+"/"+orderID+"/lines/"+lines[1])
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyDelete(t, c, purchaseOrdersPath+"/"+SeedPurchaseOrderID+"/lines/poln_000000000000")
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "create address", own: "addresses:create", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPost, refusedPath: addressesPath, refusedBody: map[string]any{"name": "e2e-cp-refused", "country": "US"}, wantOwn: http.StatusCreated,
		ownWrite:          func(t *testing.T, c *Client) (int, []byte) { return counterpartyNewAddress(t, c, SeedAccountID) },
		counterpartyWrite: counterpartyNewAddress, wantCounterparty: http.StatusCreated,
	},
	{
		name: "update address", own: "addresses:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: addressesPath + "/" + counterpartyMissingAddress, refusedBody: map[string]any{"phone": "555-0100"}, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPatch(t, c, addressesPath+"/"+counterpartyAddress(t, SeedAccountID), map[string]any{"phone": "555-0100"})
		},
		counterpartyWrite: func(t *testing.T, c *Client, account string) (int, []byte) {
			return counterpartyPatch(t, c, addressesPath+"/"+counterpartyAddress(t, account), map[string]any{"phone": "555-0100"})
		},
		wantCounterparty: http.StatusOK,
	},
	{
		name: "delete address", own: "addresses:delete", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodDelete, refusedPath: addressesPath + "/" + counterpartyMissingAddress, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyDelete(t, c, addressesPath+"/"+counterpartyAddress(t, SeedAccountID))
		},
		counterpartyWrite: func(t *testing.T, c *Client, account string) (int, []byte) {
			return counterpartyDelete(t, c, addressesPath+"/"+counterpartyAddress(t, account))
		},
		wantCounterparty: http.StatusOK,
	},
	{
		name: "replace notification recipients", own: "customers:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: notifRecipientsPath(counterpartyMissingCustomer), refusedBody: notifRecipientBody(), wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPatch(t, c, notifRecipientsPath(createContactsCustomer(t)), notifRecipientBody())
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPatch(t, c, notifRecipientsPath(SeedCustomerAccountID), notifRecipientBody())
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "create sales order line", own: "sales_orders:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPost, refusedPath: salesOrdersPath + "/" + counterpartyMissingOrder + "/lines", refusedBody: counterpartySaleLineBody(), wantOwn: http.StatusCreated,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			return counterpartyPost(t, c, salesOrdersPath+"/"+createLifecycleOrder(t)+"/lines", counterpartySaleLineBody())
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPost(t, c, salesOrdersPath+"/"+SeedSalesOrderID+"/lines", counterpartySaleLineBody())
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "update sales order line", own: "sales_orders:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPatch, refusedPath: salesOrdersPath + "/" + counterpartyMissingOrder + "/lines/" + counterpartyMissingLine, refusedBody: map[string]any{"product_sku": "E2E-CP-REFUSED"}, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			orderID, lines := counterpartySalesOrderLines(t)
			return counterpartyPatch(t, c, salesOrdersPath+"/"+orderID+"/lines/"+lines[0], map[string]any{"product_sku": uniqueName("E2E-CP-SO")})
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPatch(t, c, salesOrdersPath+"/"+SeedSalesOrderID+"/lines/"+SeedSalesOrderLineID, map[string]any{"product_sku": uniqueName("E2E-CP-SO")})
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "delete sales order line", own: "sales_orders:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodDelete, refusedPath: salesOrdersPath + "/" + counterpartyMissingOrder + "/lines/" + counterpartyMissingLine, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			orderID, lines := counterpartySalesOrderLines(t)
			return counterpartyDelete(t, c, salesOrdersPath+"/"+orderID+"/lines/"+lines[1])
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyDelete(t, c, salesOrdersPath+"/"+SeedSalesOrderID+"/lines/"+SeedSalesOrderLineID)
		},
		wantCounterparty: http.StatusNotFound,
	},
	{
		name: "reorder sales order lines", own: "sales_orders:update", customer: "customers:update", supplier: "suppliers:update",
		method: http.MethodPost, refusedPath: salesOrdersPath + "/" + counterpartyMissingOrder + "/lines/actions/reorder", refusedBody: map[string]any{"line_ids": []string{counterpartyMissingLine}}, wantOwn: http.StatusOK,
		ownWrite: func(t *testing.T, c *Client) (int, []byte) {
			orderID, lines := counterpartySalesOrderLines(t)
			return counterpartyPost(t, c, salesOrdersPath+"/"+orderID+"/lines/actions/reorder", map[string]any{"line_ids": []string{lines[1], lines[0]}})
		},
		counterpartyWrite: func(t *testing.T, c *Client, _ string) (int, []byte) {
			return counterpartyPost(t, c, salesOrdersPath+"/"+SeedSalesOrderID+"/lines/actions/reorder", map[string]any{"line_ids": []string{SeedSalesOrderLineID}})
		},
		wantCounterparty: http.StatusNotFound,
	},
}

// withRead adds the read permission of each permission's domain: a write answers with the record it wrote, which the caller reads back.
func withRead(perms ...string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, p := range perms {
		domain, _, _ := strings.Cut(p, ":")
		for _, q := range []string{p, domain + ":read"} {
			if !seen[q] {
				seen[q] = true
				out = append(out, q)
			}
		}
	}
	return out
}

func TestCounterpartyWritePermissions(t *testing.T) {
	t.Parallel()
	roles := &counterpartyRoles{}
	alternatesOf := func(w counterpartyWrite) []string {
		var alts []string
		for _, p := range []string{w.customer, w.supplier} {
			if p != w.own {
				alts = append(alts, p)
			}
		}
		return withRead(alts...)
	}
	for _, w := range counterpartyWrites {
		roles.holding(t, withRead(w.own)...)
		roles.holding(t, alternatesOf(w)...)
		roles.holding(t, withRead(w.customer)...)
		roles.holding(t, withRead(w.supplier)...)
	}
	customerAccount := createContactsCustomer(t)
	supplierAccount := createContactsSupplier(t)

	for _, w := range counterpartyWrites {
		t.Run(w.name, func(t *testing.T) {
			t.Parallel()
			own := roles.holding(t, withRead(w.own)...)

			status, body := counterpartyRefusedCall(t, roles.holding(t, alternatesOf(w)...), w)
			requirePermissionRefused(t, status, body, w.own, w.customer, w.supplier)
			status, body = w.ownWrite(t, own)
			requireStatus(t, w.wantOwn, status, body)

			for _, kind := range []struct{ account, perm string }{{customerAccount, w.customer}, {supplierAccount, w.supplier}} {
				if kind.perm != w.own {
					status, body = counterpartyRefusedCall(t, own.WithAccountID(kind.account), w)
					requirePermissionRefused(t, status, body, kind.perm, w.own)
				}
				if w.counterpartyWrite == nil {
					continue
				}
				status, body = w.counterpartyWrite(t, roles.holding(t, withRead(kind.perm)...).WithAccountID(kind.account), kind.account)
				requireStatus(t, w.wantCounterparty, status, body)
			}
		})
	}
}
