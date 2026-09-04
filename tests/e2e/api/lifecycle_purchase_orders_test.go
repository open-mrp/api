//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Purchase order lifecycle, lines, and the errors each path can produce.
//
// A purchase order is the mirror of a sales order: it is placed with a supplier, and
// issuing it creates the receiving order that stock is booked against. That coupling is
// why the status transitions matter more here than the field-level CRUD — issuing and
// unissuing create and destroy a second resource, and closing marks it complete.
//
// The existing crud_purchase_orders_test.go covers the include expansions only, so
// everything below — create, update, lines, transitions, deletion, and every rejection
// they can produce — was previously unexercised.

// purchaseOrderLineBody is one line's worth of a create payload. Quantities are in the
// seeded pair unit, which belongs to the seeded product's unit group; a unit from another
// group is rejected, and that rejection has its own case below.
func purchaseOrderLineBody(sku string) map[string]any {
	return map[string]any{
		"product_id":  SeedProductID,
		"product_sku": sku,
		"quantity":    map[string]any{"value": "4", "unit_id": SeedUnitID},
		"unit_price": map[string]any{
			"value":               "9.50",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}
}

func validPurchaseOrderBody() map[string]any {
	return map[string]any{
		"supplier_account_id": SeedSupplierAccountID,
		"priority_code":       SeedPriorityCode,
		"lines":               []map[string]any{purchaseOrderLineBody("E2E-PO-SKU")},
		// Names are supplied on both addresses on purpose. Create builds a bill-to and a ship-to
		// unconditionally, and with the name omitted it stores an empty one — which the address
		// resource declares required, so an unnamed address left lying around fails the shared
		// list-addresses contract for every other test in the suite.
		"bill_to_name":    "E2E PO bill-to",
		"bill_to_country": "US",
		"ship_to_name":    "E2E PO ship-to",
		"ship_to_country": "US",
	}
}

// createPurchaseOrder makes an estimate and cleans it up. Deletion is only permitted
// while the order is an estimate, so a test that issues one unissues it first.
func createPurchaseOrder(t *testing.T, mutate func(map[string]any)) map[string]any {
	t.Helper()

	body := validPurchaseOrderBody()
	if mutate != nil {
		mutate(body)
	}

	status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "purchase order create must not 5xx: %s", string(respBody))
	requireStatus(t, 201, status, respBody)

	created := parseJSON(respBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(purchaseOrdersPath + "/" + id) })
	return created
}

func changePurchaseOrderStatus(t *testing.T, orderID, change string) (int, []byte) {
	t.Helper()
	status, body, err := apiClient.Put(purchaseOrdersPath+"/"+orderID+"/actions/change-status", map[string]any{
		"status_change": change,
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "status change %q must not 5xx: %s", change, string(body))
	return status, body
}

func TestPurchaseOrders_CreateReturnsAnEstimateWithItsLines(t *testing.T) {
	t.Parallel()

	created := createPurchaseOrder(t, nil)

	assert.Equal(t, "purchase_order", jsonField(created, "object"))
	assert.Equal(t, "estimate", jsonField(created, "status"), "a new purchase order starts as an estimate: %v", created)
	assert.NotEmpty(t, jsonField(created, "number"), "an order must be numbered on create")

	resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+jsonField(created, "id")+"?include=lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Len(t, jsonListData(parseJSON(resp.Body), "lines"), 1, "the line supplied at create must be persisted")
}

// ──────────────────────────────────────────────
// Create — rejections
// ──────────────────────────────────────────────

func TestPurchaseOrders_CreateRejectsAMissingSupplier(t *testing.T) {
	t.Parallel()

	body := validPurchaseOrderBody()
	delete(body, "supplier_account_id")

	status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "supplier_account_id")
}

