package main

import (
	"context"
	"io"
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
