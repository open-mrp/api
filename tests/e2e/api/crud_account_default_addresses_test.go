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

// A branding field set to null is removed; one left out is kept. Not parallel: it edits the seeded
// account's branding, which it puts back.
func TestAccounts_BrandingFieldsClearWithNull(t *testing.T) {
	path := accountsPath + "/" + SeedAccountID
	read := func() map[string]any {
		t.Helper()
		status, body, err := apiClient.GetListRaw(path, url.Values{"include": {"branding"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		return jsonObject(parseJSON(body), "branding")
	}
	original := read()
	t.Cleanup(func() {
		restore := map[string]any{}
		for _, f := range []string{"phone_number", "website_url", "instagram_handle"} {
			restore[f] = original[f]
		}
		apiClient.Patch(path, restore, newIdempotencyKey())
	})

	status, body, err := apiClient.Patch(path, map[string]any{
		"phone_number": "555-0100", "website_url": "https://example.com", "instagram_handle": "e2e",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Patch(path, map[string]any{"phone_number": nil, "website_url": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	branding := read()
	assert.Nil(t, branding["phone_number"], "null removes the phone number")
	assert.Nil(t, branding["website_url"], "null removes the website")
	assert.Equal(t, "e2e", jsonField(branding, "instagram_handle"), "a field left out is kept")
}
