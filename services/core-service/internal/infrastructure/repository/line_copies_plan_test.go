//go:build plans

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// fillCorpusLineCopies copies every line of an account's orders in three statements: the corpus is written
// in plain SQL, and this test is about reading the copies, not filling them.
func fillCorpusLineCopies(t *testing.T, db *sql.DB, accountID string) {
	t.Helper()
	for _, stmt := range []string{
		`UPDATE sales_order_line sol JOIN sales_order so ON so.id = sol.sales_order_id
			LEFT JOIN quantity cq ON cq.id = sol.quantity_id LEFT JOIN rate cp ON cp.id = sol.unit_price_id LEFT JOIN rate cc ON cc.id = sol.unit_cost_id
			SET sol.quantity_value = cq.value, sol.quantity_unit_id = cq.unit_id, sol.unit_price_value = cp.value,
				sol.unit_price_numerator_unit_id = cp.numerator_unit_id, sol.unit_price_denominator_unit_id = cp.denominator_unit_id,
				sol.unit_cost_value = cc.value, sol.unit_cost_numerator_unit_id = cc.numerator_unit_id, sol.unit_cost_denominator_unit_id = cc.denominator_unit_id
			WHERE so.owner_account_id = ?`,
		`UPDATE invoice_line il JOIN sales_order_line sol ON sol.id = il.sales_order_line_id JOIN sales_order so ON so.id = sol.sales_order_id
			LEFT JOIN quantity cq ON cq.id = il.quantity_id
			SET il.quantity_value = cq.value, il.quantity_unit_id = cq.unit_id WHERE so.owner_account_id = ?`,
		`UPDATE pick_line pl JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id JOIN sales_order so ON so.id = sol.sales_order_id
			LEFT JOIN quantity cq ON cq.id = pl.quantity_id
			SET pl.quantity_value = cq.value, pl.quantity_unit_id = cq.unit_id WHERE so.owner_account_id = ?`,
	} {
		_, err := db.Exec(stmt, accountID)
		require.NoError(t, err)
	}
}

// copyReadRe matches a plan visiting a line's row or its quantity or rate rows, which reading from the
// copies is to avoid: any access to the quantity and rate aliases, or a lookup on an order, invoice or
// pick line that is not covering (pl is also product_line's alias, hence the pick_line key name).
var copyReadRe = regexp.MustCompile(`on (q_ord|q_in|r_price|r_cost|q|plq) using|-> (Single-row index lookup|Index lookup|Index range scan) on (sol|il) using|-> (Single-row index lookup|Index lookup|Index range scan) on pl using pick_line`)

