//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pagination edge case tests extend the existing pagination_errors_test.go
// with additional boundary conditions: max limit, stale cursors, empty
// results with pagination, and multi-page traversal consistency.

// ──────────────────────────────────────────────
// Max limit
// ──────────────────────────────────────────────

func TestPaginationEdge_MaxLimitAccepted(t *testing.T) {
	t.Parallel()

	// Most APIs accept limit=100 or limit=200 as max.
	for _, maxLimit := range []string{"100", "200"} {
		maxLimit := maxLimit
		t.Run("limit="+maxLimit, func(t *testing.T) {
			t.Parallel()
			statusCode, body, err := apiClient.GetListRaw(customersPath, url.Values{"limit": {maxLimit}})
			require.NoError(t, err)
			// Should be 200 (accepted) or 400 (limit too large) — never 500.
			assert.NotEqual(t, 500, statusCode,
				"limit=%s should not cause 500: %s", maxLimit, string(body))
		})
	}
}

// ──────────────────────────────────────────────
// Stale / expired cursors
// ──────────────────────────────────────────────

func TestPaginationEdge_StaleCursorHandled(t *testing.T) {
	t.Parallel()

	// First get a valid cursor.
	page1Status, page1Body, err := apiClient.GetListRaw(customersPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 200, page1Status, page1Body)

	var page1 ListResponse
	require.NoError(t, json.Unmarshal(page1Body, &page1))

	if page1.PageInfo.NextCursor == nil {
		t.Skip("No next cursor available for stale cursor test")
	}

	// Manipulate the cursor to make it stale/invalid.
	staleCursor := *page1.PageInfo.NextCursor + "_stale"

	statusCode, body, err := apiClient.GetListRaw(customersPath, url.Values{
		"limit":  {"1"},
		"cursor": {staleCursor},
	})
	require.NoError(t, err)
	// Should be 400 (invalid cursor) or 200 (silently reset) — never 500.
	assert.NotEqual(t, 500, statusCode,
		"Stale cursor should not cause 500: %s", string(body))
}

// ──────────────────────────────────────────────
// Empty results with pagination params
// ──────────────────────────────────────────────

func TestPaginationEdge_EmptyResultsWithLimit(t *testing.T) {
	t.Parallel()

	// Search for something that doesn't exist, with pagination params.
	statusCode, body, err := apiClient.GetListRaw(customersPath, url.Values{
		"q":     {"zzzznonexistent_pagination_test99999"},
		"limit": {"10"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, statusCode, body)

	var list ListResponse
	require.NoError(t, json.Unmarshal(body, &list))
	assert.Equal(t, "list", list.Object)
	assert.Empty(t, list.Data, "Search for nonsense should return empty data")
	assert.False(t, list.PageInfo.HasNextPage, "Empty results should have has_next_page=false")
	assert.False(t, list.PageInfo.HasPrevPage, "Empty results should have has_prev_page=false")
}

// ──────────────────────────────────────────────
// Multi-page full traversal
// ──────────────────────────────────────────────

func TestPaginationEdge_FullTraversal(t *testing.T) {
	t.Parallel()

	var allIDs []string
	var cursor *string
	maxPages := 20 // safety limit

	for page := 0; page < maxPages; page++ {
		params := url.Values{"limit": {"5"}}
		if cursor != nil {
			params.Set("cursor", *cursor)
		}

		list, _, err := apiClient.GetList(customersPath, params)
		require.NoError(t, err)

		for _, item := range list.Data {
			id := DataItemField(item, "id")
			allIDs = append(allIDs, id)
		}

		if !list.PageInfo.HasNextPage {
			break
		}
		cursor = list.PageInfo.NextCursor
		require.NotNil(t, cursor, "next_cursor should be set when has_next_page is true (page %d)", page)
	}

	// Verify no duplicate IDs across pages.
	seen := make(map[string]bool)
	for _, id := range allIDs {
		assert.False(t, seen[id], "ID %q appeared more than once across pages — pagination returned duplicate", id)
		seen[id] = true
	}

	assert.GreaterOrEqual(t, len(allIDs), 1, "Full traversal should find at least 1 item")
}

// ──────────────────────────────────────────────
// Non-numeric limit values
// ──────────────────────────────────────────────

func TestPaginationEdge_NonNumericLimit(t *testing.T) {
	t.Parallel()

	statusCode, body, err := apiClient.GetListRaw(customersPath, url.Values{"limit": {"abc"}})
	require.NoError(t, err)
	assert.True(t, statusCode == 400 || statusCode == 200,
		"Non-numeric limit should return 400 or be ignored (200), got %d: %s", statusCode, string(body))
	assert.NotEqual(t, 500, statusCode,
		"Non-numeric limit should not cause 500: %s", string(body))
}

func TestPaginationEdge_FloatLimit(t *testing.T) {
	t.Parallel()

	statusCode, body, err := apiClient.GetListRaw(customersPath, url.Values{"limit": {"1.5"}})
	require.NoError(t, err)
	assert.NotEqual(t, 500, statusCode,
		"Float limit should not cause 500: %s", string(body))
}

// ──────────────────────────────────────────────
// Prev cursor navigation
// ──────────────────────────────────────────────

func TestPaginationEdge_PrevCursorNavigation(t *testing.T) {
	t.Parallel()

	// Create our own customers so data is stable across requests
	// (parallel tests create/delete customers that can disappear mid-pagination).
	searchTag := "prevnav-" + uniqueName("pag")
	for i := 0; i < 3; i++ {
		createAndCleanup(t, customersPath, validCustomerBody(fmt.Sprintf("%s-%d", searchTag, i)))
	}

	params := url.Values{
		"limit": {"1"},
		"q":     {searchTag},
	}

	// Get page 1.
	page1, _, err := apiClient.GetList(customersPath, params)
	require.NoError(t, err)
	require.True(t, page1.PageInfo.HasNextPage, "Need at least 2 items for prev cursor test")

	// Get page 2.
	params.Set("cursor", *page1.PageInfo.NextCursor)
	page2, _, err := apiClient.GetList(customersPath, params)
	require.NoError(t, err)
	require.NotNil(t, page2.PageInfo.PrevCursor, "Page 2 should have prev_cursor")

	// Navigate back using prev cursor — should return data without errors.
	params.Set("cursor", *page2.PageInfo.PrevCursor)
	backToPage1, _, err := apiClient.GetList(customersPath, params)
	require.NoError(t, err)
	assert.NotEmpty(t, backToPage1.Data, "Navigating back with prev_cursor should return data")
}
