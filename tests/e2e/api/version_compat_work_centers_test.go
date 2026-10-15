//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview9DepartmentsPath = "/v1/operations/departments"

// A preview.9 caller still uses /v1/operations/departments, reads department objects with notes, writes machines
// and stations by department_id, and includes a machine's department.
func TestVersionCompatWorkCenters_Preview9Departments(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	status, raw, err := preview.Post(preview9DepartmentsPath, map[string]any{"name": uniqueName("e2e-compat-dept"), "notes": "dropped"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	dept := parseJSON(raw)
	deptID := jsonField(dept, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(departmentsPath + "/" + deptID) })
	assert.Equal(t, "department", jsonField(dept, "object"))
	assert.Contains(t, dept, "notes")
	assertNilField(t, dept, "notes")

	status, raw, err = preview.Post(machinesPath+"?include=department", map[string]any{
		"name": uniqueName("e2e-compat-mc"), "serial_number": uniqueName("SN-C"), "department_id": deptID, "notes": "dropped",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	machine := parseJSON(raw)
	machineID := jsonField(machine, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(machinesPath + "/" + machineID) })
	assert.Equal(t, deptID, jsonField(jsonObject(machine, "department"), "id"))
	assert.Equal(t, "department", jsonField(jsonObject(machine, "department"), "object"))
	assert.NotContains(t, machine, "work_center")
	assertNilField(t, machine, "notes")

	status, raw, err = preview.Post(scanningStationsPath, map[string]any{
		"name": uniqueName("e2e-compat-ss"), "type": "move_batch", "operator_requirement": "none", "department_id": deptID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	stationID := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(scanningStationsPath + "/" + stationID) })

	status, raw, err = preview.GetListRaw(preview9DepartmentsPath+"/"+deptID, url.Values{"include": {"machines,scanning_stations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	got := parseJSON(raw)
	assert.Equal(t, "department", jsonField(got, "object"))
	machines := listRows(t, jsonObject(got, "machines"))
	require.Len(t, machines, 1)
	assert.Equal(t, machineID, jsonField(machines[0], "id"))
	require.Len(t, listRows(t, jsonObject(got, "scanning_stations")), 1)

	status, raw, err = preview.Patch(preview9DepartmentsPath+"/"+deptID, map[string]any{"notes": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	list, _, err := preview.GetList(preview9DepartmentsPath, url.Values{"q": {jsonField(dept, "name")}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, "department", jsonField(parseJSON(list.Data[0]), "object"))

	// The same records through forge.
	forge := getWith(t, departmentsPath+"/"+deptID)
	assert.Equal(t, "work_center", jsonField(forge, "object"))
	assert.NotContains(t, forge, "notes")

	// Deleting reads as preview.9's empty object.
	for _, path := range []string{scanningStationsPath + "/" + stationID, machinesPath + "/" + machineID, preview9DepartmentsPath + "/" + deptID} {
		status, raw, err = preview.Delete(path)
		require.NoError(t, err)
		requireStatus(t, 200, status, raw)
		assert.Empty(t, parseJSON(raw))
	}
}

func TestVersionCompatWorkCenters_Preview9BulkUpsert(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)
	name := uniqueName("e2e-compat-bup-dept")

	status, raw, err := preview.Post(preview9DepartmentsPath+"/actions/bulk-upsert", map[string]any{
		"departments": []any{map[string]any{"name": name, "notes": "dropped"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, raw)
	job := pollJobUntilTerminal(t, jsonField(parseJSON(raw), "id"))
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	created, _ := jobResultIDs(job)
	require.Len(t, created, 1)
	defer cleanupDepartmentIDs(created)

	status, raw, err = preview.Post(machinesPath+"/actions/bulk-upsert", map[string]any{
		"machines": []any{map[string]any{"name": uniqueName("e2e-compat-bup-mc"), "serial_number": uniqueName("SN"), "department": map[string]any{"name": name}}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, status, raw)
	job = pollJobUntilTerminal(t, jsonField(parseJSON(raw), "id"))
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	machines, _ := jobResultIDs(job)
	require.Len(t, machines, 1)
	defer cleanupMachineIDs(machines)
}
