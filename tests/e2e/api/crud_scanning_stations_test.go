//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scanningStationsPath = "/v1/operations/scanning-stations"

func TestScanningStations_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-station")
	notes := "test station notes"

	// CREATE
	createResp, err := apiClient.PostFull(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"notes":                notes,
		"operator_requirement": "material_check",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "scanning_station", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, notes, jsonField(created, "notes"))
	assert.Equal(t, "init_batch", jsonField(created, "type"))
	assert.Equal(t, "material_check", jsonField(created, "operator_requirement"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	// Expandable fields are null unless explicitly included.
	assertNilField(t, created, "department")

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))

	// UPDATE
	newName := uniqueName("e2e-station-upd")
	patchStatus, patchBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name":                 newName,
		"notes":                "updated notes",
		"operator_requirement": "none",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, "updated notes", jsonField(updated, "notes"))
	assert.Equal(t, "none", jsonField(updated, "operator_requirement"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(scanningStationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus2, _, err := apiClient.GetListRaw(scanningStationsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestScanningStations_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-stn-allf")
	createResp, err := apiClient.PostFull(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"notes":                "Create notes",
		"operator_requirement": "material_check",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(scanningStationsPath + "/" + id)

	assert.Equal(t, "scanning_station", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "init_batch", jsonField(got, "type"))
	assert.Equal(t, "Create notes", jsonField(got, "notes"))
	assert.Equal(t, "material_check", jsonField(got, "operator_requirement"))
	assertNilField(t, got, "label_size_code")
	assertNilField(t, got, "label_type_code")
	assertNilField(t, got, "production_steps")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	assertNilField(t, got, "department")

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-stn-allf-u")
	patchStatus, patchBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
		"name":                 updatedName,
		"notes":                "Updated notes",
		"operator_requirement": "none",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, "Updated notes", jsonField(updated, "notes"))
	assert.Equal(t, "none", jsonField(updated, "operator_requirement"))
	assert.Equal(t, "init_batch", jsonField(updated, "type"), "type should be preserved")
	assertNilField(t, updated, "label_size_code")
	assertNilField(t, updated, "label_type_code")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	assertNilField(t, updated, "department")
}

func TestScanningStations_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(scanningStationsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 7, "should have at least 7 seeded scanning stations")
}

func TestScanningStations_ListWithLimit(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(scanningStationsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestScanningStations_ListPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(scanningStationsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.True(t, page1.PageInfo.HasNextPage, "should have a next page")
	require.NotNil(t, page1.PageInfo.NextCursor)

	page2, _, err := apiClient.GetList(scanningStationsPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "pages should return different items")
}

func TestScanningStations_ListSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(scanningStationsPath, url.Values{"q": {"Knitting"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "search for 'Knitting' should return at least 1 result")

	// Department is expandable and may be omitted, so assert against station name only.
	for _, item := range list.Data {
		m := parseJSON(item)
		stationName := strings.ToLower(jsonField(m, "name"))
		assert.True(t,
			strings.Contains(stationName, "knitting"),
			"Search result station=%q should match 'knitting' in name",
			jsonField(m, "name"),
		)
	}
}

func TestScanningStations_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(scanningStationsPath, url.Values{"q": {"zzzznotastation99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestScanningStations_GetByID(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, SeedScanningStationID, jsonField(got, "id"))
	assert.Equal(t, "scanning_station", jsonField(got, "object"))
	assert.Equal(t, "Knitting Station", jsonField(got, "name"))
	assert.Equal(t, "init_batch", jsonField(got, "type"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestScanningStations_GetByID_IncludeDepartment(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID, url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	dept := jsonObject(parseJSON(getBody), "department")
	require.NotNil(t, dept, "department should be present with ?include=department")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	assert.Equal(t, "department", jsonField(dept, "object"))
	assert.NotEmpty(t, jsonField(dept, "name"))
}

func TestScanningStations_GetByID_IncludeProductionSteps(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID, url.Values{"include": {"production_steps"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	ps := jsonObject(got, "production_steps")
	require.NotNil(t, ps, "production_steps should be present with ?include=production_steps")
	assert.Equal(t, "list", jsonField(ps, "object"))

	data, ok := ps["data"].([]interface{})
	require.True(t, ok, "production_steps.data should be an array")
	require.GreaterOrEqual(t, len(data), 1, "seeded station should have at least 1 production step")

	first, ok := data[0].(map[string]interface{})
	require.True(t, ok, "production step item should be an object")
	assert.NotEmpty(t, first["id"])
	assert.Equal(t, "production_step", first["object"])
}

func TestScanningStations_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	getStatus, _, err := apiClient.GetListRaw(scanningStationsPath+"/sgsn_nonexistent000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestScanningStations_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-station")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(scanningStationsPath + "/" + id1)
}

func TestScanningStations_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 "",
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

func TestScanningStations_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-dup-station")

	status1, body1, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "move_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status2, "Duplicate name should return 409: %s", string(body2))

	apiClient.Delete(scanningStationsPath + "/" + id)
}

func TestScanningStations_ConnectProductionSteps(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-connect-station")

	// Create a fresh scanning station
	createStatus, createBody, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)
	id := jsonField(parseJSON(createBody), "id")

	// Connect production steps by name. Use "Board" rather than "Knit" to avoid
	// moving the production steps that the seed scanning station depends on, which
	// would cause TestScanningStations_GetByID_IncludeProductionSteps to fail when
	// running in parallel.
	putStatus, putBody, err := apiClient.Put(scanningStationsPath+"/"+id+"/production-steps", map[string]any{
		"name": "Board",
	})
	require.NoError(t, err)
	requireStatus(t, 200, putStatus, putBody)

	// Verify connection via GET with include
	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+id, url.Values{"include": {"production_steps"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	ps := jsonObject(parseJSON(getBody), "production_steps")
	require.NotNil(t, ps, "production_steps should be present")
	data, ok := ps["data"].([]interface{})
	require.True(t, ok, "production_steps.data should be an array")
	assert.GreaterOrEqual(t, len(data), 1, "should have connected at least 1 production step")

	// Restore Board steps to their original scanning station.
	apiClient.Put(scanningStationsPath+"/sgsn_01k0a8201zf1bb2rhmmmxcdqzn/production-steps", map[string]any{
		"name": "Board",
	})

	apiClient.Delete(scanningStationsPath + "/" + id)
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestScanningStations_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-stn-omit")
		status, body, err := apiClient.Post(scanningStationsPath, map[string]any{
			"name":                 name,
			"type":                 "init_batch",
			"operator_requirement": "none",
			"department_id":        SeedDepartmentID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(scanningStationsPath + "/" + id)

		assertObjectField(t, got, "scanning_station")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, "init_batch", jsonField(got, "type"))
		assertNilField(t, got, "notes")
		assert.Equal(t, "none", jsonField(got, "operator_requirement"))
		assertNilField(t, got, "label_size_code")
		assertNilField(t, got, "label_type_code")
		assertNilField(t, got, "production_steps")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		assertNilField(t, got, "department")
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		// Missing name
		status, body, err := apiClient.Post(scanningStationsPath, map[string]any{
			"type":                 "init_batch",
			"operator_requirement": "none",
			"department_id":        SeedDepartmentID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"Missing name should return 400 or 422, got %d: %s", status, string(body))

		// Missing type
		status2, body2, err := apiClient.Post(scanningStationsPath, map[string]any{
			"name":                 uniqueName("e2e-stn-notype"),
			"operator_requirement": "none",
			"department_id":        SeedDepartmentID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status2 == 400 || status2 == 422,
			"Missing type should return 400 or 422, got %d: %s", status2, string(body2))

		// Missing department_id
		status3, body3, err := apiClient.Post(scanningStationsPath, map[string]any{
			"name":                 uniqueName("e2e-stn-nodept"),
			"type":                 "init_batch",
			"operator_requirement": "none",
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status3 == 400 || status3 == 422,
			"Missing department_id should return 400 or 422, got %d: %s", status3, string(body3))
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		// Create with all fields
		name := uniqueName("e2e-stn-pres")
		createStatus, createBody, err := apiClient.Post(scanningStationsPath, map[string]any{
			"name":                 name,
			"type":                 "init_batch",
			"notes":                "Original notes",
			"operator_requirement": "material_check",
			"department_id":        SeedDepartmentID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(scanningStationsPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY name
		newName := uniqueName("e2e-stn-pres-u")
		patchStatus, patchBody, err := apiClient.Patch(scanningStationsPath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "init_batch", jsonField(got, "type"), "type should be preserved")
		assert.Equal(t, "Original notes", jsonField(got, "notes"), "notes should be preserved")
		assert.Equal(t, "material_check", jsonField(got, "operator_requirement"), "operator_requirement should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		assertNilField(t, got, "department")
	})
}
