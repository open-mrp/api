//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Whoever may make a request gets everything it includes, whatever they may read directly: the request's own permission
// covers its includes. Reading an included type through its own endpoints still needs that type's permission.

const productionFlowsByItemPath = "/v1/operations/production-flows/by-item/"

// includedIDs follows a dotted include path through a response, through every element of each list on the way, and
// returns the sorted ids it reaches.
func includedIDs(v any, path string) []string {
	nodes := []any{v}
	for _, key := range strings.Split(path, ".") {
		var next []any
		for _, n := range flattenListNodes(nodes) {
			if m, ok := n.(map[string]any); ok && m[key] != nil {
				next = append(next, m[key])
			}
		}
		nodes = next
	}
	var ids []string
	for _, n := range flattenListNodes(nodes) {
		if m, ok := n.(map[string]any); ok {
			if id := jsonField(m, "id"); id != "" {
				ids = append(ids, id)
			}
		}
	}
	sort.Strings(ids)
	return ids
}

func flattenListNodes(nodes []any) []any {
	var out []any
	for _, n := range nodes {
		switch v := n.(type) {
		case []any:
			out = append(out, v...)
		case map[string]any:
			if jsonField(v, "object") == "list" {
				out = append(out, jsonArray(v, "data")...)
			} else {
				out = append(out, v)
			}
		}
	}
	return out
}

func includeParams(includes []string, extra url.Values) url.Values {
	params := url.Values{"include": {strings.Join(includes, ",")}}
	for k, vs := range extra {
		params[k] = vs
	}
	return params
}

// requireSameIncludes reads path as the admin and as client with the same includes and requires client to reach every
// record the admin reaches, none of them missing.
func requireSameIncludes(t *testing.T, client *Client, path string, includes ...string) map[string]any {
	t.Helper()
	params := includeParams(includes, nil)
	want := parseJSON(mustGetAs(t, apiClient, path, params))
	got := parseJSON(mustGetAs(t, client, path, params))
	assertSameIncludes(t, want, got, path, includes...)
	return got
}

func assertSameIncludes(t *testing.T, want, got map[string]any, where string, includes ...string) {
	t.Helper()
	for _, include := range includes {
		wantIDs := includedIDs(want, include)
		require.NotEmpty(t, wantIDs, "the fixture must reach %s on %s", include, where)
		assert.Equal(t, wantIDs, includedIDs(got, include), "%s on %s", include, where)
	}
}

// listedRow lists path with the includes as client and returns the row whose id is id, failing when it is not listed.
func listedRow(t *testing.T, client *Client, path string, extra url.Values, id string, includes ...string) map[string]any {
	t.Helper()
	body := mustGetAs(t, client, path, includeParams(includes, extra))
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row, ok := raw.(map[string]any); ok && jsonField(row, "id") == id {
			return row
		}
	}
	require.Failf(t, "row not listed", "%s %v has no %s: %s", path, extra, id, string(body))
	return nil
}

// requireRefusedRead requires client to be refused reading path for want of a permission.
func requireRefusedRead(t *testing.T, client *Client, path string) {
	t.Helper()
	status, body, err := client.GetListRaw(path, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status, "%s must stay refused on its own: %s", path, string(body))
}

// --- Internal roles holding only the request's permission ---

func TestIncludeReads_SalesOrderReaderGetsEveryInclude(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "sales_orders:read")
	includes := []string{
		"customer", "lines.product", "lines.product.item", "lines.product.product_line",
		"lines.quantity_ordered.unit", "lines.unit_price.numerator_unit",
		"related.pick", "related.production_run", "related.shipments", "related.invoices",
	}

	retrieved := requireSameIncludes(t, reader, salesOrdersPath+"/"+SeedSalesOrderID, includes...)

	number := jsonField(retrieved, "number")
	require.NotEmpty(t, number)
	listed := listedRow(t, reader, salesOrdersPath, url.Values{"q": {number}}, SeedSalesOrderID, includes...)
	assertSameIncludes(t, retrieved, listed, "the listed order", includes...)

	requireRefusedRead(t, reader, itemsPath+"/"+SeedItemID)
	requireRefusedRead(t, reader, customersPath+"/"+SeedCustomerAccountID)
	requireRefusedRead(t, reader, invoicesPath+"/"+SeedInvoiceID)
	requireRefusedRead(t, reader, picksPath+"/"+SeedPickID)
	requireRefusedRead(t, reader, productLinesPath+"/"+SeedProductLineID)
}

