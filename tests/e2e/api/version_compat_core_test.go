//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertPreview9Actor checks an actor in preview.9's expandable shape, which forge's ActorRef downgrades to: no
// handle, avatar or role any more.
func assertPreview9Actor(t *testing.T, actor map[string]any, wantType string) {
	t.Helper()
	require.NotNil(t, actor)
	assert.ElementsMatch(t, []string{"id", "object", "type", "name", "handle", "avatar_url", "role"}, mapKeys(actor))
	assert.Equal(t, "actor", jsonField(actor, "object"))
	assert.Equal(t, wantType, jsonField(actor, "type"))
	assert.NotEmpty(t, jsonField(actor, "id"))
	assert.Nil(t, actor["handle"])
	assert.Nil(t, actor["avatar_url"])
	assert.Nil(t, actor["role"])
}

// A preview.9 caller still filters audit events on starts_at/ends_at and the singular root filters, asks for the
// actor, and reads it as an actor; a system actor reads as null.
func TestVersionCompatCore_AuditEventsOnPreview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	list, status, err := preview.GetList(auditEventsPath, url.Values{
		"actor_types": {"user"}, "starts_at": {"2000-01-01T00:00:00Z"}, "include": {"actor", "changes"}, "limit": {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)
	for _, raw := range list.Data {
		assertPreview9Actor(t, jsonObject(parseJSON(raw), "actor"), "user")
	}

	future, status, err := preview.GetList(auditEventsPath, url.Values{"starts_at": {"2099-01-01T00:00:00Z"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assertEmptyListData(t, future.Data, "starts_at still bounds the list")

	status, body, err := preview.GetListRaw(auditEventsPath, url.Values{
		"root_resource_type": {"sales_order"}, "root_resource_id": {SeedSalesOrderID}, "limit": {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	system, status, err := preview.GetList(auditEventsPath, url.Values{"actor_types": {"system"}, "starts_at": {"2000-01-01T00:00:00Z"}, "limit": {"5"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	for _, raw := range system.Data {
		assert.Nil(t, parseJSON(raw)["actor"], "a system actor reads as null on preview.9")
	}
}

// A preview.9 caller still filters request logs on starts_at and reads the actor as an actor.
func TestVersionCompatCore_RequestLogsOnPreview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	list, status, err := preview.GetList(requestLogsPath, url.Values{"actor_types": {"api_key"}, "include": {"actor"}, "limit": {"5"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)
	for _, raw := range list.Data {
		assertPreview9Actor(t, jsonObject(parseJSON(raw), "actor"), "api_key")
	}

	future, status, err := preview.GetList(requestLogsPath, url.Values{"starts_at": {"2099-01-01T00:00:00Z"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assertEmptyListData(t, future.Data, "starts_at still bounds the list")
}

// A preview.9 caller reads an email log's sender as an actor and sees no related record.
func TestVersionCompatCore_EmailLogOnPreview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	status, body, err := preview.GetListRaw(emailLogsPath+"/"+SeedEmailLogID1, url.Values{"include": {"sent_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	sentBy := jsonObject(got, "sent_by")
	assertPreview9Actor(t, sentBy, "user")
	assert.Equal(t, SeedUserID, jsonField(sentBy, "id"))
	assert.NotContains(t, got, "related")
}

// A preview.9 caller reads a job at its old path with its creator as an actor.
func TestVersionCompatCore_JobOnPreview9(t *testing.T) {
	t.Parallel()
	jobID := acceptBulkUpsertUnits(t, upsertUnitInput(uniqueName("e2e-compat-job"), uniqueName("cjb"), "quantity"))
	job := pollJobUntilTerminal(t, jobID)
	if created, _ := jobResultIDs(job); len(created) > 0 {
		defer cleanupUnitIDs(created)
	}

	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw("/v1/core/jobs/"+jobID, url.Values{"include": {"created_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "job", jsonField(got, "object"))
	createdBy := jsonObject(got, "created_by")
	assertPreview9Actor(t, createdBy, "api_key")
	assert.Equal(t, SeedAPIKeyID, jsonField(createdBy, "id"))
}

// A preview.9 caller's search returns entity references named as before, and still takes customer and
// messaging_contact.
func TestVersionCompatCore_SearchOnPreview9(t *testing.T) {
	t.Parallel()
	preview := apiClient.WithAPIVersion(preview9Version)

	for _, tc := range covSearchCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			preview9 := tc
			if tc.typ == "account_user" {
				preview9.typ = "messaging_contact"
			}
			row := covSearchHit(t, preview, preview9, "id", nil)
			var handle any
			if tc.entityHandle != nil {
				handle = *tc.entityHandle
			}
			assert.Equal(t, map[string]any{
				"id": tc.id, "object": "entity", "type": preview9.typ, "name": tc.entityName, "handle": handle,
			}, row)
		})
	}

	list, status, err := preview.GetList(covCoreSearchPath, url.Values{
		"customer": {SeedCustomerAccountID}, "q": {SeedSalesOrderPONumber}, "types": {"sales_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)
	assert.Equal(t, SeedPOSalesOrderID, DataItemField(list.Data[0], "id"))
}
