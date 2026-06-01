//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const prioritiesPath = "/v1/sales/priorities"

// --- List endpoint: happy path ---

func TestListPriorities_ReturnsSeededData(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 3, "Should have at least 3 seeded priorities (low, normal, high)")

	codes := map[string]bool{}
	for _, item := range list.Data {
		codes[DataItemField(item, "code")] = true
	}
	assert.True(t, codes["low"], "Seeded priority 'low' not found")
	assert.True(t, codes["normal"], "Seeded priority 'normal' not found")
	assert.True(t, codes["high"], "Seeded priority 'high' not found")
}

func TestListPriorities_ResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.NotNil(t, list.Data)

	for _, item := range list.Data {
		assert.Equal(t, "priority", DataItemField(item, "object"))
	}
}

func TestListPriorities_ItemFields(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.NotEmpty(t, jsonField(m, "id"), "id should not be empty")
		assert.Equal(t, "priority", jsonField(m, "object"), "object should be 'priority'")
		assert.NotEmpty(t, jsonField(m, "code"), "code should not be empty")
		assert.NotEmpty(t, jsonField(m, "name"), "name should not be empty")
		assert.NotEmpty(t, jsonField(m, "created_at"), "created_at should not be empty")
		assert.NotEmpty(t, jsonField(m, "updated_at"), "updated_at should not be empty")
	}
}

// --- List endpoint: pagination ---

func TestListPriorities_LimitParam(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "limit=1 should return exactly 1 item")
	assert.True(t, list.PageInfo.HasNextPage, "Should have a next page with limit=1 and 3+ priorities")
}

func TestListPriorities_PaginationCursor(t *testing.T) {
	t.Parallel()

	// Page 1
	page1, _, err := apiClient.GetList(prioritiesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, page1.Data, 1)
	require.NotNil(t, page1.PageInfo.NextPageURL, "Page 1 should have next_page_url")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	// Page 2
	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	requirePageLen(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEmpty(t, page2ID)
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should have a different item than page 1")
}

func TestListPriorities_PrevPageURL(t *testing.T) {
	t.Parallel()

	// Get page 1 with limit=1
	page1, _, err := apiClient.GetList(prioritiesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, page1.Data, 1)
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page1ID := DataItemField(page1.Data[0], "id")

	// Get page 2
	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	requirePageLen(t, page2.Data, 1)
	assert.True(t, page2.PageInfo.HasPrevPage, "Page 2 should have has_prev_page=true")
	require.NotNil(t, page2.PageInfo.PreviousPageURL, "Page 2 should have previous_page_url")

	// Navigate back to page 1
	backToPage1, _, err := apiClient.GetListFromPageURL(page2.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Len(t, backToPage1.Data, 1)

	backID := DataItemField(backToPage1.Data[0], "id")
	assert.Equal(t, page1ID, backID, "Navigating back via previous_page_url should return the first page item")
}

// --- List endpoint: search ---

func TestListPriorities_SearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, url.Values{"q": {"High"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'High' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "high"),
			"Search result %q should contain 'high'", name,
		)
	}
}

func TestListPriorities_SearchCaseInsensitive(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, url.Values{"q": {"high"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Case-insensitive search for 'high' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "high"),
			"Search result %q should contain 'high'", name,
		)
	}
}

func TestListPriorities_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(prioritiesPath, url.Values{"q": {"zzzznotapriority99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense search should return empty data")
}

// --- Get single endpoint: happy path ---

func TestGetPriority_ByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+SeedPriorityID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedPriorityID, jsonField(got, "id"))
	assert.Equal(t, "priority", jsonField(got, "object"))
	assert.Equal(t, SeedPriorityCode, jsonField(got, "code"))
	assert.Equal(t, SeedPriorityName, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestGetPriority_AllFields(t *testing.T) {
	t.Parallel()

	// Fetch all priorities and verify each one has all fields populated
	list, _, err := apiClient.GetList(prioritiesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		id := DataItemField(item, "id")
		require.NotEmpty(t, id)

		status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+id, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		require.NotNil(t, got)
		assert.NotEmpty(t, jsonField(got, "id"), "id must be populated")
		assert.Equal(t, "priority", jsonField(got, "object"), "object must be 'priority'")
		assert.NotEmpty(t, jsonField(got, "code"), "code must be populated")
		assert.NotEmpty(t, jsonField(got, "name"), "name must be populated")
		assert.NotEmpty(t, jsonField(got, "created_at"), "created_at must be populated")
		assert.NotEmpty(t, jsonField(got, "updated_at"), "updated_at must be populated")
	}
}

// --- Get single endpoint: error cases ---

func TestGetPriority_NotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath+"/pi_00000000000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	got := parseJSON(body)
	assert.NotEmpty(t, jsonField(got, "error"), "404 response should contain an error field")
}

func TestGetPriority_InvalidID(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(prioritiesPath+"/not-a-valid-id", nil)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 404, "Invalid ID should return 400 or 404, got %d", status)
}
