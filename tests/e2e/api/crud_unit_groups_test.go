//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
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

	// Paginate until found: seed rows are the oldest and fall off the
	// first page as repeated e2e runs accumulate data.
	assertListContainsID(t, unitGroupsPath, nil, SeedUnitGroupID)
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
	// System unit groups (currency/time) are always present and undeletable,
	// so a limit=1 page always has a stable row to land on; retry absorbs the
	// hydration race when a parallel test deletes the newest row mid-request.
	assertListPageLen(t, unitGroupsPath, url.Values{"limit": {"1"}}, 1)
}

func TestUnitGroups_ListCursorPagination(t *testing.T) {
	t.Parallel()
	// Retry-bounded two-page fetch: parallel tests can delete the rows
	// behind the cursor between fetches on this shared list.
	assertCursorPaginationAdvances(t, unitGroupsPath, nil)
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

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "socks"),
			"Search result %q should contain 'socks'", name,
		)
	}
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
	createResp, err := apiClient.PostFull(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "unit_group", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
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
	createResp, err := apiClient.PostFull(unitGroupsPath+"?include=base_unit", map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
		"notes":        "Create notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
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
	createResp, err := apiClient.PostFull(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "unit_group", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, "quantity", jsonField(created, "type"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	apiClient.Delete(unitGroupsPath + "/" + id)
}

func TestUnitGroups_CreateWithAssociatedUnits(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-unitgrp-assoc")
	createResp, err := apiClient.PostFull(unitGroupsPath, map[string]any{
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
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)

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
	createResp, err := apiClient.PostFull(unitGroupsPath, map[string]any{
		"name":         name,
		"type":         "quantity",
		"base_unit_id": "each",
		"notes":        notes,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
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
	// base unit ("each") is auto-included on create, so 2 explicit + 1 base = 3.
	assert.Equal(t, 3, len(resp.Data), "Should have 3 associated units (2 explicit + auto-included base unit)")
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
	// base unit ("each") is auto-included on create, so the list has exactly 1 unit.
	assert.Len(t, resp.Data, 1, "newly created group should have exactly 1 associated unit (auto-included base unit)")
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

	createResp, err := apiClient.PostFull(unitGroupUnitsPath(groupID)+"?include=unit", map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
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
	// The base unit ("each") is auto-included on create and stays after deleting the
	// explicitly-added unit, so the list now has exactly 1 entry.
	assert.Len(t, resp.Data, 1, "only the auto-included base unit should remain after deleting the explicit unit")
	for _, raw := range resp.Data {
		m := parseJSON(raw)
		assert.NotEqual(t, unitID, jsonField(m, "id"), "deleted unit must not appear in list")
	}

	// Verify the deleted unit returns 404 by ID.
	getStatus, _, err := apiClient.GetListRaw(fmt.Sprintf("%s/%s/units/%s", unitGroupsPath, groupID, unitID), nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestUnitGroupUnits_Create(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-create"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createResp, err := apiClient.PostFull(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id": "un_01seeddozen00000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "unit_group_unit", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))
}

func TestUnitGroupUnits_CreateWithDiscountAndVisibility(t *testing.T) {
	t.Parallel()
	groupID := createTestUnitGroup(t, uniqueName("e2e-ugunit-disc"))
	defer apiClient.Delete(unitGroupsPath + "/" + groupID)

	createResp, err := apiClient.PostFull(unitGroupUnitsPath(groupID), map[string]any{
		"unit_id":                    "un_01seeddozen00000000",
		"discount_percentage":        10.5,
		"discount_fixed":             2.50,
		"customer_portal_visibility": "hidden",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	assert.Equal(t, "unit_group_unit", jsonField(created, "object"))
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
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

// --- Bulk Upsert Unit Groups ---

const unitGroupsBulkUpsertPath = "/v1/catalog/unit-groups/actions/bulk-upsert"

// bulkUpsertUGInput builds a single unit group entry for bulk-upsert payloads.
func bulkUpsertUGInput(name, unitType, baseUnitID string) map[string]any {
	return map[string]any{
		"name":      name,
		"type":      unitType,
		"base_unit": map[string]any{"id": baseUnitID},
	}
}

// cleanupUnitGroupIDs deletes the given unit groups.
func cleanupUnitGroupIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(unitGroupsPath + "/" + id)
		}
	}
}

// acceptBulkUpsertUnitGroups posts a bulk upsert, requires the 202 job acknowledgment, and
// returns the job's ID to poll.
func acceptBulkUpsertUnitGroups(t *testing.T, groups ...map[string]any) string {
	t.Helper()
	rows := make([]any, len(groups))
	for i, g := range groups {
		rows[i] = g
	}
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{"unit_groups": rows}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")
	return jobID
}

// bulkUpsertUnitGroupsJob posts a bulk upsert and returns the completed job. Bulk upsert is
// partial-success: a row that fails validation against existing rows (dimension mismatch,
// system-group modification) is recorded in the job's `errors` field, not failed — the job
// completes.
func bulkUpsertUnitGroupsJob(t *testing.T, groups ...map[string]any) map[string]any {
	t.Helper()
	job := pollJobUntilTerminal(t, acceptBulkUpsertUnitGroups(t, groups...))
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// bulkUpsertUnitGroups posts a bulk upsert, follows the job to completion, and returns the
// created/updated group IDs from its results.
func bulkUpsertUnitGroups(t *testing.T, groups ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertUnitGroupsJob(t, groups...)
	require.NotEmpty(t, jsonArray(job, "results"), "a completed job must carry results")
	return jobResultIDs(job)
}

// ugJobErrors reads the per-row failures a completed bulk-upsert job recorded.
func ugJobErrors(job map[string]any) []map[string]any {
	var out []map[string]any
	for _, raw := range jsonArray(job, "errors") {
		if m, ok := raw.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func TestUnitGroups_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	name1 := uniqueName("e2e-bug")
	name2 := uniqueName("e2e-bug")

	createdIDs, updatedIDs := bulkUpsertUnitGroups(t,
		bulkUpsertUGInput(name1, "quantity", "each"),
		bulkUpsertUGInput(name2, "quantity", "each"),
	)
	defer cleanupUnitGroupIDs(createdIDs)

	require.Len(t, createdIDs, 2, "should have 2 created IDs")
	for _, id := range createdIDs {
		assertIDFormat(t, id, "ungp")
	}
	assert.Empty(t, updatedIDs, "no updates expected")
}

func TestUnitGroups_BulkUpsert_AllUpdates(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-upd")

	// Create first via bulk upsert
	createdIDs, _ := bulkUpsertUnitGroups(t, bulkUpsertUGInput(name, "quantity", "each"))
	defer cleanupUnitGroupIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	groupID := createdIDs[0]

	notes := "Updated via bulk upsert"
	created, updated := bulkUpsertUnitGroups(t, map[string]any{
		"name":      name,
		"type":      "quantity",
		"base_unit": map[string]any{"id": "each"},
		"notes":     notes,
	})

	assert.Empty(t, created, "no creates expected")
	require.Len(t, updated, 1, "should have 1 updated ID")
	assert.Equal(t, groupID, updated[0], "updated ID should match the originally created ID")

	// Verify notes were persisted
	getStatus, getBody, err := apiClient.GetListRaw(unitGroupsPath+"/"+groupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, notes, jsonField(parseJSON(getBody), "notes"))
}

func TestUnitGroups_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-bug-mix-ex")
	newName := uniqueName("e2e-bug-mix-new")

	// Seed an existing group
	seeded, _ := bulkUpsertUnitGroups(t, bulkUpsertUGInput(existingName, "quantity", "each"))
	defer cleanupUnitGroupIDs(seeded)

	// Mix: update existing + create new
	created, updated := bulkUpsertUnitGroups(t,
		bulkUpsertUGInput(existingName, "quantity", "each"),
		bulkUpsertUGInput(newName, "quantity", "each"),
	)
	defer cleanupUnitGroupIDs(created)

	assert.Len(t, created, 1, "one new group created")
	assert.Len(t, updated, 1, "one existing group updated")
}

func TestUnitGroups_BulkUpsert_ResponseShape(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-shape")

	// 202 returns the canonical job resource.
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{bulkUpsertUGInput(name, "quantity", "each")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	require.NotNil(t, m, "response body should parse as JSON")
	assertObjectField(t, m, "job")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID)

	// The completed job carries a row-indexed result per created/updated group.
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))
	ids, _ := jobResultIDs(job)
	require.Len(t, ids, 1)
	entry := jobResults(job)[0]
	assert.Equal(t, float64(0), entry["index"])
	assert.Equal(t, "created", entry["action"])
	assertIDFormat(t, ids[0], "ungp")
	defer cleanupUnitGroupIDs([]string{ids[0]})
}

func TestUnitGroups_BulkUpsert_Idempotent(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-idem")
	idemKey := newIdempotencyKey()
	payload := map[string]any{
		"unit_groups": []any{bulkUpsertUGInput(name, "quantity", "each")},
	}

	status1, body1, err := apiClient.Post(unitGroupsBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status1, body1)

	status2, body2, err := apiClient.Post(unitGroupsBulkUpsertPath, payload, idemKey)
	require.NoError(t, err)
	requireStatus(t, 202, status2, body2)

	// Replay must return the identical job acknowledgment, not raise a second one.
	assert.JSONEq(t, string(body1), string(body2), "idempotent replay must return the same job")

	jobID := jsonField(parseJSON(body1), "id")
	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"))
	created, _ := jobResultIDs(job)
	require.Len(t, created, 1)
	defer cleanupUnitGroupIDs([]string{created[0]})
}

func TestUnitGroups_BulkUpsert_CreatedGroupAppearsInList(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-list")

	createdIDs, _ := bulkUpsertUnitGroups(t, bulkUpsertUGInput(name, "quantity", "each"))
	defer cleanupUnitGroupIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	createdID := createdIDs[0]

	// Verify it is reachable via GET
	getStatus, getBody, err := apiClient.GetListRaw(unitGroupsPath+"/"+createdID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, createdID, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "quantity", jsonField(got, "type"))
}

func TestUnitGroups_BulkUpsert_BaseUnitAutoIncluded(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-base")

	// Create a group with no explicit unit_conversions
	createdIDs, _ := bulkUpsertUnitGroups(t, bulkUpsertUGInput(name, "quantity", "each"))
	defer cleanupUnitGroupIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	groupID := createdIDs[0]

	// Fetch with associated_units and verify the base unit was auto-included
	getStatus, getBody, err := apiClient.GetListRaw(
		unitGroupsPath+"/"+groupID,
		url.Values{"include": {"associated_units"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assocList := jsonObject(got, "associated_units")
	require.NotNil(t, assocList, "associated_units should be present with ?include=associated_units")
	assert.Equal(t, "list", jsonField(assocList, "object"))

	assocData, ok := assocList["data"].([]any)
	require.True(t, ok, "associated_units.data should be an array")
	assert.GreaterOrEqual(t, len(assocData), 1,
		"base unit should be auto-included as an associated unit even when unit_conversions is omitted")
}

func TestUnitGroups_BulkUpsert_WithConversions(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-conv")
	discountPct := 5.0

	createdIDs, _ := bulkUpsertUnitGroups(t, map[string]any{
		"name":      name,
		"type":      "quantity",
		"base_unit": map[string]any{"id": "each"},
		"unit_conversions": []any{
			map[string]any{
				"unit":                map[string]any{"id": "un_01seeddozen00000000"},
				"discount_percentage": discountPct,
			},
		},
	})
	defer cleanupUnitGroupIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	groupID := createdIDs[0]
	assertIDFormat(t, groupID, "ungp")

	// Verify group is accessible and associated units include the conversion
	getStatus, getBody, err := apiClient.GetListRaw(
		unitGroupsPath+"/"+groupID,
		url.Values{"include": {"associated_units"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assocList := jsonObject(got, "associated_units")
	require.NotNil(t, assocList)
	assocData, ok := assocList["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(assocData), 2, "should have base unit + at least 1 conversion")
}

// --- Bulk Upsert Validation ---

func TestUnitGroups_BulkUpsert_EmptyGroups(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"empty unit_groups should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_BulkUpsert_TooManyGroups(t *testing.T) {
	t.Parallel()
	groups := make([]any, 1001)
	for i := range groups {
		groups[i] = bulkUpsertUGInput(uniqueName("e2e-bug"), "quantity", "each")
	}
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{"unit_groups": groups}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"1001 unit_groups should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_BulkUpsert_DuplicateNameInRequest(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-bug-dupname")
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			bulkUpsertUGInput(name, "quantity", "each"),
			bulkUpsertUGInput(name, "quantity", "each"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"duplicate name in request should return 400 or 422, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "unit_groups[1].name")
	assert.Contains(t, errObj["message"], "duplicate name")
}

func TestUnitGroups_BulkUpsert_DuplicateUnitInGroup(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			map[string]any{
				"name":      uniqueName("e2e-bug-dupunit"),
				"type":      "quantity",
				"base_unit": map[string]any{"id": "each"},
				"unit_conversions": []any{
					map[string]any{"unit": map[string]any{"id": "un_01seeddozen00000000"}, "discount_percentage": 0.0},
					map[string]any{"unit": map[string]any{"id": "un_01seeddozen00000000"}, "discount_percentage": 5.0},
				},
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "unit_groups[0].unit_conversions[1].unit")
}

// TestUnitGroups_BulkUpsert_RejectsSameUnitByIDAndAbbreviation: duplicates are detected on
// the resolved unit, so listing the same unit once by id and once by abbreviation is caught.
func TestUnitGroups_BulkUpsert_RejectsSameUnitByIDAndAbbreviation(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			map[string]any{
				"name":      uniqueName("e2e-bug-dupmixed"),
				"type":      "quantity",
				"base_unit": map[string]any{"id": "each"},
				"unit_conversions": []any{
					map[string]any{"unit": map[string]any{"id": "un_01seeddozen00000000"}, "discount_percentage": 0.0},
					map[string]any{"unit": map[string]any{"abbreviation": "dz"}, "discount_percentage": 5.0},
				},
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "unit_groups[0].unit_conversions[1].unit")
}

// TestUnitGroups_BulkUpsert_ResolvesUnitsByNameAndAbbreviation: base_unit and each
// conversion's unit are fuzzy references — they resolve by name and by abbreviation, not
// only by id.
func TestUnitGroups_BulkUpsert_ResolvesUnitsByNameAndAbbreviation(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bug-refname")
	createdIDs, _ := bulkUpsertUnitGroups(t, map[string]any{
		"name":      name,
		"type":      "quantity",
		"base_unit": map[string]any{"name": "each"}, // by name
		"unit_conversions": []any{
			map[string]any{
				"unit":                map[string]any{"abbreviation": "dz"}, // by abbreviation
				"discount_percentage": 5.0,
			},
		},
	})
	defer cleanupUnitGroupIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(
		unitGroupsPath+"/"+createdIDs[0],
		url.Values{"include": {"associated_units"}},
	)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	assocData, ok := jsonObject(parseJSON(getBody), "associated_units")["data"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(assocData), 2, "base unit + the conversion resolved by abbreviation")
}

// TestUnitGroups_BulkUpsert_RejectsUnknownBaseUnit: a base unit that does not resolve fails
// as a row-indexed validation error before anything is written.
func TestUnitGroups_BulkUpsert_RejectsUnknownBaseUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			bulkUpsertUGInput(uniqueName("e2e-bug-buok"), "quantity", "each"),
			bulkUpsertUGInput(uniqueName("e2e-bug-bubad"), "quantity", "un_does_not_exist_0000"),
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	// Unresolvable references fail in the synchronous accept phase, all-or-nothing: a
	// row-indexed 400 before any job is raised, so the valid row is never written.
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "")
	assertErrorParam(t, errObj, "unit_groups[1].base_unit")
}

func TestUnitGroups_BulkUpsert_MissingName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			map[string]any{
				"type":      "quantity",
				"base_unit": map[string]any{"id": "each"},
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing name should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_BulkUpsert_MissingType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			map[string]any{
				"name":      uniqueName("e2e-bug-notype"),
				"base_unit": map[string]any{"id": "each"},
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing type should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_BulkUpsert_MissingBaseUnit(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(unitGroupsBulkUpsertPath, map[string]any{
		"unit_groups": []any{
			map[string]any{
				"name": uniqueName("e2e-bug-nobase"),
				"type": "quantity",
			},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"missing base_unit should return 400 or 422, got %d: %s", status, string(body))
}

func TestUnitGroups_BulkUpsert_TypeMismatch(t *testing.T) {
	t.Parallel()
	// "gram" is a mass unit (valid base for a mass group); SeedUnitID is a quantity unit,
	// so putting it in this mass group's conversions fails the dimension check. That check
	// needs the group's stored type, so it runs in the execute phase per-row: the job is
	// accepted (202), completes, and records the failure.
	job := bulkUpsertUnitGroupsJob(t, map[string]any{
		"name":      uniqueName("e2e-bug-typemm"),
		"type":      "mass",
		"base_unit": map[string]any{"id": "gram"},
		"unit_conversions": []any{
			map[string]any{"unit": map[string]any{"id": SeedUnitID}, "discount_percentage": 0.0},
		},
	})

	rowErrs := ugJobErrors(job)
	require.Len(t, rowErrs, 1, "the mismatched conversion is recorded in errors")
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "Unit type does not match")
	// Nothing was created — the whole row rolled back on its savepoint.
	assert.Empty(t, jsonArray(job, "results"), "nothing was created")
}

func TestUnitGroups_BulkUpsert_BaseUnitTypeMismatch(t *testing.T) {
	t.Parallel()
	// "each" is a quantity unit, so it cannot be the base of a mass group. The base unit is
	// held to the same dimension rule as conversions, checked in the execute phase per-row:
	// accepted (202), the job completes, and the mismatch is recorded in errors.
	job := bulkUpsertUnitGroupsJob(t, map[string]any{
		"name":      uniqueName("e2e-bug-basemm"),
		"type":      "mass",
		"base_unit": map[string]any{"id": "each"},
	})

	rowErrs := ugJobErrors(job)
	require.Len(t, rowErrs, 1, "the mismatched base unit is recorded in errors")
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "Base unit type does not match")
	assert.Empty(t, jsonArray(job, "results"), "nothing was created")
}

func TestUnitGroups_BulkUpsert_SystemGroupNotModifiable(t *testing.T) {
	t.Parallel()
	// Matching a system unit group by name attempts to modify it. That check needs the
	// stored row, so it runs in the execute phase per-row: accepted (202), the job
	// completes, and the rejection is recorded in errors rather than creating a new group.
	// The base unit must share the group's dimension ("second" is a time unit) so the row
	// reaches the system-group rejection rather than tripping the base-unit type check.
	job := bulkUpsertUnitGroupsJob(t, map[string]any{
		"name":      "Time",
		"type":      "time",
		"base_unit": map[string]any{"id": "second"},
	})

	rowErrs := ugJobErrors(job)
	require.Len(t, rowErrs, 1, "the system-group modification is recorded in errors")
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "cannot be modified")
	assert.Empty(t, jsonArray(job, "results"), "nothing was created")
}

// Partial success: a batch mixing a valid new group with a row whose conversion is of the
// wrong dimension creates the good one and records the bad one — the job completes.
func TestUnitGroups_BulkUpsert_PartialSuccess(t *testing.T) {
	t.Parallel()

	goodName := uniqueName("e2e-bug-good")

	// Row 0: a valid new group. Row 1: a mass group with a quantity-unit conversion.
	job := bulkUpsertUnitGroupsJob(t,
		bulkUpsertUGInput(goodName, "quantity", "each"),
		map[string]any{
			"name":      uniqueName("e2e-bug-bad"),
			"type":      "mass",
			"base_unit": map[string]any{"id": "each"},
			"unit_conversions": []any{
				map[string]any{"unit": map[string]any{"id": SeedUnitID}, "discount_percentage": 0.0},
			},
		},
	)

	created, _ := jobResultIDs(job)
	require.Len(t, created, 1, "the valid row is still created")
	defer cleanupUnitGroupIDs([]string{created[0]})
	assert.Equal(t, float64(0), jobResults(job)[0]["index"], "the surviving result names request row 0")

	rowErrs := ugJobErrors(job)
	require.Len(t, rowErrs, 1, "the bad row is recorded")
	assert.Equal(t, float64(1), rowErrs[0]["index"], "the failure names row index 1")
}
