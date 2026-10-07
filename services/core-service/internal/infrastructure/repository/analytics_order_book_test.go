package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/pagination"
)

// A Go-built query binds positionally, so a predicate added without its arguments (or the reverse) shifts every later value onto the wrong column.
func requireBindsMatch(t *testing.T, query string, args []any) {
	t.Helper()
	require.Equal(t, strings.Count(query, "?"), len(args), "placeholders and arguments differ:\n%s", query)
}

func TestOrderBookScope_EmitsOnlyTheFiltersSet(t *testing.T) {
	t.Parallel()

	bare := orderBookScope{accountID: "ac_1"}
	query, args := bare.fromWhere("")
	requireBindsMatch(t, query, args)
	assert.Equal(t, []any{"ac_1", "ac_1"}, args, "the account scopes the orders and the invoiced quantities alike")
	assert.Contains(t, query, "so.sales_order_type_code = 'sales_order'", "purchase orders live in sales_order and are not sales")
	assert.Contains(t, query, "inv_so.sales_order_type_code = 'sales_order'")
	assert.Contains(t, query, "so.sales_order_status_code = 'issued'")
	assert.Contains(t, query, "fg.product_type_code = 'sale'")
	assert.NotContains(t, query, "completed_at")
	assert.NotContains(t, query, " IN (?")

	full := orderBookScope{
		accountID: "ac_1", openOnly: true, orderIDs: []string{"or_1"},
		buyers: []string{"b_1", "b_2"}, buyersFiltered: true,
		salesRepIDs: []string{"r_1"}, productLineIDs: []string{"pl_1"}, itemIDs: []string{"it_1"},
	}
	query, args = full.fromWhere("")
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "so.completed_at IS NULL")
	assert.Contains(t, query, "inv_so.completed_at IS NULL", "the invoiced-quantity aggregate reads only the orders the report reads")
	// Order-level filters bind in both scopes, then the line filters once.
	assert.Equal(t, []any{
		"ac_1", "or_1", "b_1", "b_2", "r_1",
		"ac_1", "or_1", "b_1", "b_2", "r_1",
		"pl_1", "it_1",
	}, args)
}

func TestOrderBookScope_AnyStatusReadsEveryOrderOfTheType(t *testing.T) {
	t.Parallel()

	query, args := orderBookScope{accountID: "ac_1", anyStatus: true, orderIDs: []string{"or_1"}}.fromWhere("")
	requireBindsMatch(t, query, args)
	assert.NotContains(t, query, "sales_order_status_code")
	assert.Contains(t, query, "so.sales_order_type_code = 'sales_order'")
}

// A buyer filter that resolved to nobody still emits its predicate rather than vanishing and widening the report; the repository answers it empty before querying.
func TestOrderBookScope_FilteredBuyersAlwaysBind(t *testing.T) {
	t.Parallel()

	preds, args := orderBookScope{accountID: "ac_1", buyers: []string{"b_1"}, buyersFiltered: true}.orderPredicates("so")
	assert.Contains(t, preds, "so.buyer_account_id IN (?)")
	assert.Equal(t, []any{"ac_1", "b_1"}, args)
}

func TestOpenOrderProductsQuery_PagesByBackOrderedThenItem(t *testing.T) {
	t.Parallel()

	scope := orderBookScope{accountID: "ac_1", openOnly: true}
	query, args := openOrderProductsQuery(scope, nil, 10)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "WHERE g.qty_ordered > 0\nORDER BY g.qty_back_ordered DESC, g.item_id ASC")
	assert.Equal(t, int32(11), args[len(args)-1], "one row past the page tells whether another follows")

	forward := &pagination.ValueCursor{Value: "12.5", ID: "it_9", Direction: pagination.DirectionForward}
	query, args = openOrderProductsQuery(scope, forward, 10)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "g.qty_back_ordered < CAST(? AS DECIMAL(65,30)) OR (g.qty_back_ordered = CAST(? AS DECIMAL(65,30)) AND g.item_id > ?)")
	assert.Equal(t, []any{"12.5", "12.5", "it_9", int32(11)}, args[len(args)-4:])

	backward := &pagination.ValueCursor{Value: "12.5", ID: "it_9", Direction: pagination.DirectionBackward}
	query, args = openOrderProductsQuery(scope, backward, 10)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "g.qty_back_ordered > CAST(? AS DECIMAL(65,30)) OR (g.qty_back_ordered = CAST(? AS DECIMAL(65,30)) AND g.item_id < ?)")
	assert.Contains(t, query, "ORDER BY g.qty_back_ordered ASC, g.item_id DESC")
}