func TestIncludeReads_ReceivingOrderReaderGetsLineItemsAndOrderLines(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "receiving_orders:read")

	requireSameIncludes(t, reader, receivingOrdersPath+"/rcor_01seedrecvorder1_0",
		"supplier", "lines.item", "lines.order_line", "lines.order_line.item", "lines.quantity.unit")

	requireRefusedRead(t, reader, itemsPath+"/"+SeedMaterialItemID)
}

func TestIncludeReads_CustomerReaderGetsCarrierAndSalesRepUser(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "customers:read")

	requireSameIncludes(t, reader, customersPath+"/"+SeedCustomerAccountID,
		"freight_preferences.carrier", "freight_preferences.service_level", "defaults.sales_rep", "defaults.sales_rep.user", "parent_account")

	requireRefusedRead(t, reader, carriersPath+"/"+SeedCarrierID)
	requireRefusedRead(t, reader, "/v1/identity/users/"+SeedUserID)
}

func TestIncludeReads_InventoryLogReaderGetsItemUserAndStation(t *testing.T) {
	t.Parallel()
	const logID = "ivcl_01seedwss000000000"
	reader := customRoleClient(t, "inventory_logs:read")
	includes := []string{"item", "responsible_user", "responsible_scanning_station"}

	retrieved := requireSameIncludes(t, reader, inventoryChangeLogsPath+"/"+logID, includes...)
	listed := listedRow(t, reader, inventoryChangeLogsPath, url.Values{"item_ids": {SeedItemID}, "limit": {"100"}}, logID, includes...)
	assertSameIncludes(t, retrieved, listed, "the listed log", includes...)

	requireRefusedRead(t, reader, itemsPath+"/"+SeedItemID)
	requireRefusedRead(t, reader, scanningStationsPath+"/"+SeedScanningStationID)
}

func TestIncludeReads_SupplierReaderGetsMaterialItems(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "suppliers:read")
	path := "/v1/operations/suppliers/" + SeedSupplierAccountID + "/materials"
	includes := []string{"material", "material.item"}

	want := listedRow(t, apiClient, path, nil, SeedMaterialID, includes...)
	got := listedRow(t, reader, path, nil, SeedMaterialID, includes...)
	assertSameIncludes(t, want, got, path, includes...)
	requireSameIncludes(t, reader, path+"/"+SeedMaterialID, includes...)

	requireRefusedRead(t, reader, materialsPath+"/"+SeedMaterialID)
}

func TestIncludeReads_TeamReaderGetsUserRoleAndDepartment(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "team:read")
	includes := []string{"user", "role", "department"}

	retrieved := requireSameIncludes(t, reader, accountUsersPath+"/"+SeedAccountUser2ID, includes...)
	listed := listedRow(t, reader, accountUsersPath, url.Values{"limit": {"100"}}, SeedAccountUser2ID, includes...)
	assertSameIncludes(t, retrieved, listed, "the listed account user", includes...)

	requireRefusedRead(t, reader, rolesPath+"/"+SeedSalesRepRoleID)
	requireRefusedRead(t, reader, departmentsPath+"/"+SeedDepartmentID)
}

func TestIncludeReads_TerritoryReaderGetsSalesRepUserAndProductLine(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "sales_rep_territories:read")

	requireSameIncludes(t, reader, territoryPath("tr_01seedterritory1_000"), "sales_rep", "sales_rep.user", "product_line")

	requireRefusedRead(t, reader, productLinesPath+"/"+SeedProductLineID)
}

func TestIncludeReads_AccountReaderGetsDefaultAddresses(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "self:read")

	got := requireSameIncludes(t, reader, accountsPath+"/"+SeedAccountID, "default_billing_address", "default_shipping_address")

	requireRefusedRead(t, reader, addressesPath+"/"+jsonField(jsonObject(got, "default_billing_address"), "id"))
}

