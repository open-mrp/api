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

// The catalog corpus is one merchant shaped like the largest production tenant's catalog and customer
// book: thousands of items (a third of them sale products, most of the rest parts, a few materials),
// about four attributes on each, a few dozen product lines, thousands of customers with product-line
// access through groups, price groups and direct grants, one customer with a long order history and
// another with a thousand addresses. Every filter the catalog and customer lists accept gets a dense
// value and a rare one, and row widths follow production (12-character SKUs, 44-character
// descriptions, ID lengths), for the reasons the transaction corpus gives.
const (
	planCatAccount = "ac_plancat"

	planCatItems      = 6_400
	planCatCategories = 27
	planCatLines      = 25
	planCatAttributes = 91
	planCatSuppliers  = 17
	planCatCustomers  = 3_500
	planCatGroups     = 16
	planCatReps       = 8
	planCatAddresses  = 1_200
	planCatOrders     = 4_000
	planCatSpan       = 4 * 365 * 24 * time.Hour

	// planCatCorpusVersion is bumped whenever the corpus's shape changes, so a stale one is rebuilt.
	planCatCorpusVersion = "Plan Catalog Merchant v3"

	// planCatRareSearch matches one SKU exactly; planCatDenseSearch is a substring of every SKU.
	planCatRareSearch  = "PLC-00001234"
	planCatDenseSearch = "PLC-0000"
)

var planCatOrigin = time.Date(2022, 10, 1, 0, 0, 0, 0, time.UTC)

func planCatItemID(i int) string        { return fmt.Sprintf("it_plancat_%016d", i) }
func planCatProductID(i int) string     { return fmt.Sprintf("pr_plancat_%016d", i) }
func planCatSKU(i int) string           { return fmt.Sprintf("PLC-%08d", i) }
func planCatCategoryID(c int) string    { return fmt.Sprintf("itcg_plancat_%04d", c) }
func planCatLineID(l int) string        { return fmt.Sprintf("prln_plancat_%04d", l) }
func planCatAttributeID(a int) string   { return fmt.Sprintf("attr_plancat_%04d", a) }
func planCatSupplierID(s int) string    { return fmt.Sprintf("%s_s%016d", planCatAccount, s) }
func planCatCustomerID(c int) string    { return fmt.Sprintf("%s_c%016d", planCatAccount, c) }
func planCatRelationID(c int) string    { return fmt.Sprintf("ar_plancat_%016d", c) }
func planCatGroupID(g int) string       { return fmt.Sprintf("ag_plancat_%04d", g) }
func planCatRepID(r int) string         { return fmt.Sprintf("acus_plancat_%012d", r) }
func planCatAddressID(a int) string     { return fmt.Sprintf("ad_plancat_%016d", a) }
func planCatGeolocationID(a int) string { return fmt.Sprintf("geo_plancat_%016d", a) }
func planCatOrderID(o int) string       { return fmt.Sprintf("so_plancat_%016d", o) }

func planCatItemCreatedAt(i int) time.Time {
	return planCatOrigin.Add(time.Duration(i) * (planCatSpan / planCatItems))
}

// planCatItemType makes a third of the items products, a sixteenth materials, and the rest parts.
func planCatItemType(i int) string {
	switch {
	case i%100 < 32:
		return "product"
	case i%100 < 38:
		return "material"
	default:
		return "part"
	}
}

// planCatRareItems hold the rare category, attribute, and product line.
var planCatRareItems = map[int]bool{1_200: true, 3_300: true, 6_300: true}

func planCatItemCategory(i int) int {
	if planCatRareItems[i] {
		return planCatCategories - 1
	}
	switch planCatItemType(i) {
	case "product":
		return (i / 100) % 10
	case "material":
		return 24 + i%2
	default:
		if i%2 == 0 {
			return 10
		}
		return 11 + i%13
	}
}

