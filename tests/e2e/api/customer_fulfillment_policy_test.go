//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A customer can be marked make-to-order. The policy rides on `defaults` and, once set,
// keeps that customer's history out of the production-schedule forecast — their orders are
// built when placed instead. This exercises the round-trip; the forecast exclusion itself
// lives in the demand query.

func customerDefaultsPolicy(m map[string]any) string {
	defaults := jsonObject(m, "defaults")
	if defaults == nil {
		return ""
	}
	return jsonField(defaults, "fulfillment_policy")
}

func TestFulfillmentPolicy_CustomerRoundTrips(t *testing.T) {
	t.Parallel()

	body := validCustomerBody(uniqueName("e2e-cust-mto"))
	body["fulfillment_policy"] = "make_to_order"
	created := createAndCleanup(t, customersPath+"?include=defaults", body)
	id := jsonField(created, "id")
	assert.Equal(t, "make_to_order", customerDefaultsPolicy(created),
		"the policy set on create should come back on defaults")

	// It has to persist, not only echo from the write.
	getStatus, getBody, err := apiClient.GetListRaw(customersPath+"/"+id, url.Values{"include": {"defaults"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "make_to_order", customerDefaultsPolicy(parseJSON(getBody)))

	// Switching to make-to-stock puts the customer's demand back into the forecast.
	patchStatus, patchBody, err := apiClient.Patch(customersPath+"/"+id+"?include=defaults", map[string]any{
		"fulfillment_policy": "make_to_stock",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "make_to_stock", customerDefaultsPolicy(parseJSON(patchBody)))

	// An update that omits the field must not wipe it: unset means "leave alone", not "clear".
	preserveStatus, preserveBody, err := apiClient.Patch(customersPath+"/"+id+"?include=defaults", map[string]any{
		"note": "unrelated edit",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, preserveStatus, preserveBody)
	assert.Equal(t, "make_to_stock", customerDefaultsPolicy(parseJSON(preserveBody)),
		"an update that omits fulfillment_policy must preserve it")

	// Clearing it returns the customer to inheriting its group, then the default.
	clearStatus, clearBody, err := apiClient.Patch(customersPath+"/"+id+"?include=defaults", map[string]any{
		"fulfillment_policy": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Empty(t, customerDefaultsPolicy(parseJSON(clearBody)),
		"clearing the policy should leave the customer inheriting, not pin it")
}

func TestFulfillmentPolicy_CustomerRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	body := validCustomerBody(uniqueName("e2e-cust-mto-bad"))
	body["fulfillment_policy"] = "make_to_whatever"
	status, respBody, err := apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a bad policy must be a client error: %s", string(respBody))
	assert.Equal(t, 400, status, "body: %s", string(respBody))
}
