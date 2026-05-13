//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
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
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"methods": {"GET"}, "limit": {"10"}})
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
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"status_codes": {"200"}, "limit": {"10"}})
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
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"actor_types": {"api_key"},
		"include":     {"actor"},
		"limit":       {"10"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	// Our test harness authenticates with an API key, so we should find results.
	require.GreaterOrEqual(t, len(list.Data), 1, "Should find at least 1 log with actor_type=api_key")
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		assert.Equal(t, "api_key", jsonField(actor, "type"), "All returned logs should have actor.type=api_key")
	}
}

func TestRequestLogs_ListFilterByActorTypeImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"actor_types": {"zzz_no_such_type"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by impossible actor_type should return no results")
}

func TestRequestLogs_ListFilterByActorIDSingle(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"include": {"actor"},
		"limit":   {"10"},
	})
	if err != nil || len(list.Data) == 0 {
		t.Skip("No request logs available to discover actor IDs")
		return
	}
	var actorID string
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		if actor == nil {
			continue
		}
		actorID = jsonField(actor, "id")
		if actorID != "" {
			break
		}
	}
	if actorID == "" {
		t.Skip("No request logs with an actor ID available")
		return
	}

	filtered, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"actor_ids": {actorID},
		"include":   {"actor"},
		"limit":     {"10"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known actor_id should return at least one log")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		assert.Equal(t, actorID, jsonField(actor, "id"), "All results should match the filtered actor_id")
	}
}

func TestRequestLogs_ListFilterByActorIDsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"actor_ids": {"actu_zzzzzzzzzzzzzzzz", "ak_zzzzzzzzzzzzzzzz"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by impossible actor_ids should return no results")
}

func TestRequestLogs_ListFilterByIdempotencyKey(t *testing.T) {
	t.Parallel()
	filtered, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"idempotency_key": {SeedRequestLogIdempotencyKey},
		"limit":           {"10"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	require.NotEmpty(t, filtered.Data, "Filtering by seeded idempotency key should return at least one log")
	for _, item := range filtered.Data {
		m := parseJSON(item)
		assert.Equal(t, SeedRequestLogIdempotencyKey, jsonField(m, "idempotency_key"), "All results should match the filtered idempotency key")
	}
}

func TestRequestLogs_ListFilterByIdempotencyKeyImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"idempotency_key": {"zzzz-idempotency-key-does-not-exist-0001"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by non-existent idempotency key should return no results")
}

func TestRequestLogs_ListFilterNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"methods": {"ZZZZ"}})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by impossible method should return no results")
}

// --- Multi-value filters ---

// discoverDistinctValues returns up to `n` distinct values of the given JSON
// field across recent request logs. Used to seed multi-value filter tests with
// real data the harness has produced.
func discoverDistinctValues(t *testing.T, field string, n int) []string {
	t.Helper()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"limit": {"50"}})
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, n)
	for _, item := range list.Data {
		m := parseJSON(item)
		v := jsonField(m, field)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
		if len(out) >= n {
			break
		}
	}
	return out
}

func TestRequestLogs_ListFilterByMultipleMethods(t *testing.T) {
	t.Parallel()
	methods := discoverDistinctValues(t, "method", 2)
	if len(methods) < 2 {
		t.Skip("Need at least 2 distinct methods in recent request logs")
		return
	}

	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"methods": methods,
		"limit":   {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by %v should return results", methods)

	allowed := map[string]bool{methods[0]: true, methods[1]: true}
	for _, item := range list.Data {
		m := parseJSON(item)
		got := jsonField(m, "method")
		assert.True(t, allowed[got], "method %q not in requested set %v", got, methods)
	}
}

func TestRequestLogs_ListFilterByMultipleStatusCodes(t *testing.T) {
	t.Parallel()
	codes := discoverDistinctValues(t, "status_code", 2)
	if len(codes) < 2 {
		t.Skip("Need at least 2 distinct status codes in recent request logs")
		return
	}

	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"status_codes": codes,
		"limit":        {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by status_codes %v should return results", codes)

	allowed := map[string]bool{codes[0]: true, codes[1]: true}
	for _, item := range list.Data {
		m := parseJSON(item)
		got := jsonField(m, "status_code")
		assert.True(t, allowed[got], "status_code %q not in requested set %v", got, codes)
	}
}

func TestRequestLogs_ListFilterByMultipleStatusCodesAllImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"status_codes": {"998", "999"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filter by impossible status codes should return no results")
}

func TestRequestLogs_ListFilterByMultipleActorTypes(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"actor_types": {"user", "api_key"},
		"include":     {"actor"},
		"limit":       {"25"},
	})
	require.NoError(t, err)

	allowed := map[string]bool{"user": true, "api_key": true}
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		got := jsonField(actor, "type")
		assert.True(t, allowed[got], "actor.type %q not in requested set", got)
	}
}

