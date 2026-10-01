//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The delivery corpus gives the sales corpus's orders what delivery performance reads: a commitment
// (ship-by date) on all but a few, the order's lines and what was packed of them, the products those
// lines sell, and their product lines. As in production nearly every order is fulfilled, a sliver
// carry no commitment, and the last few are still open.
const planDeliveryPerfCorpusVersion = "Plan Delivery v1"

var planDeliveryPerfOnce sync.Once

func ensureDeliveryPerformanceCorpus(t *testing.T) {
	t.Helper()
	ensureAnalyticsInvoices(t)
	db := planDB(t)
	planDeliveryPerfOnce.Do(func() {
		var version string
		_ = db.QueryRow("SELECT name FROM account_group WHERE id = 'ag_planana_delivery'").Scan(&version)
		if version == planDeliveryPerfCorpusVersion {
			return
		}
		t.Log("seeding the delivery plan corpus; it is kept for later runs")
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM pick_line WHERE id LIKE 'pkl\\_planana\\_%'")
		exec("DELETE FROM pick WHERE account_id = ?", planAnaAccount)
		exec("DELETE FROM sales_order_line WHERE id LIKE 'sol\\_planana\\_%'")
		// Only this corpus's quantities: the line-level corpus keeps its invoice lines' beside them.
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_planana\\_s%' OR id LIKE 'qy\\_planana\\_p%'")
		exec("DELETE FROM product WHERE id LIKE 'pd\\_planana\\_%'")
		exec("DELETE FROM product_line WHERE id LIKE 'pl\\_planana\\_%'")
		exec(`INSERT IGNORE INTO unit (id, name, abbreviation, unit_dimension_code, account_id,
		                               ratio_numerator, ratio_denominator, is_base_unit, created_at, updated_at)
		      VALUES ('un_planana', 'plantest', 'un_planana', 'quantity', NULL, 1, 1, 0, NOW(3), NOW(3))`)
		for l := range planAnaLines {
			exec("INSERT INTO product_line (id, name, unit_group_id, account_id) VALUES (?, ?, 'ug_planana', ?)", planAnaLineID(l), fmt.Sprintf("Plan Line %02d", l), planAnaAccount)
		}
		var prodVals []string
		var prodArgs []any
		for item := range planAnaItems {
			prodVals = append(prodVals, "(?, ?, 'sale', ?)")
			prodArgs = append(prodArgs, fmt.Sprintf("pd_planana_%05d", item), planAnaItemID(item), planAnaLineID(planAnaItemLine(item)))
		}
		exec("INSERT INTO product (id, item_id, product_type_code, product_line_id) VALUES "+strings.Join(prodVals, ","), prodArgs...)

		const batch = 500
		for start := 0; start < planAnaInvoices; start += batch {
			var qVals, solVals, pickVals, plVals []string
			var qArgs, solArgs, pickArgs, plArgs []any
			for i := start; i < start+batch; i++ {
				at := planAnaInvoicedAt(i)
				open := i >= planAnaInvoices-30
				var shipBy any = at.Add(-24 * time.Hour).Format("2006-01-02")
				if i%250 == 3 {
					shipBy = nil
				}
				status := "fulfilled"
				if open {
					status = "issued"
				}
				exec(`UPDATE sales_order SET ship_by_date = ?, issued_at = ?, sales_rep_id = ?, sales_order_status_code = ?, first_ship_at = ?,
				      lead_time_days = 14, lead_time_source_code = 'customer' WHERE id = ?`,
					shipBy, at.Add(-72*time.Hour), planAnaRep(i), status, at, fmt.Sprintf("or_planana_%07d", i))
				pickID := fmt.Sprintf("pk_planana_%07d", i)
				pickVals = append(pickVals, "(?, ?, ?, ?)")
				pickArgs = append(pickArgs, pickID, fmt.Sprintf("PK-%06d", i), fmt.Sprintf("or_planana_%07d", i), planAnaAccount)
				for line := range 1 + planHash(i, 3)%6 {
					item := planAnaItem(i, line)
					qty := 1 + planHash(i*8+line, 5)%50
					solQ, pickQ := fmt.Sprintf("qy_planana_s%07d_%d", i, line), fmt.Sprintf("qy_planana_p%07d_%d", i, line)
					qVals = append(qVals, "(?, ?, 'un_planana', ?, ?)", "(?, ?, 'un_planana', ?, ?)")
					qArgs = append(qArgs, solQ, qty, at, at, pickQ, qty, at, at)
					solID := fmt.Sprintf("sol_planana_%07d_%d", i, line)
					solVals = append(solVals, "(?, ?, ?, ?, ?, ?, ?)")
					solArgs = append(solArgs, solID, fmt.Sprintf("SKU-%05d", item), fmt.Sprintf("or_planana_%07d", i), solQ,
						fmt.Sprintf("rt_planana_%07d_%d", i, line), fmt.Sprintf("pd_planana_%05d", item), planAnaItemID(item))
					var packed any
					if !open {
						packed = at.Add(-time.Hour)
					}
					plVals = append(plVals, "(?, ?, ?, ?, ?)")
					plArgs = append(plArgs, fmt.Sprintf("pkl_planana_%07d_%d", i, line), pickID, pickQ, solID, packed)
				}
			}
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES "+strings.Join(qVals, ","), qArgs...)
			exec("INSERT INTO sales_order_line (id, product_sku, sales_order_id, quantity_id, unit_price_id, product_id, item_id) VALUES "+strings.Join(solVals, ","), solArgs...)
			exec("INSERT INTO pick (id, number, sales_order_id, account_id) VALUES "+strings.Join(pickVals, ","), pickArgs...)
			exec("INSERT INTO pick_line (id, pick_id, quantity_id, sales_order_line_id, packed_at) VALUES "+strings.Join(plVals, ","), plArgs...)
		}
		exec(`INSERT INTO account_group (id, owner_account_id, name, commission_status_code, freight_status_code, account_group_type_code)
		      VALUES ('ag_planana_delivery', ?, ?, 'commission_applied', 'billed_freight', 'type_group')`, planAnaAccount, planDeliveryPerfCorpusVersion)
		exec("ANALYZE TABLE sales_order, sales_order_line, pick_line, quantity, product")
	})
}

