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
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
)

// --- Order book: order data, open orders, quarterly orders ---

// orderBookPlanTables are the tables every order-book read reaches: the orders and their lines, the invoice lines
// totalled against them, and the orders and lines that total is held to (inv_so, inv_sol).
var orderBookPlanTables = []aggregateTable{
	{table: "sales_order", scopeColumn: "owner_account_id", from: "FROM sales_order so", alias: "so"},
	{table: "sales_order_line", scopeColumn: "sales_order_id", from: "JOIN sales_order_line sol ON", alias: "sol"},
	{table: "invoice_line", scopeColumn: "sales_order_line_id", from: "FROM invoice_line il", alias: "il"},
	{table: "sales_order", scopeColumn: "owner_account_id", from: "FROM sales_order inv_so", alias: "inv_so"},
	{table: "sales_order_line", scopeColumn: "sales_order_id", from: "JOIN sales_order_line inv_sol", alias: "inv_sol"},
}

// orderBookFloor is an order-book report's scope: the account's issued sales orders (open only, or all), their
// lines, and those lines' invoice lines. Every filter but the buyer and rep is on the lines, so no order key can
// narrow below it; the corpus keeps its open orders few, which makes reading the closed ones conspicuous.
func orderBookFloor(t *testing.T, db *sql.DB, openOnly bool) map[string]float64 {
	t.Helper()
	open := ""
	if openOnly {
		open = " AND so.completed_at IS NULL"
	}
	var orders, lines, invoiced float64
	require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT so.id), COUNT(DISTINCT sol.id), COUNT(il.id) FROM sales_order so
		JOIN sales_order_line sol ON sol.sales_order_id = so.id LEFT JOIN invoice_line il ON il.sales_order_line_id = sol.id
		WHERE so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order' AND so.sales_order_status_code = 'issued'`+open,
		planAnaAccount).Scan(&orders, &lines, &invoiced))
	return map[string]float64{"so": orders, "sol": lines, "il": invoiced, "inv_so": orders, "inv_sol": lines}
}

func openOrderFilterOf(p legacyPlanRequest) domain.OpenOrderFilter {
	return domain.OpenOrderFilter{AccountID: planAnaAccount, CustomerIDs: p.customerIDs, CustomerGroupIDs: p.customerGroupIDs,
		ProductLineIDs: p.productLineIDs, SalesRepIDs: p.salesRepIDs}
}

// TestOpenOrderReports_ReadTheirScope holds the open-orders summary, products, list and export to the open orders,
// their lines and those lines' invoice lines, whatever the filters (aggregatePlanSuite).
func TestOpenOrderReports_ReadTheirScope(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	aggregatePlanSuite[legacyPlanRequest]{
		tables: orderBookPlanTables,
		cases:  legacyPlanCases(false),
		report: func(ctx context.Context, q *sqlc.Queries, p legacyPlanRequest) error {
			repo := NewAnalyticsRepo(q)
			f := openOrderFilterOf(p)
			if _, apiErr := repo.GetOpenOrdersSummary(ctx, f); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.GetOpenOrderProducts(ctx, domain.AnalyzeOpenOrderProductsParams{OpenOrderFilter: f, Limit: 10}); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.ListOpenOrders(ctx, domain.ListOpenOrdersParams{OpenOrderFilter: f, Limit: 10}); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.GetOpenOrderLineEntries(ctx, f, domain.ExportRowLimit+1); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p legacyPlanRequest) map[string]float64 {
			// Four reports each read the scope once.
			floor := orderBookFloor(t, db, true)
			for k := range floor {
				floor[k] *= 4
			}
			return floor
		},
	}.run(t)
}

// openOrderPlanRequest is one page of the open-orders list.
type openOrderPlanRequest struct {
	legacyPlanRequest
	cursor *string
}

