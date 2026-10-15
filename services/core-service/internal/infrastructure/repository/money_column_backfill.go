package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// moneyColumnSource is one amount a row still also keeps in a legacy quantity or rate row in a currency unit.
type moneyColumnSource struct {
	// legacyID is the row's column holding the legacy row's id; rate says the legacy row is a rate (a price, whose
	// denominator unit is the per unit) rather than a quantity.
	legacyID string
	rate     bool
	// amount, currency and perUnit are the row's money columns; perUnit is empty for an amount.
	amount, currency, perUnit string
}

// MoneyColumnTable is a table whose rows keep money in their own columns (migration 00088).
type MoneyColumnTable struct {
	table   string
	sources []moneyColumnSource
	// accountJoin and accountWhere scope the table's rows (aliased x) to one account's.
	accountJoin, accountWhere string
}

// width is how many money columns the source has.
func (s moneyColumnSource) width() int {
	if s.perUnit != "" {
		return 3
	}
	return 2
}

func price(prefix string) moneyColumnSource {
	return moneyColumnSource{legacyID: prefix + "_id", rate: true, amount: prefix + "_amount", currency: prefix + "_currency", perUnit: prefix + "_per_unit_id"}
}

// MoneyColumnTables are the operations tables MoneyColumnBackfill fills, one backfill each.
var MoneyColumnTables = []MoneyColumnTable{
	{table: "inventory_receipt", sources: []moneyColumnSource{price("unit_cost")}, accountWhere: "x.owner_account_id = ?"},
	{table: "inventory_allocation", sources: []moneyColumnSource{price("unit_cost"),
		{legacyID: "total_cost_id", amount: "total_cost_amount", currency: "total_cost_currency"}},
		accountJoin: "JOIN inventory_receipt sr ON sr.id = x.inventory_receipt_id", accountWhere: "sr.owner_account_id = ?"},
	{table: "delivery_line", sources: []moneyColumnSource{price("unit_cost")},
		accountJoin: "JOIN delivery sd ON sd.id = x.delivery_id", accountWhere: "sd.account_id = ?"},
	{table: "department", sources: []moneyColumnSource{price("labor_rate")}, accountWhere: "x.account_id = ?"},
	{table: "production_step", sources: []moneyColumnSource{price("labor_rate"), price("overhead_rate")}, accountWhere: "x.account_id = ?"},
	{table: "journal_posting", sources: []moneyColumnSource{{legacyID: "amount_id", amount: "amount", currency: "currency"}}, accountWhere: "x.account_id = ?"},
}

// syncAccountMoneyColumns fills the money columns of an account's rows from their legacy rows: for rows written by SQL
// that bypasses the repositories (the sandbox seed).
func syncAccountMoneyColumns(ctx context.Context, q sqlc.DBTX, accountID string) error {
	for _, t := range MoneyColumnTables {
		for _, s := range t.sources {
			legacyTable, sets := "quantity", "x."+s.amount+" = l.value, x."+s.currency+" = 'usd'"
			if s.rate {
				legacyTable = "rate"
				sets += ", x." + s.perUnit + " = l.denominator_unit_id"
			}
			stmt := "UPDATE " + t.table + " x " + t.accountJoin + " JOIN " + legacyTable + " l ON l.id = x." + s.legacyID +
				" SET " + sets + " WHERE " + t.accountWhere + " AND x." + s.amount + " IS NULL"
			if _, err := q.ExecContext(ctx, stmt, accountID); err != nil {
				return err
			}
		}
	}
	return nil
}

// BackfillName is the table's backfill row in backfill_progress.
func (t MoneyColumnTable) BackfillName() string {
	return "money_columns_" + t.table
}

// columns are the money columns in source order.
func (t MoneyColumnTable) columns() []string {
	var cols []string
	for _, s := range t.sources {
		cols = append(cols, s.amount, s.currency)
		if s.perUnit != "" {
			cols = append(cols, s.perUnit)
		}
	}
	return cols
}

// MoneyColumnBackfill copies each row's legacy quantity or rate rows into its money columns, walking the table by id.
// Every currency unit is a dollar, so each amount is usd.
//
// It brings the columns level with the legacy rows rather than only filling empty ones, so a row older code changed
// in place during the deploy is corrected too. Writers keep the two equal, so on every other row it changes nothing,
// and a rerun batch changes nothing. Rows are read from read (a replica) and written to write (the primary) one row
// per statement: inventory_receipt and inventory_allocation rows are otherwise locked only under the ledger lock
// order, and a statement holding one row lock can never close a lock cycle with a ledger transaction. Each write
// applies only while the columns still hold what was read, so a writer's newer values are never overwritten from a
// lagging replica.
type MoneyColumnBackfill struct {
	read, write sqlc.DBTX
	t           MoneyColumnTable
	selectRows  string
	update      string
}

