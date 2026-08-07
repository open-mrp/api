//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The middle rung of the ship-by chain: the lead time a group hands to every
// customer in it that has not set its own.
//
// The resolution itself is covered in ship_by_commitment_test.go; what is covered
// here is the field that feeds it — that it stores, updates, clears, and refuses
// values that would date an order's ship-by before the order exists.

func TestAccountGroupLeadTime_RoundTripsAndClears(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-grp-lt", ptrInt(21))

	status, body, err := apiClient.GetListRaw(accountGroupsPath+"/"+groupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "21", jsonField(parseJSON(body), "default_lead_time_days"),
		"a lead time set at creation must be readable back")

	patchStatus, patchBody, err := apiClient.Patch(accountGroupsPath+"/"+groupID,
		map[string]any{"default_lead_time_days": 45}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, patchStatus, 500, "updating the lead time must not 5xx: %s", string(patchBody))
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "45", jsonField(parseJSON(patchBody), "default_lead_time_days"))

	// An unrelated edit must leave it alone: a rename that silently reset a contractual
	// lead time would be found only by the customer who missed a delivery.
	renameStatus, renameBody, err := apiClient.Patch(accountGroupsPath+"/"+groupID,
		map[string]any{"name": uniqueName("e2e-grp-lt-renamed")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, renameStatus, renameBody)
	assert.Equal(t, "45", jsonField(parseJSON(renameBody), "default_lead_time_days"))

	clearStatus, clearBody, err := apiClient.Patch(accountGroupsPath+"/"+groupID,
		map[string]any{"default_lead_time_days": nil}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, clearStatus, 500, "clearing the lead time must not 5xx: %s", string(clearBody))
	requireStatus(t, 200, clearStatus, clearBody)
	assert.Nil(t, parseJSON(clearBody)["default_lead_time_days"],
		"a cleared group lead time returns its customers to the account default")
}

// A group with no lead time is the normal case, and it must read as null rather
// than as zero — zero is same-day shipping, which is a very different promise.
func TestAccountGroupLeadTime_DefaultsToNull(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-grp-lt-none", nil)

	status, body, err := apiClient.GetListRaw(accountGroupsPath+"/"+groupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["default_lead_time_days"])
}

// Clearing a group's lead time has to reach the customers inheriting it, or the
// field would be editable without being meaningful.
func TestAccountGroupLeadTime_ClearingDropsCustomersToTheAccount(t *testing.T) {
	t.Parallel()

	groupID := leadTimeAccountGroup(t, "e2e-grp-lt-drop", ptrInt(21))
	customerID := leadTimeCustomer(t, "e2e-grp-lt-drop-cust", nil, groupID)

	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	require.Equal(t, "account_group", jsonField(parseJSON(body), "source"),
		"precondition: the customer is inheriting from the group")

	clearStatus, clearBody, err := apiClient.Patch(accountGroupsPath+"/"+groupID,
		map[string]any{"default_lead_time_days": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)

	afterStatus, afterBody, err := apiClient.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	requireStatus(t, 200, afterStatus, afterBody)
	assert.Equal(t, "account", jsonField(parseJSON(afterBody), "source"),
		"with the group's lead time gone the customer falls to the account default")
}

// Zero is a real commitment and must be storable; a negative one would date an
// order's ship-by before it was issued, and ten years is past any lead time a
// factory negotiates.
func TestLeadTimeDays_ValidatesItsRange(t *testing.T) {
	t.Parallel()

	t.Run("group accepts zero", func(t *testing.T) {
		groupID := leadTimeAccountGroup(t, "e2e-grp-lt-zero", ptrInt(0))

		status, body, err := apiClient.GetListRaw(accountGroupsPath+"/"+groupID, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Equal(t, "0", jsonField(parseJSON(body), "default_lead_time_days"),
			"zero is same-day shipping, not unset")
	})

	for _, tc := range []struct {
		name  string
		value int
	}{
		{"negative", -1},
		{"beyond ten years", 3651},
	} {
		t.Run("group create rejects "+tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(accountGroupsPath, map[string]any{
				"name":                   uniqueName("e2e-grp-lt-bad"),
				"type":                   "type_group",
				"default_lead_time_days": tc.value,
			}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "body: %s", string(body))
		})

		t.Run("group update rejects "+tc.name, func(t *testing.T) {
			groupID := leadTimeAccountGroup(t, "e2e-grp-lt-badupd", nil)

			status, body, err := apiClient.Patch(accountGroupsPath+"/"+groupID,
				map[string]any{"default_lead_time_days": tc.value}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "body: %s", string(body))
		})

		t.Run("customer create rejects "+tc.name, func(t *testing.T) {
			body := validCustomerBody(uniqueName("e2e-cust-lt-bad"))
			body["lead_time_days"] = tc.value

			status, respBody, err := apiClient.Post(customersPath, body, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(respBody))
			assert.Equal(t, 400, status, "body: %s", string(respBody))
		})

		t.Run("customer update rejects "+tc.name, func(t *testing.T) {
			customerID := leadTimeCustomer(t, "e2e-cust-lt-badupd", nil, "")

			status, respBody, err := apiClient.Patch(customersPath+"/"+customerID,
				map[string]any{"lead_time_days": tc.value}, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(respBody))
			assert.Equal(t, 400, status, "body: %s", string(respBody))
		})
	}
}

// A customer's own lead time is part of the customer, so it has to be visible on
// the customer rather than only through the resolved lead-time endpoint.
func TestCustomerLeadTime_AppearsOnTheCustomer(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-cust-lt-field", ptrInt(17), "")

	// Ordering defaults are expandable, so they are asked for rather than assumed.
	status, body, err := apiClient.GetListRaw(customersPath+"/"+customerID+"?include=defaults", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	defaults, ok := parseJSON(body)["defaults"].(map[string]any)
	require.True(t, ok, "a customer carries its ordering defaults: %s", string(body))
	assert.Equal(t, "17", jsonField(defaults, "lead_time_days"))
}

// Another tenant's customer must not resolve, or a competitor's negotiated lead
// times would be readable by id.
func TestCustomerLeadTime_TenantIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	customerID := leadTimeCustomer(t, "e2e-cust-lt-tenant", ptrInt(9), "")

	status, body, err := clientB.GetListRaw(customersPath+"/"+customerID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "cross-tenant read must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "tenant B must not resolve tenant A's customer: %s", string(body))
}
