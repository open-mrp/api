//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Delete behavior tests verify idempotent deletion, soft-delete exclusion
// from list results, and dependent resource deletion constraints.

// ──────────────────────────────────────────────
// Double-delete idempotency
// ──────────────────────────────────────────────

func TestDeleteBehavior_DoubleDeleteCustomer(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-dbldelete")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	// First delete.
	del1Status, del1Body, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, del1Status, del1Body)

	// Second delete — should be 200 (idempotent), 404 (not found), or 410 (gone), never 500.
	del2Status, del2Body, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, del2Status == 200 || del2Status == 404 || del2Status == 410,
		"Double-delete should return 200, 404, or 410, got %d: %s", del2Status, string(del2Body))
}

func TestDeleteBehavior_DoubleDeleteAccountGroup(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post("/v1/sales/account-groups", map[string]any{
		"name": uniqueName("e2e-dbldelete-grp"),
		"type": "type_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	del1Status, del1Body, err := apiClient.Delete("/v1/sales/account-groups/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, del1Status, del1Body)

	del2Status, del2Body, err := apiClient.Delete("/v1/sales/account-groups/" + id)
	require.NoError(t, err)
	assert.True(t, del2Status == 200 || del2Status == 404 || del2Status == 410,
		"Double-delete should return 200, 404, or 410, got %d: %s", del2Status, string(del2Body))
}

func TestDeleteBehavior_DoubleDeleteAPIKey(t *testing.T) {
	t.Parallel()

	createStatus, createBody, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-dbldelete-key"),
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(jsonObject(parseJSON(createBody), "api_key_info"), "id")

	// First revoke.
	del1Status, del1Body, err := apiClient.Delete(apiKeysPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, del1Status, del1Body)

	// Second revoke — should be idempotent.
	del2Status, del2Body, err := apiClient.Delete(apiKeysPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, del2Status == 200 || del2Status == 404 || del2Status == 410,
		"Double-revoke of API key should return 200, 404, or 410, got %d: %s", del2Status, string(del2Body))
}

func TestDeleteBehavior_CustomerDeleteConflictWhenSalesOrdersExist(t *testing.T) {
	t.Parallel()

	const productLineAccessPath = "/v1/sales/product-line-access/customers"

	name := uniqueName("e2e-cust-so-guard")
	status, body, err := apiClient.Post(customersPath, validCustomerBody(name), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	customerID := jsonField(parseJSON(body), "id")

	plStatus, plBody, err := apiClient.Post(productLineAccessPath, map[string]any{
		"customer_id":      customerID,
		"product_line_ids": []string{SeedProductLineID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, plStatus, plBody)

	orderPayload := minimalSalesOrderCreateBody(customerID)
	orderStatus, orderBody, err := apiClient.Post(salesOrdersPath, orderPayload, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, orderStatus, orderBody)
	orderID := jsonField(parseJSON(orderBody), "id")

	conflictStatus, conflictBody, err := apiClient.Delete(customersPath + "/" + customerID)
	require.NoError(t, err)
	requireStatus(t, 409, conflictStatus, conflictBody)

	delOrderStatus, delOrderBody, err := apiClient.Delete(salesOrdersPath + "/" + orderID)
	require.NoError(t, err)
	requireStatus(t, 200, delOrderStatus, delOrderBody)

	delPLStatus, delPLBody, err := apiClient.Delete(productLineAccessPath + "/" + customerID)
	require.NoError(t, err)
	requireStatus(t, 200, delPLStatus, delPLBody)

	delCustStatus, delCustBody, err := apiClient.Delete(customersPath + "/" + customerID)
	require.NoError(t, err)
	requireStatus(t, 200, delCustStatus, delCustBody)
}

// ──────────────────────────────────────────────
// Soft-delete list exclusion
// ──────────────────────────────────────────────

func TestDeleteBehavior_DeletedCustomerNotInList(t *testing.T) {
	t.Parallel()

	distinctName := uniqueName("e2e-softdel-list")
	status, body, err := apiClient.Post(customersPath, validCustomerBody(distinctName), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	// Delete it.
	delStatus, delBody, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Search for it — should not appear in list.
	list, _, err := apiClient.GetList(customersPath, url.Values{"q": {distinctName}})
	require.NoError(t, err)

	for _, item := range list.Data {
		assert.NotEqual(t, id, DataItemField(item, "id"),
			"Deleted customer should not appear in list results")
	}
}

func TestDeleteBehavior_DeletedAccountGroupNotInList(t *testing.T) {
	t.Parallel()

	distinctName := uniqueName("e2e-softdel-grp")
	status, body, err := apiClient.Post("/v1/sales/account-groups", map[string]any{
		"name": distinctName,
		"type": "type_group",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete("/v1/sales/account-groups/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	list, _, err := apiClient.GetList("/v1/sales/account-groups", url.Values{"q": {distinctName}})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.NotEqual(t, id, DataItemField(item, "id"),
			"Deleted account group should not appear in list results")
	}
}

// ──────────────────────────────────────────────
// Delete returns correct response shape
// ──────────────────────────────────────────────

func TestDeleteBehavior_DeleteReturnsCorrectShape(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(customersPath, validCustomerBody(uniqueName("e2e-delshape")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	delStatus, delBody, err := apiClient.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify the response is valid JSON.
	parsed := parseJSON(delBody)
	require.NotNil(t, parsed, "DELETE response should be valid JSON")
}

// ──────────────────────────────────────────────
// Delete non-existent resource
// ──────────────────────────────────────────────

func TestDeleteBehavior_DeleteNonExistent(t *testing.T) {
	t.Parallel()

	// Fabricate a non-existent customer ID.
	fakeID := "ac_000000000000000000000000"
	status, body, err := apiClient.Delete(customersPath + "/" + fakeID)
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"Deleting non-existent resource should return 404: %s", string(body))

	if status == 404 {
		requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
	}
}
