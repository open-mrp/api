//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const requestLogsPath = "/v1/core/request-logs"

// discoverRequestLogID fetches the first request log ID from the list endpoint.
// Returns the ID and true on success, or empty string and false if unavailable.
func discoverRequestLogID(t *testing.T) (string, bool) {
	t.Helper()
	statusCode, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if statusCode == 401 || statusCode == 403 {
		return "", false
	}
	requireStatus(t, 200, statusCode, body)

	list := parseJSON(body)
	data, ok := list["data"].([]any)
	if !ok || len(data) == 0 {
		return "", false
	}
	first, ok := data[0].(map[string]any)
	if !ok {
		return "", false
	}
	id := jsonField(first, "id")
	if id == "" {
		return "", false
	}
	return id, true
}

// --- List ---

func TestRequestLogs_List(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(requestLogsPath, nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, statusCode)
	requireStatus(t, 200, statusCode, body)

	got := parseJSON(body)
	assert.Equal(t, "list", jsonField(got, "object"))
	assert.NotNil(t, got["data"])
}

func TestRequestLogs_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"limit": {"5"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	if len(list.Data) == 0 {
		t.Skip("No request logs available")
		return
	}

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.Equal(t, "request_log", jsonField(m, "object"))
		assert.NotEmpty(t, jsonField(m, "id"))
		assert.NotEmpty(t, jsonField(m, "method"))
		assert.NotEmpty(t, jsonField(m, "host"))
		assert.NotEmpty(t, jsonField(m, "path"))
		assert.NotEmpty(t, jsonField(m, "normalized_route"))
		assert.NotEmpty(t, jsonField(m, "status_code"))
		assert.NotEmpty(t, jsonField(m, "occurred_at"))
		assert.NotEmpty(t, jsonField(m, "created_at"))
	}
}

func TestRequestLogs_ListWithLimit(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, statusCode)
	requireStatus(t, 200, statusCode, body)

	got := parseJSON(body)
	data, ok := got["data"].([]any)
	if !ok {
		t.Skip("No data in response")
		return
	}
	assert.LessOrEqual(t, len(data), 1, "limit=1 should return at most 1 item")
}

func TestRequestLogs_ListPagination(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"limit": {"1"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	if len(list.Data) == 0 || !list.PageInfo.HasNextPage {
		t.Skip("Not enough request logs for pagination test")
		return
	}
	require.NotNil(t, list.PageInfo.NextCursor)

	page1ID := DataItemField(list.Data[0], "id")

	page2, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"limit":  {"1"},
		"cursor": {*list.PageInfo.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEqual(t, page1ID, page2ID, "Page 2 should return a different item")
}

func TestRequestLogs_ListFilterByMethod(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"method": {"GET"}, "limit": {"10"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "GET", jsonField(m, "method"), "All returned logs should have method=GET")
	}
}

func TestRequestLogs_ListFilterByStatusCode(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"status_code": {"200"}, "limit": {"10"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, "200", jsonField(m, "status_code"), "All returned logs should have status_code=200")
	}
}

func TestRequestLogs_ListFilterByActorType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"actor_type": {"api_key"}, "limit": {"10"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	// Our test harness authenticates with an API key, so we should find results.
	assert.GreaterOrEqual(t, len(list.Data), 1, "Should find at least 1 log with actor_type=api_key")
}

func TestRequestLogs_ListFilterNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"method": {"ZZZZ"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by impossible method should return no results")
}

// --- Get ---

func TestRequestLogs_GetByID(t *testing.T) {
	t.Parallel()
	id, ok := discoverRequestLogID(t)
	if !ok {
		t.Skip("Could not discover a request log ID")
		return
	}

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, "request_log", jsonField(got, "object"))
	assert.NotEmpty(t, jsonField(got, "method"))
	assert.NotEmpty(t, jsonField(got, "host"))
	assert.NotEmpty(t, jsonField(got, "path"))
	assert.NotEmpty(t, jsonField(got, "normalized_route"))
	assert.NotEmpty(t, jsonField(got, "status_code"))
	assert.NotEmpty(t, jsonField(got, "occurred_at"))
	assert.NotEmpty(t, jsonField(got, "created_at"))
}

func TestRequestLogs_GetNotFound(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(requestLogsPath+"/rq_000000000000", nil)
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	assert.Equal(t, 404, status)
}

// --- Expandable Fields ---

func TestRequestLogs_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id, ok := discoverRequestLogID(t)
	if !ok {
		t.Skip("Could not discover a request log ID")
		return
	}

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["account"], "account should be null without ?include=account")
	assert.Nil(t, got["actor"], "actor should be null without ?include=actor")
	assert.Nil(t, got["query_json"], "query_json should be null without ?include=query_json")
	assert.Nil(t, got["request_body_json"], "request_body_json should be null without ?include=request_body_json")
	assert.Nil(t, got["response_body_json"], "response_body_json should be null without ?include=response_body_json")
}

func TestRequestLogs_IncludeAccount(t *testing.T) {
	t.Parallel()
	id, ok := discoverRequestLogID(t)
	if !ok {
		t.Skip("Could not discover a request log ID")
		return
	}

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+id, url.Values{"include": {"account"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	account := jsonObject(got, "account")
	require.NotNil(t, account, "account should be present with ?include=account")
	assert.NotEmpty(t, jsonField(account, "id"))
	assert.Equal(t, "account", jsonField(account, "object"))
}

func TestRequestLogs_IncludeActor(t *testing.T) {
	t.Parallel()
	id, ok := discoverRequestLogID(t)
	if !ok {
		t.Skip("Could not discover a request log ID")
		return
	}

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+id, url.Values{"include": {"actor"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	actor := jsonObject(got, "actor")
	require.NotNil(t, actor, "actor should be present with ?include=actor")
	assert.NotEmpty(t, jsonField(actor, "id"))
	assert.Equal(t, "actor", jsonField(actor, "object"))
	assert.NotEmpty(t, jsonField(actor, "type"))
}

func TestRequestLogs_IncludeActorRole(t *testing.T) {
	t.Parallel()
	id, ok := discoverRequestLogID(t)
	if !ok {
		t.Skip("Could not discover a request log ID")
		return
	}

	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+id, url.Values{"include": {"actor.role"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	actor := jsonObject(got, "actor")
	require.NotNil(t, actor, "actor should be present with ?include=actor.role")
	role := jsonObject(actor, "role")
	require.NotNil(t, role, "actor.role should be present with ?include=actor.role")
	assert.NotEmpty(t, jsonField(role, "id"))
	assert.Equal(t, "role", jsonField(role, "object"))
}

func TestRequestLogs_ListIncludeAccount(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"include": {"account"}, "limit": {"5"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	if len(list.Data) == 0 {
		t.Skip("No request logs available")
		return
	}

	for _, item := range list.Data {
		m := parseJSON(item)
		account := jsonObject(m, "account")
		require.NotNil(t, account, "account should be present on list items with ?include=account")
		assert.NotEmpty(t, jsonField(account, "id"))
		assert.Equal(t, "account", jsonField(account, "object"))
	}
}

func TestRequestLogs_ListIncludeActor(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"include": {"actor"}, "limit": {"5"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	if len(list.Data) == 0 {
		t.Skip("No request logs available")
		return
	}

	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present on list items with ?include=actor")
		assert.NotEmpty(t, jsonField(actor, "id"))
		assert.Equal(t, "actor", jsonField(actor, "object"))
		assert.NotEmpty(t, jsonField(actor, "type"))
	}
}
