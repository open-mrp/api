//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sorting and filtering validation tests verify that list endpoint filter
// parameters produce correctly filtered results, and that default ordering
// is consistent across pagination.

// ──────────────────────────────────────────────
// Default ordering consistency
// ──────────────────────────────────────────────

func TestSortingFiltering_DefaultOrderConsistent(t *testing.T) {
	t.Parallel()

	params := url.Values{
		"limit": {"10"},
	}

	list, _, err := apiClient.GetList(customersPath, params)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 2, "Need at least 2 items to verify ordering")

	// Default ordering is created_at DESC, id DESC. Verify each item's
	// created_at is >= the next item's (descending), confirming the
	// server returns a deterministic sort.
	for i := 0; i < len(list.Data)-1; i++ {
		curStr := DataItemField(list.Data[i], "created_at")
		nextStr := DataItemField(list.Data[i+1], "created_at")
		curTime, err := time.Parse(time.RFC3339Nano, curStr)
		require.NoError(t, err, "parsing created_at for item[%d]", i)
		nextTime, err := time.Parse(time.RFC3339Nano, nextStr)
		require.NoError(t, err, "parsing created_at for item[%d]", i+1)
		assert.False(t, curTime.Before(nextTime),
			"Default ordering should be created_at DESC: item[%d] (%s) should be >= item[%d] (%s)",
			i, curStr, i+1, nextStr)
	}
}

// ──────────────────────────────────────────────
// Filtering
// ──────────────────────────────────────────────

func TestSortingFiltering_FilterByCustomerTypeGroup(t *testing.T) {
	t.Parallel()

	// Create a customer with a specific type group.
	created := createAndCleanup(t, customersPath, validCustomerBody(uniqueName("e2e-filt-type")))
	id := jsonField(created, "id")

	// Filter by that type group.
	list, _, err := apiClient.GetList(customersPath, url.Values{
		"customer_group_ids": {SeedCustomerGroupID},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Filter should return at least 1 result")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "Created customer should appear in filtered results")
}

func TestSortingFiltering_SearchMatchesExpected(t *testing.T) {
	t.Parallel()

	// Create a customer with a distinctive name.
	distinctName := uniqueName("e2e-filt-search")
	created := createAndCleanup(t, customersPath, validCustomerBody(distinctName))
	id := jsonField(created, "id")

	// Search for the distinctive name.
	list, _, err := apiClient.GetList(customersPath, url.Values{"q": {distinctName}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "Search should return at least 1 result")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "Created customer should appear in search results for its name")
}

func TestSortingFiltering_FilteredPaginationConsistency(t *testing.T) {
	t.Parallel()

	// Fetch filtered results with limit=1, then paginate.
	page1Status, page1Body, err := apiClient.GetListRaw(customersPath, url.Values{
		"limit": {"1"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, page1Status, page1Body)

	var page1 ListResponse
	require.NoError(t, json.Unmarshal(page1Body, &page1))

	if !page1.PageInfo.HasNextPage || page1.PageInfo.NextCursor == nil {
		t.Skip("Not enough data to test filtered pagination")
	}

	// Get page 2.
	page2Status, page2Body, err := apiClient.GetListRaw(customersPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	requireStatus(t, 200, page2Status, page2Body)

	var page2 ListResponse
	require.NoError(t, json.Unmarshal(page2Body, &page2))

	// Pages should return different items.
	if len(page1.Data) > 0 && len(page2.Data) > 0 {
		id1 := DataItemField(page1.Data[0], "id")
		id2 := DataItemField(page2.Data[0], "id")
		assert.NotEqual(t, id1, id2, "Paginated pages should return different items")
	}
}

// ──────────────────────────────────────────────
// Invalid filter params
// ──────────────────────────────────────────────

func TestSortingFiltering_InvalidFilterParam(t *testing.T) {
	t.Parallel()

	// Unknown query param — should be ignored or return 400, never 500.
	statusCode, body, err := apiClient.GetListRaw(customersPath, url.Values{
		"nonexistent_filter": {"value"},
	})
	require.NoError(t, err)
	assert.NotEqual(t, 500, statusCode,
		"Unknown filter param should not cause 500: %s", string(body))
}

// ──────────────────────────────────────────────
// API key list filtering
// ──────────────────────────────────────────────

func TestSortingFiltering_APIKeyStatusFilter(t *testing.T) {
	t.Parallel()

	// Create and immediately revoke a key.
	created := createAPIKeyAndCleanup(t, uniqueName("e2e-filt-revkey"))
	revokedID := jsonField(jsonObject(created, "api_key_info"), "id")
	apiClient.Delete(apiKeysPath + "/" + revokedID)

	// Active filter should not include revoked key.
	activeList, _, err := apiClient.GetList(apiKeysPath, url.Values{"statuses": {"active"}})
	require.NoError(t, err)
	for _, item := range activeList.Data {
		assert.NotEqual(t, revokedID, DataItemField(item, "id"),
			"Revoked key should not appear in active-filtered list")
	}

	// Revoked filter should include it.
	revokedList, _, err := apiClient.GetList(apiKeysPath, url.Values{"statuses": {"revoked"}})
	require.NoError(t, err)
	found := false
	for _, item := range revokedList.Data {
		if DataItemField(item, "id") == revokedID {
			found = true
			break
		}
	}
	assert.True(t, found, "Revoked key should appear in revoked-filtered list")
}

// ──────────────────────────────────────────────
// Customer status filter
// ──────────────────────────────────────────────

func TestSortingFiltering_CustomerStatusFilter(t *testing.T) {
	t.Parallel()

	// Create a customer with hold_shipment status.
	statusPayload := validCustomerBody(uniqueName("e2e-filt-status"))
	statusPayload["status"] = "hold_shipment"
	created := createAndCleanup(t, customersPath, statusPayload)
	id := jsonField(created, "id")

	// Filter by hold_shipment — should include it.
	list, _, err := apiClient.GetList(customersPath, url.Values{"status_codes": {"hold_shipment"}})
	require.NoError(t, err)

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "Customer with hold_shipment status should appear when filtering by that status")

	// Filter by normal — should NOT include it.
	normalList, _, err := apiClient.GetList(customersPath, url.Values{"status_codes": {"normal"}})
	require.NoError(t, err)
	for _, item := range normalList.Data {
		assert.NotEqual(t, id, DataItemField(item, "id"),
			"Customer with hold_shipment status should not appear when filtering by normal")
	}
}