func TestOpenOrderPageQuery_ChoosesOrdersByTheirCountedLines(t *testing.T) {
	t.Parallel()

	scope := orderBookScope{accountID: "ac_1", openOnly: true, productLineIDs: []string{"pl_1"}}
	query, args := openOrderPageQuery(scope, nil, 25)
	requireBindsMatch(t, query, args)
	assert.True(t, strings.HasPrefix(query, "SELECT so.id FROM sales_order so WHERE"), "the page is chosen from the orders alone")
	assert.Contains(t, query, "EXISTS (SELECT 1 FROM sales_order_line sol JOIN product fg ON fg.id = sol.product_id WHERE sol.sales_order_id = so.id AND fg.product_type_code = 'sale' AND fg.product_line_id IN (?))")
	assert.True(t, strings.HasSuffix(query, "ORDER BY so.issued_at DESC, so.id DESC LIMIT ?"))
	assert.Equal(t, []any{"ac_1", "pl_1", int32(26)}, args)

	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	query, args = openOrderPageQuery(scope, &pagination.StringCursor{OccurredAt: at, ID: "or_5", Direction: pagination.DirectionBackward}, 25)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "(so.issued_at > ? OR (so.issued_at = ? AND so.id > ?))")
	assert.True(t, strings.HasSuffix(query, "ORDER BY so.issued_at ASC, so.id ASC LIMIT ?"))
}

func TestOpenOrderLinesAndTotalsQueries_Bind(t *testing.T) {
	t.Parallel()

	query, args := openOrderLinesQuery(orderBookScope{accountID: "ac_1", anyStatus: true, orderIDs: []string{"or_1"}})
	requireBindsMatch(t, query, args)
	assert.True(t, strings.HasSuffix(query, "ORDER BY pb.sku ASC, sol.id ASC"))

	query, args = openOrderTotalsQuery(orderBookScope{accountID: "ac_1", openOnly: true, orderIDs: []string{"or_1", "or_2"}, itemIDs: []string{"it_1"}})
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "COUNT(DISTINCT sol.id)")

	query, args = openOrdersSummaryQuery(orderBookScope{accountID: "ac_1", openOnly: true})
	requireBindsMatch(t, query, args)

	query, args = orderEntriesQuery(orderBookScope{accountID: "ac_1"}, 0)
	requireBindsMatch(t, query, args)
	assert.True(t, strings.HasSuffix(query, "ORDER BY so.issued_at ASC, so.id ASC, sol.id ASC"))
	query, args = orderEntriesQuery(orderBookScope{accountID: "ac_1", openOnly: true}, 50_001)
	requireBindsMatch(t, query, args)
	assert.Equal(t, 50_001, args[len(args)-1])
}

func TestValueKeysetPage(t *testing.T) {
	t.Parallel()

	rows := []string{"a", "b", "c"}
	ranks := []string{"3", "2", "1"}
	id := func(s string) string { return s }

	page, info := valueKeysetPage(rows, ranks, 2, nil, id)
	assert.Equal(t, []string{"a", "b"}, page)
	assert.True(t, info.HasNextPage)
	assert.False(t, info.HasPrevPage)
	require.NotNil(t, info.NextCursor)
	next, err := pagination.DecodeValueCursor(*info.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, pagination.ValueCursor{Value: "2", ID: "b", Direction: pagination.DirectionForward}, next)

	// A backward page arrives in reverse ranking order and is turned back around.
	back := &pagination.ValueCursor{Value: "0", ID: "z", Direction: pagination.DirectionBackward}
	page, info = valueKeysetPage([]string{"c", "b"}, []string{"1", "2"}, 2, back, id)
	assert.Equal(t, []string{"b", "c"}, page)
	assert.True(t, info.HasNextPage)
	assert.False(t, info.HasPrevPage)

	page, info = valueKeysetPage([]string{}, []string{}, 2, nil, id)
	assert.Empty(t, page)
	assert.Equal(t, pagination.PageInfo{}, info)
}

func TestQuarterlyOrdersQuery_EmitsOnlyTheFiltersSet(t *testing.T) {
	t.Parallel()

	from := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	query, args := quarterlyOrdersQuery("sales_order_owner_type_issued_idx", "ac_1", from, nil, false, nil, nil, nil)
	requireBindsMatch(t, query, args)
	assert.Equal(t, []any{"ac_1", from}, args)
	assert.Contains(t, query, "so.issued_at >= ?", "the bound also keeps estimates, which have no issue date, out")
	assert.Contains(t, query, "so.sales_order_type_code = 'sales_order'")
	assert.NotContains(t, query, "sales_order_status_code", "an order counts whatever its status now")

	query, args = quarterlyOrdersQuery("sales_order_owner_buyer_created_idx", "ac_1", from, []string{"b_1"}, true, []string{"r_1"}, []string{"pl_1"}, []string{"it_1"})
	requireBindsMatch(t, query, args)
	assert.Equal(t, []any{"ac_1", from, "b_1", "r_1", "pl_1", "it_1"}, args)
}

