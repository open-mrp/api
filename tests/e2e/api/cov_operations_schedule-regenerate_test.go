//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// listScheduleLines returns every campaign on a version.
func listScheduleLines(t *testing.T, scheduleID string) []map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines", nil)
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

func previewRegeneratePath(scheduleID string) string {
	return schedulePath(scheduleID) + "/actions/preview-regenerate"
}

func regeneratePath(scheduleID string) string {
	return schedulePath(scheduleID) + "/actions/regenerate"
}

func previewRegenerate(t *testing.T, scheduleID string, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Put(previewRegeneratePath(scheduleID), body)
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

func regenerate(t *testing.T, scheduleID string, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Put(regeneratePath(scheduleID), body)
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

func TestScheduleRegenerate_PreviewChangesNothing(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-preview"))
	scheduleID := jsonField(schedule, "id")

	before := listScheduleLines(t, scheduleID)

	preview := previewRegenerate(t, scheduleID, map[string]any{})
	assert.Equal(t, "production_schedule_regenerate_preview", jsonField(preview, "object"))
	previewSchedule := jsonObject(preview, "production_schedule")
	require.NotNil(t, previewSchedule, "the preview must name the draft it would act on: %v", preview)
	assert.Equal(t, scheduleID, jsonField(previewSchedule, "id"))
	assert.NotEmpty(t, jsonField(preview, "solver_version"))
	assert.NotNil(t, preview["lines"], "lines must serialize as [] rather than null")

	// A preview that mutates is not a preview.
	after := listScheduleLines(t, scheduleID)
	require.Equal(t, len(before), len(after), "previewing must not change the plan")
}

// Re-solving the same inputs must agree with what is already stored, or the plan would churn every time anyone looked at it.
//
// planningMu keeps settings writes and releases off the same clock as the two solves, but firm demand is a solver input too and orders are issued all over this suite without any lock — an order landing between the generate and the preview legitimately moves a batch a week and shows up as one added plus one removed. That is a lost race rather than churn, so the pair is retried on a fresh draft: a solver that really is unstable drifts on every attempt, while a solve that merely straddled an order settles on the next one.
func TestScheduleRegenerate_PreviewOfUntouchedDraftIsQuiet(t *testing.T) {
	t.Parallel()

	const attempts = 3
	for attempt := 1; ; attempt++ {
		// One lock span across both solves, so at least the inputs planningMu does cover cannot move between them.
		drift := func() map[string]any {
			defer lockPlanningRead()()

			scheduleID := jsonField(ownedScheduleLocked(t, uniqueName("e2e-regen-quiet")), "id")
			preview := previewRegenerate(t, scheduleID, map[string]any{})

			for _, key := range []string{"added_count", "removed_count", "changed_count"} {
				value, ok := preview[key].(float64)
				require.True(t, ok, "%s must be present and numeric", key)
				if value != 0 {
					return preview
				}
			}
			// A hand edit is nobody else's to make, so this one never races.
			assert.Zero(t, preview["manual_line_count"], "an untouched draft has no hand edits")
			return nil
		}()

		if drift == nil {
			return
		}
		if attempt == attempts {
			t.Fatalf("re-solving an untouched draft still reported churn after %d attempts: added=%v removed=%v changed=%v lines=%v",
				attempts, drift["added_count"], drift["removed_count"], drift["changed_count"], drift["lines"])
		}
	}
}

func TestScheduleRegenerate_PreviewCountsWhatReplaceAllWouldDestroy(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-cost"))
	scheduleID := jsonField(schedule, "id")

	// A campaign the solver would never place: an unusual quantity in a late week.
	added := addLine(t, scheduleID, map[string]any{"week_index": 3, "quantity": 1234})
	require.NotEmpty(t, jsonField(added, "id"))

	preview := previewRegenerate(t, scheduleID, map[string]any{})

	manual, ok := preview["manual_line_count"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, manual, float64(1), "the hand-added campaign must be counted as manual")

	discarded, ok := preview["discarded_manual_count"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, discarded, float64(1),
		"replace_all would destroy the hand-added campaign, and the preview has to say so")
	assert.LessOrEqual(t, discarded, manual,
		"a regenerate cannot discard more hand edits than exist")
}

func TestScheduleRegenerate_PreserveManualKeepsHandEdits(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-preserve"))
	scheduleID := jsonField(schedule, "id")

	added := addLine(t, scheduleID, map[string]any{"week_index": 3, "quantity": 1234})
	addedID := jsonField(added, "id")

	regenerate(t, scheduleID, map[string]any{"merge_mode": "preserve_manual"})

	lines := listScheduleLines(t, scheduleID)
	found := false
	for _, line := range lines {
		if jsonField(line, "id") == addedID {
			found = true
			assert.EqualValues(t, 1234, line["planned_quantity"], "the hand-edited quantity must survive")
			assert.Equal(t, "manual", jsonField(line, "source"), "a preserved line stays manual")
		}
	}
	assert.True(t, found, "preserve_manual must keep the hand-added campaign")
}

func TestScheduleRegenerate_ReplaceAllDiscardsAndLogsIt(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-replace"))
	scheduleID := jsonField(schedule, "id")

	added := addLine(t, scheduleID, map[string]any{"week_index": 3, "quantity": 1234})
	addedID := jsonField(added, "id")

	before := len(listDeviations(t, scheduleID, nil))

	regenerate(t, scheduleID, map[string]any{"merge_mode": "replace_all"})

	for _, line := range listScheduleLines(t, scheduleID) {
		assert.NotEqual(t, addedID, jsonField(line, "id"), "replace_all must discard the hand edit")
	}

	// "Where did my change go" has to stay answerable after the plan no longer holds it.
	after := listDeviations(t, scheduleID, nil)
	assert.Greater(t, len(after), before,
		"discarding a hand edit must be written to the deviation log")

	sawRemoval := false
	for _, deviation := range after {
		if jsonField(deviation, "deviation_type") == "line_removed" {
			sawRemoval = true
		}
	}
	assert.True(t, sawRemoval, "the discard must be logged as a removal")
}

func TestScheduleRegenerate_DefaultsToPreservingHandEdits(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-default"))
	scheduleID := jsonField(schedule, "id")

	added := addLine(t, scheduleID, map[string]any{"week_index": 3, "quantity": 999})
	addedID := jsonField(added, "id")

	// No merge_mode at all: destroying work must never be what happens by accident.
	regenerate(t, scheduleID, map[string]any{})

	found := false
	for _, line := range listScheduleLines(t, scheduleID) {
		if jsonField(line, "id") == addedID {
			found = true
		}
	}
	assert.True(t, found, "omitting merge_mode must preserve hand edits, not discard them")
}

func TestScheduleRegenerate_KeepsVersionNumber(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-version"))
	scheduleID := jsonField(schedule, "id")
	version := schedule["version"]

	regenerated := regenerate(t, scheduleID, map[string]any{})

	assert.Equal(t, version, regenerated["version"],
		"a regenerate re-solves this version rather than minting a new one")
	assert.Equal(t, "draft", jsonField(regenerated, "status"))
	assert.NotEmpty(t, jsonField(regenerated, "solver_version"))
}

func TestScheduleRegenerate_RejectsPublished(t *testing.T) {
	t.Parallel()
	lockPublishing(t)

	schedule := ownedSchedule(t, uniqueName("e2e-regen-published"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// A published version is what the floor is working to. Re-solving it in place would change what a week was measured against after the fact.
	status, body, err = apiClient.Put(regeneratePath(scheduleID), map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "regenerating a published version must be rejected: %s", string(body))

	status, body, err = apiClient.Put(previewRegeneratePath(scheduleID), map[string]any{})
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "previewing a regenerate of a published version must be rejected too")
}

func TestScheduleRegenerate_RejectsUnknownMergeMode(t *testing.T) {
	t.Parallel()

	schedule := ownedSchedule(t, uniqueName("e2e-regen-badmode"))
	scheduleID := jsonField(schedule, "id")

	status, body, err := apiClient.Put(regeneratePath(scheduleID), map[string]any{
		"merge_mode": "keep_the_good_bits",
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status)
}
