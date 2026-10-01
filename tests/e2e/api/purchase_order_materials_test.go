//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Purchase orders as the dashboard places them: materials ordered by item rather than product, the
// supplier's saved addresses, a number the buyer picked, and submission contacts on the supplier.

func materialLineBody(sku string) map[string]any {
	return map[string]any{
		"item_id":     SeedMaterialItemID,
		"product_sku": sku,
		"quantity":    map[string]any{"value": "40", "unit_id": SeedMaterialUnitID},
		"unit_price": map[string]any{
			"value":               "6",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedMaterialUnitID,
		},
	}
}

func retrievePurchaseOrder(t *testing.T, orderID string, includes ...string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+orderID, url.Values{"include": includes})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func TestPurchaseOrders_CreateAcceptsAMaterialLineWithoutAProduct(t *testing.T) {
	t.Parallel()

	created := createPurchaseOrder(t, func(b map[string]any) {
		b["lines"] = []map[string]any{materialLineBody("E2E-PO-MAT")}
	})

	po := retrievePurchaseOrder(t, jsonField(created, "id"), "lines", "lines.item", "lines.quantity_ordered.unit")
	lines := jsonListData(po, "lines")
	require.Len(t, lines, 1)
	line := lines[0].(map[string]any)
	assert.Equal(t, SeedMaterialItemID, jsonField(jsonObject(line, "item"), "id"), "the line restocks the material's item")
	assertDecimalEqual(t, "40", jsonField(jsonObject(line, "quantity_ordered"), "value"), "ordered in the item's own unit")
	assert.Equal(t, SeedMaterialUnitID, jsonField(jsonObject(jsonObject(line, "quantity_ordered"), "unit"), "id"))
}

func TestPurchaseOrders_CreateRejectsAMaterialLineInAUnitTheItemIsNotMeasuredIn(t *testing.T) {
	t.Parallel()

	line := materialLineBody("E2E-PO-MATUNIT")
	line["quantity"] = map[string]any{"value": "40", "unit_id": e2eCurrencyUnitID}
	body := validPurchaseOrderBody()
	body["lines"] = []map[string]any{line}

	status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "quantity_unit_id")
}

func TestPurchaseOrders_CreateUsesTheChosenNumberAndTheSuppliersSavedAddresses(t *testing.T) {
	t.Parallel()

	number := uniqueName("E2E-PO-NUM")
	created := createPurchaseOrder(t, func(b map[string]any) {
		b["number"] = number
		b["bill_to_address_id"] = SeedSupplierAddressID
		b["ship_to_address_id"] = SeedSupplierAddressID
	})
	assert.Equal(t, number, jsonField(created, "number"), "a chosen number is used as given")

	po := retrievePurchaseOrder(t, jsonField(created, "id"), "bill_to_address", "ship_to_address")
	assert.Equal(t, SeedSupplierAddressID, jsonField(jsonObject(po, "bill_to_address"), "id"), "the saved address is used, not copied")
	assert.Equal(t, SeedSupplierAddressID, jsonField(jsonObject(po, "ship_to_address"), "id"))
}

func TestPurchaseOrders_CreateRejectsANumberAlreadyInUse(t *testing.T) {
	t.Parallel()

	number := uniqueName("E2E-PO-DUP")
	createPurchaseOrder(t, func(b map[string]any) { b["number"] = number })

	body := validPurchaseOrderBody()
	body["number"] = number
	status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	requireStatus(t, 409, status, respBody)
}

func TestPurchaseOrders_CreateRejectsAnAddressTheSupplierDoesNotHave(t *testing.T) {
	t.Parallel()

	body := validPurchaseOrderBody()
	body["ship_to_address_id"] = SeedAddressID
	status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "ship_to_address_id")
}

