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

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedProductLineID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded product line (Socks) should appear in list")
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
	list, _, err := apiClient.GetList(productLinesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestProductLines_ListCursorPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(productLinesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough product lines for pagination test")
		return
	}
	require.NotNil(t, page1.PageInfo.NextPageURL, "next_page_url should be set when has_next_page is true")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should return a different item than page 1")
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
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
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
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(productLinesPath + "/" + id)

	assert.Equal(t, "product_line", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"))
	assert.Equal(t, "billed_freight", jsonField(got, "freight_policy"))
	assertNilField(t, got, "description")
	assertNilField(t, got, "notes")
	assertNilField(t, got, "owner")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group must be set after create")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"))
	assert.Equal(t, "unit_group", jsonField(unitGroup, "object"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-pdln-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id+"?include=unit_group", map[string]any{
		"name":              updatedName,
		"commission_policy": "commission_exempt",
		"freight_policy":    "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "commission_exempt", jsonField(updated, "commission_policy"))
	assert.Equal(t, "free_freight", jsonField(updated, "freight_policy"))
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	// unit_group should be preserved
	updUnitGroup := jsonObject(updated, "unit_group")
	require.NotNil(t, updUnitGroup, "unit_group should be preserved")
	assert.Equal(t, SeedUnitGroupID, jsonField(updUnitGroup, "id"))
}

func TestProductLines_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-shape")
	createResp, err := apiClient.PostFull(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_exempt",
		"freight_policy":    "billed_freight",
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
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestProductLines_CreateValidation_MissingUnitGroupID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              uniqueName("e2e-pdln-noug"),
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing unit_group_id should return 400 or 422, got %d: %s", status, string(body))
}

func TestProductLines_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-pdln")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
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
		"commission_policy": "commission_exempt",
		"freight_policy":    "billed_freight",
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
	assert.Equal(t, "commission_exempt", jsonField(patched, "commission_policy"), "commission_policy should be preserved when only name is updated")

	apiClient.Delete(productLinesPath + "/" + id)
}

func TestProductLines_UpdatePolicyFields(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-policy")
	createStatus, createBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	id := jsonField(created, "id")

	// Update both policy fields
	patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id, map[string]any{
		"commission_policy": "commission_exempt",
		"freight_policy":    "free_freight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "commission_exempt", jsonField(patched, "commission_policy"), "commission_policy should be updated")
	assert.Equal(t, "free_freight", jsonField(patched, "freight_policy"), "freight_policy should be updated")
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved when only policy fields are updated")

	apiClient.Delete(productLinesPath + "/" + id)
}

func TestProductLines_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-pdln-idem-upd")
	createStatus, createBody, err := apiClient.Post(productLinesPath, map[string]any{
		"name":              name,
		"unit_group_id":     SeedUnitGroupID,
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
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
		"commission_policy": "commission_applied",
		"freight_policy":    "billed_freight",
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
			"unit_group_id":     SeedUnitGroupID,
			"commission_policy": "commission_applied",
			"freight_policy":    "billed_freight",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(productLinesPath + "/" + id)

		assertObjectField(t, got, "product_line")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"))
		assert.Equal(t, "billed_freight", jsonField(got, "freight_policy"))
		assertNilField(t, got, "description")
		assertNilField(t, got, "notes")
		assertNilField(t, got, "owner")
		assertNilField(t, got, "unit_group")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-pdln-pres")
		createStatus, createBody, err := apiClient.Post(productLinesPath+"?include=unit_group", map[string]any{
			"name":              name,
			"unit_group_id":     SeedUnitGroupID,
			"commission_policy": "commission_applied",
			"freight_policy":    "billed_freight",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(productLinesPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-pdln-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(productLinesPath+"/"+id+"?include=unit_group", map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "commission_applied", jsonField(got, "commission_policy"), "commission_policy should be preserved")
		assert.Equal(t, "billed_freight", jsonField(got, "freight_policy"), "freight_policy should be preserved")
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
		"commission_policy": "commission_exempt",
		"freight_policy":    "billed_freight",
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
