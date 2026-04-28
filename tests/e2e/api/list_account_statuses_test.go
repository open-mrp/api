//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const accountStatusesPath = "/v1/sales/account-statuses"

// --- List endpoint: happy path ---

func TestListAccountStatuses_ReturnsSeededData(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 4, "Should have at least 4 seeded account statuses")

	codes := map[string]bool{}
	for _, item := range list.Data {
		codes[DataItemField(item, "code")] = true
	}
	assert.True(t, codes["normal"], "Seeded account status 'normal' not found")
	assert.True(t, codes["preferred"], "Seeded account status 'preferred' not found")
	assert.True(t, codes["hold_shipment"], "Seeded account status 'hold_shipment' not found")
	assert.True(t, codes["hold_all"], "Seeded account status 'hold_all' not found")
}

func TestListAccountStatuses_ResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.NotNil(t, list.Data)

	for _, item := range list.Data {
		assert.Equal(t, "account_status", DataItemField(item, "object"))
	}
}

func TestListAccountStatuses_ItemFields(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.NotEmpty(t, jsonField(m, "id"), "id should not be empty")
		assert.Equal(t, "account_status", jsonField(m, "object"), "object should be 'account_status'")
		assert.NotEmpty(t, jsonField(m, "code"), "code should not be empty")
		assert.NotEmpty(t, jsonField(m, "name"), "name should not be empty")
		assert.NotEmpty(t, jsonField(m, "created_at"), "created_at should not be empty")
		assert.NotEmpty(t, jsonField(m, "updated_at"), "updated_at should not be empty")
	}
}

// --- List endpoint: pagination ---

func TestListAccountStatuses_LimitParam(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "limit=1 should return exactly 1 item")
	assert.True(t, list.PageInfo.HasNextPage, "Should have a next page with limit=1 and 4+ account statuses")
}

func TestListAccountStatuses_PaginationCursor(t *testing.T) {
	t.Parallel()

	// Page 1
	page1, _, err := apiClient.GetList(accountStatusesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.NotNil(t, page1.PageInfo.NextCursor, "Page 1 should have next_cursor")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	// Page 2
	page2, _, err := apiClient.GetList(accountStatusesPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEmpty(t, page2ID)
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should have a different item than page 1")
}

func TestListAccountStatuses_PrevCursor(t *testing.T) {
	t.Parallel()

	page1, _, err := apiClient.GetList(accountStatusesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	require.NotNil(t, page1.PageInfo.NextCursor)

	page1ID := DataItemField(page1.Data[0], "id")

	page2, _, err := apiClient.GetList(accountStatusesPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page1.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)
	assert.True(t, page2.PageInfo.HasPrevPage, "Page 2 should have has_prev_page=true")
	require.NotNil(t, page2.PageInfo.PrevCursor, "Page 2 should have prev_cursor")

	backToPage1, _, err := apiClient.GetList(accountStatusesPath, url.Values{
		"limit":  {"1"},
		"cursor": {*page2.PageInfo.PrevCursor},
	})
	require.NoError(t, err)
	require.Len(t, backToPage1.Data, 1)

	backID := DataItemField(backToPage1.Data[0], "id")
	assert.Equal(t, page1ID, backID, "Navigating back via prev_cursor should return the first page item")
}

// --- List endpoint: search ---

func TestListAccountStatuses_SearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, url.Values{"q": {"Hold"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'Hold' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "hold"),
			"Search result %q should contain 'hold'", name,
		)
	}
}

func TestListAccountStatuses_SearchCaseInsensitive(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, url.Values{"q": {"preferred"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Case-insensitive search for 'preferred' should return at least 1 result")

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), "preferred"),
			"Search result %q should contain 'preferred'", name,
		)
	}
}

func TestListAccountStatuses_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, url.Values{"q": {"zzzznotanaccountstatus99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Nonsense search should return empty data")
}

// --- Get single endpoint: happy path ---

func TestGetAccountStatus_ByID(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedAccountStatusID, jsonField(got, "id"))
	assert.Equal(t, "account_status", jsonField(got, "object"))
	assert.Equal(t, SeedAccountStatusCode, jsonField(got, "code"))
	assert.Equal(t, SeedAccountStatusName, jsonField(got, "name"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
	assert.NotEmpty(t, jsonField(got, "updated_at"))
}

func TestGetAccountStatus_ByCode(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusCode, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedAccountStatusID, jsonField(got, "id"), "lookup by code should resolve to the same resource as lookup by ID")
	assert.Equal(t, "account_status", jsonField(got, "object"))
	assert.Equal(t, SeedAccountStatusCode, jsonField(got, "code"))
	assert.Equal(t, SeedAccountStatusName, jsonField(got, "name"))
}

func TestGetAccountStatus_AllFields(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(accountStatusesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		id := DataItemField(item, "id")
		require.NotEmpty(t, id)

		status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+id, nil)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		require.NotNil(t, got)
		assert.NotEmpty(t, jsonField(got, "id"), "id must be populated")
		assert.Equal(t, "account_status", jsonField(got, "object"), "object must be 'account_status'")
		assert.NotEmpty(t, jsonField(got, "code"), "code must be populated")
		assert.NotEmpty(t, jsonField(got, "name"), "name must be populated")
		assert.NotEmpty(t, jsonField(got, "created_at"), "created_at must be populated")
		assert.NotEmpty(t, jsonField(got, "updated_at"), "updated_at must be populated")
	}
}

// --- Get single endpoint: error cases ---

func TestGetAccountStatus_NotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/acss_00000000000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)

	got := parseJSON(body)
	assert.NotEmpty(t, jsonField(got, "error"), "404 response should contain an error field")
}

func TestGetAccountStatus_InvalidID(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(accountStatusesPath+"/not-a-valid-id", nil)
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 404, "Invalid ID should return 400 or 404, got %d", status)
}

func TestGetAccountStatus_UnknownCode(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(accountStatusesPath+"/not_a_real_code", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status, "Unknown code should return 404")
}

// ──────────────────────────────────────────────
// AccountStatus — Owner Include Tests
// ──────────────────────────────────────────────

func TestListAccountStatuses_OwnerNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["owner"], "owner should be null without ?include=owner")
	}
}

func TestListAccountStatuses_IncludeOwner(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	first := parseJSON(list.Data[0])
	owner := jsonObject(first, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "owner", jsonField(owner, "object"))
	ownerType := jsonField(owner, "type")
	assert.Contains(t, []string{"system", "account"}, ownerType)
}
