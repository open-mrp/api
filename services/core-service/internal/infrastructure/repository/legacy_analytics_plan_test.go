//go:build plans

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// The line-level analytics (sales entries, deliveries, open orders) join the corpus's invoices and
// orders out to their lines, items and categories; this adds what those joins need.
const planLegacyCorpusVersion = "Plan Legacy Analytics v3"

var planLegacyOnce sync.Once

func ensureLegacyAnalyticsCorpus(t *testing.T) {
	t.Helper()
	ensureDeliveryPerformanceCorpus(t)
	db := planDB(t)
	planLegacyOnce.Do(func() {
		var version string
		_ = db.QueryRow("SELECT name FROM item_category WHERE id = 'ic_planana'").Scan(&version)
		if version == planLegacyCorpusVersion {
			return
		}
		t.Log("seeding the line-level analytics plan corpus; it is kept for later runs")
		exec := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			require.NoError(t, err)
		}
		exec("DELETE FROM invoice_line WHERE id LIKE 'ivln\\_planana\\_%'")
		exec("DELETE FROM quantity WHERE id LIKE 'qy\\_planana\\_i%'")
		exec("DELETE FROM rate WHERE id LIKE 'rt\\_planana\\_%'")
		exec("DELETE FROM item WHERE account_id = ?", planAnaAccount)
		exec("DELETE FROM item_category WHERE id = 'ic_planana'")
		exec("DELETE FROM unit_group WHERE id = 'ug_planana'")
		exec("INSERT INTO unit_group (id, name, base_unit_id, unit_type_code) VALUES ('ug_planana', 'plantest', 'un_planana', 'quantity')")
		exec("INSERT INTO item_category (id, name, item_category_type_code, unit_group_id) VALUES ('ic_planana', ?, 'product', 'ug_planana')", planLegacyCorpusVersion)
		var itemVals []string
		var itemArgs []any
		for item := range planAnaItems {
			itemVals = append(itemVals, "(?, ?, ?, ?, ?, 'product', ?, 'ic_planana')")
			itemArgs = append(itemArgs, planAnaItemID(item), fmt.Sprintf("SKU-%05d", item), fmt.Sprintf("qy_planana_v%05d", item),
				fmt.Sprintf("qy_planana_b%05d", item), planAnaAccount, fmt.Sprintf("rt_planana_c%05d", item))
		}
		exec("INSERT INTO item (id, sku, unit_value_id, burn_rate_id, account_id, item_type_code, unit_cost_id, item_category_id) VALUES "+strings.Join(itemVals, ","), itemArgs...)

		const batch = 500
		for start := 0; start < planAnaInvoices; start += batch {
			var qVals, ilVals, rateVals []string
			var qArgs, ilArgs, rateArgs []any
			for i := start; i < start+batch; i++ {
				at := planAnaInvoicedAt(i)
				for line := range 1 + planHash(i, 3)%6 {
					rateVals = append(rateVals, "(?, ?, 'un_planana', 'un_planana', ?, ?)")
					rateArgs = append(rateArgs, fmt.Sprintf("rt_planana_%07d_%d", i, line), fmt.Sprintf("%d.%02d", 1+planHash(i*8+line, 6)%100, planHash(i, 9)%100), at, at)
					qID := fmt.Sprintf("qy_planana_i%07d_%d", i, line)
					qVals = append(qVals, "(?, ?, 'un_planana', ?, ?)")
					qArgs = append(qArgs, qID, 1+planHash(i*8+line, 5)%50, at, at)
					ilVals = append(ilVals, "(?, ?, ?, ?, ?, ?)")
					ilArgs = append(ilArgs, fmt.Sprintf("ivln_planana_%07d_%d", i, line), fmt.Sprintf("iv_planana_%07d", i), qID,
						fmt.Sprintf("sol_planana_%07d_%d", i, line), at, at)
				}
			}
			exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES "+strings.Join(qVals, ","), qArgs...)
			exec("INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id, created_at, updated_at) VALUES "+strings.Join(rateVals, ","), rateArgs...)
			exec("INSERT INTO invoice_line (id, invoice_id, quantity_id, sales_order_line_id, created_at, updated_at) VALUES "+strings.Join(ilVals, ","), ilArgs...)
		}
		exec("ANALYZE TABLE invoice_line, item, quantity, rate")
	})
}

