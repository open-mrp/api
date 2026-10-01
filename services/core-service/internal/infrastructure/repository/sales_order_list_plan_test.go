//go:build plans

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

func salesOrderPlanDims() []planDim[domain.ListSalesOrdersParams] {
	type P = domain.ListSalesOrdersParams
	str := func(s string) *string { return &s }
	day := func(d time.Time) *string { return str(d.Format("2006-01-02")) }
	yes, no := true, false
	recent := planSalesCreatedAt(planSalesOrders - 1)
	old := planSalesCreatedAt(planSalesOrders / 4)
	return []planDim[P]{
		{"status", []planValue[P]{
			{"fulfilled", func(p *P) { p.StatusCodes = []string{"fulfilled"} }},
			{"issued", func(p *P) { p.StatusCodes = []string{"issued"} }},
		}},
		{"customer", []planValue[P]{
			{"large", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(0)} }},
			{"mid", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(planSalesMidCustomer)} }},
			{"rare", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(planSalesRareCustomer)} }},
		}},
		{"buyer", []planValue[P]{
			{"buyer-large", func(p *P) { p.BuyerAccountID = str(planSalesCustomerID(0)) }},
			{"buyer-rare", func(p *P) { p.BuyerAccountID = str(planSalesCustomerID(planSalesRareCustomer)) }},
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
		{"created", []planValue[P]{
			{"created-last30d", func(p *P) { p.StartDate = day(recent.Add(-30 * 24 * time.Hour)) }},
			{"created-old30d", func(p *P) { p.StartDate, p.EndDate = day(old), day(old.Add(30*24*time.Hour)) }},
		}},
		{"shipby", []planValue[P]{
			{"shipby-last30d", func(p *P) { p.ShipByAfter = day(recent.Add(-30 * 24 * time.Hour)) }},
			{"shipby-old30d", func(p *P) { p.ShipByAfter, p.ShipByBefore = day(old), day(old.Add(30*24*time.Hour)) }},
		}},
		{"pastdue", []planValue[P]{
			{"pastdue", func(p *P) { p.PastDue = &yes }},
			{"not-pastdue", func(p *P) { p.PastDue = &no }},
		}},
	}
}

func salesOrderPlanCases() []planCase[domain.ListSalesOrdersParams] {
	type P = domain.ListSalesOrdersParams
	mid := planSalesCreatedAt(planSalesOrders / 2)
	cursor := func(dir pagination.Direction) func(*P) {
		return func(p *P) { p.Cursor = planCursorAt(mid, "so_plansales_~", dir) }
	}
	return planCases(
		P{AccountID: planSalesAccount, Limit: 25},
		salesOrderPlanDims(),
		[]planValue[P]{
			{"first", func(*P) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

func planCursorAt(at time.Time, id string, dir pagination.Direction) *string {
	c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: at, ID: id, Direction: dir})
	return &c
}

// salesOrderUnorderedFloor is the fewest orders any one of a request's unordered filters matches, or 0
// when it has none. An item or product line (a property of the order's lines) and a ship-by range (a
// column the list does not sort by) cannot be read in list order, so the best a plan can do is read
// the orders the most selective of them matches.
func salesOrderUnorderedFloor(t *testing.T, db *sql.DB, p domain.ListSalesOrdersParams) float64 {
	t.Helper()
	var counts []string
	var args []any
	count := func(where string, a ...any) {
		counts = append(counts, "(SELECT COUNT(*) FROM sales_order so WHERE so.owner_account_id = ? AND "+where+")")
		args = append(append(args, p.AccountID), a...)
	}
	if len(p.ItemIDs) > 0 {
		count("EXISTS (SELECT 1 FROM sales_order_line l WHERE l.sales_order_id = so.id AND l.item_id IN ("+placeholders(len(p.ItemIDs))+"))", stringArgs(p.ItemIDs)...)
	}
	if len(p.ProductLineIDs) > 0 {
		count("EXISTS (SELECT 1 FROM sales_order_line l JOIN product pr ON pr.id = l.product_id WHERE l.sales_order_id = so.id AND pr.product_line_id IN ("+
			placeholders(len(p.ProductLineIDs))+"))", stringArgs(p.ProductLineIDs)...)
	}
	if p.ShipByAfter != nil || p.ShipByBefore != nil {
		where, a := []string{"TRUE"}, []any{}
		if p.ShipByAfter != nil {
			where, a = append(where, "so.ship_by_date >= ?"), append(a, *p.ShipByAfter)
		}
		if p.ShipByBefore != nil {
			where, a = append(where, "so.ship_by_date <= ?"), append(a, *p.ShipByBefore)
		}
		count(strings.Join(where, " AND "), a...)
	}
	if len(counts) == 0 {
		return 0
	}
	return planLeastCount(t, db, counts, args)
}

// TestSalesOrderList_ReadsAboutAPage holds every filter combination ListSalesOrders accepts to reading
// about a page of sales orders (listPlanSuite).
func TestSalesOrderList_ReadsAboutAPage(t *testing.T) {
	ensureSalesCorpus(t)
	listPlanSuite[domain.ListSalesOrdersParams]{
		table: "sales_order", scopeColumn: "owner_account_id",
		from: "FROM sales_order so", alias: "so",
		cases: salesOrderPlanCases(),
		limit: func(p domain.ListSalesOrdersParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListSalesOrdersParams) error {
			if _, apiErr := NewSalesOrderRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: salesOrderUnorderedFloor,
	}.run(t)
}