func TestPurchaseOrders_CreateRejectsAnUnknownSupplier(t *testing.T) {
	t.Parallel()

	status, respBody, err := apiClient.Post(purchaseOrdersPath, map[string]any{
		"supplier_account_id": "ac_doesnotexist00000",
		"priority_code":       SeedPriorityCode,
		"lines":               []map[string]any{purchaseOrderLineBody("E2E-PO-NOSUP")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	assert.Contains(t, []int{400, 404}, status, "an unknown supplier must be refused: %s", string(respBody))
}

// A customer is an account too, so the supplier field has to check the relation and not merely that the ID names something.
func TestPurchaseOrders_CreateRejectsANonSupplierAccount(t *testing.T) {
	t.Parallel()

	status, respBody, err := apiClient.Post(purchaseOrdersPath, map[string]any{
		"supplier_account_id": SeedCustomerAccountID,
		"priority_code":       SeedPriorityCode,
		"lines":               []map[string]any{purchaseOrderLineBody("E2E-PO-CUST")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	assert.Contains(t, []int{400, 404}, status, "a customer must not be usable as a supplier: %s", string(respBody))
}

func TestPurchaseOrders_CreateRejectsAnUnknownPriority(t *testing.T) {
	t.Parallel()

	body := validPurchaseOrderBody()
	body["priority_code"] = "zzz-not-a-priority"

	status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
	assert.Contains(t, []int{400, 404}, status, "an unknown priority must be refused: %s", string(respBody))
}

// The quantity unit has to be one the product is measured in, on both write paths.
//
// Without the check an order records "1 dollar" of a product sold in pairs, and because issuing copies the lines onto the receiving order, that nonsense quantity is what stock gets booked against. The sales-order path has always rejected this; purchase orders now answer the same way.
func TestPurchaseOrderLines_RejectAQuantityUnitOutsideTheProductsGroup(t *testing.T) {
	t.Parallel()

	badUnitLine := func(sku string) map[string]any {
		line := purchaseOrderLineBody(sku)
		line["quantity"] = map[string]any{"value": "1", "unit_id": e2eCurrencyUnitID}
		return line
	}

	t.Run("on create", func(t *testing.T) {
		body := validPurchaseOrderBody()
		body["lines"] = []map[string]any{badUnitLine("E2E-PO-BADUNIT")}

		status, respBody, err := apiClient.Post(purchaseOrdersPath, body, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
		requireStatus(t, 400, status, respBody)
		errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
		assertErrorParam(t, errObj, "quantity_unit_id")
	})

	// The second write path: a line added to an order that already exists.
	t.Run("on add-line", func(t *testing.T) {
		orderID := jsonField(createPurchaseOrder(t, nil), "id")

		status, respBody, err := apiClient.Post(purchaseOrdersPath+"/"+orderID+"/lines",
			badUnitLine("E2E-PO-BADUNIT-2"), newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must reject rather than 5xx: %s", string(respBody))
		requireStatus(t, 400, status, respBody)
		errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
		assertErrorParam(t, errObj, "quantity_unit_id")
	})
}

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

func TestPurchaseOrders_UpdateEditsTheOrder(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	status, body, err := apiClient.Patch(purchaseOrdersPath+"/"+orderID, map[string]any{
		"note": "E2E updated note",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "update must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Equal(t, "E2E updated note", jsonField(parseJSON(body), "note"))
}

func TestPurchaseOrders_UpdateUnknownOrderIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Patch(purchaseOrdersPath+"/or_doesnotexist00000", map[string]any{
		"note": "nobody home",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// Lifecycle
// ──────────────────────────────────────────────

// Issuing is what creates the receiving order, which is the whole reason a purchase order exists in the system rather than as a PDF.
func TestPurchaseOrders_IssueCreatesTheReceivingOrder(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	status, body := changePurchaseOrderStatus(t, orderID, "issue")
	requireStatus(t, 200, status, body)
	assert.Equal(t, "issued", jsonField(parseJSON(body), "status"))

	resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+orderID+"?include=related&include=related.receiving_order", nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	receiving := jsonObject(jsonObject(parseJSON(resp.Body), "related"), "receiving_order")
	require.NotNil(t, receiving, "issuing must create the receiving order: %s", string(resp.Body))
	assert.NotEmpty(t, jsonField(receiving, "id"))

	// Left as an estimate so the cleanup delete can take it.
	unissueStatus, unissueBody := changePurchaseOrderStatus(t, orderID, "unissue")
	requireStatus(t, 200, unissueStatus, unissueBody)
}

func TestPurchaseOrders_UnissueRemovesTheReceivingOrder(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	issueStatus, issueBody := changePurchaseOrderStatus(t, orderID, "issue")
	requireStatus(t, 200, issueStatus, issueBody)

	status, body := changePurchaseOrderStatus(t, orderID, "unissue")
	requireStatus(t, 200, status, body)
	assert.Equal(t, "estimate", jsonField(parseJSON(body), "status"))

	resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+orderID+"?include=related&include=related.receiving_order", nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Nil(t, jsonObject(jsonObject(parseJSON(resp.Body), "related"), "receiving_order"),
		"unissuing must take the receiving order with it: %s", string(resp.Body))
}

func TestPurchaseOrders_CloseAndReopen(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	issueStatus, issueBody := changePurchaseOrderStatus(t, orderID, "issue")
	requireStatus(t, 200, issueStatus, issueBody)

	closeStatus, closeBody := changePurchaseOrderStatus(t, orderID, "close")
	requireStatus(t, 200, closeStatus, closeBody)
	assert.Equal(t, "fulfilled", jsonField(parseJSON(closeBody), "status"))

	openStatus, openBody := changePurchaseOrderStatus(t, orderID, "open")
	requireStatus(t, 200, openStatus, openBody)
	assert.Equal(t, "issued", jsonField(parseJSON(openBody), "status"))

	unissueStatus, unissueBody := changePurchaseOrderStatus(t, orderID, "unissue")
	requireStatus(t, 200, unissueStatus, unissueBody)
}

// Each transition has a starting status it is valid from, and the order is left alone when it is not.
func TestPurchaseOrders_TransitionsRejectedFromTheWrongStatus(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	for _, change := range []string{"unissue", "close", "open"} {
		status, body := changePurchaseOrderStatus(t, orderID, change)
		assert.Equal(t, 400, status, "%q must be refused on an estimate: %s", change, string(body))
	}

	resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	assert.Equal(t, "estimate", jsonField(parseJSON(resp.Body), "status"), "a refused transition must not move the order")
}

func TestPurchaseOrders_UnknownTransitionIsRejected(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	status, body := changePurchaseOrderStatus(t, orderID, "teleport")
	assert.Equal(t, 400, status, "an unknown transition must be refused: %s", string(body))
}

func TestPurchaseOrders_StatusChangeOnUnknownOrderIs404(t *testing.T) {
	t.Parallel()

	status, body := changePurchaseOrderStatus(t, "or_doesnotexist00000", "issue")
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// Lines
// ──────────────────────────────────────────────

func TestPurchaseOrderLines_CreateUpdateDelete(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")
	linesPath := purchaseOrdersPath + "/" + orderID + "/lines"

	status, body, err := apiClient.Post(linesPath, purchaseOrderLineBody("E2E-PO-LINE-2"), newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "line create must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	lineID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, lineID)

	patchStatus, patchBody, err := apiClient.Patch(linesPath+"/"+lineID, map[string]any{
		"product_description": "E2E updated line",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, patchStatus, 500, "line update must not 5xx: %s", string(patchBody))
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "E2E updated line", jsonField(parseJSON(patchBody), "product_description"))

	delStatus, delBody, err := apiClient.Delete(linesPath + "/" + lineID)
	require.NoError(t, err)
	require.Less(t, delStatus, 500, "line delete must not 5xx: %s", string(delBody))
	requireStatus(t, 200, delStatus, delBody)

	resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+orderID+"?include=lines", nil)
	require.NoError(t, err)
	assert.Len(t, jsonListData(parseJSON(resp.Body), "lines"), 1, "only the line created with the order should remain")
}

func TestPurchaseOrderLines_CreateOnUnknownOrderIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(purchaseOrdersPath+"/or_doesnotexist00000/lines",
		purchaseOrderLineBody("E2E-PO-ORPHAN"), newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	requireStatus(t, 404, status, body)
}

func TestPurchaseOrderLines_UnknownLineIs404(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")
	linesPath := purchaseOrdersPath + "/" + orderID + "/lines"

	patchStatus, patchBody, err := apiClient.Patch(linesPath+"/orln_doesnotexist00", map[string]any{
		"product_description": "nobody home",
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, patchStatus, 500, "must 404 rather than 5xx: %s", string(patchBody))
	requireStatus(t, 404, patchStatus, patchBody)

	delStatus, delBody, err := apiClient.Delete(linesPath + "/orln_doesnotexist00")
	require.NoError(t, err)
	require.Less(t, delStatus, 500, "must 404 rather than 5xx: %s", string(delBody))
	requireStatus(t, 404, delStatus, delBody)
}

func TestPurchaseOrderLines_CreateRejectsAMissingProduct(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")
	line := purchaseOrderLineBody("E2E-PO-NOPROD")
	delete(line, "product_id")

	status, body, err := apiClient.Post(purchaseOrdersPath+"/"+orderID+"/lines", line, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "must reject rather than 5xx: %s", string(body))
	requireStatus(t, 400, status, body)
	// Reported as the Go field name rather than the JSON one, which is what the embedded OrderLineInput produces today.
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "product_id")
}

// ──────────────────────────────────────────────
// Delete
// ──────────────────────────────────────────────

func TestPurchaseOrders_DeleteRemovesTheOrder(t *testing.T) {
	t.Parallel()

	orderID := jsonField(createPurchaseOrder(t, nil), "id")

	status, body, err := apiClient.Delete(purchaseOrdersPath + "/" + orderID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode, "a deleted order must be gone: %s", string(resp.Body))
}

func TestPurchaseOrders_DeleteUnknownOrderIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(purchaseOrdersPath + "/or_doesnotexist00000")
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	requireStatus(t, 404, status, body)
}

func TestPurchaseOrders_BulkDelete(t *testing.T) {
	t.Parallel()

	first := jsonField(createPurchaseOrder(t, nil), "id")
	second := jsonField(createPurchaseOrder(t, nil), "id")

	status, body, err := apiClient.Post(purchaseOrdersPath+"/actions/bulk-delete", map[string]any{
		"purchase_order_ids": []string{first, second},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "bulk delete must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	for _, id := range []string{first, second} {
		resp, err := apiClient.GetFull(purchaseOrdersPath+"/"+id, nil)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode, "order %s should be gone: %s", id, string(resp.Body))
	}
}

// ──────────────────────────────────────────────
// Statuses
// ──────────────────────────────────────────────

func TestPurchaseOrders_ListStatuses(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/statuses", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "statuses must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	// Every status a transition can produce has to be listed, or a client cannot render what it is told.
	for _, want := range []string{"estimate", "issued", "fulfilled"} {
		assert.Contains(t, string(body), want, "status %q should be listed", want)
	}
}