func TestRequestLogs_ListFilterByMultipleAccountIDs(t *testing.T) {
	t.Parallel()
	// Discover the account_id present on recent logs by including ?include=account.
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"include": {"account"},
		"limit":   {"5"},
	})
	if err != nil || len(list.Data) == 0 {
		t.Skip("No request logs available to discover account_id")
		return
	}
	first := parseJSON(list.Data[0])
	account := jsonObject(first, "account")
	require.NotNil(t, account, "expected account on first log")
	accountID := jsonField(account, "id")
	require.NotEmpty(t, accountID)

	// Filter by [real, impossible] — every result should still be the real one.
	filtered, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"account_ids": {accountID, "acct_zzzzzzzzzzzzzzzz"},
		"include":     {"account"},
		"limit":       {"10"},
	})
	require.NoError(t, err)

	for _, item := range filtered.Data {
		m := parseJSON(item)
		acct := jsonObject(m, "account")
		require.NotNil(t, acct, "account include missing")
		assert.Equal(t, accountID, jsonField(acct, "id"))
	}
}

func TestRequestLogs_ListFilterByMultipleActorIDs(t *testing.T) {
	t.Parallel()
	// Discover an actor.id from recent logs.
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"include": {"actor"},
		"limit":   {"10"},
	})
	if err != nil || len(list.Data) == 0 {
		t.Skip("No request logs available to discover actor IDs")
		return
	}
	var actorID string
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		if actor == nil {
			continue
		}
		actorID = jsonField(actor, "id")
		if actorID != "" {
			break
		}
	}
	if actorID == "" {
		t.Skip("No request logs with an actor ID available")
		return
	}

	filtered, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"actor_ids": {actorID, "actu_zzzzzzzzzzzzzzzz"},
		"include":   {"actor"},
		"limit":     {"10"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known actor ID should match at least one log")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor)
		assert.Equal(t, actorID, jsonField(actor, "id"))
	}
}

func TestRequestLogs_ListFilterByMultipleNormalizedRoutes(t *testing.T) {
	t.Parallel()
	routes := discoverDistinctValues(t, "normalized_route", 2)
	if len(routes) < 2 {
		t.Skip("Need at least 2 distinct normalized_routes in recent request logs")
		return
	}

	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"normalized_routes": routes,
		"limit":             {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by normalized_routes %v should return results", routes)

	allowed := map[string]bool{routes[0]: true, routes[1]: true}
	for _, item := range list.Data {
		m := parseJSON(item)
		got := jsonField(m, "normalized_route")
		assert.True(t, allowed[got], "normalized_route %q not in requested set %v", got, routes)
	}
}

func TestRequestLogs_ListFilterByMultipleHosts(t *testing.T) {
	t.Parallel()
	hosts := discoverDistinctValues(t, "host", 1)
	if len(hosts) < 1 {
		t.Skip("Need at least 1 distinct host in recent request logs")
		return
	}

	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"hosts": {hosts[0], "no-such-host.example.invalid"},
		"limit": {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by host %v should return results", hosts[0])

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Equal(t, hosts[0], jsonField(m, "host"))
	}
}

func TestRequestLogs_ListFilterByMultipleErrorCodesAllImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"error_codes": {"zzz_no_match_a", "zzz_no_match_b"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filter by impossible error codes should return no results")
}

func TestRequestLogs_ListFilterByNormalizedRoutesImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"normalized_routes": {"/zzz/no/such/route"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by impossible normalized_route should return no results")
}

func TestRequestLogs_ListFilterByHostsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"hosts": {"https://zzz-no-such-host.invalid"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "Filtering by impossible host should return no results")
}

