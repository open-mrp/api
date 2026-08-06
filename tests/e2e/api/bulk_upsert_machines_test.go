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

const machinesBulkUpsertPath = machinesPath + "/actions/bulk-upsert"

// seedDepartmentName is the name of SeedDepartmentID in the seed data.
const seedDepartmentName = "Knitting"

func bulkUpsertMachines(t *testing.T, machines ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(machines))
	for i, m := range machines {
		rows[i] = m
	}
	status, body, err := apiClient.Post(machinesBulkUpsertPath, map[string]any{"machines": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

func bulkMachineRow(name string) map[string]any {
	return map[string]any{
		"name":          name,
		"serial_number": uniqueName("sn"),
		"department":    map[string]any{"name": seedDepartmentName},
	}
}

func cleanupMachineIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(machinesPath + "/" + id)
		}
	}
}

// bulkUpsertMachinesJob posts a bulk upsert, requires the 202 job acknowledgment, and
// returns the completed job. Bulk upsert is partial-success: a row that fails an intent
// rule against existing machines lands in the job's `errors` field, not failed — the job
// completes.
func bulkUpsertMachinesJob(t *testing.T, machines ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertMachines(t, machines...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

// bulkUpsertMachineIDs posts a bulk upsert, follows the job to completion, and returns the
// created/updated machine IDs from its results.
func bulkUpsertMachineIDs(t *testing.T, machines ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	job := bulkUpsertMachinesJob(t, machines...)
	require.NotEmpty(t, jsonArray(job, "results"), "a completed job must carry results")
	return jobResultIDs(job)
}

// machineJobErrors reads the per-row failures a completed bulk-upsert job recorded.
func machineJobErrors(job map[string]any) []map[string]any {
	var out []map[string]any
	for _, raw := range jsonArray(job, "errors") {
		if m, ok := raw.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func TestMachines_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	createdIDs, updatedIDs := bulkUpsertMachineIDs(t,
		bulkMachineRow(uniqueName("e2e-bup-mach-a")),
		bulkMachineRow(uniqueName("e2e-bup-mach-b")),
	)
	defer cleanupMachineIDs(createdIDs)

	require.Len(t, createdIDs, 2)
	for _, createdID := range createdIDs {
		assertIDFormat(t, createdID, id.MachineIDPrefix)
	}
	assert.Empty(t, updatedIDs)
}

// TestMachines_BulkUpsert_CreateWithAllFields exercises the full create branch:
// serial number, notes, and case-insensitive department resolution.
func TestMachines_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-mach-full")
	serial := uniqueName("sn-full")
	createdIDs, _ := bulkUpsertMachineIDs(t, map[string]any{
		"name":          name,
		"serial_number": serial,
		"notes":         "full machine notes",
		"department":    map[string]any{"name": "knitting"}, // case-insensitive resolution
	})
	defer cleanupMachineIDs(createdIDs)
	require.Len(t, createdIDs, 1)

	getStatus, getBody, err := apiClient.GetListRaw(machinesPath+"/"+createdIDs[0], url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, serial, jsonField(got, "serial_number"))
	assert.Equal(t, "full machine notes", jsonField(got, "notes"))
	dept := jsonObject(got, "department")
	require.NotNil(t, dept, "department should be populated with ?include=department")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))
}

func TestMachines_BulkUpsert_RejectsDuplicateNameInRequest(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-mach-dup")
	status, body := bulkUpsertMachines(t,
		bulkMachineRow(name),
		bulkMachineRow(strings.ToUpper(name)), // duplicate differing only by casing
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "machines[1].name")
	assert.Contains(t, errObj["message"], "duplicate name")
}

