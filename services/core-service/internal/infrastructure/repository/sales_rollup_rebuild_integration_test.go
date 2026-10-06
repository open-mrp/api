//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/stretchr/testify/require"
)

// The rollup rebuild sums facts in Go; these tests hold it to what SQL's SUM and COUNT(DISTINCT) give for the same facts.

const rebuildTestAccountID = "ac_rollup_rebuild"

// expectedRollupSQL is every rollup row the account's facts imply, summed by MySQL.
func expectedRollupSQL() string {
	var parts []string
	for _, d := range []struct {
		name, column    string
		perLine, hourly bool
	}{
		{rollupDimTotal, "", true, true},
		{rollupDimBuyer, "f.buyer_account_id", true, false},
		{rollupDimItem, "f.item_id", true, false},
		{rollupDimProductLine, "f.product_line_id", false, false},
		{rollupDimSalesRep, "f.sales_rep_id", true, false},
		{rollupDimDiscount, "f.order_discount_id", true, false},
	} {
		for _, byLine := range []bool{false, true} {
			if byLine && !d.perLine {
				continue
			}
			for _, g := range []struct{ grain, bucket string }{
				{rollupGrainDay, "CAST(DATE_FORMAT(f.invoiced_at, '%Y-%m-%d 00:00:00') AS DATETIME)"},
				{rollupGrainMonth, "CAST(DATE_FORMAT(f.invoiced_at, '%Y-%m-01 00:00:00') AS DATETIME)"},
				{rollupGrainHour, "CAST(DATE_FORMAT(f.invoiced_at, '%Y-%m-%d %H:00:00') AS DATETIME)"},
			} {
				if g.grain == rollupGrainHour && !d.hourly {
					continue
				}
				dimID, lineKey := "''", "''"
				group := []string{"f.sales_order_type_code", "COALESCE(f.sales_rep_id, '')", g.bucket}
				where := "f.account_id = ?"
				if d.column != "" {
					dimID = d.column
					group = append(group, d.column)
					where += " AND " + d.column + " IS NOT NULL"
				}
				if byLine {
					lineKey = "f.product_line_id"
					group = append(group, lineKey)
				}
				// The hash is taken outside the aggregate: ONLY_FULL_GROUP_BY rejects an expression over grouped expressions.
				parts = append(parts, `SELECT g.t, '`+d.name+`', g.line_key, '`+g.grain+`', g.bucket, UNHEX(MD5(CONCAT(g.dim_id, CHAR(0), g.rep_key))), g.dim_id, g.rep_key,
CAST(g.qty AS DECIMAL(28,10)), CAST(g.inv AS DECIMAL(28,10)), CAST(g.cost AS DECIMAL(28,10)), g.ic, g.lc
FROM (SELECT f.sales_order_type_code AS t, `+lineKey+` AS line_key, `+g.bucket+` AS bucket, `+dimID+` AS dim_id, COALESCE(f.sales_rep_id, '') AS rep_key,
  SUM(f.quantity_base) AS qty, SUM(f.total_invoiced) AS inv, SUM(f.total_cost) AS cost, COUNT(DISTINCT f.invoice_id) AS ic, COUNT(*) AS lc
  FROM sales_line_fact f WHERE `+where+` GROUP BY `+strings.Join(group, ", ")+`) g`)
			}
		}
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

func assertRollupsMatchFacts(t *testing.T, ctx context.Context, pool *sql.DB) {
	t.Helper()
	q := expectedRollupSQL()
	want, err := selectRollupRows(ctx, pool, q, anyRepeat(rebuildTestAccountID, strings.Count(q, "f.account_id = ?"))...)
	require.NoError(t, err)
	got, err := selectRollupRows(ctx, pool, `SELECT `+rollupRowColumns+` FROM sales_fact_rollup WHERE account_id = ?`, rebuildTestAccountID)
	require.NoError(t, err)
	require.NotEmpty(t, want)
	for k, w := range want {
		g, ok := got[k]
		require.True(t, ok, "missing rollup row %s/%s/%s %s", k.dimension, k.lineKey, k.grain, k.bucket)
		require.Equal(t, w, g, "rollup row %s/%s/%s %s", k.dimension, k.lineKey, k.grain, k.bucket)
	}
	for k := range got {
		_, ok := want[k]
		require.True(t, ok, "extra rollup row %s/%s/%s %s", k.dimension, k.lineKey, k.grain, k.bucket)
	}
}

func anyRepeat(v any, n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestRebuildRollupDayMatchesSQLSums(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	repo := NewSalesFactRepo(sqlc.New(pool))

	clear := func() {
		_, _ = pool.Exec(`DELETE FROM sales_line_fact WHERE account_id = ?`, rebuildTestAccountID)
		_, _ = pool.Exec(`DELETE FROM sales_fact_rollup WHERE account_id = ?`, rebuildTestAccountID)
	}
	clear()
	t.Cleanup(clear)

	jan30 := time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC)
	jan31 := jan30.AddDate(0, 0, 1)
	feb1 := jan31.AddDate(0, 0, 1)
	type fact struct {
		line, invoice   string
		at              time.Time
		buyer, rep, dsc string
		item, pline     string
		qty, inv, cost  any
	}
	facts := []fact{
		{"l1", "i1", jan30.Add(9 * time.Hour), "b1", "r1", "", "it1", "pl1", "2.5", "10.000000000049999999", "4"},
		{"l2", "i1", jan30.Add(9 * time.Hour), "b1", "r1", "d1", "it2", "pl2", nil, "5.00000000005", nil},
		{"l3", "i2", jan30.Add(14 * time.Hour), "b2", "", "", "it1", "pl1", "1", "-3.25", "1"},
		{"l4", "i3", jan31.Add(23 * time.Hour), "b2", "r2", "d1", "it3", "pl2", nil, nil, nil},
		{"l5", "i4", feb1.Add(1 * time.Hour), "b1", "r1", "", "it1", "pl1", "7", "70", "35"},
	}
	for _, f := range facts {
		_, err := pool.Exec(`INSERT INTO sales_line_fact (account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code,
buyer_account_id, sales_rep_id, order_discount_id, product_id, item_id, product_line_id, quantity_base, total_invoiced, total_cost, refreshed_at)
VALUES (?, ?, ?, ?, 'or_x', 'sales_order', ?, NULLIF(?, ''), NULLIF(?, ''), 'pd_x', ?, ?, ?, ?, ?, NOW(3))`,
			rebuildTestAccountID, f.at, "ivln_rb_"+f.line, "iv_rb_"+f.invoice, f.buyer, f.rep, f.dsc, f.item, f.pline, f.qty, f.inv, f.cost)
		require.NoError(t, err)
	}
	rebuild := func(days ...time.Time) {
		for _, d := range days {
			if apiErr := repo.RebuildRollupDay(ctx, domain.SalesRollupDay{AccountID: rebuildTestAccountID, Day: d}); apiErr != nil {
				t.Fatalf("rebuild %s: %v", d, apiErr.Internal)
			}
		}
	}

	rebuild(jan30, jan31, feb1)
	assertRollupsMatchFacts(t, ctx, pool)

	// Rebuilding an unchanged day is a no-op.
	rebuild(jan30)
	assertRollupsMatchFacts(t, ctx, pool)

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := pool.Exec(q, args...)
		require.NoError(t, err)
	}
	// An amount edit, a rep removed, a line moved into another month, and a line deleted.
	exec(`UPDATE sales_line_fact SET total_invoiced = 11, quantity_base = NULL WHERE invoice_line_id = 'ivln_rb_l1'`)
	exec(`UPDATE sales_line_fact SET sales_rep_id = NULL WHERE invoice_line_id = 'ivln_rb_l2'`)
	exec(`UPDATE sales_line_fact SET invoiced_at = ? WHERE invoice_line_id = 'ivln_rb_l4'`, feb1.Add(2*time.Hour))
	exec(`DELETE FROM sales_line_fact WHERE invoice_line_id = 'ivln_rb_l3'`)
	rebuild(jan30, jan31, feb1)
	assertRollupsMatchFacts(t, ctx, pool)

	// A day whose facts are all gone takes its month rows with it.
	exec(`DELETE FROM sales_line_fact WHERE account_id = ? AND invoiced_at >= ?`, rebuildTestAccountID, feb1)
	rebuild(feb1)
	assertRollupsMatchFacts(t, ctx, pool)
	var febRows int
	require.NoError(t, pool.QueryRow(`SELECT COUNT(*) FROM sales_fact_rollup WHERE account_id = ? AND bucket_start >= ?`, rebuildTestAccountID, feb1).Scan(&febRows))
	require.Zero(t, febRows)
}

func TestRollupDecimalRoundsLikeMySQL(t *testing.T) {
	pool := testDB(t)
	for _, v := range []string{"0.00000000005", "-0.00000000005", "1.000000000049999", "123456.123456789012345", "-7.5"} {
		var stored string
		require.NoError(t, pool.QueryRow(fmt.Sprintf(`SELECT CAST(%s AS DECIMAL(28,10))`, v)).Scan(&stored))
		var sum *big.Rat
		require.NoError(t, addDecimal(&sum, sql.NullString{String: v, Valid: true}))
		require.Equal(t, stored, rollupDecimal(sum).String, v)
	}
}
