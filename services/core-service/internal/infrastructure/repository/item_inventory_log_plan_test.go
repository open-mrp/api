//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The inventory-log corpus is one tenant's level history: a busy item logged about forty times a day
// for two years, a few hundred items logged a couple of times a month, and items never logged. Logs
// are written in a pair unit as well as the each base, so every read converts.
const (
	planILogAccount = "ac_planilog"
	planILogItems   = 300
	planILogSpan    = 2 * 365 * 24 * time.Hour

	// planILogCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planILogCorpusVersion = "Plan Test Inventory Logs v1"

	planILogBusyItem   = 0
	planILogBusyLogs   = 30_000
	planILogQuietLogs  = 48
	planILogNeverItems = 10 // the last items of the corpus
)

var planILogOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

// planILogNow is the end of the corpus's history, standing in for today.
var planILogNow = planILogOrigin.Add(planILogSpan)

func planILogItemID(n int) string { return fmt.Sprintf("0000%04d-plan-4ilg-8000-%012d", n, n) }

func planILogCount(n int) int {
	switch {
	case n == planILogBusyItem:
		return planILogBusyLogs
	case n >= planILogItems-planILogNeverItems:
		return 0
	default:
		return planILogQuietLogs
	}
}

var planILogCorpusOnce sync.Once

func ensureInventoryLogCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planILogCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM inventory_log WHERE account_id = ?", planILogAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planILogAccount).Scan(&version)
		want := planILogBusyLogs + (planILogItems-1-planILogNeverItems)*planILogQuietLogs
		if have == want && version == planILogCorpusVersion {
			return
		}
		t.Logf("seeding the inventory log plan corpus (%d logs); it is kept for later runs", want)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE q FROM quantity q JOIN inventory_log l ON l.quantity_id = q.id WHERE l.account_id = ?", planILogAccount)
		exec("DELETE FROM inventory_log WHERE account_id = ?", planILogAccount)
		exec("DELETE p FROM product p JOIN item i ON i.id = p.item_id WHERE i.account_id = ?", planILogAccount)
		exec("DELETE FROM item WHERE account_id = ?", planILogAccount)
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planilg', 'planilg', 'ea', 'quantity', NULL, 1, 1, 1, NOW(3), NOW(3)),
		             ('un_planilgpr', 'planilg pair', 'pr', 'quantity', NULL, 2, 1, 0, NOW(3), NOW(3))`)
		exec(`INSERT IGNORE INTO unit_group (id, name, base_unit_id, unit_type_code) VALUES ('ug_planilg', 'Plan Log Each', 'un_planilg', 'quantity')`)
		exec(`INSERT IGNORE INTO item_category (id, name, item_category_type_code, unit_group_id, account_id)
		      VALUES ('ic_planilg', 'Plan Logged Goods', 'product', 'ug_planilg', ?)`, planILogAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planILogAccount, planILogCorpusVersion)

		var itemVals, prodVals []string
		var itemArgs, prodArgs []any
		for n := range planILogItems {
			itemVals = append(itemVals, "(?, ?, ?, ?, ?, ?, ?, 'product', 'ic_planilg', ?, ?)")
			itemArgs = append(itemArgs, planILogItemID(n), fmt.Sprintf("PLANILOG-%05d", n), fmt.Sprintf("Plan logged item %05d", n),
				fmt.Sprintf("uv_planilg_%05d", n), fmt.Sprintf("br_planilg_%05d", n), fmt.Sprintf("uc_planilg_%05d", n),
				planILogAccount, planILogOrigin, planILogOrigin)
			if n%3 == 0 {
				prodVals = append(prodVals, "(?, ?, 'sale', ?)")
				prodArgs = append(prodArgs, fmt.Sprintf("prod_planilg_%05d", n), planILogItemID(n), "pdln_planilg")
			}
		}
		exec(`INSERT INTO item (id, sku, description, unit_value_id, burn_rate_id, unit_cost_id, account_id, item_type_code,
		      item_category_id, created_at, updated_at) VALUES `+strings.Join(itemVals, ","), itemArgs...)
		exec(`INSERT INTO product (id, item_id, product_type_code, product_line_id) VALUES `+strings.Join(prodVals, ","), prodArgs...)

		var qVals, lVals []string
		var qArgs, lArgs []any
		flush := func() {
			if len(qVals) > 0 {
				exec(`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+strings.Join(qVals, ","), qArgs...)
				exec(`INSERT INTO inventory_log (id, item_id, quantity_id, account_id, created_at, updated_at) VALUES `+strings.Join(lVals, ","), lArgs...)
			}
			qVals, lVals, qArgs, lArgs = nil, nil, nil, nil
		}
		for n := range planILogItems {
			logs := planILogCount(n)
			for k := range logs {
				at := planILogOrigin.Add(time.Duration(k+1) * (planILogSpan / time.Duration(logs+1)))
				id := fmt.Sprintf("inlg_planilg_%05d_%05d", n, k)
				unit := "un_planilg"
				if k%2 == 1 {
					unit = "un_planilgpr"
				}
				qVals = append(qVals, "(?, ?, ?, ?, ?)")
				qArgs = append(qArgs, "qu_"+id, fmt.Sprintf("%d", k%500), unit, at, at)
				lVals = append(lVals, "(?, ?, ?, ?, ?, ?)")
				lArgs = append(lArgs, id, planILogItemID(n), "qu_"+id, planILogAccount, at, at)
				if len(lVals) >= 2_000 {
					flush()
				}
			}
		}
		flush()
		exec("ANALYZE TABLE item, product, quantity, inventory_log")
	})
}

