//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	auditEventsPath             = "/v1/core/audit-events"
	auditEventResourceTypesPath = "/v1/core/audit-events/resource-types"
)

func TestAuditEvents_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.NotNil(t, list.Data)
}

func TestAuditEvents_ListSearchByResourceType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"q": {"unit"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'unit' should return at least 1 result")

	for _, item := range list.Data {
		m := parseJSON(item)
		resourceType := jsonField(m, "resource_type")
		assert.True(t,
			strings.Contains(strings.ToLower(resourceType), "unit"),
			"Search result resource_type %q should contain 'unit'", resourceType,
		)
	}
}

func TestAuditEvents_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"q": {"zzzznotanevent99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestAuditEvents_FilterByAction(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"actions": {"create"}, "limit": {"5"}})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.Equal(t, "create", DataItemField(item, "action"))
	}
}

func TestAuditEvents_FilterBySingleResourceType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_types": {"user"},
		"limit":          {"25"},
	})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.Equal(t, "user", DataItemField(item, "resource_type"))
	}
}

func TestAuditEvents_FilterByMultipleResourceTypes(t *testing.T) {
	t.Parallel()
	allowed := map[string]struct{}{"user": {}, "account": {}}
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_types": {"user", "account"},
		"limit":          {"50"},
	})
	require.NoError(t, err)
	for _, item := range list.Data {
		got := DataItemField(item, "resource_type")
		_, ok := allowed[got]
		assert.True(t, ok, "resource_type %q not in {user, account}", got)
	}
}

func TestAuditEvents_FilterByResourceTypeNoMatch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_types": {"zzz_definitely_not_a_real_resource_type"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Multi-value filters ---

// discoverDistinctAuditValues returns up to `n` distinct values of the given
// JSON field across recent audit events. Used to seed multi-value filter tests
// with real data the harness has produced.
func discoverDistinctAuditValues(t *testing.T, field string, n int) []string {
	t.Helper()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"50"}})
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

func TestAuditEvents_FilterByMultipleActions(t *testing.T) {
	t.Parallel()
	actions := discoverDistinctAuditValues(t, "action", 2)
	if len(actions) < 2 {
		t.Skip("Need at least 2 distinct audit actions in recent events")
		return
	}

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actions": actions,
		"limit":   {"50"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by actions %v should return results", actions)

	allowed := map[string]bool{actions[0]: true, actions[1]: true}
	for _, item := range list.Data {
		got := DataItemField(item, "action")
		assert.True(t, allowed[got], "action %q not in requested set %v", got, actions)
	}
}

func TestAuditEvents_FilterByMultipleActionsAllImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actions": {"zzz_no_match_a", "zzz_no_match_b"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filter by impossible actions should return no results")
}

func TestAuditEvents_FilterBySingleResourceID(t *testing.T) {
	t.Parallel()
	ids := discoverDistinctAuditValues(t, "resource_id", 1)
	if len(ids) == 0 {
		t.Skip("No resource_ids available in recent audit events")
		return
	}
	resourceID := ids[0]

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids": {resourceID},
		"limit":        {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filtering by resource_id %q should return at least one event", resourceID)

	for _, item := range list.Data {
		assert.Equal(t, resourceID, DataItemField(item, "resource_id"), "All results should match the filtered resource_id")
	}
}

func TestAuditEvents_FilterByMultipleResourceIDs(t *testing.T) {
	t.Parallel()
	ids := discoverDistinctAuditValues(t, "resource_id", 2)
	if len(ids) < 2 {
		t.Skip("Need at least 2 distinct resource_ids in recent audit events")
		return
	}

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids": ids,
		"limit":        {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by resource_ids %v should return results", ids)

	allowed := map[string]bool{ids[0]: true, ids[1]: true}
	for _, item := range list.Data {
		got := DataItemField(item, "resource_id")
		assert.True(t, allowed[got], "resource_id %q not in requested set %v", got, ids)
	}
}

func TestAuditEvents_FilterByResourceIDsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids": {"zzz_no_such_resource_id"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filtering by impossible resource_id should return no results")
}