// planCatItemLine puts three in ten products on line 0 and spreads the rest; two have no line.
func planCatItemLine(i int) *string {
	if planCatRareItems[i] {
		id := planCatLineID(planCatLines - 1)
		return &id
	}
	if i == 5 || i == 6 {
		return nil
	}
	l := 0
	if (i/100)%10 >= 3 {
		l = 1 + (i*7)%(planCatLines-2)
	}
	id := planCatLineID(l)
	return &id
}

// planCatItemAttributes gives every other item attribute 0, every item three of the spread ones, and
// the rare items the last.
func planCatItemAttributes(i int) []int {
	set := map[int]bool{1 + i%89: true, 1 + (i*7)%89: true, 1 + (i*13)%89: true}
	if i%2 == 0 {
		set[0] = true
	}
	if planCatRareItems[i] {
		set[planCatAttributes-1] = true
	}
	out := make([]int, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	return out
}

// Customers: group 0 holds half of them and group 15 one; rep 0 a few hundred and rep 7 one; nearly all
// are in normal standing; every tenth is a child of customer 0.
func planCatCustomerGroup(c int) int {
	switch {
	case c == planCatCustomers-1:
		return planCatGroups - 1
	case c%2 == 0:
		return 0
	default:
		return 1 + c%(planCatGroups-2)
	}
}

var planCatCorpusOnce sync.Once

func ensureCatalogCorpus(t *testing.T) {
	t.Helper()
	db := planDB(t)
	planCatCorpusOnce.Do(func() {
		var have int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM item WHERE account_id = ?", planCatAccount).Scan(&have))
		var version string
		_ = db.QueryRow("SELECT name FROM account WHERE id = ?", planCatAccount).Scan(&version)
		if have >= planCatItems && version == planCatCorpusVersion {
			return
		}
		t.Logf("seeding the catalog plan corpus (%d items, %d customers); it is kept for later runs", planCatItems, planCatCustomers)
		seedCatalogCorpus(t, db)
	})
}

// planInserter batches multi-row INSERTs.
type planInserter struct {
	t    *testing.T
	db   *sql.DB
	head string
	row  string
	vals []string
	args []any
}

func (p *planInserter) add(args ...any) {
	p.vals = append(p.vals, p.row)
	p.args = append(p.args, args...)
	if len(p.vals) == 500 {
		p.flush()
	}
}

func (p *planInserter) flush() {
	if len(p.vals) == 0 {
		return
	}
	_, err := p.db.Exec(p.head+" VALUES "+strings.Join(p.vals, ","), p.args...)
	require.NoError(p.t, err)
	p.vals, p.args = nil, nil
}

func seedCatalogCorpus(t *testing.T, db *sql.DB) {
	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	ins := func(head, row string) *planInserter { return &planInserter{t: t, db: db, head: head, row: row} }

	for _, stmt := range []string{
		"DELETE FROM sales_order_line WHERE id LIKE 'sol\\_plancat\\_%'",
		"DELETE FROM sales_order WHERE owner_account_id = 'ac_plancat'",
		"DELETE FROM account_address WHERE id LIKE 'aa\\_plancat\\_%'",
		"DELETE FROM address WHERE id LIKE 'ad\\_plancat\\_%'",
		"DELETE FROM geolocation WHERE id LIKE 'geo\\_plancat\\_%'",
		"DELETE FROM account_relation_product_line WHERE id LIKE 'arpl\\_plancat\\_%'",
		"DELETE FROM account_relation_price_group WHERE id LIKE 'arpg\\_plancat\\_%'",
		"DELETE FROM account_group_product_line WHERE id LIKE 'agpl\\_plancat\\_%'",
		"DELETE FROM account_relation WHERE owner_account_id = 'ac_plancat'",
		"DELETE FROM account_group WHERE owner_account_id = 'ac_plancat'",
		"DELETE FROM _parent_child_production_steps WHERE A LIKE 'ps\\_plancat\\_%'",
		"DELETE FROM production WHERE id LIKE 'prd\\_plancat\\_%'",
		"DELETE FROM supplier_material WHERE owner_account_id = 'ac_plancat'",
		"DELETE FROM material WHERE id LIKE 'mat\\_plancat\\_%'",
		"DELETE FROM _item_attributes WHERE B LIKE 'it\\_plancat\\_%'",
		"DELETE FROM attribute WHERE account_id = 'ac_plancat'",
		"DELETE FROM property WHERE account_id = 'ac_plancat'",
		"DELETE FROM product WHERE id LIKE 'pr\\_plancat\\_%'",
		"DELETE FROM rate WHERE id LIKE 'rt\\_plancat\\_%'",
		"DELETE FROM item WHERE account_id = 'ac_plancat'",
		"DELETE FROM product_line WHERE account_id = 'ac_plancat'",
		"DELETE FROM item_category WHERE account_id = 'ac_plancat'",
		"DELETE FROM account_user WHERE account_id = 'ac_plancat'",
		"DELETE FROM account WHERE id LIKE 'ac\\_plancat\\_%'",
	} {
		exec(stmt)
	}

	exec(`INSERT INTO account (id, name, account_type_code, onboarding_status_code) VALUES (?, ?, 'company', 'active')
	      ON DUPLICATE KEY UPDATE name = VALUES(name)`, planCatAccount, "seeding")
	exec(`INSERT IGNORE INTO unit_group (id, name, base_unit_id, unit_type_code) VALUES ('ungp_plancat', 'plantest', 'un_plancat', 'quantity')`)
	exec(`INSERT IGNORE INTO property (id, name, account_id) VALUES ('prop_plancat', 'Plan property', ?)`, planCatAccount)

	cats := ins("INSERT INTO item_category (id, name, item_category_type_code, unit_group_id, account_id)", "(?, ?, ?, 'ungp_plancat', ?)")
	for c := range planCatCategories {
		kind := "product_category"
		if c >= 10 {
			kind = "material_category"
		}
		cats.add(planCatCategoryID(c), fmt.Sprintf("Plan Category %02d", c), kind, planCatAccount)
	}
	cats.flush()

	lines := ins("INSERT INTO product_line (id, name, unit_group_id, account_id, created_at)", "(?, ?, 'ungp_plancat', ?, ?)")
	for l := range planCatLines {
		lines.add(planCatLineID(l), fmt.Sprintf("Plan Line %02d", l), planCatAccount, planCatOrigin.Add(time.Duration(l)*time.Hour))
	}
	lines.flush()

	attrs := ins("INSERT INTO attribute (id, text, property_id, color_code, account_id)", "(?, ?, 'prop_plancat', 'gray', ?)")
	for a := range planCatAttributes {
		attrs.add(planCatAttributeID(a), fmt.Sprintf("Plan Attribute %02d", a), planCatAccount)
	}
	attrs.flush()

	items := ins("INSERT INTO item (id, sku, description, created_at, updated_at, deleted_at, unit_value_id, burn_rate_id, account_id, item_type_code, unit_cost_id, item_category_id)",
		"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	products := ins("INSERT INTO product (id, created_at, updated_at, item_id, product_type_code, product_line_id, is_portal_ready)", "(?, ?, ?, ?, ?, ?, ?)")
	itemAttrs := ins("INSERT INTO _item_attributes (A, B)", "(?, ?)")
	materials := ins("INSERT INTO material (id, item_id, order_point_id, lead_time_id)", "(?, ?, ?, ?)")
	productions := ins("INSERT INTO production (id, item_id, quantity_id, production_step_id)", "(?, ?, ?, ?)")
	rates := ins("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id)", "(?, '12.50', 'un_plancat', 'un_plancat')")
	steps := ins("INSERT INTO _parent_child_production_steps (A, B)", "(?, ?)")
	var materialItems []int
	for i := range planCatItems {
		createdAt := planCatItemCreatedAt(i)
		var deletedAt any
		if i%50 == 49 {
			deletedAt = createdAt.Add(24 * time.Hour)
		}
		itemType := planCatItemType(i)
		items.add(planCatItemID(i), planCatSKU(i), fmt.Sprintf("Plan catalog item %06d in a production tenant", i),
			createdAt, createdAt, deletedAt,
			fmt.Sprintf("rt_plancat_v%014d", i), fmt.Sprintf("rt_plancat_b%014d", i), planCatAccount, itemType,
			fmt.Sprintf("rt_plancat_c%014d", i), planCatCategoryID(planCatItemCategory(i)))
		rates.add(fmt.Sprintf("rt_plancat_v%014d", i))
		for _, a := range planCatItemAttributes(i) {
			itemAttrs.add(planCatAttributeID(a), planCatItemID(i))
		}
		switch itemType {
		case "product":
			productType := "sale"
			if i%1000 == 1 {
				productType = "service"
			}
			products.add(planCatProductID(i), createdAt, createdAt, planCatItemID(i), productType, planCatItemLine(i), i%7 != 0)
		case "material":
			materialItems = append(materialItems, i)
			materials.add(fmt.Sprintf("mat_plancat_%016d", i), planCatItemID(i), fmt.Sprintf("op_plancat_%014d", i), fmt.Sprintf("lt_plancat_%014d", i))
		}
		if itemType != "material" {
			step := fmt.Sprintf("ps_plancat_%016d", i)
			productions.add(fmt.Sprintf("prd_plancat_%016d", i), planCatItemID(i), fmt.Sprintf("qy_plancat_p%014d", i), step)
			// Nine in ten steps have an upstream step; the rest are where a flow starts.
			if i%10 != 0 {
				steps.add(step, fmt.Sprintf("ps_plancat_%016d", i-1))
			}
		}
	}
	for _, p := range []*planInserter{items, products, itemAttrs, materials, productions, steps, rates} {
		p.flush()
	}

	// Supplier 0 supplies forty materials, supplier 16 one, and the rest a couple each.
	accounts := ins("INSERT INTO account (id, name, account_type_code, onboarding_status_code)", "(?, ?, 'company', 'unclaimed')")
	supplied := ins("INSERT INTO supplier_material (id, material_id, supplier_account_id, supplier_part_number, owner_account_id)", "(?, ?, ?, ?, ?)")
	for k, i := range materialItems[:73] {
		s := 1 + k%(planCatSuppliers-2)
		switch {
		case k < 40:
			s = 0
		case k == 72:
			s = planCatSuppliers - 1
		}
		supplied.add(fmt.Sprintf("sm_plancat_%016d", i), fmt.Sprintf("mat_plancat_%016d", i), planCatSupplierID(s), fmt.Sprintf("SP-%06d", i), planCatAccount)
	}
	for s := range planCatSuppliers {
		accounts.add(planCatSupplierID(s), fmt.Sprintf("Plan Supplier %02d", s))
	}
	supplied.flush()

	// Groups grant product lines: group 0 the dense line and five more, group 15 only the rare line.
	exec(`INSERT INTO account_user (id, user_id, account_id) VALUES `+strings.TrimSuffix(strings.Repeat("(?, ?, ?),", planCatReps), ","),
		func() []any {
			var a []any
			for r := range planCatReps {
				a = append(a, planCatRepID(r), fmt.Sprintf("us_plancat_%04d", r), planCatAccount)
			}
			return a
		}()...)
	groups := ins("INSERT INTO account_group (id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code)",
		"(?, ?, ?, 'commission_applied', 'billed_freight', ?)")
	groupLines := ins("INSERT INTO account_group_product_line (id, account_group_id, product_line_id)", "(?, ?, ?)")
	for g := range planCatGroups {
		kind := "type_group"
		if g >= 12 {
			kind = "price_group"
		}
		groups.add(planCatGroupID(g), planCatAccount, fmt.Sprintf("Plan Group %02d", g), kind)
		switch g {
		case planCatGroups - 1:
			groupLines.add(fmt.Sprintf("agpl_plancat_%04d_%04d", g, planCatLines-1), planCatGroupID(g), planCatLineID(planCatLines-1))
		default:
			for k := range 6 {
				l := (g + k*4) % (planCatLines - 1)
				if g == 0 && k == 0 {
					l = 0
				}
				groupLines.add(fmt.Sprintf("agpl_plancat_%04d_%04d", g, l), planCatGroupID(g), planCatLineID(l))
			}
		}
	}
	groups.flush()
	groupLines.flush()

	relations := ins(`INSERT INTO account_relation (id, owner_account_id, counterparty_account_id, account_relation_role_code, external_number,
		priority_code, account_group_id, account_status_code, default_sales_rep_id, payment_term_id, shipping_term_id,
		commission_status_code, freight_status_code, default_carrier_id, default_carrier_option_id, parent_account_relation_id,
		alias, notes, created_at, updated_at)`, "(?, ?, ?, 'customer', ?, 'normal', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	relLines := ins("INSERT INTO account_relation_product_line (id, account_relation_id, product_line_id)", "(?, ?, ?)")
	priceGroups := ins("INSERT INTO account_relation_price_group (id, account_relation_id, account_group_id)", "(?, ?, ?)")
	for c := range planCatCustomers {
		createdAt := planCatOrigin.Add(time.Duration(c) * (planCatSpan / planCatCustomers))
		status := "normal"
		switch {
		case c == planCatCustomers-2:
			status = "hold_shipment"
		case c%27 == 0:
			status = "hold_all"
		}
		var rep, parent, carrier, option any
		switch {
		case c == planCatCustomers-1:
			rep = planCatRepID(planCatReps - 1)
		case c%15 == 0:
			rep = planCatRepID(c % (planCatReps - 1))
		}
		if c%10 == 5 {
			parent = planCatRelationID(0)
		}
		if c%3 == 0 {
			carrier, option = fmt.Sprintf("carr_plancat_%02d", c%4), fmt.Sprintf("caop_plancat_%02d", c%4)
		}
		commission, freight := "commission_applied", "billed_freight"
		if c == planCatCustomers-1 {
			commission, freight = "commission_exempt", "free_freight"
		}
		accounts.add(planCatCustomerID(c), fmt.Sprintf("Plan Customer %04d Incorporated", c))
		relations.add(planCatRelationID(c), planCatAccount, planCatCustomerID(c), fmt.Sprintf("C%05d", c),
			planCatGroupID(planCatCustomerGroup(c)), status, rep,
			fmt.Sprintf("pt_plancat_%02d", c%5+boolInt(c == planCatCustomers-1)*10), fmt.Sprintf("st_plancat_%02d", c%6+boolInt(c == planCatCustomers-1)*10),
			commission, freight, carrier, option, parent,
			fmt.Sprintf("Plan Customer %04d", c), fmt.Sprintf("notes %04d", c), createdAt, createdAt)
		if c%200 == 7 {
			relLines.add(fmt.Sprintf("arpl_plancat_%016d", c), planCatRelationID(c), planCatLineID(planCatLines-1))
		}
		if c%26 == 3 {
			priceGroups.add(fmt.Sprintf("arpg_plancat_%016d", c), planCatRelationID(c), planCatGroupID(12+c%3))
		}
	}
	accounts.flush()
	relations.flush()
	relLines.flush()
	priceGroups.flush()

	// Customer 0 holds the address book; one address in a hundred is a drop ship. Customers 1 to 1000
	// hold five each, so address and geolocation hold far more than any one account's.
	geos := ins("INSERT INTO geolocation (id, street_line_1, locality, state, postal_code, country)", "(?, ?, ?, ?, ?, 'US')")
	addrs := ins("INSERT INTO address (id, name, phone, email, is_drop_ship, geolocation_id, created_at, updated_at)", "(?, ?, ?, ?, ?, ?, ?, ?)")
	links := ins("INSERT INTO account_address (id, account_id, address_id, created_at)", "(?, ?, ?, ?)")
	for a := range planCatAddresses {
		createdAt := planCatOrigin.Add(time.Duration(a) * (planCatSpan / planCatAddresses))
		state := "NC"
		if a == planCatAddresses-1 {
			state = "AK"
		}
		geos.add(planCatGeolocationID(a), fmt.Sprintf("%d Plan Street", 100+a), fmt.Sprintf("Plan City %02d", a%40), state, fmt.Sprintf("2%04d", a%500))
		addrs.add(planCatAddressID(a), fmt.Sprintf("Plan Ship To %04d", a), "555-0100", fmt.Sprintf("dock%04d@plan.test", a),
			a%100 == 50, planCatGeolocationID(a), createdAt, createdAt)
		links.add(fmt.Sprintf("aa_plancat_%016d", a), planCatCustomerID(0), planCatAddressID(a), createdAt.Add(time.Second))
	}
	for c := 1; c <= 1000; c++ {
		for k := range 5 {
			a := planCatAddresses + (c-1)*5 + k
			createdAt := planCatOrigin.Add(time.Duration(a) * time.Hour)
			geos.add(planCatGeolocationID(a), fmt.Sprintf("%d Other Street", a), "Other City", "VA", "22000")
			addrs.add(planCatAddressID(a), fmt.Sprintf("Other Ship To %05d", a), "555-0101", nil, k == 0, planCatGeolocationID(a), createdAt, createdAt)
			links.add(fmt.Sprintf("aa_plancat_%016d", a), planCatCustomerID(c), planCatAddressID(a), createdAt)
		}
	}
	geos.flush()
	addrs.flush()
	links.flush()

	// Customer 0 has ordered for years: about three lines an order, mostly from a few dozen products.
	orders := ins(`INSERT INTO sales_order (id, billing_address_id, shipping_address_id, number, priority_code, sales_order_status_code,
		sales_order_type_code, buyer_account_id, seller_account_id, owner_account_id, created_at, updated_at)`,
		"(?, ?, ?, ?, 'normal', 'completed', 'standard', ?, ?, ?, ?, ?)")
	orderLines := ins("INSERT INTO sales_order_line (id, product_sku, product_id, item_id, sales_order_id, quantity_id, unit_price_id, created_at, updated_at)",
		"(?, ?, ?, ?, ?, ?, ?, ?, ?)")
	productItems := []int{}
	for i := range planCatItems {
		if planCatItemType(i) == "product" {
			productItems = append(productItems, i)
		}
	}
	for o := range planCatOrders {
		createdAt := planCatOrigin.Add(time.Duration(o) * (planCatSpan / planCatOrders))
		buyer := planCatCustomerID(0)
		if o%4 == 3 {
			buyer = planCatCustomerID(1 + o%50)
		}
		orders.add(planCatOrderID(o), planCatAddressID(0), planCatAddressID(0), fmt.Sprintf("PSO-%06d", o), buyer, planCatAccount, planCatAccount, createdAt, createdAt)
		for l := range 1 + o%5 {
			i := productItems[(o*3+l*l*17)%60]
			if l == 4 {
				i = productItems[(o*31)%len(productItems)]
			}
			id := fmt.Sprintf("sol_plancat_%012d_%d", o, l)
			orderLines.add(id, planCatSKU(i), planCatProductID(i), planCatItemID(i), planCatOrderID(o), "qy_"+id, "rt_"+id, createdAt, createdAt)
		}
	}
	orders.flush()
	orderLines.flush()

	exec("ANALYZE TABLE item, product, product_line, _item_attributes, account_relation, account_address, address, sales_order, sales_order_line")
	// Marked last, so a seed that failed partway is rebuilt.
	exec("UPDATE account SET name = ? WHERE id = ?", planCatCorpusVersion, planCatAccount)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
