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

// The fulfillment corpus is one merchant shaped like the largest production tenant's order flow: an
// order per pick, a shipment per closed pick, three or four lines on each, a few dozen picks still
// open among tens of thousands closed. Customers are skewed the way production's are (twenty large
// ones, a long tail, a few hundred that ordered once, one that ordered three times), and so are
// product lines (one on over half the picks, one on a few dozen, one never ordered), so every filter
// has a dense value and a rare one.
//
// Row widths follow production too: 35-character ids, five-digit numbers, a PO number on most orders,
// tracking and bill-of-lading numbers on a fraction of shipments.
const (
	planFulAccount     = "ac_planful"
	planFulOrders      = 30_000
	planFulCustomers   = 600
	planFulOnceBuyers  = 300
	planFulProducts    = 120
	planFulSpan        = 4 * 365 * 24 * time.Hour
	planFulOpenPicks   = 12
	planFulPackedShips = 5

	// planFulCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planFulCorpusVersion = "Plan Fulfillment Merchant v2"

	// planFulBigShipment is the order whose shipment has production's most lines.
	planFulBigShipment      = planFulOrders / 2
	planFulBigShipmentLines = 176
)

var planFulOrigin = time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)

func planFulID(prefix string, i int) string { return fmt.Sprintf("%s_planful_%024d", prefix, i) }
func planFulCustomerID(c int) string        { return fmt.Sprintf("%s_c%06d", planFulAccount, c) }
func planFulOnceBuyerID(c int) string       { return fmt.Sprintf("%s_o%06d", planFulAccount, c) }
func planFulProductID(p int) string         { return fmt.Sprintf("pd_planful_%04d", p) }
func planFulItemID(p int) string            { return fmt.Sprintf("it_planful_%04d", p) }
func planFulNumber(i int) string            { return fmt.Sprintf("%d", 10_000+i) }

// planFulRareCustomer holds only the orders planFulRareCustomerOrders names.
const planFulRareCustomer = planFulCustomers - 1

var planFulRareCustomerOrders = map[int]bool{100: true, 15_001: true, 29_990: true}

func planFulCreatedAt(i int) time.Time {
	return planFulOrigin.Add(time.Duration(i) * (planFulSpan / planFulOrders))
}

// planFulBuyer gives half the volume to 20 large customers, one order in a hundred to a customer that
// ordered once, and spreads the rest over the tail.
func planFulBuyer(i int) string {
	switch {
	case planFulRareCustomerOrders[i]:
		return planFulCustomerID(planFulRareCustomer)
	case i%100 == 77:
		return planFulOnceBuyerID((i / 100) % planFulOnceBuyers)
	case i%10 < 5:
		return planFulCustomerID((i / 10) % 20)
	default:
		return planFulCustomerID(20 + (i*7919)%(planFulCustomers-21))
	}
}

// The product lines: dense is on over half the orders, mid on a fifth, rare on a few dozen, and zero
// has a product nobody ordered.
const (
	planFulLineDense = "pdln_planful_dense"
	planFulLineMid   = "pdln_planful_mid"
	planFulLineTail  = "pdln_planful_tail"
	planFulLineRare  = "pdln_planful_rare"
	planFulLineZero  = "pdln_planful_zero"

	planFulRareProduct = planFulProducts - 2
	planFulZeroProduct = planFulProducts - 1
)

func planFulProductLine(p int) string {
	switch {
	case p == planFulRareProduct:
		return planFulLineRare
	case p == planFulZeroProduct:
		return planFulLineZero
	case p < 40:
		return planFulLineDense
	case p < 60:
		return planFulLineMid
	default:
		return planFulLineTail
	}
}

// planFulLineProducts is the product on each of order i's lines.
func planFulLineProducts(i int) []int {
	n := 1 + i%6
	if i == planFulBigShipment {
		n = planFulBigShipmentLines
	}
	out := make([]int, n)
	for j := range out {
		switch {
		case j == 0 && i%1000 < 1:
			out[j] = planFulRareProduct
		case j == 0 && i%3 == 0:
			out[j] = 0
		case j%2 == 0:
			out[j] = (i + j) % 40
		default:
			out[j] = 40 + (i*7+j)%(planFulRareProduct-40)
		}
	}
	return out
}

// Sales reps credited on orders: dense on every fourth, mid on one in two hundred, rare on three. They
// ignore the customer's default rep, so filtering on the order's rep and on the customer's differ.
const (
	planFulRepDense = "acus_planful_dense"
	planFulRepMid   = "acus_planful_mid"
	planFulRepRare  = "acus_planful_rare"
	// planFulRepDefault is only ever a customer's default rep; no order credits them.
	planFulRepDefault = "acus_planful_default"
)

var planFulRareRepOrders = map[int]bool{4_321: true, 14_321: true, 24_321: true}

// planFulOrderRep is the sales rep order i credits, or nil.
func planFulOrderRep(i int) any {
	switch {
	case planFulRareRepOrders[i]:
		return planFulRepRare
	case i%200 == 3:
		return planFulRepMid
	case i%4 == 1:
		return planFulRepDense
	}
	return nil
}