// The dashboard sends the promised date as the end of the picked day in the user's zone, so a
// timestamp is kept to the millisecond rather than cut to a UTC date.
func TestPurchaseOrders_PromisedDateIsKeptAndCanBeCleared(t *testing.T) {
	t.Parallel()

	created := createPurchaseOrder(t, func(b map[string]any) { b["promised_at"] = "2026-11-20T04:59:59.999Z" })
	orderID := jsonField(created, "id")
	assert.Equal(t, "2026-11-20T04:59:59.999Z", jsonField(retrievePurchaseOrder(t, orderID), "scheduled_at"))

	status, body, err := apiClient.Patch(purchaseOrdersPath+"/"+orderID, map[string]any{"promised_at": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, retrievePurchaseOrder(t, orderID)["scheduled_at"], "null clears the promised date")
}

func TestPurchaseOrders_ContactsCarryTheSupplierUsersNameAndEmailAndCanBeCleared(t *testing.T) {
	t.Parallel()

	created := createPurchaseOrder(t, func(b map[string]any) {
		b["contact_account_user_ids"] = []string{SeedSupplierAccountUserID}
	})
	orderID := jsonField(created, "id")

	contacts := jsonListData(retrievePurchaseOrder(t, orderID, "contacts"), "contacts")
	require.Len(t, contacts, 1, "the supplier's user is the order's submission contact")
	contact := contacts[0].(map[string]any)
	assert.Equal(t, "Yarn Supply Orders", jsonField(contact, "name"))
	assert.Equal(t, "orders@yarnsupply.e2e.openmrp.ai", jsonField(contact, "email"))
	assert.Equal(t, SeedSupplierAccountUserID, jsonField(jsonObject(contact, "account_user"), "id"))

	status, body, err := apiClient.Patch(purchaseOrdersPath+"/"+orderID, map[string]any{"contact_account_user_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Empty(t, jsonListData(retrievePurchaseOrder(t, orderID, "contacts"), "contacts"), "an empty list clears the contacts")
}

// Receiving records what arrives against each line, so issuing opens every receiving line at zero
// and a line added afterwards gets one too.
func TestPurchaseOrders_IssueOpensReceivingLinesAtZeroAndALaterLineGetsOne(t *testing.T) {
	t.Parallel()

	purchaseOrderID, receivingOrderID := issuedPurchaseOrderReceiving(t)
	for _, l := range receivingOrderLines(t, receivingOrderID) {
		assertDecimalEqual(t, "0", jsonField(jsonObject(l.(map[string]any), "quantity"), "value"), "nothing has been received yet")
	}

	status, body, err := apiClient.Post(purchaseOrdersPath+"/"+purchaseOrderID+"/lines", materialLineBody("E2E-PO-LATE"), newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "adding a line to an issued order must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	lines := receivingOrderLines(t, receivingOrderID)
	assert.Len(t, lines, 2, "the added line can be received against")
	for _, l := range lines {
		assertDecimalEqual(t, "0", jsonField(jsonObject(l.(map[string]any), "quantity"), "value"))
	}
}

func TestPurchaseOrders_IssueWithEmailIsMarkedSentOnlyWhenThereIsSomeoneToSendTo(t *testing.T) {
	t.Parallel()

	issue := func(contacts []string) map[string]any {
		created := createPurchaseOrder(t, func(b map[string]any) { b["contact_account_user_ids"] = contacts })
		orderID := jsonField(created, "id")
		status, body, err := apiClient.Put(purchaseOrdersPath+"/"+orderID+"/actions/change-status", map[string]any{
			"status_change": "issue",
			"send_email":    true,
		})
		require.NoError(t, err)
		require.Less(t, status, 500, "issuing with an email must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)
		t.Cleanup(func() { _, _ = changePurchaseOrderStatus(t, orderID, "unissue") })
		return parseJSON(body)
	}

	assert.Equal(t, "sent", jsonField(issue([]string{SeedSupplierAccountUserID}), "acknowledgment_status"))
	assert.Equal(t, "not_sent", jsonField(issue(nil), "acknowledgment_status"), "an order with no contacts sends nothing")
}

func TestPurchaseOrders_ListRowsCarryWhatTheListShows(t *testing.T) {
	t.Parallel()

	number := uniqueName("E2E-PO-LIST")
	created := createPurchaseOrder(t, func(b map[string]any) {
		b["number"] = number
		b["note"] = "list note"
		b["ship_to_address_id"] = SeedSupplierAddressID
		b["promised_at"] = "2026-12-01"
		b["contact_account_user_ids"] = []string{SeedSupplierAccountUserID}
		b["lines"] = []map[string]any{materialLineBody("E2E-PO-LISTLINE")}
	})
	orderID := jsonField(created, "id")
	status, body := changePurchaseOrderStatus(t, orderID, "issue")
	requireStatus(t, 200, status, body)
	t.Cleanup(func() { _, _ = changePurchaseOrderStatus(t, orderID, "unissue") })

	listStatus, listBody, err := apiClient.GetListRaw(purchaseOrdersPath, url.Values{
		"q":       {number},
		"include": {"ship_to_address", "related.receiving_order", "contacts", "lines", "lines.delivery_lines"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)
	rows, _ := parseJSON(listBody)["data"].([]any)
	require.Len(t, rows, 1, "the search finds the order by its number")
	row := rows[0].(map[string]any)

	assert.Equal(t, "list note", jsonField(row, "note"))
	assert.Equal(t, "2026-12-01T00:00:00Z", jsonField(row, "scheduled_at"))
	assert.Equal(t, SeedSupplierAddressID, jsonField(jsonObject(row, "ship_to_address"), "id"))
	assert.NotEmpty(t, jsonField(jsonObject(jsonObject(row, "related"), "receiving_order"), "id"), "an issued order has its receiving order")
	assert.Len(t, jsonListData(row, "contacts"), 1)
	lines := jsonListData(row, "lines")
	require.Len(t, lines, 1)
	assert.NotNil(t, lines[0].(map[string]any)["delivery_lines"], "included delivery lines are a list even when none are booked")
	assert.Empty(t, jsonListData(lines[0].(map[string]any), "delivery_lines"))
}

func TestPurchaseOrders_RetrieveIncludesTheCreator(t *testing.T) {
	t.Parallel()

	created := createPurchaseOrder(t, nil)
	po := retrievePurchaseOrder(t, jsonField(created, "id"), "created_by")
	require.NotNil(t, po["created_by"], "the creator resolves from the order's create event")
}
