//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const shippingTermsPath = "/v1/operations/shipping-terms"

// --- List ---

func TestShippingTerms_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(shippingTermsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded shipping term")

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assertListContainsID(t, shippingTermsPath, nil, SeedShippingTermID)
}

func TestShippingTerms_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(shippingTermsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "shipping_term", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestShippingTerms_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(shippingTermsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestShippingTerms_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, shippingTermsPath, nil)
}

func TestShippingTerms_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(shippingTermsPath, url.Values{"q": {"Freight"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Freight' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "freight"),
			"Search result %q should contain 'freight'", name,
		)
	}
}

func TestShippingTerms_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(shippingTermsPath, url.Values{"q": {"zzzznotashipterm99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestShippingTerms_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedShippingTermID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedShippingTermID, jsonField(got, "id"))
	assert.Equal(t, "shipping_term", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestShippingTerms_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(shippingTermsPath+"/shtm_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- CRUD ---

func TestShippingTerms_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm")

	// Create
	createResp, err := apiClient.PostFull(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "shipping_term", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "free_freight", jsonField(created, "type"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(shippingTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-shipterm-upd")
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(shippingTermsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(shippingTermsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(shippingTermsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- Create ---

func TestShippingTerms_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields including flat_rate ──
	name := uniqueName("e2e-st-allf")
	createResp, err := apiClient.PostFull(shippingTermsPath+"?include=flat_rate.unit", map[string]any{
		"name": name,
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value":   "9.99",
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(shippingTermsPath + "/" + id)

	assert.Equal(t, "shipping_term", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "flat_rate_freight", jsonField(got, "type"))
	assertNilField(t, got, "minimum_order_value")
	assertNilField(t, got, "owner")
	assertNilField(t, got, "free_shipping_service_levels")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	flatRate := jsonObject(got, "flat_rate")
	require.NotNil(t, flatRate, "flat_rate must be set after create")
	assert.Equal(t, "quantity", jsonField(flatRate, "object"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-st-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id+"?include=flat_rate.unit", map[string]any{
		"name": updatedName,
		"flat_rate": map[string]any{
			"value":   "19.99",
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "flat_rate_freight", jsonField(updated, "type"), "type should be preserved")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	updFlatRate := jsonObject(updated, "flat_rate")
	require.NotNil(t, updFlatRate, "flat_rate should be preserved after update")
}

func TestShippingTerms_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-shape")
	createResp, err := apiClient.PostFull(shippingTermsPath, map[string]any{
		"name": name,
		"type": "carrier_rate_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "shipping_term", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "carrier_rate_freight", jsonField(created, "type"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_CreateWithFlatRate(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-flat")
	createResp, err := apiClient.PostFull(shippingTermsPath, map[string]any{
		"name": name,
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value":   "9.99",
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "flat_rate_freight", jsonField(created, "type"))

	flatRate := jsonObject(created, "flat_rate")
	require.NotNil(t, flatRate, "flat_rate should be populated when provided")
	assert.Equal(t, "quantity", jsonField(flatRate, "object"))
	assert.Equal(t, "9.99", jsonField(flatRate, "value"))

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestShippingTerms_CreateValidation_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": uniqueName("e2e-shipterm-notype"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing type should return 400 or 422, got %d: %s", status, string(body))
}

// --- Idempotency ---

func TestShippingTerms_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-shipterm")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(shippingTermsPath + "/" + id1)
}

func TestShippingTerms_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-idem-upd")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-shipterm-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(shippingTermsPath + "/" + id)
}

// --- Update ---

func TestShippingTerms_UpdateDefaultTermFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(shippingTermsPath+"/"+SeedShippingTermID, map[string]any{
		"name": uniqueName("e2e-default-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Updating a default shipping term should fail, got %d", status)
}

func TestShippingTerms_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-pname")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "carrier_rate_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Patch only the name — type should be preserved
	newName := uniqueName("e2e-shipterm-pname2")
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "carrier_rate_freight", jsonField(patched, "type"), "type should be preserved when only name is updated")

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_UpdateOnlyType(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-ptype")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Patch only the type — name should be preserved
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"type": "flat_rate_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "flat_rate_freight", jsonField(patched, "type"))
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved when only type is updated")

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_UpdateAddFlatRate(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-addflat")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "flat_rate_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Add flat rate via patch
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"flat_rate": map[string]any{
			"value":   "15.00",
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")
	assert.Equal(t, "flat_rate_freight", jsonField(patched, "type"), "type should be preserved")
	flatRate := jsonObject(patched, "flat_rate")
	require.NotNil(t, flatRate, "flat_rate should be set after patch")
	assert.Equal(t, "15.00", jsonField(flatRate, "value"))

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_UpdateClearFlatRate(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-clrflat")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value":   "10.00",
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotNil(t, jsonObject(created, "flat_rate"), "flat_rate should be set after create")

	// Send explicit null to clear flat rate
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"flat_rate": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved")
	assert.Nil(t, patched["flat_rate"], "flat_rate should be cleared after sending null")

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_UpdatePreservesFlatRate(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-keepflat")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "flat_rate_freight",
		"flat_rate": map[string]any{
			"value":   "20.00",
			"unit_id": SeedUnitID,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Update only name — flat_rate should be preserved (not sent at all)
	newName := uniqueName("e2e-shipterm-keepflat2")
	patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	flatRate := jsonObject(patched, "flat_rate")
	require.NotNil(t, flatRate, "flat_rate should be preserved when not included in patch")
	assert.Equal(t, "20.00", jsonField(flatRate, "value"))

	apiClient.Delete(shippingTermsPath + "/" + id)
}

func TestShippingTerms_UpdateEmptyBodyRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-empty")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Empty body should be rejected
	patchStatus, _, err := apiClient.Patch(shippingTermsPath+"/"+id, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "Empty PATCH body should return 400")

	apiClient.Delete(shippingTermsPath + "/" + id)
}

// --- Delete ---

func TestShippingTerms_DeleteDefaultTermFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(shippingTermsPath + "/" + SeedShippingTermID)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Deleting a default shipping term should fail, got %d", status)
}

func TestShippingTerms_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-shipterm-deldel")
	createStatus, createBody, err := apiClient.Post(shippingTermsPath, map[string]any{
		"name": name,
		"type": "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(shippingTermsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Second delete should fail
	status2, _, err := apiClient.Delete(shippingTermsPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted shipping term should return 404 or 410, got %d", status2)
}

// --- Expandable Fields ---

func TestShippingTerms_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedShippingTermID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")
	assert.Nil(t, got["free_shipping_service_levels"], "free_shipping_service_levels should be null without ?include=free_shipping_service_levels")

	// Test on List
	list, _, err := apiClient.GetList(shippingTermsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null on list items without ?include=owner")
		assert.Nil(t, m["free_shipping_service_levels"], "free_shipping_service_levels should be null on list items without ?include=free_shipping_service_levels")
	}
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestShippingTerms_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-st-omit")
		status, body, err := apiClient.Post(shippingTermsPath, map[string]any{
			"name": name,
			"type": "free_freight",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(shippingTermsPath + "/" + id)

		assertObjectField(t, got, "shipping_term")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "free_freight", jsonField(got, "type"))
		assertNilField(t, got, "flat_rate")
		assertNilField(t, got, "minimum_order_value")
		assertNilField(t, got, "owner")
		assertNilField(t, got, "free_shipping_service_levels")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-st-pres")
		createStatus, createBody, err := apiClient.Post(shippingTermsPath+"?include=flat_rate.unit", map[string]any{
			"name": name,
			"type": "flat_rate_freight",
			"flat_rate": map[string]any{
				"value":   "25.00",
				"unit_id": SeedUnitID,
			},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(shippingTermsPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-st-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(shippingTermsPath+"/"+id+"?include=flat_rate.unit", map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "flat_rate_freight", jsonField(got, "type"), "type should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		flatRate := jsonObject(got, "flat_rate")
		require.NotNil(t, flatRate, "flat_rate should be preserved")
		assert.Equal(t, "25.00", jsonField(flatRate, "value"), "flat_rate value should be preserved")
	})
}

func TestShippingTerms_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(shippingTermsPath+"/"+SeedShippingTermID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}
