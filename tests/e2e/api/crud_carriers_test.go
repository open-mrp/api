//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const carriersPath = "/v1/operations/carriers"

func TestCarriers_CRUD(t *testing.T) {
	t.Parallel()

	// CREATE
	name := uniqueName("e2e-carrier")
	createResp, err := apiClient.PostFull(carriersPath, map[string]any{
		"name": name,
		"code": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "carrier", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "will_call", jsonField(created, "code"))
	assert.Equal(t, "visible", jsonField(created, "customer_portal_visibility"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(carriersPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// UPDATE
	newName := uniqueName("e2e-carrier-upd")
	patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"name":                       newName,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "hidden", jsonField(updated, "customer_portal_visibility"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(carriersPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// GET after delete → 404
	getStatus2, _, err := apiClient.GetListRaw(carriersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestCarriers_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-carr-allf")
	createResp, err := apiClient.PostFull(carriersPath, map[string]any{
		"name":                       name,
		"code":                       "will_call",
		"account_number":             "CARRIER-001",
		"customer_portal_visibility": "visible",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(carriersPath + "/" + id)

	assert.Equal(t, "carrier", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "will_call", jsonField(got, "code"))
	assert.Equal(t, "CARRIER-001", jsonField(got, "account_number"))
	assert.Equal(t, "visible", jsonField(got, "customer_portal_visibility"))
	assertNilField(t, got, "deleted_at")
	assertNilField(t, got, "owner")
	assertNilField(t, got, "service_levels")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-carr-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"name":                       updatedName,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "hidden", jsonField(updated, "customer_portal_visibility"))
	assert.Equal(t, "will_call", jsonField(updated, "code"), "code should be preserved")
	assert.Equal(t, "CARRIER-001", jsonField(updated, "account_number"), "account_number should be preserved")
	assertNilField(t, updated, "deleted_at")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
}

func TestCarriers_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(carriersPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedCarrierID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded carrier %q should appear in list", SeedCarrierID)
}

func TestCarriers_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(carriersPath, url.Values{"q": {"Delivery"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Delivery' should return at least 1 result")
}

func TestCarriers_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(carriersPath, url.Values{"q": {"zzzznotacarrier99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestCarriers_ListPagination(t *testing.T) {
	t.Parallel()
	list1, _, err := apiClient.GetList(carriersPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list1.Data, 1)

	if !list1.PageInfo.HasNextPage {
		t.Skip("Not enough data for pagination test")
		return
	}

	list2, _, err := apiClient.GetList(carriersPath, url.Values{
		"limit":  {"1"},
		"cursor": {*list1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	assert.Len(t, list2.Data, 1)

	id1 := DataItemField(list1.Data[0], "id")
	id2 := DataItemField(list2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "Paginated pages should return different items")
}

func TestCarriers_GetSeeded(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(carriersPath+"/"+SeedCarrierID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedCarrierID, jsonField(got, "id"))
	assert.Equal(t, "carrier", jsonField(got, "object"))
	assert.Equal(t, "Delivery", jsonField(got, "name"))
	assert.Equal(t, "delivery", jsonField(got, "code"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestCarriers_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-carrier-shape")
	createResp, err := apiClient.PostFull(carriersPath, map[string]any{
		"name":                       name,
		"code":                       "will_call",
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "carrier", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "will_call", jsonField(got, "code"))
	assert.Equal(t, "hidden", jsonField(got, "customer_portal_visibility"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// Expandable fields should be null without ?include
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")
	assert.Nil(t, got["service_levels"], "service_levels should be null without ?include=service_levels")

	apiClient.Delete(carriersPath + "/" + id)
}

func TestCarriers_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-carrier")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(carriersPath, map[string]any{"name": name, "code": "will_call"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(carriersPath, map[string]any{"name": name, "code": "will_call"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(carriersPath + "/" + id1)
}

func TestCarriers_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(carriersPath, map[string]any{
		"name": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestCarriers_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(carriersPath+"/"+SeedCarrierID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	owner := jsonObject(parseJSON(body), "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.NotEmpty(t, jsonField(owner, "type"))
}

func TestCarriers_IncludeServiceLevels(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(carriersPath+"/"+SeedCarrierID, url.Values{"include": {"service_levels"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	sl, ok := got["service_levels"]
	require.True(t, ok, "service_levels field should be present with ?include=service_levels")
	_, isArray := sl.([]any)
	assert.True(t, isArray, "service_levels should be an array, got %T", sl)
}

func TestCarriers_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-carrier-pn")
	createStatus, createBody, err := apiClient.Post(carriersPath, map[string]any{
		"name":                       name,
		"code":                       "will_call",
		"customer_portal_visibility": "visible",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-carrier-pn-upd")
	patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "visible", jsonField(patched, "customer_portal_visibility"))

	apiClient.Delete(carriersPath + "/" + id)
}

func TestCarriers_UpdateOnlyVisibility(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-carrier-pv")
	createStatus, createBody, err := apiClient.Post(carriersPath, map[string]any{
		"name": name,
		"code": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"))
	assert.Equal(t, "hidden", jsonField(patched, "customer_portal_visibility"))

	apiClient.Delete(carriersPath + "/" + id)
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestCarriers_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-carr-omit")
		status, body, err := apiClient.Post(carriersPath, map[string]any{
			"name": name,
			"code": "will_call",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(carriersPath + "/" + id)

		assertObjectField(t, got, "carrier")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "will_call", jsonField(got, "code"))
		assertNilField(t, got, "account_number")
		assert.Equal(t, "visible", jsonField(got, "customer_portal_visibility"))
		assertNilField(t, got, "deleted_at")
		assertNilField(t, got, "owner")
		assertNilField(t, got, "service_levels")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-carr-pres")
		createStatus, createBody, err := apiClient.Post(carriersPath, map[string]any{
			"name":                       name,
			"code":                       "will_call",
			"account_number":             "ACCT-PRES",
			"customer_portal_visibility": "hidden",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(carriersPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-carr-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(carriersPath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "will_call", jsonField(got, "code"), "code should be preserved")
		assert.Equal(t, "ACCT-PRES", jsonField(got, "account_number"), "account_number should be preserved")
		assert.Equal(t, "hidden", jsonField(got, "customer_portal_visibility"), "customer_portal_visibility should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

func TestCarriers_CreateDuplicateNameFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-carrier-dup")
	status1, body1, err := apiClient.Post(carriersPath, map[string]any{
		"name": name,
		"code": "will_call",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(carriersPath, map[string]any{
		"name": name,
		"code": "delivery",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status2, "Duplicate name should return 409, got %d: %s", status2, string(body2))

	apiClient.Delete(carriersPath + "/" + id)
}
