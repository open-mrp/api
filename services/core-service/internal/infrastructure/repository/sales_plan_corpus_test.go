//go:build plans

package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The sales corpus is one merchant shaped like the largest production tenant, scaled to a quarter:
// orders nearly all fulfilled and invoiced, a dozen still issued, a few hundred unpaid invoices, most
// orders with no sales rep, a handful of customers and items holding most of the volume, and long
// tails of rare customers, reps, groups, items, and product lines. Every filter of the sales order,
// invoice, receivable, and sales line lists has a dense value and a rare one here.
//
// Row widths follow production (35-character IDs, 7-digit numbers, a customer PO on a fifth of the
// orders, notes on some, carrier and terms on nearly all) so the optimizer prices an ordered walk
// against a scan-and-sort the way it does there.
const (
	planSalesAccount    = "ac_plansales"
	planSalesOrders     = 30_000
	planSalesCustomers  = 500
	planSalesReps       = 10
	planSalesItems      = 400
	planSalesLines      = 26
	planSalesSpan       = 8 * 365 * 24 * time.Hour
	planSalesInvoiceLag = 2 * 24 * time.Hour

	// planSalesCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planSalesCorpusVersion = "Plan Test Sales Merchant v5"

	// The last orders are still open: a dozen issued, two estimates.
	planSalesOpenOrders = 14
	// planSalesUnpaidRecent is how many of the newest invoices are unpaid.
	planSalesUnpaidRecent = 700
)

var planSalesOrigin = time.Date(2018, 9, 1, 0, 0, 0, 0, time.UTC)

func planSalesOrderID(i int) string   { return fmt.Sprintf("so_plansales_%022d", i) }
func planSalesInvoiceID(i int) string { return fmt.Sprintf("in_plansales_%022d", i) }
func planSalesLineID(i, j int) string { return fmt.Sprintf("sol_plansales_%019d_%d", i, j) }
func planSalesInvLineID(i, j int) string {
	return fmt.Sprintf("inl_plansales_%019d_%d", i, j)
}
func planSalesCustomerID(c int) string    { return fmt.Sprintf("ac_plansales_c%019d", c) }
func planSalesRelationID(c int) string    { return fmt.Sprintf("ar_plansales_%021d", c) }
func planSalesAddressID(c int) string     { return fmt.Sprintf("ad_plansales_%021d", c) }
func planSalesRepID(r int) string         { return fmt.Sprintf("acus_plansales_%019d", r) }
func planSalesItemID(k int) string        { return fmt.Sprintf("it_plansales_%021d", k) }
func planSalesProductID(k int) string     { return fmt.Sprintf("pd_plansales_%021d", k) }
func planSalesProductLineID(p int) string { return fmt.Sprintf("pdln_plansales_%019d", p) }
func planSalesGroupID(g string) string    { return "acgp_plansales_" + g }

func planSalesCreatedAt(i int) time.Time {
	return planSalesOrigin.Add(time.Duration(i) * (planSalesSpan / planSalesOrders))
}

func planSalesInvoicedAt(i int) time.Time { return planSalesCreatedAt(i).Add(planSalesInvoiceLag) }

// A rare value holds one order in every planSalesRareEvery, spread across the account's history: more
// than a page, so a list on it cannot stop early without a key that finds them.
const planSalesRareEvery = 750

func planSalesRareCustomerOrder(i int) bool { return i%planSalesRareEvery == 100 }
func planSalesRareRepOrder(i int) bool      { return i%planSalesRareEvery == 200 }
func planSalesRareItemOrder(i int) bool     { return i%planSalesRareEvery == 300 }

var planSalesOldUnpaidOrders = map[int]bool{1_000: true, 12_000: true, 25_000: true}

const (
	planSalesRareCustomer = planSalesCustomers - 1
	// planSalesParentCustomer is a large customer that is the parent of planSalesChildCustomers.
	planSalesParentCustomer = 1
	planSalesRareRep        = planSalesReps - 1
	planSalesRareItem       = planSalesItems - 1
	planSalesRareLine       = planSalesLines - 1
	planSalesDenseItem      = 0
	planSalesDenseLine      = 0
	// planSalesMidCustomer holds about 1% of the orders: too many to sort, too few to stumble on.
	planSalesMidCustomer = 8
)

var planSalesChildCustomers = []int{200, 201, 202, 203, 204}

// planSalesCustomer gives a quarter of the orders to 8 large customers, half to 50 mid-size ones, and
// spreads the rest over the tail.
func planSalesCustomer(i int) int {
	switch {
	case planSalesRareCustomerOrder(i):
		return planSalesRareCustomer
	case i%4 == 0:
		return (i / 4) % 8
	case i%4 == 3:
		return 58 + (i*104_729)%(planSalesCustomers-59)
	default:
		return 8 + (i*7_919)%50
	}
}

