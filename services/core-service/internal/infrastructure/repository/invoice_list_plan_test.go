//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// planSalesRareSearch matches one invoice number; planSalesDenseSearch matches most of them.
const (
	planSalesRareSearch  = "2012345"
	planSalesDenseSearch = "2"
)

func invoicePlanDims() []planDim[domain.ListInvoicesParams] {
	type P = domain.ListInvoicesParams
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planSalesInvoicedAt(planSalesOrders - 1)
	old := planSalesInvoicedAt(planSalesOrders / 4)
	return []planDim[P]{
		{"status", []planValue[P]{
			{"paid", func(p *P) { p.Status = str("paid") }},
			{"unpaid", func(p *P) { p.Status = str("unpaid") }},
			{"overpaid", func(p *P) { p.Status = str("overpaid") }},
		}},
		{"customer", []planValue[P]{
			{"large", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(0)} }},
			{"mid", func(p *P) { p.CustomerIDs = []string{planSalesCustomerID(planSalesMidCustomer)} }},
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
		{"created", []planValue[P]{
			{"last30d", func(p *P) { p.StartDate = at(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *P) { p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour)) }},
		}},
		{"number", []planValue[P]{
			{"number-one", func(p *P) { p.Numbers = []string{planSalesRareSearch} }},
			{"number-batch", func(p *P) { p.Numbers = planSalesInvoiceNumbers(planSalesOrders/2, 50) }},
			{"number-none", func(p *P) { p.Numbers = []string{"9999999"} }},
		}},
		{"search", []planValue[P]{
			{"search-one", func(p *P) { p.Query = str(planSalesRareSearch) }},
			{"search-every", func(p *P) { p.Query = str(planSalesDenseSearch) }},
		}},
	}
}

// planSalesInvoiceNumbers is n consecutive invoice numbers from the corpus, starting at invoice from.
func planSalesInvoiceNumbers(from, n int) []string {
	numbers := make([]string, n)
	for i := range numbers {
		numbers[i] = fmt.Sprintf("%07d", 2_000_000+from+i)
	}
	return numbers
}

func invoicePlanPages[P any](setCursor func(*P, *string)) []planValue[P] {
	mid := planSalesInvoicedAt(planSalesOrders / 2)
	return []planValue[P]{
		{"first", func(*P) {}},
		{"deep-next", func(p *P) { setCursor(p, planCursorAt(mid, "in_plansales_~", pagination.DirectionForward)) }},
		{"deep-prev", func(p *P) { setCursor(p, planCursorAt(mid, "in_plansales_~", pagination.DirectionBackward)) }},
	}
}

// invoiceUnorderedFloor is the fewest invoices any one of a request's order filters matches, or 0 when
// it has none. A customer, group, sales rep, item, or product line is a property of the invoice's order
// or its lines, not of the invoice, so no invoice key reads it in list order; the best a plan can do is
// read the invoices of the orders the most selective one admits. A text search is not one: a substring
// match reads every invoice it does not stop on, and a request returning less than a page is not held to one.
func invoiceUnorderedFloor(t *testing.T, db *sql.DB, p domain.ListInvoicesParams) float64 {
	t.Helper()
	var counts []string
	var args []any
	count := func(where string, a ...any) {
		counts = append(counts, "(SELECT COUNT(*) FROM invoice inv WHERE inv.account_id = ? AND "+where+")")
		args = append(append(args, p.AccountID), a...)
	}
	if len(p.ItemIDs) > 0 {
		count("EXISTS (SELECT 1 FROM sales_order_line l WHERE l.sales_order_id = inv.sales_order_id AND l.item_id IN ("+
			placeholders(len(p.ItemIDs))+"))", stringArgs(p.ItemIDs)...)
	}
	if len(p.ProductLineIDs) > 0 {
		count("EXISTS (SELECT 1 FROM sales_order_line l JOIN product pr ON pr.id = l.product_id WHERE l.sales_order_id = inv.sales_order_id AND pr.product_line_id IN ("+
			placeholders(len(p.ProductLineIDs))+"))", stringArgs(p.ProductLineIDs)...)
	}
	orderFilter := func(column string, ids []string) {
		if len(ids) > 0 {
			count("EXISTS (SELECT 1 FROM sales_order so WHERE so.id = inv.sales_order_id AND "+column+" IN ("+placeholders(len(ids))+"))",
				stringArgs(ids)...)
		}
	}
	orderFilter("so.buyer_account_id", p.CustomerIDs)
	orderFilter("so.sales_rep_id", p.SalesRepIDs)
	if len(p.CustomerGroupIDs) > 0 {
		count("EXISTS (SELECT 1 FROM sales_order so JOIN account_relation ar ON ar.owner_account_id = inv.account_id"+
			" AND ar.counterparty_account_id = so.buyer_account_id AND ar.account_relation_role_code = 'customer'"+
			" WHERE so.id = inv.sales_order_id AND ar.account_group_id IN ("+placeholders(len(p.CustomerGroupIDs))+"))",
			stringArgs(p.CustomerGroupIDs)...)
	}
	if len(counts) == 0 {
		return 0
	}
	return planLeastCount(t, db, counts, args)
}

