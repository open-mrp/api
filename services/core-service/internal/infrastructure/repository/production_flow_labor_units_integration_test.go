//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The cost rollup converts labor time with the base ratios GetFlowStep reads, so the ratios have to be
// the units' own: a step producing eaches, timed in seconds per case of fifty.
//
//	go test -tags integration ./services/core-service/internal/infrastructure/repository/ -run GetFlowStep_ReadsLaborTime
func TestGetFlowStep_ReadsLaborTimeAndProducedUnitRatios(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(prefix string) string { return prefix + "_lu_" + suffix }
	accountID := id("ac")

	type row struct {
		table string
		sql   string
		args  []any
	}
	unit := func(key, abbr, dim, num, den string) row {
		return row{"unit", `INSERT INTO unit (id, name, abbreviation, unit_dimension_code, ratio_numerator, ratio_denominator) VALUES (?, ?, ?, ?, ?, ?)`,
			[]any{id(key), id(key), id(abbr), dim, num, den}}
	}
	rate := func(key, value, num, den string) row {
		return row{"rate", `INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES (?, ?, ?, ?)`,
			[]any{id(key), value, id(num), id(den)}}
	}
	rows := []row{
		unit("un_ea", "ea", "quantity", "1", "1"),
		unit("un_cs50", "cs50ea", "quantity", "50", "1"),
		unit("un_sec", "sec", "time", "1", "3600"),
		unit("un_hr", "hr", "time", "1", "1"),
		unit("un_usd", "usd", "currency", "1", "1"),
		rate("rt_lt", "733.2", "un_sec", "un_cs50"),
		rate("rt_lr", "25.18", "un_usd", "un_hr"),
		rate("rt_oh", "24.14", "un_usd", "un_hr"),
		{"quantity", `INSERT INTO quantity (id, value, unit_id) VALUES (?, 1, ?)`, []any{id("qty"), id("un_ea")}},
		{"item", `INSERT INTO item (id, sku, account_id, item_type_code, unit_value_id, burn_rate_id, unit_cost_id, item_category_id) VALUES (?, ?, ?, 'part', ?, ?, ?, ?)`,
			[]any{id("itm"), id("sku"), accountID, id("rt_lr"), id("rt_lr"), id("rt_lr"), id("ic")}},
		{"production_step", `INSERT INTO production_step (id, name, account_id, labor_time_id, labor_rate_id, overhead_rate_id) VALUES (?, ?, ?, ?, ?, ?)`,
			[]any{id("ps"), id("Box"), accountID, id("rt_lt"), id("rt_lr"), id("rt_oh")}},
		{"production", `INSERT INTO production (id, item_id, quantity_id, production_step_id) VALUES (?, ?, ?, ?)`,
			[]any{id("prod"), id("itm"), id("qty"), id("ps")}},
	}
	for _, r := range rows {
		_, err := pool.ExecContext(ctx, r.sql, r.args...)
		require.NoError(t, err, "seed %s", r.table)
		inserted := r
		t.Cleanup(func() { deleteSeeded(pool, inserted.table, inserted.args[0]) })
	}

	step, apiErr := NewProductionFlowRepo(sqlc.New(pool)).GetFlowStep(ctx, accountID, id("ps"))
	require.Nil(t, apiErr)

	requireRatio(t, "produced unit numerator", step.Production.Quantity.Unit.RatioNumerator, "1")
	requireRatio(t, "produced unit denominator", step.Production.Quantity.Unit.RatioDenominator, "1")
	require.Equal(t, "quantity", step.Production.Quantity.Unit.Type)

	require.NotNil(t, step.LaborTime)
	requireRatio(t, "labor time numerator ratio", step.LaborTime.NumeratorRatio, "0.000277777777777777777777777778")
	requireRatio(t, "labor time denominator ratio", step.LaborTime.DenominatorRatio, "50")
	require.Equal(t, "quantity", step.LaborTime.DenominatorUnitType)

	require.NotNil(t, step.LaborRate)
	requireRatio(t, "labor rate denominator ratio", step.LaborRate.DenominatorRatio, "1")
}

func requireRatio(t *testing.T, label, got, want string) {
	t.Helper()
	g, err := decimal.NewFromString(got)
	require.NoError(t, err, "%s: %q is not a number", label, got)
	require.True(t, g.Sub(decimal.RequireFromString(want)).Abs().LessThan(decimal.RequireFromString("0.000000000000001")),
		"%s = %s, want %s", label, got, want)
}

func deleteSeeded(pool *sql.DB, table string, id any) {
	_, _ = pool.Exec("DELETE FROM "+table+" WHERE id = ?", id)
}
