//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addressOn creates an address on accountID, targeting it as the dashboard's customer page does.
func addressOn(t *testing.T, client *Client, accountID, name string) string {
	t.Helper()
	status, body, err := client.WithAccountID(accountID).Post(addressesPath, map[string]any{
		"name":    name,
		"country": "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	return jsonField(parseJSON(body), "id")
}

// A customer's default addresses are chosen from its own. Any other id used to be linked to the
// customer, so another account's address could be attached and then read through the includes.
func TestCustomers_DefaultAddressMustBeTheCustomers(t *testing.T) {
	t.Parallel()
	customerID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-defaults"))), "id")
	own := addressOn(t, apiClient, customerID, uniqueName("e2e-own-dock"))
	other := jsonField(createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-cust-other"))), "id")
	elsewhere := addressOn(t, apiClient, other, uniqueName("e2e-other-dock"))
	path := customersPath + "/" + customerID

	status, body, err := apiClient.Patch(path, map[string]any{"bill_to_address_id": own, "ship_to_address_id": own}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, param := range []string{"bill_to_address_id", "ship_to_address_id"} {
		status, body, err = apiClient.Patch(path, map[string]any{param: elsewhere}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status, "another customer's address: %s", body)
		assert.Equal(t, param, errorParam(body))
	}

	status, body, err = apiClient.GetListRaw(path, map[string][]string{"include[]": {"bill_to_address", "ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, own, jsonField(jsonObject(got, "bill_to_address"), "id"), "the refused change left the default")
	assert.Equal(t, own, jsonField(jsonObject(got, "ship_to_address"), "id"))

	var linked int
	require.NoError(t, authDB(t).QueryRow(
		"SELECT COUNT(*) FROM account_address WHERE account_id = ? AND address_id = ?", customerID, elsewhere).Scan(&linked))
	assert.Zero(t, linked, "the other address was not linked to this customer")

	// Re-saving an unchanged default is fine, whatever else changes.
	status, body, err = apiClient.Patch(path, map[string]any{"bill_to_address_id": own, "note": "e2e note"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// A rename is the seller's name for its customer; the customer's own account, which other sellers may
// also know it by, keeps its name.
func TestCustomers_RenameIsTheSellersName(t *testing.T) {
	t.Parallel()
	original := uniqueName("e2e-cust-rename")
	customerID := jsonField(createAndCleanup(t, customersPath, validCustomerBody(original)), "id")
	renamed := original + " renamed"

	status, body, err := apiClient.Patch(customersPath+"/"+customerID, map[string]any{"name": renamed}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, renamed, jsonField(parseJSON(body), "name"))

	status, body, err = apiClient.GetListRaw(customersPath+"/"+customerID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, renamed, jsonField(parseJSON(body), "name"), "the customer reads by the new name")

	var accountName, alias string
	require.NoError(t, authDB(t).QueryRow(`
		SELECT a.name, ar.alias FROM account_relation ar JOIN account a ON a.id = ar.counterparty_account_id
		WHERE ar.owner_account_id = ? AND ar.counterparty_account_id = ?`, SeedAccountID, customerID).Scan(&accountName, &alias))
	assert.Equal(t, renamed, alias)
	assert.Equal(t, original, accountName, "the customer's own account keeps its name")
}