// discoverAuditActorID fetches recent events with ?include=actor and returns
// the first actor ID found, or empty string if none.
func discoverAuditActorID(t *testing.T) string {
	t.Helper()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"include": {"actor"},
		"limit":   {"10"},
	})
	if err != nil || len(list.Data) == 0 {
		return ""
	}
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		if actor == nil {
			continue
		}
		if id := jsonField(actor, "id"); id != "" {
			return id
		}
	}
	return ""
}

func TestAuditEvents_FilterByActorIDSingle(t *testing.T) {
	t.Parallel()
	actorID := discoverAuditActorID(t)
	if actorID == "" {
		t.Skip("No audit events with an actor ID available")
		return
	}

	filtered, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {actorID},
		"include":   {"actor"},
		"limit":     {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known actor_id should return at least one event")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		assert.Equal(t, actorID, jsonField(actor, "id"), "All results should match the filtered actor_id")
	}
}

func TestAuditEvents_FilterByMultipleActorIDs(t *testing.T) {
	t.Parallel()
	actorID := discoverAuditActorID(t)
	if actorID == "" {
		t.Skip("No audit events with an actor ID available")
		return
	}

	filtered, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {actorID, "actu_zzzzzzzzzzzzzzzz"},
		"include":   {"actor"},
		"limit":     {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known actor_id should match at least one event")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor)
		assert.Equal(t, actorID, jsonField(actor, "id"))
	}
}

func TestAuditEvents_FilterByActorIDsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {"actu_zzzzzzzzzzzzzzzz", "ak_zzzzzzzzzzzzzzzz"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filtering by impossible actor_ids should return no results")
}

// account_id was a dead filter (never reached SQL) and has been removed.
// Unknown query params are explicitly rejected with 400.
func TestAuditEvents_ListRejectsRemovedAccountIDFilter(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
		"account_id": {"acct_anything"},
		"limit":      {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 400, statusCode, body)
}

func TestAuditEventResourceTypes_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventResourceTypesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.Greater(t, len(list.Data), 50, "resource type enum should be substantial")

	values := make(map[string]struct{}, len(list.Data))
	for _, item := range list.Data {
		var v string
		require.NoError(t, json.Unmarshal(item, &v))
		values[v] = struct{}{}
	}
	for _, expected := range []string{"audit_event", "user", "account", "sales_order"} {
		_, ok := values[expected]
		assert.True(t, ok, "expected resource type %q to be present", expected)
	}
}

func TestAuditEvents_GetByID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("No audit events available")
	}

	eventID := DataItemField(list.Data[0], "id")
	assert.NotEmpty(t, eventID)

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, "audit_event", jsonField(got, "object"))
	assert.Equal(t, eventID, jsonField(got, "id"))
	assert.NotEmpty(t, jsonField(got, "action"))
	assert.NotEmpty(t, jsonField(got, "resource_type"))
	assert.NotEmpty(t, jsonField(got, "resource_id"))
	assert.NotEmpty(t, jsonField(got, "occurred_at"))
}

func TestAuditEvents_IncludeChanges(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"changes"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	_, ok := got["changes"]
	assert.True(t, ok, "changes should be present with ?include=changes")
}

func TestAuditEvents_IncludeMetadata(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"metadata"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Contains(t, got, "metadata", "metadata key should be present with ?include=metadata")
}

func TestAuditEvents_IncludeRequest(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"include": {"request"},
		"limit":   {"25"},
	})
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		req := jsonObject(m, "request")
		if req == nil {
			continue
		}
		assert.Equal(t, "request_log", jsonField(req, "object"),
			"request sub-resource object type should be request_log")
		assert.NotEmpty(t, jsonField(req, "id"), "request sub-resource should have an id")
		return
	}
	t.Skip("No audit events with a request_id found in the first 25 events")
}

func TestAuditEvents_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Nil(t, got["actor"], "actor should be null without ?include=actor")
	assert.Nil(t, got["changes"], "changes should be null without ?include=changes")
	assert.Nil(t, got["metadata"], "metadata should be null without ?include=metadata")
	assert.Nil(t, got["request"], "request should be null without ?include=request")
}

func TestAuditEvents_IncludeActor(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"actor"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	actor := jsonObject(parseJSON(getBody), "actor")
	require.NotNil(t, actor)
	assert.NotEmpty(t, jsonField(actor, "id"))
	assert.Equal(t, "actor", jsonField(actor, "object"))
	assert.NotEmpty(t, jsonField(actor, "type"))
}
