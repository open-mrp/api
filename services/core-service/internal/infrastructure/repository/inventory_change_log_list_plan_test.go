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

// The change-log corpus is one manufacturer shaped like the largest production tenant: three in four
// entries are station scans, most of the rest corrections, and record creates and deletes that stopped
// a year before the newest entry, so a recent window holds none of them. A few operators log most of
// the volume, one item dominates, and a rare operator and a rare item each hold a handful of entries.
// Row widths follow production: UUID item, station, and user IDs, short prefixed change-log and
// quantity IDs.
const (
	planICLAccount = "ac_planinv"
	planICLRows    = 50_000
	planICLItems   = 2_000
	planICLUsers   = 20
	planICLSpan    = 2 * 365 * 24 * time.Hour

	// planICLCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planICLCorpusVersion = "Plan Test Inventory v1"

	// planICLRareSKU matches only the rare item's SKU; planICLDenseSKU matches every item's.
	planICLRareSKU  = "ZQRARE"
	planICLDenseSKU = "PLANINV"
)

var planICLOrigin = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

// planICLNow is the newest entry, standing in for the clock the service's default window counts back from.
var planICLNow = planICLCreatedAt(planICLRows - 1)

func planICLCreatedAt(i int) time.Time {
	return planICLOrigin.Add(time.Duration(i) * (planICLSpan / planICLRows))
}

func planICLID(i int) string       { return fmt.Sprintf("inchlg_%012d", i) }
func planICLItemID(n int) string   { return fmt.Sprintf("0000%04d-plan-4inv-8000-%012d", n, n) }
func planICLUserID(u int) string   { return fmt.Sprintf("0000%04d-plan-4usr-8000-%012d", u, u) }
func planICLStation(s int) string  { return fmt.Sprintf("0000%04d-plan-4sta-8000-%012d", s, s) }
func planICLItemSKU(n int) string {
	if n == planICLItems-1 {
		return planICLDenseSKU + "-" + planICLRareSKU
	}
	return fmt.Sprintf("%s-%05d", planICLDenseSKU, n)
}

var planICLRareRows = map[int]bool{150: true, 25_000: true, 49_990: true}

// planICLItem gives one item a twentieth of the volume and spreads the rest; the last item is rare.
func planICLItem(i int) int {
	if planICLRareRows[i] {
		return planICLItems - 1
	}
	if i%20 == 0 {
		return 0
	}
	return 1 + (i*7919)%(planICLItems-2)
}

// planICLUser gives a third of the volume to one operator and tails off; the last operator is rare.
func planICLUser(i int) *string {
	switch {
	case i%500 == 7:
		return nil
	case planICLRareRows[i]:
		u := planICLUserID(planICLUsers - 1)
		return &u
	case i%3 == 0:
		u := planICLUserID(0)
		return &u
	}
	u := planICLUserID(1 + (i*31)%(planICLUsers-2))
	return &u
}

// planICLAction is mostly scans; record creates and deletes stop at 60% of the span, and system
// actions start at 70%.
func planICLAction(i int) string {
	early := i < planICLRows*6/10
	switch {
	case early && i%1000 == 1:
		return "delete_record"
	case early && i%20 == 3:
		return "create_record"
	case !early && i >= planICLRows*7/10 && i%10 == 4:
		return "system_action"
	case i%200 == 5:
		return "user_action"
	case i%7 == 2:
		return "user_correction"
	}
	return "scan"
}

var planICLCorpusOnce sync.Once

func ensureInventoryChangeLogCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planICLCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM inventory_change_log WHERE account_id = ?", planICLAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planICLAccount).Scan(&version)
		if have >= planICLRows && version == planICLCorpusVersion {
			return
		}
		t.Logf("seeding the inventory change log plan corpus (%d entries); it is kept for later runs", planICLRows)

		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM inventory_change_log WHERE account_id = ?", planICLAccount)
		exec("DELETE FROM quantity WHERE id LIKE 'qu\\_planinv\\_%'")
		exec("DELETE FROM item WHERE account_id = ?", planICLAccount)
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planinv', 'planinv', 'un_planinv', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
		      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planICLAccount, planICLCorpusVersion)
		for u := range planICLUsers {
			exec("INSERT IGNORE INTO `user` (id, name) VALUES (?, ?)", planICLUserID(u), fmt.Sprintf("Plan Operator %02d", u))
		}

		var itemVals []string
		var itemArgs []any
		for n := range planICLItems {
			itemVals = append(itemVals, "(?, ?, ?, ?, ?, ?, 'material', ?, ?, ?)")
			itemArgs = append(itemArgs, planICLItemID(n), planICLItemSKU(n), fmt.Sprintf("uv_planinv_%05d", n),
				fmt.Sprintf("br_planinv_%05d", n), fmt.Sprintf("uc_planinv_%05d", n), planICLAccount,
				"ic_planinv", planICLOrigin, planICLOrigin)
		}
		exec(`INSERT INTO item (id, sku, unit_value_id, burn_rate_id, unit_cost_id, account_id, item_type_code,
		      item_category_id, created_at, updated_at) VALUES `+strings.Join(itemVals, ","), itemArgs...)

		const batch = 1_000
		for start := 0; start < planICLRows; start += batch {
			var qVals, lVals []string
			var qArgs, lArgs []any
			for i := start; i < start+batch; i++ {
				createdAt := planICLCreatedAt(i)
				action := planICLAction(i)
				var station any
				if action == "scan" {
					station = planICLStation(i % 6)
				}
				qID := fmt.Sprintf("qu_planinv_%06d", i)
				qVals = append(qVals, "(?, '12', 'un_planinv', ?, ?)")
				qArgs = append(qArgs, qID, createdAt, createdAt)
				lVals = append(lVals, "(?, ?, ?, ?, ?, ?, ?, ?, ?)")
				lArgs = append(lArgs, planICLID(i), planICLItemID(planICLItem(i)), qID, action, station,
					planICLAccount, planICLUser(i), createdAt, createdAt)
			}
			exec(`INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES `+strings.Join(qVals, ","), qArgs...)
			exec(`INSERT INTO inventory_change_log (id, item_id, quantity_id, action_type_code, scanning_station_id,
			      account_id, responsible_user_id, created_at, updated_at) VALUES `+strings.Join(lVals, ","), lArgs...)
		}
		exec("ANALYZE TABLE inventory_change_log, item")
	})
}

// withPlanICLWindow applies the window ListInventoryChangeLogs gives a list that names no items or start.
func withPlanICLWindow(p domain.ListInventoryChangeLogsParams) domain.ListInventoryChangeLogsParams {
	if p.StartDate != nil || len(p.ItemIDs) > 0 {
		return p
	}
	end := planICLNow
	if p.EndDate != nil {
		end = *p.EndDate
	}
	start := end.Add(-90 * 24 * time.Hour)
	p.StartDate = &start
	return p
}

func inventoryChangeLogPlanDims() []planDim[domain.ListInventoryChangeLogsParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	old := planICLCreatedAt(planICLRows / 4)
	return []planDim[domain.ListInventoryChangeLogsParams]{
		{"item", []planValue[domain.ListInventoryChangeLogsParams]{
			{"large", func(p *domain.ListInventoryChangeLogsParams) { p.ItemIDs = []string{planICLItemID(0)} }},
			{"rare", func(p *domain.ListInventoryChangeLogsParams) { p.ItemIDs = []string{planICLItemID(planICLItems - 1)} }},
			{"large+rare", func(p *domain.ListInventoryChangeLogsParams) {
				p.ItemIDs = []string{planICLItemID(0), planICLItemID(planICLItems - 1)}
			}},
		}},
		{"action", []planValue[domain.ListInventoryChangeLogsParams]{
			{"scan", func(p *domain.ListInventoryChangeLogsParams) { p.ActionTypeCodes = []string{"scan"} }},
			{"delete_record", func(p *domain.ListInventoryChangeLogsParams) { p.ActionTypeCodes = []string{"delete_record"} }},
			{"correction+system", func(p *domain.ListInventoryChangeLogsParams) {
				p.ActionTypeCodes = []string{"user_correction", "system_action"}
			}},
		}},
		{"user", []planValue[domain.ListInventoryChangeLogsParams]{
			{"busy", func(p *domain.ListInventoryChangeLogsParams) { p.ChangedByUserIDs = []string{planICLUserID(0)} }},
			{"rare", func(p *domain.ListInventoryChangeLogsParams) {
				p.ChangedByUserIDs = []string{planICLUserID(planICLUsers - 1)}
			}},
			{"busy+rare", func(p *domain.ListInventoryChangeLogsParams) {
				p.ChangedByUserIDs = []string{planICLUserID(0), planICLUserID(planICLUsers - 1)}
			}},
		}},
		{"dates", []planValue[domain.ListInventoryChangeLogsParams]{
			{"old30d", func(p *domain.ListInventoryChangeLogsParams) {
				p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour))
			}},
			{"all", func(p *domain.ListInventoryChangeLogsParams) { p.StartDate = at(planICLOrigin) }},
		}},
		{"search", []planValue[domain.ListInventoryChangeLogsParams]{
			{"one", func(p *domain.ListInventoryChangeLogsParams) { p.Query = str(planICLRareSKU) }},
			{"every", func(p *domain.ListInventoryChangeLogsParams) { p.Query = str(planICLDenseSKU) }},
		}},
	}
}

