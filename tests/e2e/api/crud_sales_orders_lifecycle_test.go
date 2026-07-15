//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for the sales-order mutations the dashboard cut over to v2:
// the issue/unissue/close/open lifecycle, line create/update/delete, the delete
// guards, and the shipping-line re-price on ship-to / carrier change. These pin the
// legacy Dashboard semantics; per the project rule they fail loudly on 5xx.

// createLifecycleOrder creates a fresh estimate order (owned by the internal actor)
// against a customer with product-line access and registers cleanup. Returns the id.
func createLifecycleOrder(t *testing.T) string {
	t.Helper()
	customerID := setupOrderCustomer(t)
	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)
	return orderID
}

// salesOrderAction PUTs a lifecycle action (issue/unissue/close/open) with a fresh
// idempotency key so each call is a distinct transition. Only issue carries a
// notify_customer flag; the other transitions take an empty body.
func salesOrderAction(t *testing.T, orderID, action string, notifyCustomer bool) (int, []byte) {
	t.Helper()
	reqBody := map[string]any{}
	if action == "issue" {
		reqBody["notify_customer"] = notifyCustomer
	}
	status, body, err := apiClient.Do(
		http.MethodPut,
		salesOrdersPath+"/"+orderID+"/actions/"+action,
		reqBody,
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	return status, body
}

// getSalesOrder fetches an order and requires a 200.
func getSalesOrder(t *testing.T, orderID string, params url.Values) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// requireClientError asserts a 4xx (client) response — never a 5xx, which per the
// project rule must be treated as a backend bug, not a business rejection.
func requireClientError(t *testing.T, status int, body []byte) {
	t.Helper()
	require.GreaterOrEqual(t, status, 400, "expected a client error: %s", string(body))
	require.Less(t, status, 500, "a business rejection must not 5xx: %s", string(body))
}

func TestSalesOrder_Lifecycle_IssueUnissueCloseOpen(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// estimate -> issued: stamps issued_at and (side effect) creates a pick.
	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	issued := parseJSON(body)
	assert.Equal(t, "issued", jsonField(issued, "status"))
	assert.NotEmpty(t, jsonField(issued, "issued_at"), "issued_at is stamped on issue")

	// A pick is created as a side effect of issuing.
	withPick := getSalesOrder(t, orderID, url.Values{"include": {"related.pick"}})
	if related := jsonObject(withPick, "related"); related != nil {
		if pick := jsonObject(related, "pick"); pick != nil {
			assert.NotEmpty(t, jsonField(pick, "id"), "issuing creates a pick")
		}
	}

	// issued -> estimate: reverses the pick + reservations.
	status, body = salesOrderAction(t, orderID, "unissue", false)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "estimate", jsonField(parseJSON(body), "status"))

	// re-issue so we can close it.
	status, body = salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	// issued -> fulfilled: stamps completed_at.
	status, body = salesOrderAction(t, orderID, "close", false)
	requireStatus(t, 200, status, body)
	closed := parseJSON(body)
	assert.Equal(t, "fulfilled", jsonField(closed, "status"))
	assert.NotEmpty(t, jsonField(closed, "completed_at"), "completed_at is stamped on close")

	// fulfilled -> issued: clears completed_at, preserves issued_at.
	status, body = salesOrderAction(t, orderID, "open", false)
	requireStatus(t, 200, status, body)
	reopened := parseJSON(body)
	assert.Equal(t, "issued", jsonField(reopened, "status"))
	assert.Empty(t, jsonField(reopened, "completed_at"), "completed_at is cleared on open")
	assert.NotEmpty(t, jsonField(reopened, "issued_at"), "issued_at is preserved on open")
}

func TestSalesOrder_Issue_PickLinesStartUnpicked(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	status, body := salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)

	// Issuing creates a pick. Every pick line must start at 0 picked — a pick line's
	// quantity is the amount picked so far, filled in by the pick action, NOT the ordered
	// quantity (which would make the order read as fully picked the instant it is issued).
	withPick := getSalesOrder(t, orderID, url.Values{"include": {"related.pick"}})
	related := jsonObject(withPick, "related")
	require.NotNil(t, related, "related is present")
	pick := jsonObject(related, "pick")
	require.NotNil(t, pick, "issuing creates a pick")
	pickID := jsonField(pick, "id")
	require.NotEmpty(t, pickID)

	pStatus, pBody, err := apiClient.GetListRaw("/v1/operations/picks/"+pickID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, pStatus, pBody)
	lines := jsonObject(parseJSON(pBody), "lines")
	require.NotNil(t, lines, "pick lines are present with ?include=lines")
	data := jsonArray(lines, "data")
	require.NotEmpty(t, data, "the pick has at least one line")

	for _, raw := range data {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		qty := jsonObject(line, "quantity")
		require.NotNil(t, qty, "pick line carries a picked quantity")
		picked, _ := strconv.ParseFloat(jsonField(qty, "value"), 64)
		assert.Equal(t, 0.0, picked, "pick lines start at 0 picked, not the ordered quantity")

		if ordered := jsonObject(line, "ordered_quantity"); ordered != nil {
			orderedVal, _ := strconv.ParseFloat(jsonField(ordered, "value"), 64)
			assert.Greater(t, orderedVal, 0.0, "ordered_quantity reflects the sales-order line quantity")
		}
	}
}

