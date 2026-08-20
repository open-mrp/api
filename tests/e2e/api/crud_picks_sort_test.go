//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Issues an order promised the given number of days out and returns the pick it creates.
func promisedOrderPick(t *testing.T, customerID string, daysOut int) string {
	t.Helper()

	promised := time.Now().UTC().AddDate(0, 0, daysOut).Format("2006-01-02") + "T00:00:00Z"
	order := issueOrderForCustomer(t, customerID, map[string]any{"promised_at": promised})

	pickID := jsonField(jsonObject(jsonObject(order, "related"), "pick"), "id")
	if pickID == "" {
		status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+jsonField(order, "id"),
			url.Values{"include": {"related.pick"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		pickID = jsonField(jsonObject(jsonObject(parseJSON(body), "related"), "pick"), "id")
	}
	require.NotEmpty(t, pickID, "issuing an order creates its pick")
	return pickID
}

func listPickIDs(t *testing.T, params url.Values) []string {
	t.Helper()

	list, status, err := apiClient.GetList(picksPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status)

	ids := make([]string, len(list.Data))
	for i, row := range list.Data {
		ids[i] = DataItemField(row, "id")
	}
	return ids
}

func picksForCustomer(customerID, sort string) url.Values {
	params := url.Values{"customer_ids": {customerID}, "limit": {"10"}}
	if sort != "" {
		params.Set("sort", sort)
	}
	return params
}

// Pins the order picks come back in. The floor works the soonest commitment first, so
// ship-by date is the default and created_at is the opt-out.
func TestPicks_SortsBySoonestShipByDateByDefault(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-sort", ptrInt(30), "")

	// The near order is created first so the two sorts disagree; one order cannot satisfy both.
	nearPick := promisedOrderPick(t, customerID, 2)
	farPick := promisedOrderPick(t, customerID, 60)

	byShipBy := listPickIDs(t, picksForCustomer(customerID, "ship_by_date"))
	assert.Equal(t, []string{nearPick, farPick}, byShipBy, "soonest ship-by first")

	byCreated := listPickIDs(t, picksForCustomer(customerID, "created_at"))
	assert.Equal(t, []string{farPick, nearPick}, byCreated, "newest pick first")

	assert.Equal(t, byShipBy, listPickIDs(t, picksForCustomer(customerID, "")),
		"omitting sort matches sort=ship_by_date")
}

// The keyset has to follow the sort, or a second page repeats or skips rows.
func TestPicks_SortByShipByDatePagesInOrder(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-pick-sort-page", ptrInt(30), "")
	nearPick := promisedOrderPick(t, customerID, 3)
	farPick := promisedOrderPick(t, customerID, 90)

	params := picksForCustomer(customerID, "ship_by_date")
	params.Set("limit", "1")

	page1, status, err := apiClient.GetList(picksPath, params)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	requirePageLen(t, page1.Data, 1)
	assert.Equal(t, nearPick, DataItemField(page1.Data[0], "id"))
	require.NotNil(t, page1.PageInfo.NextPageURL, "two picks and a limit of one means a next page")

	page2, status, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	requirePageLen(t, page2.Data, 1)
	assert.Equal(t, farPick, DataItemField(page2.Data[0], "id"))

	back, status, err := apiClient.GetListFromPageURL(page2.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	requirePageLen(t, back.Data, 1)
	assert.Equal(t, nearPick, DataItemField(back.Data[0], "id"), "paging back lands on the first pick again")
}

func TestPicks_RejectsAnUnknownSort(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(picksPath, url.Values{"sort": {"number"}})
	require.NoError(t, err)
	require.Equal(t, 400, status, "sort only accepts the documented values: %s", string(body))
}
