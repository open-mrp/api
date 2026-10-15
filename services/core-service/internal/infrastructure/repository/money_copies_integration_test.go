//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
)

// mcFixture is one row of each table whose money is copied, written the way SQL that bypasses the
// repositories writes it: quantity and rate rows, and no copies.
type mcFixture struct {
	pool                                              *sql.DB
	item, relation, term, shippingCase, target, plTgt string
	rates, quantities                                 []string
}

func newMCFixture(t *testing.T, pool *sql.DB) *mcFixture {
	t.Helper()
	tag := fmt.Sprintf("mctest%d", time.Now().UnixNano())
	f := &mcFixture{pool: pool, item: "it_" + tag, relation: "ar_" + tag, term: "st_" + tag, shippingCase: "sc_" + tag,
		target: "tg_" + tag, plTgt: "plt_" + tag}
	rate := func(id, value, num, den string) string {
		lcExec(t, pool, "INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES (?, ?, ?, ?)", id, value, num, den)
		f.rates = append(f.rates, id)
		return id
	}
	qty := func(id, value, unit string) string {
		lcExec(t, pool, "INSERT INTO quantity (id, value, unit_id) VALUES (?, ?, ?)", id, value, unit)
		f.quantities = append(f.quantities, id)
		return id
	}
	t.Cleanup(f.cleanup)

	uv, uc, br := rate("rt_uv_"+tag, "18.45", "dollar", "pair"), rate("rt_uc_"+tag, "6.125", "dollar", "each"), rate("rt_br_"+tag, "0", "each", "day")
	lcExec(t, pool, `INSERT INTO item (id, sku, unit_value_id, burn_rate_id, account_id, item_type_code, unit_cost_id, item_category_id)
		VALUES (?, ?, ?, ?, ?, 'material', ?, 'ic_none')`, f.item, f.item, uv, br, lcSeller, uc)
	lcExec(t, pool, `INSERT INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, priority_code, credit_limit_id)
		VALUES (?, ?, ?, 'customer', ?, 'normal', ?)`, f.relation, lcSeller, "ac_"+tag, tag, qty("qu_cl_"+tag, "5000", "dollar"))
	lcExec(t, pool, "INSERT INTO shipping_term (id, name, account_id, flat_rate_id) VALUES (?, ?, ?, ?)",
		f.term, tag, lcSeller, qty("qu_fr_"+tag, "12.5", "dollar"))
	lcExec(t, pool, `INSERT INTO shipping_case (id, number, freight_amount_id, freight_weight_id, shipment_id, carrier_id, account_id)
		VALUES (?, ?, ?, ?, 'sh_none', 'ca_none', ?)`, f.shippingCase, tag, qty("qu_fa_"+tag, "31.07", "dollar"), qty("qu_fw_"+tag, "2", "pound"), lcSeller)
	lcExec(t, pool, "INSERT INTO target (id, start_date, end_date, sales_rep_id, account_id, amount_id) VALUES (?, NOW(3), NOW(3), 'acus_none', ?, ?)",
		f.target, lcSeller, qty("qu_tg_"+tag, "100000", "dollar"))
	lcExec(t, pool, "INSERT INTO product_line_target (id, start_date, end_date, product_line_id, sales_rep_id, account_id, amount_id) VALUES (?, NOW(3), NOW(3), 'pl_none', 'acus_none', ?, ?)",
		f.plTgt, lcSeller, qty("qu_plt_"+tag, "2500", "dollar"))
	return f
}

func (f *mcFixture) cleanup() {
	for _, stmt := range []struct{ table, id string }{
		{"item", f.item}, {"account_relation", f.relation}, {"shipping_term", f.term},
		{"shipping_case", f.shippingCase}, {"target", f.target}, {"product_line_target", f.plTgt},
	} {
		_, _ = f.pool.Exec("DELETE FROM "+stmt.table+" WHERE id = ?", stmt.id)
	}
	for _, id := range f.rates {
		_, _ = f.pool.Exec("DELETE FROM rate WHERE id = ?", id)
	}
	for _, id := range f.quantities {
		_, _ = f.pool.Exec("DELETE FROM quantity WHERE id = ?", id)
	}
}

