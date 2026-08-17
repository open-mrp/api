//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const locationsPath = "/v1/operations/locations"
const locationTypesPath = "/v1/operations/location-types"

// ==========================================================================
// Location Types (read-only)
// ==========================================================================

func TestLocationTypes_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(locationTypesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 6, "Should have at least 6 location types (building, section, aisle, rack, shelf, bin)")
}

func TestLocationTypes_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(locationTypesPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "location_type", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "code"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestLocationTypes_GetByCode(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(locationTypesPath+"/building", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, "location_type", jsonField(got, "object"))
	assert.Equal(t, "building", jsonField(got, "code"))
	assert.NotEmpty(t, jsonField(got, "id"))
	assert.NotEmpty(t, jsonField(got, "name"))
}

func TestLocationTypes_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(locationTypesPath+"/nonexistent_type", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// ==========================================================================
// Locations — List
// ==========================================================================

func TestLocations_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(locationsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded location")

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assertListContainsID(t, locationsPath, nil, SeedLocationID)
}

func TestLocations_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(locationsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "location", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestLocations_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(locationsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestLocations_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, locationsPath, nil)
}

func TestLocations_ListSearchByName(t *testing.T) {
	t.Parallel()
	// First get the seed location name to search for.
	_, body, err := apiClient.GetListRaw(locationsPath+"/"+SeedLocationID, nil)
	require.NoError(t, err)
	seedName := jsonField(parseJSON(body), "name")
	require.NotEmpty(t, seedName)

	// Search using part of the seed location's name.
	searchTerm := seedName
	if len(seedName) > 3 {
		searchTerm = seedName[:3]
	}
	list, _, err := apiClient.GetList(locationsPath, url.Values{"q": {searchTerm}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for '%s' should return at least 1 result", searchTerm)

	lowerTerm := strings.ToLower(searchTerm)
	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), lowerTerm),
			"Search result %q should contain %q", name, searchTerm,
		)
	}
}

