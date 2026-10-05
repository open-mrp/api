//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func accountDefaultBillingAddressID(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(accountsPath+"/"+SeedAccountID, url.Values{"include": {"default_billing_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return jsonField(jsonObject(parseJSON(body), "default_billing_address"), "id")
}

// The account settings page saves the billing address and makes it the account's default. Not
// parallel: it changes the seeded account's default, which it puts back.
func TestAccounts_SetDefaultBillingAddress(t *testing.T) {
	original := accountDefaultBillingAddressID(t)
	addressID := createE2EAddress(t, uniqueName("e2e-account-billing"))
	t.Cleanup(func() {
		if original != "" {
			apiClient.Patch(accountsPath+"/"+SeedAccountID, map[string]any{"default_billing_address_id": original}, newIdempotencyKey())
		}
	})

	status, body, err := apiClient.Patch(accountsPath+"/"+SeedAccountID+"?include=default_billing_address",
		map[string]any{"default_billing_address_id": addressID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	billing := jsonObject(parseJSON(body), "default_billing_address")
	require.NotNil(t, billing, "the update returns the new default: %s", body)
	assert.Equal(t, addressID, jsonField(billing, "id"))
	assert.Equal(t, "123 Test St", jsonField(jsonObject(billing, "geolocation"), "street_line_1"))
	assert.Equal(t, addressID, accountDefaultBillingAddressID(t))

	status, body, err = apiClient.Patch(accountsPath+"/"+SeedAccountID,
		map[string]any{"default_billing_address_id": "addr_notoneofours0000"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "validation_failed", "invalid_request_error"), "default_billing_address_id")
	assert.Equal(t, addressID, accountDefaultBillingAddressID(t), "a refused address leaves the default alone")
}
