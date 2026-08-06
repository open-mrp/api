//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const itemCategoriesPath = "/v1/catalog/item-categories"

// --- List ---

func TestItemCategories_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemCategoriesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded item category")

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assertListContainsID(t, itemCategoriesPath, nil, SeedItemCategoryID)
}

func TestItemCategories_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemCategoriesPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "item_category", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestItemCategories_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, list.Data, 1)
}

func TestItemCategories_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, itemCategoriesPath, nil)
}

func TestItemCategories_ListFilterByType(t *testing.T) {
	t.Parallel()

	// Filter by material_category
	matList, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"type": {"material_category"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(matList.Data), 1, "Should have at least 1 material category")
	for _, item := range matList.Data {
		assert.Equal(t, "material_category", DataItemField(item, "type"), "All items should be material_category")
	}

	// Filter by product_category
	prodList, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"type": {"product_category"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(prodList.Data), 1, "Should have at least 1 product category")
	for _, item := range prodList.Data {
		assert.Equal(t, "product_category", DataItemField(item, "type"), "All items should be product_category")
	}
}

func TestItemCategories_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"q": {"Socks"}})
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

func TestItemCategories_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"q": {"zzzznotacategory99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestItemCategories_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+SeedItemCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedItemCategoryID, jsonField(got, "id"))
	assert.Equal(t, "item_category", jsonField(got, "object"))
	assert.Equal(t, SeedItemCategoryName, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestItemCategories_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(itemCategoriesPath+"/itcg_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- CRUD ---

func TestItemCategories_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg")

	// Create
	createResp, err := apiClient.PostFull(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "item_category", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "material_category", jsonField(created, "type"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-itcg-upd")
	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(itemCategoriesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- Create ---

func TestItemCategories_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-itcg-allf")
	createResp, err := apiClient.PostFull(itemCategoriesPath+"?include=unit_group", map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(itemCategoriesPath + "/" + id)

	assert.Equal(t, "item_category", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "material_category", jsonField(got, "type"))
	assertNilField(t, got, "notes")
	assertNilField(t, got, "owner")
	assertNilField(t, got, "properties")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group must be set after create")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-itcg-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id+"?include=unit_group", map[string]any{
		"name":  updatedName,
		"notes": "Updated notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "Updated notes", jsonField(updated, "notes"))
	assert.Equal(t, "material_category", jsonField(updated, "type"), "type should be preserved")

	// unit_group should be preserved
	updUnitGroup := jsonObject(updated, "unit_group")
	require.NotNil(t, updUnitGroup, "unit_group should be preserved")
	assert.Equal(t, SeedUnitGroupID, jsonField(updUnitGroup, "id"))
}

func TestItemCategories_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-shape")
	createResp, err := apiClient.PostFull(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "product_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "item_category", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "product_category", jsonField(created, "type"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

func TestItemCategories_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_CreateValidation_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          uniqueName("e2e-itcg-notype"),
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing type should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_CreateValidation_MissingUnitGroupID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name": uniqueName("e2e-itcg-noug"),
		"type": "material_category",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing unit_group_id should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-itcg")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(itemCategoriesPath + "/" + id1)
}

// --- Update ---

func TestItemCategories_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-pname")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-itcg-pname2")
	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "material_category", jsonField(patched, "type"), "type should be preserved when only name is updated")

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

func TestItemCategories_UpdateOnlyNotes(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-pnotes")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "product_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{
		"notes": "Some notes about this category",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, name, jsonField(patched, "name"), "name should be preserved when only notes is updated")
	assert.Equal(t, "product_category", jsonField(patched, "type"), "type should be preserved when only notes is updated")
	assert.Equal(t, "Some notes about this category", jsonField(patched, "notes"))

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

func TestItemCategories_UpdateSeededCategoryPreservesType(t *testing.T) {
	newName := uniqueName("e2e-itcg-seedupd")
	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+SeedItemCategoryID, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "product_category", jsonField(patched, "type"), "type should be preserved")

	// Restore original name
	_, _, err = apiClient.Patch(itemCategoriesPath+"/"+SeedItemCategoryID, map[string]any{
		"name": SeedItemCategoryName,
	}, newIdempotencyKey())
	require.NoError(t, err)
}

func TestItemCategories_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-idem-upd")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-itcg-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

// --- Delete ---

func TestItemCategories_DeleteNotFoundReturns404(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(itemCategoriesPath + "/itcg_000000000000")
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestItemCategories_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-deldel")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(itemCategoriesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Second delete should fail
	status2, _, err := apiClient.Delete(itemCategoriesPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted item category should return 404 or 410, got %d", status2)
}

// --- Expandable Fields ---

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestItemCategories_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-itcg-omit")
		status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
			"name":          name,
			"type":          "material_category",
			"unit_group_id": SeedUnitGroupID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(itemCategoriesPath + "/" + id)

		assertObjectField(t, got, "item_category")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "material_category", jsonField(got, "type"))
		assertNilField(t, got, "notes")
		assertNilField(t, got, "owner")
		assertNilField(t, got, "properties")
		assertNilField(t, got, "unit_group")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-itcg-pres")
		createStatus, createBody, err := apiClient.Post(itemCategoriesPath+"?include=unit_group", map[string]any{
			"name":          name,
			"type":          "material_category",
			"unit_group_id": SeedUnitGroupID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(itemCategoriesPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-itcg-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id+"?include=unit_group", map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "material_category", jsonField(got, "type"), "type should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		ug := jsonObject(got, "unit_group")
		require.NotNil(t, ug, "unit_group should be preserved")
		assert.Equal(t, SeedUnitGroupID, jsonField(ug, "id"))
	})
}

func TestItemCategories_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+SeedItemCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")
	assert.Nil(t, got["properties"], "properties should be null without ?include=properties")
	assert.Nil(t, got["unit_group"], "unit_group should be null without ?include=unit_group")

	// Test on List
	list, _, err := apiClient.GetList(itemCategoriesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null on list items without ?include=owner")
		assert.Nil(t, m["properties"], "properties should be null on list items without ?include=properties")
		assert.Nil(t, m["unit_group"], "unit_group should be null on list items without ?include=unit_group")
	}
}

func TestItemCategories_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+SeedItemCategoryID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}

func TestItemCategories_IncludeProperties(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+SeedItemCategoryID, url.Values{"include": {"properties"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	props := jsonObject(got, "properties")
	require.NotNil(t, props, "properties should be present with ?include=properties")
	assert.Equal(t, "list", jsonField(props, "object"))
}

func TestItemCategories_IncludeUnitGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+SeedItemCategoryID, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, "unit_group", jsonField(unitGroup, "object"))
	assert.NotEmpty(t, jsonField(unitGroup, "id"))
	assert.NotEmpty(t, jsonField(unitGroup, "name"))
}

// --- Property Management ---

func TestItemCategories_AddAndRemoveProperty(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-prop")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Add property
	addStatus, addBody, err := apiClient.Put(itemCategoriesPath+"/"+id+"/properties/"+SeedPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, addStatus, addBody)

	// Verify property was added via include
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"properties"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	props := jsonObject(got, "properties")
	require.NotNil(t, props, "properties should be present with ?include=properties")

	// Remove property
	removeStatus, removeBody, err := apiClient.Delete(itemCategoriesPath + "/" + id + "/properties/" + SeedPropertyID)
	require.NoError(t, err)
	requireStatus(t, 200, removeStatus, removeBody)

	// Verify property was removed — with no properties, the field is null per presenter logic
	getStatus2, getBody2, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"properties"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)

	got2 := parseJSON(getBody2)
	assert.Nil(t, got2["properties"], "properties should be null when category has no properties")

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

// --- Unit Group Management ---

func TestItemCategories_ChangeUnitGroup(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-chug")
	// Sellable Socks unit group — same type (quantity) as SeedUnitGroupID (Socks)
	sameTypeUnitGroupID := "ungp_1gf7a8200f8x8jjpq5a9kdrhd"

	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Change unit group to one with the same unit type
	changeStatus, changeBody, err := apiClient.Put(itemCategoriesPath+"/"+id+"/unit-groups/"+sameTypeUnitGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, changeStatus, changeBody)

	// Verify unit group changed via include
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, sameTypeUnitGroupID, jsonField(unitGroup, "id"))

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

func TestItemCategories_ChangeUnitGroupRejectsDifferentType(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-chug-reject")
	// Yarn unit group — type: mass, different from SeedUnitGroupID (quantity)
	differentTypeUnitGroupID := "ungp_01k0a51qxceydax5036pegvzzy"

	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Attempt to change to a unit group with a different unit type — should be rejected
	changeStatus, _, err := apiClient.Put(itemCategoriesPath+"/"+id+"/unit-groups/"+differentTypeUnitGroupID, nil)
	require.NoError(t, err)
	assert.True(t, changeStatus == 400 || changeStatus == 422,
		"Changing to a unit group with a different unit type should return 400 or 422, got %d", changeStatus)

	// Verify unit group was NOT changed
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	unitGroup := jsonObject(got, "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"), "unit group should remain unchanged after rejected change")

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

// ==========================================================================
// Bulk Upsert
// ==========================================================================

const itemCategoriesBulkUpsertPath = itemCategoriesPath + "/actions/bulk-upsert"

// bulkUpsertICInput builds a minimal item category entry for bulk-upsert payloads.
// The unit group is a fuzzy reference, so it is sent as an object keyed by id.
func bulkUpsertICInput(name string, typeCode string, unitGroupID string) map[string]any {
	return map[string]any{
		"name":       name,
		"type":       typeCode,
		"unit_group": map[string]any{"id": unitGroupID},
	}
}

func cleanupItemCategoryIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(itemCategoriesPath + "/" + id)
		}
	}
}