// customerInvoiceFloor is how many payable invoices the customer and its children have: the customer
// is a property of the invoice's order, which no invoice key reads in list order (invoiceUnorderedFloor).
func customerInvoiceFloor(t *testing.T, db *sql.DB, p domain.ListCustomerInvoicesParams) float64 {
	t.Helper()
	return planLeastCount(t, db, []string{`(SELECT COUNT(*) FROM invoice inv JOIN sales_order so ON so.id = inv.sales_order_id
		WHERE inv.account_id = ? AND inv.is_paid_in_full = false AND so.buyer_account_id IN (
			SELECT ar.counterparty_account_id FROM account_relation ar
			LEFT JOIN account_relation par ON par.id = ar.parent_account_relation_id
			WHERE ar.owner_account_id = ? AND ar.account_relation_role_code = 'customer'
			AND (ar.counterparty_account_id = ? OR par.counterparty_account_id = ?)))`},
		[]any{p.AccountID, p.AccountID, p.CustomerAccountID, p.CustomerAccountID})
}

// TestInvoiceList_ReadsAboutAPage holds every filter combination ListInvoices accepts to reading about
// a page of invoices (listPlanSuite).
func TestInvoiceList_ReadsAboutAPage(t *testing.T) {
	ensureSalesCorpus(t)
	type P = domain.ListInvoicesParams
	listPlanSuite[P]{
		table: "invoice", scopeColumn: "account_id",
		from: "FROM invoice inv", alias: "inv",
		cases: planCases(P{AccountID: planSalesAccount, Limit: 25}, invoicePlanDims(),
			invoicePlanPages(func(p *P, c *string) { p.Cursor = c })),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewInvoiceRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: invoiceUnorderedFloor,
	}.run(t)
}

// TestCustomerInvoiceList_ReadsAboutAPage holds ListCustomerInvoices (a customer's and its children's
// unpaid invoices) to reading about a page of invoices.
func TestCustomerInvoiceList_ReadsAboutAPage(t *testing.T) {
	ensureSalesCorpus(t)
	type P = domain.ListCustomerInvoicesParams
	str := func(s string) *string { return &s }
	dims := []planDim[P]{
		{"customer", []planValue[P]{
			{"mid", func(p *P) { p.CustomerAccountID = planSalesCustomerID(planSalesMidCustomer) }},
			{"parent", func(p *P) { p.CustomerAccountID = planSalesCustomerID(planSalesParentCustomer) }},
			{"rare", func(p *P) { p.CustomerAccountID = planSalesCustomerID(planSalesRareCustomer) }},
		}},
		{"search", []planValue[P]{
			{"search-one", func(p *P) { p.Query = str(planSalesRareSearch) }},
			{"search-every", func(p *P) { p.Query = str(planSalesDenseSearch) }},
		}},
	}
	listPlanSuite[P]{
		table: "invoice", scopeColumn: "account_id",
		from: "FROM invoice inv", alias: "inv",
		cases: planCases(P{AccountID: planSalesAccount, CustomerAccountID: planSalesCustomerID(0), Limit: 25}, dims,
			invoicePlanPages(func(p *P, c *string) { p.Cursor = c })),
		limit: func(p P) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p P) error {
			if _, apiErr := NewInvoiceRepo(q).ListByCustomer(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: customerInvoiceFloor,
	}.run(t)
}