// planFulIsOpen reports an order whose pick is still open (and so has no shipment yet).
func planFulIsOpen(i int) bool { return i >= planFulOrders-planFulOpenPicks }

// qualifiedPlanDims names each value by its filter ("customer=rare"), since several filters share
// value names and a case is named by its values alone.
func qualifiedPlanDims[P any](dims []planDim[P]) []planDim[P] {
	for _, d := range dims {
		for i := range d.values {
			d.values[i].label = d.name + "=" + d.values[i].label
		}
	}
	return dims
}

var planFulCorpusOnce sync.Once

func ensureFulfillmentCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planFulCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM pick WHERE account_id = ?", planFulAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planFulAccount).Scan(&version)
		if have >= planFulOrders && version == planFulCorpusVersion {
			return
		}
		t.Logf("seeding the fulfillment plan corpus (%d orders); it is kept for later runs", planFulOrders)
		seedFulfillmentCorpus(t, db)
	})
}

func seedFulfillmentCorpus(t *testing.T, db *sql.DB) {
	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	insert := func(head string, rows int, row func(i int) []any) {
		const batch = 1_000
		for start := 0; start < rows; start += batch {
			var vals []string
			var args []any
			for i := start; i < min(start+batch, rows); i++ {
				r := row(i)
				if r == nil {
					continue
				}
				vals = append(vals, "("+placeholders(len(r))+")")
				args = append(args, r...)
			}
			if len(vals) > 0 {
				exec(head+" VALUES "+strings.Join(vals, ","), args...)
			}
		}
	}

	for _, table := range []string{"shipment_line", "shipment", "pick_line", "pick", "sales_order_line", "sales_order"} {
		exec("DELETE FROM `"+table+"` WHERE id LIKE ?", "%\\_planful\\_%")
	}
	exec("DELETE FROM quantity WHERE id LIKE 'qy\\_shl\\_planful\\_%'")
	exec("DELETE FROM product WHERE id LIKE 'pd\\_planful\\_%'")
	exec("DELETE FROM product_line WHERE id LIKE 'pdln\\_planful\\_%'")
	exec("DELETE FROM account_relation WHERE owner_account_id = ?", planFulAccount)
	exec("DELETE FROM account WHERE id LIKE 'ac\\_planful\\_%'")

	exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
	                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
	      VALUES ('un_planful', 'planful', 'un_planful', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
	exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
	      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planFulAccount, planFulCorpusVersion)
	exec(`INSERT IGNORE INTO carrier (id, name) VALUES ('cr_planful', 'Plan Freight')`)
	for _, g := range []string{"big", "once", "small"} {
		exec(`INSERT IGNORE INTO account_group (id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code)
		      VALUES (?, ?, ?, 'commission_applied', 'billed_freight', 'type_group')`, "ag_planful_"+g, planFulAccount, "Plan "+g)
	}

	// Customers: the large twenty and the first of the tail are in the big group, the first ten defaulting
	// to the dense rep; the once-ordering customers are their own group, half defaulting to a rep no order
	// credits; the rare customer is alone in the small group, defaulting to the rare rep.
	type customer struct {
		id, name, group, rep string
	}
	var customers []customer
	for c := range planFulCustomers {
		cu := customer{id: planFulCustomerID(c), name: fmt.Sprintf("Plan Customer %04d", c)}
		switch {
		case c == planFulRareCustomer:
			cu.group, cu.rep = "ag_planful_small", planFulRepRare
		case c < 200:
			cu.group = "ag_planful_big"
		}
		if c < 10 {
			cu.rep = planFulRepDense
		}
		customers = append(customers, cu)
	}
	for c := range planFulOnceBuyers {
		cu := customer{id: planFulOnceBuyerID(c), name: fmt.Sprintf("Plan Once Buyer %04d", c), group: "ag_planful_once"}
		if c%2 == 0 {
			cu.rep = planFulRepDefault
		}
		customers = append(customers, cu)
	}
	nullable := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	insert("INSERT INTO account (id, name, account_type_code, onboarding_status_code)", len(customers), func(i int) []any {
		return []any{customers[i].id, customers[i].name, "company", "unclaimed"}
	})
	insert(`INSERT INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code,
	        external_number, priority_code, account_group_id, default_sales_rep_id)`, len(customers), func(i int) []any {
		return []any{fmt.Sprintf("ar_planful_%06d", i), planFulAccount, customers[i].id, "customer",
			fmt.Sprintf("C%05d", i), "normal", nullable(customers[i].group), nullable(customers[i].rep)}
	})

	for _, pl := range []string{planFulLineDense, planFulLineMid, planFulLineTail, planFulLineRare, planFulLineZero} {
		exec(`INSERT INTO product_line (id, name, account_id, unit_group_id) VALUES (?, ?, ?, 'each_group')`, pl, pl, planFulAccount)
	}
	insert("INSERT INTO product (id, item_id, product_type_code, product_line_id)", planFulProducts, func(p int) []any {
		return []any{planFulProductID(p), planFulItemID(p), "sale", planFulProductLine(p)}
	})

	insert(`INSERT INTO sales_order (id, billing_address_id, shipping_address_id, number, priority_code,
	        sales_order_status_code, sales_order_type_code, buyer_account_id, seller_account_id, owner_account_id,
	        customer_po_number, sales_rep_id, created_at, updated_at)`, planFulOrders, func(i int) []any {
		var po any
		if i%10 < 7 {
			po = fmt.Sprintf("PO-%07d", 3_000_000+i*13)
		}
		status := "fulfilled"
		if planFulIsOpen(i) {
			status = "issued"
		}
		at := planFulCreatedAt(i)
		return []any{planFulID("so", i), "ad_planful_bill", "ad_planful_ship", "SO" + planFulNumber(i), "normal",
			status, "sales_order", planFulBuyer(i), planFulAccount, planFulAccount, po, planFulOrderRep(i), at, at}
	})

	type line struct{ order, product, n int }
	var lines []line
	for i := range planFulOrders {
		for j, p := range planFulLineProducts(i) {
			lines = append(lines, line{order: i, product: p, n: j})
		}
	}
	lineID := func(prefix string, k int) string { return planFulID(prefix, k) }
	insert(`INSERT INTO sales_order_line (id, product_sku, sales_order_id, quantity_id, unit_price_id, product_id,
	        item_id, line_item_number, product_description, created_at, updated_at)`, len(lines), func(k int) []any {
		l := lines[k]
		at := planFulCreatedAt(l.order)
		return []any{lineID("sol", k), fmt.Sprintf("SKU-%04d", l.product), planFulID("so", l.order), lineID("qy_sol", k),
			lineID("qy_sop", k), planFulProductID(l.product), planFulItemID(l.product), l.n + 1,
			fmt.Sprintf("Plan product %04d, case of 12", l.product), at, at}
	})

	insert("INSERT INTO pick (id, number, sales_order_id, account_id, finished_at, ship_by_sort_date, buyer_account_id, created_at, updated_at)",
		planFulOrders, func(i int) []any {
			at := planFulCreatedAt(i)
			shipBy := at.Add(time.Duration(7+i%22) * 24 * time.Hour).Format(time.DateOnly)
			if i%1000 == 999 {
				shipBy = "9999-12-31"
			}
			var finished any
			if !planFulIsOpen(i) {
				finished = at.Add(36 * time.Hour)
			}
			return []any{planFulID("pk", i), planFulNumber(i), planFulID("so", i), planFulAccount, finished, shipBy, planFulBuyer(i), at, at}
		})
	insert("INSERT INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, packed_at, created_at, updated_at)", len(lines), func(k int) []any {
		l := lines[k]
		at := planFulCreatedAt(l.order)
		var packed any
		if !planFulIsOpen(l.order) {
			packed = at.Add(30 * time.Hour)
		}
		return []any{lineID("pkl", k), planFulID("pk", l.order), lineID("qy_pkl", k), lineID("sol", k), packed, at, at}
	})

	insert(`INSERT INTO shipment (id, number, sales_order_id, carrier_id, carrier_option_id, shipping_address_id,
	        shipment_status_code, account_id, buyer_account_id, note, bill_of_lading, master_tracking_number, invoice_id,
	        shipped_by_id, shipped_at, created_at, updated_at)`, planFulOrders, func(i int) []any {
		if planFulIsOpen(i) {
			return nil
		}
		at := planFulCreatedAt(i).Add(40 * time.Hour)
		status, shipped := "shipped", any(at.Add(2*time.Hour))
		if i >= planFulOrders-planFulOpenPicks-planFulPackedShips {
			status, shipped = "packed", nil
		}
		var note, bol, mtn, shippedBy any
		if i%33 == 0 {
			note = "Deliver to dock 4 before noon, call " + planFulNumber(i)
		}
		if i%16 == 0 {
			bol = fmt.Sprintf("BOL%010d", i)
		}
		if i%5 == 0 {
			mtn = fmt.Sprintf("1Z%016d", i)
			shippedBy = "acus_planful_shipper"
		}
		return []any{planFulID("sh", i), planFulNumber(i), planFulID("so", i), "cr_planful", "co_planful", "ad_planful_ship",
			status, planFulAccount, planFulBuyer(i), note, bol, mtn, planFulID("in", i), shippedBy, shipped, at, at}
	})
	insert("INSERT INTO shipment_line (id, shipment_id, sales_order_line_id, quantity_id, created_at, updated_at)", len(lines), func(k int) []any {
		l := lines[k]
		if planFulIsOpen(l.order) {
			return nil
		}
		at := planFulCreatedAt(l.order).Add(40*time.Hour + time.Duration(l.n)*time.Second)
		return []any{lineID("shl", k), planFulID("sh", l.order), lineID("sol", k), lineID("qy_shl", k), at, at}
	})
	insert("INSERT INTO quantity (id, value, unit_id, created_at, updated_at)", len(lines), func(k int) []any {
		if planFulIsOpen(lines[k].order) {
			return nil
		}
		return []any{lineID("qy_shl", k), "12", "un_planful", planFulOrigin, planFulOrigin}
	})

	exec("ANALYZE TABLE pick, pick_line, shipment, shipment_line, sales_order, sales_order_line, product, account_relation")
}