// legacyPlanRequest is one request to the line-level analytics: a window and the shared filters.
type legacyPlanRequest struct {
	start, end                                                 time.Time
	productLineIDs, customerIDs, customerGroupIDs, salesRepIDs []string
}

func legacyPlanCases(windowed bool) []planCase[legacyPlanRequest] {
	type v = planValue[legacyPlanRequest]
	dims := []planDim[legacyPlanRequest]{
		{"customer", []v{
			{"customer=large", func(p *legacyPlanRequest) { p.customerIDs = []string{planAnaBuyerID(0)} }},
			{"customer=rare", func(p *legacyPlanRequest) { p.customerIDs = []string{planAnaBuyerID(planAnaRareBuyer)} }},
		}},
		{"group", []v{
			{"group=big", func(p *legacyPlanRequest) { p.customerGroupIDs = []string{"ag_planana_big"} }},
			{"group=small", func(p *legacyPlanRequest) { p.customerGroupIDs = []string{"ag_planana_small"} }},
		}},
		{"line", []v{
			{"line=large", func(p *legacyPlanRequest) { p.productLineIDs = []string{planAnaLineID(0)} }},
			{"line=rare", func(p *legacyPlanRequest) { p.productLineIDs = []string{planAnaLineID(planAnaRareLine)} }},
		}},
		{"rep", []v{
			{"rep=large", func(p *legacyPlanRequest) { p.salesRepIDs = []string{planAnaRepID(0)} }},
			{"rep=rare", func(p *legacyPlanRequest) { p.salesRepIDs = []string{planAnaRepID(planAnaRareRep)} }},
		}},
	}
	last := planAnaInvoicedAt(planAnaInvoices - 1)
	window := func(start, end time.Time) func(*legacyPlanRequest) {
		return func(p *legacyPlanRequest) { p.start, p.end = start, end }
	}
	windows := []planValue[legacyPlanRequest]{{"open", func(*legacyPlanRequest) {}}}
	if windowed {
		windows = []planValue[legacyPlanRequest]{
			{"30d", window(last.AddDate(0, 0, -30), last)},
			{"1y", window(last.AddDate(-1, 0, 0), last)},
			{"old-quarter", window(time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC))},
		}
	}
	return planCases(legacyPlanRequest{}, dims, windows)
}

// invoiceWindowFloor is a windowed line-level report's scope: the account's invoices in the window and
// their lines. Every filter is on the invoice's order, its lines' products or the buyer's relation,
// none on the invoice, so none narrows which invoices a key can pin.
func invoiceWindowFloor(t *testing.T, db *sql.DB, p legacyPlanRequest) (invoices, lines float64) {
	t.Helper()
	require.NoError(t, db.QueryRow(`SELECT COUNT(DISTINCT i.id), COUNT(il.id) FROM invoice i LEFT JOIN invoice_line il ON il.invoice_id = i.id
		WHERE i.account_id = ? AND i.created_at >= ? AND i.created_at <= ?`, planAnaAccount, p.start, p.end).Scan(&invoices, &lines))
	return invoices, lines
}