func inventoryChangeLogPlanCases() []planCase[domain.ListInventoryChangeLogsParams] {
	mid := planICLCreatedAt(planICLRows * 9 / 10)
	cursor := func(dir pagination.Direction) func(*domain.ListInventoryChangeLogsParams) {
		return func(p *domain.ListInventoryChangeLogsParams) { p.Cursor = planCursorAt(mid, "inchlg_~", dir) }
	}
	return planCases(
		domain.ListInventoryChangeLogsParams{AccountID: planICLAccount, Limit: 25},
		inventoryChangeLogPlanDims(),
		[]planValue[domain.ListInventoryChangeLogsParams]{
			{"first", func(*domain.ListInventoryChangeLogsParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// planCursorAt is a keyset cursor at the given position; an id of "<prefix>~" sorts after every id
// with that prefix, so the page starts at the instant itself.
func planCursorAt(at time.Time, id string, dir pagination.Direction) *string {
	c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: at, ID: id, Direction: dir})
	return &c
}

// inventoryChangeLogSearchFloor is how many entries a SKU search matches, or 0 without one. A search
// resolves to an arbitrary list of items, which no key yields in list order; like a FULLTEXT match, the
// best a plan can do is read only the entries it matches.
func inventoryChangeLogSearchFloor(t *testing.T, db *sql.DB, p domain.ListInventoryChangeLogsParams) float64 {
	t.Helper()
	if p.Query == nil {
		return 0
	}
	p = withPlanICLWindow(p)
	where := []string{"icl.account_id = ?", "i.sku LIKE ?"}
	args := []any{p.AccountID, "%" + *p.Query + "%"}
	in := func(column string, values []string) {
		if len(values) > 0 {
			where = append(where, column+" IN ("+placeholders(len(values))+")")
			args = append(args, stringArgs(values)...)
		}
	}
	in("icl.item_id", p.ItemIDs)
	in("icl.action_type_code", p.ActionTypeCodes)
	in("icl.responsible_user_id", p.ChangedByUserIDs)
	if p.StartDate != nil {
		where, args = append(where, "icl.created_at >= ?"), append(args, *p.StartDate)
	}
	if p.EndDate != nil {
		where, args = append(where, "icl.created_at <= ?"), append(args, *p.EndDate)
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM inventory_change_log icl JOIN item i ON i.id = icl.item_id WHERE "+
		strings.Join(where, " AND "), args...).Scan(&n))
	return n
}

// TestInventoryChangeLogList_ReadsAboutAPage holds every filter combination ListInventoryChangeLogs
// accepts to reading about a page of change log (listPlanSuite).
func TestInventoryChangeLogList_ReadsAboutAPage(t *testing.T) {
	ensureInventoryChangeLogCorpus(t)
	listPlanSuite[domain.ListInventoryChangeLogsParams]{
		table: "inventory_change_log", scopeColumn: "account_id",
		from: "FROM inventory_change_log icl", alias: "icl",
		cases: inventoryChangeLogPlanCases(),
		limit: func(p domain.ListInventoryChangeLogsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListInventoryChangeLogsParams) error {
			if _, apiErr := NewInventoryChangeLogRepo(q).List(ctx, withPlanICLWindow(p)); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: inventoryChangeLogSearchFloor,
	}.run(t)
}