// TestOpenOrderList_ReadsAPage holds the open-orders list's page query to its open orders: no key yields open
// orders in issue order, so a page reads the orders its order-level filters match (listPlanSuite floor).
func TestOpenOrderList_ReadsAPage(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	db := planDB(t)

	var open []struct {
		id       string
		issuedAt time.Time
	}
	rows, err := db.Query(`SELECT id, issued_at FROM sales_order WHERE owner_account_id = ? AND sales_order_type_code = 'sales_order'
		AND sales_order_status_code = 'issued' AND completed_at IS NULL ORDER BY issued_at DESC, id DESC`, planAnaAccount)
	require.NoError(t, err)
	for rows.Next() {
		var o struct {
			id       string
			issuedAt time.Time
		}
		require.NoError(t, rows.Scan(&o.id, &o.issuedAt))
		open = append(open, o)
	}
	require.NoError(t, rows.Err())
	require.Greater(t, len(open), 20, "the corpus must keep some orders open")
	mid := open[len(open)/2]

	type v = planValue[openOrderPlanRequest]
	pages := []v{
		{"first", func(*openOrderPlanRequest) {}},
		{"deep-forward", func(p *openOrderPlanRequest) {
			p.cursor = planCursorAt(mid.issuedAt, mid.id, pagination.DirectionForward)
		}},
		{"deep-backward", func(p *openOrderPlanRequest) {
			p.cursor = planCursorAt(mid.issuedAt, mid.id, pagination.DirectionBackward)
		}},
	}
	dims := []planDim[openOrderPlanRequest]{
		{"customer", []v{
			{"customer=large", func(p *openOrderPlanRequest) { p.customerIDs = []string{planAnaBuyerID(0)} }},
			{"customer=rare", func(p *openOrderPlanRequest) { p.customerIDs = []string{planAnaBuyerID(planAnaRareBuyer)} }},
		}},
		{"group", []v{
			{"group=big", func(p *openOrderPlanRequest) { p.customerGroupIDs = []string{"ag_planana_big"} }},
			{"group=small", func(p *openOrderPlanRequest) { p.customerGroupIDs = []string{"ag_planana_small"} }},
		}},
		{"line", []v{
			{"line=large", func(p *openOrderPlanRequest) { p.productLineIDs = []string{planAnaLineID(0)} }},
			{"line=rare", func(p *openOrderPlanRequest) { p.productLineIDs = []string{planAnaLineID(planAnaRareLine)} }},
		}},
		{"rep", []v{
			{"rep=large", func(p *openOrderPlanRequest) { p.salesRepIDs = []string{planAnaRepID(0)} }},
			{"rep=rare", func(p *openOrderPlanRequest) { p.salesRepIDs = []string{planAnaRepID(planAnaRareRep)} }},
		}},
	}

	listPlanSuite[openOrderPlanRequest]{
		table: "sales_order", scopeColumn: "owner_account_id",
		from: "FROM sales_order so", alias: "so",
		statement: func(query string) bool { return strings.HasPrefix(query, "SELECT so.id FROM sales_order so") },
		cases:     planCases(openOrderPlanRequest{}, dims, pages),
		limit:     func(openOrderPlanRequest) int32 { return 10 },
		list: func(ctx context.Context, q *sqlc.Queries, p openOrderPlanRequest) error {
			_, apiErr := NewAnalyticsRepo(q).ListOpenOrders(ctx, domain.ListOpenOrdersParams{OpenOrderFilter: openOrderFilterOf(p.legacyPlanRequest), Limit: 10, Cursor: p.cursor})
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		// No key yields open orders in issue order, so a page reads the open orders.
		floor: func(t *testing.T, db *sql.DB, p openOrderPlanRequest) float64 {
			return float64(len(open))
		},
		// A filter that resolves to no buyer answers without reading.
		empty: func(openOrderPlanRequest) bool { return true },
		reads: []planRead[openOrderPlanRequest]{
			// The line filters probe each candidate order's lines.
			{alias: "sol", fanout: 8},
			{alias: "fg", fanout: 8},
		},
	}.run(t)
}

// TestOpenOrderLines_ReadsTheOrder holds one order's lines to that order's lines and their invoice lines, and the
// base-unit lookup to the platform's units.
func TestOpenOrderLines_ReadsTheOrder(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	db := planDB(t)
	lookupPlanSuite{
		tables: []string{"sales_order", "sales_order_line", "invoice_line"},
		cases: []lookupPlanCase{
			{name: "open", run: func(ctx context.Context, q *sqlc.Queries) error {
				lines, found, apiErr := NewAnalyticsRepo(q).GetOpenOrderLines(ctx, planAnaAccount, fmt.Sprintf("or_planana_%07d", planAnaInvoices-1), nil)
				if apiErr != nil {
					return apiErr
				}
				if !found || len(lines) == 0 {
					return fmt.Errorf("the corpus's last order has no lines")
				}
				return nil
			}},
			{name: "fulfilled-busy-history", run: func(ctx context.Context, q *sqlc.Queries) error {
				_, _, apiErr := NewAnalyticsRepo(q).GetOpenOrderLines(ctx, planAnaAccount, "or_planana_0001000", nil)
				if apiErr != nil {
					return apiErr
				}
				return nil
			}},
		},
		// No key pins a base unit, so the lookup reads every platform unit, and the plan database's count
		// grows with each corpus that adds one. It is held to that set, ranged from the account key.
		returned: func(t *testing.T, stmt explainedStatement) float64 {
			if !strings.Contains(stmt.query, "FROM unit WHERE account_id IS NULL") {
				return 0
			}
			require.False(t, hasFullScan(stmt.plan, "unit"), "the base units must be read from the account key\n%s", stmt.plan)
			var n float64
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM unit WHERE account_id IS NULL").Scan(&n))
			return n
		},
	}.run(t)
}

// orderWindowFloor is the most selective way any order key reaches a quarterly request's orders: the account's orders
// issued in the window, or, under a buyer or rep filter, that buyer's or rep's orders of any date (no key pairs them
// with the issue date). It returns that set's orders and their lines. Line filters cannot be pinned by an order key.
func orderWindowFloor(t *testing.T, db *sql.DB, p legacyPlanRequest, from time.Time) (orders, lines float64) {
	t.Helper()
	count := func(where string, args ...any) (float64, float64) {
		var o, l float64
		require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT so.id), COUNT(sol.id) FROM sales_order so
			LEFT JOIN sales_order_line sol ON sol.sales_order_id = so.id WHERE so.owner_account_id = ? AND `+where,
			append([]any{planAnaAccount}, args...)...).Scan(&o, &l))
		return o, l
	}
	orders, lines = count("so.sales_order_type_code = 'sales_order' AND so.issued_at >= ?", from)
	consider := func(o, l float64) {
		if o < orders {
			orders, lines = o, l
		}
	}
	if len(p.customerIDs) > 0 || len(p.customerGroupIDs) > 0 {
		buyers, _, apiErr := resolveCustomerBuyers(context.Background(), db, planAnaAccount, p.customerIDs, p.customerGroupIDs)
		require.Nil(t, apiErr)
		if len(buyers) == 0 {
			return 0, 0
		}
		consider(count("so.buyer_account_id IN ("+placeholders(len(buyers))+")", stringsToAny(buyers)...))
	}
	if len(p.salesRepIDs) > 0 {
		consider(count("so.sales_rep_id IN ("+placeholders(len(p.salesRepIDs))+")", stringsToAny(p.salesRepIDs)...))
	}
	return orders, lines
}

func quarterlyPlanCases() []planCase[legacyPlanRequest] {
	var cases []planCase[legacyPlanRequest]
	for _, c := range legacyPlanCases(false) {
		for _, w := range []struct {
			label string
			from  time.Time
		}{
			{"5y", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
			{"1y", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		} {
			p := c.params
			p.start = w.from
			cases = append(cases, planCase[legacyPlanRequest]{name: strings.TrimSuffix(c.name, "open") + w.label, params: p})
		}
	}
	return cases
}

// TestQuarterlyOrders_ReadsItsScope holds the quarterly-orders report to the sales orders issued in its years under
// the request's buyer or rep filter, and their lines (aggregatePlanSuite).
func TestQuarterlyOrders_ReadsItsScope(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	aggregatePlanSuite[legacyPlanRequest]{
		tables: []aggregateTable{
			{table: "sales_order", scopeColumn: "owner_account_id", from: "FROM sales_order so", alias: "so"},
			{table: "sales_order_line", scopeColumn: "sales_order_id", from: "JOIN sales_order_line sol ON", alias: "sol"},
		},
		cases: quarterlyPlanCases(),
		report: func(ctx context.Context, q *sqlc.Queries, p legacyPlanRequest) error {
			_, apiErr := NewAnalyticsRepo(q).GetQuarterlyOrders(ctx, domain.AnalyzeQuarterlyOrdersParams{AccountID: planAnaAccount, IssuedFrom: p.start,
				ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p legacyPlanRequest) map[string]float64 {
			orders, lines := orderWindowFloor(t, db, p, p.start)
			return map[string]float64{"so": orders, "sol": lines}
		},
	}.run(t)
}

// demandForecastPlanCases are the forecast's windows under its two filters.
func demandForecastPlanCases() []planCase[domain.GetDemandForecastWindowParams] {
	last := planAnaInvoicedAt(planAnaInvoices - 1)
	end := time.Date(last.Year(), last.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	type v = planValue[domain.GetDemandForecastWindowParams]
	dims := []planDim[domain.GetDemandForecastWindowParams]{
		{"line", []v{
			{"line=large", func(p *domain.GetDemandForecastWindowParams) { p.ProductLineIDs = []string{planAnaLineID(0)} }},
			{"line=rare", func(p *domain.GetDemandForecastWindowParams) {
				p.ProductLineIDs = []string{planAnaLineID(planAnaRareLine)}
			}},
		}},
		{"item", []v{
			{"item=large", func(p *domain.GetDemandForecastWindowParams) { p.ItemIDs = []string{planAnaItemID(0)} }},
			{"item=rare", func(p *domain.GetDemandForecastWindowParams) { p.ItemIDs = []string{planAnaItemID(planAnaRareItem)} }},
		}},
	}
	window := func(months int) func(*domain.GetDemandForecastWindowParams) {
		return func(p *domain.GetDemandForecastWindowParams) {
			p.StartDate, p.EndDate = time.Date(end.Year(), end.Month()-time.Month(months)-1, 1, 0, 0, 0, 0, time.UTC), end
		}
	}
	return planCases(domain.GetDemandForecastWindowParams{AccountID: planAnaAccount}, dims, []v{{"24m", window(24)}, {"60m", window(60)}})
}

// TestDemandForecast_ReadsItsScope holds the forecast's two reads to their windows: the account's sales orders
// created in it with their lines, and its invoices with their lines (aggregatePlanSuite). The product-line and item
// filters are on the lines, which no order or invoice key can pin.
func TestDemandForecast_ReadsItsScope(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	aggregatePlanSuite[domain.GetDemandForecastWindowParams]{
		tables: []aggregateTable{
			{table: "sales_order", scopeColumn: "owner_account_id", from: "FROM sales_order so", alias: "so"},
			{table: "sales_order_line", scopeColumn: "sales_order_id", from: "JOIN sales_order_line sol ON sol.sales_order_id", alias: "sol"},
			{table: "invoice", scopeColumn: "account_id", from: "FROM invoice inv", alias: "inv"},
			{table: "invoice_line", scopeColumn: "invoice_id", from: "JOIN invoice_line il", alias: "il"},
		},
		cases: demandForecastPlanCases(),
		report: func(ctx context.Context, q *sqlc.Queries, p domain.GetDemandForecastWindowParams) error {
			repo := NewAnalyticsRepo(q)
			if _, apiErr := repo.GetDemandForecastMonthlyDemand(ctx, p); apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.GetDemandForecastMonthlyRevenue(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p domain.GetDemandForecastWindowParams) map[string]float64 {
			var orders, lines, invoices, invoiceLines float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT so.id), COUNT(sol.id) FROM sales_order so LEFT JOIN sales_order_line sol ON sol.sales_order_id = so.id
				WHERE so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order' AND so.created_at >= ? AND so.created_at < ?`,
				planAnaAccount, p.StartDate, p.EndDate).Scan(&orders, &lines))
			require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT i.id), COUNT(il.id) FROM invoice i LEFT JOIN invoice_line il ON il.invoice_id = i.id
				WHERE i.account_id = ? AND i.created_at >= ? AND i.created_at < ?`, planAnaAccount, p.StartDate, p.EndDate).Scan(&invoices, &invoiceLines))
			return map[string]float64{"so": orders, "sol": lines, "inv": invoices, "il": invoiceLines}
		},
	}.run(t)
}

// TestWeeksOfSalesDemand_ReadsItsWindow holds weeks-of-sales demand to the sales orders issued in its window and
// their lines.
func TestWeeksOfSalesDemand_ReadsItsWindow(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	last := planAnaInvoicedAt(planAnaInvoices - 1)
	type req = domain.GetOrderQuantitiesByProductLinesParams
	lines := make([]string, planAnaLines)
	for l := range planAnaLines {
		lines[l] = planAnaLineID(l)
	}
	var cases []planCase[req]
	for _, weeks := range []int{4, 26, 104} {
		cases = append(cases, planCase[req]{name: fmt.Sprintf("%dw", weeks), params: req{AccountID: planAnaAccount, ProductLineIDs: lines,
			StartDate: last.AddDate(0, 0, -7*weeks), EndDate: last}})
	}
	aggregatePlanSuite[req]{
		tables: []aggregateTable{
			{table: "sales_order", scopeColumn: "owner_account_id", from: "FROM sales_order so", alias: "so"},
			{table: "sales_order_line", scopeColumn: "sales_order_id", from: "JOIN sales_order_line sol", alias: "sol"},
		},
		cases: cases,
		report: func(ctx context.Context, q *sqlc.Queries, p req) error {
			_, apiErr := NewAnalyticsRepo(q).GetOrderQuantitiesByProductLines(ctx, p)
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p req) map[string]float64 {
			var orders, lines float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT so.id), COUNT(sol.id) FROM sales_order so LEFT JOIN sales_order_line sol ON sol.sales_order_id = so.id
				WHERE so.owner_account_id = ? AND so.sales_order_type_code = 'sales_order' AND so.issued_at >= ? AND so.issued_at <= ?`, planAnaAccount, p.StartDate, p.EndDate).Scan(&orders, &lines))
			return map[string]float64{"so": orders, "sol": lines}
		},
	}.run(t)
}