// posts a bulk upsert, requires the 202 job acknowledgment, and returns the completed job
func bulkUpsertItemCategoriesJob(t *testing.T, rows ...map[string]any) map[string]any {
	t.Helper()
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": rows,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// posts a bulk upsert, follows the job to completion, and returns the created/updated ids
func bulkUpsertItemCategoryIDs(t *testing.T, rows ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertItemCategoriesJob(t, rows...)
	require.NotEmpty(t, jsonArray(job, "results"), "a completed job must carry results")
	return jobResultIDs(job)
}

func TestItemCategories_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()
	name1 := uniqueName("e2e-bulic-a")
	name2 := uniqueName("e2e-bulic-b")

	createdIDs, updatedIDs := bulkUpsertItemCategoryIDs(t,
		bulkUpsertICInput(name1, "material_category", SeedUnitGroupID),
		bulkUpsertICInput(name2, "product_category", SeedUnitGroupID),
	)
	defer cleanupItemCategoryIDs(createdIDs)

	require.Len(t, createdIDs, 2, "should have 2 created IDs")
	for _, id := range createdIDs {
		assertIDFormat(t, id, "itcg")
	}
	assert.Empty(t, updatedIDs, "no updates expected")
}

func TestItemCategories_BulkUpsert_AllUpdates(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-upd")

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material_category", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	categoryID := createdIDs[0]

	// Same name → matches, updates type
	created, updated := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "product_category", SeedUnitGroupID))
	assert.Empty(t, created, "no creates expected on update")
	require.Len(t, updated, 1, "should have 1 updated ID")
	assert.Equal(t, categoryID, updated[0], "updated ID must match the originally created ID")
}

