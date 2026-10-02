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

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// The inventory corpus is the production tenant's catalog and stock: a few thousand items, two in five
// sold, a tenth made only for other products, a few deleted; a couple of available receipts per item,
// each partly allocated half a dozen times, behind a long history of fully allocated ones.
const (
	planInvAccount = "ac_planinvt"
	planInvItems   = 3_000
	planInvSpan    = 2 * 365 * 24 * time.Hour

	// planInvCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planInvCorpusVersion = "Plan Test Inventories v1"

	// planInvBusyItem holds the most receipts.
	planInvBusyItem = 7
	// planInvRareSKU matches one SKU; planInvDenseSKU every SKU.
	planInvRareSKU  = "ZQRARE"
	planInvDenseSKU = "PLANINVT"
)

var planInvOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

func planInvCreatedAt(n int) time.Time {
	return planInvOrigin.Add(time.Duration(n) * (planInvSpan / planInvItems))
}

func planInvItemID(n int) string { return fmt.Sprintf("0000%04d-plan-4ivt-8000-%012d", n, n) }
func planInvSKU(n int) string {
	if n == planInvItems/2 {
		return planInvDenseSKU + "-" + planInvRareSKU
	}
	return fmt.Sprintf("%s-%05d", planInvDenseSKU, n)
}

// planInvReceipts is how many available and allocated receipts item n has.
func planInvReceipts(n int) (available, allocated int) {
	if n == planInvBusyItem {
		return 40, 200
	}
	return 2, 12
}

var planInvCorpusOnce sync.Once

func ensureInventoryCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planInvCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM item WHERE account_id = ?", planInvAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planInvAccount).Scan(&version)
		if have >= planInvItems && version == planInvCorpusVersion {
			return
		}
		t.Logf("seeding the inventory plan corpus (%d items); it is kept for later runs", planInvItems)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE ia FROM inventory_allocation ia JOIN inventory_receipt ir ON ir.id = ia.inventory_receipt_id WHERE ir.owner_account_id = ?", planInvAccount)
		exec("DELETE FROM inventory_receipt WHERE owner_account_id = ?", planInvAccount)
		exec("DELETE p FROM product p JOIN item i ON i.id = p.item_id WHERE i.account_id = ?", planInvAccount)
		exec("DELETE FROM item WHERE account_id = ?", planInvAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qu\\_planivt\\_%'")
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planivt', 'planivt', 'ea', 'quantity', NULL, 1, 1, 1, NOW(3), NOW(3))`)
		exec(`INSERT IGNORE INTO unit_group (id, name, base_unit_id, unit_type_code) VALUES ('ug_planivt', 'Plan Each', 'un_planivt', 'quantity')`)
		exec(`INSERT IGNORE INTO item_category (id, name, item_category_type_code, unit_group_id, account_id)
		      VALUES ('ic_planivt', 'Plan Goods', 'product', 'ug_planivt', ?)`, planInvAccount)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planInvAccount, planInvCorpusVersion)

		var itemVals, prodVals []string
		var itemArgs, prodArgs []any
		for n := range planInvItems {
			created := planInvCreatedAt(n)
			var deleted any
			if n%37 == 36 {
				deleted = created.Add(24 * time.Hour)
			}
			itemVals = append(itemVals, "(?, ?, ?, ?, ?, ?, ?, 'product', 'ic_planivt', ?, ?, ?)")
			itemArgs = append(itemArgs, planInvItemID(n), planInvSKU(n), fmt.Sprintf("Plan stock item %05d, 12x18 kraft", n),
				fmt.Sprintf("uv_planivt_%05d", n), fmt.Sprintf("br_planivt_%05d", n), fmt.Sprintf("uc_planivt_%05d", n),
				planInvAccount, created, created, deleted)
			switch {
			case n%5 < 2:
				prodVals = append(prodVals, "(?, ?, 'sale')")
				prodArgs = append(prodArgs, fmt.Sprintf("prod_planivt_%05d", n), planInvItemID(n))
			case n%10 == 2:
				prodVals = append(prodVals, "(?, ?, 'component')")
				prodArgs = append(prodArgs, fmt.Sprintf("prod_planivt_%05d", n), planInvItemID(n))
			}
		}
		exec(`INSERT INTO item (id, sku, description, unit_value_id, burn_rate_id, unit_cost_id, account_id, item_type_code,
		      item_category_id, created_at, updated_at, deleted_at) VALUES `+strings.Join(itemVals, ","), itemArgs...)
		exec(`INSERT INTO product (id, item_id, product_type_code) VALUES `+strings.Join(prodVals, ","), prodArgs...)

		var qVals, rVals, aVals []string
		var qArgs, rArgs, aArgs []any
		flush := func() {
			if len(qVals) > 0 {
				exec(`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+strings.Join(qVals, ","), qArgs...)
			}
			if len(rVals) > 0 {
				exec(`INSERT INTO inventory_receipt (id, owner_account_id, holder_account_id, item_id, received_at, quantity_id,
				      unit_cost_id, status_code, created_at, updated_at) VALUES `+strings.Join(rVals, ","), rArgs...)
			}
			if len(aVals) > 0 {
				exec(`INSERT INTO inventory_allocation (id, inventory_receipt_id, inventory_issue_id, quantity_id, unit_cost_id,
				      total_cost_id, created_at, updated_at) VALUES `+strings.Join(aVals, ","), aArgs...)
			}
			qVals, rVals, aVals, qArgs, rArgs, aArgs = nil, nil, nil, nil, nil, nil
		}
		for n := range planInvItems {
			available, allocated := planInvReceipts(n)
			for k := range available + allocated {
				rID := fmt.Sprintf("inrc_planivt_%05d_%03d", n, k)
				status := "allocated"
				if k < available {
					status = "available"
				}
				received := planInvCreatedAt(n).Add(time.Duration(k) * time.Hour)
				qID := "qu_planivt_" + rID
				qVals, qArgs = append(qVals, "(?, '100', 'un_planivt', ?, ?)"), append(qArgs, qID, received, received)
				rVals = append(rVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
				rArgs = append(rArgs, rID, planInvAccount, planInvAccount, planInvItemID(n), received, qID, "rt_"+rID, status, received, received)
				if status == "available" {
					for a := range 6 {
						aID := fmt.Sprintf("inal_planivt_%05d_%03d_%d", n, k, a)
						aq := "qu_planivt_" + aID
						qVals, qArgs = append(qVals, "(?, '5', 'un_planivt', ?, ?)"), append(qArgs, aq, received, received)
						aVals = append(aVals, "(?, ?, ?, ?, ?, ?, ?, ?)")
						aArgs = append(aArgs, aID, rID, "inis_"+aID, aq, "rt_uc_"+aID, "rt_tc_"+aID, received, received)
					}
				}
			}
			if len(qVals) > 2_000 {
				flush()
			}
		}
		flush()
		exec("ANALYZE TABLE item, product, inventory_receipt, inventory_allocation")
	})
}

// planInventoriesParams is what ListInventories asks of the item list.
func planInventoriesParams(p domain.ListInventoriesParams) domain.ListItemsParams {
	return domain.ListItemsParams{AccountID: planInvAccount, Cursor: p.Cursor, Limit: p.Limit, Query: p.Query}
}

func inventoryPlanCases() []planCase[domain.ListInventoriesParams] {
	str := func(s string) *string { return &s }
	mid := planInvCreatedAt(planInvItems / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListInventoriesParams) {
		return func(p *domain.ListInventoriesParams) { p.Cursor = planCursorAt(mid, "~", dir) }
	}
	type P = domain.ListInventoriesParams
	return planCases(
		P{Limit: 25},
		[]planDim[P]{
			{"search", []planValue[P]{
				{"sku-one", func(p *P) { p.Query = str(planInvRareSKU) }},
				{"sku-every", func(p *P) { p.Query = str(planInvDenseSKU) }},
			}},
		},
		[]planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// inventorySearchFloor is, for a request that searches, how many items the account has, or 0 without
// a search: a substring of a SKU or description is no B-tree's, so a plan tests every item until it
// has the page, which for a rare term is all of them.
func inventorySearchFloor(t *testing.T, db *sql.DB, p domain.ListInventoriesParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM item WHERE account_id = ?", planInvAccount).Scan(&n))
	return n
}

// TestInventoryList_ReadsAboutAPage holds the item page ListInventories reads to about a page of items
// (listPlanSuite), for every request it accepts.
func TestInventoryList_ReadsAboutAPage(t *testing.T) {
	ensureInventoryCorpus(t)
	listPlanSuite[domain.ListInventoriesParams]{
		table: "item", scopeColumn: "account_id",
		from: "FROM item i", alias: "i",
		cases: inventoryPlanCases(),
		limit: func(p domain.ListInventoriesParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListInventoriesParams) error {
			if _, apiErr := NewItemRepo(q).List(ctx, planInventoriesParams(p)); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: inventorySearchFloor,
	}.run(t)
}

// TestInventoryOnHand_ReadsWhatItTotals holds ListInventories' on-hand figures to reading only the
// available receipts and allocations they total (lookupPlanSuite).
func TestInventoryOnHand_ReadsWhatItTotals(t *testing.T) {
	ensureInventoryCorpus(t)
	db := planDB(t)
	onHand := func(ids ...string) func(ctx context.Context, q *sqlc.Queries) error {
		return func(ctx context.Context, q *sqlc.Queries) error {
			if _, apiErr := NewInventoryQueryRepo(q).FetchOnHandInventoryBulk(ctx, ids, planInvAccount); apiErr != nil {
				return apiErr
			}
			return nil
		}
	}
	var page []string
	for n := planInvItems - 1; len(page) < 25; n-- {
		page = append(page, planInvItemID(n))
	}
	lookupPlanSuite{
		tables: []string{"inventory_receipt"},
		cases: []lookupPlanCase{
			{"page", onHand(page...)},
			{"busy-item", onHand(planInvItemID(planInvBusyItem))},
		},
		// Each item's figure totals its available receipts and their allocations.
		returned: func(t *testing.T, stmt explainedStatement) float64 {
			var n float64
			require.NoError(t, db.QueryRow(`SELECT
				(SELECT COUNT(*) FROM inventory_receipt ir WHERE ir.owner_account_id = ? AND ir.status_code = 'available' AND ir.item_id IN (`+placeholders(len(page))+`, ?))
				+ (SELECT COUNT(*) FROM inventory_allocation ia JOIN inventory_receipt ir ON ir.id = ia.inventory_receipt_id
				   WHERE ir.owner_account_id = ? AND ir.status_code = 'available' AND ir.item_id IN (`+placeholders(len(page))+`, ?))`,
				append(append(append([]any{planInvAccount}, stringArgs(page)...), planInvItemID(planInvBusyItem), planInvAccount),
					append(stringArgs(page), planInvItemID(planInvBusyItem))...)...).Scan(&n))
			return n
		},
	}.run(t)
}
