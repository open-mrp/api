//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Global, seeded units of time. A downtime duration has to be expressed in one of these, which is what stops a lot size being sent as a length of time.
const (
	minuteUnitID = "minute"
	hourUnitID   = "hour"
)

// A stoppage can be written up as "it lasted 45 minutes" instead of as a clock time, which is what a supervisor filling the form in afterwards actually knows.
func TestMachineDowntimeEvents_DurationSetsTheEnd(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-2 * time.Hour)
	created := logDowntimeOn(t, machineID, "breakdown", startedAt, nil, map[string]any{
		"duration": map[string]any{"value": "45", "unit_id": minuteUnitID},
	})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	duration, ok := created["duration_seconds"].(float64)
	require.True(t, ok, "a duration must close the event and materialize its length: %v", created)
	assert.InDelta(t, 2700, duration, 1)

	endedAt := jsonField(created, "ended_at")
	require.NotEmpty(t, endedAt, "the end is derived from the start plus the duration")
	parsed, err := time.Parse(time.RFC3339, endedAt)
	require.NoError(t, err)
	assert.WithinDuration(t, startedAt.Add(45*time.Minute), parsed, time.Second)
}

// The unit is the point. Ninety minutes and an hour and a half are the same stoppage written two ways, and both have to land on the same number of seconds.
func TestMachineDowntimeEvents_DurationUnitScalesTheLength(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-4 * time.Hour)
	created := logDowntimeOn(t, machineID, "planned_maintenance", startedAt, nil, map[string]any{
		"duration": map[string]any{"value": "1.5", "unit_id": hourUnitID},
	})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	duration, ok := created["duration_seconds"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 5400, duration, 1, "1.5 hours is 5400 seconds")
}

// Sending both is a caller bug. Picking a winner would hide it, and the two disagreeing would silently move a machine's availability.
func TestMachineDowntimeEvents_DurationAndEndTogetherIsRejected(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-time.Hour)
	endedAt := startedAt.Add(30 * time.Minute)

	resp, err := apiClient.PostFull(machineDowntimeEventsPath, map[string]any{
		"machine_id": machineID,
		"reason":     "breakdown",
		"started_at": rfc3339(startedAt),
		"ended_at":   rfc3339(endedAt),
		"duration":   map[string]any{"value": "30", "unit_id": minuteUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "a rejected request must not 5xx: %s", string(resp.Body))
	assert.Equal(t, 400, resp.StatusCode, "body: %s", string(resp.Body))
}

// A lot size sent as a duration would silently become a number of seconds, which is exactly the class of error carrying the unit is meant to make impossible.
func TestMachineDowntimeEvents_DurationRejectsANonTimeUnit(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)

	resp, err := apiClient.PostFull(machineDowntimeEventsPath, map[string]any{
		"machine_id": machineID,
		"reason":     "breakdown",
		"started_at": rfc3339(time.Now().UTC().Add(-time.Hour)),
		// A pair is a count of socks, not a length of time.
		"duration": map[string]any{"value": "60", "unit_id": SeedUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "a rejected request must not 5xx: %s", string(resp.Body))
	assert.Equal(t, 400, resp.StatusCode, "body: %s", string(resp.Body))
}

// Correcting a stoppage by its length is how somebody fixes "it was actually two hours, not one" without doing clock arithmetic.
func TestMachineDowntimeEvents_UpdateDurationRecomputesTheEnd(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-3 * time.Hour)
	endedAt := startedAt.Add(time.Hour)
	created := logDowntimeOn(t, machineID, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"duration": map[string]any{"value": "2", "unit_id": hourUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	duration, ok := updated["duration_seconds"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 7200, duration, 1)

	parsed, err := time.Parse(time.RFC3339, jsonField(updated, "ended_at"))
	require.NoError(t, err)
	assert.WithinDuration(t, startedAt.Add(2*time.Hour), parsed, time.Second)
}

// A duration is applied against the start as the same request leaves it, not against the start the event had before.
func TestMachineDowntimeEvents_UpdateDurationAppliesToTheNewStart(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-5 * time.Hour)
	endedAt := startedAt.Add(time.Hour)
	created := logDowntimeOn(t, machineID, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	movedStart := startedAt.Add(90 * time.Minute)
	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"started_at": rfc3339(movedStart),
		"duration":   map[string]any{"value": "30", "unit_id": minuteUnitID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	parsed, err := time.Parse(time.RFC3339, jsonField(updated, "ended_at"))
	require.NoError(t, err)
	assert.WithinDuration(t, movedStart.Add(30*time.Minute), parsed, time.Second)
}

// Clearing the duration reopens the event, the same way clearing the end time does.
func TestMachineDowntimeEvents_ClearingDurationReopensTheEvent(t *testing.T) {
	t.Parallel()

	machineID := newTestMachine(t)
	startedAt := time.Now().UTC().Add(-time.Hour)
	endedAt := startedAt.Add(20 * time.Minute)
	created := logDowntimeOn(t, machineID, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"duration": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	reopened := parseJSON(body)
	assertNilField(t, reopened, "ended_at")
	assertNilField(t, reopened, "duration_seconds")
}

// Correcting the machine is the one thing a mis-logged event most often needs. Deleting and re-logging would lose who reported it and when.
func TestMachineDowntimeEvents_UpdateMovesTheEventToAnotherMachine(t *testing.T) {
	t.Parallel()

	fromMachine := newTestMachine(t)
	toMachine := newTestMachine(t)

	startedAt := time.Now().UTC().Add(-time.Hour)
	endedAt := startedAt.Add(15 * time.Minute)
	created := logDowntimeOn(t, fromMachine, "breakdown", startedAt, &endedAt, nil)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer deleteDowntime(t, id)

	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+id, map[string]any{
		"machine_id": toMachine,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The machine is an expandable, so the move is confirmed by reading it back with the include rather than off the patch response.
	getStatus, getBody, err := apiClient.GetListRaw(machineDowntimeEventsPath+"/"+id, url.Values{"include": {"machine", "department"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	moved := parseJSON(getBody)
	machine := jsonObject(moved, "machine")
	require.NotNil(t, machine, "the moved event must name its new machine: %v", moved)
	assert.Equal(t, toMachine, jsonField(machine, "id"))
}

// An open event must stay the only open event on its machine, however it got there. Two would double-count the same wall-clock window and drive Availability below zero.
func TestMachineDowntimeEvents_MovingOntoADownMachineIsRejected(t *testing.T) {
	t.Parallel()

	fromMachine := newTestMachine(t)
	toMachine := newTestMachine(t)

	// Both machines are down right now, on separate events.
	moving := logDowntimeOn(t, fromMachine, "breakdown", time.Now().UTC().Add(-30*time.Minute), nil, nil)
	movingID := jsonField(moving, "id")
	require.NotEmpty(t, movingID)
	defer deleteDowntime(t, movingID)

	blocking := logDowntimeOn(t, toMachine, "no_operator", time.Now().UTC().Add(-20*time.Minute), nil, nil)
	blockingID := jsonField(blocking, "id")
	require.NotEmpty(t, blockingID)
	defer deleteDowntime(t, blockingID)

	status, body, err := apiClient.Patch(machineDowntimeEventsPath+"/"+movingID, map[string]any{
		"machine_id": toMachine,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a rejected move must not 5xx: %s", string(body))
	assert.Equal(t, 409, status, "body: %s", string(body))
}
