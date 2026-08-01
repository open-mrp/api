//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func weekReleasePreview(t *testing.T, scheduleID string, weekIndex int) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/week-release-preview",
		url.Values{"week_index": {strconv.Itoa(weekIndex)}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func releaseWeek(t *testing.T, scheduleID string, weekIndex int) (int, []byte, map[string]any) {
	t.Helper()

	// A release mints batches, which moves on-hand — a solver input. Take the planning write lock so a solve never straddles the mutation (see planningMu).
	planningMu.Lock()
	defer planningMu.Unlock()

	resp, err := apiClient.PostFull(schedulePath(scheduleID)+"/actions/release-week", map[string]any{
		"week_index":          weekIndex,
		"responsible_user_id": SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	return resp.StatusCode, resp.Body, parseJSON(resp.Body)
}

// The point of the feature: a week arrives on the floor as the doffs it will actually be knitted in, not as one undifferentiated instruction.
func TestScheduleRelease_SplitsCampaignsIntoLots(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-lots"))
	scheduleID := jsonField(schedule, "id")

	// A quantity that is a whole number of 60-unit lots, so the split is unambiguous.
	line := addLine(t, scheduleID, map[string]any{"week_index": 3, "quantity": 360})
	require.NotEmpty(t, jsonField(line, "id"))

	preview := weekReleasePreview(t, scheduleID, 3)
	require.Equal(t, "true", jsonField(preview, "is_releasable"), "a planned week must be releasable: %v", preview)

	lines := jsonListData(preview, "lines")
	require.NotEmpty(t, lines)

	var sawLottedCampaign bool
	for _, raw := range lines {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		lotUnits, _ := row["lot_units"].(float64)
		planned, _ := row["planned_quantity"].(float64)
		batchCount, _ := row["batch_count"].(float64)
		require.Positive(t, batchCount, "every campaign must break into at least one batch")

		batches := jsonListData(row, "batches")
		assert.Len(t, batches, int(batchCount), "batch_count must match the batches listed")

		var total float64
		for _, rawBatch := range batches {
			batch, ok := rawBatch.(map[string]any)
			if !ok {
				continue
			}
			quantity, _ := batch["quantity"].(float64)
			// Splitting must never invent or lose units, so no lot may exceed the lot size.
			if lotUnits > 0 {
				assert.LessOrEqual(t, quantity, lotUnits+1e-6,
					"no lot may be larger than the lot size the campaign was planned at")
			}
			total += quantity
		}
		assert.InDelta(t, planned, total, 1e-6,
			"the lots must add back up to the planned quantity")

		if lotUnits > 0 && planned > lotUnits {
			sawLottedCampaign = true
			assert.Equal(t, int(batchCount), len(batches))
		}
	}

	require.True(t, sawLottedCampaign,
		"the seeded plan must contain at least one campaign larger than a single lot")
}

func TestScheduleRelease_CreatesRunAndLinksLines(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-run"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 4, "quantity": 360})

	preview := weekReleasePreview(t, scheduleID, 4)
	expectedBatches, _ := preview["batch_count"].(float64)
	require.Positive(t, expectedBatches)

	status, raw, result := releaseWeek(t, scheduleID, 4)
	requireStatus(t, 201, status, raw)

	assert.Equal(t, "production_schedule_week_release", jsonField(result, "object"))

	run := jsonObject(result, "production_run")
	require.NotNil(t, run, "a release must return the run it created: %v", result)
	runID := jsonField(run, "id")
	require.NotEmpty(t, runID)

	actualBatches, _ := result["batch_count"].(float64)
	assert.Equal(t, expectedBatches, actualBatches,
		"the preview must promise exactly what the release delivers")

	// Every released line points at the run now carrying it, which is what stops the same week being released twice and what ties attainment back to the plan.
	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines",
		url.Values{"week_index": {"4"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	sawLinked := false
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		assert.Equal(t, runID, jsonField(jsonObject(row, "production_run"), "id"),
			"every line in a released week must name the run")
		assert.Equal(t, "released", jsonField(row, "status"))
		sawLinked = true
	}
	require.True(t, sawLinked)

	// The batches really exist under the run, rather than the response describing work that was never written.
	status, body, err = apiClient.GetListRaw("/v1/operations/production-runs/"+runID+"/batches", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Len(t, jsonArray(parseJSON(body), "data"), int(actualBatches))
}

// One click twice must not issue the same work to the floor twice.
func TestScheduleRelease_RefusesToReleaseTheSameWeekTwice(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-twice"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 5, "quantity": 120})

	status, raw, _ := releaseWeek(t, scheduleID, 5)
	requireStatus(t, 201, status, raw)

	status, _, _ = releaseWeek(t, scheduleID, 5)
	assert.Equal(t, 400, status, "a second release must fail rather than create a second run")

	preview := weekReleasePreview(t, scheduleID, 5)
	assert.Equal(t, "false", jsonField(preview, "is_releasable"))
	assert.NotEmpty(t, jsonField(preview, "blocked_reason"))
	existingRun := jsonObject(preview, "existing_production_run")
	require.NotNil(t, existingRun, "the preview must name the run the week is already tied to: %v", preview)
	assert.NotEmpty(t, jsonField(existingRun, "id"))
}

func TestScheduleRelease_EmptyWeekIsNotReleasable(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-empty"))
	scheduleID := jsonField(schedule, "id")

	horizonWeeks, ok := schedule["horizon_weeks"].(float64)
	require.True(t, ok)

	// Which weeks the solver leaves empty depends on the seeded stock, so the empty one is found rather than assumed. Pinning a week index would make this silently stop testing anything the moment the seed changes.
	emptyWeek := -1
	var preview map[string]any
	for week := int(horizonWeeks) - 1; week >= 0; week-- {
		candidate := weekReleasePreview(t, scheduleID, week)
		if jsonField(candidate, "is_releasable") != "true" {
			emptyWeek = week
			preview = candidate
			break
		}
	}
	require.NotEqual(t, -1, emptyWeek, "a 13-week horizon must leave at least one week unplanned")

	assert.Zero(t, preview["batch_count"])
	assert.NotEmpty(t, jsonField(preview, "blocked_reason"))

	status, body, _ := releaseWeek(t, scheduleID, emptyWeek)
	assert.Equal(t, 400, status,
		"releasing an empty week must fail rather than create an empty run: %s", string(body))
}

func TestScheduleRelease_RejectsWeekOutsideHorizon(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-horizon"))
	scheduleID := jsonField(schedule, "id")

	horizonWeeks, ok := schedule["horizon_weeks"].(float64)
	require.True(t, ok)

	status, _, _ := releaseWeek(t, scheduleID, int(horizonWeeks)+5)
	assert.Equal(t, 400, status)

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/week-release-preview",
		url.Values{"week_index": {strconv.Itoa(int(horizonWeeks) + 5)}})
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status)
}

// The lot size is snapshotted onto the line at plan time rather than re-read at release, so the run always matches the lots shown on the plan grid.
func TestScheduleLines_CarryTheLotSizeTheyWerePlannedAt(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(schedulePath(SeedProductionScheduleID)+"/lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, rows)

	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		_, present := row["planned_lot_units"]
		require.True(t, present, "planned_lot_units must always be serialized")
	}
}

// A quantity with no unit is uninterpretable: 360 pairs and 360 eaches are different weeks. The plan carries the unit on every line and on every policy so the grid can say which.
func TestScheduleLines_CarryTheUnitTheyArePlannedIn(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-plan-units"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, lines, "a generated version must carry lines")

	for _, raw := range lines {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		_, present := row["planned_unit"]
		require.True(t, present, "planned_unit must always be serialized, even when null")
		assert.Equal(t, "pr", jsonField(row, "planned_unit_abbreviation"),
			"the seeded plan knits sock greige in pairs, so its campaigns are counted in pairs")
	}

	for _, policy := range listItemPolicies(t, scheduleID) {
		_, present := policy["unit"]
		require.True(t, present, "unit must always be serialized on a policy")
		assert.Equal(t, "pr", jsonField(policy, "unit_abbreviation"),
			"a reorder point is uninterpretable without the unit it is counted in")
	}
}

// A released batch is a ticket the floor has yet to run.
//
// Stamping it as scanned on creation both fabricates production that never happened and closes the run immediately, because a run completes once every batch is scanned — so the week arrives already finished and the floor never sees it.
func TestScheduleRelease_CreatesUnscannedWork(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-unscanned"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 7, "quantity": 180})

	status, raw, result := releaseWeek(t, scheduleID, 7)
	requireStatus(t, 201, status, raw)

	run := jsonObject(result, "production_run")
	require.NotNil(t, run, "a release must return its production run: %v", result)
	runID := jsonField(run, "id")

	// The run itself must still be open; a run that completes on creation is not a run.
	assertNilField(t, run, "completed_at")

	status, body, err := apiClient.GetListRaw("/v1/operations/production-runs/"+runID+"/batches", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	batches := jsonArray(parseJSON(body), "data")
	require.NotEmpty(t, batches)
	for _, rawBatch := range batches {
		batch, ok := rawBatch.(map[string]any)
		if !ok {
			continue
		}
		assertNilField(t, batch, "scanned_at")
	}
}

// Attainment attributes production through the batch-to-machine link, so a released batch with no machine is work no machine ever gets credit for.
//
// Asserted through the machine filter on the runs list, which resolves runs by joining _batches_machines: a released run is findable by the machine it runs on only if its batches carry the link.
func TestScheduleRelease_LinksBatchesToTheirMachine(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-machines"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 8, "quantity": 120})

	preview := weekReleasePreview(t, scheduleID, 8)
	machineIDs := map[string]bool{}
	for _, raw := range jsonListData(preview, "lines") {
		if row, ok := raw.(map[string]any); ok {
			machineIDs[jsonField(jsonObject(row, "machine"), "id")] = true
		}
	}
	require.NotEmpty(t, machineIDs, "the week must name the machines it runs on")

	status, raw, result := releaseWeek(t, scheduleID, 8)
	requireStatus(t, 201, status, raw)
	run := jsonObject(result, "production_run")
	require.NotNil(t, run, "release must return its production run")
	runID := jsonField(run, "id")

	for machineID := range machineIDs {
		status, body, err := apiClient.GetListRaw("/v1/operations/production-runs",
			url.Values{"machine_ids": {machineID}, "limit": {"100"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		found := false
		for _, rawRun := range jsonArray(parseJSON(body), "data") {
			if row, ok := rawRun.(map[string]any); ok && jsonField(row, "id") == runID {
				found = true
			}
		}
		assert.True(t, found,
			"the released run must be findable by machine %s, which requires its batches to be linked to it",
			machineID)
	}
}

// Deleting the run has to give the week back. Otherwise it reads as issued with no run behind it, and the release guard refuses to ever issue it again.
func TestScheduleRelease_DeletingTheRunReturnsTheWeekToPlanned(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-release-delete"))
	scheduleID := jsonField(schedule, "id")
	addLine(t, scheduleID, map[string]any{"week_index": 9, "quantity": 120})

	status, raw, result := releaseWeek(t, scheduleID, 9)
	requireStatus(t, 201, status, raw)
	run := jsonObject(result, "production_run")
	require.NotNil(t, run, "release must return its production run")
	runID := jsonField(run, "id")

	status, body, err := apiClient.Delete("/v1/operations/production-runs/" + runID)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	require.Contains(t, []int{200, 204}, status)

	status, body, err = apiClient.GetListRaw(schedulePath(scheduleID)+"/lines",
		url.Values{"week_index": {"9"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, rawRow := range jsonArray(parseJSON(body), "data") {
		row, ok := rawRow.(map[string]any)
		if !ok {
			continue
		}
		assertNilField(t, row, "production_run")
		assert.Equal(t, "planned", jsonField(row, "status"))
	}

	// And the week can be released again, which is the whole point of giving it back.
	preview := weekReleasePreview(t, scheduleID, 9)
	assert.Equal(t, "true", jsonField(preview, "is_releasable"),
		"a week whose run was deleted must be releasable again: %v", preview["blocked_reason"])
}
