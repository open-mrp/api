//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Editing a scheduled message is one request under messaging:update: the reschedule action moves the message to a new time and can revise its body, and the message is still the one the worker sends.

func reschedulePath(messageID string) string {
	return "/v1/messaging/messages/" + messageID + "/actions/reschedule"
}

// scheduleIn schedules body in a conversation, d from now, and returns the message id.
func scheduleIn(t *testing.T, c *Client, conversationID, body string, d time.Duration) string {
	t.Helper()
	resp, err := c.PostFull(conversationsPath+"/"+conversationID+"/messages", map[string]any{
		"body":              body,
		"client_message_id": uniqueName("cmid"),
		"scheduled_at":      time.Now().Add(d).UTC().Format(time.RFC3339),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, resp.StatusCode, resp.Body)
	return jsonField(parseJSON(resp.Body), "id")
}

// threadBodies counts the sent messages in a conversation by body, read by a participant who may read it.
func threadBodies(t *testing.T, reader *Client, conversationID string) map[string]int {
	t.Helper()
	msgs, _, err := reader.GetList(conversationsPath+"/"+conversationID+"/messages", url.Values{"limit": {"50"}})
	require.NoError(t, err)
	out := map[string]int{}
	for _, raw := range msgs.Data {
		m := parseJSON(raw)
		if jsonField(m, "status") == "sent" {
			out[jsonField(m, "body")]++
		}
	}
	return out
}

func TestScheduledMessages_RescheduleUnderMessagingUpdateAlone(t *testing.T) {
	t.Parallel()
	author := newChatRoleUser(t, "messaging:create", "messaging:update")
	reader := chatUserClient(t)
	convID := jsonField(createDM(t, author.client, SeedAccountUserID), "id")
	oldBody, newBody := uniqueName("before the edit"), uniqueName("after the edit")
	messageID := scheduleIn(t, author.client, convID, oldBody, time.Hour)

	// The author keeps only messaging:update, so editing cannot lean on messaging:create.
	status, body, err := apiClient.Patch(rolesPath+"/"+author.roleID, map[string]any{"permissions": []string{"messaging:update"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	editor := loginAsUser(t, author.email, covAuthUsersPassword, SeedAccountID)
	t.Cleanup(func() { _, _, _ = editor.Post("/v1/messaging/messages/"+messageID+"/actions/cancel", map[string]any{}, newIdempotencyKey()) })
	status, body, err = editor.Post(conversationsPath+"/"+convID+"/messages", map[string]any{"body": "x", "client_message_id": uniqueName("cmid")}, newIdempotencyKey())
	require.NoError(t, err)
	requirePermissionRefused(t, status, body, "messaging:create")

	at := time.Now().Add(5 * time.Second).UTC().Truncate(time.Second)
	key := newIdempotencyKey()
	status, body, err = editor.Post(reschedulePath(messageID), map[string]any{"body": newBody, "scheduled_at": at.Format(time.RFC3339)}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	moved := parseJSON(body)
	assert.Equal(t, messageID, jsonField(moved, "id"), "the edit keeps the message")
	assert.Equal(t, "chat_message", jsonField(moved, "object"))
	assert.Equal(t, "scheduled", jsonField(moved, "status"))
	assert.Equal(t, newBody, jsonField(moved, "body"))
	scheduledAt, err := time.Parse(time.RFC3339Nano, jsonField(moved, "scheduled_at"))
	require.NoError(t, err)
	assert.True(t, at.Equal(scheduledAt), "scheduled_at is the new time: got %s, want %s", scheduledAt, at)

	status, body, err = editor.Post(reschedulePath(messageID), map[string]any{"body": newBody, "scheduled_at": at.Format(time.RFC3339)}, key)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, messageID, jsonField(parseJSON(body), "id"), "a replay answers with the same message")

	// It is sent once, with the edited text; the text before the edit never goes out.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if threadBodies(t, reader, convID)[newBody] == 0 {
			return fmt.Errorf("the rescheduled message has not been sent yet")
		}
		return nil
	})
	sent := threadBodies(t, reader, convID)
	assert.Equal(t, 1, sent[newBody], "sent exactly once")
	assert.Zero(t, sent[oldBody], "the text before the edit is never sent")

	// Once sent it can no longer be edited.
	status, body, err = editor.Post(reschedulePath(messageID), map[string]any{"scheduled_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusBadRequest, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

// Moving only the time keeps the body; the message stays the caller's one scheduled message.
func TestScheduledMessages_RescheduleTimeOnlyKeepsTheBody(t *testing.T) {
	t.Parallel()
	author := chatUserClient(t)
	convID := jsonField(createDM(t, author, SeedAccountUser2ID), "id")
	text := uniqueName("only the time moves")
	messageID := scheduleIn(t, author, convID, text, time.Hour)
	t.Cleanup(func() { _, _, _ = author.Post("/v1/messaging/messages/"+messageID+"/actions/cancel", map[string]any{}, newIdempotencyKey()) })

	at := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	status, body, err := author.Post(reschedulePath(messageID), map[string]any{"scheduled_at": at.Format(time.RFC3339)}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	assert.Equal(t, text, jsonField(parseJSON(body), "body"))

	list, _, err := author.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"status": {"scheduled"}})
	require.NoError(t, err)
	var found int
	for _, raw := range list.Data {
		m := parseJSON(raw)
		if jsonField(m, "id") == messageID {
			found++
			scheduledAt, err := time.Parse(time.RFC3339Nano, jsonField(m, "scheduled_at"))
			require.NoError(t, err)
			assert.True(t, at.Equal(scheduledAt), "listed at the new time")
		}
	}
	assert.Equal(t, 1, found, "still one scheduled message, at the new time")
}

func TestScheduledMessages_RescheduleRefusals(t *testing.T) {
	t.Parallel()
	author := chatUserClient(t)
	convID := jsonField(createDM(t, author, SeedAccountUser2ID), "id")
	messageID := scheduleIn(t, author, convID, uniqueName("refusals"), time.Hour)
	t.Cleanup(func() { _, _, _ = author.Post("/v1/messaging/messages/"+messageID+"/actions/cancel", map[string]any{}, newIdempotencyKey()) })
	canceledID := scheduleIn(t, author, convID, uniqueName("canceled"), time.Hour)
	status, body, err := author.Post("/v1/messaging/messages/"+canceledID+"/actions/cancel", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)

	future := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	creator, _ := customRoleChatUser(t, "messaging:create")
	stranger, _ := customRoleChatUser(t, "messaging:update")

	cases := []struct {
		name   string
		client *Client
		id     string
		body   map[string]any
		status int
		code   string
		param  string
	}{
		{"a role holding messaging:create alone", creator, messageID, map[string]any{"scheduled_at": future}, http.StatusForbidden, "insufficient_permissions", ""},
		{"someone else's scheduled message", stranger, messageID, map[string]any{"scheduled_at": future}, http.StatusNotFound, "resource_not_found", ""},
		{"a message that does not exist", author, "mg_01e2edoesnotexist00", map[string]any{"scheduled_at": future}, http.StatusNotFound, "resource_not_found", ""},
		{"a canceled message", author, canceledID, map[string]any{"scheduled_at": future}, http.StatusBadRequest, "validation_failed", ""},
		{"a time that has passed", author, messageID, map[string]any{"scheduled_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}, http.StatusBadRequest, "parameter_invalid", "scheduled_at"},
		{"no time", author, messageID, map[string]any{"body": "x"}, http.StatusBadRequest, "", "scheduled_at"},
		{"a blank body", author, messageID, map[string]any{"scheduled_at": future, "body": ""}, http.StatusBadRequest, "invalid_format", "body"},
	}
	for _, tc := range cases {
		status, body, err := tc.client.Post(reschedulePath(tc.id), tc.body, newIdempotencyKey())
		require.NoError(t, err)
		t.Run(tc.name, func(t *testing.T) {
			requireStatus(t, tc.status, status, body)
			errObj, _ := parseJSON(body)["error"].(map[string]any)
			require.NotNil(t, errObj, string(body))
			if tc.code != "" {
				assert.Equal(t, tc.code, errObj["code"])
			}
			if tc.param != "" {
				assertErrorParam(t, errObj, tc.param)
			}
			if tc.status == http.StatusForbidden {
				assert.Contains(t, errObj["message"], "messaging:update")
			}
		})
	}

	list, _, err := author.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"status": {"scheduled"}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		m := parseJSON(raw)
		if jsonField(m, "id") == messageID {
			assert.NotEqual(t, "x", jsonField(m, "body"), "no refused request changed the message")
		}
	}
}