// TestLineCopies_ReportsAnswerAsFromTheRows requires every plan-tested order-book, delivery, quarterly-orders and
// demand-forecast request to answer exactly the same from the copies lines carry as from their quantity and
// rate rows, and the copy reads never to visit a line, quantity or rate row.
func TestLineCopies_ReportsAnswerAsFromTheRows(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	ensureDeliveryPerformanceCorpus(t)
	db := planDB(t)
	fillCorpusLineCopies(t, db, planAnaAccount)
	prev := lineCopiesDone.Load()
	t.Cleanup(func() { lineCopiesDone.Store(prev) })
	ctx := context.Background()

	read := func(q *sqlc.Queries) map[string]string {
		analytics, delivery := NewAnalyticsRepo(q), NewProductionScheduleInputRepo(q)
		got := map[string]string{}
		digest := func(key string, v any, err error) {
			t.Helper()
			require.NoError(t, err, key)
			got[key] = planDigest(t, v)
		}
		for _, c := range legacyPlanCases(false) {
			f := openOrderFilterOf(c.params)
			summary, apiErr := analytics.GetOpenOrdersSummary(ctx, f)
			digest("open_summary/"+c.name, summary, nilIfNoErr(apiErr))
			products, apiErr := analytics.GetOpenOrderProducts(ctx, domain.AnalyzeOpenOrderProductsParams{OpenOrderFilter: f, Limit: 10})
			digest("open_products/"+c.name, products, nilIfNoErr(apiErr))
			orders, apiErr := analytics.ListOpenOrders(ctx, domain.ListOpenOrdersParams{OpenOrderFilter: f, Limit: 10})
			digest("open_orders/"+c.name, orders, nilIfNoErr(apiErr))
			lines, apiErr := analytics.GetOpenOrderLineEntries(ctx, f, 200)
			digest("open_lines/"+c.name, lines, nilIfNoErr(apiErr))
		}
		lines, _, apiErr := analytics.GetOpenOrderLines(ctx, planAnaAccount, fmt.Sprintf("or_planana_%07d", planAnaInvoices-1), nil)
		digest("order_lines", lines, nilIfNoErr(apiErr))
		for _, c := range deliveryPerfPlanCases() {
			outcomes, apiErr := delivery.ListDeliveryOutcomes(ctx, planAnaAccount, c.params.start, c.params.end, c.params.filters)
			digest("delivery/"+c.name, outcomes, nilIfNoErr(apiErr))
		}
		for _, c := range quarterlyPlanCases() {
			p := c.params
			quarters, apiErr := analytics.GetQuarterlyOrders(ctx, domain.AnalyzeQuarterlyOrdersParams{AccountID: planAnaAccount, IssuedFrom: p.start,
				ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
			digest("quarterly/"+c.name, quarters, nilIfNoErr(apiErr))
		}
		for _, c := range demandForecastPlanCases() {
			demand, apiErr := analytics.GetDemandForecastMonthlyDemand(ctx, c.params)
			digest("forecast_demand/"+c.name, demand, nilIfNoErr(apiErr))
			revenue, apiErr := analytics.GetDemandForecastMonthlyRevenue(ctx, c.params)
			digest("forecast_revenue/"+c.name, revenue, nilIfNoErr(apiErr))
		}
		return got
	}

	lineCopiesDone.Store(false)
	var completed bool
	err := db.QueryRow("SELECT completed_at IS NOT NULL FROM backfill_progress WHERE name = ?", LineCopyBackfillName).Scan(&completed)
	require.True(t, err == sql.ErrNoRows || (err == nil && !completed), "the plan database must not record the backfill as finished")
	fromRows := read(sqlc.New(db))

	lineCopiesDone.Store(true)
	edb := &explainingDB{db: db, explainRows: true}
	fromCopies := read(sqlc.New(edb))
	require.Equal(t, fromRows, fromCopies)

	reads := 0
	for _, st := range edb.statements {
		if !regexp.MustCompile(`sol\.quantity_value|il\.quantity_value|pl\.quantity_value`).MatchString(st.query) {
			continue
		}
		reads++
		require.NotRegexp(t, copyReadRe, st.plan, "a copy read visits a row it copies\n%s\n%s", st.query, st.plan)
	}
	require.Positive(t, reads, "the copy reads were measured")
}

// TestLineCopies_FulfillmentRollupsAnswerAsFromTheRows requires the sales order lines, sales order fulfillment
// progress and pick progress to answer exactly the same from the copies as from the quantity rows, and the
// copy reads to sum pick and invoice lines from their copy keys alone.
func TestLineCopies_FulfillmentRollupsAnswerAsFromTheRows(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	ensureDeliveryPerformanceCorpus(t)
	db := planDB(t)
	fillCorpusLineCopies(t, db, planAnaAccount)
	prev := lineCopiesDone.Load()
	t.Cleanup(func() { lineCopiesDone.Store(prev) })
	ctx := context.Background()

	ids := func(query string) []string {
		t.Helper()
		rows, err := db.Query(query, planAnaAccount)
		require.NoError(t, err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			out = append(out, id)
		}
		require.NoError(t, rows.Err())
		require.NotEmpty(t, out, query)
		return out
	}
	// The corpus picks and invoices every line, so a page of the oldest orders and one of the newest.
	picked := ids(`SELECT so.id FROM sales_order so WHERE so.owner_account_id = ? ORDER BY so.id LIMIT 25`)
	newest := ids(`SELECT so.id FROM sales_order so WHERE so.owner_account_id = ? ORDER BY so.id DESC LIMIT 25`)
	picks := ids(`SELECT p.id FROM pick p WHERE p.account_id = ? ORDER BY p.id LIMIT 50`)
	orders := append(append([]string{}, picked...), newest...)

	read := func(q *sqlc.Queries) map[string]string {
		orderRepo, pickRepo := NewSalesOrderRepo(q), &pickRepoImpl{queries: q}
		got := map[string]string{}
		lines, apiErr := orderRepo.GetLinesForOrders(ctx, orders)
		require.Nil(t, apiErr)
		got["lines_for_orders"] = planDigest(t, lines)
		one, apiErr := orderRepo.GetLines(ctx, picked[0])
		require.Nil(t, apiErr)
		got["lines"] = planDigest(t, one)
		progress, apiErr := orderRepo.GetFulfillmentProgress(ctx, orders)
		require.Nil(t, apiErr)
		got["order_progress"] = planDigest(t, progress)
		pickProgress, apiErr := pickRepo.GetProgress(ctx, picks)
		require.Nil(t, apiErr)
		got["pick_progress"] = planDigest(t, pickProgress)
		return got
	}

	lineCopiesDone.Store(false)
	fromRows := read(sqlc.New(db))

	lineCopiesDone.Store(true)
	edb := &explainingDB{db: db, explainRows: true}
	fromCopies := read(sqlc.New(edb))
	require.Equal(t, fromRows, fromCopies)

	reads := 0
	for _, st := range edb.statements {
		if !strings.Contains(st.query, "FORCE INDEX (pick_line_sol_packed_copies_idx)") {
			continue
		}
		reads++
		require.Contains(t, st.plan, "Covering index lookup on pl using pick_line_sol_packed_copies_idx", "%s\n%s", st.query, st.plan)
		require.Contains(t, st.plan, "Covering index lookup on il using invoice_line_sol_copies_idx", "%s\n%s", st.query, st.plan)
		require.NotRegexp(t, `on (plq|ilq) using`, st.plan, "%s\n%s", st.query, st.plan)
		if strings.Contains(st.query, "GetSalesOrderFulfillmentProgressFromCopies") {
			require.Contains(t, st.plan, "Covering index range scan on sol using sales_order_line_order_copies_idx", "%s\n%s", st.query, st.plan)
		}
	}
	require.Equal(t, 3, reads, "the lines for orders, the lines and the order progress read the copies")
}
