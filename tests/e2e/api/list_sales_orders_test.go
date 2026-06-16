//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// SalesOrder — list endpoint filter coverage
// ──────────────────────────────────────────────
//
// These tests fill the gaps the generic list harnesses leave for
// GET /v1/sales/sales-orders. Generic shape/limit/nonsense-search/cursor
// pagination live in list_test.go, and array-filter union/exclusion for
// status_codes/customer_ids/sales_rep_ids/item_ids/product_line_ids lives in
// array_filter_union_test.go — they are intentionally not duplicated here.
//
// Each test exercises a distinct code path of the list query: exact-match
// search (the new behavior — exact on number/customer_po_number, no substring,
// no customer name), the status_codes "all" wildcard, customer_group_ids, the
// start_date/end_date range, exclude_internal_orders, the batched line_count
// path, and several filters combined.
//
// Seed data lives in shared/db/seed/0016_e2e_filter_coverage.sql; the
// SeedInternalSalesOrderID / SeedPOSalesOrderID orders use a far-future
// created_at so they stay on the first list page under parallel churn.

// futureDate returns a YYYY-MM-DD date five years out — after every "now"-based
// order other tests create, but before the nine-year-out filter-coverage seed
// orders, so it isolates those seed rows deterministically.
func futureDate() string {
	return time.Now().AddDate(5, 0, 0).Format("2006-01-02")
}

// --- Exact-match search (new behavior) ---

func TestListSalesOrders_SearchByNumberExact(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(salesOrdersPath, url.Values{"q": {"ORD-001"}})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "exact search for ORD-001 should return the seed order")
	for _, item := range list.Data {
		assert.Equal(t, "ORD-001", DataItemField(item, "number"),
			"exact-match search must only return rows whose number equals the query")
	}
	assertListContainsID(t, salesOrdersPath, url.Values{"q": {"ORD-001"}}, SeedSalesOrderID)
}