// deliveryPerfPlanRequest is one delivery performance request: its window and filters.
type deliveryPerfPlanRequest struct {
	start, end time.Time
	filters    domain.DeliveryFilters
}

func deliveryPerfPlanCases() []planCase[deliveryPerfPlanRequest] {
	type v = planValue[deliveryPerfPlanRequest]
	dims := []planDim[deliveryPerfPlanRequest]{
		{"customer", []v{
			{"customer=large", func(p *deliveryPerfPlanRequest) { p.filters.CustomerIDs = []string{planAnaBuyerID(0)} }},
			{"customer=rare", func(p *deliveryPerfPlanRequest) { p.filters.CustomerIDs = []string{planAnaBuyerID(planAnaRareBuyer)} }},
		}},
		{"group", []v{
			{"group=big", func(p *deliveryPerfPlanRequest) { p.filters.CustomerGroupIDs = []string{"ag_planana_big"} }},
			{"group=small", func(p *deliveryPerfPlanRequest) { p.filters.CustomerGroupIDs = []string{"ag_planana_small"} }},
		}},
		{"line", []v{
			{"line=large", func(p *deliveryPerfPlanRequest) { p.filters.ProductLineIDs = []string{planAnaLineID(0)} }},
			{"line=rare", func(p *deliveryPerfPlanRequest) { p.filters.ProductLineIDs = []string{planAnaLineID(planAnaRareLine)} }},
		}},
		{"rep", []v{
			{"rep=large", func(p *deliveryPerfPlanRequest) { p.filters.SalesRepIDs = []string{planAnaRepID(0)} }},
			{"rep=rare", func(p *deliveryPerfPlanRequest) { p.filters.SalesRepIDs = []string{planAnaRepID(planAnaRareRep)} }},
		}},
	}
	window := func(start, end time.Time) func(*deliveryPerfPlanRequest) {
		return func(p *deliveryPerfPlanRequest) { p.start, p.end = start, end }
	}
	last := planAnaInvoicedAt(planAnaInvoices - 1)
	windows := []planValue[deliveryPerfPlanRequest]{
		{"30d", window(last.AddDate(0, 0, -30), last)},
		{"1y", window(last.AddDate(-1, 0, 0), last)},
		{"old-quarter", window(time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC))},
	}
	return planCases(deliveryPerfPlanRequest{}, dims, windows)
}

