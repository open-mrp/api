package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// Order, invoice and pick lines carry copies of the values and units of the quantity and rate rows they
// own (migration 00081), so reports read them off the line instead of looking each row up. Every write
// to an owned row, or to the line's link to it, re-copies in the same transaction through the helpers
// below; LineCopyBackfill fills the lines written before the copies existed. Once it has finished, a
// NULL copy means the row it copies is missing, which readers treat as the LEFT JOIN to it did.

// copyTable is one table's copies of the quantity and rate rows its rows own.
type copyTable struct {
	table, alias string
	// joins reach the owned quantity and rate rows.
	joins string
	// copies are the row's copy columns, sources the expressions they copy, in the same order.
	copies, sources []string
	// uncopied are the columns that are all NULL only on a row with no copy at all: one the backfill
	// has yet to fill.
	uncopied []string
}

// uncopiedWhere is the predicate for a row with no copy, qualified by alias when one is given.
func (t copyTable) uncopiedWhere(alias string) string {
	preds := make([]string, len(t.uncopied))
	for i, c := range t.uncopied {
		if alias != "" {
			c = alias + "." + c
		}
		preds[i] = c + " IS NULL"
	}
	return strings.Join(preds, " AND ")
}

var (
	orderLineCopies = copyTable{
		table: "sales_order_line", alias: "sol",
		joins: `LEFT JOIN quantity cq ON cq.id = sol.quantity_id
LEFT JOIN rate cp ON cp.id = sol.unit_price_id
LEFT JOIN rate cc ON cc.id = sol.unit_cost_id`,
		copies: []string{"quantity_value", "quantity_unit_id", "unit_price_value", "unit_price_numerator_unit_id",
			"unit_price_denominator_unit_id", "unit_cost_value", "unit_cost_numerator_unit_id", "unit_cost_denominator_unit_id",
			"unit_price_currency", "unit_cost_currency"},
		sources: []string{"cq.value", "cq.unit_id", "cp.value", "cp.numerator_unit_id",
			"cp.denominator_unit_id", "cc.value", "cc.numerator_unit_id", "cc.denominator_unit_id",
			currencyOf("cp"), currencyOf("cc")},
		uncopied: []string{"quantity_unit_id", "unit_price_numerator_unit_id", "unit_cost_numerator_unit_id"},
	}
	invoiceLineCopies = copyTable{
		table: "invoice_line", alias: "il",
		joins:  "LEFT JOIN quantity cq ON cq.id = il.quantity_id",
		copies: []string{"quantity_value", "quantity_unit_id"}, sources: []string{"cq.value", "cq.unit_id"},
		uncopied: []string{"quantity_unit_id"},
	}
	pickLineCopies = copyTable{
		table: "pick_line", alias: "pl",
		joins:  "LEFT JOIN quantity cq ON cq.id = pl.quantity_id",
		copies: []string{"quantity_value", "quantity_unit_id"}, sources: []string{"cq.value", "cq.unit_id"},
		uncopied: []string{"quantity_unit_id"},
	}
)

// copyRow is one row's sources (what its copies should hold) and its current copies.
type copyRow struct {
	id               string
	sources, current []sql.NullString
}

func (r copyRow) stale() bool {
	for i := range r.sources {
		if r.sources[i] != r.current[i] {
			return true
		}
	}
	return false
}

