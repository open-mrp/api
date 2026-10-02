package repository

import (
	"context"
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/pagination"
)

// The keys that yield an account's items in list order: one per item-column filter, and one for none.
const (
	itemCreatedIndex  = "item_account_created_idx"
	itemTypeIndex     = "item_account_type_created_idx"
	itemCategoryIndex = "item_account_category_created_idx"
)

const itemListColumns = `
	i.id, i.sku, i.description, i.notes, i.item_type_code, i.item_category_id,
	ic.name, ic.item_category_type_code, ic.unit_group_id,
	i.unit_value_id, i.unit_cost_id, i.burn_rate_id, i.account_id, i.is_dirty,
	i.created_at, i.updated_at, ic.created_at, ic.updated_at`

// catalogFilter is a catalog list's WHERE on item i, and the search ranking its page sorts by.
type catalogFilter struct {
	where []string
	args  []any
	// search is the active search, or the zero value; its SKU tier leads the sort.
	search db.CatalogSearch
}

func (f *catalogFilter) add(cond string, args ...any) {
	f.where = append(f.where, cond)
	f.args = append(f.args, args...)
}

func (f *catalogFilter) in(col string, values []string) {
	if len(values) > 0 {
		f.add(col+" IN ("+placeholders(len(values))+")", stringArgs(values)...)
	}
}

// page applies the cursor to a list ordered by (createdCol, idCol) and returns the ORDER BY. Rows are
// read best tier and newest first; a previous page reads the other way and
// pagination.BuildPageStringWithSearchRank restores the order.
func (f *catalogFilter) page(cur *pagination.StringCursor, createdCol, idCol string) (string, []any) {
	tier, tierArgs := catalogSearchTier(f.search)
	backward := cur != nil && cur.Direction == pagination.DirectionBackward
	if cur != nil {
		// past moves beyond the cursor in read order; a worse tier sorts after it.
		past, worseTier := "<", ">"
		if backward {
			past, worseTier = ">", "<"
		}
		keyset := "(" + createdCol + " " + past + " ? OR (" + createdCol + " = ? AND " + idCol + " " + past + " ?))"
		keysetArgs := []any{cur.OccurredAt, cur.OccurredAt, cur.ID}
		if cur.MatchTier == nil {
			f.add(keyset, keysetArgs...)
		} else {
			args := append(append(append(append([]any{}, tierArgs...), *cur.MatchTier), tierArgs...), *cur.MatchTier)
			f.add("("+tier+" "+worseTier+" ? OR ("+tier+" = ? AND "+keyset+"))", append(args, keysetArgs...)...)
		}
	}
	order := createdCol + " DESC, " + idCol + " DESC"
	tierOrder := " ASC, "
	if backward {
		order = createdCol + " ASC, " + idCol + " ASC"
		tierOrder = " DESC, "
	}
	if f.search.Contains.Valid {
		return tier + tierOrder + order, tierArgs
	}
	return order, nil
}

// itemListQuery is one ListItems page.
type itemListQuery struct {
	catalogFilter
	// categories and types report the item-column filters a list-order key can pin.
	categories, types bool
	// driven reports a filter on another table whose matches the page may be read from instead, by id.
	driven bool
}

// catalogSearchTier is the SQL for item i's search rank (0 exact SKU, 1 SKU token, 2 SKU prefix,
// 3 other match), mirroring db.CatalogSearchRank, or 0 for every item when there is no search.
func catalogSearchTier(search db.CatalogSearch) (string, []any) {
	if !search.Contains.Valid {
		return "0", nil
	}
	exact := search.Exact.String
	return `(CASE
		WHEN i.sku COLLATE utf8mb4_general_ci = ? THEN 0
		WHEN i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', ?, ' %')
			OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT(?, ' %')
			OR i.sku COLLATE utf8mb4_general_ci LIKE CONCAT('% ', ?) THEN 1
		WHEN i.sku COLLATE utf8mb4_general_ci LIKE ? THEN 2
		ELSE 3
	END)`, []any{exact, exact, exact, exact, search.Prefix.String}
}

