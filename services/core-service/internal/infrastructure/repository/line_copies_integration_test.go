//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/apikit/db/backfill"
	"github.com/open-mrp/apikit/field"
)

// The seed account's own records the line-copy fixtures hang off.
const (
	lcSeller   = "ac_01k0a5smf9ekb8rqg12555zjqa"
	lcBuyer    = "ac_01k09wm2fgevdsc344gpbcj30f"
	lcBilling  = "ad_01k09wnac0e1ar211e0sy0ba4g"
	lcShipping = "ad_01k09wnpvrea0awz7vem2j8j7g"
	lcProduct  = "pd_01k0a65nx2e2crfxrvryyxnmdh"
	lcProduct2 = "pd_01k0a65nx5e3haz2fgfm34hmcz"
)

// lcExec runs fixture SQL, failing the test on error.
func lcExec(t *testing.T, pool *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := pool.Exec(query, args...)
	require.NoError(t, err, query)
}

// lcOrder inserts an issued order of the given type due today, and removes it, its lines, their invoice
// and pick lines and every quantity and rate they own when the test ends.
func lcOrder(t *testing.T, pool *sql.DB, orderType string) string {
	t.Helper()
	orderID := fmt.Sprintf("or_lctest%d", time.Now().UnixNano())
	lcExec(t, pool, `INSERT INTO sales_order (id, number, billing_address_id, shipping_address_id, priority_code, sales_order_status_code,
		sales_order_type_code, buyer_account_id, seller_account_id, owner_account_id, issued_at, ship_by_date)
		VALUES (?, ?, ?, ?, 'normal', 'issued', ?, ?, ?, ?, NOW(3), UTC_DATE())`,
		orderID, orderID, lcBilling, lcShipping, orderType, lcBuyer, lcSeller, lcSeller)
	t.Cleanup(func() {
		lines := "SELECT id FROM sales_order_line WHERE sales_order_id = '" + orderID + "'"
		for _, stmt := range []string{
			"DELETE q FROM quantity q JOIN invoice_line il ON il.quantity_id = q.id WHERE il.sales_order_line_id IN (" + lines + ")",
			"DELETE q FROM quantity q JOIN pick_line pl ON pl.quantity_id = q.id WHERE pl.sales_order_line_id IN (" + lines + ")",
			"DELETE FROM invoice_line WHERE sales_order_line_id IN (" + lines + ")",
			"DELETE FROM pick_line WHERE sales_order_line_id IN (" + lines + ")",
			"DELETE FROM invoice WHERE sales_order_id = '" + orderID + "'",
			"DELETE FROM pick WHERE sales_order_id = '" + orderID + "'",
			"DELETE q FROM quantity q JOIN sales_order_line sol ON sol.quantity_id = q.id WHERE sol.sales_order_id = '" + orderID + "'",
			"DELETE r FROM rate r JOIN sales_order_line sol ON r.id IN (sol.unit_price_id, sol.unit_cost_id) WHERE sol.sales_order_id = '" + orderID + "'",
			"DELETE FROM sales_order_line WHERE sales_order_id = '" + orderID + "'",
			"DELETE FROM sales_order WHERE id = '" + orderID + "'",
		} {
			_, _ = pool.Exec(stmt)
		}
	})
	return orderID
}

// lcLine inserts an order line the way the SQL that bypasses the repositories does, with no copies.
// qty or price "" leaves the line pointing at a row that does not exist; cost "" gives it no cost.
func lcLine(t *testing.T, pool *sql.DB, orderID, lineID, productID, qty, qtyUnit, price, priceNum, priceDen, cost, costNum, costDen string) {
	t.Helper()
	qID, pID := "qu_"+lineID, "rt_p_"+lineID
	if qty != "" {
		lcExec(t, pool, "INSERT INTO quantity (id, value, unit_id) VALUES (?, ?, ?)", qID, qty, qtyUnit)
	}
	if price != "" {
		lcExec(t, pool, "INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES (?, ?, ?, ?)", pID, price, priceNum, priceDen)
	}
	var cID any
	if cost != "" {
		cID = "rt_c_" + lineID
		lcExec(t, pool, "INSERT INTO rate (id, value, numerator_unit_id, denominator_unit_id) VALUES (?, ?, ?, ?)", cID, cost, costNum, costDen)
	}
	lcExec(t, pool, `INSERT INTO sales_order_line (id, product_sku, line_item_number, product_id, sales_order_id, quantity_id, unit_price_id, unit_cost_id)
		VALUES (?, 'SKU', 1, ?, ?, ?, ?, ?)`, lineID, productID, orderID, qID, pID, cID)
}

