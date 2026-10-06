//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Scans that race: two stations scanning one batch, or a scanner firing twice. Each test sends the same
// scan several times at once, each under its own idempotency key, and asserts the batches and inventory
// end exactly as the same scans made one after another would leave them. A refused scan is refused
// with the message the station shows when the scan is repeated.

const scanRaceWidth = 4

type scanRaceResult struct {
	status int
	body   []byte
}

func (r scanRaceResult) String() string { return fmt.Sprintf("%d %s", r.status, r.body) }

// race makes n calls at once and returns what each got.
func race(n int, call func(i int) (int, []byte, error)) []scanRaceResult {
	results := make([]scanRaceResult, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, raw, err := call(i)
			if err != nil {
				raw = []byte(err.Error())
			}
			results[i] = scanRaceResult{status: status, body: raw}
		}()
	}
	close(start)
	wg.Wait()
	return results
}

func raceScan(action string, body map[string]any) []scanRaceResult {
	return race(scanRaceWidth, func(int) (int, []byte, error) {
		return apiClient.Post(batchesPath+"/actions/"+action, body, newIdempotencyKey())
	})
}

// requireScansWon asserts that exactly want of the raced scans went through and that every other one was
// refused with refusal, and returns the batches the winners made.
func requireScansWon(t *testing.T, action string, results []scanRaceResult, want int, refusal string) []map[string]any {
	t.Helper()
	var won []map[string]any
	for _, r := range results {
		if r.status == http.StatusCreated {
			batch := parseJSON(r.body)
			settleScanAtCleanup(t, action, jsonField(batch, "id"))
			won = append(won, batch)
		}
	}
	require.Len(t, won, want, "%d of %d racing %s scans should go through: %v", want, len(results), action, results)
	for _, r := range results {
		if r.status == http.StatusCreated {
			continue
		}
		require.Equal(t, http.StatusBadRequest, r.status, "a %s that lost the race is refused, not failed: %s", action, r)
		assert.Equal(t, refusal, jsonField(jsonObject(parseJSON(r.body), "error"), "message"), "a %s that lost the race", action)
	}
	return won
}

// repeatRefusal makes the scan once more, after the race, and returns the message it is refused with.
func repeatRefusal(t *testing.T, action string, body map[string]any) string {
	t.Helper()
	status, got, raw := postScan(t, action, body)
	requireStatus(t, http.StatusBadRequest, status, raw)
	return jsonField(jsonObject(got, "error"), "message")
}

func settleScans(t *testing.T, batchIDs ...string) {
	t.Helper()
	for _, id := range batchIDs {
		waitForMessagesSettled(t, "core.event.batch_scanned", "core.batch_scanned_inventory", id)
	}
}

func flowSize(t *testing.T, batchID string) int {
	t.Helper()
	status, raw, err := apiClient.GetListRaw(batchesPath+"/"+batchID+"/flow", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	return len(jsonArray(parseJSON(raw), "data"))
}

func TestBatchScanningRace_InitializeScansABatchOnce(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	body := map[string]any{"batch_id": planned, "scanning_station_id": c.initStation}

	results := raceScan("initialize", body)
	requireScansWon(t, "initialize", results, 1, repeatRefusal(t, "initialize", body))

	settleScans(t, planned)
	requireOneBatchScanned(t, planned)
	waitForOnHand(t, c.a.id, "10")
	waitForOnHand(t, c.material.id, "980")
}

func TestBatchScanningRace_MoveTakesABatchOnce(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	settleScans(t, planned)
	body := map[string]any{"batch_ids": []string{planned}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation}

	results := raceScan("move", body)
	won := requireScansWon(t, "move", results, 1, repeatRefusal(t, "move", body))
	moved := jsonField(won[0], "id")

	settleScans(t, moved)
	requireOneBatchScanned(t, moved)
	requireClosed(t, planned, planned)
	assert.Equal(t, 2, flowSize(t, planned), "the batch and the one batch it moved into")
	waitForOnHand(t, c.b.id, "10")
	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.material.id, "970")
}

// Splits may ask for more than is left of a batch, as on the dashboard: the split that uses it up takes
// what is left and closes it, and the ones after are refused. Racing splits must end the same way.
func TestBatchScanningRace_SplitsCloseTheBatchTheyUseUp(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")
	settleScans(t, planned, moved)
	body := splitBody(c.splitStation, c.splitStep, []string{planned}, "6", "", "", false)

	results := raceScan("split", body)
	won := requireScansWon(t, "split", results, 2, repeatRefusal(t, "split", body))

	settleScans(t, jsonField(won[0], "id"), jsonField(won[1], "id"))
	requireClosed(t, planned, moved)
	waitForOnHand(t, c.c.id, "12")
	waitForOnHand(t, c.b.id, "0")
}

