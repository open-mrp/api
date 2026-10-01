//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

func itemPlanDims() []planDim[domain.ListItemsParams] {
	type P = domain.ListItemsParams
	str := func(s string) *string { return &s }
	recent := planCatItemCreatedAt(planCatItems - 1)
	old := planCatItemCreatedAt(planCatItems / 4)
	return []planDim[P]{
		{"type", []planValue[P]{
			{"type=part", func(p *P) { p.Types = []string{"part"} }},
			{"type=material", func(p *P) { p.Types = []string{"material"} }},
		}},
		{"category", []planValue[P]{
			{"cate=dense", func(p *P) { p.CategoryIDs = []string{planCatCategoryID(10)} }},
			{"cate=rare", func(p *P) { p.CategoryIDs = []string{planCatCategoryID(planCatCategories - 1)} }},
		}},
		{"attribute", []planValue[P]{
			{"attr=dense", func(p *P) { p.AttributeIDs = []string{planCatAttributeID(0)} }},
			{"attr=rare", func(p *P) { p.AttributeIDs = []string{planCatAttributeID(planCatAttributes - 1)} }},
		}},
		{"supplier", []planValue[P]{
			{"supp=dense", func(p *P) { p.SupplierID = str(planCatSupplierID(0)) }},
			{"supp=rare", func(p *P) { p.SupplierID = str(planCatSupplierID(planCatSuppliers - 1)) }},
		}},
		{"created", []planValue[P]{
			{"crea=last30d", func(p *P) { at := recent.AddDate(0, 0, -30); p.StartDate = &at }},
			{"crea=old30d", func(p *P) { s, e := old, old.AddDate(0, 0, 30); p.StartDate, p.EndDate = &s, &e }},
		}},
		{"search", []planValue[P]{
			{"sear=one", func(p *P) { p.Query = str(planCatRareSearch) }},
			{"sear=one-exact", func(p *P) { p.Query, p.IsExactMatch = str(planCatRareSearch), true }},
			{"sear=every", func(p *P) { p.Query = str(planCatDenseSearch) }},
		}},
		{"subassembly", []planValue[P]{
			{"suba=initial", func(p *P) { p.OnlyInitialSubassemblies = true }},
		}},
		{"line", []planValue[P]{
			{"line=dense", func(p *P) { p.ProductLineIDs = []string{planCatLineID(0)} }},
			{"line=rare", func(p *P) { p.ProductLineIDs = []string{planCatLineID(planCatLines - 1)} }},
		}},
		{"customer", []planValue[P]{
			{"cust=large", func(p *P) { p.CustomerIDs = []string{planCatCustomerID(0)} }},
			{"cust=rare", func(p *P) { p.CustomerIDs = []string{planCatCustomerID(planCatCustomers - 1)} }},
		}},
	}
}

func itemPlanCases() []planCase[domain.ListItemsParams] {
	type P = domain.ListItemsParams
	mid := planCatItemCreatedAt(planCatItems / 2)
	cases := planCases(P{AccountID: planCatAccount, Limit: 25}, itemPlanDims(), []planValue[P]{
		{"first", func(*P) {}},
		{"deep-next", func(*P) {}},
		{"deep-prev", func(*P) {}},
	})
	// The cursor carries the search tier, so it is set once the filters are.
	for i := range cases {
		dir := pagination.DirectionForward
		switch {
		case strings.HasSuffix(cases[i].name, "/first"):
			continue
		case strings.HasSuffix(cases[i].name, "/deep-prev"):
			dir = pagination.DirectionBackward
		}
		c := pagination.StringCursor{OccurredAt: mid, ID: "it_plancat_~", Direction: dir}
		if cases[i].params.Query != nil {
			tier := 3
			c.MatchTier = &tier
		}
		cur := pagination.EncodeStringCursor(c)
		cases[i].params.Cursor = &cur
	}
	return cases
}

