//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
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
		}
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), strings.ToLower(distinctName)),
			"Search result %q should contain search term %q", name, distinctName,
		)
	}
	assert.True(t, found, "Created customer should appear in search results for its name")
}

func TestSortingFiltering_FilteredPaginationConsistency(t *testing.T) {
	t.Parallel()

	// Paginate over customers this test owns, scoped by a unique search term,
	// so parallel tests creating or deleting customers cannot shift the
	// cursor window.
	prefix := uniqueName("e2e-filt-pg")
	var ids []string
	for i := 0; i < 2; i++ {
		status, body, err := apiClient.Post(customersPath,
			validCustomerBody(fmt.Sprintf("%s-%d", prefix, i)), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		id := jsonField(parseJSON(body), "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(customersPath + "/" + id)
		ids = append(ids, id)
	}

	assertScopedCursorPagination(t, customersPath, url.Values{"q": {prefix}}, ids)
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

	// Active filter should not include the revoked key. Active keys are either
	// un-revoked or scheduled for a future revocation (rotate's revoke_at), so
	// revoked_at must be empty or in the future. Allow a little slack for
	// clock skew between the harness and the database.
	activeList, _, err := apiClient.GetList(apiKeysPath, url.Values{"statuses": {"active"}})
	require.NoError(t, err)
	for _, item := range activeList.Data {
		m := parseJSON(item)
		assert.NotEqual(t, revokedID, DataItemField(item, "id"),
			"Revoked key should not appear in active-filtered list")
		if raw := jsonField(m, "revoked_at"); raw != "" {
			revokedAt, err := time.Parse(time.RFC3339, raw)
			require.NoError(t, err, "Active-filtered key %q has unparseable revoked_at %q", DataItemField(item, "id"), raw)
			assert.True(t, revokedAt.After(time.Now().UTC().Add(-5*time.Second)),
				"Active-filtered key %q should only have a future (scheduled) revoked_at, got %s", DataItemField(item, "id"), raw)
		}
	}

	// Revoked filter should include the key, and all returned keys must be revoked (revoked_at is non-null).
	revokedList, _, err := apiClient.GetList(apiKeysPath, url.Values{"statuses": {"revoked"}})
	require.NoError(t, err)
	found := false
	for _, item := range revokedList.Data {
		m := parseJSON(item)
		if DataItemField(item, "id") == revokedID {
			found = true
		}
		assert.NotEmpty(t, jsonField(m, "revoked_at"),
			"Revoked-filtered key %q should have a non-null revoked_at", DataItemField(item, "id"))
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

	// Filter by hold_shipment — should include the created customer, and all results must share that status.
	list, _, err := apiClient.GetList(customersPath, url.Values{"status_codes": {"hold_shipment"}})
	require.NoError(t, err)

	found := false
	for _, item := range list.Data {
		m := parseJSON(item)
		if DataItemField(item, "id") == id {
			found = true
		}
		assert.Equal(t, "hold_shipment", jsonField(m, "status"),
			"All hold_shipment-filtered results should have status=hold_shipment")
	}
	assert.True(t, found, "Customer with hold_shipment status should appear when filtering by that status")

	// Filter by normal — should NOT include the hold_shipment customer, and all results must be normal.
	normalList, _, err := apiClient.GetList(customersPath, url.Values{"status_codes": {"normal"}})
	require.NoError(t, err)
	for _, item := range normalList.Data {
		m := parseJSON(item)
		assert.NotEqual(t, id, DataItemField(item, "id"),
			"Customer with hold_shipment status should not appear when filtering by normal")
		assert.Equal(t, "normal", jsonField(m, "status"),
			"All normal-filtered results should have status=normal")
	}
}
