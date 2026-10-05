//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// Labor efficiency weighs every batch by the standard labor hours it stands for, so steps timed in
// different units have to land on one footing:
//
//	go test -tags integration ./services/core-service/internal/infrastructure/repository/ -run LaborEfficiency
//
// Boxing: 733.2 s per case of fifty (14.664 s an each); 100 good and 10 waste, counted in eaches.
// Sewing: 0.5 min per pair (15 s an each); 20 good and 2 waste pairs, 4 seconds-quality eaches.
// Dyeing: 6 s per pound, counted in eaches; no conversion exists, so 6 s an each.
func TestManufacturingLaborEfficiency_WeighsBatchesInStandardHours(t *testing.T) {
	ctx := context.Background()
	pool := testDB(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(key string) string { return key + "_le_" + suffix }
	accountID := id("ac")

	var seeded []func()
	t.Cleanup(func() {
		for i := len(seeded) - 1; i >= 0; i-- {
			seeded[i]()
		}
	})
	exec := func(table, query string, args ...any) {
		t.Helper()
		_, err := pool.ExecContext(ctx, query, args...)
		require.NoError(t, err, "seed %s", table)
		rowID := args[0]
		seeded = append(seeded, func() { _, _ = pool.Exec("DELETE FROM "+table+" WHERE id = ?", rowID) })
	}
	unit := func(key, dimension, num, den string) {
		exec("unit", `INSERT INTO unit (id, name, abbreviation, unit_dimension_code, ratio_numerator, ratio_denominator, account_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id(key), id(key), id(key), dimension, num, den, accountID)
	}
	quantity := func(key, value, u string) string {
		exec("quantity", `INSERT INTO quantity (id, value, unit_id) VALUES (?, ?, ?)`, id(key), value, id(u))
		return id(key)
	}
	rate := func(key, value, num, den string) string {
		exec("rate", `INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES (?, ?, ?, ?)`, id(key), value, id(num), id(den))
		return id(key)
	}
	step := func(key, laborTime, timeUnit, perUnit string) {
		exec("production_step", `INSERT INTO production_step (id, name, account_id, labor_time_id, labor_rate_id, overhead_rate_id) VALUES (?, ?, ?, ?, ?, ?)`,
			id(key), id(key), accountID, rate("lt_"+key, laborTime, timeUnit, perUnit), rate("lr_"+key, "25", "usd", "hr"), rate("oh_"+key, "20", "usd", "hr"))
	}
	scannedAt := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	batch := func(key, stepKey, good, waste, seconds, u string) {
		var stepID any
		if stepKey != "" {
			stepID = id(stepKey)
		}
		exec("batch", `INSERT INTO batch (id, account_id, item_id, quantity_id, waste_quantity_id, seconds_quantity_id, production_step_id, scanned_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id(key), accountID, id("itm"), quantity("qg_"+key, good, u), quantity("qw_"+key, waste, u), quantity("qs_"+key, seconds, u), stepID, scannedAt)
	}

	unit("ea", "quantity", "1", "1")
	unit("pr", "quantity", "2", "1")
	unit("cs50", "quantity", "50", "1")
	unit("lbs", "mass", "453.59237", "1")
	unit("sec", "time", "1", "3600")
	unit("min", "time", "1", "60")
	unit("hr", "time", "1", "1")
	unit("usd", "currency", "1", "1")

	step("box", "733.2", "sec", "cs50")
	step("sew", "0.5", "min", "pr")
	step("dye", "6", "sec", "lbs")
	batch("b_box", "box", "100", "10", "0", "ea")
	batch("b_sew", "sew", "20", "2", "0", "pr")
	batch("b_sew_seconds", "sew", "0", "0", "4", "ea")
	batch("b_dye", "dye", "10", "0", "0", "ea")
	batch("b_no_step", "", "1000", "1000", "1000", "ea")

	queries := sqlc.New(pool)
	window := struct{ start, end time.Time }{scannedAt.Add(-time.Hour), scannedAt.Add(time.Hour)}

	row, err := queries.GetManufacturingLaborEfficiency(ctx, sqlc.GetManufacturingLaborEfficiencyParams{
		OwnerAccountID: accountID,
		StartDate:      sql.NullTime{Time: window.start, Valid: true},
		EndDate:        sql.NullTime{Time: window.end, Valid: true},
	})
	require.NoError(t, err)
	requireHours(t, "labor quantity", row.LaborQuantity, "0.590666666666666666666666666667")
	requireHours(t, "labor waste", row.LaborWaste, "0.0574")
	requireHours(t, "labor seconds", row.LaborSeconds, "0.016666666666666666666666666667")

	batchRow, err := queries.GetManufacturingBatchBatchMetrics(ctx, sqlc.GetManufacturingBatchBatchMetricsParams{
		OwnerAccountID: accountID,
		StartDate:      sql.NullTime{Time: window.start, Valid: true},
		EndDate:        sql.NullTime{Time: window.end, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, row.LaborQuantity, batchRow.LaborQuantity, "the two queries must weigh batches identically")
	require.Equal(t, row.LaborWaste, batchRow.LaborWaste)
	require.Equal(t, row.LaborSeconds, batchRow.LaborSeconds)

	repo := NewAnalyticsRepo(queries)
	const wantEfficiency = 0.8885768729315013539
	got, apiErr := repo.GetManufacturingMetric(ctx, domain.AnalyzeManufacturingParams{
		AccountID: accountID, StartDate: window.start, EndDate: window.end, Type: "laborEfficiency",
	})
	require.Nil(t, apiErr)
	require.InDelta(t, wantEfficiency, got, 1e-12)

	result, apiErr := repo.GetManufacturingBatch(ctx, domain.AnalyzeManufacturingBatchParams{
		AccountID: accountID, StartDate: window.start, EndDate: window.end,
		ComparisonStartDate: window.start.AddDate(0, -1, 0), ComparisonEndDate: window.end.AddDate(0, -1, 0),
	})
	require.Nil(t, apiErr)
	require.InDelta(t, wantEfficiency, result.Current.LaborEfficiency, 1e-12)
	require.True(t, math.Abs(result.Comparison.LaborEfficiency) < 1e-12, "an empty period has no efficiency")
}

func requireHours(t *testing.T, label, got, want string) {
	t.Helper()
	g, err := decimal.NewFromString(got)
	require.NoError(t, err)
	require.True(t, g.Sub(decimal.RequireFromString(want)).Abs().LessThan(decimal.RequireFromString("0.000000000001")),
		"%s = %s hours, want %s", label, got, want)
}