func TestRequestLogs_ListFilterByMinLatency(t *testing.T) {
	t.Parallel()
	// A latency of 0 should not exclude any logs.
	zero := int64(0)
	_ = zero
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"min_latency_us": {"0"},
		"limit":          {"5"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	// Any log should satisfy latency >= 0.
	for _, item := range list.Data {
		m := parseJSON(item)
		latency := jsonField(m, "latency_us")
		assert.NotEmpty(t, latency, "latency_us should be present on each log")
	}

	// A very large latency threshold should return no results.
	list2, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"min_latency_us": {"9999999999999"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list2.Data, "Filtering by extremely high min_latency_us should return no results")
}

// Removed-filter regression: actor_name and exact_match are no longer
// supported. Unknown query params are explicitly rejected with 400.
func TestRequestLogs_ListRejectsRemovedFilters(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(requestLogsPath, url.Values{
		"actor_name":  {"someone"},
		"exact_match": {"true"},
		"limit":       {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 400, statusCode, body)
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
	assert.Nil(t, got["query_params"], "query_params should be null without ?include=query_params")
	assert.Nil(t, got["request_body"], "request_body should be null without ?include=request_body")
	assert.Nil(t, got["response_body"], "response_body should be null without ?include=response_body")
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

func TestRequestLogs_ListFilterByErrorCodeExcludesNonMatching(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"limit": {"50"}})
	if err != nil || len(list.Data) == 0 {
		t.Skip("No request logs available to discover error codes")
		return
	}
	var errorCode string
	for _, item := range list.Data {
		m := parseJSON(item)
		if ec := jsonField(m, "error_code"); ec != "" {
			errorCode = ec
			break
		}
	}
	if errorCode == "" {
		t.Skip("No request logs with an error_code available")
		return
	}

	filtered, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"error_codes": {errorCode},
		"limit":       {"200"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known error_code should return at least one log")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		assert.Equal(t, errorCode, jsonField(m, "error_code"), "All results should match the filtered error_code")
	}
}

func TestRequestLogs_ListFilterByMinLatencyVerifiesThreshold(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{"limit": {"25"}})
	if err != nil || len(list.Data) == 0 {
		t.Skip("No request logs available")
		return
	}

	var latencies []float64
	for _, item := range list.Data {
		m := parseJSON(item)
		latStr := jsonField(m, "latency_us")
		if latStr == "" {
			continue
		}
		lat, parseErr := strconv.ParseFloat(latStr, 64)
		if parseErr != nil {
			continue
		}
		latencies = append(latencies, lat)
	}
	if len(latencies) == 0 {
		t.Skip("Could not determine latency values from recent logs")
		return
	}
	sort.Float64s(latencies)
	threshold := int64(latencies[len(latencies)/2])
	if threshold == 0 {
		t.Skip("Median latency is 0; cannot perform a meaningful threshold check")
		return
	}

	filtered, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"min_latency_us": {strconv.FormatInt(threshold, 10)},
		"limit":          {"25"},
	})
	require.NoError(t, err)

	for _, item := range filtered.Data {
		m := parseJSON(item)
		latStr := jsonField(m, "latency_us")
		require.NotEmpty(t, latStr, "latency_us should be present on each log")
		lat, parseErr := strconv.ParseFloat(latStr, 64)
		require.NoError(t, parseErr)
		assert.GreaterOrEqual(t, int64(lat), threshold, "All results should have latency_us >= %d", threshold)
	}
}

func TestRequestLogs_ListFilterByStartDateExcludesAll(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"start_date": {"2099-01-01T00:00:00Z"},
		"limit":      {"5"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "start_date far in the future should exclude all logs")
}

func TestRequestLogs_ListFilterByEndDateExcludesAll(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(requestLogsPath, url.Values{
		"end_date": {"2000-01-01T00:00:00Z"},
		"limit":    {"5"},
	})
	if err != nil {
		t.Skip("Request logs endpoint not accessible")
		return
	}
	assertEmptyListData(t, list.Data, "end_date far in the past should exclude all logs")
}

func TestRequestLogs_IncludeQueryParams(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(requestLogsPath+"/"+SeedRequestLogQueryParamsID, url.Values{"include": {"query_params"}})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, status)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedRequestLogQueryParamsID, jsonField(got, "id"))

	qp := jsonObject(got, "query_params")
	require.NotNil(t, qp, "query_params should be present with ?include=query_params")
	assert.Equal(t, "10", jsonField(qp, "limit"), "query_params.limit should match seeded value")
}

func TestRequestLogs_CapturesPayloads(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-rqlog")
	idemKey := newIdempotencyKey()
	path := itemCategoriesPath + "?include=unit_group"
	createBody := map[string]any{
		"name":          name,
		"type":          "material_category",
		"unit_group_id": SeedUnitGroupID,
	}
	status, respBody, err := apiClient.Post(path, createBody, idemKey)
	require.NoError(t, err)
	skipOnNonClientError(t, itemCategoriesPath, status)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	createdID := jsonField(created, "id")
	require.NotEmpty(t, createdID)
	t.Cleanup(func() { apiClient.Delete(itemCategoriesPath + "/" + createdID) })

	var logID string
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		list, _, err := apiClient.GetList(requestLogsPath, url.Values{
			"idempotency_key": {idemKey},
			"limit":           {"1"},
		})
		if err != nil {
			return err
		}
		if len(list.Data) == 0 {
			return fmt.Errorf("no request log yet for idempotency key %s", idemKey)
		}
		m := parseJSON(list.Data[0])
		logID = jsonField(m, "id")
		if logID == "" {
			return fmt.Errorf("log item missing id")
		}
		return nil
	})
	require.NotEmpty(t, logID)

	getStatus, getBody, err := apiClient.GetListRaw(requestLogsPath+"/"+logID, url.Values{
		"include": {"request_body", "response_body", "query_params"},
	})
	require.NoError(t, err)
	skipOnNonClientError(t, requestLogsPath, getStatus)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	reqBody := jsonObject(got, "request_body")
	require.NotNil(t, reqBody, "request_body should be present with ?include=request_body")
	assert.Equal(t, name, jsonField(reqBody, "name"))

	respObj := jsonObject(got, "response_body")
	require.NotNil(t, respObj, "response_body should be present with ?include=response_body")
	assert.Equal(t, "item_category", jsonField(respObj, "object"))

	qp := jsonObject(got, "query_params")
	require.NotNil(t, qp, "query_params should be present with ?include=query_params")
	assert.Equal(t, "unit_group", jsonField(qp, "include"))
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
