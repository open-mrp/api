//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scanningStationsBulkUpsertPath = scanningStationsPath + "/actions/bulk-upsert"

func bulkUpsertScanningStations(t *testing.T, stations ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(stations))
	for i, s := range stations {
		rows[i] = s
	}
	status, body, err := apiClient.Post(scanningStationsBulkUpsertPath, map[string]any{"scanning_stations": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func bulkStationRow(name string) map[string]any {
	return map[string]any{
		"name":                 name,
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department":           map[string]any{"name": seedDepartmentName},
	}
}

func cleanupScanningStationIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(scanningStationsPath + "/" + id)
		}
	}
}

// bulkUpsertScanningStationsJob posts a bulk upsert, requires the 202 job acknowledgment, and
// returns the completed job. Bulk upsert is partial-success: a row that fails against existing
// rows (an immutable department or type change) lands in the job's `errors` field, not failed —
// the job completes.
func bulkUpsertScanningStationsJob(t *testing.T, stations ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertScanningStations(t, stations...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// bulkUpsertScanningStationIDs posts a bulk upsert, follows the job to completion, and returns
// the created/updated station IDs from its results.
func bulkUpsertScanningStationIDs(t *testing.T, stations ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertScanningStationsJob(t, stations...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

func TestScanningStations_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	createdIDs, updatedIDs := bulkUpsertScanningStationIDs(t,
		bulkStationRow(uniqueName("e2e-bup-stn-a")),
		bulkStationRow(uniqueName("e2e-bup-stn-b")),
	)
	defer cleanupScanningStationIDs(createdIDs)

	require.Len(t, createdIDs, 2)
	for _, createdID := range createdIDs {
		assertIDFormat(t, createdID, id.ScanningStationIDPrefix)
	}
	assert.Empty(t, updatedIDs)
}

// TestScanningStations_BulkUpsert_CreateWithAllFields exercises the full create branch:
// notes, type, operator requirement, label codes, and case-insensitive department
// resolution.
func TestScanningStations_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-stn-full")
	createdIDs, _ := bulkUpsertScanningStationIDs(t, map[string]any{
		"name":                 name,
		"notes":                "full station notes",
		"type":                 "move_batch",
		"operator_requirement": "material_check",
		"label_size":           "1x1",
		"label_type":           "tag",
		"department":           map[string]any{"name": "knitting"}, // by name, case-insensitive
	})
	defer cleanupScanningStationIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+createdIDs[0], url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "full station notes", jsonField(got, "notes"))
	assert.Equal(t, "move_batch", jsonField(got, "type"))
	assert.Equal(t, "material_check", jsonField(got, "operator_requirement"))
	assert.Equal(t, "1x1", jsonField(got, "label_size"))
	assert.Equal(t, "tag", jsonField(got, "label_type"))
	dept := jsonObject(got, "department")
	require.NotNil(t, dept, "department should be populated with ?include=department")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
}

func TestScanningStations_BulkUpsert_RejectsDuplicateNameInRequest(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-stn-dup")
	status, body := bulkUpsertScanningStations(t,
		bulkStationRow(name),
		bulkStationRow(strings.ToUpper(name)), // duplicate differing only by casing
	)
	requireStatus(t, 400, status, body)
	assert.Equal(t, "invalid_request_error", jsonField(jsonObject(parseJSON(body), "error"), "type"))
}

func TestScanningStations_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(scanningStationsBulkUpsertPath, map[string]any{"scanning_stations": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestScanningStations_BulkUpsert_RejectsInvalidType: a bad enum is a structural error, so it
// fails synchronously before a job is raised.
func TestScanningStations_BulkUpsert_RejectsInvalidType(t *testing.T) {
	t.Parallel()

	row := bulkStationRow(uniqueName("e2e-bup-stn-badtype"))
	row["type"] = "teleport_batch"
	status, body := bulkUpsertScanningStations(t, row)
	requireStatus(t, 400, status, body)
	assert.Equal(t, "invalid_request_error", jsonField(jsonObject(parseJSON(body), "error"), "type"))
}

// TestScanningStations_BulkUpsert_RejectsUnknownDepartment: references resolve at accept, so an
// unresolvable department is a synchronous 400 naming the offending row and no job is raised.
func TestScanningStations_BulkUpsert_RejectsUnknownDepartment(t *testing.T) {
	t.Parallel()

	badRow := bulkStationRow(uniqueName("e2e-bup-stn-baddept"))
	badRow["department"] = map[string]any{"name": uniqueName("no-such-department")}
	status, body := bulkUpsertScanningStations(t,
		bulkStationRow(uniqueName("e2e-bup-stn-ok")),
		badRow,
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "scanning_stations[1].department")
}

// TestScanningStations_BulkUpsert_RejectsDepartmentMove: the department is immutable. The check
// needs the existing row, so it runs in the execute phase: the job completes and the row lands
// in `errors`, keyed to `department`.
func TestScanningStations_BulkUpsert_RejectsDepartmentMove(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-stn-xdept")
	seedIDs, _ := bulkUpsertScanningStationIDs(t, bulkStationRow(name)) // in Knitting
	defer cleanupScanningStationIDs(seedIDs)

	row := bulkStationRow(name)
	row["department"] = map[string]any{"name": "Washing"}
	job := bulkUpsertScanningStationsJob(t, row)
	assert.Empty(t, jobWrittenResults(job), "an immutable-department move must not be written")
	errs := jobErrors(job)
	require.Len(t, errs, 1)
	assert.Equal(t, "department", jsonField(jobRowError(errs[0]), "param"))
}

// TestScanningStations_BulkUpsert_RejectsTypeChange: the type is immutable. Like the department
// move, the check needs the existing row and surfaces as a per-row `errors` entry keyed to `type`.
func TestScanningStations_BulkUpsert_RejectsTypeChange(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-stn-retype")
	seedIDs, _ := bulkUpsertScanningStationIDs(t, bulkStationRow(name)) // init_batch
	defer cleanupScanningStationIDs(seedIDs)

	row := bulkStationRow(name)
	row["type"] = "split_batch"
	job := bulkUpsertScanningStationsJob(t, row)
	assert.Empty(t, jobWrittenResults(job), "an immutable-type change must not be written")
	errs := jobErrors(job)
	require.Len(t, errs, 1)
	assert.Equal(t, "type", jsonField(jobRowError(errs[0]), "param"))
}

// TestScanningStations_BulkUpsert_UpdatesEveryField creates a station, then upserts the
// same name (different casing) changing notes, operator requirement, and label codes —
// asserting the match is case-insensitive, the ID is stable, the name adopts the new
// casing, notes and label codes are preserved when omitted, and the type and
// department (stated as the station's current ones, as updates must) do not change.
func TestScanningStations_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-stn-upd")

	createRow := bulkStationRow(name)
	createRow["notes"] = "original notes"
	createRow["label_size"] = "1x1"
	createRow["label_type"] = "tag"
	createdIDs, _ := bulkUpsertScanningStationIDs(t, createRow)
	defer cleanupScanningStationIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	stationID := createdIDs[0]

	upper := strings.ToUpper(name)
	updRow := bulkStationRow(upper)
	updRow["notes"] = "updated notes"
	updRow["operator_requirement"] = "material_check"
	updRow["label_size"] = "2x4"
	updRow["label_type"] = "traveler"
	created, updated := bulkUpsertScanningStationIDs(t, updRow)
	assert.Empty(t, created, "existing name must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, stationID, updated[0])

	getStatus, getBody, err := apiClient.GetListRaw(scanningStationsPath+"/"+stationID, url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, upper, jsonField(got, "name"), "name adopts the request's casing")
	assert.Equal(t, "updated notes", jsonField(got, "notes"))
	assert.Equal(t, "material_check", jsonField(got, "operator_requirement"))
	assert.Equal(t, "2x4", jsonField(got, "label_size"))
	assert.Equal(t, "traveler", jsonField(got, "label_type"))
	assert.Equal(t, "init_batch", jsonField(got, "type"), "type is immutable and must not change")
	dept := jsonObject(got, "department")
	require.NotNil(t, dept)
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"), "department is create-only and must not change on update")

	// Omitted notes and label codes are preserved on a subsequent update.
	reRow := bulkStationRow(upper)
	reRow["operator_requirement"] = "material_check"
	bulkUpsertScanningStationIDs(t, reRow)
	getStatus2, getBody2, err := apiClient.GetListRaw(scanningStationsPath+"/"+stationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	got2 := parseJSON(getBody2)
	assert.Equal(t, "updated notes", jsonField(got2, "notes"))
	assert.Equal(t, "2x4", jsonField(got2, "label_size"))
	assert.Equal(t, "traveler", jsonField(got2, "label_type"))

	// An explicit null or empty string clears a label code (a blank spreadsheet
	// cell arrives as "").
	clearRow := bulkStationRow(upper)
	clearRow["operator_requirement"] = "material_check"
	clearRow["label_size"] = nil
	clearRow["label_type"] = ""
	bulkUpsertScanningStationIDs(t, clearRow)
	getStatus3, getBody3, err := apiClient.GetListRaw(scanningStationsPath+"/"+stationID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus3, getBody3)
	got3 := parseJSON(getBody3)
	assertNilField(t, got3, "label_size")
	assertNilField(t, got3, "label_type")
}

func TestScanningStations_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-bup-stn-mix-exist")
	newName := uniqueName("e2e-bup-stn-mix-new")

	seeded, _ := bulkUpsertScanningStationIDs(t, bulkStationRow(existingName))
	defer cleanupScanningStationIDs(seeded)

	updRow := bulkStationRow(existingName)
	updRow["notes"] = "touched"
	created, updated := bulkUpsertScanningStationIDs(t, updRow, bulkStationRow(newName))
	defer cleanupScanningStationIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}
