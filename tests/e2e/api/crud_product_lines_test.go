//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productLinesPath = "/v1/catalog/product-lines"

// --- List ---

func TestProductLines_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productLinesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded product line")

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assertListContainsID(t, productLinesPath, nil, SeedProductLineID)
}

func TestProductLines_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productLinesPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "product_line", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestProductLines_ListPagination(t *testing.T) {
	t.Parallel()
	assertListPageLen(t, productLinesPath, url.Values{"limit": {"1"}}, 1)
}

func TestProductLines_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, productLinesPath, nil)
}

func TestProductLines_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productLinesPath, url.Values{"q": {"Socks"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Socks' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "socks"),
			"Search result %q should contain 'socks'", name,
		)
	}
}

func TestProductLines_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(productLinesPath, url.Values{"q": {"zzzznotaproductline99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestProductLines_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+SeedProductLineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedProductLineID, jsonField(got, "id"))
	assert.Equal(t, "product_line", jsonField(got, "object"))
	assert.Equal(t, SeedProductLineName, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestProductLines_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(productLinesPath+"/pdln_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- CRUD ---

func TestProductLines_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln")

	// Create
	createResp, err := apiClient.PostFull(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "product_line", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-pdln-upd")
	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(productLinesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(productLinesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- Create ---

func TestProductLines_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-pdln-allf")
	createResp, err := apiClient.PostFull(productLinesPath+"?include=unit_group", map[string]any{
		"name":              name,
		"description":       "Wool and cotton socks.",
		"status":            "active",
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
		"supply":            "make",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertIDFormat(t, id, "pdln")
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(productLinesPath + "/" + id)

	assert.Equal(t, "product_line", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "Wool and cotton socks.", jsonField(got, "description"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, "applied", jsonField(got, "commission_policy"))
	assert.Equal(t, "billed", jsonField(got, "freight_policy"))
	assert.Equal(t, "make", jsonField(got, "supply"))
	assertNilField(t, got, "owner")
	assert.NotContains(t, got, "notes")
	assert.NotContains(t, got, "default_lot")
	assert.NotContains(t, got, "fulfillment_policy")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group must be set after create")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-pdln-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id+"?include=unit_group", map[string]any{
		"name":              updatedName,
		"description":       "Socks only.",
		"status":            "archived",
		"commission_policy": "exempt",
		"freight_policy":    "free",
		"supply":            "buy",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "Socks only.", jsonField(updated, "description"))
	assert.Equal(t, "archived", jsonField(updated, "status"))
	assert.Equal(t, "exempt", jsonField(updated, "commission_policy"))
	assert.Equal(t, "free", jsonField(updated, "freight_policy"))
	assert.Equal(t, "buy", jsonField(updated, "supply"))
	assert.Equal(t, jsonField(got, "created_at"), jsonField(updated, "created_at"))
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
	updUnitGroup := jsonObject(updated, "unit_group")
	require.NotNil(t, updUnitGroup, "unit_group should be preserved")
	assert.Equal(t, SeedUnitGroupID, jsonField(updUnitGroup, "id"))

	// ── CLEAR the clearable fields ──
	clearStatus, clearBody, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"description": nil,
		"supply":      nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	cleared := parseJSON(clearBody)
	assertNilField(t, cleared, "description")
	assertNilField(t, cleared, "supply")
	assert.Equal(t, updatedName, jsonField(cleared, "name"))
	assert.Equal(t, "archived", jsonField(cleared, "status"))
}

func TestProductLines_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-shape")
	createResp, err := apiClient.PostFull(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "exempt",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "product_line", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	// Policy fields should be present
	assert.NotEmpty(t, jsonField(created, "commission_policy"), "commission_policy should be present in response")
	assert.NotEmpty(t, jsonField(created, "freight_policy"), "freight_policy should be present in response")

	apiClient.Delete(productLinesPath + "/" + id)
}

func TestProductLines_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestProductLines_CreateWithoutUnitGroup(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-noug"),
		"commission_policy": "applied",
		"freight_policy":    "billed",
	})
	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+jsonField(created, "id"), url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "unit_group")
}

func TestProductLines_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-pdln")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(productLinesPath + "/" + id1)
}

// --- Update ---

func TestProductLines_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-pname")
	createStatus, createBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "exempt",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")

	newName := uniqueName("e2e-pdln-pname2")
	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "exempt", jsonField(patched, "commission_policy"), "commission_policy should be preserved when only name is updated")

	apiClient.Delete(productLinesPath + "/" + id)
}

func TestProductLines_UpdatePolicyFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-policy")
	createStatus, createBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")

	// Update both policy fields
	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"commission_policy": "exempt",
		"freight_policy":    "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "exempt", jsonField(patched, "commission_policy"), "commission_policy should be updated")
	assert.Equal(t, "free", jsonField(patched, "freight_policy"), "freight_policy should be updated")
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved when only policy fields are updated")

	apiClient.Delete(productLinesPath + "/" + id)
}

func TestProductLines_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-idem-upd")
	createStatus, createBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-pdln-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(productLinesPath + "/" + id)
}

func TestProductLines_UpdateDefaultProductLineRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(productLinesPath+"/shipping", map[string]any{
		"name": uniqueName("e2e-pdln-default"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 422,
		"Updating a default product line should return 400, 403, or 422, got %d: %s", status, string(body))
}

// --- Delete ---

func TestProductLines_DeleteNotFoundReturns404(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(productLinesPath + "/pdln_000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestProductLines_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-deldel")
	createStatus, createBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Second delete should fail
	status2, _, err := apiClient.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted product line should return 404 or 410, got %d", status2)
}

func TestProductLines_DeleteReturnsStub(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-stub"),
		"commission_policy": "applied",
		"freight_policy":    "billed",
	})
	id := jsonField(created, "id")

	status, body, err := apiClient.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	stub := parseJSON(body)
	assert.Equal(t, id, jsonField(stub, "id"))
	assert.Equal(t, "product_line", jsonField(stub, "object"))
	assert.Equal(t, "true", jsonField(stub, "deleted"))
}

