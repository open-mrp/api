package repository

import (
	"context"
	gosql "database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
)

const productListColumns = `
	p.id, p.product_type_code, p.is_portal_ready, p.product_line_id, p.item_id, p.created_at, p.updated_at,
	i.sku, i.description, i.notes, i.item_type_code, i.item_category_id, i.unit_value_id, i.unit_cost_id,
	i.burn_rate_id, i.account_id, i.is_dirty, i.created_at, i.updated_at,
	ic.name, ic.item_category_type_code, ic.unit_group_id, ic.created_at, ic.updated_at,
	pt.id, pt.name, pt.code, pt.created_at, pt.updated_at`

// productListQuery is one ListProductsFull page: sale products p of items i in the account.
type productListQuery struct {
	catalogFilter
	// categories and dates report the item-column filters the item keys pin or range.
	categories, dates bool
	// lines and portal report the product-column filters a product key can pin.
	lines, portal bool
	// attributes reports an attribute filter, whose matches the page may be read from by item id.
	attributes bool
}

// newProductListQuery applies every filter but the page's cursor. lineIDs are the product lines the
// products must be on, or nil for any.
func newProductListQuery(params domain.ListProductsFullParams, lineIDs []string) *productListQuery {
	q := &productListQuery{catalogFilter: catalogFilter{search: db.NewCatalogSearch(params.Query)}}
	q.add("i.account_id = ?", params.AccountID)
	q.add("i.deleted_at IS NULL")
	// Every product's item is a product item, so this narrows the read to them on the type key.
	q.add("i.item_type_code = 'product'")
	q.add("p.product_type_code = 'sale'")
	q.in("i.item_category_id", params.CategoryIDs)
	q.categories = len(params.CategoryIDs) > 0
	q.dates = params.StartDate != nil || params.EndDate != nil
	if params.StartDate != nil {
		q.add("i.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		q.add("i.created_at <= ?", *params.EndDate)
	}
	if params.IsPortalReady != nil {
		q.portal = true
		q.add("p.is_portal_ready = ?", *params.IsPortalReady)
	}
	if q.search.Contains.Valid {
		q.add("(i.sku LIKE ? OR i.description LIKE ?)", q.search.Contains.String, q.search.Contains.String)
	}
	if lineIDs != nil {
		q.lines = true
		q.in("p.product_line_id", lineIDs)
	}
	if len(params.AttributeIDs) > 0 {
		q.attributes = true
		q.add("i.id IN (SELECT ia.B FROM _item_attributes ia WHERE ia.A IN ("+placeholders(len(params.AttributeIDs))+"))",
			stringArgs(params.AttributeIDs)...)
	}
	return q
}

// The product keys a list may drive from: a line's products (portal readiness pinned with it), and
// portal readiness alone. Lines belong to one account; readiness spans every tenant, but its rare
// value is far narrower than the account's product items.
const (
	productLinePortalIndex = "product_line_portal_created_idx"
	productPortalIndex     = "product_portal_created_idx"
)

// itemIndexes are the keys item may be read through. Products carry no account, so none yields an
// account's products in list order: the type key reads the account's product items, the category key
// a category's. A filter the page can be read from elsewhere adds the primary key, so the planner
// can drive from that filter's matches and look each item up by id.
func (q *productListQuery) itemIndexes() []string {
	keys := []string{itemTypeIndex}
	if q.categories {
		keys = []string{itemCategoryIndex}
	}
	if q.lines || q.portal || q.attributes {
		keys = append(keys, "PRIMARY")
	}
	return keys
}

// productIndexes are the keys product may be read through: by item, always, and by a product-column
// filter's key when one is set. product holds every tenant's rows, so it is never scanned.
func (q *productListQuery) productIndexes() []string {
	keys := []string{"product_item_id_key"}
	if q.lines {
		keys = append(keys, productLinePortalIndex)
	}
	if q.portal {
		keys = append(keys, productPortalIndex)
	}
	return keys
}

// list reads one page. Every product the filters leave is sorted, so the page is chosen from item and
// product keys alone and the rows joined after.
func (r *productRepoImpl) list(ctx context.Context, q *productListQuery, orderBy string, orderArgs []any, limit int32) ([]*domain.ProductFull, error) {
	var sb strings.Builder
	sb.WriteString("SELECT")
	sb.WriteString(productListColumns)
	if q.lines && !q.categories && !q.dates {
		// The lines' products are a subset of the account's product items, so with no category or dates
		// for the item keys to narrow by they lead; left to itself the planner walks every product item
		// to find a few lines' worth.
		sb.WriteString("\nFROM (SELECT p.id FROM product p FORCE INDEX (" + productLinePortalIndex + ")")
		sb.WriteString("\nSTRAIGHT_JOIN item i FORCE INDEX (PRIMARY) ON i.id = p.item_id")
	} else {
		sb.WriteString("\nFROM (SELECT p.id FROM item i FORCE INDEX (" + strings.Join(q.itemIndexes(), ", ") + ")")
		sb.WriteString("\nJOIN product p FORCE INDEX (" + strings.Join(q.productIndexes(), ", ") + ") ON p.item_id = i.id")
	}
	sb.WriteString("\nWHERE ")
	sb.WriteString(strings.Join(q.where, "\nAND "))
	sb.WriteString("\nORDER BY " + orderBy + "\nLIMIT ?")
	sb.WriteString(") page\nJOIN product p ON p.id = page.id\nJOIN item i ON i.id = p.item_id")
	sb.WriteString("\nJOIN item_category ic ON ic.id = i.item_category_id\nJOIN product_type pt ON pt.code = p.product_type_code")
	sb.WriteString("\nORDER BY " + orderBy)

	args := append(append(append(append([]any{}, q.args...), orderArgs...), limit), orderArgs...)
	rows, err := r.queries.DB().QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	products := []*domain.ProductFull{}
	for rows.Next() {
		var (
			id, productTypeCode, itemID, sku, itemTypeCode, itemCategoryID string
			unitValueID, unitCostID, burnRateID, accountID                 string
			categoryName, categoryTypeCode, categoryUnitGroupID            string
			ptID, ptName, ptCode                                           string
			productLineID, description, notes                              gosql.NullString
			isPortalReady, isDirty                                         bool
			createdAt, updatedAt, itemCreatedAt, itemUpdatedAt             time.Time
			categoryCreatedAt, categoryUpdatedAt, ptCreatedAt, ptUpdatedAt time.Time
		)
		if err := rows.Scan(&id, &productTypeCode, &isPortalReady, &productLineID, &itemID, &createdAt, &updatedAt,
			&sku, &description, &notes, &itemTypeCode, &itemCategoryID, &unitValueID, &unitCostID,
			&burnRateID, &accountID, &isDirty, &itemCreatedAt, &itemUpdatedAt,
			&categoryName, &categoryTypeCode, &categoryUnitGroupID, &categoryCreatedAt, &categoryUpdatedAt,
			&ptID, &ptName, &ptCode, &ptCreatedAt, &ptUpdatedAt); err != nil {
			return nil, err
		}
		var line *string
		if productLineID.Valid {
			line = &productLineID.String
		}
		products = append(products, &domain.ProductFull{
			ID:              id,
			ProductTypeCode: productTypeCode,
			IsPortalReady:   isPortalReady,
			ProductLineID:   line,
			ItemID:          itemID,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
			Item: mapProductBaseItem(itemID, sku, description, notes, itemTypeCode, itemCategoryID, categoryName, categoryTypeCode,
				categoryUnitGroupID, unitValueID, unitCostID, burnRateID, accountID, isDirty, itemCreatedAt, itemUpdatedAt,
				categoryCreatedAt, categoryUpdatedAt),
			ProductType: mapProductBaseProductType(ptID, ptName, ptCode, ptCreatedAt, ptUpdatedAt),
		})
	}
	return products, rows.Err()
}

// productListLines is the product lines a list's products must be on, or nil when it filters by
// neither product line nor customer. A product on a requested line or a line a customer may buy from
// is listed. Empty means no product can match.
func productListLines(ctx context.Context, queries sqlc.DBTX, params domain.ListProductsFullParams) ([]string, error) {
	if len(params.CustomerIDs) == 0 {
		if len(params.ProductLineIDs) == 0 {
			return nil, nil
		}
		return params.ProductLineIDs, nil
	}
	lines, err := customerProductLines(ctx, queries, params.AccountID, params.CustomerIDs)
	if err != nil {
		return nil, err
	}
	return append(lines, params.ProductLineIDs...), nil
}