// planILogWindow is the trend window the service would ask for on the corpus's last day.
func planILogWindow() (start, end time.Time) {
	today := planILogNow.Add(-time.Nanosecond).Truncate(24 * time.Hour)
	return today.AddDate(0, 0, -29), today.AddDate(0, 0, 1)
}

// planILogLogsBetween is how many logs the item has in [from, to): what a day-by-day close of the
// window has to look at, read off the index.
func planILogLogsBetween(t *testing.T, db *sql.DB, itemID string, from, to time.Time) float64 {
	t.Helper()
	var n float64
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM inventory_log WHERE account_id = ? AND item_id = ? AND created_at >= ? AND created_at < ?`,
		planILogAccount, itemID, from, to).Scan(&n))
	return n
}

// TestItemTrends_ReadOnlyTheWindow holds an item's trend to one backward dive for the level before
// the window and an index-only pass over the window's logs, reading the closing log of each day in
// full and nothing else, for the busiest item as for one never logged.
func TestItemTrends_ReadOnlyTheWindow(t *testing.T) {
	ensureInventoryLogCorpus(t)
	db := planDB(t)
	start, end := planILogWindow()

	seed := func(n int) func(ctx context.Context, q *sqlc.Queries) error {
		return func(ctx context.Context, q *sqlc.Queries) error {
			_, apiErr := NewItemRepo(q).GetInventoryLevelBefore(ctx, planILogAccount, planILogItemID(n), start)
			if apiErr != nil {
				return apiErr
			}
			return nil
		}
	}
	closings := func(n int) func(ctx context.Context, q *sqlc.Queries) error {
		return func(ctx context.Context, q *sqlc.Queries) error {
			levels, apiErr := NewItemRepo(q).ListDailyClosingInventoryLevels(ctx, planILogAccount, planILogItemID(n), start, end)
			if apiErr != nil {
				return apiErr
			}
			if n == planILogBusyItem && len(levels) != 30 {
				return fmt.Errorf("the busy item logs every day, so it closes 30 of them; got %d", len(levels))
			}
			return nil
		}
	}

	lookupPlanSuite{
		cases: []lookupPlanCase{
			{"busy-seed", seed(planILogBusyItem)},
			{"busy-closings", closings(planILogBusyItem)},
			{"quiet-seed", seed(1)},
			{"quiet-closings", closings(1)},
			{"never-logged-seed", seed(planILogItems - 1)},
			{"never-logged-closings", closings(planILogItems - 1)},
		},
		explainRows: true,
		// The closings statement finds each day's last log by scanning the window on the index, so it
		// stands for every log in the window.
		returned: func(t *testing.T, stmt explainedStatement) float64 {
			if !strings.Contains(stmt.query, "closed_at") {
				return 0
			}
			return planILogLogsBetween(t, db, stmt.args[1].(string), start, end)
		},
	}.run(t)
}

// TestInventoryLevelsAsOf_OneDivePerItem holds the as-of inventory list to one backward dive per item
// on a page, however long each item's history, and the page's product lines to one lookup per item.
func TestInventoryLevelsAsOf_OneDivePerItem(t *testing.T) {
	ensureInventoryLogCorpus(t)

	page := make([]string, 0, 25)
	for n := range 15 {
		page = append(page, planILogItemID(n))
	}
	for n := planILogItems - 10; n < planILogItems; n++ {
		page = append(page, planILogItemID(n))
	}
	asOf := planILogOrigin.Add(planILogSpan / 2)

	lookupPlanSuite{
		cases: []lookupPlanCase{
			{"page-mid-history", func(ctx context.Context, q *sqlc.Queries) error {
				rows, apiErr := NewInventoryQueryRepo(q).FetchInventoryLevelsAsOf(ctx, page, planILogAccount, asOf)
				if apiErr != nil {
					return apiErr
				}
				if len(rows) != len(page) {
					return fmt.Errorf("every item on the page is reported, logged or not; got %d of %d", len(rows), len(page))
				}
				return nil
			}},
			{"busy-item-now", func(ctx context.Context, q *sqlc.Queries) error {
				_, apiErr := NewInventoryQueryRepo(q).FetchInventoryLevelsAsOf(ctx, []string{planILogItemID(planILogBusyItem)}, planILogAccount, planILogNow)
				if apiErr != nil {
					return apiErr
				}
				return nil
			}},
			{"page-product-lines", func(ctx context.Context, q *sqlc.Queries) error {
				_, apiErr := NewItemRepo(q).GetProductLineIDs(ctx, planILogAccount, page)
				if apiErr != nil {
					return apiErr
				}
				return nil
			}},
		},
	}.run(t)
}

// TestItemsExport_ReadsWhatItTotals holds the items export to reading each item once and only the
// available receipts and allocations its on-hand figures total (lookupPlanSuite).
func TestItemsExport_ReadsWhatItTotals(t *testing.T) {
	ensureInventoryCorpus(t)
	db := planDB(t)

	lookupPlanSuite{
		tables: []string{"inventory_receipt"},
		cases: []lookupPlanCase{
			{"whole-catalog", func(ctx context.Context, q *sqlc.Queries) error {
				result, apiErr := NewItemRepo(q).ExportWithInventory(ctx, planInvAccount)
				if apiErr != nil {
					return apiErr
				}
				if len(result.Items) == 0 {
					return fmt.Errorf("the corpus has items to export")
				}
				return nil
			}},
		},
		// The export totals every live item's available receipts and their allocations.
		returned: func(t *testing.T, _ explainedStatement) float64 {
			var n float64
			require.NoError(t, db.QueryRow(`SELECT
				(SELECT COUNT(*) FROM item WHERE account_id = ? AND deleted_at IS NULL)
				+ (SELECT COUNT(*) FROM inventory_receipt ir WHERE ir.owner_account_id = ? AND ir.status_code = 'available')
				+ (SELECT COUNT(*) FROM inventory_allocation ia JOIN inventory_receipt ir ON ir.id = ia.inventory_receipt_id
				   WHERE ir.owner_account_id = ? AND ir.status_code = 'available')`,
				planInvAccount, planInvAccount, planInvAccount).Scan(&n))
			return n
		},
	}.run(t)
}