// A line that a product still sells under can't be deleted; archiving is the way to retire it.
func TestProductLines_DeleteInUseConflicts(t *testing.T) {
	t.Parallel()
	lineID := jsonField(createAndCleanup(t, productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-inuse"),
		"commission_policy": "applied",
		"freight_policy":    "billed",
	}), "id")
	createSellableItemInLine(t, uniqueName("e2e-pdln-inuse-sku"), lineID)

	status, body, err := apiClient.Delete(productLinesPath + "/" + lineID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "products")
	assert.Contains(t, jsonField(errObj, "message"), "archived")

	getStatus, _, err := apiClient.GetListRaw(productLinesPath+"/"+lineID, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, getStatus, "the line survives a refused delete")
}

// Customer and account-group access to a line goes with it.
func TestProductLines_DeleteRemovesAccess(t *testing.T) {
	t.Parallel()
	fx := newAccessFixture(t)
	customerPath := customerAccessPath + "/" + fx.customerID
	grantAccess(t, customerAccessPath, map[string]any{"customer_id": fx.customerID, "product_line_ids": []string{fx.lineA, fx.lineB}})
	t.Cleanup(func() { _, _, _ = apiClient.Delete(customerPath) })

	status, body, err := apiClient.Delete(productLinesPath + "/" + fx.lineA)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	readStatus, record := readAccess(t, customerPath)
	require.Equal(t, 200, readStatus)
	assert.Equal(t, []string{fx.lineB}, grantedLines(t, record))
}

func TestProductLines_DeleteDefaultProductLineRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(productLinesPath + "/shipping")
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 422,
		"Deleting a default product line should return 400, 403, or 422, got %d: %s", status, string(body))
}

