package repository

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// The invoice list's keys. The created, paid, and over-paid keys yield an account's invoices in list
// order; the order key finds the invoices of orders chosen first by a filter on the order.
const (
	invoiceCreatedIndex  = "invoice_account_created_idx"
	invoicePaidIndex     = "invoice_account_unpaid_created_idx"
	invoiceOverPaidIndex = "invoice_account_over_paid_created_idx"
	invoiceOrderIndex    = "invoice_account_sales_order_idx"
)

// invoiceListIndexes is the keys an invoice page may be read from when it is not driven from a set of
// orders. A status is read from its own key, which is in list order and pins it; the created key walks
// past every invoice of the other status (the newest invoices are the unpaid ones). The order key is
// offered for an order filter's invoices to be looked up by.
func invoiceListIndexes(status string) []string {
	switch status {
	case "paid", "unpaid":
		return []string{invoicePaidIndex, invoiceOrderIndex}
	case "overpaid":
		return []string{invoiceOverPaidIndex, invoiceOrderIndex}
	default:
		return []string{invoiceCreatedIndex, invoicePaidIndex, invoiceOverPaidIndex, invoiceOrderIndex}
	}
}

// invoicePageJoins are the inner joins an invoice must survive to be listed, and the tables the list's
// filters and search read. JOIN_SUFFIX keeps them after the invoice: driving from the account's customer
// relations instead reads every invoice of the account to sort it.
const invoicePageJoins = `
JOIN sales_order so ON so.id = inv.sales_order_id
JOIN account_relation ar ON ar.owner_account_id = inv.account_id
    AND ar.counterparty_account_id = so.buyer_account_id
    AND ar.account_relation_role_code = 'customer'
JOIN account buyer ON buyer.id = so.buyer_account_id`

// invoicePage is the statement choosing one page of invoices (inv).
type invoicePage struct {
	f *listFilter
	// indexes are FORCE INDEX'd on inv.
	indexes []string
	// joins are invoicePageJoins and any others, aliased joined.
	joins  string
	joined []string
	// driver, when set, selects the sales_order_id of every order whose invoices may list (args
	// driverArgs). The page is then read from its orders' invoices alone, looked up by the order key.
	driver     string
	driverArgs []any
}

// drive reads the page from subquery's orders: a substring search cannot stop a walk in list order early,
// so beside a filter on the order it reads only that filter's orders' invoices, a bounded read, rather
// than walking the whole account. subquery selects the orders' ids as its only column.
func (p *invoicePage) drive(subquery string, args ...any) {
	p.driver, p.driverArgs = subquery, args
	p.indexes = []string{invoiceOrderIndex}
}

// ids runs the page query and returns its invoice ids in page order.
func (p *invoicePage) ids(ctx context.Context, q sqlc.DBTX, cursor *pagination.StringCursor, limit int32) ([]string, error) {
	orderBy := p.f.keyset(cursor, "inv.created_at", "inv.id")
	hint := "JOIN_SUFFIX(" + strings.Join(p.joined, ", ") + ")"
	from := "invoice inv FORCE INDEX (" + strings.Join(p.indexes, ", ") + ")"
	var args []any
	if p.driver != "" {
		hint = "JOIN_PREFIX(drv, inv) " + hint
		from = "(SELECT DISTINCT d.id FROM (" + p.driver + ") d) drv\nJOIN " + from + " ON inv.sales_order_id = drv.id"
		args = append(args, p.driverArgs...)
	}
	query := "SELECT /*+ " + hint + " */ inv.id FROM " + from + p.joins + p.f.whereSQL() + "\nORDER BY " + orderBy + "\nLIMIT ?"
	return selectStrings(ctx, q, query, append(append(args, p.f.args...), limit)...)
}

// orderSemijoin admits invoices whose order is in a set read from another table first. A filter on the
// order or its lines is not on the invoice, so no invoice key yields its invoices in list order; as a
// semijoin the planner can read a rare value's orders and look their invoices up, or walk the invoices
// in order and probe a common one.
func (f *listFilter) orderSemijoin(subquery string, args ...any) {
	f.add("inv.sales_order_id IN ("+subquery+")", args...)
}

