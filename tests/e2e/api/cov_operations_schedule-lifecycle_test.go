//go:build e2e

package api_test

import (
	"math"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// publishMu serializes every test that depends on its own schedule staying published.
//
// Publishing is account-global: it supersedes every published version overlapping the horizon. Two publishing tests running concurrently therefore supersede each other, and the loser's schedule becomes uneditable — which is the API behaving correctly and the tests fighting. Tests still run in parallel with the rest of the suite; they only serialize against each other.
var publishMu sync.Mutex

// lockPublishing claims the account-wide publish slot for the duration of one test.
func lockPublishing(t *testing.T) {
	t.Helper()
	publishMu.Lock()
	t.Cleanup(publishMu.Unlock)
}

const scheduleDeviationTypesPath = "/v1/operations/schedule-deviation-types"

func schedulePath(scheduleID string) string {
	return productionSchedulesPath + "/" + scheduleID
}

// ownedSchedule creates a fresh draft owned by this test and cleans it up afterwards.
//
// Every lifecycle test needs its own version: publishing supersedes whatever else covers the horizon, so sharing one would make the tests fight each other rather than test anything.
func ownedSchedule(t *testing.T, name string) map[string]any {
	t.Helper()
	return ownSchedule(t, generateSchedule(t, map[string]any{"name": name}))
}

// ownedScheduleLocked is ownedSchedule for callers already holding the planning lock (see generateScheduleLocked).
func ownedScheduleLocked(t *testing.T, name string) map[string]any {
	t.Helper()
	return ownSchedule(t, generateScheduleLocked(t, map[string]any{"name": name}))
}

func ownSchedule(t *testing.T, schedule map[string]any) map[string]any {
	t.Helper()

	id := jsonField(schedule, "id")
	require.NotEmpty(t, id)

	t.Cleanup(func() {
		// Only drafts delete; a published version is retired by archiving it.
		status, _, _ := apiClient.Delete(schedulePath(id))
		if status == 400 {
			_, _, _ = apiClient.Put(schedulePath(id)+"/actions/archive", map[string]any{})
		}
	})
	return schedule
}

func addLine(t *testing.T, scheduleID string, body map[string]any) map[string]any {
	t.Helper()

	full := map[string]any{
		"week_index": 2,
		"machine_id": SeedMachineID,
		"item_id":    SeedItemID,
		"quantity":   600,
	}
	for k, v := range body {
		full[k] = v
	}

	resp, err := apiClient.PostFull(schedulePath(scheduleID)+"/lines", full, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

func listDeviations(t *testing.T, scheduleID string, params url.Values) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/deviations", params)
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

// ──────────────────────────────────────────────
// Deviation types
// ──────────────────────────────────────────────

func TestScheduleDeviationTypes_ListReturnsSeededTaxonomy(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(scheduleDeviationTypesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	seen := map[string]bool{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		entry, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "schedule_deviation_type", jsonField(entry, "object"))
		seen[jsonField(entry, "code")] = true
	}

	for _, code := range []string{"line_added", "line_removed", "quantity_changed", "machine_changed"} {
		assert.True(t, seen[code], "deviation type %q must be seeded", code)
	}
}

// ──────────────────────────────────────────────
// Editing a draft
// ──────────────────────────────────────────────

func TestScheduleLifecycle_AddLineLogsDeviation(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-add-line"))
	scheduleID := jsonField(schedule, "id")

	line := addLine(t, scheduleID, nil)
	assert.Equal(t, "production_schedule_line", jsonField(line, "object"))
	// A hand-added line is manual from birth so a regenerate can tell it from solver output.
	assert.Equal(t, "manual", jsonField(line, "source"))
	assert.Equal(t, "flexible", jsonField(line, "freeze_status"), "a draft freezes nothing")

	deviations := listDeviations(t, scheduleID, nil)
	require.Len(t, deviations, 1)
	assert.Equal(t, "line_added", jsonField(deviations[0], "deviation_type"))
	assert.Equal(t, "flexible", jsonField(deviations[0], "freeze_status"))
	assertNilField(t, deviations[0], "before")

	after := jsonObject(deviations[0], "after")
	require.NotNil(t, after, "an added line must carry an after snapshot")
	assert.Equal(t, jsonField(line, "id"), jsonField(after, "id"))

	delta, ok := deviations[0]["delta_quantity"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 600, delta, 0.001)
}

func TestScheduleLifecycle_UpdateLineLogsSignedDelta(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-update-line"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 900}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	quantity, ok := updated["planned_quantity"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 900, quantity, 0.001)

	deviations := listDeviations(t, scheduleID, nil)
	require.Len(t, deviations, 2, "the add and the edit are both logged")

	// Newest first.
	edit := deviations[0]
	assert.Equal(t, "quantity_changed", jsonField(edit, "deviation_type"))
	delta, ok := edit["delta_quantity"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 300, delta, 0.001, "the delta is signed and relative to the previous value")

	before := jsonObject(edit, "before")
	require.NotNil(t, before, "an edit must carry a before snapshot")
	beforeQty, ok := before["planned_quantity"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 600, beforeQty, 0.001, "the before snapshot preserves the prior value")
}

func TestScheduleLifecycle_MoveLineToAnotherMachineIsClassified(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-move-machine"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	otherMachine := newTestMachine(t)

	status, body, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"machine_id": otherMachine}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.Equal(t, otherMachine, jsonField(jsonObject(parseJSON(body), "machine"), "id"))

	deviations := listDeviations(t, scheduleID, nil)
	require.NotEmpty(t, deviations)
	assert.Equal(t, "machine_changed", jsonField(deviations[0], "deviation_type"))
}

func TestScheduleLifecycle_DeleteLineKeepsSnapshot(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-delete-line"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Delete(schedulePath(scheduleID) + "/lines/" + lineID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	deviations := listDeviations(t, scheduleID, nil)
	require.NotEmpty(t, deviations)

	removal := deviations[0]
	assert.Equal(t, "line_removed", jsonField(removal, "deviation_type"))
	// The whole point of an append-only log: the record outlives the line.
	before := jsonObject(removal, "before")
	require.NotNil(t, before, "a removed line must leave its snapshot behind")
	assert.Equal(t, lineID, jsonField(before, "id"))
	assertNilField(t, removal, "after")

	delta, ok := removal["delta_quantity"].(float64)
	require.True(t, ok)
	assert.InDelta(t, -600, delta, 0.001, "removing units is a negative delta")
}

func TestScheduleLifecycle_AddLineRejectsWeekOutsideHorizon(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-bad-week"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.Post(schedulePath(scheduleID)+"/lines", map[string]any{
		"week_index": 99,
		"machine_id": SeedMachineID,
		"item_id":    SeedItemID,
		"quantity":   100,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "week_index")
}

func TestScheduleLines_CreateValidation_MissingFields(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-line-missing"))
	scheduleID := jsonField(schedule, "id")

	// week_index is deliberately not in this loop: omitting it yields a valid week-0 line.
	for _, field := range []string{"machine_id", "item_id", "quantity"} {
		body := map[string]any{
			"week_index": 2,
			"machine_id": SeedMachineID,
			"item_id":    SeedItemID,
			"quantity":   600,
		}
		delete(body, field)

		status, respBody, err := apiClient.Post(schedulePath(scheduleID)+"/lines", body, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing %s should be 400 or 422, got %d: %s", field, status, string(respBody))
	}
}

func TestScheduleLines_CreateResponseShape(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-line-shape"))
	scheduleID := jsonField(schedule, "id")

	line := addLine(t, scheduleID, nil)

	lineID := jsonField(line, "id")
	assertIDFormat(t, lineID, "pnscln")
	assertObjectField(t, line, "production_schedule_line")
	assert.Equal(t, scheduleID, jsonField(jsonObject(line, "production_schedule"), "id"))
	assert.EqualValues(t, 2, line["week_index"])
	assertValidTimestamp(t, jsonField(line, "week_starts_at"), "week_starts_at")
	assert.Equal(t, SeedMachineID, jsonField(jsonObject(line, "machine"), "id"))
	assert.Equal(t, SeedItemID, jsonField(jsonObject(line, "item"), "id"))
	assert.EqualValues(t, 600, line["planned_quantity"])
	assert.Equal(t, "planned", jsonField(line, "status"))
	assert.Equal(t, "manual", jsonField(line, "source"))
	assert.Equal(t, "flexible", jsonField(line, "freeze_status"))
	assertNilField(t, line, "production_run")
	assertValidTimestamp(t, jsonField(line, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(line, "updated_at"), "updated_at")

	// Every planning figure must always be serialized, even before any release or scan.
	for _, field := range []string{
		"planned_lots", "planned_lot_units", "planned_run_hours", "planned_changeover_minutes",
		"sequence_index", "projected_on_hand_before", "projected_on_hand_after",
		"released_batch_count", "scanned_batch_count", "scanned_quantity",
	} {
		_, ok := line[field].(float64)
		assert.True(t, ok, "%s must be present and numeric, got %v", field, line[field])
	}

	// Nullable references and display fields must be serialized even when null.
	for _, field := range []string{"planned_unit", "planned_unit_abbreviation", "production_step", "department", "reason"} {
		_, present := line[field]
		assert.True(t, present, "%s must always be serialized, even when null", field)
	}
}

func TestScheduleLines_CreateIdempotent(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-line-idem"))
	scheduleID := jsonField(schedule, "id")

	idemKey := newIdempotencyKey()
	body := map[string]any{
		"week_index": 2,
		"machine_id": SeedMachineID,
		"item_id":    SeedItemID,
		"quantity":   600,
	}

	resp1, err := apiClient.PostFull(schedulePath(scheduleID)+"/lines", body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp1.StatusCode, resp1.Body)
	id1 := jsonField(parseJSON(resp1.Body), "id")
	require.NotEmpty(t, id1)

	resp2, err := apiClient.PostFull(schedulePath(scheduleID)+"/lines", body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, resp2.StatusCode, resp2.Body)
	assert.Equal(t, id1, jsonField(parseJSON(resp2.Body), "id"),
		"a replayed create must return the line the first call made, not a second campaign")
}

func TestScheduleLines_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-line-idem-upd"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 900}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 900}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, jsonField(parseJSON(body1), "updated_at"), jsonField(parseJSON(body2), "updated_at"),
		"a replayed update must return the cached response rather than applying again")
}

func TestScheduleLines_UpdatePreservesOmittedFields(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-line-preserve"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 700}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	assert.EqualValues(t, 700, updated["planned_quantity"])
	assert.Equal(t, jsonField(jsonObject(line, "machine"), "id"),
		jsonField(jsonObject(updated, "machine"), "id"), "machine should be preserved")
	assert.Equal(t, jsonField(jsonObject(line, "item"), "id"),
		jsonField(jsonObject(updated, "item"), "id"), "item should be preserved")
	assert.Equal(t, line["week_index"], updated["week_index"], "week_index should be preserved")
	assert.Equal(t, "manual", jsonField(updated, "source"))
	assert.Equal(t, "flexible", jsonField(updated, "freeze_status"))
	assert.Equal(t, jsonField(line, "created_at"), jsonField(updated, "created_at"),
		"created_at should not change")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// Publish and freeze
// ──────────────────────────────────────────────

func TestScheduleLifecycle_PublishFreezesAndSnapshots(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-publish"))
	scheduleID := jsonField(schedule, "id")

	// One campaign in the frozen week and one well outside it.
	frozenLine := addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 400})
	laterLine := addLine(t, scheduleID, map[string]any{"week_index": 5, "quantity": 700})

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	published := parseJSON(body)
	assert.Equal(t, "published", jsonField(published, "status"))
	assert.NotEmpty(t, jsonField(published, "frozen_through_at"))
	assert.NotEmpty(t, jsonField(published, "published_at"))

	frozenCount, ok := published["frozen_line_count"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, frozenCount, float64(1), "the week-0 line must be counted as frozen")

	frozenQty, ok := published["frozen_planned_quantity"].(float64)
	require.True(t, ok)
	assert.Greater(t, frozenQty, float64(0), "the frozen quantity is snapshotted at publish")

	// Only the frozen week's lines are frozen.
	status, body, err = apiClient.GetListRaw(schedulePath(scheduleID)+"/lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	byID := map[string]map[string]any{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		if row, ok := raw.(map[string]any); ok {
			byID[jsonField(row, "id")] = row
		}
	}
	require.Contains(t, byID, jsonField(frozenLine, "id"))
	require.Contains(t, byID, jsonField(laterLine, "id"))
	assert.Equal(t, "frozen", jsonField(byID[jsonField(frozenLine, "id")], "freeze_status"), "week 0 freezes")
	assert.Equal(t, "flexible", jsonField(byID[jsonField(laterLine, "id")], "freeze_status"), "week 5 stays flexible")
}

