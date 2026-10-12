//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Audit coverage for notification-service mutations: email bridge configuration, reply drafts,
// scheduled messages, a participant's own view of a conversation, broadcasts, and preferences.

// auditWindowStart is the starts_at bound that keeps an earlier run's events on the same resource out of a match.
func auditWindowStart() string {
	return time.Now().Add(-5 * time.Second).UTC().Format(time.RFC3339)
}

// findAuditEvent polls the caller's audit log for an event on the resource that satisfies match.
func findAuditEvent(t *testing.T, c *Client, since, resourceID, resourceType, action string, match func(event map[string]any) bool) map[string]any {
	t.Helper()
	var found map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := c.GetListRaw(auditEventsPath, url.Values{
			"resource_ids":   {resourceID},
			"resource_types": {resourceType},
			"actions":        {action},
			"starts_at":      {since},
			"include":        {"changes", "metadata"},
			"limit":          {"100"},
		})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("audit events: %d %s", status, body)
		}
		for _, item := range jsonArray(parseJSON(body), "data") {
			event, _ := item.(map[string]any)
			if event != nil && match(event) {
				found = event
				return nil
			}
		}
		return fmt.Errorf("no matching %s %s audit event yet for %s", resourceType, action, resourceID)
	})
	return found
}

func anyEvent(map[string]any) bool { return true }

// changedTo matches an event whose changes move field to the given value.
func changedTo(field, newValue string) func(map[string]any) bool {
	return func(event map[string]any) bool {
		change, ok := changeForField(jsonListData(event, "changes"), field)
		return ok && jsonField(change, "new_value") == newValue
	}
}

// changed matches an event whose changes record field at all.
func changed(field string) func(map[string]any) bool {
	return func(event map[string]any) bool {
		_, ok := changeForField(jsonListData(event, "changes"), field)
		return ok
	}
}

// --- Email bridge ---