func TestIncludeReads_CustomerInvoicesShowTheParentAccount(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "invoices:read")
	path := customerInvoicesPathFor(SeedCustomerAccountID)
	includes := []string{"customer", "parent_account"}

	first := jsonArray(parseJSON(mustGetAs(t, apiClient, path, includeParams(includes, url.Values{"limit": {"1"}}))), "data")
	require.NotEmpty(t, first, "the seed customer has invoices to pay")
	want := first[0].(map[string]any)

	got := listedRow(t, reader, path, url.Values{"q": {jsonField(want, "number")}}, jsonField(want, "id"), includes...)
	assertSameIncludes(t, want, got, path, includes...)

	requireRefusedRead(t, reader, accountsPath+"/"+jsonField(jsonObject(want, "parent_account"), "id"))
}

func TestIncludeReads_ProductionStepReaderGetsItemsMachinesStationAndDepartment(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "production_steps:read")
	includes := []string{
		"production.produced_item", "consumptions.consumed_item", "machines", "machines.department",
		"scanning_station", "department",
	}

	retrieved := requireSameIncludes(t, reader, productionStepsPath+"/"+SeedSewLargeProductionStepID, includes...)
	listed := listedRow(t, reader, productionStepsPath, url.Values{"item_ids": {SeedLsnItemID}, "limit": {"100"}}, SeedSewLargeProductionStepID, includes...)
	assertSameIncludes(t, retrieved, listed, "the listed step", includes...)

	requireSameIncludes(t, reader, productionFlowsByItemPath+SeedLsnItemID,
		"steps", "steps.production.produced_item", "steps.consumptions.consumed_item", "steps.machines", "steps.scanning_station", "steps.department")

	requireRefusedRead(t, reader, itemsPath+"/"+SeedLsnItemID)
	requireRefusedRead(t, reader, machinesPath+"/"+SeedMachineID)
	requireRefusedRead(t, reader, departmentsPath+"/"+SeedDepartmentID)
}

func TestIncludeReads_DeliveryReaderGetsLineItemsLocationsAndOrderLines(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "deliveries:read")

	requireSameIncludes(t, reader, deliveriesPath+"/"+SeedDeliveryID, "lines.item", "lines.location", "lines.order_line", "lines.quantity.unit")

	requireRefusedRead(t, reader, locationsPath+"/"+SeedLocationID)
}

func TestIncludeReads_PickAndInvoiceReadersGetCustomerAndRelatedRecords(t *testing.T) {
	t.Parallel()

	picks := customRoleClient(t, "picks:read")
	requireSameIncludes(t, picks, picksPath+"/"+SeedPickID,
		"customer", "related.sales_order", "related.shipments", "lines.item", "lines.sales_order_line")
	requireRefusedRead(t, picks, customersPath+"/"+SeedCustomerAccountID)
	requireRefusedRead(t, picks, salesOrdersPath+"/"+SeedSalesOrderID)

	invoices := customRoleClient(t, "invoices:read")
	requireSameIncludes(t, invoices, invoicesPath+"/"+SeedInvoiceID,
		"customer", "order", "shipment", "related.sales_order", "related.shipment", "lines.item", "lines.order_line")
	requireRefusedRead(t, invoices, shipmentsPath+"/"+SeedShipmentID)
	requireRefusedRead(t, invoices, salesOrdersPath+"/"+SeedSalesOrderID)
}

func TestIncludeReads_ProductLineAccessEmbedsItsCustomerGroupAndLines(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "relevant_products:read")

	path := customerAccessPath + "/" + SeedCustomerAccountID
	want := parseJSON(mustGetAs(t, apiClient, path, nil))
	got := parseJSON(mustGetAs(t, reader, path, nil))
	assertSameIncludes(t, want, got, path, "customer", "product_lines")

	groupAccess := func(client *Client) map[string]any {
		for _, raw := range jsonArray(parseJSON(mustGetAs(t, client, accountGroupAccessPath, url.Values{"limit": {"100"}})), "data") {
			row := raw.(map[string]any)
			if jsonField(jsonObject(row, "account_group"), "id") == SeedCustomerGroupID {
				return row
			}
		}
		return nil
	}
	wantGroup := groupAccess(apiClient)
	require.NotNil(t, wantGroup, "the seed customer group has product line access")
	assertSameIncludes(t, wantGroup, groupAccess(reader), accountGroupAccessPath, "account_group", "product_lines")

	requireRefusedRead(t, reader, customersPath+"/"+SeedCustomerAccountID)
	requireRefusedRead(t, reader, productLinesPath+"/"+SeedProductLineID)
}

