//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/augno/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const departmentsBulkUpsertPath = departmentsPath + "/actions/bulk-upsert"

func bulkUpsertDepartments(t *testing.T, departments ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(departments))
	for i, d := range departments {
		rows[i] = d
	}
	status, body, err := apiClient.Post(departmentsBulkUpsertPath, map[string]any{"departments": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func cleanupDepartmentIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(departmentsPath + "/" + id)
		}
	}
}

// bulkUpsertDepartmentsJob posts a bulk upsert and returns the completed job. Bulk upsert is
// partial-success: a row that fails against existing rows lands in `errors` and the job completes.
func bulkUpsertDepartmentsJob(t *testing.T, departments ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertDepartments(t, departments...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// bulkUpsertDepartmentIDs posts a bulk upsert, follows the job to completion, and returns
// the created/updated department IDs from its results.
func bulkUpsertDepartmentIDs(t *testing.T, departments ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertDepartmentsJob(t, departments...)
	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

func TestDepartments_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	createdIDs, updatedIDs := bulkUpsertDepartmentIDs(t,
		map[string]any{"name": uniqueName("e2e-bup-dept-a")},
		map[string]any{"name": uniqueName("e2e-bup-dept-b")},
	)
	defer cleanupDepartmentIDs(createdIDs)

	require.Len(t, createdIDs, 2)
	for _, createdID := range createdIDs {
		assertIDFormat(t, createdID, id.DepartmentIDPrefix)
	}
	assert.Empty(t, updatedIDs)
}

// TestDepartments_BulkUpsert_CreateWithAllFields exercises the full create branch:
// notes and location.
func TestDepartments_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-dept-full")
	createdIDs, _ := bulkUpsertDepartmentIDs(t, map[string]any{
		"name":     name,
		"notes":    "full department notes",
		"location": map[string]any{"name": "Main Building"}, // resolved to SeedLocationID server-side
	})
	defer cleanupDepartmentIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(departmentsPath+"/"+createdIDs[0], url.Values{"include": {"location"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "full department notes", jsonField(got, "notes"))
	loc := jsonObject(got, "location")
	require.NotNil(t, loc, "location should be populated with ?include=location")
	assert.Equal(t, SeedLocationID, jsonField(loc, "id"))
}

// TestDepartments_BulkUpsert_ResolvesLocationByID: the location is a fuzzy reference —
// it resolves by id, not only by name.
func TestDepartments_BulkUpsert_ResolvesLocationByID(t *testing.T) {
	t.Parallel()

	createdIDs, _ := bulkUpsertDepartmentIDs(t, map[string]any{
		"name":     uniqueName("e2e-bup-dept-locid"),
		"location": map[string]any{"id": SeedLocationID},
	})
	defer cleanupDepartmentIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(departmentsPath+"/"+createdIDs[0], url.Values{"include": {"location"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	loc := jsonObject(parseJSON(getBody), "location")
	require.NotNil(t, loc)
	assert.Equal(t, SeedLocationID, jsonField(loc, "id"))
}

func TestDepartments_BulkUpsert_RejectsDuplicateNameInRequest(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-dept-dup")
	status, body := bulkUpsertDepartments(t,
		map[string]any{"name": name},
		map[string]any{"name": strings.ToUpper(name)}, // duplicate differing only by casing
	)
	requireStatus(t, 400, status, body)
	assert.Equal(t, "invalid_request_error", jsonField(jsonObject(parseJSON(body), "error"), "type"))
}

func TestDepartments_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(departmentsBulkUpsertPath, map[string]any{"departments": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestDepartments_BulkUpsert_RejectsUnknownLocation: references resolve at accept, so an
// unresolvable location is a synchronous 400 and no job is raised.
func TestDepartments_BulkUpsert_RejectsUnknownLocation(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertDepartments(t,
		map[string]any{"name": uniqueName("e2e-bup-dept-ok")},
		map[string]any{"name": uniqueName("e2e-bup-dept-badloc"), "location": map[string]any{"name": uniqueName("no-such-location")}},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "departments[1].location")
}

// TestDepartments_BulkUpsert_UpdatesEveryField creates a department, then upserts the
// same name (different casing) changing notes and location — asserting the match is
// case-insensitive, the ID is stable, the name adopts the new casing, and omitted
// fields are preserved on a subsequent update.
func TestDepartments_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-dept-upd")

	createdIDs, _ := bulkUpsertDepartmentIDs(t, map[string]any{
		"name":  name,
		"notes": "original notes",
	})
	defer cleanupDepartmentIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	departmentID := createdIDs[0]

	upper := strings.ToUpper(name)
	created, updated := bulkUpsertDepartmentIDs(t, map[string]any{
		"name":     upper,
		"notes":    "updated notes",
		"location": map[string]any{"name": "main building"}, // case-insensitive resolution
	})
	assert.Empty(t, created, "existing name must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, departmentID, updated[0])

	getStatus, getBody, err := apiClient.GetListRaw(departmentsPath+"/"+departmentID, url.Values{"include": {"location"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, upper, jsonField(got, "name"), "name adopts the request's casing")
	assert.Equal(t, "updated notes", jsonField(got, "notes"))
	loc := jsonObject(got, "location")
	require.NotNil(t, loc, "location should be populated with ?include=location")
	assert.Equal(t, SeedLocationID, jsonField(loc, "id"))

	// Omitted notes and location are preserved on a subsequent update.
	bulkUpsertDepartmentIDs(t, map[string]any{"name": upper})
	getStatus2, getBody2, err := apiClient.GetListRaw(departmentsPath+"/"+departmentID, url.Values{"include": {"location"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	got2 := parseJSON(getBody2)
	assert.Equal(t, "updated notes", jsonField(got2, "notes"))
	loc2 := jsonObject(got2, "location")
	require.NotNil(t, loc2, "location should be preserved when omitted on update")
	assert.Equal(t, SeedLocationID, jsonField(loc2, "id"))
}

func TestDepartments_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-bup-dept-mix-exist")
	newName := uniqueName("e2e-bup-dept-mix-new")

	seeded, _ := bulkUpsertDepartmentIDs(t, map[string]any{"name": existingName})
	defer cleanupDepartmentIDs(seeded)

	created, updated := bulkUpsertDepartmentIDs(t,
		map[string]any{"name": existingName, "notes": "touched"},
		map[string]any{"name": newName},
	)
	defer cleanupDepartmentIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}
