package versiontransforms

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

func TestPreview9_AuditEventActor(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeAuditEvent, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"id":"aev_1","object":"audit_event","actor":{"type":"user","id":"acu_1","name":"Ada","client":null},
		 "request":{"id":"rq_1","object":"request_log","actor":{"type":"api_key","id":"ak_1","name":"Sync","client":"Claude Code"},"query_params":{"limit":["10"],"ids":["a","b"]}}},
		{"id":"aev_2","object":"audit_event","actor":{"type":"system","id":null,"name":"Stripe","client":null}}]}`)
	data := got["data"].([]any)
	first := data[0].(map[string]any)
	assert.Equal(t, map[string]any{"id": "acu_1", "object": "actor", "type": "user", "name": "Ada", "handle": nil, "avatar_url": nil, "role": nil}, first["actor"])
	req := first["request"].(map[string]any)
	assert.Equal(t, "ak_1", req["actor"].(map[string]any)["id"])
	assert.NotContains(t, req["actor"], "client")
	assert.Equal(t, map[string]any{"limit": "10", "ids": []any{"a", "b"}}, req["query_params"])
	assert.Nil(t, data[1].(map[string]any)["actor"])
}

func TestPreview9_EmailLogAndJob(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeEmailLog, `{"id":"eml_1","object":"email_log",
		"sent_by":{"type":"user","id":"usr_1","name":"Ada","client":null},
		"related":{"object":"email_log_related","record":null}}`)
	assert.Equal(t, "actor", got["sent_by"].(map[string]any)["object"])
	assert.NotContains(t, got, "related")

	job := downgrade(t, constants.ObjectTypeAsyncJob, `{"id":"jb_1","object":"async_job","status":"queued",
		"created_by":{"type":"system","id":null,"name":"OpenMRP","client":null}}`)
	assert.Equal(t, "job", job["object"])
	assert.Nil(t, job["created_by"])
}

func TestPreview9_SearchResultIsAnEntity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"entity"}, version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeSearchResult))

	got := downgrade(t, constants.ObjectTypeSearchResult, `{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[
		{"object":"search_result","type":"customer","resource_id":"ac_1","resource":null,
		 "entity":{"id":"ac_1","object":"entity","type":"customer","name":"Acme","handle":"100"}},
		{"object":"search_result","type":"account_user","resource_id":"acu_1","resource":null,
		 "entity":{"id":"acu_1","object":"entity","type":"account_user","name":"Ada","handle":"user"}},
		{"object":"search_result","type":"item","resource_id":"itm_1","resource":null,"entity":null}]}`)
	data := got["data"].([]any)
	assert.Equal(t, map[string]any{"id": "ac_1", "object": "entity", "type": "customer", "name": "Acme", "handle": "100"}, data[0])
	assert.Equal(t, "messaging_contact", data[1].(map[string]any)["type"])
	assert.Equal(t, map[string]any{"id": "itm_1", "object": "entity", "type": "item", "name": nil, "handle": nil}, data[2])
}

func TestPreview9_CoreLogQueries(t *testing.T) {
	RegisterRouteIncludes(auditEventsRoute, []string{"account", "request"})

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeAuditEvent, auditEventsRoute, url.Values{
		"starts_at":          {"2026-01-01T00:00:00Z"},
		"ends_at":            {"2026-02-01T00:00:00Z"},
		"root_resource_type": {"sales_order"},
		"root_resource_id":   {"so_1"},
		"include[]":          {"actor", "actor.role", "account", "request.actor"},
	})
	assert.Equal(t, []string{"2026-01-01T00:00:00Z"}, q["occurred_after"])
	assert.Equal(t, []string{"2026-02-01T00:00:00Z"}, q["occurred_before"])
	assert.Equal(t, []string{"sales_order"}, q["root_resource_types[]"])
	assert.Equal(t, []string{"so_1"}, q["root_resource_ids[]"])
	require.NotContains(t, q, "starts_at")
	assert.Equal(t, []string{"account"}, q["include[]"])

	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeEmailLog, "/v1/core/email-logs", url.Values{
		"include": {"sent_by"},
	})
	assert.NotContains(t, q, "include")

	q = version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeSearchResult, searchRoute, url.Values{
		"customer": {"ac_1"},
		"types[]":  {"messaging_contact", "sales_order"},
	})
	assert.Equal(t, []string{"ac_1"}, q["customer_ids[]"])
	assert.Equal(t, []string{"account_user", "sales_order"}, q["types[]"])
}
