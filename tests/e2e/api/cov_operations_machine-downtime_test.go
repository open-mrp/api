//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	machineDowntimeEventsPath  = "/v1/operations/machine-downtime-events"
	machineDowntimeReasonsPath = "/v1/operations/machine-downtime-reasons"
)

// rfc3339 renders a timestamp the way the API expects it.
func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// logDowntimeOn creates an event and returns its parsed body. endedAt nil leaves the event open, meaning the machine is still down.
func logDowntimeOn(t *testing.T, machineID, reason string, startedAt time.Time, endedAt *time.Time, extra map[string]any) map[string]any {
	t.Helper()

	body := map[string]any{
		"machine_id": machineID,
		"reason":     reason,
		"started_at": rfc3339(startedAt),
	}
	if endedAt != nil {
		body["ended_at"] = rfc3339(*endedAt)
	}
	for k, v := range extra {
		body[k] = v
	}

	resp, err := apiClient.PostFull(machineDowntimeEventsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// logDowntime logs against the seeded machine. Only safe for CLOSED events; anything that leaves an event open must use logDowntimeOn with its own machine.
func logDowntime(t *testing.T, reason string, startedAt time.Time, endedAt *time.Time, extra map[string]any) map[string]any {
	t.Helper()
	return logDowntimeOn(t, SeedMachineID, reason, startedAt, endedAt, extra)
}

func deleteDowntime(t *testing.T, id string) {
	t.Helper()
	status, body, err := apiClient.Delete(machineDowntimeEventsPath + "/" + id)
	require.NoError(t, err)
	// 404 is fine: a test that deletes explicitly still runs its deferred cleanup.
	if status != 200 && status != 204 && status != 404 {
		t.Fatalf("cleanup delete %s returned %d: %s", id, status, string(body))
	}
}

// newTestMachine creates a machine owned by this test and removes it afterwards.
//
// Tests that log an OPEN event must not share a machine: the API allows only one open event per machine, so parallel tests on the seeded machine would conflict with each other rather than exercise what they mean to.
func newTestMachine(t *testing.T) string {
	t.Helper()

	resp, err := apiClient.PostFull(machinesPath, map[string]any{
		"name":          uniqueName("e2e-downtime-mc"),
		"serial_number": uniqueName("SN-DT"),
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	id := jsonField(parseJSON(resp.Body), "id")
	require.NotEmpty(t, id)

	// Runs after the test's deferred downtime cleanup, so the machine is empty by then.
	t.Cleanup(func() {
		if status, body, err := apiClient.Delete(machinesPath + "/" + id); err == nil && status >= 500 {
			t.Errorf("cleanup delete machine %s returned %d: %s", id, status, string(body))
		}
	})
	return id
}

// ──────────────────────────────────────────────
// Reasons (the seeded, global taxonomy)
// ──────────────────────────────────────────────

func TestMachineDowntimeReasons_ListReturnsSeededTaxonomy(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(machineDowntimeReasonsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	list := parseJSON(body)
	reasons := jsonArray(list, "data")
	require.NotEmpty(t, reasons, "the downtime reason taxonomy must be seeded")

	byCode := map[string]map[string]any{}
	var lastSort float64 = -1
	for _, raw := range reasons {
		reason, ok := raw.(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "machine_downtime_reason", jsonField(reason, "object"))
		code := jsonField(reason, "code")
		assert.NotEmpty(t, code)
		assert.NotEmpty(t, jsonField(reason, "name"))
		byCode[code] = reason

		// Ordered for display; the logging dialog renders tiles in this order.
		sortOrder, ok := reason["sort_order"].(float64)
		require.True(t, ok, "sort_order should be numeric")
		assert.GreaterOrEqual(t, sortOrder, lastSort, "reasons must come back in ascending sort order")
		lastSort = sortOrder
	}

	// oee_bucket is the load-bearing field: it decides which OEE term a stoppage charges. If these drift, OEE silently changes meaning.
	expectedBuckets := map[string]string{
		"breakdown":           "availability",
		"changeover":          "availability",
		"material_shortage":   "availability",
		"no_operator":         "availability",
		"planned_maintenance": "availability",
		"minor_stop":          "performance",
		"quality_hold":        "quality",
		"no_schedule":         "not_scheduled",
	}
	for code, wantBucket := range expectedBuckets {
		reason, ok := byCode[code]
		require.True(t, ok, "reason %q must be seeded", code)
		assert.Equal(t, wantBucket, jsonField(reason, "oee_bucket"),
			"reason %q charges the wrong OEE term", code)
	}

	// Planned stoppages must be distinguishable from unplanned ones.
	assert.Equal(t, "planned", jsonField(byCode["planned_maintenance"], "planning_status"))
	assert.Equal(t, "unplanned", jsonField(byCode["breakdown"], "planning_status"))
}

// ──────────────────────────────────────────────
// Event CRUD
// ──────────────────────────────────────────────

func TestMachineDowntimeEvents_CRUD(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	endedAt := startedAt.Add(30 * time.Minute)

	created := logDowntime(t, "breakdown", startedAt, &endedAt, map[string]any{
		"note": "needle bar jam",
	})

	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	assert.Equal(t, "machine_downtime_event", jsonField(created, "object"))
	assert.Equal(t, "needle bar jam", jsonField(created, "note"))
	assertValidTimestamp(t, jsonField(created, "started_at"), "started_at")
	assertValidTimestamp(t, jsonField(created, "ended_at"), "ended_at")
	assertValidTimestamp(t, jsonField(created, "shift_at"), "shift_at")
	assert.Equal(t, "manual", jsonField(created, "source"))

	// The reason's display name and OEE bucket ride along as a sub-object so a list render needs no second lookup.
	reason := jsonObject(created, "reason")
	require.NotNil(t, reason, "every event must carry its reason: %v", created)
	assert.Equal(t, "machine_downtime_reason", jsonField(reason, "object"))
	assert.Equal(t, "breakdown", jsonField(reason, "code"))
	assert.Equal(t, "Breakdown", jsonField(reason, "name"))
	assert.Equal(t, "availability", jsonField(reason, "oee_bucket"))

	// Duration is materialized on close so aggregation sums a column.
	duration, ok := created["duration_seconds"].(float64)
	require.True(t, ok, "duration_seconds should be set once the event is closed")
	assert.InDelta(t, 1800, duration, 1, "30 minutes should be 1800 seconds")

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, id, jsonField(parseJSON(getBody), "id"))

	// UPDATE: reclassify and extend.
	newEnd := startedAt.Add(90 * time.Minute)
	patchStatus, patchBody, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"reason":   "material_shortage",
		"ended_at": rfc3339(newEnd),
		"note":     "actually waiting on yarn",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	updatedReason := jsonObject(updated, "reason")
	require.NotNil(t, updatedReason, "a reclassified event must carry its new reason: %v", updated)
	assert.Equal(t, "material_shortage", jsonField(updatedReason, "code"))
	assert.Equal(t, "Material Shortage", jsonField(updatedReason, "name"))
	assert.Equal(t, "actually waiting on yarn", jsonField(updated, "note"))

	updatedDuration, ok := updated["duration_seconds"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 5400, updatedDuration, 1, "duration must be recomputed when the end moves")

	// DELETE
	delStatus, delBody, err := apiClient.Delete(machineDowntimeEventsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus2, _, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2, "a deleted event must be gone")
}

// An open event is the "machine is down right now" state: no end, no duration.
func TestMachineDowntimeEvents_OpenEventHasNoEndOrDuration(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	created := logDowntimeOn(t, machineID, "no_operator", time.Now().UTC().Add(-10*time.Minute), nil, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	assertNilField(t, created, "ended_at")
	assertNilField(t, created, "duration_seconds")
}

// Closing an open event is how the floor ends a stoppage.
func TestMachineDowntimeEvents_CloseOpenEventMaterializesDuration(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-45 * time.Minute)
	created := logDowntimeOn(t, machineID, "breakdown", startedAt, nil, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	endedAt := startedAt.Add(45 * time.Minute)
	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"ended_at": rfc3339(endedAt),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	closed := parseJSON(body)
	duration, ok := closed["duration_seconds"].(float64)
	require.True(t, ok, "closing an event must materialize its duration")
	assert.InDelta(t, 2700, duration, 1)
}

func TestMachineDowntimeEvents_CreateResponseShape(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-30 * time.Minute)
	endedAt := startedAt.Add(10 * time.Minute)
	created := logDowntimeOn(t, machineID, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	assertIDFormat(t, id, "mcdt")
	assertObjectField(t, created, "machine_downtime_event")
	assertValidTimestamp(t, jsonField(created, "started_at"), "started_at")
	assertValidTimestamp(t, jsonField(created, "ended_at"), "ended_at")
	assertValidTimestamp(t, jsonField(created, "shift_at"), "shift_at")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
}

func TestMachineDowntimeEvents_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-3 * time.Hour)
	endedAt := startedAt.Add(time.Hour)

	created := logDowntimeOn(t, machineID, "breakdown", startedAt, &endedAt, map[string]any{
		"item_id": SeedItemID,
		"note":    "all-fields event",
	})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	// Assert EVERY field of the resource.
	assertIDFormat(t, id, "mcdt")
	assertObjectField(t, created, "machine_downtime_event")
	assertNilField(t, created, "machine")     // expandable, not included
	assertNilField(t, created, "department")  // expandable, not included
	assertNilField(t, created, "item")        // expandable, not included
	assertNilField(t, created, "reported_by") // expandable, not included
	reason := jsonObject(created, "reason")
	require.NotNil(t, reason, "every event must carry its reason: %v", created)
	assert.Equal(t, "machine_downtime_reason", jsonField(reason, "object"))
	assert.Equal(t, "breakdown", jsonField(reason, "code"))
	assert.Equal(t, "Breakdown", jsonField(reason, "name"))
	assert.Equal(t, "availability", jsonField(reason, "oee_bucket"))
	assertValidTimestamp(t, jsonField(created, "started_at"), "started_at")
	assertValidTimestamp(t, jsonField(created, "ended_at"), "ended_at")
	duration, ok := created["duration_seconds"].(float64)
	require.True(t, ok, "a closed event must materialize a duration")
	assert.InDelta(t, 3600, duration, 1)
	assertValidTimestamp(t, jsonField(created, "shift_at"), "shift_at")
	_, shiftCodePresent := created["shift_code"]
	assert.True(t, shiftCodePresent, "shift_code must always be serialized, even when null")
	assertNilField(t, created, "production_run")
	assertNilField(t, created, "batch")
	assertNilField(t, created, "schedule_line")
	assert.Equal(t, "all-fields event", jsonField(created, "note"))
	assert.Equal(t, "manual", jsonField(created, "source"))
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// UPDATE the changeable fields and re-assert updated and preserved values.
	newEnd := startedAt.Add(2 * time.Hour)
	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"reason":   "changeover",
		"ended_at": rfc3339(newEnd),
		"note":     "reclassified",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	updatedReason := jsonObject(updated, "reason")
	require.NotNil(t, updatedReason)
	assert.Equal(t, "changeover", jsonField(updatedReason, "code"))
	assert.Equal(t, "reclassified", jsonField(updated, "note"))
	assert.Equal(t, jsonField(created, "started_at"), jsonField(updated, "started_at"),
		"started_at should be preserved")
	updatedDuration, ok := updated["duration_seconds"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 7200, updatedDuration, 1, "duration must be recomputed when the end moves")
	assert.Equal(t, "manual", jsonField(updated, "source"), "source should be preserved")
	assert.Equal(t, jsonField(created, "created_at"), jsonField(updated, "created_at"),
		"created_at should not change")
}

func TestMachineDowntimeEvents_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		machineID := newTestMachine(t)
		created := logDowntimeOn(t, machineID, "breakdown", time.Now().UTC().Add(-15*time.Minute), nil, nil)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer deleteDowntime(t, id)

		assertObjectField(t, created, "machine_downtime_event")
		assertNilField(t, created, "ended_at")
		assertNilField(t, created, "duration_seconds")
		assertNilField(t, created, "note")
		assertNilField(t, created, "machine")
		assertNilField(t, created, "department")
		assertNilField(t, created, "item")
		assertNilField(t, created, "reported_by")
		assertNilField(t, created, "production_run")
		assertNilField(t, created, "batch")
		assertNilField(t, created, "schedule_line")
		assert.Equal(t, "manual", jsonField(created, "source"))
		assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		for _, field := range []string{"machine_id", "reason", "started_at"} {
			body := map[string]any{
				"machine_id": SeedMachineID,
				"reason":     "breakdown",
				"started_at": rfc3339(time.Now().UTC().Add(-time.Hour)),
				"ended_at":   rfc3339(time.Now().UTC().Add(-30 * time.Minute)),
			}
			delete(body, field)

			status, respBody, err := apiClient.Post(machineDowntimeEventsPath, body, newIdempotencyKey())
			require.NoError(t, err)
			assert.True(t, status == 400 || status == 422,
				"missing %s should return 400 or 422, got %d: %s", field, status, string(respBody))
		}
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		startedAt := time.Now().UTC().Add(-2 * time.Hour)
		endedAt := startedAt.Add(30 * time.Minute)
		created := logDowntime(t, "breakdown", startedAt, &endedAt, map[string]any{"note": "kept note"})
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer deleteDowntime(t, id)

		status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
			"reason": "changeover",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		updated := parseJSON(body)
		assert.Equal(t, "changeover", jsonField(jsonObject(updated, "reason"), "code"))
		assert.Equal(t, "kept note", jsonField(updated, "note"), "note should be preserved")
		assert.Equal(t, jsonField(created, "started_at"), jsonField(updated, "started_at"))
		assert.Equal(t, jsonField(created, "ended_at"), jsonField(updated, "ended_at"))
		assert.Equal(t, jsonField(created, "created_at"), jsonField(updated, "created_at"),
			"created_at should not change")
		assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
	})
}

