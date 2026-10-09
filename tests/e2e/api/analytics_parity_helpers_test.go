//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/textutil"
)

// Shared fixtures for the analytics parity tests: each test builds its own customers, orders and stock and
// filters every report to them, so totals are exact and parallel tests never see each other's data.

const (
	openOrdersPath          = "/v1/core/analytics/open-orders"
	openOrdersSummaryPath   = openOrdersPath + "/summary"
	openOrdersBreakdownPath = openOrdersPath + "/breakdown"
	openOrderLinesExport    = "/v1/core/analytics/open-order-lines/actions/export"
	quarterlyOrdersPath     = "/v1/core/analytics/quarterly-orders"
	inventoryReceiptsPath   = "/v1/core/analytics/inventory-receipts"
	analyticsMaterialsPath  = "/v1/core/analytics/materials"
	openBatchesPath         = "/v1/core/analytics/open-batches"
	weeksOfSalesPath        = "/v1/core/analytics/weeks-of-sales"
	demandForecastPath      = "/v1/core/analytics/demand-forecast"

	// Seed socks on SeedProductLineID, all counted in pairs (2 ea); a dozen is 12 ea, so 6 pr.
	parityProductSCK002 = "pd_01k0a65nx5e3haz2fgfm34hmcz"
	parityItemSCK002    = "it_01k0a7100aedgv8416p4p2v9ks"
	parityProductSCK003 = "pd_01k0a65nx5fjz8m1s3ytayfdby"
	parityItemSCK003    = "it_01k0a7100afdnr1b41917qs27k"
	parityProductSCK004 = "pd_01k0a65nx5eeavcs322b06pgr8"
)

// parityLine is one sale line: qty of unitID priced per pair.
func parityLine(productID, qty, unitID, pricePerPair string) map[string]any {
	return map[string]any{
		"product_id": productID,
		"quantity":   map[string]any{"value": qty, "unit_id": unitID},
		"unit_price": map[string]any{"value": pricePerPair, "numerator_unit_id": dollarUnitID, "denominator_unit_id": SeedUnitID},
	}
}

// parityCustomer creates a customer of its own in groupID (or the seed group when empty) with access to the
// socks and any further product lines.
func parityCustomer(t *testing.T, groupID string, productLineIDs ...string) string {
	t.Helper()
	body := validCustomerBody(uniqueName("e2e-parity-cust"))
	if groupID != "" {
		body["customer_type_group_id"] = groupID
	}
	created := createAndCleanup(t, customersPath, body)
	customerID := jsonField(created, "id")
	plStatus, plBody, err := apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id":      customerID,
		"product_line_ids": append([]string{SeedProductLineID}, productLineIDs...),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, plStatus, plBody)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productLineAccessPath + "/" + customerID) })
	return customerID
}