// --- Stock: inventory receipts, materials, weeks-of-sales on hand ---

// The stock reports read the inventory corpus's receipts with their unit costs, and its materials' issues. This
// adds what those reads need beside it: a cost for each available receipt, a material (with an order point and lead
// time) for every third item, and reserved and open issues against each material.
const planStockCorpusVersion = "Plan Stock v1"

var planStockOnce sync.Once

func ensureStockCorpus(t *testing.T) {
	t.Helper()
	ensureInventoryCorpus(t)
	db := planDB(t)
	planStockOnce.Do(func() {
		var version string
		_ = db.QueryRow("SELECT name FROM item_category WHERE id = 'ic_planstk'").Scan(&version)
		if version == planStockCorpusVersion {
			return
		}
		t.Log("seeding the stock plan corpus; it is kept for later runs")
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM rate WHERE id LIKE 'rt\\_inrc\\_planivt\\_%'")
		exec("DELETE FROM material WHERE id LIKE 'ml\\_planstk\\_%'")
		exec("DELETE FROM inventory_issue WHERE id LIKE 'inis\\_planstk\\_%'")
		exec("DELETE FROM quantity WHERE id LIKE 'qu\\_planstk\\_%'")
		exec("DELETE FROM item_category WHERE id = 'ic_planstk'")

		var rateVals []string
		var rateArgs []any
		flushRates := func() {
			if len(rateVals) > 0 {
				exec("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES "+strings.Join(rateVals, ","), rateArgs...)
			}
			rateVals, rateArgs = nil, nil
		}
		var qVals, mVals, iVals []string
		var qArgs, mArgs, iArgs []any
		flush := func() {
			if len(qVals) > 0 {
				exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES "+strings.Join(qVals, ","), qArgs...)
			}
			if len(mVals) > 0 {
				exec("INSERT INTO material (id, item_id, order_point_id, lead_time_id) VALUES "+strings.Join(mVals, ","), mArgs...)
			}
			if len(iVals) > 0 {
				exec(`INSERT INTO inventory_issue (id, account_id, item_id, status_code, quantity_id, issued_at, created_at, updated_at)
				      VALUES `+strings.Join(iVals, ","), iArgs...)
			}
			qVals, mVals, iVals, qArgs, mArgs, iArgs = nil, nil, nil, nil, nil, nil
		}
		for n := range planInvItems {
			available, _ := planInvReceipts(n)
			at := planInvCreatedAt(n)
			for k := range available {
				rID := fmt.Sprintf("inrc_planivt_%05d_%03d", n, k)
				rateVals = append(rateVals, "(?, ?, 'dollar', 'un_planivt', ?, ?)")
				rateArgs = append(rateArgs, "rt_"+rID, fmt.Sprintf("%d.%02d", 1+planHash(n*64+k, 31)%40, planHash(n, 32)%100), at, at)
			}
			if n%3 == 0 {
				op, lt := fmt.Sprintf("qu_planstk_op_%05d", n), fmt.Sprintf("qu_planstk_lt_%05d", n)
				qVals = append(qVals, "(?, '50', 'un_planivt', ?, ?)", "(?, '10', 'day', ?, ?)")
				qArgs = append(qArgs, op, at, at, lt, at, at)
				mVals = append(mVals, "(?, ?, ?, ?)")
				mArgs = append(mArgs, fmt.Sprintf("ml_planstk_%05d", n), planInvItemID(n), op, lt)
				for k, status := range []string{"reserved", "reserved", "open", "issued", "issued", "issued"} {
					id := fmt.Sprintf("inis_planstk_%05d_%d", n, k)
					qVals = append(qVals, "(?, '7', 'un_planivt', ?, ?)")
					qArgs = append(qArgs, "qu_planstk_"+id, at, at)
					iVals = append(iVals, "(?, ?, ?, ?, ?, ?, ?, ?)")
					iArgs = append(iArgs, id, planInvAccount, planInvItemID(n), status, "qu_planstk_"+id, at, at, at)
				}
			}
			if len(rateVals) > 2_000 {
				flushRates()
			}
			if len(qVals) > 2_000 {
				flush()
			}
		}
		flushRates()
		flush()
		exec(`INSERT INTO item_category (id, name, item_category_type_code, unit_group_id, account_id) VALUES ('ic_planstk', ?, 'product', 'ug_planivt', ?)`,
			planStockCorpusVersion, planInvAccount)
		exec("ANALYZE TABLE rate, material, inventory_issue, quantity")
	})
}