func TestAddQuarter_TotalsTheQuartersInOrder(t *testing.T) {
	t.Parallel()

	var d domain.QuarterlyData
	addQuarter(&d, 2, 10.25)
	addQuarter(&d, 4, 5)
	assert.Equal(t, domain.QuarterlyData{Q2: 10.25, Q4: 5, Total: 15.25}, d)
}

func TestInventoryReceiptSummaryQuery_PushesFiltersDown(t *testing.T) {
	t.Parallel()

	query, args := inventoryReceiptSummaryQuery(domain.AnalyzeInventoryReceiptsParams{AccountID: "ac_1"})
	requireBindsMatch(t, query, args)
	assert.Equal(t, []any{"ac_1", "ac_1"}, args)
	assert.NotContains(t, query, "ir.item_id IN")
	assert.True(t, strings.HasSuffix(query, "ORDER BY g.oldest ASC, g.item_id ASC, g.storage_location_id ASC, g.lot_id ASC, g.owner_account_id ASC, g.holder_account_id ASC"),
		"the dashboard listed groups oldest receipt first")

	query, args = inventoryReceiptSummaryQuery(domain.AnalyzeInventoryReceiptsParams{AccountID: "ac_1", ItemIDs: []string{"it_1", "it_2"}, LocationIDs: []string{"sl_1"}, LotIDs: []string{"lt_1"}})
	requireBindsMatch(t, query, args)
	assert.Equal(t, []any{"ac_1", "ac_1", "it_1", "it_2", "sl_1", "lt_1"}, args)
	assert.Contains(t, query, "ir.item_id IN (?,?)")
	assert.Contains(t, query, "ir.storage_location_id IN (?)")
	assert.Contains(t, query, "ir.lot_id IN (?)")
	assert.NotContains(t, query, "GROUP BY r.item_id, r.storage_location_id, r.lot_id, r.owner_account_id, r.holder_account_id, ", "groups never split by unit")
}

func TestDemandForecastQueries_PushFiltersDown(t *testing.T) {
	t.Parallel()

	start, end := time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	w := domain.GetDemandForecastWindowParams{AccountID: "ac_1", StartDate: start, EndDate: end}

	query, args := demandForecastDemandQuery(w)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "so.sales_order_type_code = 'sales_order'")
	assert.NotContains(t, query, "fg.product_line_id IN")

	query, args = demandForecastRevenueQuery(w)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "inv.account_id = ?")
	assert.Contains(t, query, "FORCE INDEX (invoice_account_created_idx)")

	w.ProductLineIDs, w.ItemIDs = []string{"pl_1"}, []string{"it_1", "it_2"}
	for _, build := range []func(domain.GetDemandForecastWindowParams) (string, []any){demandForecastDemandQuery, demandForecastRevenueQuery} {
		query, args = build(w)
		requireBindsMatch(t, query, args)
		assert.Contains(t, query, "fg.product_line_id IN (?)")
		assert.Contains(t, query, "pb.id IN (?,?)")
		assert.Equal(t, []any{"ac_1", start, end, "pl_1", "it_1", "it_2"}, args)
	}
}

func TestOpenBatchSummaryQuery_GroupsByStationAndItem(t *testing.T) {
	t.Parallel()

	query, args := openBatchSummaryQuery("batch_account_closed_scanned_idx", "ac_1", nil)
	requireBindsMatch(t, query, args)
	assert.Contains(t, query, "GROUP BY b.scanning_station_id, b.item_id\n", "a station's item is one row whatever units its batches were recorded in")
	assert.NotContains(t, query, "b.item_id IN")

	query, args = openBatchSummaryQuery("batch_item_id_idx", "ac_1", []string{"it_1", "it_2"})
	requireBindsMatch(t, query, args)
	assert.Equal(t, []any{"ac_1", "it_1", "it_2"}, args)
	assert.True(t, strings.HasSuffix(query, "ORDER BY d.name ASC, g.scanning_station_id ASC, g.total_count DESC, g.item_id ASC"))
}

func TestFromDimensionBase(t *testing.T) {
	t.Parallel()

	d := func(s string) decimal.Decimal { return decimal.RequireFromString(s) }
	cases := []struct {
		name           string
		base           string
		rn, rd, on, od string
		want           string
	}{
		{"each is the base", "24", "1", "1", "0", "1", "24"},
		{"dozens", "24", "12", "1", "0", "1", "2"},
		{"pairs", "-3", "2", "1", "0", "1", "-1.5"},
		{"a ratio stored as a fraction", "453.592", "453592", "1000", "0", "1", "1"},
		{"an offset unit subtracts its offset first", "40", "1", "1", "32", "1", "8"},
		{"a missing ratio leaves the amount", "7", "", "", "", "", "7"},
	}
	for _, c := range cases {
		got := fromDimensionBase(d(c.base), c.rn, c.rd, c.on, c.od)
		assert.True(t, got.Equal(d(c.want)), "%s: got %s, want %s", c.name, got, c.want)
	}
}