// lcFulfillmentLine inserts an invoice or pick line on an order line, with no copy. qty "" leaves it pointing
// at a quantity row that does not exist.
func lcFulfillmentLine(t *testing.T, pool *sql.DB, table, parentColumn, parentID, lineID, orderLineID, qty, unit string, packed bool) {
	t.Helper()
	qID := "qu_" + lineID
	if qty != "" {
		lcExec(t, pool, "INSERT INTO quantity (id, value, unit_id) VALUES (?, ?, ?)", qID, qty, unit)
	}
	packedCol, packedVal := "", ""
	if packed {
		packedCol, packedVal = ", packed_at", ", NOW(3)"
	}
	lcExec(t, pool, "INSERT INTO "+table+" (id, "+parentColumn+", quantity_id, sales_order_line_id"+packedCol+") VALUES (?, ?, ?, ?"+packedVal+")",
		lineID, parentID, qID, orderLineID)
}

// lcStaleCopies counts the order's order, invoice and pick lines whose copies differ from the rows they copy.
func lcStaleCopies(t *testing.T, pool *sql.DB, orderID string) int {
	t.Helper()
	var stale int
	require.NoError(t, pool.QueryRow(`SELECT
		(SELECT COUNT(*) FROM sales_order_line sol
			LEFT JOIN quantity q ON q.id = sol.quantity_id
			LEFT JOIN rate p ON p.id = sol.unit_price_id
			LEFT JOIN rate c ON c.id = sol.unit_cost_id
			WHERE sol.sales_order_id = ? AND NOT (sol.quantity_value <=> q.value AND sol.quantity_unit_id <=> q.unit_id
				AND sol.unit_price_value <=> p.value AND sol.unit_price_numerator_unit_id <=> p.numerator_unit_id
				AND sol.unit_price_denominator_unit_id <=> p.denominator_unit_id AND sol.unit_cost_value <=> c.value
				AND sol.unit_cost_numerator_unit_id <=> c.numerator_unit_id AND sol.unit_cost_denominator_unit_id <=> c.denominator_unit_id))
		+ (SELECT COUNT(*) FROM invoice_line il JOIN sales_order_line sol ON sol.id = il.sales_order_line_id
			LEFT JOIN quantity q ON q.id = il.quantity_id
			WHERE sol.sales_order_id = ? AND NOT (il.quantity_value <=> q.value AND il.quantity_unit_id <=> q.unit_id))
		+ (SELECT COUNT(*) FROM pick_line pl JOIN sales_order_line sol ON sol.id = pl.sales_order_line_id
			LEFT JOIN quantity q ON q.id = pl.quantity_id
			WHERE sol.sales_order_id = ? AND NOT (pl.quantity_value <=> q.value AND pl.quantity_unit_id <=> q.unit_id))`,
		orderID, orderID, orderID).Scan(&stale))
	return stale
}

// useLineCopies sets whether reports read the copies, restoring the latch when the test ends. The progress
// row is set aside so the latch alone decides.
func useLineCopies(t *testing.T, pool *sql.DB, on bool) {
	t.Helper()
	prev := lineCopiesDone.Load()
	lineCopiesDone.Store(on)
	var cursor string
	var completed sql.NullTime
	err := pool.QueryRow("SELECT cursor_value, completed_at FROM backfill_progress WHERE name = ?", LineCopyBackfillName).Scan(&cursor, &completed)
	if err == nil {
		lcExec(t, pool, "DELETE FROM backfill_progress WHERE name = ?", LineCopyBackfillName)
		t.Cleanup(func() {
			_, _ = pool.Exec("INSERT INTO backfill_progress (name, cursor_value, completed_at) VALUES (?, ?, ?)", LineCopyBackfillName, cursor, completed)
		})
	}
	t.Cleanup(func() { lineCopiesDone.Store(prev) })
}

