//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
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
}

func TestAuditEvents_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"q": {"zzzznotanevent99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestAuditEvents_FilterByAction(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"action": {"create"}, "limit": {"5"}})
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