// --- Writes ---

// A write that answers with includes gets them under the write permission alone, and the include rule never lets a
// role write what it may only read.
func TestIncludeReads_WriteIncludesNeedOnlyTheWritePermission(t *testing.T) {
	t.Parallel()
	body := validCustomerBody(uniqueName("e2e-incl-write"))
	body["default_sales_rep_id"] = SeedAccountUserID
	customerID := jsonField(createAndCleanup(t, customersPath, body), "id")
	path := customersPath + "/" + customerID
	includes := []string{"freight_preferences.carrier", "defaults.sales_rep", "defaults.sales_rep.user"}
	params := includeParams(includes, nil)

	updater := customRoleClient(t, "customers:update")
	status, raw, err := updater.Patch(path+"?"+params.Encode(), map[string]any{"note": "e2e include write"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	want := parseJSON(mustGetAs(t, apiClient, path, params))
	assert.Equal(t, "e2e include write", jsonField(want, "note"))
	assertSameIncludes(t, want, parseJSON(raw), "the patched customer", includes...)

	reader := customRoleClient(t, "customers:read")
	status, raw, err = reader.Patch(path+"?"+params.Encode(), map[string]any{"note": "e2e read-only write"}, newIdempotencyKey())
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status, "a reader may not write, includes or not: %s", string(raw))
	assert.Equal(t, "e2e include write", jsonField(parseJSON(mustGetAs(t, apiClient, path, nil)), "note"), "the refused write changed nothing")
}

// --- Customer portal ---

// A buyer reading its own order gets everything the order includes, the seller's internal records among them.
func TestIncludeReads_PortalBuyerGetsEveryIncludeOnItsOwnOrder(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()

	requireSameIncludes(t, portal, salesOrdersPath+"/"+SeedSalesOrderID,
		"customer", "lines.product", "lines.product.item", "lines.product.product_line", "lines.quantity_ordered.unit",
		"related.pick", "related.production_run", "related.shipments", "related.invoices")

	requireSameIncludes(t, portal, customersPath+"/"+SeedCustomerAccountID,
		"freight_preferences.carrier", "defaults.sales_rep", "defaults.sales_rep.user")
}

// Includes never widen what the buyer may request: another buyer's order and the seller's own records stay out of reach.
func TestIncludeReads_PortalBuyerStillCannotRequestWhatIsNotItsOwn(t *testing.T) {
	t.Parallel()
	portal := getCustomerPortalClient()

	status, body, err := portal.GetListRaw(salesOrdersPath+"/"+SeedInternalSalesOrderID, url.Values{"include": {"customer,lines.product.item,related.invoices"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status, "the seller's own order is not the buyer's: %s", string(body))

	requireRefusedRead(t, portal, itemsPath+"/"+SeedItemID)
	requireRefusedRead(t, portal, invoicesPath+"/"+SeedInvoiceID)
	requireRefusedRead(t, portal, picksPath+"/"+SeedPickID)
}

// --- Tenant isolation ---

// Even a record that names another account's record never brings it back through an include: loads stay on the
// requesting account.
func TestIncludeReads_NeverReachAnotherTenant(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	status, raw, err := tenantB.Post(departmentsPath, map[string]any{"name": uniqueName("e2e-incl-tenant")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	departmentID := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { _, _, _ = tenantB.Delete(departmentsPath + "/" + departmentID) })

	_, err = authDB(t).Exec(`UPDATE department SET location_id = ? WHERE id = ? AND account_id = ?`, SeedLocationID, departmentID, SeedTenantBAccountID)
	require.NoError(t, err)

	got := parseJSON(mustGetAs(t, tenantB, departmentsPath+"/"+departmentID, url.Values{"include": {"location"}}))
	assertNilField(t, got, "location")
	listed := listedRow(t, tenantB, departmentsPath, url.Values{"limit": {"100"}}, departmentID, "location")
	assertNilField(t, listed, "location")

	status, body, err := tenantB.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"customer,lines.product.item,related.invoices"}})
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status, "another tenant's order stays unknown, includes or not: %s", string(body))
}