// listInvoicePage chooses one page of the invoice list and returns its ids in page order.
func (r *invoiceRepoImpl) listInvoicePage(ctx context.Context, params domain.ListInvoicesParams, cursor *pagination.StringCursor) ([]string, error) {
	var productIDs []string
	if len(params.ProductLineIDs) > 0 {
		var err error
		if productIDs, err = productsInLines(ctx, r.queries.DB(), params.ProductLineIDs); err != nil || len(productIDs) == 0 {
			return nil, err
		}
	}
	largeGroup := false
	if len(params.CustomerGroupIDs) > 0 {
		var err error
		if largeGroup, err = groupIsLarge(ctx, r.queries.DB(), params.AccountID, params.CustomerGroupIDs); err != nil {
			return nil, err
		}
	}

	f := &listFilter{}
	f.add("inv.account_id = ?", params.AccountID)
	search := buildInvoiceSearchParams(params.Query)
	if search.Valid {
		// Reaches the order and relation joined one-to-one, so the search widens without fanning rows out.
		f.add("(inv.number LIKE ? OR inv.note LIKE ? OR buyer.name LIKE ? OR so.number LIKE ? OR so.customer_po_number LIKE ?"+
			" OR ar.external_number LIKE ? OR ar.alias LIKE ? OR ar.notes LIKE ?)",
			search.String, search.String, search.String, search.String, search.String, search.String, search.String, search.String)
	}
	var status string
	if params.Status != nil && *params.Status != "" && *params.Status != "all" {
		status = *params.Status
		switch status {
		case "paid":
			f.add("inv.is_paid_in_full = true")
		case "unpaid":
			// An overpaid invoice is also paid in full, so it lists under paid, not here.
			f.add("inv.is_paid_in_full = false")
		case "overpaid":
			f.add("inv.is_over_paid = true")
		default:
			f.add("FALSE")
		}
	}
	f.in("so.buyer_account_id", params.CustomerIDs)
	f.in("ar.account_group_id", params.CustomerGroupIDs)
	f.in("so.sales_rep_id", params.SalesRepIDs)
	if params.StartDate != nil {
		f.add("inv.created_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		f.add("inv.created_at <= ?", *params.EndDate)
	}

	// The order filters again, each as a set of orders the planner can read first. A large group is left
	// out: walking the invoices in order meets a page of it sooner than reading it whole.
	orders := &listFilter{}
	orders.in("so2.buyer_account_id", params.CustomerIDs)
	if len(params.CustomerGroupIDs) > 0 && !largeGroup {
		orders.add("so2.buyer_account_id IN (SELECT gr.counterparty_account_id FROM account_relation gr WHERE gr.owner_account_id = ?"+
			" AND gr.account_group_id IN ("+placeholders(len(params.CustomerGroupIDs))+"))",
			append([]any{params.AccountID}, stringArgs(params.CustomerGroupIDs)...)...)
	}
	orders.in("so2.sales_rep_id", params.SalesRepIDs)
	// Scoped to the order's lines, not the invoice's: an invoice bills a shipment's subset of them.
	sets, setArgs := orderLineSubqueries(params.ItemIDs, productIDs)
	if len(orders.where) > 0 {
		sets = append(sets, "SELECT so2.id FROM sales_order so2"+orders.whereSQL())
		setArgs = append(setArgs, orders.args)
	}

	page := &invoicePage{
		f: f, indexes: invoiceListIndexes(status),
		joins: invoicePageJoins + `
JOIN address addr ON addr.id = inv.billing_address_id
JOIN geolocation geo ON geo.id = addr.geolocation_id`,
		joined: []string{"so", "ar", "buyer", "addr", "geo"},
	}
	for i, set := range sets {
		if i == 0 && search.Valid {
			page.drive(set, setArgs[i]...)
			continue
		}
		f.orderSemijoin(set, setArgs[i]...)
	}
	return page.ids(ctx, r.queries.DB(), cursor, params.Limit+1)
}

// listCustomerInvoicePage chooses one page of a customer's payable invoices and returns its ids in page
// order. They are read from the customer's orders: an account's unpaid invoices are few, but walking
// them in order still reads every other customer's.
func (r *invoiceRepoImpl) listCustomerInvoicePage(ctx context.Context, params domain.ListCustomerInvoicesParams, cursor *pagination.StringCursor) ([]string, error) {
	// A parent settles for its children, so their invoices are payable here too.
	buyers, err := r.customerAndChildren(ctx, params.AccountID, params.CustomerAccountID)
	if err != nil {
		return nil, err
	}
	f := &listFilter{}
	f.add("inv.account_id = ?", params.AccountID)
	// An overpaid invoice is also paid in full, so it owes nothing and is left out.
	f.add("inv.is_paid_in_full = false")
	f.in("so.buyer_account_id", buyers)
	if search := buildInvoiceSearchParams(params.Query); search.Valid {
		f.add("(inv.number LIKE ? OR so.number LIKE ? OR so.customer_po_number LIKE ?)", search.String, search.String, search.String)
	}
	page := &invoicePage{f: f, joins: invoicePageJoins, joined: []string{"so", "ar", "buyer"}}
	page.drive("SELECT so2.id FROM sales_order so2 WHERE so2.buyer_account_id IN ("+placeholders(len(buyers))+")", stringArgs(buyers)...)
	return page.ids(ctx, r.queries.DB(), cursor, params.Limit+1)
}

// customerAndChildren is the customer and every customer of the account whose relation names the
// customer's relation as its parent.
func (r *invoiceRepoImpl) customerAndChildren(ctx context.Context, accountID, customerID string) ([]string, error) {
	children, err := selectStrings(ctx, r.queries.DB(), `
SELECT child.counterparty_account_id FROM account_relation child
WHERE child.owner_account_id = ? AND child.account_relation_role_code = 'customer'
AND child.parent_account_relation_id = (
    SELECT par.id FROM account_relation par
    WHERE par.owner_account_id = ? AND par.counterparty_account_id = ? AND par.account_relation_role_code = 'customer')`,
		accountID, accountID, customerID)
	if err != nil {
		return nil, err
	}
	return append([]string{customerID}, children...), nil
}
