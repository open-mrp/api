//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unitGroupsPath = "/v1/catalog/unit-groups"

// --- List ---

func TestUnitGroups_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitGroupsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 seeded unit group")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedUnitGroupID {
			found = true
			break
		}
	}
	assert.True(t, found, "Seeded unit group should appear in list")
}

func TestUnitGroups_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitGroupsPath, nil)
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "unit_group", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "name"))
		assert.NotEmpty(t, jsonField(m, "type"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestUnitGroups_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitGroupsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestUnitGroups_ListCursorPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(unitGroupsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)

	if !page1.PageInfo.HasNextPage {
		t.Skip("Not enough unit groups for pagination test")
		return
	}
	require.NotNil(t, page1.PageInfo.NextCursor, "next_cursor should be set when has_next_page is true")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	page2, _, err := apiClient.GetList(unitGroupsPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should return a different item than page 1")
}

func TestUnitGroups_ListFilterByType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitGroupsPath, url.Values{"type": {"mass"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 mass unit group")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "mass", jsonField(m, "type"), "All results should have type=mass")
	}
}

func TestUnitGroups_ListSearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitGroupsPath, url.Values{"q": {"Socks"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Socks' should return at least 1 result")
}

func TestUnitGroups_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(unitGroupsPath, url.Values{"q": {"zzzznotaunitgroup99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Get ---

func TestUnitGroups_GetByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedUnitGroupID, jsonField(got, "id"))
	assert.Equal(t, "unit_group", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestUnitGroups_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(unitGroupsPath+"/ungp_000000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

// --- CRUD ---

func TestUnitGroups_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp")

	// Create
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	assert.Equal(t, "unit_group", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "quantity", jsonField(created, "type"))

	// Get
	getStatus, getBody, err := apiClient.GetListRaw(unitGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, name, jsonField(parseJSON(getBody), "name"))

	// Update name
	newName := uniqueName("e2e-unitgrp-upd")
	patchStatus, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, newName, jsonField(parseJSON(patchBody), "name"))

	// Verify update
	getStatus2, getBody2, err := apiClient.GetListRaw(unitGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, newName, jsonField(parseJSON(getBody2), "name"))

	// Delete
	delStatus, delBody, err := apiClient.Delete(unitGroupsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus3, _, err := apiClient.GetListRaw(unitGroupsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus3)
}

// --- Create ---

func TestUnitGroups_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-ug-allf")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath+"?include=base_unit", map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
		"notes":        "Create notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(unitGroupsPath + "/" + id)

	assert.Equal(t, "unit_group", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "quantity", jsonField(got, "type"))
	assert.Equal(t, "Create notes", jsonField(got, "notes"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	baseUnit := jsonObject(got, "base_unit")
	require.NotNil(t, baseUnit, "base_unit must be set after create")
	assert.Equal(t, "unit", jsonField(baseUnit, "object"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-ug-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id+"?include=base_unit", map[string]any{
		"name":  updatedName,
		"notes": "Updated notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "Updated notes", jsonField(updated, "notes"))
	assert.Equal(t, "quantity", jsonField(updated, "type"), "type should be preserved")

	// base_unit should be preserved
	updBaseUnit := jsonObject(updated, "base_unit")
	require.NotNil(t, updBaseUnit, "base_unit should be preserved")
	assert.Equal(t, "unit", jsonField(updBaseUnit, "object"))
}

func TestUnitGroups_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-shape")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	assert.NotEmpty(t, jsonField(created, "id"))
	assert.Equal(t, "unit_group", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "quantity", jsonField(created, "type"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(unitGroupsPath + "/" + jsonField(created, "id"))
}

func TestUnitGroups_CreateWithAssociatedUnits(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-assoc")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
		"associated_units": []map[string]any{
			{
				"unit_id":             "un_01seeddozen00000000",
				"discount_percentage": 5.0,
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)

	// Fetch with include to verify associated units
	getStatus, getBody, err := apiClient.GetListRaw(unitGroupsPath+"/"+id, url.Values{"include": {"associated_units"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assocList := jsonObject(got, "associated_units")
	require.NotNil(t, assocList, "associated_units should be present with ?include=associated_units")
	assert.Equal(t, "list", jsonField(assocList, "object"))

	assocData, ok := assocList["data"].([]any)
	require.True(t, ok, "associated_units.data should be an array")
	assert.GreaterOrEqual(t, len(assocData), 1, "Should have at least 1 associated unit")

	apiClient.Delete(unitGroupsPath + "/" + id)
}

func TestUnitGroups_CreateWithNotes(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-notes")
	notes := "Test notes for unit group"
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
		"notes":        notes,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	assert.Equal(t, notes, jsonField(created, "notes"))

	apiClient.Delete(unitGroupsPath + "/" + id)
}

func TestUnitGroups_CreateValidation_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_CreateValidation_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         uniqueName("e2e-unitgrp-notype"),
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing type should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_CreateValidation_MissingBaseUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name": uniqueName("e2e-unitgrp-nobase"),
		"type": "quantity",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing base_unit_id should return 400 or 422, got %d: %s", status, string(body))
}

// --- Idempotency ---

func TestUnitGroups_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-unitgrp")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(unitGroupsPath + "/" + id1)
}

func TestUnitGroups_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-idem-upd")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-unitgrp-idem-upd2")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{"name": newName}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, jsonField(parseJSON(body1), "name"), jsonField(parseJSON(body2), "name"))
	assert.Equal(t, jsonField(parseJSON(body1), "id"), jsonField(parseJSON(body2), "id"))

	apiClient.Delete(unitGroupsPath + "/" + id)
}

// --- Update ---

func TestUnitGroups_UpdateOnlyName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-pname")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	newName := uniqueName("e2e-unitgrp-pname2")
	patchStatus, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(patched, "name"))
	assert.Equal(t, "quantity", jsonField(patched, "type"), "type should be preserved when only name is updated")

	apiClient.Delete(unitGroupsPath + "/" + id)
}

func TestUnitGroups_UpdateNotes(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-notes-upd")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Set notes
	patchStatus, patchBody, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"notes": "Updated notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "Updated notes", jsonField(parseJSON(patchBody), "notes"))

	// Update notes to a different value
	patchStatus2, patchBody2, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"notes": "Changed notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus2, patchBody2)
	assert.Equal(t, "Changed notes", jsonField(parseJSON(patchBody2), "notes"))

	// Verify notes preserved when patching only name
	newName := uniqueName("e2e-unitgrp-notes-upd2")
	patchStatus3, patchBody3, err := apiClient.Patch(unitGroupsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus3, patchBody3)
	assert.Equal(t, "Changed notes", jsonField(parseJSON(patchBody3), "notes"), "notes should be preserved when only name is updated")

	apiClient.Delete(unitGroupsPath + "/" + id)
}

func TestUnitGroups_UpdateSystemGroupFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(unitGroupsPath+"/time_group", map[string]any{
		"name": uniqueName("e2e-sys-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Updating a system unit group should fail, got %d", status)
}

// --- Delete ---

func TestUnitGroups_DeleteSystemGroupFails(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(unitGroupsPath + "/time_group")
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 403 || status == 409 || status == 422,
		"Deleting a system unit group should fail, got %d", status)
}

func TestUnitGroups_DeleteAlreadyDeletedFails(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-deldel")
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	delStatus, delBody, err := apiClient.Delete(unitGroupsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Second delete should fail
	status2, _, err := apiClient.Delete(unitGroupsPath + "/" + id)
	require.NoError(t, err)
	assert.True(t, status2 == 404 || status2 == 410,
		"Deleting an already-deleted unit group should return 404 or 410, got %d", status2)
}

// --- Expandable Fields ---

func TestUnitGroups_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	// Test on Get
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["base_unit"], "base_unit should be null without ?include=base_unit")
	assert.Nil(t, got["associated_units"], "associated_units should be null without ?include=associated_units")
	assert.Nil(t, got["owner"], "owner should be null without ?include=owner")

	// Test on List
	list, _, err := apiClient.GetList(unitGroupsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["base_unit"], "base_unit should be null on list items without ?include=base_unit")
		assert.Nil(t, m["associated_units"], "associated_units should be null on list items without ?include=associated_units")
		assert.Nil(t, m["owner"], "owner should be null on list items without ?include=owner")
	}
}

func TestUnitGroups_IncludeBaseUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, url.Values{"include": {"base_unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	baseUnit := jsonObject(got, "base_unit")
	require.NotNil(t, baseUnit, "base_unit should be present with ?include=base_unit")
	assert.Equal(t, "unit", jsonField(baseUnit, "object"))
	assert.NotEmpty(t, jsonField(baseUnit, "id"))
}

func TestUnitGroups_IncludeAssociatedUnits(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, url.Values{"include": {"associated_units"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assocList := jsonObject(got, "associated_units")
	require.NotNil(t, assocList, "associated_units should be present with ?include=associated_units")
	assert.Equal(t, "list", jsonField(assocList, "object"))

	assocData, ok := assocList["data"].([]any)
	require.True(t, ok, "associated_units.data should be an array")
	assert.GreaterOrEqual(t, len(assocData), 1, "Seeded unit group should have associated units")

	// Verify shape of first associated unit
	firstRaw, err := json.Marshal(assocData[0])
	require.NoError(t, err)
	first := parseJSON(firstRaw)
	assert.Equal(t, "unit_group_unit", jsonField(first, "object"))
	assert.NotEmpty(t, jsonField(first, "id"))
	assert.NotEmpty(t, jsonField(first, "created_at"))
	assert.NotEmpty(t, jsonField(first, "updated_at"))
}

func TestUnitGroups_IncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupsPath+"/"+SeedUnitGroupID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}

// --- Unit Group Units (Nested CRUD) ---

// createTestUnitGroup is a helper that creates a unit group and returns its ID.
// It fails the test if creation fails.
func createTestUnitGroup(t *testing.T, name string) string {
	t.Helper()
	createStatus, createBody, err := apiClient.Post(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	return jsonField(parseJSON(createBody), "id")
}

func unitGroupUnitsPath(unitGroupID string) string {
	return fmt.Sprintf("%s/%s/units", unitGroupsPath, unitGroupID)
}

func TestUnitGroupUnits_List(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupUnitsPath(SeedUnitGroupID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var resp struct {
		Object string            `json:"object"`
		Data   []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "list", resp.Object)
	assert.GreaterOrEqual(t, len(resp.Data), 1, "Seeded unit group should have associated units")

	// Verify response shape
	for _, item := range resp.Data {
		m := parseJSON(item)
		assert.Equal(t, "unit_group_unit", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
		// unit should be null without ?include=unit
		assert.Nil(t, m["unit"], "unit should be null without ?include=unit")
	}
}

func TestUnitGroupUnits_ListIncludeUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupUnitsPath(SeedUnitGroupID), url.Values{"include": {"unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	require.GreaterOrEqual(t, len(resp.Data), 1)

	for _, item := range resp.Data {
		m := parseJSON(item)
		unit := jsonObject(m, "unit")
		require.NotNil(t, unit, "unit should be present with ?include=unit")
		assert.Equal(t, "unit", jsonField(unit, "object"))
		assert.NotEmpty(t, jsonField(unit, "id"))
	}
}

func TestUnitGroupUnits_GetByID(t *testing.T) {
	t.Parallel()
	path := fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, SeedUnitGroupID, SeedUnitGroupUnitID)
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedUnitGroupUnitID, jsonField(got, "id"))
	assert.Equal(t, "unit_group_unit", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
	// unit should be null without include
	assert.Nil(t, got["unit"], "unit should be null without ?include=unit")
}

func TestUnitGroupUnits_GetNotFound(t *testing.T) {
	t.Parallel()
	path := fmt.Sprintf("%s/%s/units/ungpun_000000000000", unitGroupsPath, SeedUnitGroupID)
	status, _, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestUnitGroupUnits_GetIncludeUnit(t *testing.T) {
	t.Parallel()
	path := fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, SeedUnitGroupID, SeedUnitGroupUnitID)
	status, body, err := apiClient.GetListRaw(path, url.Values{"include": {"unit"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	unit := jsonObject(got, "unit")
	require.NotNil(t, unit, "unit should be present with ?include=unit")
	assert.Equal(t, "unit", jsonField(unit, "object"))
	assert.NotEmpty(t, jsonField(unit, "id"))
}

func TestUnitGroupUnits_ListResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(unitGroupUnitsPath(SeedUnitGroupID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var resp struct {
		Object string            `json:"object"`
		Data   []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "list", resp.Object)

	for _, item := range resp.Data {
		m := parseJSON(item)
		assert.Equal(t, "unit_group_unit", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "customer_portal_visibility"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
		assert.NotEmpty(t, jsonField(m, "updated_at"))
	}
}

func TestUnitGroupUnits_ListForCreatedGroup(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-listcr"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Add two associated units
	status1, body1, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)

	status2, body2, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": SeedUnitID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)

	// List should return both
	status, body, err := apiClient.GetListRaw(unitGroupUnitsPath(groupID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, 2, len(resp.Data), "Should have 2 associated units")
}

func TestUnitGroupUnits_ListEmpty(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-listempty"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	status, body, err := apiClient.GetListRaw(unitGroupUnitsPath(groupID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var resp struct {
		Object string            `json:"object"`
		Data   []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "list", resp.Object)
	assertEmptyListData(t, resp.Data)
}

func TestUnitGroupUnits_GetResponseShape(t *testing.T) {
	t.Parallel()
	path := fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, SeedUnitGroupID, SeedUnitGroupUnitID)
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedUnitGroupUnitID, jsonField(got, "id"))
	assert.Equal(t, "unit_group_unit", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "customer_portal_visibility"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestUnitGroupUnits_CreateIncludeUnit(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-crincl"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID)+"?include=unit", map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	unit := jsonObject(created, "unit")
	require.NotNil(t, unit, "unit should be present with ?include=unit on create")
	assert.Equal(t, "unit", jsonField(unit, "object"))
	assert.NotEmpty(t, jsonField(unit, "id"))
}

func TestUnitGroupUnits_UpdateIncludeUnit(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-updincl"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Create
	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	// Update with include
	patchPath := fmt.Sprintf("%s/%s/units/%s?include=unit", unitGroupsPath, groupID, unitID)
	patchStatus, patchBody, err := apiClient.Patch(patchPath, map[string]any{
		"discount_percentage": 20.0,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	unit := jsonObject(patched, "unit")
	require.NotNil(t, unit, "unit should be present with ?include=unit on update")
	assert.Equal(t, "unit", jsonField(unit, "object"))
	assert.NotEmpty(t, jsonField(unit, "id"))
}

func TestUnitGroupUnits_DeleteVerifyViaList(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-dellst"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Create an associated unit
	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	// Delete the associated unit
	delStatus, delBody, err := apiClient.Delete(fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID))
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify via list endpoint
	listStatus, listBody, err := apiClient.GetListRaw(unitGroupUnitsPath(groupID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, listStatus, listBody)

	var resp struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listBody, &resp))
	assertEmptyListData(t, resp.Data, "List should be empty after deleting the only associated unit")

	// Verify via get endpoint returns 404
	getStatus, _, err := apiClient.GetListRaw(fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestUnitGroupUnits_Create(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-create"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	assert.Equal(t, "unit_group_unit", jsonField(created, "object"))
	assert.NotEmpty(t, jsonField(created, "id"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))
}

func TestUnitGroupUnits_CreateWithDiscountAndVisibility(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-disc"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id":                    "un_01seeddozen00000000",
		"discount_percentage":        10.5,
		"discount_fixed":             2.50,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	assert.Equal(t, "unit_group_unit", jsonField(created, "object"))
	assert.Equal(t, "10.5", jsonField(created, "discount_percentage"))
	assert.Equal(t, "2.5", jsonField(created, "discount_fixed"))
	assert.Equal(t, "hidden", jsonField(created, "customer_portal_visibility"))
}

func TestUnitGroupUnits_Update(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-upd"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Create an associated unit
	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id":             "un_01seeddozen00000000",
		"discount_percentage": 5.0,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	// Update discount_percentage
	patchStatus, patchBody, err := apiClient.Patch(
		fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID),
		map[string]any{"discount_percentage": 15.0},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "15", jsonField(patched, "discount_percentage"))
}

func TestUnitGroupUnits_UpdateDiscountFixed(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-dfx"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Create an associated unit with default discount_fixed
	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	// Update discount_fixed
	patchStatus, patchBody, err := apiClient.Patch(
		fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID),
		map[string]any{"discount_fixed": 7.25},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	assert.Equal(t, "7.25", jsonField(patched, "discount_fixed"))
}

func TestUnitGroupUnits_UpdateVisibility(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-vis"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Create an associated unit with default visibility
	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	created := parseJSON(createBody)
	unitID := jsonField(created, "id")
	assert.Equal(t, "visible", jsonField(created, "customer_portal_visibility"))

	// Update to hidden
	patchStatus, patchBody, err := apiClient.Patch(
		fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID),
		map[string]any{"customer_portal_visibility": "hidden"},
		newIdempotencyKey(),
	)
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	assert.Equal(t, "hidden", jsonField(parseJSON(patchBody), "customer_portal_visibility"))
}

func TestUnitGroupUnits_Delete(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-del"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	// Create an associated unit
	createStatus, createBody, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	unitID := jsonField(parseJSON(createBody), "id")

	// Delete the associated unit
	delStatus, delBody, err := apiClient.Delete(fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID))
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify the associated unit no longer appears when fetching the group with includes
	getStatus, getBody, err := apiClient.GetListRaw(unitGroupsPath+"/"+groupID, url.Values{"include": {"associated_units"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assocRaw := got["associated_units"]
	if assocRaw != nil {
		assocSlice, ok := assocRaw.([]any)
		if ok {
			for _, item := range assocSlice {
				itemMap, ok := item.(map[string]any)
				if ok {
					assert.NotEqual(t, unitID, jsonField(itemMap, "id"), "Deleted associated unit should not appear")
				}
			}
		}
	}
}

func TestUnitGroupUnits_CreateIdempotent(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-idem"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}

	status1, body1, err := apiClient.Post(unitGroupUnitsPath(groupID), payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(unitGroupUnitsPath(groupID), payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))
}

func TestUnitGroupUnits_CreateValidation_MissingUnitID(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-val"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	status, body, err := apiClient.Post(unitGroupUnitsPath(groupID), map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Missing unit_id should return 400 or 422, got %d: %s", status, string(body))
}
