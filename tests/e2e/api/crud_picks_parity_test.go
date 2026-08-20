//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Response-parity gate for the picking pages: one assertion per field the dashboard renders,
// so a projection, presenter or include-registry change that drops one fails here rather than
// as a blank cell.

// The include set the pick detail page requests in one call, tested together because the
// resolver recurses through them rather than resolving each in isolation.
var pickDetailPageIncludes = []string{
	"related.sales_order",
	"lines",
	"lines.sales_order_line",
	"lines.quantity",
	"lines.quantity.unit",
}

func pickPageParams(includes []string, extra url.Values) url.Values {
	params := url.Values{}
	for k, vs := range extra {
		for _, v := range vs {
			params.Add(k, v)
		}
	}
	for _, inc := range includes {
		params.Add("include", inc)
	}
	return params
}

func TestPicksParity_DetailResolvesEveryIncludeThePageRequests(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath+"/"+SeedPickID, pickPageParams(pickDetailPageIncludes, nil))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	pick := parseJSON(body)

	require.NotEmpty(t, jsonField(jsonObject(jsonObject(pick, "related"), "sales_order"), "id"),
		"the page reads related.sales_order.id to fetch the order for its header")

	lines := jsonObject(pick, "lines")
	require.NotNil(t, lines, "lines must expand")
	rows, ok := lines["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, rows, "the seeded pick must have lines for this to prove anything")

	for _, r := range rows {
		line := r.(map[string]any)

		require.NotEmpty(t, jsonField(jsonObject(line, "sales_order_line"), "id"),
			"each line is joined to its order line by id")

		quantity := jsonObject(line, "quantity")
		require.NotNil(t, quantity, "the quantity input is bound to lines.quantity")
		assert.NotEmpty(t, jsonField(quantity, "value"))
		require.NotNil(t, quantity["unit"],
			"lines.quantity.unit is two levels down; without it the input has no unit to render")
		assert.NotEmpty(t, jsonField(jsonObject(quantity, "unit"), "abbreviation"))

		ordered := jsonObject(line, "ordered_quantity")
		require.NotNil(t, ordered, "ordered_quantity is a base scalar, always present")
		assert.NotEmpty(t, jsonField(ordered, "value"))
	}
}

func TestPicksParity_ListRowCarriesEveryColumnTheTableRenders(t *testing.T) {
	t.Parallel()

	params := pickPageParams([]string{"customer"}, url.Values{"limit": {"25"}})
	status, body, err := apiClient.GetListRaw(picksPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	rows, ok := parseJSON(body)["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, rows)

	var sawShipTo bool
	for _, r := range rows {
		row := r.(map[string]any)

		// Base scalars the row renders without expanding anything.
		assert.Contains(t, []any{"low", "normal", "high"}, row["priority"], "the priority pill reads a code")
		assert.NotNil(t, row["line_count"], "the row shows a line count rather than expanding lines")
		assert.NotNil(t, row["created_at"], "the Date column")

		totals := jsonObject(row, "totals")
		require.NotNil(t, totals, "the progress bars read totals")
		require.NotNil(t, jsonObject(totals, "picked")["completion"], "picked completion drives a bar")
		require.NotNil(t, jsonObject(totals, "packed")["completion"], "packed completion drives a bar")

		require.NotNil(t, row["customer"], "the customer cell needs the customer include")
		assert.NotEmpty(t, jsonField(jsonObject(row, "customer"), "name"))

		if row["ship_to"] != nil {
			sawShipTo = true
			assert.NotEmpty(t, jsonField(jsonObject(row, "ship_to"), "id"))
		}
	}
	assert.True(t, sawShipTo, "the ship-to cell needs at least one seeded pick whose order has an address")
}

// Progress is a second query keyed by the page's ids, so it has to run on every page rather
// than only the first — the bars are blank otherwise.
func TestPicksParity_ProgressSurvivesPagination(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-progress", ptrInt(30), "")
	first := promisedOrderPick(t, customerID, 5)
	second := promisedOrderPick(t, customerID, 10)

	for _, pickID := range []string{first, second} {
		status, body, err := apiClient.Put(picksPath+"/"+pickID+"/actions/pick", nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	}

	params := pickPageParams(nil, url.Values{
		"customer_ids": {customerID},
		"sort":         {"ship_by_date"},
		"limit":        {"1"},
	})
	page1, status, err := apiClient.GetList(picksPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	requirePageLen(t, page1.Data, 1)
	assert.InDelta(t, 1.0, pickedCompletion(t, page1.Data[0]), 0.001, "page 1 reports the progress it has")
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, status, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	requirePageLen(t, page2.Data, 1)
	assert.InDelta(t, 1.0, pickedCompletion(t, page2.Data[0]), 0.001,
		"a fully picked pick reports completion 1 on page 2 as well as page 1")

	back, status, err := apiClient.GetListFromPageURL(page2.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	requirePageLen(t, back.Data, 1)
	assert.InDelta(t, 1.0, pickedCompletion(t, back.Data[0]), 0.001, "and on the way back")
}

func pickedCompletion(t *testing.T, row []byte) float64 {
	t.Helper()

	totals := jsonObject(parseJSON(row), "totals")
	require.NotNil(t, totals, "every list row carries totals")
	completion, ok := jsonObject(totals, "picked")["completion"].(float64)
	require.True(t, ok, "picked completion must be a number")
	return completion
}