func TestItemCategories_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()
	existingName := uniqueName("e2e-bulic-mix-exist")
	newName := uniqueName("e2e-bulic-mix-new")

	seeded, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(existingName, "material_category", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(seeded)

	created, updated := bulkUpsertItemCategoryIDs(t,
		bulkUpsertICInput(existingName, "material_category", SeedUnitGroupID),
		bulkUpsertICInput(newName, "product_category", SeedUnitGroupID),
	)
	defer cleanupItemCategoryIDs(created)

	assert.Len(t, created, 1, "one new item category created")
	assert.Len(t, updated, 1, "one existing item category updated")
}

func TestItemCategories_BulkUpsert_ResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-shape")

	job := bulkUpsertItemCategoriesJob(t, bulkUpsertICInput(name, "material_category", SeedUnitGroupID))
	created, _ := jobResultIDs(job)
	defer cleanupItemCategoryIDs(created)

	assertObjectField(t, job, "job")
	assert.Equal(t, "bulkupsert", jsonField(job, "type"))
	_, hasResults := job["results"]
	assert.True(t, hasResults, "job must have a results field")
	_, hasErrors := job["errors"]
	assert.True(t, hasErrors, "job must have an errors field")
}

func TestItemCategories_BulkUpsert_Idempotency(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-idem")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"item_categories": []any{bulkUpsertICInput(name, "material_category", SeedUnitGroupID)},
	}

	status1, body1, err := apiClient.Post(itemCategoriesBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status1, body1)

	status2, body2, err := apiClient.Post(itemCategoriesBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status2, body2)

	jobID := jsonField(parseJSON(body1), "id")
	assert.Equal(t, jobID, jsonField(parseJSON(body2), "id"), "a replay must hand back the same job")

	created, _ := jobResultIDs(pollJobUntilTerminal(t, jobID))
	defer cleanupItemCategoryIDs(created)
	require.Len(t, created, 1)
}

func TestItemCategories_BulkUpsert_CreatedAppearsInList(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-list")

	ids, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material_category", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(ids)
	require.Len(t, ids, 1)
	categoryID := ids[0]

	// Fetch via GET by ID
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+categoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, "item_category", jsonField(got, "object"))
	assert.Equal(t, categoryID, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))
}

func TestItemCategories_BulkUpsert_UpdatePreservesID(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-preserve-id")

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material_category", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	originalID := createdIDs[0]

	// Update with the same name. The incoming type differs deliberately: a category's
	// type is immutable (changing it would strand existing items of the old type), so
	// the update must preserve the stored type — mirroring how unit groups treat type.
	_, updatedIDs := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "product_category", SeedUnitGroupID))
	require.Len(t, updatedIDs, 1)
	assert.Equal(t, originalID, updatedIDs[0], "ID must be stable across updates")

	// Verify the type was preserved, not overwritten.
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+originalID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "material_category", jsonField(got, "type"))
}