// copies reads each fixture row's money columns as strings, NULL as "<nil>".
func (f *mcFixture) copies(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, c := range []struct {
		name, query, id string
	}{
		{"item", "SELECT sales_unit_price_amount, sales_unit_price_currency, sales_unit_price_per_unit_id, unit_cost_amount, unit_cost_currency, unit_cost_per_unit_id FROM item WHERE id = ?", f.item},
		{"account_relation", "SELECT credit_limit_amount, credit_limit_currency FROM account_relation WHERE id = ?", f.relation},
		{"shipping_term", "SELECT flat_rate_amount, flat_rate_currency, minimum_order_value_amount, minimum_order_value_currency FROM shipping_term WHERE id = ?", f.term},
		{"shipping_case", "SELECT freight_amount, freight_currency FROM shipping_case WHERE id = ?", f.shippingCase},
		{"target", "SELECT amount, currency FROM target WHERE id = ?", f.target},
		{"product_line_target", "SELECT amount, currency FROM product_line_target WHERE id = ?", f.plTgt},
	} {
		rows, err := f.pool.Query(c.query, c.id)
		require.NoError(t, err)
		cols, err := rows.Columns()
		require.NoError(t, err)
		require.True(t, rows.Next(), c.name)
		vals := make([]sql.NullString, len(cols))
		dest := make([]any, len(cols))
		for i := range vals {
			dest[i] = &vals[i]
		}
		require.NoError(t, rows.Scan(dest...))
		require.NoError(t, rows.Close())
		strs := make([]string, len(vals))
		for i, v := range vals {
			strs[i] = "<nil>"
			if v.Valid {
				strs[i] = v.String
			}
		}
		out[c.name] = strs
	}
	return out
}

// dec is s as a decimal(65,30) column reads back.
func dec(s string) string {
	whole, frac, _ := strings.Cut(s, ".")
	return whole + "." + frac + strings.Repeat("0", 30-len(frac))
}

// wantMCCopies is what every fixture row's money columns hold once copied.
func wantMCCopies() map[string][]string {
	return map[string][]string{
		"item":                {dec("18.45"), "usd", "pair", dec("6.125"), "usd", "each"},
		"account_relation":    {dec("5000"), "usd"},
		"shipping_term":       {dec("12.5"), "usd", "<nil>", "<nil>"},
		"shipping_case":       {dec("31.07"), "usd"},
		"target":              {dec("100000"), "usd"},
		"product_line_target": {dec("2500"), "usd"},
	}
}

// runMoneyBackfills runs every money column backfill over its whole table.
func runMoneyBackfills(t *testing.T, q *sqlc.Queries) {
	t.Helper()
	for _, b := range NewMoneyCopyBackfills(q, q) {
		cursor := ""
		for {
			next, _, done, err := b.Batch(context.Background(), &backfill.Meter{}, cursor, 500)
			require.NoError(t, err, b.Name)
			if done {
				break
			}
			cursor = next
		}
	}
}

// TestMoneyCopies_BackfillFillsUncopiedRows requires the backfills to copy every row written without its
// money columns, and to leave a row a writer has already copied alone.
func TestMoneyCopies_BackfillFillsUncopiedRows(t *testing.T) {
	pool := testDB(t)
	q := sqlc.New(pool)
	f := newMCFixture(t, pool)
	for name, cols := range f.copies(t) {
		for _, c := range cols {
			require.Equal(t, "<nil>", c, "%s starts uncopied", name)
		}
	}

	// A writer copied this target since the backfill read it; its copy must survive.
	lcExec(t, pool, "UPDATE target SET amount = 7, currency = 'usd' WHERE id = ?", f.target)

	runMoneyBackfills(t, q)

	want := wantMCCopies()
	want["target"] = []string{dec("7"), "usd"}
	require.Equal(t, want, f.copies(t))
}

// TestMoneyCopies_SyncFollowsTheRows changes each fixture's quantity and rate rows and requires the sync
// each repository write calls to bring the copies along, including a limit that is removed.
func TestMoneyCopies_SyncFollowsTheRows(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	f := newMCFixture(t, pool)
	require.NoError(t, syncAccountMoneyCopies(ctx, pool, lcSeller))
	require.Equal(t, wantMCCopies(), f.copies(t))

	lcExec(t, pool, "UPDATE rate SET value = 20, denominator_unit_id = 'each' WHERE id = ?", "rt_uv_"+f.item[3:])
	require.NoError(t, syncItemMoneyByRate(ctx, pool, "rt_uv_"+f.item[3:]))
	lcExec(t, pool, "UPDATE account_relation SET credit_limit_id = NULL WHERE id = ?", f.relation)
	require.NoError(t, syncCreditLimit(ctx, pool, f.relation))
	lcExec(t, pool, "UPDATE quantity SET value = 0 WHERE id = ?", "qu_fa_"+f.shippingCase[3:])
	require.NoError(t, syncShippingCaseFreightByQuantity(ctx, pool, "qu_fa_"+f.shippingCase[3:]))
	lcExec(t, pool, "UPDATE quantity SET value = 90000 WHERE id = ?", "qu_tg_"+f.target[3:])
	require.NoError(t, syncTargetAmountByQuantity(ctx, pool, "qu_tg_"+f.target[3:]))

	want := wantMCCopies()
	want["item"] = []string{dec("20"), "usd", "each", dec("6.125"), "usd", "each"}
	want["account_relation"] = []string{"<nil>", "<nil>"}
	want["shipping_case"] = []string{dec("0"), "usd"}
	want["target"] = []string{dec("90000"), "usd"}
	require.Equal(t, want, f.copies(t))

	drift, err := MoneyColumnDrift(ctx, pool)
	require.NoError(t, err)
	for table, n := range drift {
		require.Zero(t, n, "%s has copies that disagree with their rows", table)
	}
}