func TestMachineDowntimeEvents_CreateIdempotent(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	idemKey := newIdempotencyKey()
	body := map[string]any{
		"machine_id": machineID,
		"reason":     "breakdown",
		"started_at": rfc3339(startedAt),
		"ended_at":   rfc3339(startedAt.Add(30 * time.Minute)),
	}

	resp1, err := apiClient.PostFull(machineDowntimeEventsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp1.StatusCode, resp1.Body)
	id1 := jsonField(parseJSON(resp1.Body), "id")
	require.NotEmpty(t, id1)
	defer deleteDowntime(t, id1)

	resp2, err := apiClient.PostFull(machineDowntimeEventsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp2.StatusCode, resp2.Body)
	assert.Equal(t, id1, jsonField(parseJSON(resp2.Body), "id"),
		"a replayed create must return the event the first call made")
}

func TestMachineDowntimeEvents_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	endedAt := startedAt.Add(30 * time.Minute)
	created := logDowntime(t, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id,
		map[string]any{"note": "idem"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id,
		map[string]any{"note": "idem"}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, jsonField(parseJSON(body1), "updated_at"), jsonField(parseJSON(body2), "updated_at"),
		"a replayed update must return the cached response rather than applying again")
}

func TestMachineDowntimeEvents_List(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	endedAt := startedAt.Add(10 * time.Minute)
	created := logDowntime(t, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	defer deleteDowntime(t, id)

	list, _, err := apiClient.GetList(machineDowntimeEventsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "the logged event must be listable")
}

func TestMachineDowntimeEvents_ListPagination(t *testing.T) {
	t.Parallel()

	// Both rows sit on a machine this test owns, so the walk is immune to parallel churn.
	machineID := newTestMachine(t)
	base := time.Now().UTC().Add(-6 * time.Hour)
	firstEnd := base.Add(20 * time.Minute)
	first := logDowntimeOn(t, machineID, "breakdown", base, &firstEnd, nil)
	firstID := jsonField(first, "id")
	defer deleteDowntime(t, firstID)

	secondStart := firstEnd.Add(10 * time.Minute)
	secondEnd := secondStart.Add(20 * time.Minute)
	second := logDowntimeOn(t, machineID, "changeover", secondStart, &secondEnd, nil)
	secondID := jsonField(second, "id")
	defer deleteDowntime(t, secondID)

	assertScopedCursorPagination(t, machineDowntimeEventsPath,
		url.Values{"machine_ids": {machineID}}, []string{firstID, secondID})
}

func TestMachineDowntimeEvents_ListSearchNoResults(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(machineDowntimeEventsPath, url.Values{"q": {"zzzznotadowntime99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// ──────────────────────────────────────────────
// Invariants
// ──────────────────────────────────────────────

// Two concurrent open events on one machine would double-count the same wall-clock window and drive Availability below zero.
func TestMachineDowntimeEvents_RejectsSecondOpenEventForSameMachine(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	first := logDowntimeOn(t, machineID, "breakdown", time.Now().UTC().Add(-20*time.Minute), nil, nil)
	firstID := jsonField(first, "id")
	require.NotEmpty(t, firstID)
	defer deleteDowntime(t, firstID)

	status, body, err := apiClient.Post(machineDowntimeEventsPath, map[string]any{
		"machine_id": machineID,
		"reason":     "no_operator",
		"started_at": rfc3339(time.Now().UTC()),
	}, newIdempotencyKey())
	require.NoError(t, err)

	assert.Less(t, status, 500, "a duplicate open event must be a client error, not a 5xx: %s", string(body))
	assert.Equal(t, 409, status,
		"a second open event on the same machine must conflict, got %d: %s", status, string(body))
}

// A closed event does not block a new one — the machine stopped, started, stopped again.
func TestMachineDowntimeEvents_AllowsNewEventAfterPreviousClosed(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	start := time.Now().UTC().Add(-3 * time.Hour)
	end := start.Add(30 * time.Minute)
	first := logDowntimeOn(t, machineID, "breakdown", start, &end, nil)
	firstID := jsonField(first, "id")
	defer deleteDowntime(t, firstID)

	second := logDowntimeOn(t, machineID, "changeover", end.Add(10*time.Minute), nil, nil)
	secondID := jsonField(second, "id")
	require.NotEmpty(t, secondID)
	defer deleteDowntime(t, secondID)
}

// An unknown reason code must be rejected rather than stored: the reason is what maps a stoppage onto an OEE term, so an unmapped code would fall outside every bucket.
func TestMachineDowntimeEvents_RejectsUnknownReasonCode(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(machineDowntimeEventsPath, map[string]any{
		"machine_id": SeedMachineID,
		"reason":     "not_a_real_reason",
		"started_at": rfc3339(time.Now().UTC().Add(-time.Hour)),
	}, newIdempotencyKey())
	require.NoError(t, err)

	assert.Less(t, status, 500, "an unknown reason must be a client error, not a 5xx: %s", string(body))
	assert.Equal(t, 400, status, "unknown reason should be 400, got %d: %s", status, string(body))
}

func TestMachineDowntimeEvents_RejectsEndBeforeStart(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().UTC().Add(-time.Hour)
	status, body, err := apiClient.Post(machineDowntimeEventsPath, map[string]any{
		"machine_id": SeedMachineID,
		"reason":     "breakdown",
		"started_at": rfc3339(startedAt),
		"ended_at":   rfc3339(startedAt.Add(-10 * time.Minute)),
	}, newIdempotencyKey())
	require.NoError(t, err)

	assert.Less(t, status, 500, "an inverted window must be a client error, not a 5xx: %s", string(body))
	assert.Equal(t, 400, status, "ended_at before started_at should be 400, got %d: %s", status, string(body))
}

func TestMachineDowntimeEvents_RejectsFutureStart(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(machineDowntimeEventsPath, map[string]any{
		"machine_id": SeedMachineID,
		"reason":     "breakdown",
		"started_at": rfc3339(time.Now().UTC().Add(24 * time.Hour)),
	}, newIdempotencyKey())
	require.NoError(t, err)

	assert.Less(t, status, 500, "a future start must be a client error, not a 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a start far in the future should be 400, got %d: %s", status, string(body))
}

func TestMachineDowntimeEvents_RejectsUnknownMachine(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(machineDowntimeEventsPath, map[string]any{
		"machine_id": "mc_00000000000000000000000000",
		"reason":     "breakdown",
		"started_at": rfc3339(time.Now().UTC().Add(-time.Hour)),
	}, newIdempotencyKey())
	require.NoError(t, err)

	assert.Less(t, status, 500, "an unknown machine must be a client error, not a 5xx: %s", string(body))
	assert.Contains(t, []int{400, 404}, status,
		"an unknown machine should be 400 or 404, got %d: %s", status, string(body))
}

func TestMachineDowntimeEvents_RetrieveUnknownIDIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/mcdt_0000000000000000000000", nil)
	require.NoError(t, err)
	assert.Less(t, status, 500, "an unknown id must not 5xx: %s", string(body))
	assert.Equal(t, 404, status)
}

// ──────────────────────────────────────────────
// Listing, filters and includes
// ──────────────────────────────────────────────

func TestMachineDowntimeEvents_ListAndFilters(t *testing.T) {
	t.Parallel()

	base := time.Now().UTC().Add(-6 * time.Hour)
	closedEnd := base.Add(20 * time.Minute)

	machineID := newTestMachine(t)
	closed := logDowntimeOn(t, machineID, "changeover", base, &closedEnd, nil)
	closedID := jsonField(closed, "id")
	defer deleteDowntime(t, closedID)

	open := logDowntimeOn(t, machineID, "material_shortage", time.Now().UTC().Add(-5*time.Minute), nil, nil)
	openID := jsonField(open, "id")
	defer deleteDowntime(t, openID)

	t.Run("unfiltered list includes both", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{"limit": {"100"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		ids := downtimeIDs(t, body)
		assert.Contains(t, ids, closedID)
		assert.Contains(t, ids, openID)
	})

	t.Run("open filter excludes closed events", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{
			"open":  {"true"},
			"limit": {"100"},
		})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		ids := downtimeIDs(t, body)
		assert.Contains(t, ids, openID, "the open event must be listed")
		assert.NotContains(t, ids, closedID, "a closed event must not appear under open=true")
	})

	t.Run("reason filter narrows to one code", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{
			"reasons": {"changeover"},
			"limit":   {"100"},
		})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		ids := downtimeIDs(t, body)
		assert.Contains(t, ids, closedID)
		assert.NotContains(t, ids, openID, "material_shortage must not match a changeover filter")
	})

	t.Run("machine filter matches the seeded machine", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{
			"machine_ids": {machineID},
			"limit":       {"100"},
		})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Contains(t, downtimeIDs(t, body), openID)
	})

	t.Run("machine filter excludes other machines", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{
			"machine_ids": {"mc_00000000000000000000000000"},
			"limit":       {"100"},
		})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Empty(t, downtimeIDs(t, body), "a machine with no downtime must return nothing")
	})

	t.Run("date window excludes events outside it", func(t *testing.T) {
		// A window entirely in the future contains neither event.
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{
			"start_date": {rfc3339(time.Now().UTC().Add(48 * time.Hour))},
			"limit":      {"100"},
		})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		ids := downtimeIDs(t, body)
		assert.NotContains(t, ids, openID)
		assert.NotContains(t, ids, closedID)
	})

	t.Run("results are newest first", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath, url.Values{"limit": {"100"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		var previous time.Time
		for i, raw := range jsonArray(parseJSON(body), "data") {
			event, ok := raw.(map[string]any)
			require.True(t, ok)
			startedAt, err := time.Parse(time.RFC3339, jsonField(event, "started_at"))
			require.NoError(t, err)
			if i > 0 {
				assert.False(t, startedAt.After(previous), "list must be ordered by started_at descending")
			}
			previous = startedAt
		}
	})
}