// lcBookFixture is an order exercising every way a line's values can be stored: quantity and price in
// different units (dozens priced per pair, cost per each), no cost, a missing quantity row, a missing price
// row, invoiced and picked lines, and an invoice and a pick line whose quantity row is missing.
func lcBookFixture(t *testing.T, pool *sql.DB) string {
	t.Helper()
	orderID := lcOrder(t, pool, "sales_order")
	l1, l2, l3, l4 := orderID+"_l1", orderID+"_l2", orderID+"_l3", orderID+"_l4"
	lcLine(t, pool, orderID, l1, lcProduct, "3", "dozen", "1.5", "dollar", "pair", "0.4", "dollar", "each")
	lcLine(t, pool, orderID, l2, lcProduct2, "5", "pair", "2", "dollar", "pair", "", "", "")
	lcLine(t, pool, orderID, l3, lcProduct, "", "", "4.25", "dollar", "pair", "", "", "")
	lcLine(t, pool, orderID, l4, lcProduct2, "7", "pair", "", "", "", "1", "dollar", "pair")
	invoiceID, pickID := orderID+"_inv", orderID+"_pk"
	lcFulfillmentLine(t, pool, "invoice_line", "invoice_id", invoiceID, orderID+"_il1", l1, "1", "dozen", false)
	lcFulfillmentLine(t, pool, "invoice_line", "invoice_id", invoiceID, orderID+"_il2", l2, "", "", false)
	lcFulfillmentLine(t, pool, "invoice_line", "invoice_id", invoiceID, orderID+"_il3", l2, "2", "pair", false)
	lcFulfillmentLine(t, pool, "pick_line", "pick_id", pickID, orderID+"_pl1", l1, "2", "dozen", true)
	lcFulfillmentLine(t, pool, "pick_line", "pick_id", pickID, orderID+"_pl2", l2, "1", "pair", false)
	lcFulfillmentLine(t, pool, "pick_line", "pick_id", pickID, orderID+"_pl3", l2, "3", "pair", true)
	lcFulfillmentLine(t, pool, "pick_line", "pick_id", pickID, orderID+"_pl4", l2, "", "", true)
	return orderID
}

// runLineCopyBackfill runs the backfill over every line in the database, from the start to the end.
func runLineCopyBackfill(t *testing.T, q *sqlc.Queries) {
	t.Helper()
	b := NewLineCopyBackfill(q, q)
	cursor := ""
	for {
		next, _, done, err := b.Batch(context.Background(), &backfill.Meter{}, cursor, 50)
		require.NoError(t, err)
		if done {
			return
		}
		cursor = next
	}
}

// TestLineCopies_ReportsReadTheSameFromTheCopies fills an account's copies with the backfill, then requires
// every report that reads them to answer exactly as it does from the quantity and rate rows.
func TestLineCopies_ReportsReadTheSameFromTheCopies(t *testing.T) {
	pool := testDB(t)
	q := sqlc.New(pool)
	orderID := lcBookFixture(t, pool)
	require.Positive(t, lcStaleCopies(t, pool, orderID), "the fixture starts uncopied")

	runLineCopyBackfill(t, q)
	require.Zero(t, lcStaleCopies(t, pool, orderID), "the backfill copies every line")

	ctx := context.Background()
	analytics := NewAnalyticsRepo(q)
	delivery := NewProductionScheduleInputRepo(q)
	var productLine string
	require.NoError(t, pool.QueryRow("SELECT product_line_id FROM product WHERE id = ?", lcProduct).Scan(&productLine))
	var item string
	require.NoError(t, pool.QueryRow("SELECT item_id FROM product WHERE id = ?", lcProduct2).Scan(&item))

	filters := map[string]domain.OpenOrderFilter{
		"account":      {AccountID: lcSeller},
		"customer":     {AccountID: lcSeller, CustomerIDs: []string{lcBuyer}},
		"product line": {AccountID: lcSeller, ProductLineIDs: []string{productLine}},
		"item":         {AccountID: lcSeller, ItemIDs: []string{item}},
	}
	read := func() map[string]any {
		out := map[string]any{}
		for name, f := range filters {
			summary, apiErr := analytics.GetOpenOrdersSummary(ctx, f)
			require.Nil(t, apiErr)
			products, apiErr := analytics.GetOpenOrderProducts(ctx, domain.AnalyzeOpenOrderProductsParams{OpenOrderFilter: f, Limit: 100})
			require.Nil(t, apiErr)
			orders, apiErr := analytics.ListOpenOrders(ctx, domain.ListOpenOrdersParams{OpenOrderFilter: f, Limit: 100})
			require.Nil(t, apiErr)
			lines, apiErr := analytics.GetOpenOrderLineEntries(ctx, f, 1000)
			require.Nil(t, apiErr)
			entries, apiErr := analytics.GetOrderEntries(ctx, domain.AnalyzeOrdersParams{AccountID: f.AccountID, CustomerIDs: f.CustomerIDs, ProductLineIDs: f.ProductLineIDs})
			require.Nil(t, apiErr)
			outcomes, apiErr := delivery.ListDeliveryOutcomes(ctx, lcSeller, time.Now().UTC().AddDate(0, 0, -2), time.Now().UTC().AddDate(0, 0, 2),
				domain.DeliveryFilters{CustomerIDs: f.CustomerIDs, ProductLineIDs: f.ProductLineIDs})
			require.Nil(t, apiErr)
			out[name] = []any{summary, products, orders, lines, entries, outcomes}
		}
		lines, found, apiErr := analytics.GetOpenOrderLines(ctx, lcSeller, orderID, nil)
		require.Nil(t, apiErr)
		require.True(t, found)
		out["order lines"] = lines
		return out
	}

	useLineCopies(t, pool, false)
	fromRows := read()
	lineCopiesDone.Store(true)
	fromCopies := read()
	require.Equal(t, fromRows, fromCopies)

	lines := fromCopies["order lines"].([]domain.OpenOrderLine)
	require.Len(t, lines, 4, "the fixture's lines are all read")
}

