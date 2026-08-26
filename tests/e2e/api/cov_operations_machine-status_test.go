//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const machineStatusPath = "/v1/operations/machine-status"

func machineStatus(t *testing.T, params url.Values) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(machineStatusPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	out := []map[string]any{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row, ok := raw.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

// Every machine appears, whether or not the plan gave it work. A machine with nothing scheduled is idle, which is a state management needs to see rather than an absence.
func TestMachineStatus_ListsEveryMachineWithAState(t *testing.T) {
	t.Parallel()

	machines := machineStatus(t, nil)
	require.NotEmpty(t, machines, "the account has machines, so the floor is not empty")

	for _, machine := range machines {
		assert.Equal(t, "machine_status", jsonField(machine, "object"))
		machineRef := jsonObject(machine, "machine")
		require.NotNil(t, machineRef, "every row must name its machine: %v", machine)
		assert.Equal(t, "entity", jsonField(machineRef, "object"))
		assert.NotEmpty(t, jsonField(machineRef, "id"))
		assert.NotEmpty(t, jsonField(machineRef, "name"))
		assert.Contains(t, []string{"running", "idle", "down"}, jsonField(machine, "status"),
			"every machine must report a state")

		// current and next are always serialized so a caller can tell "no work" from "field not requested".
		_, hasCurrent := machine["current"]
		require.True(t, hasCurrent, "current must always be present, even when null")
		_, hasNext := machine["next"]
		require.True(t, hasNext, "next must always be present, even when null")
	}
}

// A released campaign is what the machine is on, and it carries its own progress so a floor display can show how far through the shift is.
func TestMachineStatus_ShowsProgressOnReleasedWork(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-machine-status"))
	scheduleID := jsonField(schedule, "id")

	// Only a published version drives the floor; a draft regenerating underneath a wall display would make machines appear to change job on their own.
	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, raw, result := releaseWeek(t, scheduleID, 0)
	if status != 201 {
		t.Skipf("week 0 is not releasable on this seed: %s", string(raw))
	}
	releasedLines := jsonListData(result, "lines")
	require.NotEmpty(t, releasedLines)

	machineIDs := map[string]bool{}
	for _, rawLine := range releasedLines {
		if line, ok := rawLine.(map[string]any); ok {
			machineIDs[jsonField(jsonObject(line, "machine"), "id")] = true
		}
	}

	sawRunning := false
	for _, machine := range machineStatus(t, nil) {
		if !machineIDs[jsonField(jsonObject(machine, "machine"), "id")] {
			continue
		}
		current, ok := machine["current"].(map[string]any)
		if !ok {
			continue
		}
		sawRunning = true

		planned, _ := current["planned_quantity"].(float64)
		scanned, _ := current["scanned_quantity"].(float64)
		remaining, _ := current["remaining_quantity"].(float64)

		assert.Positive(t, planned, "a current campaign must have something planned")
		assert.GreaterOrEqual(t, remaining, float64(0),
			"remaining never goes negative; an over-run shows up in scanned instead")
		assert.InDelta(t, planned-scanned, remaining, 0.001)
		assert.Positive(t, current["released_batch_count"],
			"a released campaign has issued batches to the floor")
		currentRun := jsonObject(current, "production_run")
		require.NotNil(t, currentRun, "a current campaign names the run carrying its work: %v", current)
		assert.NotEmpty(t, jsonField(currentRun, "id"))
	}

	require.True(t, sawRunning, "a released week must put its machines on a current campaign")
}

// Down outranks running: a broken machine is not producing whatever the plan says, and a display that showed it as running would be describing the plan rather than the plant.
func TestMachineStatus_DownOutranksRunning(t *testing.T) {
	t.Parallel()

	// Its own machine: a machine picked out of the account-wide list may belong to a parallel test that deletes it before this one posts.
	machineID := newTestMachine(t)

	resp, err := apiClient.PostFull(machineDowntimeEventsPath, map[string]any{
		"machine_id": machineID,
		"reason":     "breakdown",
		"started_at": time.Now().UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	eventID := jsonField(parseJSON(resp.Body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(machineDowntimeEventsPath + "/" + eventID) })

	for _, machine := range machineStatus(t, nil) {
		if jsonField(jsonObject(machine, "machine"), "id") != machineID {
			continue
		}
		assert.Equal(t, "down", jsonField(machine, "status"))

		downtime := jsonObject(machine, "downtime")
		require.NotNil(t, downtime, "a down machine must say why: %v", machine)
		event := jsonObject(downtime, "event")
		require.NotNil(t, event, "the downtime must name its event: %v", downtime)
		assert.Equal(t, eventID, jsonField(event, "id"))
		reason := jsonObject(downtime, "reason")
		require.NotNil(t, reason, "the downtime must say why: %v", downtime)
		assert.Equal(t, "breakdown", jsonField(reason, "code"))
		assert.NotEmpty(t, jsonField(downtime, "started_at"))
		return
	}
	t.Fatal("the machine disappeared from the floor after going down")
}

func TestMachineStatus_FiltersByDepartment(t *testing.T) {
	t.Parallel()

	all := machineStatus(t, nil)
	require.NotEmpty(t, all)

	departmentID := ""
	for _, machine := range all {
		if id := jsonField(jsonObject(machine, "department"), "id"); id != "" {
			departmentID = id
			break
		}
	}
	if departmentID == "" {
		t.Skip("no machine on this seed belongs to a department")
	}

	filtered := machineStatus(t, url.Values{"department_ids": {departmentID}})
	require.NotEmpty(t, filtered)
	for _, machine := range filtered {
		assert.Equal(t, departmentID, jsonField(jsonObject(machine, "department"), "id"))
	}
	assert.LessOrEqual(t, len(filtered), len(all))
}

// The whole point of the feature: a scan on the shop floor moves the schedule and the machine view without anyone touching the plan.
func TestMachineStatus_ScanningAdvancesProgress(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-scan-progress"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, raw, result := releaseWeek(t, scheduleID, 0)
	if status != 201 {
		t.Skipf("week 0 is not releasable on this seed: %s", string(raw))
	}
	run := jsonObject(result, "production_run")
	require.NotNil(t, run, "release must return its production run: %v", result)
	runID := jsonField(run, "id")

	status, body, err = apiClient.GetListRaw("/v1/operations/production-runs/"+runID+"/batches", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	batches := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, batches)

	batchID := jsonField(batches[0].(map[string]any), "id")

	before := lineProgressFor(t, scheduleID, 0)
	require.Positive(t, before.released, "a released week must have issued batches")
	assert.Zero(t, before.scanned, "nothing is scanned before the floor touches it")

	resp, err := apiClient.PostFull("/v1/operations/batches/actions/initialize", map[string]any{
		"batch_id":            batchID,
		"scanning_station_id": SeedScanningStationID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Skipf("this batch cannot be scanned at the seeded station: %s", string(resp.Body))
	}

	after := lineProgressFor(t, scheduleID, 0)
	assert.Equal(t, before.released, after.released, "scanning does not change what was issued")
	assert.Equal(t, before.scanned+1, after.scanned,
		"a scan must show up on the schedule without anyone editing the plan")
	assert.Greater(t, after.scannedQuantity, before.scannedQuantity,
		"the quantity made must move with the batch")
}

type weekProgress struct {
	released        int
	scanned         int
	scannedQuantity float64
}

func lineProgressFor(t *testing.T, scheduleID string, weekIndex int) weekProgress {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines",
		url.Values{"week_index": {"0"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	out := weekProgress{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		released, _ := row["released_batch_count"].(float64)
		scanned, _ := row["scanned_batch_count"].(float64)
		quantity, _ := row["scanned_quantity"].(float64)
		out.released += int(released)
		out.scanned += int(scanned)
		out.scannedQuantity += quantity
	}
	return out
}