// planSalesGroup puts most customers in "big", the rest in "small", and only the rare customer in "rare".
func planSalesGroup(c int) string {
	switch {
	case c == planSalesRareCustomer:
		return "rare"
	case c%8 < 5:
		return "big"
	default:
		return "small"
	}
}

// planSalesRep is the order's sales rep. As in production, a rep belongs to the customer: one large
// customer's, ten mid-size ones', and a sprinkling of the tail's, so nearly nine orders in ten have
// none. The rare rep is the exception, set on orders spread across every customer.
func planSalesRep(i int) (int, bool) {
	c := planSalesCustomer(i)
	switch {
	case planSalesRareRepOrder(i):
		return planSalesRareRep, true
	case c == 5:
		return 0, true
	case c >= 8 && c < 58 && c%5 == 0:
		return 1 + c%10/5, true
	case c >= 58 && c%40 == 3:
		return 3 + (c/40)%6, true
	default:
		return 0, false
	}
}

func planSalesStatus(i int) string {
	switch {
	case i == planSalesOrders-1 || i == planSalesOrders-5:
		return "estimate"
	case i >= planSalesOrders-planSalesOpenOrders:
		return "issued"
	default:
		return "fulfilled"
	}
}

func planSalesLineCount(i int) int {
	if i%5 < 3 {
		return 4
	}
	return 3
}

// planSalesItem puts a third of the lines on ten items and spreads the rest over the catalog.
func planSalesItem(i, j int) int {
	switch {
	case j == 0 && planSalesRareItemOrder(i):
		return planSalesRareItem
	case (i+j)%3 == 0:
		return (i*3 + j) % 10
	default:
		return 10 + (i*31+j*7_919)%(planSalesItems-11)
	}
}

// planSalesProductLine gives the ten dense items one line and the rare item its own.
func planSalesProductLine(k int) int {
	switch {
	case k == planSalesRareItem:
		return planSalesRareLine
	case k < 10:
		return planSalesDenseLine
	default:
		return 1 + k%23
	}
}

func planSalesUnpaid(i int) bool {
	return i >= planSalesOrders-planSalesOpenOrders-planSalesUnpaidRecent || planSalesOldUnpaidOrders[i]
}

func planSalesOverpaid(i int) bool { return i%4_000 == 777 }

// planSalesInvoiced reports whether order i has an invoice: every fulfilled sales order does.
func planSalesInvoiced(i int) bool {
	return planSalesStatus(i) == "fulfilled" && !planSalesPurchaseOrder(i)
}

func planSalesPurchaseOrder(i int) bool { return i%1_000 == 500 }

// planSalesForeignSeller marks the few orders the account holds but did not sell, which lists exclude.
func planSalesForeignSeller(i int) bool { return i%3_000 == 1_234 }

var planSalesCorpusOnce sync.Once

func ensureSalesCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planSalesCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sales_line_fact WHERE account_id = ?", planSalesAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planSalesAccount).Scan(&version)
		if have > 0 && version == planSalesCorpusVersion {
			return
		}
		t.Logf("seeding the sales plan corpus (%d orders); it is kept for later runs", planSalesOrders)
		seedSalesCorpus(t, db)
	})
}

