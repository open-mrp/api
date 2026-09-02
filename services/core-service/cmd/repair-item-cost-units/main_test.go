package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
)

func discardProgress() *progress {
	return &progress{out: io.Discard, start: time.Now()}
}

func refs(ids ...string) []itemRef {
	out := make([]itemRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, itemRef{ID: id, SKU: id})
	}
	return out
}

func ids(items []itemRef) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

// edgeRows feeds leavesFirst a consumption graph as (input item, output item) pairs.
func edgeRows(pairs ...[2]string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"input_item_id", "output_item_id"})
	for _, p := range pairs {
		rows.AddRow(p[0], p[1])
	}
	return rows
}

// An item's cost is rolled up from its inputs' costs, so repairing a finished good before the
// sub-assemblies under it restates it from numbers that are still wrong. The order this produces is
// the whole reason the repair converges in one run.
func TestLeavesFirst_OrdersInputsBeforeWhatConsumesThem(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"raw", "sub"}, [2]string{"sub", "finished"}))

	got, err := leavesFirst(context.Background(), db, []string{"ac_1"}, refs("finished", "sub", "raw"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirst: %v", err)
	}

	want := []string{"raw", "sub", "finished"}
	for i, id := range ids(got) {
		if id != want[i] {
			t.Fatalf("order = %v, want %v", ids(got), want)
		}
	}
}

// Edges reaching outside the set being ordered constrain nothing — an input that is already correct
// does not need repairing before the item that consumes it.
func TestLeavesFirst_IgnoresEdgesOutsideTheSet(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"not_in_set", "a"}, [2]string{"a", "b"}))

	got, err := leavesFirst(context.Background(), db, []string{"ac_1"}, refs("b", "a"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirst: %v", err)
	}

	if want := []string{"a", "b"}; ids(got)[0] != want[0] || ids(got)[1] != want[1] {
		t.Fatalf("order = %v, want %v", ids(got), want)
	}
}

// A routing that consumes its own output has no order that satisfies it. Dropping those items would
// strand every item downstream of them, so they are repaired in scan order and everything else keeps
// its dependency order.
func TestLeavesFirst_AppendsCyclesRatherThanDroppingThem(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"a", "b"}, [2]string{"b", "a"}))

	got, err := leavesFirst(context.Background(), db, []string{"ac_1"}, refs("a", "b", "loose"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirst: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("every item must survive a cycle, got %v", ids(got))
	}
	if ids(got)[0] != "loose" {
		t.Errorf("the unconstrained item should place first, got %v", ids(got))
	}
}

// The plan's before-and-after is the number a reviewer checks against a physical count, so the
// arithmetic behind it is pinned here: fifty cartons of a $100.564364 cost mislabelled per-each
// value at eight times what they are worth, and the relabel takes exactly that factor back off.
func TestLayerValuation_RestatesByTheUnitGroupRatio(t *testing.T) {
	t.Parallel()

	l := mislabelledLayer{
		Quantity:                decimal.NewFromInt(50),
		QuantityInStockingUnits: decimal.NewFromInt(1),
		Cost: mislabelledCost{
			Value:           decimal.RequireFromString("100.564364"),
			ValuationFactor: decimal.NewFromInt(8),
		},
	}

	after := l.Quantity.Mul(l.QuantityInStockingUnits).Mul(l.Cost.Value)
	before := after.Mul(l.Cost.ValuationFactor)

	if got, want := after.StringFixed(2), "5028.22"; got != want {
		t.Errorf("restated valuation = %s, want %s", got, want)
	}
	if got, want := before.StringFixed(2), "40225.75"; got != want {
		t.Errorf("current valuation = %s, want %s", got, want)
	}
	if got, want := before.Sub(after).StringFixed(2), "35197.53"; got != want {
		t.Errorf("overstatement removed = %s, want %s", got, want)
	}
}

// costs builds the item side of a relabel set, every rate heading for the same stocking unit.
func costs(stockingID string, rateIDs ...string) []mislabelledCost {
	out := make([]mislabelledCost, 0, len(rateIDs))
	for _, id := range rateIDs {
		out = append(out, mislabelledCost{RateID: id, StockingID: stockingID})
	}
	return out
}

