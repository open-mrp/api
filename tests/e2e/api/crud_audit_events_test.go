//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const auditEventsPath = "/v1/core/audit-events"

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
	assert.NotEmpty(t, jsonField(actor, "object"))
}
