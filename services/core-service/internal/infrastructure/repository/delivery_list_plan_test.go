//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// The delivery corpus is the fulfillment merchant's receiving side: two deliveries against each
// purchase order, half the orders from one supplier and three from a rare one, an item on two orders
// in five and one on a couple. Production holds only 76 deliveries, so this is a tenant receiving at
// volume rather than today's shape: the list has to stay a page read as it grows.
const (
	planDlvDeliveries = 6_000
	planDlvOrders     = planDlvDeliveries / 2
	planDlvSuppliers  = 40
	planDlvRejected   = 5

	planDlvRareSupplier = planDlvSuppliers - 1
	planDlvDenseItem    = "it_planful_d000"
	planDlvRareItem     = "it_planful_d001"
	planDlvZeroItem     = "it_planful_d002"

	// planDlvSoldItem is the fulfillment corpus's densest product's item. Two purchase orders buy it by
	// that product, naming no item, and thousands of sales orders sell it: nearly all of the product's
	// order lines are sales, which the item filter must not read row by row.
	planDlvSoldProduct = "pd_planful_0000"
	planDlvSoldItem    = "it_planful_0000"
)

var planDlvSoldItemOrders = []int{900, 2400}

func planDlvSupplierID(s int) string { return fmt.Sprintf("%s_s%03d", planFulAccount, s) }

var planDlvRareSupplierOrders = map[int]bool{50: true, 1500: true, 2990: true}
var planDlvRareItemOrders = map[int]bool{70: true, 1700: true}

func planDlvSupplier(po int) string {
	switch {
	case planDlvRareSupplierOrders[po]:
		return planDlvSupplierID(planDlvRareSupplier)
	case po%2 == 0:
		return planDlvSupplierID(0)
	default:
		return planDlvSupplierID(1 + po%(planDlvSuppliers-2))
	}
}

func planDlvLineItems(po int) []string {
	first := fmt.Sprintf("it_planful_d%03d", 3+po%50)
	switch {
	case planDlvRareItemOrders[po]:
		first = planDlvRareItem
	case po%5 < 2:
		first = planDlvDenseItem
	}
	if po%2 == 1 {
		return []string{first, fmt.Sprintf("it_planful_d%03d", 3+(po*7)%50)}
	}
	return []string{first}
}

func planDlvCreatedAt(d int) time.Time {
	return planFulOrigin.Add(time.Duration(d) * (planFulSpan / planDlvDeliveries))
}

var planDlvCorpusOnce sync.Once

func ensureDeliveryCorpus(t *testing.T) {
	t.Helper()
	ensureFulfillmentCorpus(t)
	db := planDB(t)
	planDlvCorpusOnce.Do(func() {
		// Reseeding the fulfillment corpus deletes the purchase orders and leaves the deliveries.
		var have, orders int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM delivery WHERE account_id = ?", planFulAccount).Scan(&have))
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sales_order WHERE id LIKE 'po\\_planful\\_%'").Scan(&orders))
		if have < planDlvDeliveries || orders < planDlvOrders {
			t.Logf("seeding the delivery plan corpus (%d deliveries); it is kept for later runs", planDlvDeliveries)
			seedDeliveryCorpus(t, db)
		}
		seedDeliveryProductLines(t, db)
	})
}

