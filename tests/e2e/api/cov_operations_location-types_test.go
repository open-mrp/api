//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage tests for GET /v1/operations/location-types and
// GET /v1/operations/location-types/{id}.
//
// This is a read-only, global, system-owned lookup resource (the fixed
// 6-row storage_location_type enum). There is no create/update/delete, so
// only list/get/search/expandable-na/403 categories apply here; see
// tests/e2e/api/crud_locations_test.go for the existing basic list/shape/
// get-by-code/not-found tests (locationTypesPath is declared there and
// reused below).
//
// Gaps closed here (see TASK-operations_location-types.md):
//  1. Get-by-ID (not just by-code) — GetLocationType matches `id OR code`.
//  2. Positive `q` search match (not just the generic nonsense-query sweep).
//  3. Customer-portal / non-internal-actor 403 on both list and get.

// TestCovOperationsLocationTypes_GetByID resolves a real row ID from the
// list endpoint and confirms GET /{id} (as opposed to GET /{code}) returns
// the same row. GetLocationType matches `WHERE slt.id = ? OR slt.code = ?`,
// but every pre-existing test only exercised the `code` branch.
func TestCovOperationsLocationTypes_GetByID(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(locationTypesPath, nil)
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "expected at least one seeded location type")

	first := parseJSON(list.Data[0])
	id := jsonField(first, "id")
	code := jsonField(first, "code")
	name := jsonField(first, "name")
	require.NotEmpty(t, id)
	require.NotEmpty(t, code)

	status, body, err := apiClient.GetListRaw(locationTypesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertObjectField(t, got, "location_type")
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, code, jsonField(got, "code"))
	assert.Equal(t, name, jsonField(got, "name"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// TestCovOperationsLocationTypes_ListSearchByName proves that `q` actually
// filters server-side on `name` (LIKE '%term%'), not just that a nonsense
// query returns empty (already covered by the generic
// TestListEndpoints_SearchNonsense sweep). Uses a partial substring match to
// also confirm it's a substring LIKE, not an exact match.
func TestCovOperationsLocationTypes_ListSearchByName(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(locationTypesPath, url.Values{"q": {"Building"}})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "expected the Building row to match q=Building")
	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "building", jsonField(m, "code"))
		assert.Equal(t, "Building", jsonField(m, "name"))
	}

	// Partial substring match ("uild" is a substring of "Building").
	subList, _, err := apiClient.GetList(locationTypesPath, url.Values{"q": {"uild"}})
	require.NoError(t, err)
	require.NotEmpty(t, subList.Data, "expected a substring query to match the Building row")
	for _, item := range subList.Data {
		m := parseJSON(item)
		assert.Equal(t, "building", jsonField(m, "code"))
	}
}

// TestCovOperationsLocationTypes_CustomerPortalForbidden_List proves
// CheckIsInternalActor() is enforced on ListLocationTypes: a customer-portal
// actor (non-internal user) must be rejected with 403, not 200, even though
// storage locations/location-types are not customer-facing.
func TestCovOperationsLocationTypes_CustomerPortalForbidden_List(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw(locationTypesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovOperationsLocationTypes_CustomerPortalForbidden_Get extends the
// CheckIsInternalActor() coverage to RetrieveLocationType.
func TestCovOperationsLocationTypes_CustomerPortalForbidden_Get(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	status, body, err := client.GetListRaw(locationTypesPath+"/building", nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}