func NewMoneyColumnBackfill(read, write *sqlc.Queries, t MoneyColumnTable) *MoneyColumnBackfill {
	cols := []string{"x.id"}
	var joins []string
	for i, s := range t.sources {
		alias := fmt.Sprintf("l%d", i)
		cols = append(cols, "x."+s.amount, "x."+s.currency)
		if s.perUnit != "" {
			cols = append(cols, "x."+s.perUnit)
		}
		cols = append(cols, alias+".value")
		legacyTable := "quantity"
		if s.rate {
			legacyTable = "rate"
			cols = append(cols, alias+".denominator_unit_id")
		}
		joins = append(joins, fmt.Sprintf("LEFT JOIN %s %s ON %s.id = x.%s", legacyTable, alias, alias, s.legacyID))
	}
	selectRows := "SELECT " + strings.Join(cols, ", ") + " FROM " + t.table + " x " + strings.Join(joins, " ") +
		" WHERE x.id > ? ORDER BY x.id LIMIT ?"

	moneyCols := t.columns()
	sets := make([]string, len(moneyCols))
	guards := make([]string, len(moneyCols))
	for i, c := range moneyCols {
		sets[i] = c + " = ?"
		guards[i] = c + " <=> ?"
	}
	update := "UPDATE " + t.table + " SET " + strings.Join(sets, ", ") + " WHERE id = ? AND " + strings.Join(guards, " AND ")
	return &MoneyColumnBackfill{read: read.DB(), write: write.DB(), t: t, selectRows: selectRows, update: update}
}

// moneyColumnRow is one row's money columns as read, and what they should hold.
type moneyColumnRow struct {
	id            string
	current, want []gosql.NullString
}

func (b *MoneyColumnBackfill) readRows(ctx context.Context, cursor string, limit int) ([]moneyColumnRow, error) {
	rows, err := b.read.QueryContext(ctx, b.selectRows, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	width := len(b.t.columns())
	var out []moneyColumnRow
	for rows.Next() {
		r := moneyColumnRow{current: make([]gosql.NullString, width)}
		legacyValues := make([]gosql.NullString, len(b.t.sources))
		legacyPerUnits := make([]gosql.NullString, len(b.t.sources))
		dest := []any{&r.id}
		pos := 0
		for i, s := range b.t.sources {
			for range s.width() {
				dest = append(dest, &r.current[pos])
				pos++
			}
			dest = append(dest, &legacyValues[i])
			if s.rate {
				dest = append(dest, &legacyPerUnits[i])
			}
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		r.want = make([]gosql.NullString, 0, width)
		pos = 0
		for i, s := range b.t.sources {
			if legacyValues[i].Valid {
				r.want = append(r.want, legacyValues[i], gosql.NullString{String: "usd", Valid: true})
				if s.perUnit != "" {
					r.want = append(r.want, legacyPerUnits[i])
				}
			} else {
				// No legacy row to copy: whatever the columns hold stays.
				r.want = append(r.want, r.current[pos:pos+s.width()]...)
			}
			pos += s.width()
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r moneyColumnRow) level() bool {
	for i := range r.current {
		if r.current[i] != r.want[i] {
			return false
		}
	}
	return true
}

// Batch is a backfill.Batch over the table's ids.
func (b *MoneyColumnBackfill) Batch(ctx context.Context, m *backfill.Meter, cursor string, limit int) (string, int, bool, error) {
	var rows []moneyColumnRow
	if err := m.Time(func() (err error) {
		rows, err = b.readRows(ctx, cursor, limit)
		return err
	}); err != nil {
		return cursor, 0, false, err
	}
	if len(rows) == 0 {
		return cursor, 0, true, nil
	}
	for _, r := range rows {
		if r.level() {
			continue
		}
		args := make([]any, 0, 2*len(r.want)+1)
		for _, v := range r.want {
			args = append(args, v)
		}
		args = append(args, r.id)
		for _, v := range r.current {
			args = append(args, v)
		}
		if err := m.Time(func() error {
			_, err := b.write.ExecContext(ctx, b.update, args...)
			return err
		}); err != nil {
			return cursor, 0, false, err
		}
	}
	return rows[len(rows)-1].id, len(rows), len(rows) < limit, nil
}
