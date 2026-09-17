//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// releasedCountsByLine reads a week's campaigns and returns each line's released batch count.
func releasedCountsByLine(t *testing.T, scheduleID string, weekIndex int) map[string]float64 {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/lines",
		url.Values{"week_index": {strconv.Itoa(weekIndex)}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	out := map[string]float64{}
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)
		released, _ := row["released_batch_count"].(float64)
		out[jsonField(row, "id")] = released
	}
	return out
}

// One item planned on two machines in the same week is two campaigns. Each has to report its own
// tickets; crediting by item alone gave both lines the pair's combined count.
func TestScheduleProgress_SameItemOnTwoMachinesCountsSeparately(t *testing.T) {
	t.Parallel()

	// Its own item, so no other release test's batches share the run and item.
	itemIDs := createItemsViaMaterials(t, uniqueName("e2e-progress-split"), 1)
	require.Len(t, itemIDs, 1)
	itemID := itemIDs[0]

	schedule := ownedSchedule(t, uniqueName("e2e-progress-split"))
	scheduleID := jsonField(schedule, "id")

	const week = 9
	small := addLine(t, scheduleID, map[string]any{"week_index": week, "item_id": itemID, "quantity": 120})
	large := addLine(t, scheduleID, map[string]any{
		"week_index": week, "item_id": itemID, "quantity": 360, "machine_id": newTestMachine(t),
	})

	run := releaseWeekWith(t, scheduleID, week, true)
	batchesByLine := map[string]int{}
	for _, raw := range jsonListData(run, "lines") {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		batchesByLine[jsonField(jsonObject(line, "line"), "id")] = len(jsonListData(line, "batches"))
	}
	smallID, largeID := jsonField(small, "id"), jsonField(large, "id")
	require.Positive(t, batchesByLine[smallID], "the smaller campaign was released: %v", run)
	require.Greater(t, batchesByLine[largeID], batchesByLine[smallID], "the larger campaign has more lots: %v", run)

	counts := releasedCountsByLine(t, scheduleID, week)
	assert.Equal(t, float64(batchesByLine[smallID]), counts[smallID], "a campaign counts only its own machine's tickets")
	assert.Equal(t, float64(batchesByLine[largeID]), counts[largeID], "a campaign counts only its own machine's tickets")

	// Moving a released campaign moves its tickets with it, rather than leaving it reading as unstarted.
	status, body, err := apiClient.Patch(schedulePath(scheduleID)+"/lines/"+smallID, map[string]any{
		"machine_id": newTestMachine(t),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	moved := releasedCountsByLine(t, scheduleID, week)
	assert.Equal(t, float64(batchesByLine[smallID]), moved[smallID], "the moved campaign keeps its tickets")
	assert.Equal(t, float64(batchesByLine[largeID]), moved[largeID], "the other campaign is untouched by the move")
}