func TestListSalesOrders_SearchSubstringNoMatch(t *testing.T) {
	t.Parallel()
	// "ORD" is a substring of ORD-001/ORD-002/... but search is now exact, so
	// no order is numbered literally "ORD" and the result must be empty.
	list, _, err := apiClient.GetList(salesOrdersPath, url.Values{"q": {"ORD"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "substring search must not match (search is exact on number/customer_po_number)")
}

func TestListSalesOrders_SearchByCustomerNameNoMatch(t *testing.T) {
	t.Parallel()
	// Customer-name search was removed (it forced full table scans); use the
	// customer_ids filter instead. Searching the name must return nothing.
	list, _, err := apiClient.GetList(salesOrdersPath, url.Values{"q": {SeedCustomerName}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "customer-name search must not match anymore")
}

func TestListSalesOrders_SearchByCustomerPOExact(t *testing.T) {
	t.Parallel()
	assertListContainsID(t, salesOrdersPath, url.Values{"q": {SeedSalesOrderPONumber}}, SeedPOSalesOrderID)
}

// --- status_codes "all" wildcard ---

func TestListSalesOrders_StatusCodesAllWildcard(t *testing.T) {
	t.Parallel()
	// "all" is a wildcard meaning "no status filter": with it set, orders of
	// different statuses all appear. Scope to the seed customer (orthogonal to
	// status) so the lookup stays reliable as the global order set grows; both
	// the issued order and the estimate order belong to that customer.
	params := url.Values{"status_codes": {"all"}, "customer_ids": {SeedCustomerAccountID}}
	assertListContainsID(t, salesOrdersPath, params, SeedSalesOrderID)                   // ORD-001, issued
	assertListContainsID(t, salesOrdersPath, params, SeedIncludePutEstimateSalesOrderID) // EST-001, estimate
}

// --- customer_group_ids ---

func TestListSalesOrders_FilterByCustomerGroup(t *testing.T) {
	t.Parallel()
	// The seed customer belongs to the DME group, so its orders match.
	assertListContainsID(t, salesOrdersPath, url.Values{"customer_group_ids": {SeedCustomerGroupID}}, SeedSalesOrderID)
}

func TestListSalesOrders_FilterByCustomerGroupNoMatch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(salesOrdersPath, url.Values{"customer_group_ids": {"acgp_00000000000000000000"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "filtering by a nonexistent customer group should return no orders")
}

// --- Date range ---

func TestListSalesOrders_StartDateExcludesOlderOrders(t *testing.T) {
	t.Parallel()
	// start_date is inclusive on created_at. With a far-future start_date only
	// the nine-year-out seed orders qualify; the "now"-created seed order must
	// be excluded.
	params := url.Values{"start_date": {futureDate()}}
	assertListContainsID(t, salesOrdersPath, params, SeedPOSalesOrderID)
	assert.Nil(t, listFindByField(t, salesOrdersPath, params, "id", SeedSalesOrderID),
		"a recent order must be excluded by a far-future start_date")
}

func TestListSalesOrders_EndDateExcludesEverything(t *testing.T) {
	t.Parallel()
	// end_date is inclusive on created_at; nothing predates 2000.
	list, _, err := apiClient.GetList(salesOrdersPath, url.Values{"end_date": {"2000-01-01"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "no orders exist before 2000-01-01")
}

// --- exclude_internal_orders ---

func TestListSalesOrders_ExcludeInternalOrders(t *testing.T) {
	t.Parallel()
	// Scope to the far-future seed orders so the assertion is immune to the
	// "now"-ish orders parallel tests churn. The internal order (buyer == owner)
	// is present by default and dropped when exclude_internal_orders=true, while
	// the external PO order survives either way.
	scoped := url.Values{"start_date": {futureDate()}}

	assertListContainsID(t, salesOrdersPath, scoped, SeedInternalSalesOrderID)

	excluded := url.Values{"start_date": {futureDate()}, "exclude_internal_orders": {"true"}}
	assert.Nil(t, listFindByField(t, salesOrdersPath, excluded, "id", SeedInternalSalesOrderID),
		"internal order (buyer == owner) must be excluded when exclude_internal_orders=true")
	assertListContainsID(t, salesOrdersPath, excluded, SeedPOSalesOrderID)
}

// --- line_count (batched attachLineCounts path) ---

func TestListSalesOrders_LineCountMatchesIncludedLines(t *testing.T) {
	t.Parallel()
	// line_count is populated by the batched GetSalesOrderLineCounts query, not a
	// per-row subquery; it must equal the number of rows in ?include=lines.
	row := salesOrderListRow(t, url.Values{"include": {"lines"}})

	lineCount, ok := row["line_count"].(float64)
	require.True(t, ok, "line_count should be a number on a list row")
	require.Greater(t, lineCount, float64(0), "seed sales order has lines")

	lines := jsonObject(row, "lines")
	require.NotNil(t, lines, "lines should be populated with ?include=lines")
	data := jsonArray(lines, "data")
	assert.Equal(t, int(lineCount), len(data),
		"line_count must equal the number of included lines")
}

// --- Combined filters ---

func TestListSalesOrders_CombinedFilters(t *testing.T) {
	t.Parallel()
	// status_codes + customer_ids + start_date applied together: only the
	// far-future issued order for the seed customer qualifies. The internal
	// order (different buyer) must not appear.
	params := url.Values{
		"status_codes": {"issued"},
		"customer_ids": {SeedCustomerAccountID},
		"start_date":   {futureDate()},
		"include":      {"customer"},
	}

	list, _, err := apiClient.GetList(salesOrdersPath, params)
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "combined filter should match the far-future PO order")
	for _, item := range list.Data {
		assert.Equal(t, "issued", DataItemField(item, "status"), "every row must satisfy status_codes")
		m := parseJSON(item)
		cust := jsonObject(m, "customer")
		require.NotNil(t, cust, "customer should be expanded with ?include=customer")
		assert.Equal(t, SeedCustomerAccountID, jsonField(cust, "id"), "every row must satisfy customer_ids")
	}

	assertListContainsID(t, salesOrdersPath, params, SeedPOSalesOrderID)
	assert.Nil(t, listFindByField(t, salesOrdersPath, params, "id", SeedInternalSalesOrderID),
		"internal order has a different buyer and must not match customer_ids")
}
