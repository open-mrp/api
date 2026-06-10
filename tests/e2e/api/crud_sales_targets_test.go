//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountUsersForSalesTargetsPath = "/v1/identity/account-users"

// discoverActiveAccountUserID returns the seeded admin account user, which is
// guaranteed active and cannot be disabled or removed. Picking an arbitrary
// active user from the list races with parallel account-user tests that remove
// their transient users mid-flight.
func discoverActiveAccountUserID(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(accountUsersForSalesTargetsPath+"/"+SeedAccountUserID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	require.Equal(t, "active", jsonField(parseJSON(body), "status"), "seed account user must be active")
	return SeedAccountUserID
}

func salesTargetsPathFor(accountUserID string) string {
	return "/v1/sales/account-users/" + accountUserID + "/sales-targets"
}

func TestSalesTargets_List(t *testing.T) {
	t.Parallel()
	userID := discoverActiveAccountUserID(t)
	path := salesTargetsPathFor(userID)
	list, _, err := apiClient.GetList(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
}

func TestSalesTargets_ListResponseShape(t *testing.T) {
	t.Parallel()
	userID := discoverActiveAccountUserID(t)
	path := salesTargetsPathFor(userID)

	createBody := map[string]any{
		"start_date":     "2028-01-01T00:00:00Z",
		"end_date":       "2028-03-31T00:00:00Z",
		"amount_value":   "10000.00",
		"amount_unit_id": SeedUnitID,
	}
	status, respBody, err := apiClient.Post(path, createBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)

	m := parseJSON(respBody)
	assert.Equal(t, "sales_target", jsonField(m, "object"))
	assert.NotEmpty(t, jsonField(m, "id"))
	assert.NotEmpty(t, jsonField(m, "start_at"))
	assert.NotEmpty(t, jsonField(m, "end_at"))
	assert.NotEmpty(t, jsonField(m, "created_at"))
	assert.NotEmpty(t, jsonField(m, "updated_at"))

	salesRep := jsonObject(m, "sales_rep")
	require.NotNil(t, salesRep, "sales_rep should be a stub object")
	assert.NotEmpty(t, jsonField(salesRep, "id"))
	assert.Equal(t, "user", jsonField(salesRep, "object"))

	amount := jsonObject(m, "amount")
	require.NotNil(t, amount, "amount should be present")
	assert.NotEmpty(t, jsonField(amount, "id"))
	assert.Equal(t, "quantity", jsonField(amount, "object"))
	assert.NotEmpty(t, jsonField(amount, "value"))

	unit := jsonObject(amount, "unit")
	require.NotNil(t, unit, "amount.unit should be present")
	assert.NotEmpty(t, jsonField(unit, "id"))
	assert.Equal(t, "unit", jsonField(unit, "object"))
}

func TestSalesTargets_CreateAndUpsert(t *testing.T) {
	t.Parallel()
	userID := discoverActiveAccountUserID(t)
	path := salesTargetsPathFor(userID)

	createBody := map[string]any{
		"start_date":     "2027-01-01T00:00:00Z",
		"end_date":       "2027-03-31T00:00:00Z",
		"amount_value":   "50000.00",
		"amount_unit_id": SeedUnitID,
	}
	status, respBody, err := apiClient.Post(path, createBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)

	created := parseJSON(respBody)
	targetID := jsonField(created, "id")
	assert.NotEmpty(t, targetID)
	assert.Equal(t, "sales_target", jsonField(created, "object"))

	salesRep := jsonObject(created, "sales_rep")
	require.NotNil(t, salesRep)
	assert.Equal(t, userID, jsonField(salesRep, "id"))

	amount := jsonObject(created, "amount")
	require.NotNil(t, amount)
	assert.Contains(t, jsonField(amount, "value"), "50000")

	upsertBody := map[string]any{
		"start_date":     "2027-04-01T00:00:00Z",
		"end_date":       "2027-06-30T00:00:00Z",
		"amount_value":   "75000.00",
		"amount_unit_id": SeedUnitID,
	}
	upsertPath := path + "/" + targetID
	uStatus, uBody, err := apiClient.Put(upsertPath, upsertBody)
	require.NoError(t, err)
	requireStatus(t, 200, uStatus, uBody)

	updated := parseJSON(uBody)
	assert.Equal(t, targetID, jsonField(updated, "id"))
	assert.Equal(t, "sales_target", jsonField(updated, "object"))

	updatedAmount := jsonObject(updated, "amount")
	require.NotNil(t, updatedAmount)
	assert.Contains(t, jsonField(updatedAmount, "value"), "75000")
}