// --- Expandable Fields ---

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestProductLines_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-pdln-omit")
		status, body, err := apiClient.Post(productLinesPath, map[string]any{
			"name":              name,
			"commission_policy": "applied",
			"freight_policy":    "billed",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(productLinesPath + "/" + id)

		assertObjectField(t, got, "product_line")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "active", jsonField(got, "status"), "status defaults to active")
		assert.Equal(t, "applied", jsonField(got, "commission_policy"))
		assert.Equal(t, "billed", jsonField(got, "freight_policy"))
		assertNilField(t, got, "description")
		assertNilField(t, got, "supply")
		assertNilField(t, got, "owner")
		assertNilField(t, got, "unit_group")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("MissingRequiredPolicies", func(t *testing.T) {
		status, body, err := apiClient.Post(productLinesPath, map[string]any{
			"name": uniqueName("e2e-pdln-nopol"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-pdln-pres")
		createStatus, createBody, err := apiClient.Post(productLinesPath+"?include=unit_group", map[string]any{
			"name":              name,
			"description":       "Keep me.",
			"unit_group_id":     SeedUnitGroupID,
			"commission_policy": "applied",
			"freight_policy":    "billed",
			"supply":            "stock",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(productLinesPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		newName := uniqueName("e2e-pdln-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id+"?include=unit_group", map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "Keep me.", jsonField(got, "description"))
		assert.Equal(t, "active", jsonField(got, "status"))
		assert.Equal(t, "applied", jsonField(got, "commission_policy"), "commission_policy should be preserved")
		assert.Equal(t, "billed", jsonField(got, "freight_policy"), "freight_policy should be preserved")
		assert.Equal(t, "stock", jsonField(got, "supply"), "supply should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		ug := jsonObject(got, "unit_group")
		require.NotNil(t, ug, "unit_group should be preserved")
		assert.Equal(t, SeedUnitGroupID, jsonField(ug, "id"))
	})
}

func TestProductLines_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+SeedProductLineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")
	assert.Nil(t, got["unit_group"], "unit_group should be null without ?include=unit_group")

	// Test on List
	list, _, err := apiClient.GetList(productLinesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null on list items without ?include=owner")
		assert.Nil(t, m["unit_group"], "unit_group should be null on list items without ?include=unit_group")
	}
}

func TestProductLines_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+SeedProductLineID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}

func TestProductLines_IncludeUnitGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productLinesPath+"/"+SeedProductLineID, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, "unit_group", jsonField(unitGroup, "object"))
	assert.NotEmpty(t, jsonField(unitGroup, "id"))
	assert.NotEmpty(t, jsonField(unitGroup, "name"))
}

func TestProductLines_IncludeOwnerAccount(t *testing.T) {
	t.Parallel()

	// Create an account-owned product line so owner.account is always populated.
	name := uniqueName("e2e-pdln-owneracct")
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "exempt",
		"freight_policy":    "billed",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(productLinesPath + "/" + id)

	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+id, url.Values{"include": {"owner.account"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner.account")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	assert.Equal(t, "account", jsonField(owner, "type"))

	account := jsonObject(owner, "account")
	require.NotNil(t, account, "account should be present inside owner with ?include=owner.account")
	assert.Equal(t, "account", jsonField(account, "object"))
	assert.NotEmpty(t, jsonField(account, "id"))
}

func TestProductLines_ListIncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productLinesPath, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)

	for _, raw := range data {
		m, ok := raw.(map[string]any)
		require.True(t, ok)
		owner := jsonObject(m, "owner")
		require.NotNil(t, owner, "owner should be present on each list item with ?include=owner")
		assert.Equal(t, "owner", jsonField(owner, "object"))
		ownerType := jsonField(owner, "type")
		assert.Contains(t, []string{"system", "account"}, ownerType)
	}
}

func TestProductLines_ListIncludeUnitGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productLinesPath, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)

	found := false
	for _, raw := range data {
		m, ok := raw.(map[string]any)
		require.True(t, ok)
		if ug := jsonObject(m, "unit_group"); ug != nil {
			assert.Equal(t, "unit_group", jsonField(ug, "object"))
			assert.NotEmpty(t, jsonField(ug, "id"))
			found = true
		}
	}
	assert.True(t, found, "at least one list item should have a unit_group with ?include=unit_group")
}

// ==========================================================================
// Bulk Upsert
// ==========================================================================

const productLinesBulkUpsertPath = productLinesPath + "/actions/bulk-upsert"

// bulkUpsertPLInput builds a minimal product line entry for bulk-upsert payloads.
// The unit group is a fuzzy reference, so it is sent as an object keyed by id.
func bulkUpsertPLInput(name, unitGroupID, commissionPolicy, freightPolicy string) map[string]any {
	return map[string]any{
		"name":              name,
		"unit_group":        map[string]any{"id": unitGroupID},
		"commission_policy": commissionPolicy,
		"freight_policy":    freightPolicy,
	}
}

func cleanupProductLineIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(productLinesPath + "/" + id)
		}
	}
}