// readCopies reads t's rows matching where (written against t's alias) with their sources. It is a
// plain consistent read: in a transaction it sees the transaction's own writes without locking the
// quantity and rate rows.
func readCopies(ctx context.Context, q sqlc.DBTX, t copyTable, where string, args ...any) ([]copyRow, error) {
	cols := make([]string, 0, 1+2*len(t.copies))
	cols = append(cols, t.alias+".id")
	for i, c := range t.copies {
		cols = append(cols, t.sources[i], t.alias+"."+c)
	}
	rows, err := q.QueryContext(ctx, "SELECT "+strings.Join(cols, ", ")+" FROM "+t.table+" "+t.alias+"\n"+t.joins+"\nWHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []copyRow
	for rows.Next() {
		r := copyRow{sources: make([]sql.NullString, len(t.copies)), current: make([]sql.NullString, len(t.copies))}
		dest := make([]any, 0, len(cols))
		dest = append(dest, &r.id)
		for i := range t.copies {
			dest = append(dest, &r.sources[i], &r.current[i])
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// syncCopies re-copies t's rows matching where, rewriting by primary key each row whose copies differ.
func syncCopies(ctx context.Context, q sqlc.DBTX, t copyTable, where string, args ...any) error {
	rows, err := readCopies(ctx, q, t, where, args...)
	if err != nil {
		return err
	}
	set := make([]string, len(t.copies))
	for i, c := range t.copies {
		set[i] = c + " = ?"
	}
	stmt := "UPDATE " + t.table + " SET " + strings.Join(set, ", ") + " WHERE id = ?"
	for _, r := range rows {
		if !r.stale() {
			continue
		}
		args := make([]any, 0, len(t.copies)+1)
		for _, v := range r.sources {
			args = append(args, v)
		}
		if _, err := q.ExecContext(ctx, stmt, append(args, r.id)...); err != nil {
			return err
		}
	}
	return nil
}

// fillCopies writes rows' sources into rows that still have no copy, in one statement. A row a
// writer copied in the meantime is left alone: its copies are newer than rows' sources can be.
func fillCopies(ctx context.Context, q sqlc.DBTX, t copyTable, rows []copyRow) error {
	if len(rows) == 0 {
		return nil
	}
	set := make([]string, len(t.copies))
	var args []any
	for i, c := range t.copies {
		cases := make([]string, len(rows))
		for j, r := range rows {
			cases[j] = "WHEN ? THEN ?"
			args = append(args, r.id, r.sources[i])
		}
		set[i] = c + " = CASE id " + strings.Join(cases, " ") + " END"
	}
	for _, r := range rows {
		args = append(args, r.id)
	}
	_, err := q.ExecContext(ctx, "UPDATE "+t.table+" SET "+strings.Join(set, ", ")+
		" WHERE id IN ("+placeholders(len(rows))+") AND "+t.uncopiedWhere(""), args...)
	return err
}

// syncOrderLineCopies re-copies the given order lines' quantity, price and cost.
func syncOrderLineCopies(ctx context.Context, q sqlc.DBTX, lineIDs ...string) error {
	if len(lineIDs) == 0 {
		return nil
	}
	return syncCopies(ctx, q, orderLineCopies, "sol.id IN ("+placeholders(len(lineIDs))+")", stringsToAny(lineIDs)...)
}

// syncPickLineCopies re-copies the given pick lines' quantities.
func syncPickLineCopies(ctx context.Context, q sqlc.DBTX, pickLineIDs ...string) error {
	if len(pickLineIDs) == 0 {
		return nil
	}
	return syncCopies(ctx, q, pickLineCopies, "pl.id IN ("+placeholders(len(pickLineIDs))+")", stringsToAny(pickLineIDs)...)
}

// syncPickCopies re-copies every line of a pick.
func syncPickCopies(ctx context.Context, q sqlc.DBTX, pickID string) error {
	return syncCopies(ctx, q, pickLineCopies, "pl.pick_id = ?", pickID)
}

// syncOrderLineCopiesByQuantity re-copies the order line owning a quantity row, if one does.
func syncOrderLineCopiesByQuantity(ctx context.Context, q sqlc.DBTX, quantityID string) error {
	return syncCopies(ctx, q, orderLineCopies, "sol.quantity_id = ?", quantityID)
}

// syncOrderLineCopiesByRate re-copies the order line owning a rate row as its price or cost, if one does.
// Each is its own unique-key lookup.
func syncOrderLineCopiesByRate(ctx context.Context, q sqlc.DBTX, rateID string) error {
	if err := syncCopies(ctx, q, orderLineCopies, "sol.unit_price_id = ?", rateID); err != nil {
		return err
	}
	return syncCopies(ctx, q, orderLineCopies, "sol.unit_cost_id = ?", rateID)
}

// syncFulfillmentCopiesByQuantity re-copies the invoice or pick line owning a quantity row, if one does.
func syncFulfillmentCopiesByQuantity(ctx context.Context, q sqlc.DBTX, quantityID string) error {
	if err := syncCopies(ctx, q, invoiceLineCopies, "il.quantity_id = ?", quantityID); err != nil {
		return err
	}
	return syncCopies(ctx, q, pickLineCopies, "pl.quantity_id = ?", quantityID)
}

// syncAccountOrderCopies re-copies every order, invoice and pick line on the account's orders: for rows
// written by SQL that bypasses the repositories (the sandbox seed).
func syncAccountOrderCopies(ctx context.Context, q sqlc.DBTX, accountID string) error {
	orders := "SELECT so.id FROM sales_order so WHERE so.owner_account_id = ?"
	if err := syncCopies(ctx, q, orderLineCopies, "sol.sales_order_id IN ("+orders+")", accountID); err != nil {
		return err
	}
	lines := "SELECT sol2.id FROM sales_order_line sol2 JOIN sales_order so ON so.id = sol2.sales_order_id WHERE so.owner_account_id = ?"
	if err := syncCopies(ctx, q, invoiceLineCopies, "il.sales_order_line_id IN ("+lines+")", accountID); err != nil {
		return err
	}
	return syncCopies(ctx, q, pickLineCopies, "pl.sales_order_line_id IN ("+lines+")", accountID)
}

// LineCopyBackfillName is the line-copy backfill's row in backfill_progress.
const LineCopyBackfillName = "line_quantity_copies"

// lineCopiesDone latches once the backfill is seen finished: from then on every line has its copies.
var lineCopiesDone atomic.Bool

// lineCopiesComplete reports whether the line-copy backfill has finished, so reports may read the copies.
// Until then they read the quantity and rate rows themselves.
func lineCopiesComplete(ctx context.Context, q sqlc.DBTX) (bool, error) {
	if lineCopiesDone.Load() {
		return true, nil
	}
	var done bool
	err := q.QueryRowContext(ctx, "SELECT completed_at IS NOT NULL FROM backfill_progress WHERE name = ?", LineCopyBackfillName).Scan(&done)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if done {
		lineCopiesDone.Store(true)
	}
	return done, nil
}
