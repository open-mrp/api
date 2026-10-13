//go:build e2e

package api_test

import "testing"

// Splitting is a hand count. A split that closes its source takes what was split off as the source's
// true quantity, carries it up through the batches the source was moved from whole, and trues up each
// one's step inventory, so the parts in between neither go negative nor keep phantom stock.

func requireBatchQuantity(t *testing.T, flowOf, batchID, want string) {
	t.Helper()
	assertDecimalEqual(t, want, jsonField(jsonObject(flowBatch(t, flowOf, batchID), "quantity"), "value"),
		"quantity of "+batchID)
}

// A batch recorded as 10 that splits into 4, 4 and 3 made 11.
func TestBatchSplitRecount_OverCountReceivesAndConsumesTheDifference(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")
	waitForOnHand(t, c.material.id, "970")

	mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "4", "", "", false))
	mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "4", "", "", false))
	requireRemaining(t, c.splitStep, "2", unitEach, moved)
	last := jsonField(mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "3", "", "", false)), "id")

	requireClosed(t, planned, moved)
	requireBatchQuantity(t, planned, moved, "11")
	requireBatchQuantity(t, planned, planned, "11")
	waitForOnHand(t, c.c.id, "11")
	// One more B made and split off, from one more A, made and consumed.
	waitForOnHand(t, c.b.id, "0")
	waitForOnHand(t, c.a.id, "0")
	// 2 a piece for A and 1 a piece for B, for the eleventh piece too.
	waitForOnHand(t, c.material.id, "967")

	// Undoing the split that recounted puts the recorded quantities and the inventory back.
	deleteBatch(t, last)
	requireOpen(t, planned, moved)
	requireBatchQuantity(t, planned, moved, "10")
	requireBatchQuantity(t, planned, planned, "10")
	requireRemaining(t, c.splitStep, "2", unitEach, moved)
	waitForOnHand(t, c.c.id, "8")
	waitForOnHand(t, c.b.id, "2")
	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.material.id, "970")
}

// A batch recorded as 10 closed after 4 and 3 made 7.
func TestBatchSplitRecount_UnderCountTakesBackAndReleasesTheDifference(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")
	waitForOnHand(t, c.material.id, "970")

	mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "4", "", "", false))
	last := jsonField(mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "2", "", "1", true)), "id")

	requireClosed(t, planned, moved)
	// Waste was in the batch too, so it counts.
	requireBatchQuantity(t, planned, moved, "7")
	requireBatchQuantity(t, planned, planned, "7")
	waitForOnHand(t, c.c.id, "6")
	waitForOnHand(t, c.b.id, "0")
	waitForOnHand(t, c.a.id, "0")
	// The three pieces never made hand back 2 and 1 of material each.
	waitForOnHand(t, c.material.id, "979")

	deleteBatch(t, last)
	requireOpen(t, planned, moved)
	requireBatchQuantity(t, planned, moved, "10")
	requireBatchQuantity(t, planned, planned, "10")
	requireRemaining(t, c.splitStep, "6", unitEach, moved)
	waitForOnHand(t, c.c.id, "4")
	waitForOnHand(t, c.b.id, "6")
	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.material.id, "970")
}

// A split that ends exactly at the recorded quantity changes nothing.
func TestBatchSplitRecount_ExactCountLeavesTheSourceAlone(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")

	mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "6", "", "", false))
	mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "4", "", "", false))

	requireClosed(t, planned, moved)
	requireBatchQuantity(t, planned, moved, "10")
	requireBatchQuantity(t, planned, planned, "10")
	waitForOnHand(t, c.b.id, "0")
	waitForOnHand(t, c.material.id, "970")
}

// Closing an assembly's parts early recounts every part by what it put into the split.
func TestBatchSplitRecount_MultiPartCloseRecountsEveryPart(t *testing.T) {
	t.Parallel()
	f := newMultiPartFixture(t, "split_batch")
	p1 := f.initialized(t, f.p1, "6")
	p2 := f.initialized(t, f.p2, "12")
	waitForOnHand(t, f.p2.id, "12")

	last := jsonField(mustScan(t, "split", splitBody(f.station, f.step, []string{p1, p2}, "2", "", "", true)), "id")

	requireClosed(t, p1, p1)
	requireClosed(t, p2, p2)
	requireBatchQuantity(t, p1, p1, "2")
	requireBatchQuantity(t, p2, p2, "4")
	waitForOnHand(t, f.out.id, "2")
	waitForOnHand(t, f.p1.id, "0")
	waitForOnHand(t, f.p2.id, "0")

	deleteBatch(t, last)
	requireBatchQuantity(t, p1, p1, "6")
	requireBatchQuantity(t, p2, p2, "12")
	waitForOnHand(t, f.p1.id, "6")
	waitForOnHand(t, f.p2.id, "12")
}