// bulkUpsertProductLines posts a bulk upsert of the given rows.
func bulkUpsertProductLines(t *testing.T, rows ...map[string]any) (int, []byte) {
	t.Helper()
	items := make([]any, len(rows))
	for i, r := range rows {
		items[i] = r
	}
	status, body, err := apiClient.Post(productLinesBulkUpsertPath, map[string]any{"product_lines": items}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

// bulkUpsertProductLinesJob posts a bulk upsert, requires the 202 job acknowledgment, and
// returns the completed job.
func bulkUpsertProductLinesJob(t *testing.T, rows ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertProductLines(t, rows...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "async_job", jsonField(m, "object"), "202 returns the async job")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// bulkUpsertProductLineIDs posts a bulk upsert, follows the job to completion, and returns
// the created/updated product line IDs from its results.
func bulkUpsertProductLineIDs(t *testing.T, rows ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertProductLinesJob(t, rows...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

func TestProductLines_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()
	name1 := uniqueName("e2e-bulkpl-a")
	name2 := uniqueName("e2e-bulkpl-b")

	createdIDs, updatedIDs := bulkUpsertProductLineIDs(t,
		bulkUpsertPLInput(name1, SeedUnitGroupID, "applied", "billed"),
		bulkUpsertPLInput(name2, SeedUnitGroupID, "exempt", "free"),
	)
	defer cleanupProductLineIDs(createdIDs)

	require.Len(t, createdIDs, 2, "should have 2 created IDs")
	for _, id := range createdIDs {
		assertIDFormat(t, id, "pdln")
	}
	assert.Empty(t, updatedIDs, "no updates expected")
}

func TestProductLines_BulkUpsert_AllUpdates(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulkpl-upd")

	// Create
	createdIDs, _ := bulkUpsertProductLineIDs(t, bulkUpsertPLInput(name, SeedUnitGroupID, "applied", "billed"))
	defer cleanupProductLineIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	productLineID := createdIDs[0]

	// Update (same name → matches, flips policies)
	created, updated := bulkUpsertProductLineIDs(t, bulkUpsertPLInput(name, SeedUnitGroupID, "exempt", "free"))
	assert.Empty(t, created, "no creates expected on update")
	require.Len(t, updated, 1, "should have 1 updated ID")
	assert.Equal(t, productLineID, updated[0], "updated ID must match the originally created ID")

	// Verify the policy change persisted.
	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+productLineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "exempt", jsonField(got, "commission_policy"))
	assert.Equal(t, "free", jsonField(got, "freight_policy"))
}

func TestProductLines_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()
	existingName := uniqueName("e2e-bulkpl-mix-exist")
	newName := uniqueName("e2e-bulkpl-mix-new")

	// Seed existing
	seeded, _ := bulkUpsertProductLineIDs(t, bulkUpsertPLInput(existingName, SeedUnitGroupID, "applied", "billed"))
	defer cleanupProductLineIDs(seeded)

	// Mix: update existing + create new
	created, updated := bulkUpsertProductLineIDs(t,
		bulkUpsertPLInput(existingName, SeedUnitGroupID, "applied", "billed"),
		bulkUpsertPLInput(newName, SeedUnitGroupID, "exempt", "free"),
	)
	defer cleanupProductLineIDs(created)

	assert.Len(t, created, 1, "one new product line created")
	assert.Len(t, updated, 1, "one existing product line updated")
}

// TestProductLines_BulkUpsert_Idempotency: replaying the accept with the same idempotency
// key returns the same job, so the work is enqueued once.
func TestProductLines_BulkUpsert_Idempotency(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulkpl-idem")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"product_lines": []any{bulkUpsertPLInput(name, SeedUnitGroupID, "applied", "billed")},
	}

	status1, body1, err := apiClient.Post(productLinesBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status1, body1)
	jobID := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, jobID)

	status2, body2, err := apiClient.Post(productLinesBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status2, body2)
	assert.Equal(t, jobID, jsonField(parseJSON(body2), "id"), "idempotent request must return the same job")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))
	created, _ := jobResultIDs(job)
	defer cleanupProductLineIDs(created)
	require.Len(t, created, 1)
}

func TestProductLines_BulkUpsert_DuplicateNameRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulkpl-dup")

	status, body := bulkUpsertProductLines(t,
		bulkUpsertPLInput(name, SeedUnitGroupID, "applied", "billed"),
		bulkUpsertPLInput(name, SeedUnitGroupID, "exempt", "free"),
	)
	requireStatus(t, 422, status, body)
}

// TestProductLines_BulkUpsert_ResolvesUnitGroupByName: the unit group is a fuzzy
// reference — it resolves by name, not only by id.
func TestProductLines_BulkUpsert_ResolvesUnitGroupByName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulkpl-ugname")

	createdIDs, _ := bulkUpsertProductLineIDs(t, map[string]any{
		"name":              name,
		"unit_group":        map[string]any{"name": "socks"}, // by name, case-insensitive
		"commission_policy": "applied",
		"freight_policy":    "billed",
	})
	defer cleanupProductLineIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(productLinesPath+"/"+createdIDs[0], url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	ug := jsonObject(parseJSON(getBody), "unit_group")
	require.NotNil(t, ug, "unit_group should be populated with ?include=unit_group")
	assert.Equal(t, SeedUnitGroupID, jsonField(ug, "id"))
}

// TestProductLines_BulkUpsert_RejectsUnknownUnitGroup: references resolve at accept, so an
// unresolvable unit group is a synchronous 400 naming the offending row and no job is raised.
func TestProductLines_BulkUpsert_RejectsUnknownUnitGroup(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertProductLines(t,
		bulkUpsertPLInput(uniqueName("e2e-bulkpl-ugok"), SeedUnitGroupID, "applied", "billed"),
		bulkUpsertPLInput(uniqueName("e2e-bulkpl-ugbad"), "ungp_does_not_exist_00000", "exempt", "free"),
	)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "product_lines[1].unit_group")
}