func TestBatchScanningRace_MergeTakesTheBatchesOnce(t *testing.T) {
	t.Parallel()
	material := newScanMaterial(t)
	part := newScanPart(t, scanShippingCategory)
	out := newScanPart(t, scanShippingCategory)
	initStation := newScanStation(t, "init_batch")
	mergeStation := newScanStation(t, "merge_batch")
	newScanStep(t, initStation, scanQty{part.id, "1", unitEach})
	mergeStep := newScanStep(t, mergeStation, scanQty{out.id, "1", unitEach},
		scanConsumption{itemID: part.id, value: "1", unitID: unitEach},
		scanConsumption{itemID: material.id, value: "1", unitID: unitEach})
	stockScanItem(t, material.id, "50", unitEach)
	five := planScanBatch(t, part.id, "5", unitEach)
	seven := planScanBatch(t, part.id, "7", unitEach)
	initializeScan(t, five, initStation, nil)
	initializeScan(t, seven, initStation, nil)
	settleScans(t, five, seven)
	body := map[string]any{"batch_ids": []string{five, seven}, "production_step_id": mergeStep, "scanning_station_id": mergeStation}

	results := raceScan("merge", body)
	won := requireScansWon(t, "merge", results, 1, repeatRefusal(t, "merge", body))
	merged := jsonField(won[0], "id")

	settleScans(t, merged)
	requireOneBatchScanned(t, merged)
	requireClosed(t, five, five)
	requireClosed(t, seven, seven)
	waitForOnHand(t, out.id, "12")
	waitForOnHand(t, part.id, "0")
	waitForOnHand(t, material.id, "38")
}

// The scans an operator makes while a supervisor undoes one: the batch is either moved on or put back,
// never both.
func TestBatchScanningRace_MoveAndUndoOfTheSameBatch(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	settleScans(t, planned)
	moveBody := map[string]any{"batch_ids": []string{planned}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation}

	results := race(2, func(i int) (int, []byte, error) {
		if i == 0 {
			return apiClient.Post(batchesPath+"/actions/move", moveBody, newIdempotencyKey())
		}
		return apiClient.Delete(batchesPath + "/" + planned)
	})
	move, undo := results[0], results[1]
	require.Less(t, move.status, 500, "move: %s", move)
	require.Less(t, undo.status, 500, "undo: %s", undo)

	if move.status == http.StatusCreated {
		moved := jsonField(parseJSON(move.body), "id")
		settleScanAtCleanup(t, "move", moved)
		require.Equal(t, http.StatusBadRequest, undo.status, "a batch a later scan used is not undone: %s", undo)
		settleScans(t, moved)
		waitForOnHand(t, c.b.id, "10")
		waitForOnHand(t, c.a.id, "0")
		waitForOnHand(t, c.material.id, "970")
		return
	}
	require.Less(t, undo.status, 300, "when the move is refused the undo goes through: move %s, undo %s", move, undo)
	waitForMessagesSettled(t, "core.cmd.undo_batch_scan", "core.undo_batch_scan", planned)
	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.b.id, "0")
	waitForOnHand(t, c.material.id, "1000")
}

func TestBatchScanningRace_UndoPutsInventoryBackOnce(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")
	settleScans(t, planned, moved)

	results := race(scanRaceWidth, func(int) (int, []byte, error) {
		return apiClient.Delete(batchesPath + "/" + moved)
	})
	var undone int
	for _, r := range results {
		require.Less(t, r.status, 500, "a racing undo is refused, not failed: %s", r)
		if r.status < 300 {
			undone++
		}
	}
	require.Equal(t, 1, undone, "one undo goes through: %v", results)

	waitForMessagesSettled(t, "core.cmd.undo_batch_scan", "core.undo_batch_scan", moved)
	requireOpen(t, planned, planned)
	waitForOnHand(t, c.b.id, "0")
	waitForOnHand(t, c.a.id, "10")
	waitForOnHand(t, c.material.id, "980")
}

// A scanner that sends the same request twice, under the same idempotency key, scans once.
func TestBatchScanningRace_SameKeyScansOnce(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	body := map[string]any{"batch_id": planned, "scanning_station_id": c.initStation}
	key := newIdempotencyKey()

	results := race(scanRaceWidth, func(int) (int, []byte, error) {
		return apiClient.Post(batchesPath+"/actions/initialize", body, key)
	})
	var created int
	for _, r := range results {
		require.Less(t, r.status, 500, "a repeated request is answered, not failed: %s", r)
		if r.status == http.StatusCreated {
			created++
			assert.Equal(t, planned, jsonField(parseJSON(r.body), "id"))
			continue
		}
		require.Equal(t, http.StatusConflict, r.status, "a repeat still in flight is told so: %s", r)
	}
	require.GreaterOrEqual(t, created, 1, "the request goes through: %v", results)
	settleScanAtCleanup(t, "initialize", planned)

	settleScans(t, planned)
	requireOneBatchScanned(t, planned)
	waitForOnHand(t, c.a.id, "10")
	waitForOnHand(t, c.material.id, "980")
}