// deliveryPerfPlanFloor is a delivery request's scope: the orders whose commitment fell due in the window
// (and, for the uncommitted count, those issued in it without one), under whichever filter on the order
// itself matches fewest, with those orders' lines. A product line filter is on the order's lines'
// products, two joins away, so no key on the order can pin it: it narrows nothing here.
//
// The packed quantity is a subquery in the select list, which EXPLAIN ANALYZE does not show, so its
// pick_line reads go unmeasured; it is one lookup per line on pick_line_sol_packed_qty_idx.
func deliveryPerfPlanFloor(t *testing.T, db *sql.DB, p deliveryPerfPlanRequest) map[string]float64 {
	t.Helper()
	repo := &salesReportRepoImpl{queries: sqlc.New(db)}
	buyers, filtered, apiErr := repo.resolveBuyers(context.Background(), domain.SalesReportFilter{
		AccountID: planAnaAccount, CustomerIDs: p.filters.CustomerIDs, CustomerGroupIDs: p.filters.CustomerGroupIDs})
	require.Nil(t, apiErr)
	type filter struct {
		clause string
		args   []any
	}
	candidates := []filter{{"TRUE", nil}}
	if filtered {
		if len(buyers) == 0 {
			return map[string]float64{}
		}
		candidates = append(candidates, filter{"so.buyer_account_id IN (" + placeholders(len(buyers)) + ")", stringsToAny(buyers)})
	}
	if ids := p.filters.SalesRepIDs; len(ids) > 0 {
		candidates = append(candidates, filter{"so.sales_rep_id IN (" + placeholders(len(ids)) + ")", stringsToAny(ids)})
	}
	count := func(window string, windowArgs []any) (orders, lines float64) {
		orders = math.Inf(1)
		for _, c := range candidates {
			where := "so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order' AND so.sales_order_status_code IN ('issued', 'fulfilled') AND " + window + " AND " + c.clause
			args := append(append([]any{planAnaAccount}, windowArgs...), c.args...)
			var o, l float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT so.id), COUNT(sol.id) FROM sales_order so
				LEFT JOIN sales_order_line sol ON sol.sales_order_id = so.id
				WHERE `+where, args...).Scan(&o, &l))
			if o < orders {
				orders, lines = o, l
			}
		}
		return orders, lines
	}
	dueOrders, dueLines := count("so.ship_by_date >= ? AND so.ship_by_date <= ?", []any{p.start, p.end})
	openOrders, openLines := count("so.ship_by_date IS NULL AND so.issued_at >= ? AND so.issued_at <= ?", []any{p.start, p.end})
	// The outcomes and their product lines each read the due orders and their lines; the uncommitted
	// count reads its orders, and their lines when a product line filter asks after them.
	floors := map[string]float64{"so": 2*dueOrders + openOrders, "sol": 2 * dueLines}
	if len(p.filters.ProductLineIDs) > 0 {
		floors["sol"] += openLines
	}
	return floors
}

// TestDeliveryPerformance_ResultsUnchanged pins what every plan-tested delivery request returns on the
// corpus, so a change made for its plan cannot change its answer.
func TestDeliveryPerformance_ResultsUnchanged(t *testing.T) {
	ensureDeliveryPerformanceCorpus(t)
	repo := NewProductionScheduleInputRepo(sqlc.New(planDB(t)))
	ctx := context.Background()
	got := map[string]string{}
	for _, c := range deliveryPerfPlanCases() {
		outcomes, apiErr := repo.ListDeliveryOutcomes(ctx, planAnaAccount, c.params.start, c.params.end, c.params.filters)
		require.Nil(t, apiErr)
		uncommitted, apiErr := repo.CountUncommittedOrders(ctx, planAnaAccount, c.params.start, c.params.end, c.params.filters)
		require.Nil(t, apiErr)
		got[c.name] = planDigest(t, struct {
			Outcomes    any
			Uncommitted int
		}{outcomes, uncommitted})
	}
	checkPlanResults(t, "delivery_performance.json", got)
}

// TestDeliveryPerformance_ReadsItsScope holds every delivery performance filter pair and window to
// reading no more orders and lines than its scope (aggregatePlanSuite).
func TestDeliveryPerformance_ReadsItsScope(t *testing.T) {
	ensureDeliveryPerformanceCorpus(t)
	aggregatePlanSuite[deliveryPerfPlanRequest]{
		tables: []aggregateTable{
			{table: "sales_order", scopeColumn: "owner_account_id", from: "FROM sales_order so", alias: "so"},
			{table: "sales_order_line", scopeColumn: "sales_order_id", from: "JOIN sales_order_line sol", alias: "sol"},
		},
		cases: deliveryPerfPlanCases(),
		report: func(ctx context.Context, q *sqlc.Queries, p deliveryPerfPlanRequest) error {
			repo := NewProductionScheduleInputRepo(q)
			if _, apiErr := repo.ListDeliveryOutcomes(ctx, planAnaAccount, p.start, p.end, p.filters); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.CountUncommittedOrders(ctx, planAnaAccount, p.start, p.end, p.filters); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: deliveryPerfPlanFloor,
	}.run(t)
}