// TestLineCopies_SyncFillsAnAccountsOrders is the sandbox seed's path: lines written by plain SQL are copied
// for the account in one call.
func TestLineCopies_SyncFillsAnAccountsOrders(t *testing.T) {
	pool := testDB(t)
	orderID := lcBookFixture(t, pool)
	require.Positive(t, lcStaleCopies(t, pool, orderID))
	require.NoError(t, syncAccountOrderCopies(context.Background(), pool, lcSeller))
	require.Zero(t, lcStaleCopies(t, pool, orderID))
}

// TestLineCopies_EveryWriteKeepsThemCurrent drives each repository write that creates or changes a line's
// quantity or rate, or links a line to one, and requires the copies to match the rows after each.
func TestLineCopies_EveryWriteKeepsThemCurrent(t *testing.T) {
	pool := testDB(t)
	q := sqlc.New(pool)
	ctx := context.Background()
	str := func(s string) *string { return &s }
	orderID := lcOrder(t, pool, "sales_order")
	current := func(step string) {
		t.Helper()
		require.Zero(t, lcStaleCopies(t, pool, orderID), "copies are current after %s", step)
	}

	lines := NewSalesOrderLineRepo(q)
	line1 := orderID + "_l1"
	_, apiErr := lines.Create(ctx, line1, domain.CreateSalesOrderLineParams{
		SalesOrderID: orderID, ProductID: lcProduct, ProductSKU: "SCK-001",
		QuantityValue: "3", QuantityUnitID: "dozen",
		UnitPriceValue: "1.5", UnitPriceNumeratorUnitID: "dollar", UnitPriceDenominatorUnitID: "pair",
		UnitCostValue: str("0.4"), UnitCostNumeratorUnitID: str("dollar"), UnitCostDenominatorUnitID: str("each"),
	})
	require.Nil(t, apiErr)
	current("creating a line")
	var copied sql.NullString
	require.NoError(t, pool.QueryRow("SELECT unit_cost_denominator_unit_id FROM sales_order_line WHERE id = ?", line1).Scan(&copied))
	require.Equal(t, "each", copied.String, "the copies are written, not left NULL")

	line2 := orderID + "_l2"
	_, apiErr = lines.Create(ctx, line2, domain.CreateSalesOrderLineParams{
		SalesOrderID: orderID, ProductID: lcProduct2, ProductSKU: "SCK-002",
		QuantityValue: "5", QuantityUnitID: "pair",
		UnitPriceValue: "2", UnitPriceNumeratorUnitID: "dollar", UnitPriceDenominatorUnitID: "pair",
	})
	require.Nil(t, apiErr)
	current("creating a line with no cost")

	require.Nil(t, lines.CreateMany(ctx, []domain.CreateSalesOrderLineParams{
		{SalesOrderID: orderID, ProductID: lcProduct, ProductSKU: "SCK-001", QuantityValue: "1", QuantityUnitID: "each",
			UnitPriceValue: "3", UnitPriceNumeratorUnitID: "dollar", UnitPriceDenominatorUnitID: "each",
			UnitCostValue: str("1"), UnitCostNumeratorUnitID: str("dollar"), UnitCostDenominatorUnitID: str("each")},
		{SalesOrderID: orderID, ProductID: lcProduct2, ProductSKU: "SCK-002", QuantityValue: "2", QuantityUnitID: "dozen",
			UnitPriceValue: "9", UnitPriceNumeratorUnitID: "dollar", UnitPriceDenominatorUnitID: "dozen"},
	}))
	current("creating lines in bulk")

	_, apiErr = lines.Update(ctx, domain.UpdateSalesOrderLineParams{SalesOrderLineID: line1, SalesOrderID: orderID, AccountID: lcSeller,
		ProductDescription: field.Clearable[string]{},
		QuantityValue:      str("4"), QuantityUnitID: str("pair"), UnitPriceValue: str("1.75"), UnitPriceDenominatorUnitID: str("each"),
		UnitCostValue: str("0.5")})
	require.Nil(t, apiErr)
	current("changing a line's quantity, price and cost")

	_, apiErr = lines.Update(ctx, domain.UpdateSalesOrderLineParams{SalesOrderLineID: line2, SalesOrderID: orderID, AccountID: lcSeller,
		UnitCostValue: str("0.9"), UnitCostNumeratorUnitID: str("dollar"), UnitCostDenominatorUnitID: str("pair")})
	require.Nil(t, apiErr)
	current("giving a line a cost")

	invoices := NewInvoiceRepo(q)
	_, apiErr = invoices.CreateFromShipment(ctx, domain.CreateInvoiceFromShipmentParams{AccountID: lcSeller, InvoiceID: orderID + "_inv",
		Number: orderID, SalesOrderID: orderID, ShippedLines: []domain.InvoiceLineDraft{{SalesOrderLineID: line1, QuantityValue: "4", QuantityUnitID: "pair"}}})
	require.Nil(t, apiErr)
	current("invoicing")

	require.Nil(t, lines.SyncInvoiceLineQuantities(ctx, line1, "4", "6", "each"))
	current("following an order line's quantity onto its invoice lines")

	pickID := orderID + "_pk"
	lcExec(t, pool, "INSERT INTO pick (id, number, sales_order_id, account_id) VALUES (?, ?, ?, ?)", pickID, pickID, orderID, lcSeller)
	picks := NewPickLineRepo(q)
	require.Nil(t, lines.CreateQuantity(ctx, "qu_"+orderID+"_pl1", "0", "pair"))
	require.Nil(t, picks.CreateForRemaining(ctx, orderID+"_pl1", "qu_"+orderID+"_pl1", pickID, line1))
	current("creating a pick line")
	require.Nil(t, picks.UpdateQuantity(ctx, orderID+"_pl1", str("2"), nil))
	current("picking a quantity")
	require.Nil(t, picks.PickRemainingQuantity(ctx, orderID+"_pl1"))
	current("picking the remainder")
	require.Nil(t, picks.VoidLine(ctx, orderID+"_pl1"))
	current("voiding a pick line")

	require.Nil(t, lines.CreateQuantity(ctx, "qu_"+orderID+"_pl2", "0", "pair"))
	require.Nil(t, NewSalesOrderRepo(q).CreatePickLine(ctx, orderID+"_pl2", pickID, "qu_"+orderID+"_pl2", line2))
	current("creating an order's pick line")
	pickRepo := NewPickRepo(q)
	require.Nil(t, pickRepo.PickAllLines(ctx, pickID))
	current("picking a whole pick")
	require.Nil(t, pickRepo.VoidAllLines(ctx, pickID))
	current("voiding a whole pick")
	require.Nil(t, lines.SyncPickLineQuantityUnits(ctx, line2, "dozen"))
	current("relabeling pick lines to the order line's unit")

	poID := lcOrder(t, pool, "purchase_order")
	poLines := NewPurchaseOrderLineRepo(q)
	po, apiErr := poLines.Create(ctx, poID+"_l1", domain.CreatePurchaseOrderLineParams{SalesOrderID: poID, AccountID: lcSeller, ProductID: str(lcProduct),
		ProductSKU: "SCK-001", QuantityValue: "10", QuantityUnitID: "pair",
		UnitPriceValue: "1", UnitPriceNumeratorUnitID: "dollar", UnitPriceDenominatorUnitID: "pair",
		UnitCostValue: str("1"), UnitCostNumeratorUnitID: str("dollar"), UnitCostDenominatorUnitID: str("pair")})
	require.Nil(t, apiErr)
	require.Zero(t, lcStaleCopies(t, pool, poID), "after creating a purchase order line")
	_, apiErr = poLines.Update(ctx, domain.UpdatePurchaseOrderLineParams{PurchaseOrderLineID: po.ID, SalesOrderID: poID, AccountID: lcSeller,
		QuantityValue: str("12"), UnitPriceValue: str("1.1"), UnitCostValue: str("1.2")})
	require.Nil(t, apiErr)
	require.Zero(t, lcStaleCopies(t, pool, poID), "after changing a purchase order line")
	var qID, pID string
	require.NoError(t, pool.QueryRow("SELECT quantity_id, unit_price_id FROM sales_order_line WHERE id = ?", po.ID).Scan(&qID, &pID))
	require.Nil(t, poLines.UpdateQuantityValue(ctx, qID, "14"))
	require.Nil(t, poLines.UpdateRateValue(ctx, pID, "1.3", nil, nil))
	require.Zero(t, lcStaleCopies(t, pool, poID), "after changing a purchase order line's quantity and rate rows")
}
