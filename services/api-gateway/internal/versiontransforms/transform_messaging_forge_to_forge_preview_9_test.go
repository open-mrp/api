package versiontransforms

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/version"
)

const sampleRecordJSON = `{"id":"so_1","object":"record","type":"sales_order","number":"SO-1042","status":"open","metadata":{}}`

func TestPreview9_ConversationTopicAndUnread(t *testing.T) {
	t.Parallel()

	got := downgrade(t, constants.ObjectTypeConversation, `{"id":"cv_1","object":"conversation","status":"active","unread_count":3,
		"related":{"object":"conversation_related","record":`+sampleRecordJSON+`,"continues":null},
		"participants":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[{"id":"cvpt_1","object":"conversation_participant","membership":"hidden"}]}}`)
	assert.Equal(t, float64(3), got["unread"])
	assert.NotContains(t, got, "unread_count")
	assert.NotContains(t, got, "related")
	assert.Equal(t, map[string]any{"id": "so_1", "object": "entity", "type": "sales_order", "name": "SO-1042", "handle": nil}, got["topic"])
	p := got["participants"].(map[string]any)["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "active", p["membership"])

	bare := downgrade(t, constants.ObjectTypeConversation, `{"id":"cv_2","object":"conversation","unread_count":0,"related":{"object":"conversation_related","record":null,"continues":null}}`)
	assert.Contains(t, bare, "topic")
	assert.Nil(t, bare["topic"])
}

func TestPreview9_MessageResourceStatusAndAgentFailure(t *testing.T) {
	t.Parallel()

	failed := downgrade(t, constants.ObjectTypeChatMessage, `{"id":"mg_1","object":"chat_message","status":"failed","body":"Sorry.",
		"error":{"code":"agent_spending_cap_reached","message":"Sorry."},"related":{"object":"chat_message_related","record":`+sampleRecordJSON+`},
		"attachments":{"object":"list","page_info":{"next_page_url":null,"previous_page_url":null},"data":[{"id":"mgah_1","object":"message_attachment","record":`+sampleRecordJSON+`}]}}`)
	assert.Equal(t, "sent", failed["status"])
	assert.Equal(t, true, failed["agent_run_failed"])
	assert.Equal(t, "agent_spending_cap_reached", failed["agent_error_code"])
	assert.NotContains(t, failed, "error")
	assert.Equal(t, "so_1", failed["resource"].(map[string]any)["id"])
	att := failed["attachments"].(map[string]any)["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "entity", att["resource"].(map[string]any)["object"])
	assert.NotContains(t, att, "record")

	deleted := downgrade(t, constants.ObjectTypeChatMessage, `{"id":"mg_2","object":"chat_message","status":"deleted","body":null,"error":null,"related":{"object":"chat_message_related","record":null}}`)
	assert.Equal(t, "sent", deleted["status"])
	assert.Equal(t, false, deleted["agent_run_failed"])
	assert.Nil(t, deleted["agent_error_code"])
	assert.Nil(t, deleted["resource"])

	scheduledFailure := downgrade(t, constants.ObjectTypeChatMessage, `{"id":"mg_3","object":"chat_message","status":"failed","error":null}`)
	assert.Equal(t, "failed", scheduledFailure["status"], "a scheduled send that failed stays failed")
}

func TestPreview9_LinkAndAnnouncementRecords(t *testing.T) {
	t.Parallel()

	link := downgrade(t, constants.ObjectTypeConversationLink, `{"id":"cvlk_1","object":"conversation_link","record":`+sampleRecordJSON+`,"conversation":null}`)
	assert.Equal(t, "so_1", link["resource"].(map[string]any)["id"])
	assert.Contains(t, version.ForcedIncludes(apiversion.V1_0_Forge, apiversion.V1_0_Forge_Preview9, constants.ObjectTypeConversationLink), "record")

	ann := downgrade(t, constants.ObjectTypeAnnouncement, `{"id":"an_1","object":"announcement","related":{"object":"announcement_related","record":null}}`)
	assert.Nil(t, ann["resource"])
	assert.NotContains(t, ann, "related")
}

func upgradeBody(t *testing.T, ot constants.ObjectType, route string, body map[string]any) map[string]any {
	t.Helper()
	return version.TransformEndpointRequest(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, ot, http.MethodPost, route, body)
}

func TestPreview9_CreateConversationBody(t *testing.T) {
	t.Parallel()

	dm := upgradeBody(t, constants.ObjectTypeConversation, "/v1/messaging/conversations", map[string]any{
		"type": "direct_message", "participant_account_user_ids": []any{"acus_b"}})
	assert.Equal(t, []any{map[string]any{"account_user_id": "acus_b"}}, dm["participants"])
	assert.NotContains(t, dm, "type")

	group := upgradeBody(t, constants.ObjectTypeConversation, "/v1/messaging/conversations", map[string]any{
		"type": "group", "title": "Ops", "participant_account_user_ids": []any{"acus_b"}})
	assert.Equal(t, []any{map[string]any{"account_user_id": "acus_b", "role": "member"}}, group["participants"], "a one-member group stays a group")

	anchored := upgradeBody(t, constants.ObjectTypeConversation, "/v1/messaging/conversations", map[string]any{
		"type": "group", "participant_account_user_ids": []any{"acus_b", "acus_c"},
		"participants":        []any{map[string]any{"account_user_id": "acus_c", "role": "admin"}},
		"topic_resource_type": "sales_order", "topic_resource_id": "so_1"})
	require.Len(t, anchored["participants"], 2)
	assert.Equal(t, map[string]any{"account_user_id": "acus_b"}, anchored["participants"].([]any)[0])
	assert.Equal(t, map[string]any{"account_user_id": "acus_c", "role": "admin"}, anchored["participants"].([]any)[1])
	assert.Equal(t, map[string]any{"type": "sales_order", "id": "so_1"}, anchored["record"])
	assert.NotContains(t, anchored, "topic_resource_type")
}

func TestPreview9_SendMessageAndLinkBodies(t *testing.T) {
	t.Parallel()

	msg := upgradeBody(t, constants.ObjectTypeChatMessage, "/v1/messaging/conversations/{id}/messages", map[string]any{
		"body": "see", "link_resource_type": "invoice", "link_resource_id": "inv_1",
		"attachments": []any{map[string]any{"kind": "resource", "resource_type": "sales_order", "resource_id": "so_1"}}})
	assert.Equal(t, map[string]any{"type": "invoice", "id": "inv_1"}, msg["record"])
	assert.Equal(t, map[string]any{"type": "sales_order", "id": "so_1"}, msg["attachments"].([]any)[0].(map[string]any)["record"])

	link := upgradeBody(t, constants.ObjectTypeConversationLink, "/v1/messaging/conversations/{id}/links", map[string]any{
		"resource_type": "customer", "resource_id": "cu_1"})
	assert.Equal(t, map[string]any{"type": "customer", "id": "cu_1"}, link["record"])
}

func TestPreview9_ConversationQueries(t *testing.T) {
	t.Parallel()

	q := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeConversation, "/v1/messaging/conversations",
		url.Values{"status": {"hidden"}, "topic_resource_type": {"sales_order"}, "topic_resource_id": {"so_1"}, "q": {"x"},
			"include[]": {"topic", "last_message.resource", "last_message.attachments.resource"}})
	assert.Equal(t, "hidden", q.Get("memberships[]"))
	assert.Equal(t, "sales_order", q.Get("record_type"))
	assert.Equal(t, "so_1", q.Get("record_id"))
	assert.Empty(t, q.Get("q"))
	assert.Equal(t, []string{"related.record", "last_message.related.record", "last_message.attachments.record"}, q["include[]"])

	links := version.TransformQuery(apiversion.V1_0_Forge_Preview9, apiversion.V1_0_Forge, constants.ObjectTypeConversationLink, "/v1/messaging/conversations/{id}/links",
		url.Values{"include": {"resource,conversation.topic"}})
	assert.Equal(t, "record,conversation.related.record", links.Get("include"))
	assert.Empty(t, links.Get("limit"), "a create on the same route must not gain list parameters")
}