func TestMachines_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(machinesBulkUpsertPath, map[string]any{"machines": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestMachines_BulkUpsert_RejectsUnknownDepartment: references resolve at accept, so an
// unresolvable department is a synchronous 400 naming the offending row and no job is raised.
func TestMachines_BulkUpsert_RejectsUnknownDepartment(t *testing.T) {
	t.Parallel()

	status, body := bulkUpsertMachines(t,
		bulkMachineRow(uniqueName("e2e-bup-mach-ok")),
		map[string]any{
			"name":          uniqueName("e2e-bup-mach-baddept"),
			"serial_number": uniqueName("sn"),
			"department":    map[string]any{"name": uniqueName("no-such-department")},
		},
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "machines[1].department")
}

func TestMachines_BulkUpsert_RejectsDuplicateSerialInRequest(t *testing.T) {
	t.Parallel()

	serial := uniqueName("sn-dup")
	rowA := bulkMachineRow(uniqueName("e2e-bup-mach-sd-a"))
	rowA["serial_number"] = serial
	rowB := bulkMachineRow(uniqueName("e2e-bup-mach-sd-b"))
	rowB["serial_number"] = strings.ToUpper(serial) // duplicate differing only by casing

	status, body := bulkUpsertMachines(t, rowA, rowB)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "machines[1].serial_number")
	assert.Contains(t, errObj["message"], "duplicate serial number")
}

// TestMachines_BulkUpsert_RenamesMachineKeepingSerial: a row whose serial number
// matches an existing machine but whose name does not RENAMES that machine — the
// serial is the matching identity.
func TestMachines_BulkUpsert_RenamesMachineKeepingSerial(t *testing.T) {
	t.Parallel()

	serial := uniqueName("sn-rename")
	row := bulkMachineRow(uniqueName("e2e-bup-mach-oldname"))
	row["serial_number"] = serial
	createdIDs, _ := bulkUpsertMachineIDs(t, row)
	defer cleanupMachineIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	machineID := createdIDs[0]

	newName := uniqueName("e2e-bup-mach-newname")
	renameRow := bulkMachineRow(newName)
	renameRow["serial_number"] = serial
	created, updated := bulkUpsertMachineIDs(t, renameRow)
	assert.Empty(t, created, "matching serial must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, machineID, updated[0])

	getStatus, getBody, err := apiClient.GetListRaw(machinesPath+"/"+machineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, newName, jsonField(got, "name"))
	assert.Equal(t, serial, jsonField(got, "serial_number"))
}

// TestMachines_BulkUpsert_RejectsNameCollisionAcrossDepartments: a row with an existing
// machine's name but a DIFFERENT department is an attempted create (the department
// signals a different machine was meant). The check needs the existing row, so it runs in
// the execute phase: the job completes and the row lands in `errors`, keyed to `name`.
func TestMachines_BulkUpsert_RejectsNameCollisionAcrossDepartments(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-mach-xdept-name")
	seedIDs, _ := bulkUpsertMachineIDs(t, bulkMachineRow(name)) // in Knitting
	defer cleanupMachineIDs(seedIDs)

	job := bulkUpsertMachinesJob(t, map[string]any{
		"name":          name,
		"serial_number": uniqueName("sn-other"),
		"department":    map[string]any{"name": "Washing"},
	})
	assert.Empty(t, jobResults(job), "the colliding row must not be written")
	errs := machineJobErrors(job)
	require.Len(t, errs, 1)
	assert.Equal(t, "name", jsonField(jobRowError(errs[0]), "param"))
}

// TestMachines_BulkUpsert_RejectsSerialCollisionAcrossDepartments: a row with an
// existing machine's serial number but a different name AND department is an attempted
// create, recorded as a per-row error keyed to `serial_number`.
func TestMachines_BulkUpsert_RejectsSerialCollisionAcrossDepartments(t *testing.T) {
	t.Parallel()

	serial := uniqueName("sn-xdept")
	row := bulkMachineRow(uniqueName("e2e-bup-mach-xdept-owner"))
	row["serial_number"] = serial
	seedIDs, _ := bulkUpsertMachineIDs(t, row) // in Knitting
	defer cleanupMachineIDs(seedIDs)

	job := bulkUpsertMachinesJob(t, map[string]any{
		"name":          uniqueName("e2e-bup-mach-xdept-new"),
		"serial_number": serial,
		"department":    map[string]any{"name": "Washing"},
	})
	assert.Empty(t, jobResults(job), "the colliding row must not be written")
	errs := machineJobErrors(job)
	require.Len(t, errs, 1)
	assert.Equal(t, "serial_number", jsonField(jobRowError(errs[0]), "param"))
}

// TestMachines_BulkUpsert_RejectsDepartmentMove: a row matching an existing machine by
// BOTH name and serial number but naming a different department is an illegal department
// move, recorded as a per-row error keyed to `department`.
func TestMachines_BulkUpsert_RejectsDepartmentMove(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-mach-move")
	serial := uniqueName("sn-move")
	row := bulkMachineRow(name)
	row["serial_number"] = serial
	seedIDs, _ := bulkUpsertMachineIDs(t, row) // in Knitting
	defer cleanupMachineIDs(seedIDs)

	job := bulkUpsertMachinesJob(t, map[string]any{
		"name":          name,
		"serial_number": serial,
		"department":    map[string]any{"name": "Washing"},
	})
	assert.Empty(t, jobResults(job), "an illegal move must not be written")
	errs := machineJobErrors(job)
	require.Len(t, errs, 1)
	assert.Equal(t, "department", jsonField(jobRowError(errs[0]), "param"))
}

// TestMachines_BulkUpsert_RejectsAmbiguousNameAndSerial: a row whose name matches one
// machine and whose serial matches a different machine is ambiguous. The check needs both
// existing rows, so it runs in the execute phase and lands in `errors`.
func TestMachines_BulkUpsert_RejectsAmbiguousNameAndSerial(t *testing.T) {
	t.Parallel()

	nameA := uniqueName("e2e-bup-mach-amb-a")
	serialB := uniqueName("sn-amb-b")
	rowA := bulkMachineRow(nameA)
	rowB := bulkMachineRow(uniqueName("e2e-bup-mach-amb-b"))
	rowB["serial_number"] = serialB
	seedIDs, _ := bulkUpsertMachineIDs(t, rowA, rowB)
	defer cleanupMachineIDs(seedIDs)

	job := bulkUpsertMachinesJob(t, map[string]any{
		"name":          nameA,
		"serial_number": serialB,
		"department":    map[string]any{"name": seedDepartmentName},
	})
	assert.Empty(t, jobResults(job), "an ambiguous row must not be written")
	errs := machineJobErrors(job)
	require.Len(t, errs, 1)
	assert.Equal(t, "name, serial_number", jsonField(jobRowError(errs[0]), "param"))
}

// TestMachines_BulkUpsert_UpdatesEveryField creates a machine, then upserts the same
// name (different casing) changing serial number and notes — asserting the match is
// case-insensitive, the ID is stable, the name adopts the new casing, notes are
// preserved when omitted, and the department (stated as the machine's current one, as
// updates must) does not change.
func TestMachines_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-mach-upd")

	createdIDs, _ := bulkUpsertMachineIDs(t, map[string]any{
		"name":          name,
		"serial_number": uniqueName("sn-orig"),
		"notes":         "original notes",
		"department":    map[string]any{"name": seedDepartmentName},
	})
	defer cleanupMachineIDs(createdIDs)
	require.Len(t, createdIDs, 1)
	machineID := createdIDs[0]

	upper := strings.ToUpper(name)
	newSerial := uniqueName("sn-upd")
	created, updated := bulkUpsertMachineIDs(t, map[string]any{
		"name":          upper,
		"serial_number": newSerial,
		"notes":         "updated notes",
		"department":    map[string]any{"name": seedDepartmentName}, // must state the machine's current department
	})
	assert.Empty(t, created, "existing name must update, not create")
	require.Len(t, updated, 1)
	assert.Equal(t, machineID, updated[0])

	getStatus, getBody, err := apiClient.GetListRaw(machinesPath+"/"+machineID, url.Values{"include": {"department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, upper, jsonField(got, "name"), "name adopts the request's casing")
	assert.Equal(t, newSerial, jsonField(got, "serial_number"))
	assert.Equal(t, "updated notes", jsonField(got, "notes"))
	dept := jsonObject(got, "department")
	require.NotNil(t, dept)
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"), "department is create-only and must not change on update")

	// Omitted notes are preserved on a subsequent update.
	bulkUpsertMachineIDs(t, map[string]any{
		"name":          upper,
		"serial_number": newSerial,
		"department":    map[string]any{"name": seedDepartmentName},
	})
	getStatus2, getBody2, err := apiClient.GetListRaw(machinesPath+"/"+machineID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus2, getBody2)
	assert.Equal(t, "updated notes", jsonField(parseJSON(getBody2), "notes"))
}

func TestMachines_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-bup-mach-mix-exist")
	newName := uniqueName("e2e-bup-mach-mix-new")

	seedIDs, _ := bulkUpsertMachineIDs(t, bulkMachineRow(existingName))
	defer cleanupMachineIDs(seedIDs)

	updRow := bulkMachineRow(existingName)
	updRow["notes"] = "touched"
	created, updated := bulkUpsertMachineIDs(t, updRow, bulkMachineRow(newName))
	defer cleanupMachineIDs(created)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}
