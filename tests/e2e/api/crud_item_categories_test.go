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

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedItemCategoryID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded item category (Socks) should appear in list")
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
	assert.Len(t, list.Data, 1)
}

func TestItemCategories_ListCursorPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough item categories for pagination test")
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
