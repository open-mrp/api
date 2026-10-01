//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func productPlanDims() []planDim[domain.ListProductsFullParams] {
	type P = domain.ListProductsFullParams
	str := func(s string) *string { return &s }
	yes, no := true, false
	recent := planCatItemCreatedAt(planCatItems - 1)
	old := planCatItemCreatedAt(planCatItems / 4)
	return []planDim[P]{
		{"search", []planValue[P]{
			{"search=one", func(p *P) { p.Query = str(planCatSKU(1234)) }},
			{"search=every", func(p *P) { p.Query = str(planCatDenseSearch) }},
		}},
		{"line", []planValue[P]{
			{"line=dense", func(p *P) { p.ProductLineIDs = []string{planCatLineID(0)} }},
			{"line=rare", func(p *P) { p.ProductLineIDs = []string{planCatLineID(planCatLines - 1)} }},
		}},
		{"customer", []planValue[P]{
			{"cust=large", func(p *P) { p.CustomerIDs = []string{planCatCustomerID(0)} }},
			{"cust=rare", func(p *P) { p.CustomerIDs = []string{planCatCustomerID(planCatCustomers - 1)} }},
		}},
		{"category", []planValue[P]{
			{"cat=dense", func(p *P) { p.CategoryIDs = []string{planCatCategoryID(0)} }},
			{"cat=rare", func(p *P) { p.CategoryIDs = []string{planCatCategoryID(planCatCategories - 1)} }},
		}},
		{"attribute", []planValue[P]{
			{"attr=dense", func(p *P) { p.AttributeIDs = []string{planCatAttributeID(0)} }},
			{"attr=rare", func(p *P) { p.AttributeIDs = []string{planCatAttributeID(planCatAttributes - 1)} }},
		}},
		{"portal", []planValue[P]{
			{"portal=yes", func(p *P) { p.IsPortalReady = &yes }},
			{"portal=no", func(p *P) { p.IsPortalReady = &no }},
		}},
		{"created", []planValue[P]{
			{"created=last30d", func(p *P) { at := recent.AddDate(0, 0, -30); p.StartDate = &at }},
			{"created=old30d", func(p *P) { s, e := old, old.AddDate(0, 0, 30); p.StartDate, p.EndDate = &s, &e }},
		}},
	}
}

func productPlanCases() []planCase[domain.ListProductsFullParams] {
	type P = domain.ListProductsFullParams
	mid := planCatItemCreatedAt(planCatItems / 2)
	cases := planCases(P{AccountID: planCatAccount, Limit: 25}, productPlanDims(), []planValue[P]{
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
		c := pagination.StringCursor{OccurredAt: mid, ID: "pr_plancat_~", Direction: dir}
		if cases[i].params.Query != nil {
			tier := 3
			c.MatchTier = &tier
		}
		s := pagination.EncodeStringCursor(c)
		cases[i].params.Cursor = &s
	}
	return cases
}

// productPlanFloor is how many products a request matches before its search. Products carry no
// account, so no key yields one account's products in list order: the bar for every request is reading
// only the products it may return (and, searching, every one of those the substring must examine).
func productPlanFloor(t *testing.T, sqlDB *sql.DB, p domain.ListProductsFullParams) float64 {
	t.Helper()
	where := []string{"i.account_id = ?", "i.deleted_at IS NULL", "p.product_type_code = 'sale'"}
	args := []any{p.AccountID}
	in := func(col string, vals []string) {
		if len(vals) > 0 {
			where = append(where, col+" IN ("+placeholders(len(vals))+")")
			args = append(args, stringArgs(vals)...)
		}
	}
	in("i.item_category_id", p.CategoryIDs)
	in("p.product_line_id", p.ProductLineIDs)
	if len(p.CustomerIDs) > 0 {
		lines := planCustomerLines(t, sqlDB, p.AccountID, p.CustomerIDs)
		if len(lines) == 0 {
			return 0
		}
		in("p.product_line_id", lines)
	}
	if len(p.AttributeIDs) > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM _item_attributes ia WHERE ia.B = i.id AND ia.A IN ("+placeholders(len(p.AttributeIDs))+"))")
		args = append(args, stringArgs(p.AttributeIDs)...)
	}
	if p.IsPortalReady != nil {
		where, args = append(where, "p.is_portal_ready = ?"), append(args, *p.IsPortalReady)
	}
	if p.StartDate != nil {
		where, args = append(where, "i.created_at >= ?"), append(args, *p.StartDate)
	}
	if p.EndDate != nil {
		where, args = append(where, "i.created_at <= ?"), append(args, *p.EndDate)
	}
	return planCount(t, sqlDB, "SELECT COUNT(*) FROM product p JOIN item i ON i.id = p.item_id WHERE "+strings.Join(where, " AND "), args...)
}

// TestProductList_ReadsAboutAPage holds every filter combination ListProductsFull accepts to reading
// only the products it may return (listPlanSuite, joinScoped).
func TestProductList_ReadsAboutAPage(t *testing.T) {
	ensureCatalogCorpus(t)
	listPlanSuite[domain.ListProductsFullParams]{
		table: "product", joinScoped: true,
		from: "FROM product p", alias: "p",
		cases: productPlanCases(),
		limit: func(p domain.ListProductsFullParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListProductsFullParams) error {
			if _, apiErr := NewProductRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor:       productPlanFloor,
		statsTables: []string{"item", "_item_attributes"},
		reads: []planRead[domain.ListProductsFullParams]{
			{alias: "i", fanout: 1}, {alias: "ia", fanout: 1},
		},
	}.run(t)
}
