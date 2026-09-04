//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// weekReleasePreviewSkipping previews a week as if every batch were newly issued.
func weekReleasePreviewSkipping(t *testing.T, scheduleID string, weekIndex int) map[string]any {
	t.Helper()

	status, body, err := apiClient.GetListRaw(schedulePath(scheduleID)+"/week-release-preview", url.Values{
		"week_index":         {strconv.Itoa(weekIndex)},
		"skip_carry_forward": {"true"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// releasedBatchIDs is every batch a release put into its run, carried forward or newly created.
func releasedBatchIDs(result map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, rawLine := range jsonListData(result, "lines") {
		line, ok := rawLine.(map[string]any)
		if !ok {
			continue
		}
		for _, rawBatch := range jsonListData(line, "batches") {
			batch, ok := rawBatch.(map[string]any)
			if !ok {
				continue
			}
			if ref := jsonObject(batch, "batch"); ref != nil {
				if id := jsonField(ref, "id"); id != "" {
					out[id] = true
				}
			}
		}
	}
	return out
}

// carriedBatchIDs is only the batches a release moved off an earlier run.
func carriedBatchIDs(result map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, rawLine := range jsonListData(result, "lines") {
		line, ok := rawLine.(map[string]any)
		if !ok {
			continue
		}
		for _, rawBatch := range jsonListData(line, "batches") {
			batch, ok := rawBatch.(map[string]any)
			if !ok {
				continue
			}
			if jsonField(batch, "carried_forward_from") == "" {
				continue
			}
			if ref := jsonObject(batch, "batch"); ref != nil {
				if id := jsonField(ref, "id"); id != "" {
					out[id] = true
				}
			}
		}
	}
	return out
}

// releaseWeekWith releases a week with the carry-forward behavior spelled out.
func releaseWeekWith(t *testing.T, scheduleID string, weekIndex int, skipCarryForward bool) map[string]any {
	t.Helper()

	// A release mints batches, which moves on-hand — a solver input. Take the planning write lock so a solve never straddles the mutation (see planningMu).
	planningMu.Lock()
	defer planningMu.Unlock()

	body := map[string]any{
		"week_index":          weekIndex,
		"responsible_user_id": SeedAccountUserID,
	}
	if skipCarryForward {
		body["skip_carry_forward"] = true
	}

	resp, err := apiClient.PostFull(schedulePath(scheduleID)+"/actions/release-week", body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "must not 5xx: %s", string(resp.Body))
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// The whole point: a week that fell short leaves printed tickets on the floor, and the next week reuses them rather than issuing their replacements.
//
// Both campaigns sit in one version so the earlier week is unambiguously earlier. Carrying forward is scoped to weeks that have already begun, which is what stops a release raiding work nobody was late on.
func TestScheduleRelease_CarriesUnworkedBatchesForward(t *testing.T) {
	t.Parallel()

	// Its own item, because carrying forward is account-wide by item: sharing the seeded SKU would let a parallel release test claim these tickets, or these claim theirs.
	itemIDs := createItemsViaMaterials(t, uniqueName("e2e-carryfwd"), 1)
	require.Len(t, itemIDs, 1)
	itemID := itemIDs[0]

	schedule := ownedSchedule(t, uniqueName("e2e-carry-forward"))
	scheduleID := jsonField(schedule, "id")

	addLine(t, scheduleID, map[string]any{"week_index": 5, "item_id": itemID, "quantity": 180})
	addLine(t, scheduleID, map[string]any{"week_index": 6, "item_id": itemID, "quantity": 180})

	// Week 5 goes to the floor and nobody scans any of it.
	firstRun := releaseWeekWith(t, scheduleID, 5, false)
	firstBatches := releasedBatchIDs(firstRun)
	require.NotEmpty(t, firstBatches, "the first release must create batches: %v", firstRun)
	assert.Zero(t, firstRun["carried_forward_batch_count"], "nothing existed to carry into the first run")

	// Week 6 asks for the same work again, and the doffs for it are already printed.
	preview := weekReleasePreview(t, scheduleID, 6)
	carriedPreview, _ := preview["carried_forward_batch_count"].(float64)
	require.Positive(t, carriedPreview, "week 6 should reuse week 5's unworked tickets: %v", preview)

	secondRun := releaseWeekWith(t, scheduleID, 6, false)
	carried, _ := secondRun["carried_forward_batch_count"].(float64)
	require.Positive(t, carried, "the release must carry forward what the preview promised: %v", secondRun)

	carriedIDs := carriedBatchIDs(secondRun)
	require.NotEmpty(t, carriedIDs, "a carried batch must name itself, so the ticket can be matched to paper")
	for id := range carriedIDs {
		assert.True(t, firstBatches[id],
			"a carried batch must be one the earlier run created, not a new one: %s", id)
	}
}

// Skipping is a deliberate choice, and it has to change what the preview promises as well as what the release does — a preview that disagreed with the release is worse than no preview.
func TestScheduleRelease_SkipCarryForwardIssuesEverythingNew(t *testing.T) {
	t.Parallel()

	itemIDs := createItemsViaMaterials(t, uniqueName("e2e-carryskip"), 1)
	require.Len(t, itemIDs, 1)
	itemID := itemIDs[0]

	schedule := ownedSchedule(t, uniqueName("e2e-carry-skip"))
	scheduleID := jsonField(schedule, "id")

	addLine(t, scheduleID, map[string]any{"week_index": 7, "item_id": itemID, "quantity": 180})
	addLine(t, scheduleID, map[string]any{"week_index": 8, "item_id": itemID, "quantity": 180})

	firstRun := releaseWeekWith(t, scheduleID, 7, false)
	firstBatches := releasedBatchIDs(firstRun)
	require.NotEmpty(t, firstBatches)

	// The default would reuse; asking to skip must not.
	skipped := weekReleasePreviewSkipping(t, scheduleID, 8)
	assert.Zero(t, skipped["carried_forward_batch_count"],
		"skipping must preview as if nothing were reusable: %v", skipped)

	secondRun := releaseWeekWith(t, scheduleID, 8, true)
	assert.Zero(t, secondRun["carried_forward_batch_count"],
		"skipping must issue every batch new: %v", secondRun)
	assert.Empty(t, carriedBatchIDs(secondRun))

	// And the earlier run keeps its work rather than quietly losing it.
	for id := range releasedBatchIDs(secondRun) {
		assert.False(t, firstBatches[id], "a skipped release must not take the earlier run's batches: %s", id)
	}
}
