//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const machinesPath = "/v1/operations/machines"

func TestMachines_CRUD(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-machine")
	serialNumber := uniqueName("SN")

	// CREATE
	createResp, err := apiClient.PostFull(machinesPath, map[string]any{
		"name":          name,
		"serial_number": serialNumber,
		"notes":         "test machine notes",
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	created := parseJSON(createResp.Body)
	id := jsonField(created, "id")
	assert.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	assert.Equal(t, "machine", jsonField(created, "object"))
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, serialNumber, jsonField(created, "serial_number"))
	assert.Equal(t, "test machine notes", jsonField(created, "notes"))
	assert.NotEmpty(t, jsonField(created, "created_at"))
	assert.NotEmpty(t, jsonField(created, "updated_at"))

	// Department sub-resource
	dept := jsonObject(created, "department")
	require.NotNil(t, dept, "department should be present")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	assert.Equal(t, "department", jsonField(dept, "object"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(machinesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))

	// UPDATE
	newName := uniqueName("e2e-machine-upd")
	newSerial := uniqueName("SN-UPD")
	patchStatus, patchBody, err := apiClient.Patch(machinesPath+"/"+id, map[string]any{
		"name":          newName,
		"serial_number": newSerial,
		"notes":         "updated notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, newSerial, jsonField(updated, "serial_number"))
	assert.Equal(t, "updated notes", jsonField(updated, "notes"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(machinesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus2, _, err := apiClient.GetListRaw(machinesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

func TestMachines_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	// ── CREATE with all fields ──
	name := uniqueName("e2e-mc-allf")
	serial := uniqueName("SN-ALLF")
	createResp, err := apiClient.PostFull(machinesPath, map[string]any{
		"name":          name,
		"serial_number": serial,
		"notes":         "Create notes",
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createResp.StatusCode, createResp.Body)

	got := parseJSON(createResp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	assertCreatedLocation(t, createResp.Header, id)
	defer apiClient.Delete(machinesPath + "/" + id)

	assert.Equal(t, "machine", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, serial, jsonField(got, "serial_number"))
	assert.Equal(t, "Create notes", jsonField(got, "notes"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))

	dept := jsonObject(got, "department")
	require.NotNil(t, dept, "department must be set after create")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	assert.Equal(t, "department", jsonField(dept, "object"))

	// ── UPDATE with different values ──
	updatedName := uniqueName("e2e-mc-allf-u")
	updatedSerial := uniqueName("SN-ALLF-U")
	patchStatus, patchBody, err := apiClient.Patch(machinesPath+"/"+id, map[string]any{
		"name":          updatedName,
		"serial_number": updatedSerial,
		"notes":         "Updated notes",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	assert.Equal(t, updatedSerial, jsonField(updated, "serial_number"))
	assert.Equal(t, "Updated notes", jsonField(updated, "notes"))

	// Department should be preserved
	updDept := jsonObject(updated, "department")
	require.NotNil(t, updDept, "department should be preserved")
	assert.Equal(t, SeedDepartmentID, jsonField(updDept, "id"))
}

func TestMachines_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(machinesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 3, "should have at least 3 seeded machines")
}

func TestMachines_ListWithLimit(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(machinesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1)
}

func TestMachines_ListPagination(t *testing.T) {
	t.Parallel()
	page1, _, err := apiClient.GetList(machinesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.True(t, page1.PageInfo.HasNextPage, "should have a next page")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	id1 := DataItemField(page1.Data[0], "id")
	id2 := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, id1, id2, "pages should return different items")
}

func TestMachines_ListSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(machinesPath, url.Values{"q": {"Knitting"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "search for 'Knitting' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "knitting"),
			"Search result %q should contain 'knitting'", name,
		)
	}
}

func TestMachines_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(machinesPath, url.Values{"q": {"zzzznotamachine99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestMachines_GetByID(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(machinesPath+"/"+SeedMachineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, SeedMachineID, jsonField(got, "id"))
	assert.Equal(t, "machine", jsonField(got, "object"))
	assert.Equal(t, "Knitting Machine 1", jsonField(got, "name"))
	assert.Equal(t, "J24-001", jsonField(got, "serial_number"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestMachines_GetByID_IncludeDepartment(t *testing.T) {
	t.Parallel()
	getStatus, getBody, err := apiClient.GetListRaw(machinesPath+"/"+SeedMachineID, url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	dept := jsonObject(parseJSON(getBody), "department")
	require.NotNil(t, dept, "department should be present with ?include=department")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	assert.Equal(t, "department", jsonField(dept, "object"))
	assert.NotEmpty(t, jsonField(dept, "name"))
}

func TestMachines_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	getStatus, _, err := apiClient.GetListRaw(machinesPath+"/mc_nonexistent000000000", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)
}

func TestMachines_CreateIdempotent(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-idem-machine")
	serial := uniqueName("SN-IDEM")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(machinesPath, map[string]any{
		"name":          name,
		"serial_number": serial,
		"department_id": SeedDepartmentID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(machinesPath, map[string]any{
		"name":          name,
		"serial_number": serial,
		"department_id": SeedDepartmentID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))

	apiClient.Delete(machinesPath + "/" + id1)
}

func TestMachines_CreateValidation_EmptyName(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(machinesPath, map[string]any{
		"name":          "",
		"serial_number": "SN-EMPTY",
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"Empty name should return 400 or 422, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestMachines_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		name := uniqueName("e2e-mc-omit")
		serial := uniqueName("SN-OMIT")
		status, body, err := apiClient.Post(machinesPath, map[string]any{
			"name":          name,
			"serial_number": serial,
			"department_id": SeedDepartmentID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(machinesPath + "/" + id)

		assertObjectField(t, got, "machine")
		assert.Equal(t, name, jsonField(got, "name"))
		assert.Equal(t, serial, jsonField(got, "serial_number"))
		assertNilField(t, got, "notes")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		dept := jsonObject(got, "department")
		require.NotNil(t, dept, "department should be populated")
		assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("e2e-mc-pres")
		serial := uniqueName("SN-PRES")
		createStatus, createBody, err := apiClient.Post(machinesPath, map[string]any{
			"name":          name,
			"serial_number": serial,
			"notes":         "Original notes",
			"department_id": SeedDepartmentID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(machinesPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// Update ONLY notes
		patchStatus, patchBody, err := apiClient.Patch(machinesPath+"/"+id, map[string]any{
			"notes": "Changed notes",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, "Changed notes", jsonField(got, "notes"))
		assert.Equal(t, name, jsonField(got, "name"), "name should be preserved")
		assert.Equal(t, serial, jsonField(got, "serial_number"), "serial_number should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		dept := jsonObject(got, "department")
		require.NotNil(t, dept, "department should be preserved")
		assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
	})
}

func TestMachines_CreateDuplicateName(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-dup-machine")

	status1, body1, err := apiClient.Post(machinesPath, map[string]any{
		"name":          name,
		"serial_number": uniqueName("SN"),
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id := jsonField(parseJSON(body1), "id")

	status2, body2, err := apiClient.Post(machinesPath, map[string]any{
		"name":          name,
		"serial_number": uniqueName("SN"),
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 409, status2, "Duplicate name should return 409: %s", string(body2))

	apiClient.Delete(machinesPath + "/" + id)
}