func TestSalesOrder_Update_PersistsPromisedAtAndCustomer(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// promised_at is forwarded on update (it used to be silently dropped by the gateway).
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"promised_at": "2026-12-24T00:00:00Z"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Contains(t, jsonField(getSalesOrder(t, orderID, nil), "promised_at"), "2026-12-24",
		"promised_at persists on update")

	// customer_id is forwarded on update (also previously dropped). Re-point to a new buyer.
	customerB := setupOrderCustomer(t)
	status, body, err = apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"customer_id": customerB}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	cust := jsonObject(getSalesOrder(t, orderID, url.Values{"include": {"customer"}}), "customer")
	require.NotNil(t, cust, "customer is present with ?include=customer")
	assert.Equal(t, customerB, jsonField(cust, "id"), "customer_id persists on update")
}

func TestSalesOrder_Update_RejectsNumberField(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// The order number is not updatable: `number` is not part of the update request, so
	// sending it is rejected as an unknown field rather than silently ignored.
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"number": "SHOULD-BE-REJECTED"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assert.Contains(t, []any{"parameter_unknown", "validation_failed"}, errObj["code"],
		"number must be rejected as an unknown update field: %s", string(body))
}

func TestSalesOrder_Lifecycle_RejectsOutOfOrderTransitions(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// close / open require an issued order; on a fresh estimate both are rejected.
	status, body := salesOrderAction(t, orderID, "close", false)
	requireClientError(t, status, body)
	status, body = salesOrderAction(t, orderID, "open", false)
	requireClientError(t, status, body)

	// Issue once succeeds; issuing an already-issued order is rejected (not a 5xx).
	status, body = salesOrderAction(t, orderID, "issue", false)
	requireStatus(t, 200, status, body)
	status, body = salesOrderAction(t, orderID, "issue", false)
	requireClientError(t, status, body)

	// Restore to estimate so cleanup can delete it.
	status, body = salesOrderAction(t, orderID, "unissue", false)
	requireStatus(t, 200, status, body)
}

func TestSalesOrder_UpdateRepricesShippingLineOnShipToAndCarrierChange(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// One product line + one synthesized shipping line.
	assert.EqualValues(t, 2, getSalesOrder(t, orderID, nil)["line_count"])

	// Re-pointing the ship-to address re-estimates the shipping line. The full cascade
	// (address resolution, weight, exemptions, live-rate fallback, line update) must run
	// against real data without a 5xx, and must update the shipping line in place rather
	// than adding or dropping a line.
	newAddr := createE2EAddress(t, "E2E Ship-To Reprice")
	status, body, err := apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"shipping_address_id": newAddr}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.EqualValues(t, 2, getSalesOrder(t, orderID, nil)["line_count"],
		"re-pricing updates the shipping line in place, not adding/removing lines")

	// Changing the carrier also triggers a re-price.
	status, body, err = apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"carrier_id": SeedCarrierID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.EqualValues(t, 2, getSalesOrder(t, orderID, nil)["line_count"])

	// A scalar-only update (note) leaves the line structure untouched.
	status, body, err = apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"note": "no reprice for this one"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.EqualValues(t, 2, getSalesOrder(t, orderID, nil)["line_count"])
}

func TestSalesOrder_Checkout_NoStripeIntegrationRejected(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)

	// The seed account has no active Stripe integration, so checkout is rejected with a
	// client error (never a 5xx).
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/checkout",
		map[string]any{"email": "buyer@example.com"}, newIdempotencyKey())
	require.NoError(t, err)
	requireClientError(t, status, body)
}

