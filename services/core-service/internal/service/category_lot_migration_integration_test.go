//go:build integration

package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
)

// The data migration rewrites every account at once, so it only runs against a scratch database migrated to the
// current schema:
//
//	DATA_MIGRATION_TEST_DSN='root:Testing123!@tcp(localhost:3306)/scratch?parseTime=true' \
//	  go test -tags integration ./services/core-service/internal/service/ -run CategoryLotMigration
const categoryLotMigration = "../../../../shared/db/data-migrations/00018_product_line_supply_and_category_lots.sql"

func TestCategoryLotMigration_ItemsKeepTheirLineLot(t *testing.T) {
	dsn := os.Getenv("DATA_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("DATA_MIGRATION_TEST_DSN not set; this test rewrites every account and needs a scratch database")
	}
	pool, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, pool.Ping())

	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	id := func(prefix string) string { return prefix + "_lm" + sfx }
	exec := func(table, rowID, query string, args ...any) {
		t.Helper()
		_, err := pool.Exec(query, args...)
		require.NoError(t, err, query)
		t.Cleanup(func() { _, _ = pool.Exec("DELETE FROM `"+table+"` WHERE id = ?", rowID) })
	}

	account := id("ac")
	pair := id("un_pair")
	exec("unit", pair, "INSERT INTO unit (id, name, abbreviation, unit_dimension_code, is_base_unit) VALUES (?, ?, ?, 'count', 1)", pair, pair, pair)
	group := id("ug")
	exec("unit_group", group, "INSERT INTO unit_group (id, name, base_unit_id, unit_type_code) VALUES (?, ?, ?, 'count')", group, group, pair)

	category := id("ic")
	exec("item_category", category, "INSERT INTO item_category (id, name, item_category_type_code, unit_group_id, account_id) VALUES (?, ?, 'product', ?, ?)",
		category, category, group, account)
	systemCategory := id("ic_sys")
	exec("item_category", systemCategory, "INSERT INTO item_category (id, name, item_category_type_code, unit_group_id) VALUES (?, ?, 'product', ?)",
		systemCategory, systemCategory, group)

	quantity := func(name, value string) string {
		qid := id("qt_" + name)
		exec("quantity", qid, "INSERT INTO quantity (id, value, unit_id) VALUES (?, ?, ?)", qid, value, pair)
		return qid
	}
	line := func(name, value string) string {
		lid := id("pl_" + name)
		exec("product_line", lid, "INSERT INTO product_line (id, name, account_id, default_lot_id) VALUES (?, ?, ?, ?)", lid, lid, account, quantity(name, value))
		return lid
	}
	item := func(name, categoryID string) string {
		iid := id("it_" + name)
		exec("item", iid, "INSERT INTO item (id, item_category_id, unit_cost_id, item_type_code, account_id, burn_rate_id, unit_value_id, sku) VALUES (?, ?, ?, 'product', ?, ?, ?, ?)",
			iid, categoryID, iid+"_c", account, iid+"_b", iid+"_v", iid)
		return iid
	}
	sell := func(itemID, lineID string) {
		pid := id("pr_" + itemID)
		exec("product", pid, "INSERT INTO product (id, item_id, product_type_code, product_line_id) VALUES (?, ?, 'standard', ?)", pid, itemID, lineID)
	}
	// produceFrom makes finished out of each input at one step, which is how an intermediate is tied to the line it becomes.
	produceFrom := func(finished string, inputs ...string) {
		step := id("ps_" + finished)
		exec("production_step", step, "INSERT INTO production_step (id, account_id, name, overhead_rate_id, labor_time_id, labor_rate_id) VALUES (?, ?, ?, ?, ?, ?)",
			step, account, step, step+"_o", step+"_t", step+"_r")
		pn := id("pn_" + finished)
		exec("production", pn, "INSERT INTO production (id, item_id, quantity_id, production_step_id) VALUES (?, ?, ?, ?)", pn, finished, pn+"_q", step)
		for _, in := range inputs {
			cn := id("cn_" + finished + in)
			exec("consumption", cn, "INSERT INTO consumption (id, item_id, quantity_id, waste_quantity_id, production_step_id) VALUES (?, ?, ?, ?, ?)", cn, in, cn+"_q", cn+"_w", step)
		}
	}

	// Line A (60) carries most of the category; line D holds an equal 60 in its own quantity row; line B is 48.
	lineA, lineB, lineD := line("a", "60"), line("b", "48"), line("d", "60")
	a1, a2, a3, a4 := item("a1", category), item("a2", category), item("a3", category), item("a4", category)
	d1 := item("d1", category)
	b1 := item("b1", category)
	b2 := item("b2", category)
	sys := item("sys", systemCategory)
	for _, it := range []string{a1, a2, a3, a4} {
		sell(it, lineA)
	}
	sell(d1, lineD)
	sell(b1, lineB)
	sell(b2, lineB)
	sell(sys, lineA)

	// g1 only becomes a line B product; g2 becomes products of both lines, so it has no single old lot.
	g1, g2 := item("g1", category), item("g2", category)
	produceFrom(b1, g1, g2)
	produceFrom(a1, g2)

	// b2 already has a hand-set override, which the migration must not touch.
	presetID := id("pnscitsd")
	exec("production_schedule_item_setting", presetID, "INSERT INTO production_schedule_item_setting (id, account_id, item_id, lot_multiple_units) VALUES (?, ?, ?, 30)",
		presetID, account, b2)
	for _, it := range []string{a1, a2, a3, a4, d1, b1, sys, g1, g2} {
		itemID := it
		t.Cleanup(func() { _, _ = pool.Exec("DELETE FROM production_schedule_item_setting WHERE item_id = ?", itemID) })
	}

	runCategoryLotMigration(t, pool)
	runCategoryLotMigration(t, pool)

	var categoryLot string
	require.NoError(t, pool.QueryRow("SELECT default_lot_size_id FROM item_category WHERE id = ?", category).Scan(&categoryLot))
	assert.Equal(t, id("qt_a"), categoryLot)

	var overrides int
	require.NoError(t, pool.QueryRow("SELECT COUNT(*) FROM production_schedule_item_setting WHERE account_id = ?", account).Scan(&overrides))
	assert.Equal(t, 4, overrides, "b1, g1 and sys gain an override; b2 keeps its own")

	repos := repository.NewRepoFactory(sqlc.New(pool), nil)
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		itemID   string
		quantity float64
		source   string
	}{
		{"majority line", a1, 60, scheduling.LotSourceItemCategory},
		{"equal lot in another row", d1, 60, scheduling.LotSourceItemCategory},
		{"minority line", b1, 48, scheduling.LotSourceItemOverride},
		{"existing override", b2, 30, scheduling.LotSourceItemOverride},
		{"system category", sys, 60, scheduling.LotSourceItemOverride},
		{"intermediate of minority line", g1, 48, scheduling.LotSourceItemOverride},
		{"intermediate of both lines", g2, 60, scheduling.LotSourceItemCategory},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lot, apiErr := resolveItemLotDefault(ctx, repos, account, tc.itemID, pair)
			require.Nil(t, apiErr)
			assert.Equal(t, tc.quantity, lot.Quantity)
			assert.Equal(t, pair, lot.UnitID)
			assert.Equal(t, tc.source, lot.Source)
		})
	}
}

// runCategoryLotMigration applies the migration's Up statements one at a time, as goose does.
func runCategoryLotMigration(t *testing.T, pool *sql.DB) {
	t.Helper()
	raw, err := os.ReadFile(categoryLotMigration)
	require.NoError(t, err)
	up := strings.SplitN(strings.SplitN(string(raw), "-- +goose Down", 2)[0], "-- +goose Up", 2)[1]
	for _, stmt := range strings.Split(up, ";\n") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		_, err := pool.Exec(stmt)
		require.NoError(t, err, stmt)
	}
}
