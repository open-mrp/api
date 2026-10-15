//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage-gap tests for the `/v1/messaging/conversations` endpoint group (conversationep +
// participantep), on top of the extensive existing coverage in messaging_chat_test.go,
// messaging_chat_advanced_test.go, messaging_chat_groups_test.go, messaging_chat_lifecycle_test.go,
// messaging_chat_visibility_test.go, messaging_chat_agents_test.go, messaging_external_cases_test.go,
// messaging_customer_support_test.go, messaging_reports_test.go, messaging_redaction_test.go, and
// messaging_conversation_lifecycle_extra_test.go. This file fills the gaps identified in the e2e
// task audit: full CreateConversationRequest/UpdateConversationRequest/AssignConversationRequest/
// SetWorkflowStatusRequest validation, previously-unasserted response fields (topic, last_message,
// created_at/updated_at, agent_trigger_policy/keywords nil, ConversationLink.conversation/created_at),
// Idempotency-Key replay semantics, list pagination/unknown-query-param/invalid-enum-filter coverage,
// and unknown-id 404s on the participant/agent/link removal endpoints.
//
// Tests that mutate the shared per-customer support case (seedSupportRoute + openCustomerCase, both
// defined in messaging_customer_support_test.go / messaging_external_cases_test.go) intentionally do
// NOT call t.Parallel(), mirroring the existing TestExternalCase_* convention: that conversation is a
// single shared row across the whole suite, so those tests must run sequentially relative to each
// other and to messaging_external_cases_test.go / messaging_customer_support_test.go.

// ──────────────────────────────────────────────
// CreateConversationRequest validation
// ──────────────────────────────────────────────