func TestSalesOrder_Lines_CreateUpdateDelete(t *testing.T) {
	t.Parallel()
	orderID := createLifecycleOrder(t)
	assert.EqualValues(t, 2, getSalesOrder(t, orderID, nil)["line_count"])

	// Create an additional line.
	lineBody := map[string]any{
		"product_id":  SeedProductID,
		"product_sku": "E2E-LINE-SKU",
		"quantity":    map[string]any{"value": "3", "unit_id": SeedUnitID},
		"unit_price": map[string]any{
			"value":               "12.50",
			"numerator_unit_id":   e2eCurrencyUnitID,
			"denominator_unit_id": SeedUnitID,
		},
	}
	status, body, err := apiClient.Post(salesOrdersPath+"/"+orderID+"/lines", lineBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	lineID := jsonField(parseJSON(body), "id")
	assert.EqualValues(t, 3, getSalesOrder(t, orderID, nil)["line_count"], "creating a line increments line_count")

	// Update the line's quantity (v2 nested Quantity input shape).
	status, body, err = apiClient.Patch(salesOrdersPath+"/"+orderID+"/lines/"+lineID,
		map[string]any{"quantity": map[string]any{"value": "5", "unit_id": SeedUnitID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Delete the line.
	status, body, err = apiClient.Delete(salesOrdersPath + "/" + orderID + "/lines/" + lineID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, status, 200, "delete line: %s", string(body))
	require.Less(t, status, 300, "delete line: %s", string(body))
	assert.EqualValues(t, 2, getSalesOrder(t, orderID, nil)["line_count"], "deleting the line restores line_count")
}

// TestSalesOrder_Issue_NotifyCustomer_SendsAcknowledgement pins the acknowledgement-email
// side effect of issuing with notify_customer: true. The email is published to the outbox
// inside the same transaction that flips is_acknowledgment_sent, so a successful 200 with
// acknowledgment_status == "sent" proves the publish succeeded. This guards the regression
// where the outbox publisher couldn't read the RepoFactory from context (WithRepos was not
// injected), which 5xx'd the issue action and left acknowledgment_status stuck at "not_sent".
// Every other lifecycle test issues with notify_customer: false, so this path was uncovered.
func TestSalesOrder_Issue_NotifyCustomer_SendsAcknowledgement(t *testing.T) {
	t.Parallel()

	// Order email contacts must reference a user of the buyer's account, so use the
	// seeded customer (which owns SeedCustomerAccountUserID) as the buyer — a freshly
	// created customer has no account_users to name as a recipient.
	bill := createE2EAddress(t, "E2E AckEmail Bill")
	ship := createE2EAddress(t, "E2E AckEmail Ship")
	po := uniqueName("PO")
	body := covSalesSalesOrdersFullCreateBody(SeedCustomerAccountID, bill, ship, po, SeedCustomerAccountUserID)

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)
	assert.Equal(t, "not_sent", jsonField(getSalesOrder(t, orderID, nil), "acknowledgment_status"), "new order is not acknowledged")

	// Issue with notify_customer: true — must not 5xx, and must publish the ack email.
	status, respBody = salesOrderAction(t, orderID, "issue", true)
	requireStatus(t, 200, status, respBody)

	assert.Equal(t, "sent", jsonField(getSalesOrder(t, orderID, nil), "acknowledgment_status"),
		"issuing with notify_customer: true must send the acknowledgement email and set acknowledgment_status to sent")
}

func TestSalesOrder_Delete_EstimateAllowed_FulfilledBlocked(t *testing.T) {
	t.Parallel()

	// An estimate order deletes cleanly and is then gone.
	customerID := setupOrderCustomer(t)
	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	estimateID := jsonField(parseJSON(body), "id")

	dStatus, dBody, err := apiClient.Delete(salesOrdersPath + "/" + estimateID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, dStatus, 200, "delete estimate: %s", string(dBody))
	require.Less(t, dStatus, 300, "delete estimate: %s", string(dBody))

	gStatus, _, err := apiClient.GetListRaw(salesOrdersPath+"/"+estimateID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, gStatus, "deleted order is gone")

	// A fulfilled order cannot be deleted.
	fulfilledID := createLifecycleOrder(t)
	status, body = salesOrderAction(t, fulfilledID, "issue", false)
	requireStatus(t, 200, status, body)
	status, body = salesOrderAction(t, fulfilledID, "close", false)
	requireStatus(t, 200, status, body)

	fStatus, fBody, err := apiClient.Delete(salesOrdersPath + "/" + fulfilledID)
	require.NoError(t, err)
	requireClientError(t, fStatus, fBody)
}
