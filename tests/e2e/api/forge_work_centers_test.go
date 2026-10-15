//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

// createWorkCenter creates a work center and deletes it when the test ends.
func createWorkCenter(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	if _, ok := body["name"]; !ok {
		body["name"] = uniqueName("e2e-wc")
	}
	return createAndCleanup(t, departmentsPath, body)
}

// createMachineAt creates a machine at workCenterID and deletes it when the test ends (before the work center).
func createMachineAt(t *testing.T, workCenterID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"name": uniqueName("e2e-wc-mc"), "serial_number": uniqueName("SN-WC"), "work_center_id": workCenterID}
	for k, v := range extra {
		body[k] = v
	}
	return createAndCleanup(t, machinesPath, body)
}

func createStationAt(t *testing.T, workCenterID string) map[string]any {
	t.Helper()
	return createAndCleanup(t, scanningStationsPath, map[string]any{
		"name": uniqueName("e2e-wc-ss"), "type": "move_batch", "operator_requirement": "none", "work_center_id": workCenterID,
	})
}

func getWith(t *testing.T, path string, include ...string) map[string]any {
	t.Helper()
	var q url.Values
	if len(include) > 0 {
		q = url.Values{"include[]": include}
	}
	status, raw, err := apiClient.GetListRaw(path, q)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	return parseJSON(raw)
}

// --- CRUD ---