func TestScheduleLifecycle_FrozenEditRequiresReason(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-frozen-reason"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 400})
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Without a reason the change is refused: an unexplained break of a commitment is exactly what the deviation log exists to prevent.
	status, body, err = apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 500}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "reason")

	// With one it succeeds and is recorded as a frozen-week deviation.
	status, body, err = apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 500, "reason": "rush_order", "reason_note": "Northwind pulled in"},
		newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	frozen := url.Values{}
	frozen.Set("frozen", "true")
	deviations := listDeviations(t, scheduleID, frozen)
	require.NotEmpty(t, deviations, "the frozen-week edit must be logged as frozen")

	assert.Equal(t, "frozen", jsonField(deviations[0], "freeze_status"))
	assert.Equal(t, "rush_order", jsonField(deviations[0], "reason"))
	assert.Equal(t, "Northwind pulled in", jsonField(deviations[0], "reason_note"))
	actor := jsonObject(deviations[0], "actor")
	require.NotNil(t, actor, "a frozen-week edit must record who made it: %v", deviations[0])
	assert.NotEmpty(t, jsonField(actor, "id"))
}

// An edit made BEFORE publish stays non-frozen forever. If the flag were derived at read time, publishing would retroactively reclassify it and adherence would drift.
func TestScheduleLifecycle_PublishDoesNotReclassifyEarlierDeviations(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-no-reclassify"))
	scheduleID := jsonField(schedule, "id")

	// Edit week 0 while it is still a draft, so no reason is required and nothing is frozen.
	addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 400})

	before := listDeviations(t, scheduleID, nil)
	require.Len(t, before, 1)
	require.Equal(t, "flexible", jsonField(before[0], "freeze_status"))
	deviationID := jsonField(before[0], "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	after := listDeviations(t, scheduleID, nil)
	require.NotEmpty(t, after)

	var found map[string]any
	for _, d := range after {
		if jsonField(d, "id") == deviationID {
			found = d
		}
	}
	require.NotNil(t, found, "the pre-publish deviation must still be there")
	assert.Equal(t, "flexible", jsonField(found, "freeze_status"),
		"publishing must not retroactively mark an earlier edit as frozen")
}