// itemPlanFloor is how many items a request must read when some filter cannot be served in list order.
// A substring search examines every item the item-column filters leave (no B-tree finds a substring);
// otherwise a filter on another table (attribute, supplier, product line, customer, subassembly) is
// held to the items it matches.
func itemPlanFloor(t *testing.T, sqlDB *sql.DB, p domain.ListItemsParams) float64 {
	t.Helper()
	where := []string{"i.account_id = ?", "i.deleted_at IS NULL"}
	args := []any{p.AccountID}
	in := func(col string, vals []string) {
		if len(vals) == 0 {
			return
		}
		where = append(where, col+" IN ("+placeholders(len(vals))+")")
		args = append(args, stringArgs(vals)...)
	}
	in("i.item_type_code", p.Types)
	in("i.item_category_id", p.CategoryIDs)
	if p.StartDate != nil {
		where, args = append(where, "i.created_at >= ?"), append(args, *p.StartDate)
	}
	if p.EndDate != nil {
		where, args = append(where, "i.created_at <= ?"), append(args, *p.EndDate)
	}
	if db.NewCatalogSearch(p.Query).Contains.Valid {
		return planCount(t, sqlDB, "SELECT COUNT(*) FROM item i WHERE "+strings.Join(where, " AND "), args...)
	}
	joined := false
	if len(p.AttributeIDs) > 0 {
		joined = true
		where = append(where, "EXISTS (SELECT 1 FROM _item_attributes ia WHERE ia.B = i.id AND ia.A IN ("+placeholders(len(p.AttributeIDs))+"))")
		args = append(args, stringArgs(p.AttributeIDs)...)
	}
	if p.SupplierID != nil {
		joined = true
		where = append(where, `EXISTS (SELECT 1 FROM material m JOIN supplier_material sm ON sm.material_id = m.id
			WHERE m.item_id = i.id AND sm.supplier_account_id = ? AND sm.owner_account_id = i.account_id)`)
		args = append(args, *p.SupplierID)
	}
	lines := p.ProductLineIDs
	if len(p.CustomerIDs) > 0 {
		joined = true
		lines = append(lines, planCustomerLines(t, sqlDB, p.AccountID, p.CustomerIDs)...)
		if len(lines) == 0 {
			return 0
		}
	}
	if len(lines) > 0 {
		joined = true
		where = append(where, "EXISTS (SELECT 1 FROM product p WHERE p.item_id = i.id AND p.product_line_id IN ("+placeholders(len(lines))+"))")
		args = append(args, stringArgs(lines)...)
	}
	// The subassembly filter is a per-item probe, never read from: the bar is the other filters' matches.
	if p.OnlyInitialSubassemblies && !joined {
		joined = true
		where = append(where, `EXISTS (SELECT 1 FROM production prd WHERE prd.item_id = i.id AND prd.production_step_id IS NOT NULL
			AND NOT EXISTS (SELECT 1 FROM _parent_child_production_steps pcps WHERE pcps.A = prd.production_step_id))`)
	}
	if !joined {
		return 0
	}
	return planCount(t, sqlDB, "SELECT COUNT(*) FROM item i WHERE "+strings.Join(where, " AND "), args...)
}

func planCount(t *testing.T, sqlDB *sql.DB, query string, args ...any) float64 {
	t.Helper()
	var n float64
	require.NoError(t, sqlDB.QueryRow(query, args...).Scan(&n))
	return n
}

func itemPlanSupplied(t *testing.T, sqlDB *sql.DB, p domain.ListItemsParams) float64 {
	return planCount(t, sqlDB, "SELECT COUNT(*) FROM supplier_material WHERE supplier_account_id = ?", *p.SupplierID)
}

// planCustomerLines is every product line any of customerIDs may buy from.
func planCustomerLines(t *testing.T, sqlDB *sql.DB, accountID string, customerIDs []string) []string {
	t.Helper()
	if len(customerIDs) == 0 {
		return nil
	}
	args := append([]any{accountID}, stringArgs(customerIDs)...)
	rows, err := sqlDB.Query(`
SELECT arpl.product_line_id FROM account_relation ar JOIN account_relation_product_line arpl ON arpl.account_relation_id = ar.id
WHERE ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer' AND ar.counterparty_account_id IN (`+placeholders(len(customerIDs))+`)
UNION
SELECT agpl.product_line_id FROM account_relation ar JOIN account_group_product_line agpl ON agpl.account_group_id = ar.account_group_id
WHERE ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer' AND ar.counterparty_account_id IN (`+placeholders(len(customerIDs))+`)
UNION
SELECT agpl.product_line_id FROM account_relation ar JOIN account_relation_price_group arpg ON arpg.account_relation_id = ar.id
JOIN account_group_product_line agpl ON agpl.account_group_id = arpg.account_group_id
WHERE ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer' AND ar.counterparty_account_id IN (`+placeholders(len(customerIDs))+`)`,
		append(append(args, args...), args...)...)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		out = append(out, id)
	}
	require.NoError(t, rows.Err())
	return out
}

// TestItemList_ReadsAboutAPage holds every filter combination ListItems accepts to reading about a page
// of items (listPlanSuite), and each filter's other tables to probing only the items it reads.
func TestItemList_ReadsAboutAPage(t *testing.T) {
	ensureCatalogCorpus(t)
	listPlanSuite[domain.ListItemsParams]{
		table: "item", scopeColumn: "account_id",
		from: "FROM item i", alias: "i",
		cases: itemPlanCases(),
		limit: func(p domain.ListItemsParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListItemsParams) error {
			if _, apiErr := NewItemRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor:       itemPlanFloor,
		statsTables: []string{"product", "_item_attributes"},
		reads: []planRead[domain.ListItemsParams]{
			{alias: "p", fanout: 1}, {alias: "prd", fanout: 1}, {alias: "pcps", fanout: 1},
			{alias: "ia", fanout: 1, matches: func(t *testing.T, sqlDB *sql.DB, p domain.ListItemsParams) float64 {
				return planCount(t, sqlDB, "SELECT COUNT(*) FROM _item_attributes WHERE A IN ("+placeholders(len(p.AttributeIDs))+")", stringArgs(p.AttributeIDs)...)
			}},
			{alias: "sm", fanout: 1, matches: itemPlanSupplied},
			{alias: "m", fanout: 1, matches: itemPlanSupplied},
			{alias: "pl", fanout: 1, matches: func(t *testing.T, sqlDB *sql.DB, p domain.ListItemsParams) float64 {
				lines := append(append([]string{}, p.ProductLineIDs...), planCustomerLines(t, sqlDB, p.AccountID, p.CustomerIDs)...)
				return planCount(t, sqlDB, "SELECT COUNT(*) FROM product WHERE product_line_id IN ("+placeholders(len(lines))+")", stringArgs(lines)...)
			}},
		},
	}.run(t)
}