func TestItemCategories_BulkUpsert_CaseInsensitiveNameMatch(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-case")

	seeded, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(strings.ToLower(name), "material_category", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(seeded)

	// Upsert with uppercase name — should match the existing and update
	created, updated := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(strings.ToUpper(name), "product_category", SeedUnitGroupID))
	assert.Empty(t, created, "upper-case match should update, not create")
	assert.Len(t, updated, 1, "should have 1 updated ID")
}

func TestItemCategories_BulkUpsert_UpdatePreservesOmittedNotes(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-notes")
	notes := "bulk notes should survive omitted update"
	createInput := bulkUpsertICInput(name, "material_category", SeedUnitGroupID)
	createInput["notes"] = notes

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, createInput)
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	categoryID := createdIDs[0]

	_, updatedIDs := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "product_category", SeedUnitGroupID))
	require.Len(t, updatedIDs, 1)
	assert.Equal(t, categoryID, updatedIDs[0])

	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+categoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, notes, jsonField(got, "notes"), "bulk update should preserve notes when notes is omitted")
}

// checks the same-unit-type rule, which needs the existing row: it runs in the execute
// phase, so the job completes and the row lands in `errors` instead of failing the request
func TestItemCategories_BulkUpsert_RejectsDifferentTypeUnitGroup(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-ugtype")
	differentTypeUnitGroupID := "ungp_01k0a51qxceydax5036pegvzzy"

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material_category", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	categoryID := createdIDs[0]

	job := bulkUpsertItemCategoriesJob(t, bulkUpsertICInput(name, "material_category", differentTypeUnitGroupID))
	assert.Empty(t, jobResults(job), "an incompatible unit group must not be written")
	rowErrs := jsonArray(job, "errors")
	require.Len(t, rowErrs, 1, "the rejected row is recorded in errors")

	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+categoryID, url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	unitGroup := jsonObject(parseJSON(getBody), "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"), "unit group should remain unchanged after rejected bulk upsert")
}

// TestItemCategories_BulkUpsert_RejectsUnknownUnitGroup: the unit group is resolved for
// every row before any write, so one that does not resolve fails as a row-indexed
// validation error and the whole batch is atomic.
func TestItemCategories_BulkUpsert_RejectsUnknownUnitGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			bulkUpsertICInput(uniqueName("e2e-bulic-ugok"), "material_category", SeedUnitGroupID),
			bulkUpsertICInput(uniqueName("e2e-bulic-badug"), "material_category", "ungp_000000000000000000000000"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "item_categories[1].unit_group")
}

// TestItemCategories_BulkUpsert_ResolvesUnitGroupByName: the unit group is a fuzzy
// reference — it resolves by name, not only by id.
func TestItemCategories_BulkUpsert_ResolvesUnitGroupByName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-ugname")

	created, _ := bulkUpsertItemCategoryIDs(t, map[string]any{
		"name":       name,
		"type":       "material_category",
		"unit_group": map[string]any{"name": "socks"}, // by name, case-insensitive
	})
	defer cleanupItemCategoryIDs(created)
	require.Len(t, created, 1)

	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+created[0], url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	unitGroup := jsonObject(parseJSON(getBody), "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=unit_group")
	assert.Equal(t, SeedUnitGroupID, jsonField(unitGroup, "id"))
}

func TestItemCategories_BulkUpsert_EmptyList(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"empty list should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_BulkUpsert_TooMany(t *testing.T) {
	t.Parallel()
	categories := make([]any, 1001)
	for i := range categories {
		categories[i] = bulkUpsertICInput(uniqueName("e2e-bulic"), "material_category", SeedUnitGroupID)
	}
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{"item_categories": categories}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"1001 item categories should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_BulkUpsert_DuplicateNameInRequest(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-dupname")

	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			bulkUpsertICInput(name, "material_category", SeedUnitGroupID),
			bulkUpsertICInput(name, "product_category", SeedUnitGroupID),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"duplicate name in request should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "item_categories[1].name")
	assert.Contains(t, errObj["message"], "duplicate name")
}

func TestItemCategories_BulkUpsert_CaseInsensitiveDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-cidup")

	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			bulkUpsertICInput(strings.ToLower(name), "material_category", SeedUnitGroupID),
			bulkUpsertICInput(strings.ToUpper(name), "product_category", SeedUnitGroupID),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"case-insensitive duplicate name should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_BulkUpsert_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			map[string]any{"type": "material_category", "unit_group": map[string]any{"id": SeedUnitGroupID}},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_BulkUpsert_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			map[string]any{"name": uniqueName("e2e-bulic-notype"), "unit_group": map[string]any{"id": SeedUnitGroupID}},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing type should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_BulkUpsert_MissingUnitGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			map[string]any{"name": uniqueName("e2e-bulic-noug"), "type": "material_category"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing unit_group should return 400 or 422, got %d: %s", status, string(body))
}