// makeChildCustomer files child under parent, which only an operator (or the parent account itself) can do.
func makeChildCustomer(t *testing.T, parentID, childID string) {
	t.Helper()
	db := authDB(t)
	var parentRelationID string
	require.NoError(t, db.QueryRow(`SELECT id FROM account_relation WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		SeedAccountID, parentID).Scan(&parentRelationID))
	_, err := db.Exec(`UPDATE account_relation SET parent_account_relation_id = ? WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
		parentRelationID, SeedAccountID, childID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE account_relation SET parent_account_relation_id = NULL WHERE owner_account_id = ? AND counterparty_account_id = ? AND account_relation_role_code = 'customer'`,
			SeedAccountID, childID)
	})
}

// issueParityOrder creates and issues an order of lines to customerID, optionally owned by a sales rep (account user).
func issueParityOrder(t *testing.T, customerID, salesRepID string, lines ...map[string]any) map[string]any {
	t.Helper()
	extra := map[string]any{"lines": lines}
	if salesRepID != "" {
		extra["sales_rep_id"] = salesRepID
	}
	return issueOrderForCustomer(t, customerID, extra)
}

// createParityEstimate creates an order and leaves it an estimate.
func createParityEstimate(t *testing.T, customerID string, lines ...map[string]any) map[string]any {
	t.Helper()
	body := minimalSalesOrderCreateBody(t, customerID)
	body["lines"] = lines
	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	order := parseJSON(respBody)
	require.Equal(t, "estimate", jsonField(order, "status"))
	deleteOrder(t, jsonField(order, "id"))
	return order
}

// shipPick packs what has been picked into one case and ships it, which invoices it.
func shipPick(t *testing.T, pickID string) {
	t.Helper()
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
}

// shipPartOfOrder picks qty of a one-line order (in the line's own unit) and ships it.
func shipPartOfOrder(t *testing.T, orderID, qty string) {
	t.Helper()
	pickID := orderPickID(t, orderID)
	lines := readPickLineQuantities(t, pickID)
	require.Len(t, lines, 1, "a one-line order raises a one-line pick")
	for lineID := range lines {
		status, body, err := apiClient.Patch(picksPath+"/"+pickID+"/lines/"+lineID, map[string]any{"quantity_value": qty}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}
	shipPick(t, pickID)
}

// shipWholeOrder picks, packs and ships every line, which completes the order.
func shipWholeOrder(t *testing.T, orderID string) {
	t.Helper()
	pickID := orderPickID(t, orderID)
	pickAllLines(t, pickID)
	shipPick(t, pickID)
}

func readOrder(t *testing.T, orderID string) map[string]any {
	t.Helper()
	return parseJSON(mustGet(t, salesOrdersPath+"/"+orderID))
}

// paddedNumber is a record number as the dashboard prints it.
func paddedNumber(number string) string {
	return textutil.FormatRecordNumber(number)
}

func putAnalytics(t *testing.T, client *Client, path string, params url.Values, body map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	if body == nil {
		body = map[string]any{}
	}
	status, respBody, err := client.PutRaw(path, params, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", path, string(respBody))
	if status != 200 {
		return status, nil, respBody
	}
	return status, parseJSON(respBody), respBody
}

func mustPutAnalytics(t *testing.T, client *Client, path string, params url.Values, body map[string]any) map[string]any {
	t.Helper()
	status, parsed, raw := putAnalytics(t, client, path, params, body)
	requireStatus(t, 200, status, raw)
	return parsed
}

func listRows(t *testing.T, list map[string]any) []map[string]any {
	t.Helper()
	require.Equal(t, "list", jsonField(list, "object"))
	var out []map[string]any
	for _, row := range jsonArray(list, "data") {
		out = append(out, row.(map[string]any))
	}
	return out
}

// paritySalesRep is a user signed in to the seed account under a sales-rep role of its own, holding exactly perms
// (permission codes, read only). Role assignment has no API: an operator grants it in SQL, as this does. Adding a member under a sales-rep role assigns the account's own sales-rep role, so the test role is set by an update afterwards.
func paritySalesRep(t *testing.T, perms ...string) (client *Client, accountUserID string) {
	t.Helper()
	db := authDB(t)
	suffix := uuid.New().String()[:12]
	roleID := "rl_e2eparity_" + suffix
	_, err := db.Exec(`INSERT INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES (?, ?, 'sales_rep', ?, NOW(3), NOW(3))`,
		roleID, "E2E parity rep "+suffix, SeedAccountID)
	require.NoError(t, err)
	for i, perm := range perms {
		_, err = db.Exec("INSERT INTO role_permission (id, `create`, `read`, `update`, `delete`, role_id, permission_code, created_at, updated_at) VALUES (?, 0, 1, 0, 0, ?, ?, NOW(3), NOW(3))",
			fmt.Sprintf("rlpm_e2eparity_%s_%d", suffix, i), roleID, perm)
		require.NoError(t, err)
	}

	_, email := covAuthPasswordsRegisterUser(t, "e2e-parity-rep")
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{"email": email, "role_id": roleID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID = jsonField(parseJSON(body), "id")
	require.NotEmpty(t, accountUserID)
	t.Cleanup(func() {
		removeAccountUser(accountUserID)
		_, _ = db.Exec("DELETE FROM role_permission WHERE role_id = ?", roleID)
		_, _ = db.Exec("DELETE FROM role WHERE id = ?", roleID)
	})
	status, body, err = apiClient.Patch(accountUsersPath+"/"+accountUserID+"?include=role", map[string]any{"role_id": roleID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	require.Equal(t, roleID, jsonField(jsonObject(parseJSON(body), "role"), "id"))
	return loginAsUser(t, email, covAuthUsersPassword, SeedAccountID), accountUserID
}

// customRoleClient is an API key on the seed account under a custom role holding perms.
func customRoleClient(t *testing.T, perms ...string) *Client {
	t.Helper()
	role := createAndCleanup(t, "/v1/identity/roles", map[string]any{"name": uniqueName("e2e-parity-role"), "permissions": perms})
	return roleScopedClient(t, jsonField(role, "id"))
}

// otherTenantSalesOrderID is a sales order some other account owns.
func otherTenantSalesOrderID(t *testing.T) string {
	t.Helper()
	var id string
	require.NoError(t, authDB(t).QueryRow(`SELECT id FROM sales_order WHERE owner_account_id <> ? AND sales_order_type_code = 'sales_order' LIMIT 1`, SeedAccountID).Scan(&id))
	return id
}

// parityProductLine creates a product line counted in pairs, the socks' unit group.
func parityProductLine(t *testing.T) string {
	t.Helper()
	created := createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-parity-pdln"),
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	})
	return jsonField(created, "id")
}

// parityProduct creates a sale product (a sock, counted in pairs) on productLineID and returns it and its item.
func parityProduct(t *testing.T, productLineID string) (productID, itemID string) {
	t.Helper()
	sku := uniqueName("e2e-parity-sku")
	body := map[string]any{"sku": sku, "type": "sale", "category_id": SeedItemCategoryID}
	if productLineID != "" {
		body["product_line_id"] = productLineID
	}
	status, raw, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	productID = jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productsPath + "/" + productID) })
	list, _, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	return productID, DataItemField(list.Data[0], "id")
}