// vtgate aborts a transaction at 20s, which the per-row form spent on round trips alone. Rates
// sharing a destination unit have to move in one statement for the repair to fit.
func TestApplyRelabels_MovesRatesSharingAUnitInOneStatement(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE rate SET denominator_unit_id").
		WithArgs("un_ct8ea", "rt_a", "rt_b", "rt_c").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	items := costs("un_ct8ea", "rt_a", "rt_b")
	layers := []mislabelledLayer{{Cost: mislabelledCost{RateID: "rt_c", StockingID: "un_ct8ea"}}}

	if err := applyRelabels(context.Background(), db, items, layers, discardProgress()); err != nil {
		t.Fatalf("applyRelabels: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Rates bound for different units cannot share a statement, but they still share the transaction.
func TestApplyRelabels_SplitsAStatementPerDestinationUnit(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE rate SET denominator_unit_id").
		WithArgs("un_ct8ea", "rt_a").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE rate SET denominator_unit_id").
		WithArgs("un_kg", "rt_b").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	items := append(costs("un_ct8ea", "rt_a"), costs("un_kg", "rt_b")...)

	if err := applyRelabels(context.Background(), db, items, nil, discardProgress()); err != nil {
		t.Fatalf("applyRelabels: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A failed batch must leave the batches before it committed, and say how far it got: the run is
// resumable only because the scan re-finds whatever is still mislabelled.
func TestApplyRelabels_KeepsCommittedBatchesAndReportsProgress(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rateIDs := make([]string, 0, relabelBatchSize+1)
	for i := range relabelBatchSize + 1 {
		rateIDs = append(rateIDs, fmt.Sprintf("rt_%d", i))
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE rate SET denominator_unit_id").WillReturnResult(sqlmock.NewResult(0, relabelBatchSize))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE rate SET denominator_unit_id").WillReturnError(errors.New("exceeded timeout: 20s"))
	mock.ExpectRollback()

	err = applyRelabels(context.Background(), db, costs("un_ct8ea", rateIDs...), nil, discardProgress())
	if err == nil {
		t.Fatal("applyRelabels succeeded, want the second batch to fail")
	}
	if want := fmt.Sprintf("%d of %d rate(s) relabelled and committed", relabelBatchSize, len(rateIDs)); !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to report %q", err, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A rollup spends seconds on an item, so a count-gated line every progressEvery is minutes of
// silence — exactly the hang the progress lines exist to rule out.
func TestProgressStep_PrintsOnTheIntervalBetweenCountMilestones(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	p := &progress{out: &out, start: time.Now().Add(-time.Minute)}

	p.lastStep = time.Now().Add(-progressInterval * 2)
	p.step(3, 5284, "items recomputed")
	if !strings.Contains(out.String(), "3/5284 items recomputed") {
		t.Fatalf("no line after the interval elapsed; got %q", out.String())
	}

	out.Reset()
	p.step(4, 5284, "items recomputed")
	if out.String() != "" {
		t.Fatalf("line printed inside the interval and off a milestone; got %q", out.String())
	}

	p.step(5284, 5284, "items recomputed")
	if !strings.Contains(out.String(), "5284/5284 items recomputed") {
		t.Fatalf("final item did not print; got %q", out.String())
	}
}

// levelIDs renders the grouping as a slice per level, which is what the concurrency contract rests
// on: an item must never share a level with something it consumes.
func levelIDs(levels [][]itemRef) [][]string {
	out := make([][]string, 0, len(levels))
	for _, level := range levels {
		out = append(out, ids(level))
	}
	return out
}

// A level's items recompute concurrently, so an item sharing a level with one of its inputs would be
// rolled up from a cost that has not been restated yet.
func TestLeavesFirstLevels_SeparatesAnItemFromWhatItConsumes(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"raw", "sub"}, [2]string{"sub", "finished"}))

	levels, err := leavesFirstLevels(context.Background(), db, []string{"ac_1"}, refs("finished", "sub", "raw"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirstLevels: %v", err)
	}

	want := [][]string{{"raw"}, {"sub"}, {"finished"}}
	if got := levelIDs(levels); !reflect.DeepEqual(got, want) {
		t.Fatalf("levels = %v, want %v", got, want)
	}
}

// Items with no edge between them are the whole point of the grouping — they are what runs at once.
func TestLeavesFirstLevels_GroupsIndependentItemsTogether(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"raw_a", "finished"}, [2]string{"raw_b", "finished"}))

	levels, err := leavesFirstLevels(context.Background(), db, []string{"ac_1"}, refs("finished", "raw_a", "raw_b"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirstLevels: %v", err)
	}

	want := [][]string{{"raw_a", "raw_b"}, {"finished"}}
	if got := levelIDs(levels); !reflect.DeepEqual(got, want) {
		t.Fatalf("levels = %v, want %v", got, want)
	}
}

// A cycle satisfies no order, so its items must not be widened into one concurrent level.
func TestLeavesFirstLevels_KeepsCycleMembersOnTheirOwn(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"a", "b"}, [2]string{"b", "a"}))

	levels, err := leavesFirstLevels(context.Background(), db, []string{"ac_1"}, refs("a", "b"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirstLevels: %v", err)
	}

	for _, level := range levels {
		if len(level) != 1 {
			t.Fatalf("cycle members shared a level: %v", levelIDs(levels))
		}
	}
}

// leavesFirst is the flattening of the levels, so the order it has always returned must survive them.
func TestLeavesFirst_StillFlattensTheLevelsInOrder(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM consumption").WithArgs("ac_1").
		WillReturnRows(edgeRows([2]string{"raw_a", "finished"}, [2]string{"raw_b", "finished"}))

	got, err := leavesFirst(context.Background(), db, []string{"ac_1"}, refs("finished", "raw_a", "raw_b"), discardProgress())
	if err != nil {
		t.Fatalf("leavesFirst: %v", err)
	}

	want := []string{"raw_a", "raw_b", "finished"}
	if !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("order = %v, want %v", ids(got), want)
	}
}
