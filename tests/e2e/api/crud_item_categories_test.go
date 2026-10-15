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
	assertListPageLen(t, itemCategoriesPath, url.Values{"limit": {"1"}}, 1)
}

func TestItemCategories_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, itemCategoriesPath, nil)
}

func TestItemCategories_ListFilterByType(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"material", "product"} {
		list, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"types[]": {typ}})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least 1 %s category", typ)
		for _, item := range list.Data {
			assert.Equal(t, typ, DataItemField(item, "type"))
		}
	}

	both, _, err := apiClient.GetList(itemCategoriesPath, url.Values{"types[]": {"material", "product"}, "limit": {"100"}})
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, item := range both.Data {
		seen[DataItemField(item, "type")] = true
	}
	assert.True(t, seen["material"] || seen["product"], "both types filter returns categories")

	status, body, err := apiClient.GetListRaw(itemCategoriesPath, url.Values{"type": {"material"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "the singular type filter is gone: %s", body)
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
		"name": name,
		"type": "material",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "item_category", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "material", jsonField(created, "type"))

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
	colorID := createPropertyForTest(t, uniqueName("e2e-itcg-allf-p1"))
	sizeID := createPropertyForTest(t, uniqueName("e2e-itcg-allf-p2"))

	// ── CREATE with all fields ──
	name := uniqueName("e2e-itcg-allf")
	createResp, err := apiClient.PostFull(itemCategoriesPath+"?include=properties", map[string]any{
		"name":         name,
		"type":         "material",
		"property_ids": []string{colorID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + id) })

	assert.Equal(t, "item_category", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "material", jsonField(got, "type"))
	assert.NotContains(t, got, "notes")
	assertNilField(t, got, "owner")
	assertNilField(t, got, "unit_group")
	assertNilField(t, got, "default_lot_size")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	assert.Equal(t, []string{colorID}, embeddedIDs(t, got, "properties"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-itcg-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+id+"?include=properties", map[string]any{
		"name":         updatedName,
		"property_ids": []string{sizeID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "material", jsonField(updated, "type"), "type should be preserved")
	assert.Equal(t, []string{sizeID}, embeddedIDs(t, updated, "properties"), "property_ids replaces the list")

	// An update without property_ids leaves them; an empty list clears them.
	patchStatus, patchBody, err = apiClient.Patch(itemCategoriesPath+"/"+id+"?include=properties", map[string]any{"name": updatedName}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, []string{sizeID}, embeddedIDs(t, parseJSON(patchBody), "properties"))

	patchStatus, patchBody, err = apiClient.Patch(itemCategoriesPath+"/"+id+"?include=properties", map[string]any{"property_ids": []string{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Empty(t, embeddedIDs(t, parseJSON(patchBody), "properties"))
}

// createPropertyForTest creates a property and deletes it when the test ends.
func createPropertyForTest(t *testing.T, name string) string {
	t.Helper()
	status, body, err := apiClient.Post(propertiesPath, map[string]any{"name": name}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { apiClient.Delete(propertiesPath + "/" + id) })
	return id
}

// embeddedIDs is the ids in an expanded list field, in order.
func embeddedIDs(t *testing.T, obj map[string]any, field string) []string {
	t.Helper()
	list := jsonObject(obj, field)
	require.NotNil(t, list, "%s should be expanded", field)
	assert.Equal(t, "list", list["object"])
	data, _ := list["data"].([]any)
	ids := make([]string, 0, len(data))
	for _, d := range data {
		ids = append(ids, d.(map[string]any)["id"].(string))
	}
	return ids
}

func TestItemCategories_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-shape")
	createResp, err := apiClient.PostFull(itemCategoriesPath, map[string]any{
		"name": name,
		"type": "product",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "item_category", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "product", jsonField(created, "type"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

func TestItemCategories_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"type": "material",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestItemCategories_CreateValidation_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name": uniqueName("e2e-itcg-notype"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing type should return 400 or 422, got %d: %s", status, string(body))
}

// The unit group is internal until items carry a stocking unit; a category created without one counts in each.
func TestItemCategories_CreateWithoutUnitGroupCountsInEach(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath+"?include=unit_group", map[string]any{
		"name": uniqueName("e2e-itcg-noug"),
		"type": "material",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + jsonField(got, "id")) })
	assert.Equal(t, "each_group", jsonField(jsonObject(got, "unit_group"), "id"))
}

func TestItemCategories_CreateValidation_PropertyIDs(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":         uniqueName("e2e-itcg-badprop"),
		"type":         "material",
		"property_ids": []string{"pp_000000000000"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "property_ids[0]")

	status, body, err = apiClient.Post(itemCategoriesPath, map[string]any{
		"name": uniqueName("e2e-itcg-oldtype"),
		"type": "material_category",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
}

// A repeated property id attaches the property once.
func TestItemCategories_CreateDeduplicatesPropertyIDs(t *testing.T) {
	t.Parallel()
	propID := createPropertyForTest(t, uniqueName("e2e-itcg-dup-p"))
	status, body, err := apiClient.Post(itemCategoriesPath+"?include=properties", map[string]any{
		"name":         uniqueName("e2e-itcg-dup"),
		"type":         "product",
		"property_ids": []string{propID, propID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + jsonField(got, "id")) })
	assert.Equal(t, []string{propID}, embeddedIDs(t, got, "properties"))
}

func TestItemCategories_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-itcg")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name": name,
		"type": "material",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name": name,
		"type": "material",
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
		"name": name,
		"type": "material",
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
	assert.Equal(t, "material", jsonField(patched, "type"), "type should be preserved when only name is updated")

	apiClient.Delete(itemCategoriesPath + "/" + id)
}

// notes left the category: forge refuses the field, and preview.9 callers have it dropped.
func TestItemCategories_NotesAreGone(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-pnotes")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name": name,
		"type": "product",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + id) })

	status, body, err := apiClient.Patch(itemCategoriesPath+"/"+id, map[string]any{"notes": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")

	preview := apiClient.WithAPIVersion(preview9Version)
	status, body, err = preview.Patch(itemCategoriesPath+"/"+id, map[string]any{"notes": "x", "name": name + "-p9"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, name+"-p9", jsonField(got, "name"))
	assert.Contains(t, got, "notes")
	assert.Nil(t, got["notes"])
	assert.Equal(t, "product_category", jsonField(got, "type"))
}

func TestItemCategories_UpdateSeededCategoryPreservesType(t *testing.T) {
	// Not parallel: renames the seeded item category, whose name other tests read.
	newName := uniqueName("e2e-itcg-seedupd")
	patchStatus, patchBody, err := apiClient.Patch(itemCategoriesPath+"/"+SeedItemCategoryID, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "product", jsonField(patched, "type"), "type should be preserved")

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
		"name": name,
		"type": "material",
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
		"name": name,
		"type": "material",
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
			"name": name,
			"type": "material",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(itemCategoriesPath + "/" + id)

		assertObjectField(t, got, "item_category")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "material", jsonField(got, "type"))
		assert.NotContains(t, got, "notes")
		assertNilField(t, got, "owner")
		assertNilField(t, got, "properties")
		assertNilField(t, got, "unit_group")
		assertNilField(t, got, "default_lot_size")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-itcg-pres")
		createStatus, createBody, err := apiClient.Post(itemCategoriesPath+"?include=unit_group", map[string]any{
			"name":          name,
			"type":          "material",
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
		assert.Equal(t, "material", jsonField(got, "type"), "type should be preserved")
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

// The attach and detach routes are gone from forge; preview.9 callers still reach them.
func TestItemCategories_AddAndRemoveProperty(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-prop")
	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name": name,
		"type": "material",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + id) })

	status, _, err := apiClient.Put(itemCategoriesPath+"/"+id+"/properties/"+SeedPropertyID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "forge has no attach route")

	preview := apiClient.WithAPIVersion(preview9Version)
	addStatus, addBody, err := preview.Put(itemCategoriesPath+"/"+id+"/properties/"+SeedPropertyID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, addStatus, addBody)

	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"properties"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, []string{SeedPropertyID}, embeddedIDs(t, parseJSON(getBody), "properties"))

	removeStatus, removeBody, err := preview.Delete(itemCategoriesPath + "/" + id + "/properties/" + SeedPropertyID)
	require.NoError(t, err)
	requireStatus(t, 200, removeStatus, removeBody)

	getStatus2, getBody2, err := apiClient.GetListRaw(itemCategoriesPath+"/"+id, url.Values{"include": {"properties"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Empty(t, embeddedIDs(t, parseJSON(getBody2), "properties"), "an included empty list is an empty list, not null")
}

// The category's embedded properties stop at ten and link to the rest.
func TestItemCategories_IncludePropertiesPagesAfterTen(t *testing.T) {
	t.Parallel()
	ids := make([]string, 11)
	for i := range ids {
		ids[i] = createPropertyForTest(t, uniqueName("e2e-itcg-page-p"))
	}
	status, body, err := apiClient.Post(itemCategoriesPath+"?include=properties", map[string]any{
		"name":         uniqueName("e2e-itcg-page"),
		"type":         "product",
		"property_ids": ids,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	got := parseJSON(body)
	id := jsonField(got, "id")
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + id) })

	props := jsonObject(got, "properties")
	require.NotNil(t, props)
	assert.Len(t, props["data"], 10)
	next := jsonField(jsonObject(props, "page_info"), "next_page_url")
	require.NotEmpty(t, next)

	nextURL, err := url.Parse(next)
	require.NoError(t, err)
	assert.Equal(t, propertiesPath, nextURL.Path)
	list, _, err := apiClient.GetList(propertiesPath, url.Values{"item_category_ids[]": {id}, "limit": {"100"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 11, "the link's filter lists every property of the category")
}

// Deleting a category items use is refused; one nothing uses is deleted with its property links.
func TestItemCategories_DeleteInUse(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(itemCategoriesPath + "/" + SeedMaterialCategoryID)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_in_use", "")
	assert.Contains(t, jsonField(errObj, "message"), "items")

	status, body, err = apiClient.GetListRaw(itemCategoriesPath+"/"+SeedMaterialCategoryID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// --- Unit Group Management ---

func TestItemCategories_ChangeUnitGroup(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-itcg-chug")
	// Sellable Socks unit group — same type (quantity) as SeedUnitGroupID (Socks)
	sameTypeUnitGroupID := "ungp_1gf7a8200f8x8jjpq5a9kdrhd"

	createStatus, createBody, err := apiClient.Post(itemCategoriesPath, map[string]any{
		"name":          name,
		"type":          "material",
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
		"type":          "material",
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
	assert.Equal(t, "async_job", jsonField(m, "object"), "202 returns the async job")
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
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

func TestItemCategories_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()
	name1 := uniqueName("e2e-bulic-a")
	name2 := uniqueName("e2e-bulic-b")

	createdIDs, updatedIDs := bulkUpsertItemCategoryIDs(t,
		bulkUpsertICInput(name1, "material", SeedUnitGroupID),
		bulkUpsertICInput(name2, "product", SeedUnitGroupID),
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

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	categoryID := createdIDs[0]

	// Same name → matches, updates type
	created, updated := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "product", SeedUnitGroupID))
	assert.Empty(t, created, "no creates expected on update")
	require.Len(t, updated, 1, "should have 1 updated ID")
	assert.Equal(t, categoryID, updated[0], "updated ID must match the originally created ID")
}

func TestItemCategories_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()
	existingName := uniqueName("e2e-bulic-mix-exist")
	newName := uniqueName("e2e-bulic-mix-new")

	seeded, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(existingName, "material", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(seeded)

	created, updated := bulkUpsertItemCategoryIDs(t,
		bulkUpsertICInput(existingName, "material", SeedUnitGroupID),
		bulkUpsertICInput(newName, "product", SeedUnitGroupID),
	)
	defer cleanupItemCategoryIDs(created)

	assert.Len(t, created, 1, "one new item category created")
	assert.Len(t, updated, 1, "one existing item category updated")
}

func TestItemCategories_BulkUpsert_ResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-shape")

	job := bulkUpsertItemCategoriesJob(t, bulkUpsertICInput(name, "material", SeedUnitGroupID))
	created, _ := jobResultIDs(job)
	defer cleanupItemCategoryIDs(created)

	assertObjectField(t, job, "async_job")
	assert.Equal(t, "bulk_upsert", jsonField(job, "type"))
	assert.Equal(t, "item_category", jsonField(job, "resource_type"), "the job must say what it operated on")
	_, hasResults := job["results"]
	assert.True(t, hasResults, "job must have a results field")
	_, hasError := job["error"]
	assert.True(t, hasError, "job must have an error field")
	assert.Nil(t, job["error"], "a job that completed did not fail as a whole")
	// Every row lands in results now; the separate errors array is gone.
	_, hasErrors := job["errors"]
	assert.False(t, hasErrors, "row failures belong in results, not a separate errors array")
}

func TestItemCategories_BulkUpsert_Idempotency(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-idem")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"item_categories": []any{bulkUpsertICInput(name, "material", SeedUnitGroupID)},
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

	ids, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material", SeedUnitGroupID))
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

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	originalID := createdIDs[0]

	// Update with the same name. The incoming type differs deliberately: a category's
	// type is immutable (changing it would strand existing items of the old type), so
	// the update must preserve the stored type — mirroring how unit groups treat type.
	_, updatedIDs := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "product", SeedUnitGroupID))
	require.Len(t, updatedIDs, 1)
	assert.Equal(t, originalID, updatedIDs[0], "ID must be stable across updates")

	// Verify the type was preserved, not overwritten.
	getStatus, getBody, err := apiClient.GetListRaw(itemCategoriesPath+"/"+originalID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, "material", jsonField(got, "type"))
}

func TestItemCategories_BulkUpsert_CaseInsensitiveNameMatch(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-case")

	seeded, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(strings.ToLower(name), "material", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(seeded)

	// Upsert with uppercase name — should match the existing and update
	created, updated := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(strings.ToUpper(name), "product", SeedUnitGroupID))
	assert.Empty(t, created, "upper-case match should update, not create")
	assert.Len(t, updated, 1, "should have 1 updated ID")
}

// A bulk row naming no unit group keeps the matched category's and creates new ones counting in each.
func TestItemCategories_BulkUpsert_UnitGroupIsOptional(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-noug")
	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	newName := uniqueName("e2e-bulic-noug-new")
	created, updated := bulkUpsertItemCategoryIDs(t,
		map[string]any{"name": name, "type": "material"},
		map[string]any{"name": newName, "type": "product", "property_names": []string{SeedPropertyName}},
	)
	defer cleanupItemCategoryIDs(created)
	require.Len(t, updated, 1)
	require.Len(t, created, 1)

	status, body, err := apiClient.GetListRaw(itemCategoriesPath+"/"+createdIDs[0], url.Values{"include": {"unit_group"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, SeedUnitGroupID, jsonField(jsonObject(parseJSON(body), "unit_group"), "id"), "the update keeps its unit group")

	status, body, err = apiClient.GetListRaw(itemCategoriesPath+"/"+created[0], url.Values{"include": {"unit_group,properties"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "each_group", jsonField(jsonObject(got, "unit_group"), "id"))
	assert.Equal(t, []string{SeedPropertyID}, embeddedIDs(t, got, "properties"))
}

// checks the same-unit-type rule, which needs the existing row: it runs in the execute
// phase, so the job completes and the row lands in `errors` instead of failing the request
func TestItemCategories_BulkUpsert_RejectsDifferentTypeUnitGroup(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-ugtype")
	differentTypeUnitGroupID := "ungp_01k0a51qxceydax5036pegvzzy"

	createdIDs, _ := bulkUpsertItemCategoryIDs(t, bulkUpsertICInput(name, "material", SeedUnitGroupID))
	defer cleanupItemCategoryIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	categoryID := createdIDs[0]

	job := bulkUpsertItemCategoriesJob(t, bulkUpsertICInput(name, "material", differentTypeUnitGroupID))
	assert.Empty(t, jobWrittenResults(job), "an incompatible unit group must not be written")
	rowErrs := jobErrors(job)
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
			bulkUpsertICInput(uniqueName("e2e-bulic-ugok"), "material", SeedUnitGroupID),
			bulkUpsertICInput(uniqueName("e2e-bulic-badug"), "material", "ungp_000000000000000000000000"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
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
		"type":       "material",
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
		categories[i] = bulkUpsertICInput(uniqueName("e2e-bulic"), "material", SeedUnitGroupID)
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
			bulkUpsertICInput(name, "material", SeedUnitGroupID),
			bulkUpsertICInput(name, "product", SeedUnitGroupID),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 422,
		"duplicate name in request should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "item_categories[1].name")
	assert.Contains(t, errorMessages(errObj), "duplicate name")
}

func TestItemCategories_BulkUpsert_CaseInsensitiveDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bulic-cidup")

	status, body, err := apiClient.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{
			bulkUpsertICInput(strings.ToLower(name), "material", SeedUnitGroupID),
			bulkUpsertICInput(strings.ToUpper(name), "product", SeedUnitGroupID),
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
			map[string]any{"type": "material", "unit_group": map[string]any{"id": SeedUnitGroupID}},
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

// preview.9 callers send the old type names and notes; the rows are upgraded.
func TestItemCategories_BulkUpsert_Preview9TypeNames(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	name := uniqueName("e2e-bulic-p9")
	status, body, err := preview.Post(itemCategoriesBulkUpsertPath, map[string]any{
		"item_categories": []any{map[string]any{"name": name, "type": "material_category", "unit_group": map[string]any{"id": SeedUnitGroupID}, "notes": "old"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)
	created, _ := jobResultIDs(pollJobUntilTerminal(t, jsonField(parseJSON(body), "id")))
	defer cleanupItemCategoryIDs(created)
	require.Len(t, created, 1)

	status, body, err = preview.GetListRaw(itemCategoriesPath, url.Values{"type": {"material_category"}, "q": {name}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data, _ := parseJSON(body)["data"].([]any)
	require.Len(t, data, 1)
	assert.Equal(t, "material_category", data[0].(map[string]any)["type"])
}
