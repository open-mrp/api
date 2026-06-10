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
	status, _, err := apiClient.GetListRaw(locationsPath+"/sglc_000000000000", nil)
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