func TestForgeWorkCenters_AllFields(t *testing.T) {
	t.Parallel()
	home := createWorkCenter(t, map[string]any{})
	machine := createMachineAt(t, jsonField(home, "id"), nil)
	station := createStationAt(t, jsonField(home, "id"))

	name := uniqueName("e2e-wc-all")
	status, raw, err := apiClient.Post(departmentsPath+"?include[]=location&include[]=machines&include[]=scanning_stations", map[string]any{
		"name":                 name,
		"location_id":          SeedLocationID,
		"machine_ids":          []string{jsonField(machine, "id")},
		"scanning_station_ids": []string{jsonField(station, "id")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	got := parseJSON(raw)
	wcID := jsonField(got, "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(departmentsPath + "/" + wcID) })

	assertIDFormat(t, wcID, id.WorkCenterIDPrefix)
	assert.Equal(t, "work_center", jsonField(got, "object"))
	assert.Equal(t, name, jsonField(got, "name"))
	assertNilField(t, got, "labor_rate")
	assert.NotContains(t, got, "notes")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	assert.Equal(t, SeedLocationID, jsonField(jsonObject(got, "location"), "id"))

	machines := listRows(t, jsonObject(got, "machines"))
	require.Len(t, machines, 1, "the moved machine is the work center's only one")
	assert.Equal(t, jsonField(machine, "id"), jsonField(machines[0], "id"))
	assert.Equal(t, "active", jsonField(machines[0], "status"), "embedded machines are full records")
	stations := listRows(t, jsonObject(got, "scanning_stations"))
	require.Len(t, stations, 1)
	assert.Equal(t, "move_batch", jsonField(stations[0], "type"))

	plain := getWith(t, departmentsPath+"/"+wcID)
	for _, key := range []string{"location", "machines", "scanning_stations"} {
		assertNilField(t, plain, key)
	}

	// The members moved: they now name the new work center.
	moved := getWith(t, machinesPath+"/"+jsonField(machine, "id"), "work_center")
	assert.Equal(t, wcID, jsonField(jsonObject(moved, "work_center"), "id"))
	assert.Equal(t, "work_center", jsonField(jsonObject(moved, "work_center"), "object"))

	renamed := uniqueName("e2e-wc-all-r")
	status, raw, err = apiClient.Patch(departmentsPath+"/"+wcID+"?include[]=location", map[string]any{"name": renamed}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	updated := parseJSON(raw)
	assert.Equal(t, renamed, jsonField(updated, "name"))
	assert.Equal(t, SeedLocationID, jsonField(jsonObject(updated, "location"), "id"), "an omitted location is kept")
}

func TestForgeWorkCenters_NotesAreGone(t *testing.T) {
	t.Parallel()
	status, raw, err := apiClient.Post(departmentsPath, map[string]any{"name": uniqueName("e2e-wc-notes"), "notes": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
	requireErrorResponse(t, raw, "parameter_unknown", "invalid_request_error")

	status, raw, err = apiClient.Post(machinesPath, map[string]any{
		"name": uniqueName("e2e-wc-notes-mc"), "serial_number": uniqueName("SN"), "work_center_id": SeedDepartmentID, "notes": "x",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
}

// --- Embedded lists ---

func TestForgeWorkCenters_EmbeddedMachinesPage(t *testing.T) {
	t.Parallel()
	wc := createWorkCenter(t, map[string]any{})
	wcID := jsonField(wc, "id")
	for range 11 {
		createMachineAt(t, wcID, nil)
	}

	got := getWith(t, departmentsPath+"/"+wcID, "machines")
	machines := jsonObject(got, "machines")
	assert.Len(t, listRows(t, machines), 10, "an embedded list holds the first 10")
	next := jsonField(jsonObject(machines, "page_info"), "next_page_url")
	require.NotEmpty(t, next, "more than 10 links to the filtered list: %v", machines)
	assert.Contains(t, next, "/v1/operations/machines?work_center_ids")

	list, _, err := apiClient.GetList(machinesPath, url.Values{"work_center_ids[]": {wcID}, "limit": {"100"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 11, "the list filtered to the work center holds them all")
}

// --- Filters, moves and status ---

func TestForgeWorkCenters_MachineAndStationFiltersAndMoves(t *testing.T) {
	t.Parallel()
	a := jsonField(createWorkCenter(t, map[string]any{}), "id")
	b := jsonField(createWorkCenter(t, map[string]any{}), "id")
	machine := jsonField(createMachineAt(t, a, nil), "id")
	archived := jsonField(createMachineAt(t, a, map[string]any{"status": "archived"}), "id")
	station := jsonField(createStationAt(t, a), "id")

	list, _, err := apiClient.GetList(machinesPath, url.Values{"work_center_ids[]": {a}, "statuses[]": {"archived"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, archived, jsonField(parseJSON(list.Data[0]), "id"))
	list, _, err = apiClient.GetList(machinesPath, url.Values{"work_center_ids[]": {a}, "statuses[]": {"active"}})
	require.NoError(t, err)
	require.Len(t, list.Data, 1)
	assert.Equal(t, machine, jsonField(parseJSON(list.Data[0]), "id"))

	status, raw, err := apiClient.Patch(machinesPath+"/"+machine+"?include[]=work_center", map[string]any{"work_center_id": b}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, b, jsonField(jsonObject(parseJSON(raw), "work_center"), "id"))

	status, raw, err = apiClient.Patch(scanningStationsPath+"/"+station+"?include[]=work_center", map[string]any{"work_center_id": b}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, b, jsonField(jsonObject(parseJSON(raw), "work_center"), "id"))

	stations, _, err := apiClient.GetList(scanningStationsPath, url.Values{"work_center_ids[]": {b}})
	require.NoError(t, err)
	require.Len(t, stations.Data, 1)
	assert.Equal(t, station, jsonField(parseJSON(stations.Data[0]), "id"))
	stations, _, err = apiClient.GetList(scanningStationsPath, url.Values{"work_center_ids[]": {a}})
	require.NoError(t, err)
	assert.Empty(t, stations.Data)
}

// --- Archived references ---

func TestForgeWorkCenters_ArchivedReferences(t *testing.T) {
	t.Parallel()
	archivedLocation := jsonField(createLocation(t, map[string]any{"name": uniqueName("e2e-wc-arch-loc"), "type": "building", "status": "archived"}), "id")

	status, raw, err := apiClient.Post(departmentsPath, map[string]any{"name": uniqueName("e2e-wc-arch"), "location_id": archivedLocation}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "location_id")

	wc := jsonField(createWorkCenter(t, map[string]any{}), "id")
	status, raw, err = apiClient.Patch(departmentsPath+"/"+wc, map[string]any{"location_id": archivedLocation}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "location_id")

	archivedMachine := jsonField(createMachineAt(t, SeedDepartmentID, map[string]any{"status": "archived"}), "id")
	status, raw, err = apiClient.Patch(departmentsPath+"/"+wc, map[string]any{"machine_ids": []string{archivedMachine}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "machine_ids")

	status, raw, err = apiClient.Post(machineDowntimeEventsPath, map[string]any{
		"machine_id": archivedMachine, "reason": "breakdown", "started_at": rfc3339(time.Now().Add(-time.Hour)), "ended_at": rfc3339(time.Now()),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "resource_archived", "invalid_request_error"), "machine_id")

	status, raw, err = apiClient.Patch(departmentsPath+"/"+wc, map[string]any{"machine_ids": []string{"mch_doesnotexist0000"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
}

func TestForgeWorkCenters_BulkUpsertRejectsAnArchivedLocation(t *testing.T) {
	t.Parallel()
	archivedLocation := jsonField(createLocation(t, map[string]any{"name": uniqueName("e2e-wc-bup-loc"), "type": "building", "status": "archived"}), "id")

	status, raw, err := apiClient.Post(departmentsBulkUpsertPath, map[string]any{"work_centers": []any{
		map[string]any{"name": uniqueName("e2e-wc-bup-ok")},
		map[string]any{"name": uniqueName("e2e-wc-bup-arch"), "location": map[string]any{"id": archivedLocation}},
	}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, raw)
	errObj := requireErrorResponse(t, raw, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "work_centers[1].location")
}

// --- Delete ---

func TestForgeWorkCenters_DeleteBlockedWhileInUse(t *testing.T) {
	t.Parallel()
	wc := jsonField(createWorkCenter(t, map[string]any{}), "id")
	machine := jsonField(createMachineAt(t, wc, nil), "id")

	status, raw, err := apiClient.Delete(departmentsPath + "/" + wc)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	errObj := requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "machines")

	status, raw, err = apiClient.Delete(machinesPath + "/" + machine)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	stub := parseJSON(raw)
	assert.Equal(t, map[string]any{"id": machine, "object": "machine", "deleted": true}, stub)

	status, raw, err = apiClient.Delete(departmentsPath + "/" + wc)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, map[string]any{"id": wc, "object": "work_center", "deleted": true}, parseJSON(raw))

	status, raw, err = apiClient.Delete(departmentsPath + "/" + wc)
	require.NoError(t, err)
	requireStatus(t, 410, status, raw)
}

func TestForgeWorkCenters_MachineDeleteBlockedByDowntime(t *testing.T) {
	t.Parallel()
	machine := newTestMachine(t)
	end := time.Now().UTC().Add(-time.Hour)
	event := logDowntimeOn(t, machine, "breakdown", end.Add(-time.Hour), &end, nil)

	status, raw, err := apiClient.Delete(machinesPath + "/" + machine)
	require.NoError(t, err)
	requireStatus(t, 409, status, raw)
	errObj := requireErrorResponse(t, raw, "resource_in_use", "invalid_request_error")
	assert.Contains(t, jsonField(errObj, "message"), "Archive the machine")

	deleteDowntime(t, jsonField(event, "id"))
	status, raw, err = apiClient.Delete(machinesPath + "/" + machine)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
}

// --- Idempotency ---

func TestForgeWorkCenters_CreateIdempotent(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()
	body := map[string]any{"name": uniqueName("e2e-wc-idem")}
	status, raw, err := apiClient.Post(departmentsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	first := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(departmentsPath + "/" + first) })

	status, raw, err = apiClient.Post(departmentsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	assert.Equal(t, first, jsonField(parseJSON(raw), "id"))
}