func TestScheduleLifecycle_PublishSupersedesPrevious(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	first := ownedSchedule(t, uniqueName("e2e-supersede-1"))
	firstID := jsonField(first, "id")
	addLine(t, firstID, map[string]any{"week_index": 0, "quantity": 100})

	status, body, err := apiClient.Put(schedulePath(firstID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	second := ownedSchedule(t, uniqueName("e2e-supersede-2"))
	secondID := jsonField(second, "id")
	addLine(t, secondID, map[string]any{"week_index": 0, "quantity": 200})

	status, body, err = apiClient.Put(schedulePath(secondID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// The first version becomes history pointed at its replacement, not a rewritten row.
	status, body, err = apiClient.GetListRaw(schedulePath(firstID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	superseded := parseJSON(body)
	assert.Equal(t, "superseded", jsonField(superseded, "status"))
	// The pointer names whichever version published last over this horizon, which under parallel tests is not necessarily this test's second version. What is guaranteed — and what matters — is that the old version stopped being live and says what replaced it.
	supersededByRef := jsonObject(superseded, "superseded_by")
	require.NotNil(t, supersededByRef, "a superseded version must point at its replacement: %v", superseded)
	supersededBy := jsonField(supersededByRef, "id")
	assert.NotEmpty(t, supersededBy, "a superseded version must point at its replacement")
	assert.NotEqual(t, firstID, supersededBy, "a version cannot supersede itself")
	assert.NotEmpty(t, jsonField(superseded, "frozen_through_at"),
		"a superseded version keeps the freeze it was published with")

	// The version this test published is the one it should still be able to read back.
	status, body, err = apiClient.GetListRaw(schedulePath(secondID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Contains(t, []string{"published", "superseded"}, jsonField(parseJSON(body), "status"))
}

func TestScheduleLifecycle_PublishTwiceIsRejected(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-double-publish"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 100})

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// ──────────────────────────────────────────────
// Archive and delete
// ──────────────────────────────────────────────

func TestScheduleLifecycle_ArchiveRetiresVersion(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-archive"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/archive", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assert.Equal(t, "archived", jsonField(parseJSON(body), "status"))
}

func TestScheduleLifecycle_DeleteDraftSucceeds(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-delete-draft"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.Delete(schedulePath(scheduleID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw(schedulePath(scheduleID), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// A published version is the baseline attainment is measured against, so deleting it would erase the record of what was promised.
func TestScheduleLifecycle_DeletePublishedIsRejected(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-delete-published"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 0, "quantity": 100})

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Delete(schedulePath(scheduleID))
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "Archive")
}

func TestScheduleLifecycle_EditArchivedIsRejected(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-edit-archived"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/archive", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 800}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// plannedItem returns a SKU the version actually planned, with the machine it runs on.
//
// The seeded finished goods are NOT this: only the constraint items the solver scheduled carry a measured rate, and those are exactly the rows the grid offers an empty cell on.
func plannedItem(t *testing.T, scheduleID string) (itemID, machineID string, secondsPerUnit float64) {
	t.Helper()

	for _, policy := range listItemPolicies(t, scheduleID) {
		seconds, _ := policy["seconds_per_unit"].(float64)
		machine := jsonField(jsonObject(policy, "primary_machine"), "id")
		if seconds > 0 && machine != "" {
			return jsonField(jsonObject(policy, "item"), "id"), machine, seconds
		}
	}

	t.Fatal("this version planned nothing with a measured rate — the seed must provide a constraint item the solver schedules, and every planned item's policy must carry its primary machine")
	return "", "", 0
}

// A campaign added by hand has to be priced in constraint time like any other, or the week's utilisation reads low — which is the very number a planner adds a campaign against.
func TestScheduleLifecycle_AddedLineIsPricedInConstraintTime(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-added-run-hours"))
	scheduleID := jsonField(schedule, "id")

	itemID, machineID, secondsPerUnit := plannedItem(t, scheduleID)

	line := addLine(t, scheduleID, map[string]any{
		"week_index": 6,
		"item_id":    itemID,
		"machine_id": machineID,
		"quantity":   600,
	})

	runHours, ok := line["planned_run_hours"].(float64)
	require.True(t, ok, "planned_run_hours must be present: %v", line)
	assert.InDelta(t, 600*secondsPerUnit/3600, runHours, 0.01,
		"a hand-added campaign must be costed from the version's own measured rate")

	// The same lot size the solver plans in, so releasing the week splits it identically.
	lotUnits, ok := line["planned_lot_units"].(float64)
	require.True(t, ok)
	assert.Positive(t, lotUnits,
		"a hand-added campaign must carry a lot size or it releases as one giant batch")

	lots, ok := line["planned_lots"].(float64)
	require.True(t, ok)
	assert.Positive(t, lots)
}

// Resizing a campaign has to reprice it. A campaign that keeps the hours it was first sized at makes the week's utilisation report work that is no longer planned.
func TestScheduleLifecycle_ResizingALineRepricesIt(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-resize-reprice"))
	scheduleID := jsonField(schedule, "id")

	itemID, machineID, secondsPerUnit := plannedItem(t, scheduleID)

	line := addLine(t, scheduleID, map[string]any{
		"week_index": 6,
		"item_id":    itemID,
		"machine_id": machineID,
		"quantity":   600,
	})
	lineID := jsonField(line, "id")
	lotUnits, ok := line["planned_lot_units"].(float64)
	require.True(t, ok)

	status, body, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 300}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	updated := parseJSON(body)
	runHours, ok := updated["planned_run_hours"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 300*secondsPerUnit/3600, runHours, 0.01,
		"halving the quantity must halve the machine time the campaign claims")

	lots, ok := updated["planned_lots"].(float64)
	require.True(t, ok)
	assert.InDelta(t, math.Round(300/lotUnits), lots, 0.001, "the lot count follows the quantity too")
}

// A campaign builds something by definition; zero is a campaign being removed, and the plan has to say so rather than keep a job that produces nothing while holding its machine time.
func TestScheduleLifecycle_ZeroQuantityIsRejected(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-zero-quantity"))
	scheduleID := jsonField(schedule, "id")
	line := addLine(t, scheduleID, nil)
	lineID := jsonField(line, "id")

	status, body, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+lineID,
		map[string]any{"quantity": 0}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// Overriding an empty slot is a create, and the plan has to show it in that week afterwards rather than only in the deviation log.
func TestScheduleLifecycle_AddLineFillsAnEmptyWeek(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-fill-empty-week"))
	scheduleID := jsonField(schedule, "id")
	itemID, machineID, _ := plannedItem(t, scheduleID)

	horizonWeeks, ok := schedule["horizon_weeks"].(float64)
	require.True(t, ok)

	// Find a week this SKU has nothing planned in — the case the grid could not previously express, because there was no cell to type into.
	emptyWeek := -1
	for week := int(horizonWeeks) - 1; week >= 1; week-- {
		status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines",
			url.Values{"week_index": {strconv.Itoa(week)}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		planned := false
		for _, raw := range jsonArray(parseJSON(body), "data") {
			if row, ok := raw.(map[string]any); ok && jsonField(jsonObject(row, "item"), "id") == itemID {
				planned = true
				break
			}
		}
		if !planned {
			emptyWeek = week
			break
		}
	}
	require.NotEqual(t, -1, emptyWeek, "the plan must leave this SKU idle in some week")

	created := addLine(t, scheduleID, map[string]any{
		"week_index": emptyWeek,
		"item_id":    itemID,
		"machine_id": machineID,
		"quantity":   180,
	})
	assert.Equal(t, "manual", jsonField(created, "source"),
		"a hand-added campaign is manual from birth, so a regenerate can tell it apart")

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines",
		url.Values{"week_index": {strconv.Itoa(emptyWeek)}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	found := false
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		if ok && jsonField(row, "id") == jsonField(created, "id") {
			found = true
			assert.Equal(t, float64(180), row["planned_quantity"])
		}
	}
	assert.True(t, found, "the added campaign must appear in the week it was added to")
}