// TestInventoryReceiptSummary_ReadsTheAccountsReceipts holds the receipt summary to the available receipts the account
// owns or holds (aggregatePlanSuite): once through the owner key and once through the holder key, which for this
// corpus are the same rows. An item filter reads only that item's.
func TestInventoryReceiptSummary_ReadsTheAccountsReceipts(t *testing.T) {
	ensureStockCorpus(t)
	type req = domain.AnalyzeInventoryReceiptsParams
	cases := []planCase[req]{
		{name: "unfiltered", params: req{AccountID: planInvAccount}},
		{name: "item=busy", params: req{AccountID: planInvAccount, ItemIDs: []string{planInvItemID(planInvBusyItem)}}},
		{name: "item=quiet", params: req{AccountID: planInvAccount, ItemIDs: []string{planInvItemID(planInvItems - 2)}}},
		{name: "location=none", params: req{AccountID: planInvAccount, LocationIDs: []string{"sl_planivt_none"}}},
		{name: "lot=none", params: req{AccountID: planInvAccount, LotIDs: []string{"lt_planivt_none"}}},
	}
	aggregatePlanSuite[req]{
		// inventory_allocation has no production statistics snapshot; its reads are one keyed probe per available receipt.
		tables: []aggregateTable{
			{table: "inventory_receipt", scopeColumn: "owner_account_id", from: "FROM inventory_receipt ir", alias: "ir"},
		},
		cases: cases,
		report: func(ctx context.Context, q *sqlc.Queries, p req) error {
			_, apiErr := NewAnalyticsRepo(q).GetInventoryReceiptAnalytics(ctx, p)
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p req) map[string]float64 {
			// Without a filter the owner and holder keys each read their available receipts.
			if len(p.ItemIDs) == 0 {
				var owned, held float64
				require.NoError(t, db.QueryRow(`SELECT COALESCE(SUM(owner_account_id = ?), 0), COALESCE(SUM(holder_account_id = ?), 0) FROM inventory_receipt
					WHERE (owner_account_id = ? OR holder_account_id = ?) AND status_code = 'available'`, planInvAccount, planInvAccount, planInvAccount, planInvAccount).Scan(&owned, &held))
				return map[string]float64{"ir": owned + held}
			}
			var receipts float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM inventory_receipt WHERE (owner_account_id = ? OR holder_account_id = ?) AND status_code = 'available' AND item_id IN (`+placeholders(len(p.ItemIDs))+`)`,
				append([]any{planInvAccount, planInvAccount}, stringsToAny(p.ItemIDs)...)...).Scan(&receipts))
			return map[string]float64{"ir": receipts}
		},
	}.run(t)
}

// requireTenantLedReads runs report under both statistics modes and fails it for reaching a table through a key
// every tenant's rows share (a status, a type) or by scanning it: the corpora are too much of this database for a
// floor to notice what such a read costs in production. forbidden maps a table alias to the keys it must not use.
func requireTenantLedReads(t *testing.T, statsTables []string, forbidden map[string][]string, report func(q *sqlc.Queries) error) {
	t.Helper()
	db := planDB(t)
	for _, mode := range planStatsModesFor(t, db, statsTables) {
		t.Run("stats="+mode, func(t *testing.T) {
			for _, table := range statsTables {
				usePlanStats(t, db, table, mode)
				t.Cleanup(func() { usePlanStats(t, db, table, "analyzed") })
			}
			edb := &explainingDB{db: db}
			require.NoError(t, report(sqlc.New(edb)))
			require.NotEmpty(t, edb.statements)
			for _, stmt := range edb.statements {
				for alias, keys := range forbidden {
					require.False(t, hasFullScan(stmt.plan, alias), "scanned %s\n%s\n%s", alias, stmt.query, stmt.plan)
					for _, index := range tableAccess(stmt.plan, alias).indexes {
						require.NotContains(t, keys, index, "read %s through a key every tenant shares\n%s\n%s", alias, stmt.query, stmt.plan)
					}
				}
			}
		})
	}
}

func TestInventoryReceiptSummary_NeverReadsAcrossTenants(t *testing.T) {
	ensureStockCorpus(t)
	for _, p := range []domain.AnalyzeInventoryReceiptsParams{
		{AccountID: planInvAccount},
		{AccountID: planInvAccount, ItemIDs: []string{planInvItemID(planInvBusyItem)}},
		{AccountID: planInvAccount, LocationIDs: []string{"sl_planivt_none"}},
	} {
		requireTenantLedReads(t, []string{"inventory_receipt"}, map[string][]string{"ir": {"inventory_receipt_status_code_idx"}}, func(q *sqlc.Queries) error {
			return nilIfNoErr(apiErrOf(NewAnalyticsRepo(q).GetInventoryReceiptAnalytics(context.Background(), p)))
		})
	}
}

func TestMaterialAnalytics_NeverReadsAcrossTenants(t *testing.T) {
	ensureStockCorpus(t)
	requireTenantLedReads(t, []string{"inventory_receipt", "item"}, map[string][]string{
		"it": nil,
		"m":  nil,
		"ir": {"inventory_receipt_status_code_idx"},
		"ii": {"inventory_issue_status_code_idx"},
	}, func(q *sqlc.Queries) error {
		return nilIfNoErr(apiErrOf(NewAnalyticsRepo(q).GetMaterialAnalytics(context.Background(), domain.AnalyzeMaterialsParams{AccountID: planInvAccount})))
	})
}

func TestWeeksOfSalesProducts_NeverReadAcrossTenants(t *testing.T) {
	ensureStockCorpus(t)
	requireTenantLedReads(t, []string{"product", "item"}, map[string][]string{
		"i": nil,
		"p": {"product_product_type_code_idx"},
	}, func(q *sqlc.Queries) error {
		return nilIfNoErr(apiErrOf(NewAnalyticsRepo(q).GetSaleProductItemIDs(context.Background(), planInvAccount)))
	})
}

func apiErrOf[T any](_ T, apiErr *apierror.APIError) *apierror.APIError { return apiErr }

// TestMaterialAnalytics_ReadsItsMaterials holds the materials report to the account's materials and, for their
// items, the available receipts and reserved and open issues with what has been drawn on them.
func TestMaterialAnalytics_ReadsItsMaterials(t *testing.T) {
	ensureStockCorpus(t)
	type req = domain.AnalyzeMaterialsParams
	aggregatePlanSuite[req]{
		// inventory_issue has no production statistics snapshot; its reads go by (account, item, status).
		tables: []aggregateTable{
			{table: "inventory_receipt", scopeColumn: "item_id", from: "FROM inventory_receipt ir", alias: "ir"},
		},
		cases: []planCase[req]{{name: "all", params: req{AccountID: planInvAccount}}},
		report: func(ctx context.Context, q *sqlc.Queries, p req) error {
			_, apiErr := NewAnalyticsRepo(q).GetMaterialAnalytics(ctx, p)
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p req) map[string]float64 {
			var receipts float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM inventory_receipt ir JOIN material m ON m.item_id = ir.item_id
				WHERE ir.status_code = 'available' AND ir.owner_account_id = ?`, planInvAccount).Scan(&receipts))
			return map[string]float64{"ir": receipts}
		},
	}.run(t)
}