func TestLocations_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(locationsPath, url.Values{"q": {"zzzznotaplace99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// ==========================================================================
// Locations — Get
// ==========================================================================

func TestLocations_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(locationsPath+"/"+SeedLocationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedLocationID, jsonField(got, "id"))
	assert.Equal(t, "location", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestLocations_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(locationsPath+"/lc_000000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// ==========================================================================
// Locations — CRUD
// ==========================================================================

func TestLocations_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-loc")

	// Create
	createResp, err := apiClient.PostFull(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "location", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "building", jsonField(created, "type"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-loc-upd")
	patchStatus, patchBody, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(locationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(locationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(locationsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- Create ---

func TestLocations_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE parent ──
	parentName := uniqueName("e2e-loc-allf-p")
	pStatus, pBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, pStatus, pBody)
	parentID := jsonField(parseJSON(pBody), "id")
	defer apiClient.Delete(locationsPath + "/" + parentID)

	// ── CREATE child with parent_id ──
	name := uniqueName("e2e-loc-allf")
	createResp, err := apiClient.PostFull(locationsPath, map[string]any{
		"name":      name,
		"type":      "section",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(locationsPath + "/" + id)

	assert.Equal(t, "location", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "section", jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	// Verify parent via include
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	parent := jsonObject(parseJSON(getBody), "parent")
	require.NotNil(t, parent, "parent must be set after create with parent_id")
	assert.Equal(t, parentID, jsonField(parent, "id"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-loc-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"name": updatedName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "section", jsonField(updated, "type"), "type_code should be preserved")

	// Verify parent still set after name-only update
	getStatus2, getBody2, err := apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	updParent := jsonObject(parseJSON(getBody2), "parent")
	require.NotNil(t, updParent, "parent should be preserved after name-only update")
	assert.Equal(t, parentID, jsonField(updParent, "id"))
}

func TestLocations_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-loc-shape")
	createResp, err := apiClient.PostFull(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "location", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "building", jsonField(created, "type"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(locationsPath + "/" + id)
}

func TestLocations_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_CreateValidation_MissingTypeCode(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsPath, map[string]any{
		"name": uniqueName("e2e-loc-notype"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing type_code should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-loc")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(locationsPath + "/" + id1)
}

// --- Update ---

func TestLocations_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-loc-pname")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "section",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-loc-pname2")
	patchStatus, patchBody, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "section", jsonField(patched, "type"), "type_code should be preserved when only name is updated")

	apiClient.Delete(locationsPath + "/" + id)
}

func TestLocations_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-loc-idem-upd")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-loc-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(locationsPath + "/" + id)
}

func TestLocations_UpdateEmptyBodyRejected(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-loc-empty")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	patchStatus, _, err := apiClient.Patch(locationsPath+"/"+id, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, patchStatus, "Empty PATCH body should return 400")

	apiClient.Delete(locationsPath + "/" + id)
}

// --- Delete ---

func TestLocations_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-loc-deldel")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": name,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(locationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	status2, _, err := apiClient.Delete(locationsPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted location should return 404 or 410, got %d", status2)
}

func TestLocations_DeleteWithChildrenFails(t *testing.T) {
	t.Parallel()

	// Create parent
	parentName := uniqueName("e2e-loc-parent")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	parentID := jsonField(parseJSON(createBody), "id")

	// Create child with parent
	childName := uniqueName("e2e-loc-child")
	createStatus2, createBody2, err := apiClient.Post(locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus2, createBody2)
	childID := jsonField(parseJSON(createBody2), "id")

	// Attempt to delete parent — should fail because it has children
	delStatus, _, err := apiClient.Delete(locationsPath + "/" + parentID)
	require.NoError(t, err)
	assert.True(t, delStatus == 400 || delStatus == 409 || delStatus == 422,
		"Deleting a location with children should fail, got %d", delStatus)

	// Cleanup: delete child then parent
	apiClient.Delete(locationsPath + "/" + childID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

// ==========================================================================
// Locations — Parent/Child Relationships
// ==========================================================================

func TestLocations_CreateWithParent(t *testing.T) {
	t.Parallel()

	// Create parent
	parentName := uniqueName("e2e-loc-par")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	parentID := jsonField(parseJSON(createBody), "id")

	// Create child with parent_id
	childName := uniqueName("e2e-loc-cld")
	childResp, err := apiClient.PostFull(locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, childResp.StatusCode, childResp.Body)
	childID := jsonField(parseJSON(childResp.Body), "id")
	assertCreatedLocation(t, childResp.Header, childID)

	// Get child with include=parent
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+childID, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	parent := jsonObject(got, "parent")
	require.NotNil(t, parent, "parent should be present with ?include=parent")
	assert.Equal(t, parentID, jsonField(parent, "id"))
	assert.Equal(t, "location", jsonField(parent, "object"))

	// Cleanup
	apiClient.Delete(locationsPath + "/" + childID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

func TestLocations_CreateWithChildren(t *testing.T) {
	t.Parallel()

	// Create two children first (without parent)
	child1Name := uniqueName("e2e-loc-c1")
	c1Status, c1Body, err := apiClient.Post(locationsPath, map[string]any{
		"name": child1Name,
		"type": "section",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, c1Status, c1Body)
	child1ID := jsonField(parseJSON(c1Body), "id")

	child2Name := uniqueName("e2e-loc-c2")
	c2Status, c2Body, err := apiClient.Post(locationsPath, map[string]any{
		"name": child2Name,
		"type": "section",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, c2Status, c2Body)
	child2ID := jsonField(parseJSON(c2Body), "id")

	// Create parent with child_ids
	parentName := uniqueName("e2e-loc-pwc")
	parentResp, err := apiClient.PostFull(locationsPath, map[string]any{
		"name":      parentName,
		"type":      "building",
		"child_ids": []string{child1ID, child2ID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, parentResp.StatusCode, parentResp.Body)
	parentID := jsonField(parseJSON(parentResp.Body), "id")
	assertCreatedLocation(t, parentResp.Header, parentID)

	// Get parent with include=children
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+parentID, url.Values{"include": {"children"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	children := jsonObject(got, "children")
	require.NotNil(t, children, "children should be present with ?include=children")
	assert.Equal(t, "list", jsonField(children, "object"))

	// Cleanup
	apiClient.Delete(locationsPath + "/" + child1ID)
	apiClient.Delete(locationsPath + "/" + child2ID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

func TestLocations_UpdateClearParent(t *testing.T) {
	t.Parallel()

	// Create parent
	parentName := uniqueName("e2e-loc-clrp")
	createStatus, createBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	parentID := jsonField(parseJSON(createBody), "id")

	// Create child with parent
	childName := uniqueName("e2e-loc-clrc")
	c2Status, c2Body, err := apiClient.Post(locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, c2Status, c2Body)
	childID := jsonField(parseJSON(c2Body), "id")

	// Clear parent by sending null
	patchStatus, patchBody, err := apiClient.Patch(locationsPath+"/"+childID, map[string]any{
		"parent_id": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	// Verify parent is gone
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+childID, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Nil(t, got["parent"], "parent should be null after sending parent_id: null")

	// Cleanup
	apiClient.Delete(locationsPath + "/" + childID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

// ==========================================================================
// Locations — Expandable Fields
// ==========================================================================

func TestLocations_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(locationsPath+"/"+SeedLocationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["parent"], "parent should be null without ?include=parent")
	assert.Nil(t, got["children"], "children should be null without ?include=children")

	// Test on List
	list, _, err := apiClient.GetList(locationsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["parent"], "parent should be null on list items without ?include=parent")
		assert.Nil(t, m["children"], "children should be null on list items without ?include=children")
	}
}

func TestLocations_IncludeChildren(t *testing.T) {
	t.Parallel()

	// Create parent with a child to guarantee children are present
	parentName := uniqueName("e2e-loc-inclch")
	pStatus, pBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, pStatus, pBody)
	parentID := jsonField(parseJSON(pBody), "id")

	childName := uniqueName("e2e-loc-inclc2")
	cStatus, cBody, err := apiClient.Post(locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, cStatus, cBody)
	childID := jsonField(parseJSON(cBody), "id")

	// Get parent with include=children
	status, body, err := apiClient.GetListRaw(locationsPath+"/"+parentID, url.Values{"include": {"children"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	children := jsonObject(got, "children")
	require.NotNil(t, children, "children should be present with ?include=children")
	assert.Equal(t, "list", jsonField(children, "object"))

	// Cleanup
	apiClient.Delete(locationsPath + "/" + childID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

// ==========================================================================
// Locations — Bulk Upsert
// ==========================================================================

const locationsBulkUpsertPath = "/v1/operations/locations/actions/bulk-upsert"

// bulkUpsertLocInput builds a minimal location entry for bulk-upsert payloads.
func bulkUpsertLocInput(name, typeCode string) map[string]any {
	return map[string]any{
		"name": name,
		"type": typeCode,
	}
}

// cleanupLocationIDs deletes the given locations.
func cleanupLocationIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(locationsPath + "/" + id)
		}
	}
}

// acceptBulkUpsertLocations posts a bulk upsert, requires the 202 job acknowledgment, and
// returns the job's ID to poll.
func acceptBulkUpsertLocations(t *testing.T, locations ...map[string]any) string {
	t.Helper()
	rows := make([]any, len(locations))
	for i, l := range locations {
		rows[i] = l
	}
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{"locations": rows}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")
	return jobID
}

// bulkUpsertLocationsJob posts a bulk upsert and returns the completed job. Bulk upsert is
// partial-success: a row that fails against existing rows (a bad parent/child link) is
// recorded in the job's `errors` field, not failed — the job completes.
func bulkUpsertLocationsJob(t *testing.T, locations ...map[string]any) map[string]any {
	t.Helper()
	job := pollJobUntilTerminal(t, acceptBulkUpsertLocations(t, locations...))
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// bulkUpsertLocations posts a bulk upsert, follows the job to completion, and returns the
// created/updated location IDs from its results.
func bulkUpsertLocations(t *testing.T, locations ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertLocationsJob(t, locations...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

func TestLocations_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	name1 := uniqueName("e2e-buloc")
	name2 := uniqueName("e2e-buloc")

	createdIDs, updatedIDs := bulkUpsertLocations(t,
		bulkUpsertLocInput(name1, "building"),
		bulkUpsertLocInput(name2, "section"),
	)
	defer cleanupLocationIDs(createdIDs)

	require.Len(t, createdIDs, 2, "should have 2 created IDs")
	for _, id := range createdIDs {
		assertIDFormat(t, id, "lc")
	}
	assert.Empty(t, updatedIDs, "no updates expected")
}

func TestLocations_BulkUpsert_AllUpdates(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-buloc-upd")

	// Create via bulk upsert first
	createdIDs, _ := bulkUpsertLocations(t, bulkUpsertLocInput(name, "building"))
	defer cleanupLocationIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	locationID := createdIDs[0]

	// Upsert again with the same name but different type_code → update
	created, updated := bulkUpsertLocations(t, bulkUpsertLocInput(name, "section"))
	assert.Empty(t, created, "no creates expected")
	require.Len(t, updated, 1, "should have 1 updated ID")
	assert.Equal(t, locationID, updated[0], "updated ID must match the originally created ID")

	// Verify the type_code was updated
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+locationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "section", jsonField(parseJSON(getBody), "type"))
}

func TestLocations_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-buloc-mix-ex")
	newName := uniqueName("e2e-buloc-mix-new")

	// Seed an existing location
	seeded, _ := bulkUpsertLocations(t, bulkUpsertLocInput(existingName, "building"))
	defer cleanupLocationIDs(seeded)

	// Mix: update existing + create new
	created, updated := bulkUpsertLocations(t,
		bulkUpsertLocInput(existingName, "building"),
		bulkUpsertLocInput(newName, "section"),
	)
	defer cleanupLocationIDs(created)

	assert.Len(t, created, 1, "one new location created")
	assert.Len(t, updated, 1, "one existing location updated")
}

func TestLocations_BulkUpsert_ResponseShape(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-buloc-shape")

	// 202 returns the canonical job resource.
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{bulkUpsertLocInput(name, "bin")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	require.NotNil(t, m, "response body should parse as JSON")
	assertObjectField(t, m, "job")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID)

	// The completed job carries a row-indexed result per created/updated location.
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))
	ids, _ := jobResultIDs(job)
	require.Len(t, ids, 1)
	entry := jobResults(job)[0]
	assert.Equal(t, float64(0), entry["index"])
	assert.Equal(t, "created", entry["status"])
	assertIDFormat(t, ids[0], "lc")
	defer cleanupLocationIDs([]string{ids[0]})
}

func TestLocations_BulkUpsert_Idempotent(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-buloc-idem")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"locations": []any{bulkUpsertLocInput(name, "building")},
	}

	status1, body1, err := apiClient.Post(locationsBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status1, body1)

	status2, body2, err := apiClient.Post(locationsBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status2, body2)

	// Replay must return the identical job acknowledgment, not raise a second one.
	assert.JSONEq(t, string(body1), string(body2), "idempotent replay must return the same job")

	job := pollJobUntilTerminal(t, jsonField(parseJSON(body1), "id"))
	require.Equal(t, "completed", jsonField(job, "status"))
	created, _ := jobResultIDs(job)
	require.Len(t, created, 1)
	defer cleanupLocationIDs([]string{created[0]})
}

func TestLocations_BulkUpsert_CreatedLocationAppearsInList(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-buloc-list")

	createdIDs, _ := bulkUpsertLocations(t, bulkUpsertLocInput(name, "bin"))
	defer cleanupLocationIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	locationID := createdIDs[0]

	// Fetch via GET by ID
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+locationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, locationID, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "bin", jsonField(got, "type"))
}

func TestLocations_BulkUpsert_WithParentID(t *testing.T) {
	t.Parallel()

	// Create parent location first (standalone create)
	parentName := uniqueName("e2e-buloc-par")
	pStatus, pBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, pStatus, pBody)
	parentID := jsonField(parseJSON(pBody), "id")
	defer apiClient.Delete(locationsPath + "/" + parentID)

	// Bulk upsert a child that references the parent by ID
	childName := uniqueName("e2e-buloc-cld")
	createdIDs, _ := bulkUpsertLocations(t, map[string]any{
		"name":   childName,
		"type":   "section",
		"parent": map[string]any{"id": parentID},
	})
	defer cleanupLocationIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	childID := createdIDs[0]

	// Verify the parent link is set
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+childID, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	parent := jsonObject(parseJSON(getBody), "parent")
	require.NotNil(t, parent, "parent should be set after bulk upsert with parent")
	assert.Equal(t, parentID, jsonField(parent, "id"))
	assert.Equal(t, "location", jsonField(parent, "object"))
}

// TestLocations_BulkUpsert_ResolvesParentByName: the parent is a fuzzy reference — it
// resolves by name, not only by id.
func TestLocations_BulkUpsert_ResolvesParentByName(t *testing.T) {
	t.Parallel()

	parentName := uniqueName("e2e-buloc-pname")
	pStatus, pBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, pStatus, pBody)
	parentID := jsonField(parseJSON(pBody), "id")
	defer apiClient.Delete(locationsPath + "/" + parentID)

	createdIDs, _ := bulkUpsertLocations(t, map[string]any{
		"name":   uniqueName("e2e-buloc-cname"),
		"type":   "section",
		"parent": map[string]any{"name": strings.ToUpper(parentName)}, // case-insensitive
	})
	defer cleanupLocationIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+createdIDs[0], url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	parent := jsonObject(parseJSON(getBody), "parent")
	require.NotNil(t, parent, "parent should be set after resolving the ref by name")
	assert.Equal(t, parentID, jsonField(parent, "id"))
}

// TestLocations_BulkUpsert_RejectsUnknownParent: a parent that does not resolve fails as a
// row-indexed validation error before anything is written, and the batch is atomic.
func TestLocations_BulkUpsert_RejectsUnknownParent(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{
			map[string]any{"name": uniqueName("e2e-buloc-pok"), "type": "building"},
			map[string]any{
				"name":   uniqueName("e2e-buloc-pbad"),
				"type":   "section",
				"parent": map[string]any{"id": "loc_does_not_exist_00000"},
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	// Unresolvable references fail in the synchronous accept phase, all-or-nothing: a
	// row-indexed 400 before any job is raised, so the valid row is never written.
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "locations[1].parent")
}

func TestLocations_BulkUpsert_WithChildIDs(t *testing.T) {
	t.Parallel()

	// Create two child locations first
	child1Name := uniqueName("e2e-buloc-c1")
	c1Status, c1Body, err := apiClient.Post(locationsPath, map[string]any{
		"name": child1Name,
		"type": "section",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, c1Status, c1Body)
	child1ID := jsonField(parseJSON(c1Body), "id")

	child2Name := uniqueName("e2e-buloc-c2")
	c2Status, c2Body, err := apiClient.Post(locationsPath, map[string]any{
		"name": child2Name,
		"type": "section",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, c2Status, c2Body)
	child2ID := jsonField(parseJSON(c2Body), "id")

	// Bulk upsert a parent that claims the two children by id
	parentName := uniqueName("e2e-buloc-pwc")
	createdIDs, _ := bulkUpsertLocations(t, map[string]any{
		"name":     parentName,
		"type":     "building",
		"children": []any{map[string]any{"id": child1ID}, map[string]any{"id": child2ID}},
	})
	require.Len(t, createdIDs, 1)
	parentID := createdIDs[0]

	// Verify children now have the parent set
	for _, childID := range []string{child1ID, child2ID} {
		getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+childID, url.Values{"include": {"parent"}})
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)

		parent := jsonObject(parseJSON(getBody), "parent")
		require.NotNil(t, parent, "child %s should have parent set after bulk upsert with child_ids", childID)
		assert.Equal(t, parentID, jsonField(parent, "id"))
	}

	// Cleanup: children first (parent has children so can't delete parent before children)
	apiClient.Delete(locationsPath + "/" + child1ID)
	apiClient.Delete(locationsPath + "/" + child2ID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

func TestLocations_BulkUpsert_DeduplicatesLinks(t *testing.T) {
	t.Parallel()

	// Create two existing locations
	warehouseName := uniqueName("e2e-buloc-ded-wh")
	wStatus, wBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": warehouseName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, wStatus, wBody)
	warehouseID := jsonField(parseJSON(wBody), "id")

	binName := uniqueName("e2e-buloc-ded-bn")
	bStatus, bBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": binName,
		"type": "bin",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, bStatus, bBody)
	binID := jsonField(parseJSON(bBody), "id")

	// Bulk upsert both: warehouse names bin as child; bin names warehouse as parent.
	// The same link is described from both sides; the per-row link writes are idempotent,
	// so both rows apply cleanly and the final link is the single correct one.
	created, updated := bulkUpsertLocations(t,
		map[string]any{
			"name":     warehouseName,
			"type":     "building",
			"children": []any{map[string]any{"id": binID}},
		},
		map[string]any{
			"name":   binName,
			"type":   "bin",
			"parent": map[string]any{"id": warehouseID},
		},
	)

	assert.Len(t, updated, 2, "both locations should be updated")
	assert.Empty(t, created)

	// Verify the link is correct: bin's parent should be warehouse
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+binID, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	parent := jsonObject(parseJSON(getBody), "parent")
	require.NotNil(t, parent, "bin should have warehouse as parent")
	assert.Equal(t, warehouseID, jsonField(parent, "id"))

	// Cleanup
	apiClient.Delete(locationsPath + "/" + binID)
	apiClient.Delete(locationsPath + "/" + warehouseID)
}

// A self-contained hierarchy import: every parent/child reference names a sibling defined
// in the same batch (by name), and the whole tree — Warehouse → Aisles → Bins — is created
// in one request. This is the shape a spreadsheet import produces.
func TestLocations_BulkUpsert_IntraBatchHierarchy(t *testing.T) {
	t.Parallel()

	wh := uniqueName("e2e-hier-wh")
	a1 := uniqueName("e2e-hier-a1")
	a2 := uniqueName("e2e-hier-a2")
	b1 := uniqueName("e2e-hier-b1")
	b2 := uniqueName("e2e-hier-b2")

	created, updated := bulkUpsertLocations(t,
		map[string]any{"name": wh, "type": "building", "children": []any{map[string]any{"name": a1}, map[string]any{"name": a2}}},
		map[string]any{"name": a1, "type": "aisle", "parent": map[string]any{"name": wh}, "children": []any{map[string]any{"name": b1}, map[string]any{"name": b2}}},
		map[string]any{"name": a2, "type": "aisle", "parent": map[string]any{"name": wh}},
		map[string]any{"name": b1, "type": "bin", "parent": map[string]any{"name": a1}},
		map[string]any{"name": b2, "type": "bin", "parent": map[string]any{"name": a1}},
	)
	// Delete leaves first — a location with children cannot be removed before them.
	defer func() {
		for i := len(created) - 1; i >= 0; i-- {
			apiClient.Delete(locationsPath + "/" + created[i])
		}
	}()

	require.Len(t, created, 5, "the whole hierarchy is created in one batch")
	assert.Empty(t, updated)

	// results are row-indexed and recorded in input order, so map each row to its id.
	whID, a1ID, a2ID, b1ID, b2ID := created[0], created[1], created[2], created[3], created[4]

	parentOf := func(id string) map[string]any {
		getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+id, url.Values{"include": {"parent"}})
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody)
		return jsonObject(parseJSON(getBody), "parent")
	}

	assert.Nil(t, parentOf(whID), "warehouse is top-level")
	assert.Equal(t, whID, jsonField(parentOf(a1ID), "id"), "aisle 1 → warehouse")
	assert.Equal(t, whID, jsonField(parentOf(a2ID), "id"), "aisle 2 → warehouse")
	assert.Equal(t, a1ID, jsonField(parentOf(b1ID), "id"), "bin 1 → aisle 1")
	assert.Equal(t, a1ID, jsonField(parentOf(b2ID), "id"), "bin 2 → aisle 1")
}

func TestLocations_BulkUpsert_UpdatePreservesExistingParent(t *testing.T) {
	t.Parallel()

	// Create parent + child with a relationship
	parentName := uniqueName("e2e-buloc-pres-p")
	pStatus, pBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": parentName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, pStatus, pBody)
	parentID := jsonField(parseJSON(pBody), "id")

	childName := uniqueName("e2e-buloc-pres-c")
	cStatus, cBody, err := apiClient.Post(locationsPath, map[string]any{
		"name":      childName,
		"type":      "section",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, cStatus, cBody)
	childID := jsonField(parseJSON(cBody), "id")

	// Bulk upsert only the child (no parent in payload) — parent should be preserved
	bulkUpsertLocations(t, map[string]any{
		"name": childName,
		"type": "aisle", // change type_code
	})

	// Verify: parent link is preserved, type_code is updated
	getStatus, getBody, err := apiClient.GetListRaw(locationsPath+"/"+childID, url.Values{"include": {"parent"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, "aisle", jsonField(got, "type"), "type_code should be updated")
	parent := jsonObject(got, "parent")
	require.NotNil(t, parent, "existing parent should be preserved when bulk upsert doesn't send parent")
	assert.Equal(t, parentID, jsonField(parent, "id"))

	// Cleanup
	apiClient.Delete(locationsPath + "/" + childID)
	apiClient.Delete(locationsPath + "/" + parentID)
}

// --- Bulk Upsert Validation ---

func TestLocations_BulkUpsert_EmptyLocations(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"empty locations should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_BulkUpsert_TooManyLocations(t *testing.T) {
	t.Parallel()
	locs := make([]any, 1001)
	for i := range locs {
		locs[i] = bulkUpsertLocInput(uniqueName("e2e-buloc"), "bin")
	}
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{"locations": locs}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"1001 locations should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_BulkUpsert_DuplicateNameInRequest(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-buloc-dupname")
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{
			bulkUpsertLocInput(name, "bin"),
			bulkUpsertLocInput(name, "bin"), // same name
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"duplicate name in request should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "locations[1].name")
	assert.Contains(t, errObj["message"], "duplicate name")
}

func TestLocations_BulkUpsert_CaseInsensitiveDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-buloc-ci")
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{
			bulkUpsertLocInput(name, "bin"),
			bulkUpsertLocInput(strings.ToUpper(name), "bin"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"case-insensitive duplicate name should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_BulkUpsert_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{
			map[string]any{"type": "bin"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_BulkUpsert_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{
		"locations": []any{
			map[string]any{"name": uniqueName("e2e-buloc-notype")},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing type should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_BulkUpsert_MissingLocationsField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(locationsBulkUpsertPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing locations field should return 400 or 422, got %d: %s", status, string(body))
}

func TestLocations_IncludeParentAndChildren(t *testing.T) {
	t.Parallel()

	// Create grandparent -> parent -> child
	gpName := uniqueName("e2e-loc-gp")
	gpStatus, gpBody, err := apiClient.Post(locationsPath, map[string]any{
		"name": gpName,
		"type": "building",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, gpStatus, gpBody)
	gpID := jsonField(parseJSON(gpBody), "id")

	parentName := uniqueName("e2e-loc-mid")
	pStatus, pBody, err := apiClient.Post(locationsPath, map[string]any{
		"name":      parentName,
		"type":      "section",
		"parent_id": gpID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, pStatus, pBody)
	parentID := jsonField(parseJSON(pBody), "id")

	childName := uniqueName("e2e-loc-leaf")
	cStatus, cBody, err := apiClient.Post(locationsPath, map[string]any{
		"name":      childName,
		"type":      "aisle",
		"parent_id": parentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, cStatus, cBody)
	childID := jsonField(parseJSON(cBody), "id")

	// Get the middle location with both includes
	status, body, err := apiClient.GetListRaw(locationsPath+"/"+parentID, url.Values{"include": {"parent,children"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	parent := jsonObject(got, "parent")
	require.NotNil(t, parent, "parent should be present")
	assert.Equal(t, gpID, jsonField(parent, "id"))

	children := jsonObject(got, "children")
	require.NotNil(t, children, "children should be present")
	assert.Equal(t, "list", jsonField(children, "object"))

	// Cleanup (leaf first)
	apiClient.Delete(locationsPath + "/" + childID)
	apiClient.Delete(locationsPath + "/" + parentID)
	apiClient.Delete(locationsPath + "/" + gpID)
}
