//go:build plans

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func salesLinePlanDims() []planDim[domain.ListSalesLinesParams] {
	type P = domain.ListSalesLinesParams
	recent := planSalesInvoicedAt(planSalesOrders - 1)
	old := planSalesInvoicedAt(planSalesOrders / 4)
	window := func(from, to time.Time) func(*P) {
		return func(p *P) { p.HasWindow, p.StartsAt, p.EndsAt = true, from, to }
	}
	return []planDim[P]{
		{"window", []planValue[P]{
			{"last30d", window(recent.Add(-30*24*time.Hour), recent)},
			{"old30d", window(old, old.Add(30*24*time.Hour))},
		}},
		{"customer", []planValue[P]{
			{"large", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(0)} }},
			{"parent", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(planSalesParentCustomer)} }},
			{"rare", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(planSalesRareCustomer)} }},
		}},
		{"group", []planValue[P]{
			{"group-big", func(p *P) { p.CustomerGroupIDs = []string{planSalesGroupID("big")} }},
			{"group-rare", func(p *P) { p.CustomerGroupIDs = []string{planSalesGroupID("rare")} }},
		}},
		{"rep", []planValue[P]{
			{"rep-dense", func(p *P) { p.SalesRepIDs = []string{planSalesRepID(0)} }},
			{"rep-rare", func(p *P) { p.SalesRepIDs = []string{planSalesRepID(planSalesRareRep)} }},
		}},
		{"item", []planValue[P]{
			{"item-dense", func(p *P) { p.ItemIDs = []string{planSalesItemID(planSalesDenseItem)} }},
			{"item-rare", func(p *P) { p.ItemIDs = []string{planSalesItemID(planSalesRareItem)} }},
		}},
		{"line", []planValue[P]{
			{"line-dense", func(p *P) { p.ProductLineIDs = []string{planSalesProductLineID(planSalesDenseLine)} }},
			{"line-rare", func(p *P) { p.ProductLineIDs = []string{planSalesProductLineID(planSalesRareLine)} }},
		}},
	}
}

// TestSalesLineList_ReadsAboutAPage holds every filter combination ListSalesLines accepts to reading
// about a page of sales line facts (listPlanSuite).
func TestSalesLineList_ReadsAboutAPage(t *testing.T) {
	ensureSalesCorpus(t)
	type P = domain.ListSalesLinesParams
	mid := planSalesInvoicedAt(planSalesOrders / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) { p.Cursor = planCursorAt(mid, "inl_plansales_~", dir) }
	}
	base := P{Limit: 25}
	base.AccountID = planSalesAccount
	listPlanSuite[P]{
		table: "sales_line_fact", scopeColumn: "account_id",
		from: "FROM sales_line_fact f", alias: "f",
		cases: planCases(base, salesLinePlanDims(), []planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		}),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewSalesReportRepo(q).GetLinePage(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
	}.run(t)
}