// type is set from who is in the conversation; it is not a request field.
func TestCovMessagingConversations_Create_TypeIsNotAField(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath, map[string]any{
		"type":         "group",
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
}

func TestCovMessagingConversations_Create_DerivesType(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	create := func(body map[string]any) map[string]any {
		resp, err := user.PostFull(conversationsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, resp.StatusCode, resp.Body)
		return parseJSON(resp.Body)
	}
	one := []map[string]any{{"account_user_id": SeedAccountUser2ID}}
	assert.Equal(t, "direct_message", jsonField(create(map[string]any{"participants": one}), "type"))
	assert.Equal(t, "direct_message", jsonField(create(map[string]any{"participants": one, "title": "Plans"}), "type"), "a title alone keeps a direct message")
	assert.Equal(t, "group", jsonField(create(map[string]any{"participants": []map[string]any{
		{"account_user_id": SeedAccountUser2ID}, {"account_user_id": SeedAdmin2AccountUserID}}}), "type"))
	assert.Equal(t, "group", jsonField(create(map[string]any{"participants": one,
		"record": map[string]any{"type": "sales_order", "id": SeedSalesOrderID}}), "type"), "a record discussion is a group")
	assert.Equal(t, "group", jsonField(create(map[string]any{"participants": []map[string]any{
		{"account_user_id": SeedAccountUser2ID, "role": "member"}}}), "type"), "a role makes a group")
}

func TestCovMessagingConversations_Create_InvalidRoleEnum(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID, "role": "boss"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovMessagingConversations_Create_NoOtherParticipant(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath, map[string]any{"participants": []map[string]any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "participants")
}

func TestCovMessagingConversations_Create_NonexistentParticipant(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": "acus_doesnotexist000"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "participants")
}

// A self-DM ("note to self") is allowed, and like any conversation every create starts a new one.
func TestCovMessagingConversations_Create_SelfDM(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	body := map[string]any{"participants": []map[string]any{{"account_user_id": SeedAccountUserID}}}
	resp, err := user.PostFull(conversationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	conv := parseJSON(resp.Body)
	assert.Equal(t, "direct_message", jsonField(conv, "type"))

	again, err := user.PostFull(conversationsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, again.StatusCode, again.Body)
	assert.NotEqual(t, jsonField(conv, "id"), jsonField(parseJSON(again.Body), "id"))
}

func TestCovMessagingConversations_Create_GroupIDNonexistent(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath, map[string]any{
		"group_id": "mg_doesnotexist",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "group_id")
}

func TestCovMessagingConversations_Create_InvalidRecordType(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
		"record":       map[string]any{"type": "not_a_type", "id": "x"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "record.type")
}

// The record a conversation is about appears in related.record: a lightweight stub, null unless included.
func TestCovMessagingConversations_Create_RecordIsRelatedStub(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	resp, err := user.PostFull(conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
		"record":       map[string]any{"type": "sales_order", "id": SeedSalesOrderID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	created := parseJSON(resp.Body)
	convID := jsonField(created, "id")
	related := jsonObject(created, "related")
	require.NotNil(t, related, "related is always present")
	assert.Equal(t, "conversation_related", jsonField(related, "object"))
	assertNilField(t, related, "record")
	assertNilField(t, related, "continues")

	withInclude, err := user.GetFull(conversationsPath+"/"+convID, url.Values{"include": {"related.record"}})
	require.NoError(t, err)
	requireStatus(t, 200, withInclude.StatusCode, withInclude.Body)
	record := jsonObject(jsonObject(parseJSON(withInclude.Body), "related"), "record")
	require.NotNil(t, record, "related.record should be populated with ?include=related.record")
	assert.Equal(t, "record", jsonField(record, "object"))
	assert.Equal(t, SeedSalesOrderID, jsonField(record, "id"))
	assert.Equal(t, "sales_order", jsonField(record, "type"))
	assert.NotEmpty(t, jsonField(record, "number"))

	// The record's discussions list it.
	list, _, err := user.GetList(conversationsPath, url.Values{"record_type": {"sales_order"}, "record_id": {SeedSalesOrderID}})
	require.NoError(t, err)
	var found bool
	for _, raw := range list.Data {
		found = found || jsonField(parseJSON(raw), "id") == convID
	}
	assert.True(t, found, "the conversation is among the record's discussions")
}

// A first message is posted by the creator as the conversation's opening message.
func TestCovMessagingConversations_Create_WithFirstMessage(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	body := uniqueName("first")
	resp, err := user.PostFull(withQuery(conversationsPath, url.Values{"include": {"last_message", "last_message.sender"}}), map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
		"message":      map[string]any{"body": body},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	conv := parseJSON(resp.Body)
	require.NotEmpty(t, jsonField(conv, "last_message_at"), "the first message was posted")

	msgs, _, err := user.GetList(conversationsPath+"/"+jsonField(conv, "id")+"/messages", nil)
	require.NoError(t, err)
	require.Len(t, msgs.Data, 1)
	first := parseJSON(msgs.Data[0])
	assert.Equal(t, body, jsonField(first, "body"))
	assert.Equal(t, "sent", jsonField(first, "status"))
}

// continues links the new conversation to one the caller is in, through related.continues.
func TestCovMessagingConversations_Create_Continues(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	old := createDM(t, user, SeedAccountUser2ID)
	oldID := jsonField(old, "id")

	resp, err := user.PostFull(withQuery(conversationsPath, url.Values{"include": {"related.continues"}}), map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
		"continues":    oldID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	cont := jsonObject(jsonObject(parseJSON(resp.Body), "related"), "continues")
	require.NotNil(t, cont)
	assert.Equal(t, "record", jsonField(cont, "object"))
	assert.Equal(t, "conversation", jsonField(cont, "type"))
	assert.Equal(t, oldID, jsonField(cont, "id"))
	assert.Equal(t, "active", jsonField(cont, "status"))

	// A conversation the caller isn't in can't be continued.
	other := chatUser2Client(t)
	theirs := createDM(t, other, SeedAdmin2AccountUserID)
	status, body, err := user.Post(conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
		"continues":    jsonField(theirs, "id"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "continues")
}

// ──────────────────────────────────────────────
// Response shape: all fields, defaults, id formats, timestamps
// ──────────────────────────────────────────────

func TestCovMessagingConversations_Create_AllFieldsAndDefaults(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	resp, err := owner.PostFull(withQuery(conversationsPath, conversationIncludeQuery), map[string]any{
		"title":        uniqueName("cov-allfields"),
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID, "role": "member"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	conv := parseJSON(resp.Body)

	assertObjectField(t, conv, "conversation")
	assertIDFormat(t, jsonField(conv, "id"), "cv")
	assert.Equal(t, "group", jsonField(conv, "type"))
	assert.Equal(t, "internal", jsonField(conv, "audience"))
	assert.NotEmpty(t, jsonField(conv, "title"))
	assertNilField(t, conv, "workflow_status")
	assertNilField(t, conv, "group")
	assert.Equal(t, "active", jsonField(conv, "status"))
	assert.Equal(t, "released", jsonField(conv, "legal_hold"))
	assertNilField(t, conv, "assignee")
	related := jsonObject(conv, "related")
	require.NotNil(t, related)
	assertNilField(t, related, "record")
	assertNilField(t, related, "continues")
	assert.NotContains(t, conv, "topic")
	assert.NotContains(t, conv, "unread")
	assert.Equal(t, "0", jsonField(conv, "unread_count"))
	assertNilField(t, conv, "last_message_at")
	assertNilField(t, conv, "last_message")
	assertValidTimestamp(t, jsonField(conv, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(conv, "updated_at"), "updated_at")

	// Participants: the owner (creator) plus the invited member, both with agent_trigger_policy/
	// agent_trigger_keywords nil (those only apply to agent participants).
	participants, ok := listData(conv, "participants")
	require.True(t, ok)
	require.Len(t, participants, 2)
	for _, raw := range participants {
		p, _ := raw.(map[string]any)
		require.NotNil(t, p)
		assertObjectField(t, p, "conversation_participant")
		assertIDFormat(t, jsonField(p, "id"), "cvpt")
		assert.Equal(t, "user", jsonField(p, "type"))
		assert.Equal(t, "active", jsonField(p, "membership"))
		assert.Equal(t, "unmuted", jsonField(p, "notifications"))
		assertNilField(t, p, "agent_trigger_policy")
		assertNilField(t, p, "agent_trigger_keywords")
		actor, _ := p["actor"].(map[string]any)
		require.NotNil(t, actor, "a user participant always carries an actor")
		assert.Equal(t, "user", jsonField(actor, "type"))
	}
}

// Anyone in a direct message can name it.
func TestCovMessagingConversations_Update_DirectMessageTitle(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	dm := createDM(t, user, SeedAccountUser2ID)
	title := uniqueName("dm-title")
	other := chatUser2Client(t)
	patch, err := other.PatchFull(conversationsPath+"/"+jsonField(dm, "id"), map[string]any{"title": title}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	assert.Equal(t, title, jsonField(parseJSON(patch.Body), "title"))
}

// ──────────────────────────────────────────────
// DELETE: only conversations with no other person
// ──────────────────────────────────────────────

func TestCovMessagingConversations_Delete(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	// A conversation with only the caller deletes, returning the stub; it is gone afterwards.
	self := parseJSON(mustPost(t, user, conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUserID}},
		"message":      map[string]any{"body": "note to self"},
	}))
	selfID := jsonField(self, "id")
	resp, err := user.DeleteFull(conversationsPath + "/" + selfID)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, map[string]any{"id": selfID, "object": "conversation", "deleted": true}, parseJSON(resp.Body))
	gone, err := user.GetFull(conversationsPath+"/"+selfID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, gone.StatusCode)

	// A conversation with another person is archived or left, never deleted.
	dm := createDM(t, user, SeedAccountUser2ID)
	status, body, err := user.Delete(conversationsPath + "/" + jsonField(dm, "id"))
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")

	// Someone who isn't in the conversation can't see it to delete it.
	other := chatUser2Client(t)
	mine := parseJSON(mustPost(t, user, conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUserID}},
	}))
	status, body, err = other.Delete(conversationsPath + "/" + jsonField(mine, "id"))
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

func TestCovMessagingConversations_Delete_LegalHold(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	self := parseJSON(mustPost(t, user, conversationsPath, map[string]any{
		"participants": []map[string]any{{"account_user_id": SeedAccountUserID}},
	}))
	id := jsonField(self, "id")
	admin := user
	hold, err := admin.PostFull(conversationsPath+"/"+id+"/actions/set-legal-hold", map[string]any{"legal_hold": "held"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, hold.StatusCode, hold.Body)

	status, body, err := user.Delete(conversationsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_in_use", "invalid_request_error")
}

// mustPost posts body with a fresh idempotency key and returns the 201 response body.
func mustPost(t *testing.T, c *Client, path string, body map[string]any) []byte {
	t.Helper()
	resp, err := c.PostFull(path, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return resp.Body
}

// ──────────────────────────────────────────────
// UpdateConversationRequest: Clearable title semantics + PATCH preserves other fields
// ──────────────────────────────────────────────

// A blank string ("") on a field.Clearable field is a real value ("set to empty string"), not a
// clear (which only happens on explicit JSON null) — see docs/patterns/nullable-field-patterns.md
// and the established precedent TestCovCatalogProducts_Update_BlankStringDoesNotClear. This is a
// shared/db.NullStringPtr collapses "" to SQL NULL, so the
// UpdateConversation query's `COALESCE(sqlc.narg('title'), title)` silently falls back to the OLD
// title instead of writing "". The endpoint returns 200 but the title is left unchanged, which is a
// silent no-op masquerading as success.
func TestCovMessagingConversations_Update_BlankTitleSetsEmptyString(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	original := uniqueName("cov-blank-title")
	conv := createGroupConversation(t, owner, original, SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	patch, err := owner.PatchFull(conversationsPath+"/"+convID, map[string]any{"title": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	updated := parseJSON(patch.Body)
	title, hasTitle := updated["title"]
	require.True(t, hasTitle, "title key should be present after a blank-string update")
	assert.Equal(t, "", title, "a blank-string PATCH should set title to \"\", not silently preserve the old value")
}

func TestCovMessagingConversations_Update_NullTitleClears(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-clear-title"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	patch, err := owner.PatchFull(conversationsPath+"/"+convID, map[string]any{"title": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	assertNilField(t, parseJSON(patch.Body), "title")
}

// A rename leaves every other field on the conversation untouched.
func TestCovMessagingConversations_Update_PreservesOtherFields(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-preserve"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	before := parseJSON(covMessagingConversationsMustGetFull(t, owner, conversationsPath+"/"+convID))

	newTitle := uniqueName("cov-preserve-renamed")
	patch, err := owner.PatchFull(conversationsPath+"/"+convID, map[string]any{"title": newTitle}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	after := parseJSON(patch.Body)

	assert.Equal(t, newTitle, jsonField(after, "title"))
	assert.Equal(t, jsonField(before, "type"), jsonField(after, "type"))
	assert.Equal(t, jsonField(before, "audience"), jsonField(after, "audience"))
	assert.Equal(t, jsonField(before, "status"), jsonField(after, "status"))
	assert.Equal(t, jsonField(before, "legal_hold"), jsonField(after, "legal_hold"))
	assert.Equal(t, jsonField(before, "created_at"), jsonField(after, "created_at"))
}

// covMessagingConversationsMustGetFull is a small helper that fails the test on transport error and returns the raw body,
// used where only the body (not status) needs inspecting for a pre-condition snapshot.
func covMessagingConversationsMustGetFull(t *testing.T, c *Client, path string) []byte {
	t.Helper()
	resp, err := c.GetFull(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	return resp.Body
}

// ──────────────────────────────────────────────
// Expandable fields not covered elsewhere: last_message (full), and its asymmetry vs. list
// ──────────────────────────────────────────────

// RetrieveConversation (GET /v1/messaging/conversations/{id}) never
// hydrates `last_message` even with ?include=last_message, unlike ListConversations which batch-
// hydrates it correctly (see services/notification-service/internal/service/conversation_service.go:
// GetConversation never sets conv.LastMessage, while the list path around ListConversations does via
// lastMessageIDs / byID). last_message_at updates correctly on both paths; only the nested
// last_message object is missing on the single-conversation GET.
func TestCovMessagingConversations_Expandable_LastMessagePopulatesOnRetrieve(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-lastmsg"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	body := uniqueName("cov last message body")
	msg := sendMessage(t, owner, convID, body, newIdempotencyKey())
	msgID := jsonField(msg, "id")

	resp, err := owner.GetFull(conversationsPath+"/"+convID, url.Values{"include": {"last_message", "last_message.sender"}})
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	got := parseJSON(resp.Body)

	// last_message_at is correctly populated on both paths.
	assertValidTimestamp(t, jsonField(got, "last_message_at"), "last_message_at")

	lastMessage := jsonObject(got, "last_message")
	require.NotNil(t, lastMessage, "GET .../{id}?include=last_message should hydrate last_message the same way the list endpoint does")
	assert.Equal(t, msgID, jsonField(lastMessage, "id"))
	assert.Equal(t, body, jsonField(lastMessage, "body"))
}

// The list endpoint, by contrast, hydrates last_message correctly today — pinning the working half
// of the asymmetry so a regression there is caught too.
func TestCovMessagingConversations_Expandable_LastMessagePopulatesOnList(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-lastmsg-list"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	body := uniqueName("cov last message list body")
	msg := sendMessage(t, owner, convID, body, newIdempotencyKey())
	msgID := jsonField(msg, "id")

	// listFindByField is not usable here: it hits the endpoint via the shared apiClient (an API key),
	// but chat requires an account-member session (see meta_includes_test.go's exclusion comment for
	// this route). Walk the owner's own list instead.
	list, _, err := owner.GetList(conversationsPath, url.Values{"include": {"last_message"}, "limit": {"100"}})
	require.NoError(t, err)
	var row map[string]any
	for _, raw := range list.Data {
		candidate := parseJSON(raw)
		if jsonField(candidate, "id") == convID {
			row = candidate
			break
		}
	}
	require.NotNil(t, row, "the conversation should appear in the list")
	lastMessage := jsonObject(row, "last_message")
	require.NotNil(t, lastMessage, "the list endpoint should hydrate last_message")
	assert.Equal(t, msgID, jsonField(lastMessage, "id"))
}

// ──────────────────────────────────────────────
// Idempotency-Key replay semantics (not business-level dedup)
// ──────────────────────────────────────────────

func TestCovMessagingConversations_Idempotency_CreateConversation(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	idemKey := newIdempotencyKey()
	title := uniqueName("cov-idem-create")

	first, err := user.PostFull(conversationsPath, map[string]any{
		"title":        title,
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, first.StatusCode, first.Body)
	id := jsonField(parseJSON(first.Body), "id")

	// Same key, same body -> the same resource.
	second, err := user.PostFull(conversationsPath, map[string]any{
		"title":        title,
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, second.StatusCode, second.Body)
	assert.Equal(t, id, jsonField(parseJSON(second.Body), "id"))

	// Same key, different body -> idempotency_error.
	status, body, err := user.Post(conversationsPath, map[string]any{
		"title":        uniqueName("cov-idem-create-DIFFERENT"),
		"participants": []map[string]any{{"account_user_id": SeedAccountUser2ID}},
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	requireErrorResponse(t, body, "", "idempotency_error")
}

// Idempotency replay pinned on an action endpoint too (not just create): AssignConversation, run
// against the shared support case (so no t.Parallel — see file header).
func TestCovMessagingConversations_Idempotency_AssignAction(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	caseID := openCustomerCase(t, customer)

	idemKey := newIdempotencyKey()
	first, err := dane.PostFull(covMessagingConversationsCaseAssignPath(caseID), map[string]any{
		"assignee_resource_type": "account_user",
		"assignee_resource_id":   SeedAccountUserID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, first.StatusCode, first.Body)
	id := jsonField(parseJSON(first.Body), "id")

	second, err := dane.PostFull(covMessagingConversationsCaseAssignPath(caseID), map[string]any{
		"assignee_resource_type": "account_user",
		"assignee_resource_id":   SeedAccountUserID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, second.StatusCode, second.Body)
	assert.Equal(t, id, jsonField(parseJSON(second.Body), "id"))

	status, body, err := dane.Post(covMessagingConversationsCaseAssignPath(caseID), map[string]any{
		"assignee_resource_type": "account_user",
		"assignee_resource_id":   SeedAccountUser2ID,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	requireErrorResponse(t, body, "", "idempotency_error")

	// Leave the shared case unassigned for other tests.
	_, _, _ = dane.Post(covMessagingConversationsCaseAssignPath(caseID), map[string]any{}, newIdempotencyKey())
}

func covMessagingConversationsCaseAssignPath(conversationID string) string {
	return conversationsPath + "/" + conversationID + "/actions/assign"
}

// ──────────────────────────────────────────────
// AssignConversationRequest: requires a customer case, team assignment, clear, loose typing
// ──────────────────────────────────────────────

func TestCovMessagingConversations_Assign_RequiresCustomerCase(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-assign-internal"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	status, body, err := owner.Post(covMessagingConversationsCaseAssignPath(convID), map[string]any{
		"assignee_resource_type": "account_user",
		"assignee_resource_id":   SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assert.Contains(t, string(body), "not a customer-facing case")
}

// Team assignment (assignee_resource_type=account_group), clearing an existing assignment by omitting both fields, and enum rejection of an unrecognized assignee_resource_type — all against the shared support case, so no t.Parallel (see file header).
func TestCovMessagingConversations_Assign_TeamClearAndInvalidType(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	caseID := openCustomerCase(t, customer)
	assignWithInclude := withQuery(covMessagingConversationsCaseAssignPath(caseID), url.Values{"include": {"assignee"}})

	// Assign to a team. The server does not verify assignee_resource_id refers to a real team (no
	// existence check, same "plain reference" pattern used for topic/link resources); SeedCustomerGroupID
	// is reused purely as a well-formed account_group id.
	teamResp, err := dane.PostFull(assignWithInclude, map[string]any{
		"assignee_resource_type": "account_group",
		"assignee_resource_id":   SeedCustomerGroupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, teamResp.StatusCode, teamResp.Body)
	teamAssignee := jsonObject(parseJSON(teamResp.Body), "assignee")
	require.NotNil(t, teamAssignee)
	assert.Equal(t, SeedCustomerGroupID, jsonField(teamAssignee, "id"))
	assert.Equal(t, "group", jsonField(teamAssignee, "type"), "an account_group assignee maps to a group actor")

	// Omitting both fields clears the assignment.
	clearResp, err := dane.PostFull(assignWithInclude, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearResp.StatusCode, clearResp.Body)
	assertNilField(t, parseJSON(clearResp.Body), "assignee")

	// assignee_resource_type is a constants.ConversationAssigneeType enum, so an unrecognized value is rejected by the generic enum validator rather than falling through to a default actor type.
	invalidResp, err := dane.PostFull(assignWithInclude, map[string]any{
		"assignee_resource_type": "widget",
		"assignee_resource_id":   "abc123",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, invalidResp.StatusCode, invalidResp.Body)
	invalidErr := requireErrorResponse(t, invalidResp.Body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, invalidErr, "assignee_resource_type")

	// Leave the shared case unassigned for other tests.
	_, _, _ = dane.Post(covMessagingConversationsCaseAssignPath(caseID), map[string]any{}, newIdempotencyKey())
}

// ──────────────────────────────────────────────
// SetWorkflowStatusRequest validation
// ──────────────────────────────────────────────

func TestCovMessagingConversations_SetWorkflowStatus_InvalidEnum(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-setstatus-invalid"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	status, body, err := owner.Post(conversationsPath+"/"+convID+"/actions/set-status",
		map[string]any{"workflow_status": "bogus_status"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "workflow_status")
}

func TestCovMessagingConversations_SetWorkflowStatus_MissingField(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-setstatus-missing"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	status, body, err := owner.Post(conversationsPath+"/"+convID+"/actions/set-status", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "workflow_status")
}

// ──────────────────────────────────────────────
// UpdateParticipantRoleRequest: ownership "transfer" and invalid enum
// ──────────────────────────────────────────────

// Setting a second participant's role to "owner" grants co-ownership; it does not demote the caller
// (the conversation ends up with two owners). Pinning this behavior since the design intent
// ("transfer" vs. "add a co-owner") was previously untested.
func TestCovMessagingConversations_SetRole_OwnerGrantsCoOwnership(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-owner-transfer"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)
	require.NotEmpty(t, pid)

	resp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/participants/"+pid+"/actions/set-role", participantIncludeQuery),
		map[string]any{"role": "owner"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	updated := parseJSON(resp.Body)

	_, callerRole, _ := participantInfo(t, updated, SeedAccountUserID)
	_, newOwnerRole, _ := participantInfo(t, updated, SeedAccountUser2ID)
	assert.Equal(t, "owner", newOwnerRole, "the promoted participant is now an owner")
	assert.Equal(t, "owner", callerRole, "the original owner is not demoted — set-role adds a co-owner")
}

func TestCovMessagingConversations_SetRole_InvalidEnum(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-setrole-invalid"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)
	require.NotEmpty(t, pid)

	status, body, err := owner.Post(conversationsPath+"/"+convID+"/participants/"+pid+"/actions/set-role",
		map[string]any{"role": "wizard"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "role")
}

// ──────────────────────────────────────────────
// Sole-owner guards: cannot leave, cannot self-remove via remove-participant
// ──────────────────────────────────────────────

func TestCovMessagingConversations_SoleOwnerCannotLeave(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-sole-owner-leave"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	status, body, err := owner.Post(conversationsPath+"/"+convID+"/actions/leave", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	assert.Contains(t, string(body), "Transfer ownership first")
}

func TestCovMessagingConversations_RemoveParticipant_SelfRejected(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-remove-self"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUserID)
	require.NotEmpty(t, pid)

	status, body, err := owner.Delete(conversationsPath + "/" + convID + "/participants/" + pid)
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	assert.Contains(t, string(body), "Use leave to remove yourself")
}

// ──────────────────────────────────────────────
// Unknown-id 404s: remove participant / remove agent / set-role / remove link
// ──────────────────────────────────────────────

func TestCovMessagingConversations_UnknownParticipantOrAgentID_404(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-unknown-ids"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	t.Run("remove-participant", func(t *testing.T) {
		status, body, err := owner.Delete(conversationsPath + "/" + convID + "/participants/cvpt_doesnotexist")
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("remove-agent", func(t *testing.T) {
		status, body, err := owner.Delete(conversationsPath + "/" + convID + "/agents/cvpt_doesnotexist")
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("set-role", func(t *testing.T) {
		status, body, err := owner.Post(conversationsPath+"/"+convID+"/participants/cvpt_doesnotexist/actions/set-role",
			map[string]any{"role": "admin"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})
}

func TestCovMessagingConversations_RemoveLink_UnknownID_404(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-unknown-link"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	status, body, err := owner.Delete(conversationsPath + "/" + convID + "/links/cvlk_doesnotexist")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// AddParticipantRequest / AddAgentParticipantRequest validation
// ──────────────────────────────────────────────

func TestCovMessagingConversations_AddParticipant_Validation(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-add-participant-validation"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	t.Run("missing account_user_id", func(t *testing.T) {
		status, body, err := owner.Post(conversationsPath+"/"+convID+"/participants", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "account_user_id")
	})

	t.Run("nonexistent account_user_id", func(t *testing.T) {
		status, body, err := owner.Post(conversationsPath+"/"+convID+"/participants",
			map[string]any{"account_user_id": "acus_doesnotexist000"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		assert.Contains(t, string(body), "does not exist")
	})
}

func TestCovMessagingConversations_AddAgent_Validation(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-add-agent-validation"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	t.Run("missing agent_config_id", func(t *testing.T) {
		status, body, err := owner.Post(conversationsPath+"/"+convID+"/agents", map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "agent_config_id")
	})

	t.Run("invalid trigger_policy", func(t *testing.T) {
		status, body, err := owner.Post(conversationsPath+"/"+convID+"/agents",
			map[string]any{"agent_config_id": SeedAgentConfigID, "trigger_policy": "bogus"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
		errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
		assertErrorParam(t, errObj, "trigger_policy")
	})

	// A nonexistent agent_config_id is accepted — no existence check, the same "plain reference"
	// pattern as topic/link resources and assignee ids.
	t.Run("nonexistent agent_config_id is a plain reference", func(t *testing.T) {
		resp, err := owner.PostFull(conversationsPath+"/"+convID+"/agents",
			map[string]any{"agent_config_id": "agcf_doesnotexist000"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, resp.StatusCode, resp.Body)
		p := parseJSON(resp.Body)
		assert.Equal(t, "agent", jsonField(p, "type"))
		assert.Equal(t, "mention", jsonField(p, "agent_trigger_policy"), "trigger_policy defaults to mention")
	})
}

// ──────────────────────────────────────────────
// AddConversationLinkRequest / ConversationLink fields
// ──────────────────────────────────────────────

func TestCovMessagingConversations_AddLink_InvalidResourceType(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-link-invalid-type"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	status, body, err := owner.Post(conversationsPath+"/"+convID+"/links",
		map[string]any{"record": map[string]any{"type": "not_a_type", "id": "x"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 422, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "record.type")
}

// AddConversationLink (POST .../links) returns created_at as the Go zero
// time ("0001-01-01T00:00:00Z") rather than the actual creation timestamp — the service builds the
// domain.ConversationLink in memory (services/notification-service/internal/service/
// conversation_case_service.go AddConversationLink) and returns it directly without ever setting
// CreatedAt or re-fetching the row after insert, even though the DB column defaults it and the
// subsequent ListConversationLinks call for the same link returns the correct value. The resource
// struct tags CreatedAt `validate:"required"`, so a zero time is a contract violation.
func TestCovMessagingConversations_AddLink_CreatedAtIsPopulated(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-link-created-at"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	resp, err := owner.PostFull(conversationsPath+"/"+convID+"/links",
		map[string]any{"record": map[string]any{"type": "sales_order", "id": SeedSalesOrderID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	link := parseJSON(resp.Body)
	assertValidTimestamp(t, jsonField(link, "created_at"), "created_at")
}

// `record` and `conversation` are both expandable: null unless included.
func TestCovMessagingConversations_Links_RecordAndConversationExpandable(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("cov-link-fields"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	addResp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/links", url.Values{"include": {"record"}}),
		map[string]any{"record": map[string]any{"type": "sales_order", "id": SeedSalesOrderID}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addResp.StatusCode, addResp.Body)
	added := parseJSON(addResp.Body)
	assertObjectField(t, added, "conversation_link")
	assertIDFormat(t, jsonField(added, "id"), "cvlk")
	record := jsonObject(added, "record")
	require.NotNil(t, record, "record is populated with ?include=record")
	assert.Equal(t, "record", jsonField(record, "object"))
	assert.Equal(t, SeedSalesOrderID, jsonField(record, "id"))
	assert.Equal(t, "sales_order", jsonField(record, "type"))
	assertNilField(t, added, "conversation")

	// List without include: record and conversation are null.
	noInclude, _, err := owner.GetList(conversationsPath+"/"+convID+"/links", nil)
	require.NoError(t, err)
	require.NotEmpty(t, noInclude.Data)
	row := parseJSON(noInclude.Data[0])
	assertNilField(t, row, "record")
	assertNilField(t, row, "conversation")

	// List with include=conversation: conversation now populated with the parent id.
	withInclude, _, err := owner.GetList(conversationsPath+"/"+convID+"/links", url.Values{"include": {"conversation"}})
	require.NoError(t, err)
	require.NotEmpty(t, withInclude.Data)
	rowWithInclude := parseJSON(withInclude.Data[0])
	conversation := jsonObject(rowWithInclude, "conversation")
	require.NotNil(t, conversation, "conversation should be populated with ?include=conversation")
	assert.Equal(t, convID, jsonField(conversation, "id"))
	assertObjectField(t, conversation, "conversation")
}

// ──────────────────────────────────────────────
// ListConversationsRequest: pagination, unknown query param, invalid enum filters, list filters
// ──────────────────────────────────────────────

func TestCovMessagingConversations_List_Pagination(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	createGroupConversation(t, user, uniqueName("cov-page-a"), SeedAccountUser2ID)
	createGroupConversation(t, user, uniqueName("cov-page-b"), SeedAccountUser2ID)

	const attempts = 3
	for attempt := 1; ; attempt++ {
		page1, _, err := user.GetList(conversationsPath, url.Values{"limit": {"1"}})
		require.NoError(t, err)
		if (len(page1.Data) != 1 || !page1.PageInfo.HasNextPage() || page1.PageInfo.NextPageURL == nil) && attempt < attempts {
			continue
		}
		require.Len(t, page1.Data, 1, "first page should hold exactly one conversation (after %d attempts)", attempt)
		require.True(t, page1.PageInfo.HasNextPage(), "the caller has more than one conversation")
		require.NotNil(t, page1.PageInfo.NextPageURL)

		page2, _, err := user.GetListFromPageURL(page1.PageInfo.NextPageURL)
		require.NoError(t, err)
		if len(page2.Data) == 0 && attempt < attempts {
			continue
		}
		require.Len(t, page2.Data, 1, "second page should hold exactly one conversation (after %d attempts)", attempt)
		assert.NotEqual(t, DataItemField(page1.Data[0], "id"), DataItemField(page2.Data[0], "id"),
			"consecutive pages should return different conversations")
		return
	}
}

func TestCovMessagingConversations_List_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.GetListRaw(conversationsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, conversationsPath, status, body)
}

func TestCovMessagingConversations_List_InvalidEnumFilters(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)

	cases := []struct {
		name      string
		params    url.Values
		wantParam string
	}{
		{"type", url.Values{"type": {"bogus"}}, "type"},
		{"audience", url.Values{"audience": {"bogus"}}, "audience"},
		{"memberships", url.Values{"memberships[]": {"bogus"}}, "memberships"},
		{"workflow_status", url.Values{"workflow_status": {"bogus"}}, "workflow_status"},
		{"record_type", url.Values{"record_type": {"bogus"}}, "record_type"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := user.GetListRaw(conversationsPath, tc.params)
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
			assertErrorParam(t, errObj, tc.wantParam)
		})
	}
}

// The assignee_resource_id and unassigned filters, run against the shared support case, so no
// t.Parallel (see file header).
func TestCovMessagingConversations_List_AssigneeAndUnassignedFilters(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	caseID := openCustomerCase(t, customer)

	assignResp, err := dane.PostFull(covMessagingConversationsCaseAssignPath(caseID), map[string]any{
		"assignee_resource_type": "account_user",
		"assignee_resource_id":   SeedAccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, assignResp.StatusCode, assignResp.Body)

	assert.True(t, listContainsConversation(t, dane, caseID, url.Values{"assignee_resource_id": {SeedAccountUserID}}),
		"assignee_resource_id should filter the inbox to cases owned by that user")

	unassigned, _, err := dane.GetList(conversationsPath, url.Values{"unassigned": {"true"}, "limit": {"100"}})
	require.NoError(t, err)
	var foundAssignedInUnassignedList bool
	for _, raw := range unassigned.Data {
		if jsonField(parseJSON(raw), "id") == caseID {
			foundAssignedInUnassignedList = true
		}
	}
	assert.False(t, foundAssignedInUnassignedList, "an assigned case must not appear in the unassigned filter")

	// Leave the shared case unassigned for other tests.
	_, _, _ = dane.Post(covMessagingConversationsCaseAssignPath(caseID), map[string]any{}, newIdempotencyKey())
}

// include_archived is accepted as a query param without error (it only takes effect in the support
// inbox branch alongside another inbox filter).
func TestCovMessagingConversations_List_IncludeArchivedAccepted(t *testing.T) {
	t.Parallel()
	user := chatUserClient(t)
	status, body, err := user.GetListRaw(conversationsPath, url.Values{"include_archived": {"true"}, "workflow_status": {"new"}, "limit": {"1"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}