// --- Status ---

func TestProductLines_ListFiltersByStatus(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-pdln-st")
	activeID := jsonField(createAndCleanup(t, productLinesPath, map[string]any{
		"name": prefix + "-a", "commission_policy": "applied", "freight_policy": "billed",
	}), "id")
	archivedID := jsonField(createAndCleanup(t, productLinesPath, map[string]any{
		"name": prefix + "-b", "status": "archived", "commission_policy": "applied", "freight_policy": "billed",
	}), "id")

	ids := func(q url.Values) []string {
		list, _, err := apiClient.GetList(productLinesPath, q)
		require.NoError(t, err)
		var out []string
		for _, item := range list.Data {
			out = append(out, DataItemField(item, "id"))
		}
		return out
	}

	all := ids(url.Values{"q": {prefix}})
	assert.ElementsMatch(t, []string{activeID, archivedID}, all)

	archived := ids(url.Values{"q": {prefix}, "statuses[]": {"archived"}})
	assert.Equal(t, []string{archivedID}, archived)

	active := ids(url.Values{"q": {prefix}, "statuses[]": {"active"}})
	assert.Equal(t, []string{activeID}, active)

	status, body, err := apiClient.GetListRaw(productLinesPath, url.Values{"statuses[]": {"inactive"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// Search is a case-insensitive substring match on the name, not a word match.
func TestProductLines_ListSearchIsSubstring(t *testing.T) {
	t.Parallel()
	token := strings.ToLower(uniqueName("Zq"))
	id := jsonField(createAndCleanup(t, productLinesPath, map[string]any{
		"name": "Line" + strings.ToUpper(token) + "Tail", "commission_policy": "applied", "freight_policy": "billed",
	}), "id")

	list, _, err := apiClient.GetList(productLinesPath, url.Values{"q": {token}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, id, DataItemField(list.Data[0], "id"))
}

// An archived line stays on its products but can't be assigned to a new one or newly granted.
func TestProductLines_ArchivedCannotBeNewlyAssigned(t *testing.T) {
	t.Parallel()
	fx := newAccessFixture(t)
	setStatus := func(id, status string) {
		code, body, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{"status": status}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, code, body)
	}

	itemID := createSellableItemInLine(t, uniqueName("e2e-pdln-arch-sku"), fx.lineA)
	_ = itemID
	setStatus(fx.lineA, "archived")

	t.Run("new product", func(t *testing.T) {
		body := validProductBody(uniqueName("e2e-pdln-arch-new"))
		body["product_line_id"] = fx.lineA
		status, resp, err := apiClient.Post(productsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, resp)
		errObj := requireErrorResponse(t, resp, "resource_archived", "invalid_request_error")
		assertErrorParam(t, errObj, "product_line_id")
	})

	t.Run("customer grant", func(t *testing.T) {
		status, resp, err := apiClient.Post(customerAccessPath, map[string]any{
			"customer_id": fx.customerID, "product_line_ids": []string{fx.lineA},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, resp)
		errObj := requireErrorResponse(t, resp, "resource_archived", "invalid_request_error")
		assertErrorParam(t, errObj, "product_line_ids")
	})

	t.Run("existing grant keeps the archived line", func(t *testing.T) {
		setStatus(fx.lineA, "active")
		customerPath := customerAccessPath + "/" + fx.customerID
		grantAccess(t, customerAccessPath, map[string]any{"customer_id": fx.customerID, "product_line_ids": []string{fx.lineA}})
		t.Cleanup(func() { _, _, _ = apiClient.Delete(customerPath) })
		setStatus(fx.lineA, "archived")

		status, resp, err := apiClient.Patch(customerPath, map[string]any{
			"product_line_ids": []string{fx.lineA, fx.lineB},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
	})
}

func TestProductLines_InvalidSupplyRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name": uniqueName("e2e-pdln-badsupply"), "commission_policy": "applied", "freight_policy": "billed", "supply": "make_to_order",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
}

func TestProductLines_DeleteIdempotentReplayIsGone(t *testing.T) {
	t.Parallel()
	id := jsonField(createAndCleanup(t, productLinesPath, map[string]any{
		"name": uniqueName("e2e-pdln-gone"), "commission_policy": "applied", "freight_policy": "billed",
	}), "id")
	status, body, err := apiClient.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	status, _, err = apiClient.Delete(productLinesPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, status)
}