// TestWeeksOfSalesOnHand_ReadsTheItemsReceipts holds weeks-of-sales stock to the sale items' available receipts and
// the allocations against them, deleted items included.
func TestWeeksOfSalesOnHand_ReadsTheItemsReceipts(t *testing.T) {
	ensureStockCorpus(t)
	db := planDB(t)
	rows, err := db.Query(`SELECT p.item_id FROM product p JOIN item i ON i.id = p.item_id WHERE i.account_id = ? AND p.product_type_code = 'sale'`, planInvAccount)
	require.NoError(t, err)
	var items []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		items = append(items, id)
	}
	require.NoError(t, rows.Err())
	require.NotEmpty(t, items)

	type req struct{ items []string }
	aggregatePlanSuite[req]{
		tables: []aggregateTable{
			{table: "inventory_receipt", scopeColumn: "item_id", from: "FROM inventory_receipt ir\n", alias: "ir"},
			{table: "inventory_receipt", scopeColumn: "item_id", from: "FROM inventory_receipt ir2", alias: "ir2"},
		},
		cases: []planCase[req]{{name: "sale-items", params: req{items}}, {name: "one-busy-item", params: req{[]string{planInvItemID(planInvBusyItem)}}}},
		report: func(ctx context.Context, q *sqlc.Queries, p req) error {
			_, apiErr := NewAnalyticsRepo(q).GetWeeksOfSalesOnHand(ctx, planInvAccount, p.items)
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p req) map[string]float64 {
			var receipts float64
			require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM inventory_receipt WHERE status_code = 'available' AND item_id IN (`+placeholders(len(p.items))+`)`,
				stringsToAny(p.items)...).Scan(&receipts))
			return map[string]float64{"ir": receipts, "ir2": receipts}
		},
	}.run(t)
}

// --- Open batches ---

// TestOpenBatchSummary_ReadsTheAccountsOpenBatches holds the open-batches report to the account's open scanned
// batches, or under an item filter to those items' batches when they are fewer (aggregatePlanSuite). _batch_flow has no
// production statistics snapshot; its reads are one keyed probe per batch.
func TestOpenBatchSummary_ReadsTheAccountsOpenBatches(t *testing.T) {
	ensureBatchCorpus(t)
	type req struct{ items []string }
	aggregatePlanSuite[req]{
		tables: []aggregateTable{
			{table: "batch", scopeColumn: "account_id", from: "FROM batch b", alias: "b"},
		},
		cases: []planCase[req]{
			{name: "unfiltered", params: req{}},
			{name: "item=busy", params: req{[]string{planBatchItemID(0)}}},
			{name: "item=rare", params: req{[]string{planBatchItemID(planBatchItems - 1)}}},
		},
		report: func(ctx context.Context, q *sqlc.Queries, p req) error {
			_, apiErr := NewBatchRepo(q).FindOpenBatches(ctx, planBatchAccount, p.items, nil)
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p req) map[string]float64 {
			var scanned float64
			require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM batch WHERE account_id = ? AND closed_at IS NULL AND scanned_at IS NOT NULL", planBatchAccount).Scan(&scanned))
			if len(p.items) > 0 {
				var itemBatches float64
				require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM batch WHERE item_id IN ("+placeholders(len(p.items))+")", stringsToAny(p.items)...).Scan(&itemBatches))
				scanned = min(scanned, itemBatches)
			}
			return map[string]float64{"b": scanned}
		},
	}.run(t)
}

// --- Results ---

// TestAnalyticsParity_ResultsUnchanged pins what the order-book and stock reports return, so a plan change cannot
// change a figure unnoticed.
func TestAnalyticsParity_ResultsUnchanged(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	ensureStockCorpus(t)
	ensureBatchCorpus(t)
	repo := NewAnalyticsRepo(sqlc.New(planDB(t)))
	ctx := context.Background()
	got := map[string]string{}
	digest := func(key string, v any, apiErr error) {
		require.NoError(t, apiErr)
		got[key] = planDigest(t, v)
	}
	for _, c := range legacyPlanCases(false) {
		f := openOrderFilterOf(c.params)
		summary, apiErr := repo.GetOpenOrdersSummary(ctx, f)
		digest("open_summary/"+c.name, summary, nilIfNoErr(apiErr))
		products, apiErr := repo.GetOpenOrderProducts(ctx, domain.AnalyzeOpenOrderProductsParams{OpenOrderFilter: f, Limit: 10})
		digest("open_products/"+c.name, products, nilIfNoErr(apiErr))
		orders, apiErr := repo.ListOpenOrders(ctx, domain.ListOpenOrdersParams{OpenOrderFilter: f, Limit: 10})
		digest("open_orders/"+c.name, orders, nilIfNoErr(apiErr))
	}
	for _, c := range quarterlyPlanCases() {
		p := c.params
		quarters, apiErr := repo.GetQuarterlyOrders(ctx, domain.AnalyzeQuarterlyOrdersParams{AccountID: planAnaAccount, IssuedFrom: p.start,
			ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
		digest("quarterly/"+c.name, quarters, nilIfNoErr(apiErr))
	}
	for _, c := range demandForecastPlanCases() {
		demand, apiErr := repo.GetDemandForecastMonthlyDemand(ctx, c.params)
		digest("demand/"+c.name, demand, nilIfNoErr(apiErr))
		revenue, apiErr := repo.GetDemandForecastMonthlyRevenue(ctx, c.params)
		digest("revenue/"+c.name, revenue, nilIfNoErr(apiErr))
	}
	lines, _, apiErr := repo.GetOpenOrderLines(ctx, planAnaAccount, fmt.Sprintf("or_planana_%07d", planAnaInvoices-1), nil)
	digest("open_lines", lines, nilIfNoErr(apiErr))
	receipts, apiErr := repo.GetInventoryReceiptAnalytics(ctx, domain.AnalyzeInventoryReceiptsParams{AccountID: planInvAccount})
	digest("receipts", receipts, nilIfNoErr(apiErr))
	materials, apiErr := repo.GetMaterialAnalytics(ctx, domain.AnalyzeMaterialsParams{AccountID: planInvAccount})
	digest("materials", materials, nilIfNoErr(apiErr))
	batches, apiErr := NewBatchRepo(sqlc.New(planDB(t))).FindOpenBatches(ctx, planBatchAccount, nil, nil)
	digest("open_batches", batches, nilIfNoErr(apiErr))
	checkPlanResults(t, "analytics_parity.json", got)
}

func nilIfNoErr(apiErr *apierror.APIError) error {
	if apiErr == nil {
		return nil
	}
	return apiErr
}