func TestEmailDomains_AuditEvents(t *testing.T) {
	t.Parallel()
	since := auditWindowStart()
	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-audit")
	id := jsonField(covMessagingEmailDomainsCreate(t, domain), "id")

	created := findAuditEvent(t, apiClient, since, id, "email_domain", "create", changedTo("domain", domain))
	change, ok := changeForField(jsonListData(created, "changes"), "status")
	require.True(t, ok, "the create event records the status")
	assert.Equal(t, "pending", jsonField(change, "new_value"))

	status, body, err := apiClient.Post(covMessagingEmailDomainsPath+"/"+id+"/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	verified := findAuditEvent(t, apiClient, since, id, "email_domain", "update", changedTo("status", "verified"))
	change, _ = changeForField(jsonListData(verified, "changes"), "status")
	assert.Equal(t, "pending", jsonField(change, "old_value"))

	status, body, err = apiClient.Delete(covMessagingEmailDomainsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	findAuditEvent(t, apiClient, since, id, "email_domain", "delete", anyEvent)
}

func TestEmailInboxes_AuditEvents(t *testing.T) {
	t.Parallel()
	since := auditWindowStart()
	localPart := uniqueName("e2e-eminb-audit")
	status, body, err := apiClient.Post(covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(localPart), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	findAuditEvent(t, apiClient, since, id, "email_inbox", "create", changedTo("address", localPart+"@mail.e2e.openmrp.ai"))

	status, body, err = apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, map[string]any{"status": "disabled"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := findAuditEvent(t, apiClient, since, id, "email_inbox", "update", changedTo("status", "disabled"))
	change, _ := changeForField(jsonListData(updated, "changes"), "status")
	assert.Equal(t, "active", jsonField(change, "old_value"))

	status, body, err = apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	findAuditEvent(t, apiClient, since, id, "email_inbox", "delete", anyEvent)
}

const emailSenderPath = "/v1/messaging/email-sender"

// The sender is account-wide, so this runs on an account of its own rather than reconfiguring the seed account's mail.
func TestEmailSender_AuditEvents(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	owner := acct.owner
	since := auditWindowStart()

	newVerifiedDomain := func() string {
		status, body, err := owner.Post(covMessagingEmailDomainsPath, map[string]any{
			"domain": covMessagingEmailDomainsUniqueDomain("e2e-sender-audit"),
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		id := jsonField(parseJSON(body), "id")
		status, body, err = owner.Post(covMessagingEmailDomainsPath+"/"+id+"/actions/verify", nil, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		return id
	}
	setSender := func(domainID, localPart string) string {
		status, body, err := owner.Put(emailSenderPath, map[string]any{"email_domain_id": domainID, "local_part": localPart})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		return jsonField(parseJSON(body), "id")
	}

	domainID := newVerifiedDomain()
	senderID := setSender(domainID, "orders")
	findAuditEvent(t, owner, since, senderID, "email_sender", "create", changedTo("local_part", "orders"))

	// Setting it again replaces the same sender.
	require.Equal(t, senderID, setSender(domainID, "billing"))
	replaced := findAuditEvent(t, owner, since, senderID, "email_sender", "update", changedTo("local_part", "billing"))
	change, _ := changeForField(jsonListData(replaced, "changes"), "local_part")
	assert.Equal(t, "orders", jsonField(change, "old_value"))

	status, body, err := owner.Delete(emailSenderPath)
	require.NoError(t, err)
	requireStatus(t, 204, status, body)
	findAuditEvent(t, owner, since, senderID, "email_sender", "delete", anyEvent)

	// Deleting a domain also removes the sender bound to it, and both deletions are recorded.
	boundID := setSender(domainID, "orders")
	findAuditEvent(t, owner, since, boundID, "email_sender", "create", anyEvent)
	status, body, err = owner.Delete(covMessagingEmailDomainsPath + "/" + domainID)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	findAuditEvent(t, owner, since, domainID, "email_domain", "delete", anyEvent)
	findAuditEvent(t, owner, since, boundID, "email_sender", "delete", anyEvent)
}

// --- Reply drafts ---

// Not parallel: the draft lifecycle moves the shared per-customer support case between triage lanes,
// and the lane change it records is asserted here.
func TestReplyDrafts_AuditEvents(t *testing.T) {
	seedSupportRoute(t)
	customer := getCustomerPortalClient()
	dane := chatUserClient(t)
	convID := openCustomerCase(t, customer)
	since := auditWindowStart()

	draftID := jsonField(covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("draft v1"), nil), "id")
	created := findAuditEvent(t, apiClient, since, draftID, "reply_draft", "create", changedTo("status", "draft"))
	change, _ := changeForField(jsonListData(created, "changes"), "conversation_id")
	assert.Equal(t, convID, jsonField(change, "new_value"))
	_, hasBody := changeForField(jsonListData(created, "changes"), "body")
	assert.False(t, hasBody, "draft content stays out of the audit trail so redaction can clear it")

	// Editing records which content fields changed without their values.
	status, body, err := dane.Patch("/v1/messaging/messages/"+draftID, map[string]any{"body": uniqueName("draft v2")}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	edited := findAuditEvent(t, apiClient, since, draftID, "reply_draft", "update", func(event map[string]any) bool {
		metadata, _ := event["metadata"].(map[string]any)
		fields, _ := metadata["edited_fields"].([]any)
		return len(fields) == 1 && fields[0] == "body"
	})
	assert.Empty(t, jsonListData(edited, "changes"), "an edit records no field values")

	status, body, err = dane.Post("/v1/messaging/messages/"+draftID+"/actions/reject", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	rejected := findAuditEvent(t, apiClient, since, draftID, "reply_draft", "update", changedTo("status", "rejected"))
	change, _ = changeForField(jsonListData(rejected, "changes"), "status")
	assert.Equal(t, "draft", jsonField(change, "old_value"))

	// Rejecting the open draft hands the case back to the team, and that lane change is recorded too.
	findAuditEvent(t, apiClient, since, convID, "conversation", "update", func(event map[string]any) bool {
		change, ok := changeForField(jsonListData(event, "changes"), "workflow_status")
		return ok && jsonField(change, "old_value") == "needs_approval" && jsonField(change, "new_value") == "waiting_internal"
	})

	approvedID := jsonField(covMessagingMessagesCreateDraft(t, dane, convID, uniqueName("approve me"), nil), "id")
	status, body, err = dane.Post("/v1/messaging/messages/"+approvedID+"/actions/approve-send", map[string]any{
		"client_message_id": uniqueName("cmid"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	approved := findAuditEvent(t, apiClient, since, approvedID, "reply_draft", "update", changedTo("status", "sent"))
	change, _ = changeForField(jsonListData(approved, "changes"), "visibility")
	assert.Equal(t, "external", jsonField(change, "new_value"))
	_, ok := changeForField(jsonListData(approved, "changes"), "approved_by_account_user_id")
	assert.True(t, ok, "the approval records who approved it")
}

// --- Chat messages ---

func auditEventIDsForRoot(t *testing.T, c *Client, since, rootType, rootID, resourceType string) []string {
	t.Helper()
	status, body, err := c.GetListRaw(auditEventsPath, url.Values{
		"root_resource_type": {rootType},
		"root_resource_id":   {rootID},
		"resource_types":     {resourceType},
		"starts_at":          {since},
		"limit":              {"100"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var ids []string
	for _, item := range jsonArray(parseJSON(body), "data") {
		if event, ok := item.(map[string]any); ok {
			ids = append(ids, jsonField(event, "resource_id"))
		}
	}
	return ids
}

// A message on a conversation about a sales order is audited under the conversation, never the order, and without its text.
func TestChatMessages_AuditEvents(t *testing.T) {
	t.Parallel()
	since := auditWindowStart()
	user := chatUserClient(t)

	resp, err := user.PostFull(conversationsPath, map[string]any{
		"type":                         "group",
		"participant_account_user_ids": []string{SeedAccountUser2ID},
		"topic_resource_type":          "sales_order",
		"topic_resource_id":            SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	convID := jsonField(parseJSON(resp.Body), "id")

	text := uniqueName("audited message")
	clientMessageID := uniqueName("cmid")
	status, body, err := user.Post(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":               text,
		"client_message_id":  clientMessageID,
		"link_resource_type": "sales_order",
		"link_resource_id":   SeedSalesOrderID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	messageID := jsonField(parseJSON(body), "id")
	assertIDFormat(t, messageID, id.MessageIDPrefix)

	created := findAuditEvent(t, apiClient, since, messageID, "chat_message", "create", changedTo("conversation_id", convID))
	change, _ := changeForField(jsonListData(created, "changes"), "status")
	assert.Equal(t, "sent", jsonField(change, "new_value"))
	change, _ = changeForField(jsonListData(created, "changes"), "visibility")
	assert.Equal(t, "internal", jsonField(change, "new_value"))
	_, hasBody := changeForField(jsonListData(created, "changes"), "body")
	assert.False(t, hasBody, "message content stays out of the audit trail so redaction can clear it")
	raw, err := json.Marshal(created)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), text)

	status, body, err = user.Post(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              text,
		"client_message_id": clientMessageID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	require.Equal(t, messageID, jsonField(parseJSON(body), "id"))

	assert.Equal(t, []string{messageID}, auditEventIDsForRoot(t, apiClient, since, "conversation", convID, "chat_message"))
	assert.NotContains(t, auditEventIDsForRoot(t, apiClient, since, "sales_order", SeedSalesOrderID, "chat_message"), messageID)
}

// --- Scheduled messages ---

func TestScheduledMessages_AuditEvents(t *testing.T) {
	t.Parallel()
	dane := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, dane, uniqueName("sched-audit"), SeedAccountUser2ID), "id")
	since := auditWindowStart()

	id := jsonField(covMessagingMessagesScheduleMessage(t, dane, convID), "id")
	findAuditEvent(t, apiClient, since, id, "scheduled_message", "create", changedTo("status", "scheduled"))

	at := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)
	status, body, err := dane.Post("/v1/messaging/messages/"+id+"/actions/reschedule", map[string]any{
		"scheduled_at": at.Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	findAuditEvent(t, apiClient, since, id, "scheduled_message", "update", changed("scheduled_for"))

	status, body, err = dane.Post("/v1/messaging/messages/"+id+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	canceled := findAuditEvent(t, apiClient, since, id, "scheduled_message", "update", changedTo("status", "canceled"))
	change, _ := changeForField(jsonListData(canceled, "changes"), "status")
	assert.Equal(t, "scheduled", jsonField(change, "old_value"))
}

// --- A participant's own view of a conversation ---

func TestConversationParticipant_HideAndMuteAuditEvents(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	member := chatUser2Client(t)
	conv := createGroupConversation(t, owner, uniqueName("view-audit"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	memberPID, _, _ := participantInfo(t, conv, SeedAccountUser2ID)
	since := auditWindowStart()

	post := func(action string) {
		resp, err := member.PostFull(conversationsPath+"/"+convID+"/actions/"+action, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, resp.StatusCode, resp.Body)
	}

	post("hide")
	hidden := findAuditEvent(t, apiClient, since, memberPID, "conversation_participant", "update", changed("hidden_at"))
	change, _ := changeForField(jsonListData(hidden, "changes"), "hidden_at")
	assert.Equal(t, "", jsonField(change, "old_value"), "the participant was visible before hiding")

	post("unhide")
	findAuditEvent(t, apiClient, since, memberPID, "conversation_participant", "update", func(event map[string]any) bool {
		change, ok := changeForField(jsonListData(event, "changes"), "hidden_at")
		return ok && jsonField(change, "new_value") == ""
	})

	post("mute")
	muted := findAuditEvent(t, apiClient, since, memberPID, "conversation_participant", "update", changedTo("notifications", "muted"))
	change, _ = changeForField(jsonListData(muted, "changes"), "notifications")
	assert.Equal(t, "unmuted", jsonField(change, "old_value"))

	post("unmute")
	findAuditEvent(t, apiClient, since, memberPID, "conversation_participant", "update", changedTo("notifications", "unmuted"))
}

// --- Broadcasts and preferences ---

func TestAnnouncements_AuditEvent(t *testing.T) {
	t.Parallel()
	user := notifUserClient(t)
	since := auditWindowStart()
	title := uniqueName("e2e-announce-audit")
	broadcastAnnouncement(t, user, title)

	id := jsonField(findAnnouncement(t, user, title), "id")
	created := findAuditEvent(t, apiClient, since, id, "announcement", "create", changedTo("title", title))
	change, _ := changeForField(jsonListData(created, "changes"), "scope")
	assert.Equal(t, "account", jsonField(change, "new_value"))
}

// Preferences are per user, so this runs on an account of its own rather than rewriting a seed user's.
func TestNotificationPreferences_AuditEvents(t *testing.T) {
	t.Parallel()
	acct := registerE2EAccount(t)
	since := auditWindowStart()

	created := upsertPreference(t, acct.owner, map[string]any{
		"category":       "chat.mention",
		"in_app_enabled": true,
		"email_enabled":  false,
		"digest":         "instant",
	})
	id := jsonField(created, "id")
	findAuditEvent(t, acct.owner, since, id, "notification_preference", "create", changedTo("category", "chat.mention"))

	upsertPreference(t, acct.owner, map[string]any{
		"category":       "chat.mention",
		"in_app_enabled": true,
		"email_enabled":  true,
		"digest":         "instant",
	})
	updated := findAuditEvent(t, acct.owner, since, id, "notification_preference", "update", changedTo("email_enabled", "true"))
	change, _ := changeForField(jsonListData(updated, "changes"), "email_enabled")
	assert.Equal(t, "false", jsonField(change, "old_value"))
	_, ok := changeForField(jsonListData(updated, "changes"), "digest")
	assert.False(t, ok, "an unchanged field is not recorded")
}