func TestMachineDowntimeEvents_Includes(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	created := logDowntimeOn(t, machineID, "breakdown", time.Now().UTC().Add(-30*time.Minute), nil, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	t.Run("expandables are null without include", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		event := parseJSON(body)
		assertNilField(t, event, "machine")
		assertNilField(t, event, "department")
		assertNilField(t, event, "item")
		assertNilField(t, event, "reported_by")
	})

	t.Run("machine expands", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id,
			url.Values{"include": {"machine"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		machine := jsonObject(parseJSON(body), "machine")
		require.NotNil(t, machine, "machine should expand")
		assert.Equal(t, machineID, jsonField(machine, "id"))
		assert.Equal(t, "machine", jsonField(machine, "object"))
	})

	t.Run("department expands from the machine's department", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id,
			url.Values{"include": {"department"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		// The department is denormalized onto the event at write time from the machine, so it must resolve without the caller passing it.
		department := jsonObject(parseJSON(body), "department")
		require.NotNil(t, department, "department should expand")
		assert.Equal(t, "department", jsonField(department, "object"))
	})

	// The reporter is recorded whatever the identity type — user, API key, or agent — so the expandable is a polymorphic actor rather than an account user, and it must resolve for every actor kind.
	t.Run("reported_by expands to an actor for any identity type", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id,
			url.Values{"include": {"reported_by"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		reporter := jsonObject(parseJSON(body), "reported_by")
		require.NotNil(t, reporter, "the actor that logged the event must always be recoverable")
		assert.Equal(t, "actor", jsonField(reporter, "object"))
		assert.NotEmpty(t, jsonField(reporter, "id"))
		assert.Contains(t, []string{"user", "api_key", "agent"}, jsonField(reporter, "type"),
			"the actor type must identify what kind of identity logged the event")
	})

	t.Run("multiple includes resolve together", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(machineDowntimeEventsPath,
			url.Values{"include": {"machine", "department"}, "limit": {"100"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		for _, raw := range jsonArray(parseJSON(body), "data") {
			event, ok := raw.(map[string]any)
			require.True(t, ok)
			if jsonField(event, "id") != id {
				continue
			}
			assert.NotNil(t, event["machine"], "machine should expand in list responses too")
			return
		}
		t.Fatalf("event %s not found in the list response", id)
	})
}

// downtimeIDs collects the event ids out of a list response.
func downtimeIDs(t *testing.T, body []byte) []string {
	t.Helper()
	var ids []string
	for _, raw := range jsonArray(parseJSON(body), "data") {
		event, ok := raw.(map[string]any)
		require.True(t, ok)
		ids = append(ids, jsonField(event, "id"))
	}
	return ids
}