// newItemListQuery applies every filter but the page's cursor. lineIDs are the product lines the
// items' products must be on, already narrowed to the customers' lines when customers are filtered.
func newItemListQuery(params domain.ListItemsParams, lineIDs []string) *itemListQuery {
	q := &itemListQuery{catalogFilter: catalogFilter{search: db.NewCatalogSearch(params.Query)}}
	q.add("i.account_id = ?", params.AccountID)
	q.add("i.deleted_at IS NULL")
	// The page holds only items the list hydrates, which inner-joins the category: one whose category was
	// deleted would take a page slot and then be dropped, leaving the page short.
	q.add("EXISTS (SELECT 1 FROM item_category pic WHERE pic.id = i.item_category_id)")
	q.in("i.item_type_code", params.Types)
	q.in("i.item_category_id", params.CategoryIDs)
	q.types, q.categories = len(params.Types) > 0, len(params.CategoryIDs) > 0
	if params.StartDate != nil {
		q.add("i.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		q.add("i.created_at <= ?", *params.EndDate)
	}
	// An item whose product is not for sale is not listed. A scalar probe of the unique item key, where
	// NOT EXISTS would let the planner materialize every product to anti-join against.
	q.add("COALESCE((SELECT p.product_type_code FROM product p WHERE p.item_id = i.id), 'sale') = 'sale'")

	if q.search.Contains.Valid {
		if params.IsExactMatch {
			q.add("(i.sku COLLATE utf8mb4_general_ci = ? OR i.description LIKE ?)", q.search.Exact.String, q.search.Contains.String)
		} else {
			q.add("(i.sku LIKE ? OR i.description LIKE ?)", q.search.Contains.String, q.search.Contains.String)
		}
	}
	// Filters on other tables are IN subqueries so the planner may drive from a rare value's matches;
	// a common one it walks list order and probes.
	if len(params.AttributeIDs) > 0 {
		q.driven = true
		q.add("i.id IN (SELECT ia.B FROM _item_attributes ia WHERE ia.A IN ("+placeholders(len(params.AttributeIDs))+"))",
			stringArgs(params.AttributeIDs)...)
	}
	if params.SupplierID != nil {
		q.driven = true
		q.add(`i.id IN (SELECT m.item_id FROM supplier_material sm JOIN material m ON m.id = sm.material_id
			WHERE sm.supplier_account_id = ? AND sm.owner_account_id = ?)`, *params.SupplierID, params.AccountID)
	}
	if lineIDs != nil {
		q.driven = true
		// product is not tenant-scoped; read from, it is ranged by line, never scanned.
		q.add("i.id IN (SELECT pl.item_id FROM product pl FORCE INDEX (product_product_line_id_idx, product_item_id_key)"+
			" WHERE pl.product_line_id IN ("+placeholders(len(lineIDs))+"))", stringArgs(lineIDs)...)
	}
	if params.OnlyInitialSubassemblies {
		// production is not tenant-scoped, so this stays a per-item probe rather than a subquery to drive from.
		q.add(`EXISTS (SELECT 1 FROM production prd WHERE prd.item_id = i.id AND prd.production_step_id IS NOT NULL
			AND NOT EXISTS (SELECT 1 FROM _parent_child_production_steps pcps WHERE pcps.A = prd.production_step_id))`)
	}
	return q
}

// indexes are the keys item may be read through: the list-order keys that pin an item-column filter.
// Offered every list-order key, the planner walks created_at past every item a category rejects; left
// to itself it reads the SKU key and sorts the account. A filter on another table adds the primary key,
// so a rare value's matches are read by id and sorted rather than found by walking the account.
func (q *itemListQuery) indexes() []string {
	var keys []string
	if q.categories {
		keys = append(keys, itemCategoryIndex)
	}
	if q.types {
		keys = append(keys, itemTypeIndex)
	}
	if keys == nil {
		keys = []string{itemCreatedIndex}
	}
	if q.driven {
		keys = append(keys, "PRIMARY")
	}
	return keys
}

// list reads one page. The page is chosen from item alone and joined after, so a filter no key serves
// in list order reads its matches and not their categories too.
func (r *itemRepoImpl) list(ctx context.Context, q *itemListQuery, orderBy string, orderArgs []any, limit int32) ([]*domain.Item, error) {
	var sb strings.Builder
	sb.WriteString("SELECT")
	sb.WriteString(itemListColumns)
	sb.WriteString("\nFROM (SELECT i.id FROM item i FORCE INDEX (" + strings.Join(q.indexes(), ", ") + ")")
	sb.WriteString("\nWHERE ")
	sb.WriteString(strings.Join(q.where, "\nAND "))
	sb.WriteString("\nORDER BY " + orderBy + "\nLIMIT ?")
	sb.WriteString(") page\nJOIN item i ON i.id = page.id\nJOIN item_category ic ON ic.id = i.item_category_id")
	sb.WriteString("\nORDER BY " + orderBy)

	args := append(append(append(append([]any{}, q.args...), orderArgs...), limit), orderArgs...)
	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := []*domain.Item{}
	for rows.Next() {
		var (
			id, sku, itemTypeCode, itemCategoryID, categoryName, categoryTypeCode, categoryUnitGroupID string
			unitValueID, unitCostID, burnRateID, accountID                                             string
			description, notes                                                                         gosql.NullString
			isDirty                                                                                    bool
			createdAt, updatedAt, categoryCreatedAt, categoryUpdatedAt                                 gosql.NullTime
		)
		if err := rows.Scan(&id, &sku, &description, &notes, &itemTypeCode, &itemCategoryID,
			&categoryName, &categoryTypeCode, &categoryUnitGroupID,
			&unitValueID, &unitCostID, &burnRateID, &accountID, &isDirty,
			&createdAt, &updatedAt, &categoryCreatedAt, &categoryUpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, mapItemBaseRow(id, sku, description, notes, itemTypeCode, itemCategoryID, categoryName,
			categoryTypeCode, categoryUnitGroupID, unitValueID, unitCostID, burnRateID, accountID, isDirty,
			createdAt.Time, updatedAt.Time, categoryCreatedAt.Time, categoryUpdatedAt.Time))
	}
	return items, rows.Err()
}

// customerProductLines is every product line any of customerIDs may buy from: granted to the customer
// directly, to its group, or to one of its price groups.
func customerProductLines(ctx context.Context, queries sqlc.DBTX, accountID string, customerIDs []string) ([]string, error) {
	in := placeholders(len(customerIDs))
	relation := "ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer' AND ar.counterparty_account_id IN (" + in + ")"
	args := append([]any{accountID}, stringArgs(customerIDs)...)
	rows, err := queries.QueryContext(ctx, `
SELECT arpl.product_line_id FROM account_relation ar
JOIN account_relation_product_line arpl ON arpl.account_relation_id = ar.id
WHERE `+relation+`
UNION
SELECT agpl.product_line_id FROM account_relation ar
JOIN account_group_product_line agpl ON agpl.account_group_id = ar.account_group_id
WHERE `+relation+`
UNION
SELECT agpl.product_line_id FROM account_relation ar
JOIN account_relation_price_group arpg ON arpg.account_relation_id = ar.id
JOIN account_group_product_line agpl ON agpl.account_group_id = arpg.account_group_id
WHERE `+relation, append(append(args, args...), args...)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	lines := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		lines = append(lines, id)
	}
	return lines, rows.Err()
}

// itemListLines is the product lines a list's items must be on, or nil when it filters by neither
// product line nor customer. Empty means no item can match.
func itemListLines(ctx context.Context, queries sqlc.DBTX, params domain.ListItemsParams) ([]string, error) {
	if len(params.CustomerIDs) == 0 {
		if len(params.ProductLineIDs) == 0 {
			return nil, nil
		}
		return params.ProductLineIDs, nil
	}
	lines, err := customerProductLines(ctx, queries, params.AccountID, params.CustomerIDs)
	if err != nil || len(params.ProductLineIDs) == 0 {
		return lines, err
	}
	both := []string{}
	for _, l := range lines {
		for _, want := range params.ProductLineIDs {
			if l == want {
				both = append(both, l)
				break
			}
		}
	}
	return both, nil
}