func seedSalesCorpus(t *testing.T, db *sql.DB) {
	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, q := range []string{
		"DELETE FROM sales_line_fact WHERE account_id = ?",
		"DELETE FROM invoice WHERE account_id = ?",
		"DELETE FROM sales_order WHERE owner_account_id = ?",
		"DELETE FROM account_relation WHERE owner_account_id = ?",
		"DELETE FROM account_group WHERE owner_account_id = ?",
		"DELETE FROM account_user WHERE account_id = ?",
		"DELETE FROM item WHERE account_id = ?",
		"DELETE FROM product_line WHERE account_id = ?",
		"DELETE FROM item_category WHERE account_id = ?",
	} {
		exec(q, planSalesAccount)
	}
	for _, table := range []string{"invoice_line", "sales_order_line", "quantity", "rate", "product", "address", "geolocation", "`user`"} {
		exec("DELETE FROM " + table + " WHERE id LIKE '%\\_plansales\\_%'")
	}
	exec("DELETE FROM account WHERE id LIKE 'ac\\_plansales\\_%'")
	exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
	      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planSalesAccount, "Plan Test Sales Merchant (seeding)")

	ins := &planInserter{t: t, db: db}

	for r := range planSalesReps {
		userID := fmt.Sprintf("us_plansales_%021d", r)
		ins.add("`user`", "id, name, username", userID, fmt.Sprintf("Plan Rep %02d", r), fmt.Sprintf("planrep%02d", r))
		ins.add("account_user", "id, user_id, account_id", planSalesRepID(r), userID, planSalesAccount)
	}
	for _, g := range []string{"big", "small", "rare"} {
		ins.add("account_group", "id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code",
			planSalesGroupID(g), planSalesAccount, "Plan "+g, "commission_applied", "billed_freight", "type_group")
	}
	ins.flush()

	for c := range planSalesCustomers {
		id := planSalesCustomerID(c)
		ins.add("account", "id, name, account_type_code, onboarding_status_code", id, fmt.Sprintf("Plan Sales Customer %04d Inc", c), "company", "unclaimed")
		ins.add("geolocation", "id, street_line_1, locality, state, postal_code, country",
			fmt.Sprintf("geo_plansales_%021d", c), fmt.Sprintf("%d Industrial Pkwy", 100+c), "Charlotte", "NC", "28202", "US")
		ins.add("address", "id, name, geolocation_id", planSalesAddressID(c), fmt.Sprintf("Plan Customer %04d Receiving", c), fmt.Sprintf("geo_plansales_%021d", c))
	}
	ins.flush()
	parentOf := map[int]int{}
	for _, child := range planSalesChildCustomers {
		parentOf[child] = planSalesParentCustomer
	}
	for c := range planSalesCustomers {
		var parent any
		if p, ok := parentOf[c]; ok {
			parent = planSalesRelationID(p)
		}
		ins.add("account_relation", "id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number, priority_code, account_group_id, parent_account_relation_id",
			planSalesRelationID(c), planSalesAccount, planSalesCustomerID(c), "customer", fmt.Sprintf("C%05d", c), "normal", planSalesGroupID(planSalesGroup(c)), parent)
	}
	ins.flush()

	ins.add("item_category", "id, name, account_id, item_category_type_code, unit_group_id", "ic_plansales", "Plan Products", planSalesAccount, "product_category", "each_group")
	for p := range planSalesLines {
		ins.add("product_line", "id, name, account_id, unit_group_id", planSalesProductLineID(p), fmt.Sprintf("Plan Line %02d", p), planSalesAccount, "each_group")
	}
	for k := range planSalesItems {
		ins.add("item", "id, sku, description, account_id, item_type_code, unit_value_id, burn_rate_id, unit_cost_id, item_category_id",
			planSalesItemID(k), fmt.Sprintf("PS-%05d", k), fmt.Sprintf("Plan sales product %05d, 12in x 18in", k), planSalesAccount, "product",
			fmt.Sprintf("qy_plansales_uv_%05d", k), fmt.Sprintf("qy_plansales_br_%05d", k), fmt.Sprintf("rt_plansales_uc_%05d", k), "ic_plansales")
		ins.add("product", "id, item_id, product_type_code, product_line_id", planSalesProductID(k), planSalesItemID(k), "sale", planSalesProductLineID(planSalesProductLine(k)))
	}
	ins.flush()

	const someID = "zz_plansales_0000000000000000000000"
	for i := range planSalesOrders {
		createdAt := planSalesCreatedAt(i)
		c := planSalesCustomer(i)
		var rep, po, note any
		if r, ok := planSalesRep(i); ok {
			rep = planSalesRepID(r)
		}
		if i%5 == 2 {
			po = fmt.Sprintf("PO%07d", 3_000_000+i)
		}
		if i%6 == 1 {
			note = "Deliver to dock 4; call ahead before arrival."
		}
		typeCode := "sales_order"
		if planSalesPurchaseOrder(i) {
			typeCode = "purchase_order"
		}
		seller := planSalesAccount
		if planSalesForeignSeller(i) {
			seller = planSalesCustomerID(c)
		}
		status := planSalesStatus(i)
		var completedAt any
		if status == "fulfilled" {
			completedAt = createdAt.Add(planSalesInvoiceLag)
		}
		ins.add("sales_order", "id, number, customer_po_number, note, billing_address_id, shipping_address_id, carrier_id, carrier_option_id, "+
			"priority_code, sales_rep_id, shipping_term_id, payment_term_id, sales_order_status_code, sales_order_type_code, "+
			"buyer_account_id, seller_account_id, owner_account_id, issued_at, completed_at, ship_by_date, created_at, updated_at",
			planSalesOrderID(i), fmt.Sprintf("%07d", 1_000_000+i), po, note, planSalesAddressID(c), planSalesAddressID(c), someID, someID,
			"normal", rep, someID, someID, status, typeCode,
			planSalesCustomerID(c), seller, planSalesAccount, createdAt, completedAt, createdAt.Add(7*24*time.Hour).Format("2006-01-02"), createdAt, createdAt)

		invoiced := planSalesInvoiced(i)
		invoicedAt := planSalesInvoicedAt(i)
		if invoiced {
			var invNote any
			if i%16 == 3 {
				invNote = "Net 30, remit to lockbox."
			}
			ins.add("invoice", "id, number, note, is_paid_in_full, is_over_paid, sales_order_id, billing_address_id, account_id, created_at, updated_at",
				planSalesInvoiceID(i), fmt.Sprintf("%07d", 2_000_000+i), invNote, !planSalesUnpaid(i), planSalesOverpaid(i),
				planSalesOrderID(i), planSalesAddressID(c), planSalesAccount, invoicedAt, invoicedAt)
		}
		for j := range planSalesLineCount(i) {
			k := planSalesItem(i, j)
			qID := fmt.Sprintf("qy_plansales_%019d_%d", i, j)
			rID := fmt.Sprintf("rt_plansales_%019d_%d", i, j)
			qty := 10 + (i+j)%90
			price := 3 + (i*7+j)%40
			ins.add("quantity", "id, value, unit_id, created_at, updated_at", qID, qty, "each", createdAt, createdAt)
			ins.add("rate", "id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at", rID, price, "dollar", "each", createdAt, createdAt)
			ins.add("sales_order_line", "id, product_sku, product_description, line_item_number, product_id, item_id, sales_order_id, quantity_id, unit_price_id, created_at, updated_at",
				planSalesLineID(i, j), fmt.Sprintf("PS-%05d", k), fmt.Sprintf("Plan sales product %05d, 12in x 18in", k), j+1,
				planSalesProductID(k), planSalesItemID(k), planSalesOrderID(i), qID, rID, createdAt, createdAt)
			if !invoiced {
				continue
			}
			iqID := fmt.Sprintf("qy_plansales_%019d_%di", i, j)
			ins.add("quantity", "id, value, unit_id, created_at, updated_at", iqID, qty, "each", invoicedAt, invoicedAt)
			ins.add("invoice_line", "id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at",
				planSalesInvLineID(i, j), planSalesInvoiceID(i), iqID, planSalesLineID(i, j), invoicedAt, invoicedAt)
			ins.add("sales_line_fact", "account_id, invoiced_at, invoice_line_id, invoice_id, sales_order_id, sales_order_type_code, buyer_account_id, "+
				"sales_rep_id, product_id, item_id, product_line_id, quantity_base, total_invoiced, total_cost, refreshed_at, ordered_at, is_priced",
				planSalesAccount, invoicedAt, planSalesInvLineID(i, j), planSalesInvoiceID(i), planSalesOrderID(i), "sales_order", planSalesCustomerID(c),
				rep, planSalesProductID(k), planSalesItemID(k), planSalesProductLineID(planSalesProductLine(k)), qty, qty*price, qty*price/2, invoicedAt, createdAt, true)
		}
	}
	ins.flush()
	exec("ANALYZE TABLE sales_order, sales_order_line, invoice, invoice_line, sales_line_fact, account_relation, product")
	// Marked last, so a run interrupted mid-seed rebuilds it.
	exec("UPDATE account SET name = ? WHERE id = ?", planSalesCorpusVersion, planSalesAccount)
}