// seedDeliveryProductLines adds the purchase order lines that order planDlvSoldProduct, with a receiving
// line each and a line on each of their orders' deliveries. It only inserts what is missing, so a corpus
// seeded before these lines existed gains them.
func seedDeliveryProductLines(t *testing.T, db *sql.DB) {
	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, po := range planDlvSoldItemOrders {
		k := 90_000 + po
		at := planDlvCreatedAt(2 * po)
		exec(`INSERT IGNORE INTO sales_order_line (id, product_sku, sales_order_id, quantity_id, unit_price_id, product_id,
		      line_item_number, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			planFulID("pol", k), "SKU-0000", planFulID("po", po), planFulID("qy_pol", k), planFulID("qy_pop", k),
			planDlvSoldProduct, 3, at, at)
		exec(`INSERT IGNORE INTO receiving_order_line (id, receiving_order_id, quantity_id, sales_order_line_id, created_at,
		      updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			planFulID("rol", k), planFulID("ro", po), planFulID("qy_rol", k), planFulID("pol", k), at, at)
		for _, d := range []int{2 * po, 2*po + 1} {
			exec(`INSERT IGNORE INTO delivery_line (id, delivery_id, receiving_order_line_id, quantity_id, unit_cost_id,
			      created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				planFulID("dll", 900_000+d), planFulID("dlv", d), planFulID("rol", k), planFulID("qy_dll", 900_000+d),
				planFulID("qy_dlc", 900_000+d), planDlvCreatedAt(d), planDlvCreatedAt(d))
		}
	}
}

func seedDeliveryCorpus(t *testing.T, db *sql.DB) {
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
				vals = append(vals, "("+placeholders(len(r))+")")
				args = append(args, r...)
			}
			exec(head+" VALUES "+strings.Join(vals, ","), args...)
		}
	}
	for _, table := range []string{"delivery_line", "delivery", "receiving_order_line", "receiving_order"} {
		exec("DELETE FROM `"+table+"` WHERE id LIKE ?", "%\\_planful\\_%")
	}
	exec("DELETE FROM sales_order_line WHERE id LIKE 'pol\\_planful\\_%'")
	exec("DELETE FROM sales_order WHERE id LIKE 'po\\_planful\\_%'")
	exec("DELETE FROM account WHERE id LIKE 'ac\\_planful\\_s%'")

	insert("INSERT INTO account (id, name, account_type_code, onboarding_status_code)", planDlvSuppliers, func(s int) []any {
		return []any{planDlvSupplierID(s), fmt.Sprintf("Plan Supplier %03d", s), "company", "unclaimed"}
	})
	insert(`INSERT INTO sales_order (id, billing_address_id, shipping_address_id, number, priority_code,
	        sales_order_status_code, sales_order_type_code, buyer_account_id, seller_account_id, owner_account_id,
	        created_at, updated_at)`, planDlvOrders, func(po int) []any {
		at := planDlvCreatedAt(2 * po).Add(-72 * time.Hour)
		return []any{planFulID("po", po), "ad_planful_bill", "ad_planful_ship", fmt.Sprintf("PO%05d", 20_000+po), "normal",
			"issued", "purchase_order", planFulAccount, planDlvSupplier(po), planFulAccount, at, at}
	})
	insert("INSERT INTO receiving_order (id, number, order_id, account_id, created_at, updated_at)", planDlvOrders, func(po int) []any {
		at := planDlvCreatedAt(2 * po)
		return []any{planFulID("ro", po), fmt.Sprintf("RO%05d", po), planFulID("po", po), planFulAccount, at, at}
	})

	type line struct {
		po, n int
		item  string
	}
	var lines []line
	for po := range planDlvOrders {
		for n, item := range planDlvLineItems(po) {
			lines = append(lines, line{po: po, n: n, item: item})
		}
	}
	insert(`INSERT INTO sales_order_line (id, product_sku, sales_order_id, quantity_id, unit_price_id, item_id,
	        line_item_number, created_at, updated_at)`, len(lines), func(k int) []any {
		l := lines[k]
		at := planDlvCreatedAt(2 * l.po)
		return []any{planFulID("pol", k), "RAW-" + l.item, planFulID("po", l.po), planFulID("qy_pol", k), planFulID("qy_pop", k),
			l.item, l.n + 1, at, at}
	})
	insert("INSERT INTO receiving_order_line (id, receiving_order_id, quantity_id, sales_order_line_id, created_at, updated_at)", len(lines), func(k int) []any {
		at := planDlvCreatedAt(2 * lines[k].po)
		return []any{planFulID("rol", k), planFulID("ro", lines[k].po), planFulID("qy_rol", k), planFulID("pol", k), at, at}
	})
	insert(`INSERT INTO delivery (id, number, sales_order_id, account_id, delivery_status_code, accepted_at, rejected_at,
	        created_at, updated_at)`, planDlvDeliveries, func(d int) []any {
		at := planDlvCreatedAt(d)
		status, accepted, rejected := "accepted", any(at), any(nil)
		if d%1200 == 600 && d/1200 < planDlvRejected {
			status, accepted, rejected = "rejected", nil, at
		}
		return []any{planFulID("dlv", d), fmt.Sprintf("DLV%05d", d), planFulID("po", d/2), planFulAccount, status, accepted, rejected, at, at}
	})
	var dLines [][2]int // delivery, line
	for d := range planDlvDeliveries {
		for k, l := range lines {
			if l.po == d/2 {
				dLines = append(dLines, [2]int{d, k})
			}
			if l.po > d/2 {
				break
			}
		}
	}
	insert("INSERT INTO delivery_line (id, delivery_id, receiving_order_line_id, quantity_id, unit_cost_id, created_at, updated_at)", len(dLines), func(i int) []any {
		d, k := dLines[i][0], dLines[i][1]
		at := planDlvCreatedAt(d)
		return []any{planFulID("dll", i), planFulID("dlv", d), planFulID("rol", k), planFulID("qy_dll", i), planFulID("qy_dlc", i), at, at}
	})
	exec("ANALYZE TABLE delivery, delivery_line, receiving_order_line, sales_order, sales_order_line")
}

func deliveryPlanDims() []planDim[domain.ListDeliveriesParams] {
	str := func(s string) *string { return &s }
	at := func(d time.Time) *time.Time { return &d }
	recent := planDlvCreatedAt(planDlvDeliveries - 1)
	old := planDlvCreatedAt(planDlvDeliveries / 4)
	return []planDim[domain.ListDeliveriesParams]{
		{"status", []planValue[domain.ListDeliveriesParams]{
			{"accepted", func(p *domain.ListDeliveriesParams) { p.Status = str("accepted") }},
			{"rejected", func(p *domain.ListDeliveriesParams) { p.Status = str("rejected") }},
		}},
		{"supplier", []planValue[domain.ListDeliveriesParams]{
			{"large", func(p *domain.ListDeliveriesParams) { p.SupplierIDs = []string{planDlvSupplierID(0)} }},
			{"rare", func(p *domain.ListDeliveriesParams) { p.SupplierIDs = []string{planDlvSupplierID(planDlvRareSupplier)} }},
			{"two", func(p *domain.ListDeliveriesParams) {
				p.SupplierIDs = []string{planDlvSupplierID(1), planDlvSupplierID(planDlvRareSupplier)}
			}},
		}},
		{"item", []planValue[domain.ListDeliveriesParams]{
			{"dense", func(p *domain.ListDeliveriesParams) { p.ItemIDs = []string{planDlvDenseItem} }},
			{"rare", func(p *domain.ListDeliveriesParams) { p.ItemIDs = []string{planDlvRareItem} }},
			{"zero", func(p *domain.ListDeliveriesParams) { p.ItemIDs = []string{planDlvZeroItem} }},
			{"sold", func(p *domain.ListDeliveriesParams) { p.ItemIDs = []string{planDlvSoldItem} }},
		}},
		{"created", []planValue[domain.ListDeliveriesParams]{
			{"last30d", func(p *domain.ListDeliveriesParams) { p.StartDate = at(recent.Add(-30 * 24 * time.Hour)) }},
			{"old30d", func(p *domain.ListDeliveriesParams) { p.StartDate, p.EndDate = at(old), at(old.Add(30*24*time.Hour)) }},
		}},
	}
}

func deliveryPlanCases() []planCase[domain.ListDeliveriesParams] {
	mid := planDlvCreatedAt(planDlvDeliveries / 2)
	cursor := func(dir pagination.Direction) func(*domain.ListDeliveriesParams) {
		return func(p *domain.ListDeliveriesParams) {
			c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "dlv_planful_~", Direction: dir})
			p.Cursor = &c
		}
	}
	return planCases(
		domain.ListDeliveriesParams{AccountID: planFulAccount, Limit: 25},
		qualifiedPlanDims(deliveryPlanDims()),
		[]planValue[domain.ListDeliveriesParams]{
			{"first", func(*domain.ListDeliveriesParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// deliveryUnorderedFloor is how many deliveries the request's narrowest unordered filter matches, or 0
// when it has none. No key yields these in list order: suppliers (on the order, a joined table) and
// items (child tables). The best any plan can do is read one filter's matches.
func deliveryUnorderedFloor(t *testing.T, db *sql.DB, p domain.ListDeliveriesParams) float64 {
	t.Helper()
	var floors []float64
	count := func(matched string, args []any) {
		var n float64
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM delivery d WHERE d.account_id = ? AND d.id IN ("+matched+")",
			append([]any{p.AccountID}, args...)...).Scan(&n))
		floors = append(floors, n)
	}
	if len(p.SupplierIDs) > 0 {
		count(deliveriesFromSuppliers(p.SupplierIDs))
	}
	if len(p.ItemIDs) > 0 {
		count(deliveriesWithItems(p.ItemIDs))
	}
	if len(floors) == 0 {
		return 0
	}
	floor := floors[0]
	for _, f := range floors[1:] {
		floor = min(floor, f)
	}
	return max(floor, 1)
}

// TestDeliveryList_ReadsAboutAPage holds every filter combination ListDeliveries accepts to reading
// about a page of deliveries (listPlanSuite). Analyzed statistics only: production's describe its 76
// deliveries, under which scanning the account in order is the right plan, not this corpus's 6,000.
// Snapshot production's into testdata/plan_stats/delivery.json and add the mode once it holds thousands.
func TestDeliveryList_ReadsAboutAPage(t *testing.T) {
	ensureDeliveryCorpus(t)
	listPlanSuite[domain.ListDeliveriesParams]{
		statsModes: []string{"analyzed"},
		table:      "delivery", scopeColumn: "account_id",
		from: "FROM delivery d", alias: "d",
		statement: func(query string) bool { return strings.HasPrefix(query, "SELECT STRAIGHT_JOIN d.id") },
		cases:     deliveryPlanCases(),
		limit:     func(p domain.ListDeliveriesParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListDeliveriesParams) error {
			if _, apiErr := NewDeliveryRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: deliveryUnorderedFloor,
		// An item filter that drives reads each order line of the item, and of its products, once and from
		// their keys: an item sold on thousands of orders is read through, never scanned for.
		reads: []planRead[domain.ListDeliveriesParams]{
			{alias: "sol3", matches: planDlvItemOrderLines("sales_order_line sol WHERE sol.item_id")},
			{alias: "sol4", matches: planDlvItemOrderLines("product p JOIN sales_order_line sol ON sol.product_id = p.id WHERE p.item_id")},
		},
	}.run(t)
}

// planDlvItemOrderLines counts the order lines the request's item filter reaches through from.
func planDlvItemOrderLines(from string) func(*testing.T, *sql.DB, domain.ListDeliveriesParams) float64 {
	return func(t *testing.T, db *sql.DB, p domain.ListDeliveriesParams) float64 {
		var n float64
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+from+" IN ("+placeholders(len(p.ItemIDs))+")",
			stringArgs(p.ItemIDs)...).Scan(&n))
		return n
	}
}