// TestSalesEntries_ReadsItsScope holds AnalyzeSales, every invoice line in a window with its order,
// product and pricing, to reading only the window's invoices and their lines (aggregatePlanSuite).
func TestSalesEntries_ReadsItsScope(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	aggregatePlanSuite[legacyPlanRequest]{
		tables: []aggregateTable{
			{table: "invoice", scopeColumn: "account_id", from: "JOIN invoice inv", alias: "inv"},
			{table: "invoice_line", scopeColumn: "invoice_id", from: "FROM invoice_line il", alias: "il"},
		},
		cases: legacyPlanCases(true),
		report: func(ctx context.Context, q *sqlc.Queries, p legacyPlanRequest) error {
			_, apiErr := NewAnalyticsRepo(q).GetSalesEntries(ctx, domain.AnalyzeSalesParams{AccountID: planAnaAccount, StartDate: p.start, EndDate: p.end,
				ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p legacyPlanRequest) map[string]float64 {
			invoices, lines := invoiceWindowFloor(t, db, p)
			return map[string]float64{"inv": invoices, "il": lines}
		},
	}.run(t)
}

// TestDeliveryEntries_ReadsItsScope holds AnalyzeDeliveries, every invoice in a window with its order's
// dates, to reading only the window's invoices (aggregatePlanSuite). It takes no filters.
func TestDeliveryEntries_ReadsItsScope(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	var cases []planCase[legacyPlanRequest]
	for _, c := range legacyPlanCases(true) {
		if strings.HasPrefix(c.name, "unfiltered/") {
			cases = append(cases, c)
		}
	}
	aggregatePlanSuite[legacyPlanRequest]{
		tables: []aggregateTable{{table: "invoice", scopeColumn: "account_id", from: "FROM invoice inv", alias: "inv"}},
		cases:  cases,
		report: func(ctx context.Context, q *sqlc.Queries, p legacyPlanRequest) error {
			_, apiErr := NewAnalyticsRepo(q).GetDeliveryAnalytics(ctx, domain.AnalyzeDeliveriesParams{AccountID: planAnaAccount, StartDate: p.start, EndDate: p.end})
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p legacyPlanRequest) map[string]float64 {
			invoices, _ := invoiceWindowFloor(t, db, p)
			return map[string]float64{"inv": invoices}
		},
	}.run(t)
}

// TestOrderEntries_ReadsItsScope holds AnalyzeOrders, every line of the open orders with what has been
// invoiced of it, to reading only the open orders (under whichever order-level filter matches fewest),
// their lines, and those lines' invoice lines (aggregatePlanSuite).
func TestOrderEntries_ReadsItsScope(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	aggregatePlanSuite[legacyPlanRequest]{
		tables: orderBookPlanTables,
		cases:  legacyPlanCases(false),
		report: func(ctx context.Context, q *sqlc.Queries, p legacyPlanRequest) error {
			_, apiErr := NewAnalyticsRepo(q).GetOrderEntries(ctx, domain.AnalyzeOrdersParams{AccountID: planAnaAccount,
				ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
			if apiErr != nil {
				return apiErr
			}
			return nil
		},
		floor: func(t *testing.T, db *sql.DB, p legacyPlanRequest) map[string]float64 {
			return orderBookFloor(t, db, false)
		},
	}.run(t)
}

// unorderedRows is rows as sorted JSON, for comparing results that order only by a timestamp the lines
// of one invoice or order share: their order among themselves is whatever the plan reads.
func unorderedRows[T any](t *testing.T, rows []T) []string {
	t.Helper()
	out := make([]string, len(rows))
	for i, r := range rows {
		raw, err := json.Marshal(r)
		require.NoError(t, err)
		out[i] = string(raw)
	}
	sort.Strings(out)
	return out
}

// TestLegacyAnalytics_ResultsUnchanged pins what every plan-tested line-level report returns.
func TestLegacyAnalytics_ResultsUnchanged(t *testing.T) {
	ensureLegacyAnalyticsCorpus(t)
	repo := NewAnalyticsRepo(sqlc.New(planDB(t)))
	ctx := context.Background()
	got := map[string]string{}
	for _, c := range legacyPlanCases(true) {
		p := c.params
		sales, apiErr := repo.GetSalesEntries(ctx, domain.AnalyzeSalesParams{AccountID: planAnaAccount, StartDate: p.start, EndDate: p.end,
			ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
		require.Nil(t, apiErr)
		got["sales/"+c.name] = planDigest(t, unorderedRows(t, sales))
		deliveries, apiErr := repo.GetDeliveryAnalytics(ctx, domain.AnalyzeDeliveriesParams{AccountID: planAnaAccount, StartDate: p.start, EndDate: p.end})
		require.Nil(t, apiErr)
		got["deliveries/"+c.name] = planDigest(t, deliveries)
	}
	for _, c := range legacyPlanCases(false) {
		p := c.params
		orders, apiErr := repo.GetOrderEntries(ctx, domain.AnalyzeOrdersParams{AccountID: planAnaAccount,
			ProductLineIDs: p.productLineIDs, CustomerIDs: p.customerIDs, SalesRepIDs: p.salesRepIDs, CustomerGroupIDs: p.customerGroupIDs})
		require.Nil(t, apiErr)
		got["orders/"+c.name] = planDigest(t, unorderedRows(t, orders))
	}
	checkPlanResults(t, "legacy_analytics.json", got)
}