// planLeastCount is the smallest of counts, each a scalar COUNT subquery, whose placeholders args fills in order.
func planLeastCount(t *testing.T, db *sql.DB, counts []string, args []any) float64 {
	t.Helper()
	expr := counts[0]
	if len(counts) > 1 {
		expr = "LEAST(" + strings.Join(counts, ", ") + ")"
	}
	var n float64
	require.NoError(t, db.QueryRow("SELECT "+expr, args...).Scan(&n))
	return n
}

// planInserter batches multi-row INSERTs per table and column list.
type planInserter struct {
	t       *testing.T
	db      *sql.DB
	order   []string
	pending map[string]*planInsert
}

type planInsert struct {
	table, cols string
	rows        int
	args        []any
}

const planInsertBatch = 500

func (p *planInserter) add(table, cols string, values ...any) {
	if p.pending == nil {
		p.pending = map[string]*planInsert{}
	}
	key := table + "(" + cols + ")"
	b, ok := p.pending[key]
	if !ok {
		b = &planInsert{table: table, cols: cols}
		p.pending[key] = b
		p.order = append(p.order, key)
	}
	b.rows++
	b.args = append(b.args, values...)
	if b.rows >= planInsertBatch {
		p.write(b)
	}
}

func (p *planInserter) write(b *planInsert) {
	if b.rows == 0 {
		return
	}
	row := "(" + strings.TrimSuffix(strings.Repeat("?, ", len(b.args)/b.rows), ", ") + ")"
	rows := strings.TrimSuffix(strings.Repeat(row+", ", b.rows), ", ")
	_, err := p.db.Exec("INSERT INTO "+b.table+" ("+b.cols+") VALUES "+rows, b.args...)
	require.NoError(p.t, err, b.table)
	b.rows, b.args = 0, nil
}

// flush writes every pending row, tables in the order they were first added.
func (p *planInserter) flush() {
	for _, key := range p.order {
		p.write(p.pending[key])
	}
}
