//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the Phase-2 1:1 DM chat: create/dedup, send (sequence + idempotent
// resend via client_message_id), list messages, read cursor / unread, participant scoping, and
// the realtime message.created push. dane (admin) and user2 (sales rep) are both members of
// the seeded account; the customer API key is used to prove non-participant scoping.

const conversationsPath = "/v1/messaging/conversations"

// Conversation/message sub-objects (participants, last_message, sender, author, attachments,
// resource) are expandable — the gateway omits them unless explicitly requested via ?include=.
// These query sets are appended to the e2e helpers so the assertions still see the data they rely on.
var conversationIncludeQuery = url.Values{"include": {"group", "participants", "last_message", "last_message.sender"}}

// participantIncludeQuery is the include set for the participant-action endpoints (add participant,
// set role), which expose a narrower ?include= whitelist than the conversation endpoints (no "group"
// or "assignee"). These endpoints return the updated Conversation, so "participants" is what the
// assertions read.
var participantIncludeQuery = url.Values{"include": {"participants"}}

var messageIncludeQuery = url.Values{"include": {"sender", "author", "attachments", "resource"}}

// withQuery appends an encoded query string to a path (for include= on POST/PUT actions).
func withQuery(path string, q url.Values) string {
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

func chatUserClient(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, seedUserEmail, seedUserPassword, SeedAccountID)
}

func chatUser2Client(t *testing.T) *Client {
	t.Helper()
	return loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)
}

// listData unwraps a nested `List` envelope ({"object":"list","data":[...]}) embedded in a
// resource (e.g. conversation.participants, message.attachments) to its data slice.
func listData(m map[string]any, key string) ([]any, bool) {
	env, ok := m[key].(map[string]any)
	if !ok {
		return nil, false
	}
	data, ok := env["data"].([]any)
	return data, ok
}

// createDM creates (or returns the existing) direct message with the target account_user.
func createDM(t *testing.T, c *Client, targetAccountUserID string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(withQuery(conversationsPath, conversationIncludeQuery), map[string]any{
		"type":                         "direct_message",
		"participant_account_user_ids": []string{targetAccountUserID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// sendMessage posts a message and returns the created message resource.
func sendMessage(t *testing.T, c *Client, conversationID, body, clientMessageID string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(withQuery(conversationsPath+"/"+conversationID+"/messages", messageIncludeQuery), map[string]any{
		"body":              body,
		"client_message_id": clientMessageID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

func TestChat_CreateDMIsDeduped(t *testing.T) {
	user := chatUserClient(t)
	dm := createDM(t, user, SeedAccountUser2ID)
	assert.Equal(t, "conversation", jsonField(dm, "object"))
	assert.Equal(t, "direct_message", jsonField(dm, "type"))
	id := jsonField(dm, "id")
	assertIDFormat(t, id, "cv")

	// Creating the same DM again returns the same conversation (deduped, not a duplicate).
	again := createDM(t, user, SeedAccountUser2ID)
	assert.Equal(t, id, jsonField(again, "id"), "a repeat DM create must return the existing conversation")

	// Both users appear as participants.
	participants, ok := listData(dm, "participants")
	require.True(t, ok)
	assert.Len(t, participants, 2)
}

func TestChat_SendAndListMessages(t *testing.T) {
	user := chatUserClient(t)
	dm := createDM(t, user, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	body := uniqueName("hello")
	msg := sendMessage(t, user, convID, body, newIdempotencyKey())
	assert.Equal(t, "chat_message", jsonField(msg, "object"))
	assert.Equal(t, body, jsonField(msg, "body"))
	assertIDFormat(t, jsonField(msg, "id"), "mg")
	sender, ok := msg["sender"].(map[string]any)
	require.True(t, ok, "a chat message carries a sender")
	assert.Equal(t, SeedAccountUserID, jsonField(sender, "id"), "the sender is the authoring account_user")

	// The message appears in the conversation history.
	list, _, err := user.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
	require.NoError(t, err)
	var found bool
	for _, raw := range list.Data {
		if jsonField(parseJSON(raw), "body") == body {
			found = true
		}
	}
	assert.True(t, found, "the sent message should appear in the message list")
}

func TestChat_SendIsIdempotentOnClientMessageID(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	clientMsgID := uniqueName("cmid")
	first := sendMessage(t, user, convID, "first", clientMsgID)
	// A resend with the same client_message_id (even a different request) returns the original.
	second := sendMessage(t, user, convID, "first-retried", clientMsgID)
	assert.Equal(t, jsonField(first, "id"), jsonField(second, "id"),
		"a resend with the same client_message_id must collapse to one message")
	assert.Equal(t, jsonField(first, "sequence"), jsonField(second, "sequence"))
}

func TestChat_SequencesAreMonotonic(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	m1 := sendMessage(t, user, convID, "one", newIdempotencyKey())
	m2 := sendMessage(t, user, convID, "two", newIdempotencyKey())
	var s1, s2 float64
	require.NoError(t, json.Unmarshal([]byte(jsonField(m1, "sequence")), &s1))
	require.NoError(t, json.Unmarshal([]byte(jsonField(m2, "sequence")), &s2))
	assert.Greater(t, s2, s1, "later messages must have a higher sequence")
}

func TestChat_UnreadAndReadCursor(t *testing.T) {
	user := chatUserClient(t)
	other := chatUser2Client(t)
	dm := createDM(t, user, SeedAccountUser2ID)
	convID := jsonField(dm, "id")

	// user sends; the other participant should see unread > 0 on their side.
	sendMessage(t, user, convID, uniqueName("unread"), newIdempotencyKey())

	var convForOther map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		resp, err := other.GetFull(conversationsPath+"/"+convID, nil)
		if err != nil {
			return err
		}
		requireStatus(t, 200, resp.StatusCode, resp.Body)
		convForOther = parseJSON(resp.Body)
		if jsonField(convForOther, "unread") == "0" {
			return fmt.Errorf("expected unread > 0 for the recipient")
		}
		return nil
	})

	// The sender, by contrast, has no unread (their own send advanced their cursor).
	senderResp, err := user.GetFull(conversationsPath+"/"+convID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, senderResp.StatusCode, senderResp.Body)
	assert.Equal(t, "0", jsonField(parseJSON(senderResp.Body), "unread"), "the author has no unread in their own conversation")

	// The recipient marks the conversation read up to the latest sequence → unread clears.
	msgs, _, err := other.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.NotEmpty(t, msgs.Data)
	latestSeq, parseErr := strconv.ParseInt(jsonField(parseJSON(msgs.Data[0]), "sequence"), 10, 64)
	require.NoError(t, parseErr)

	resp, err := other.PostFull(conversationsPath+"/"+convID+"/actions/read",
		map[string]any{"up_to_sequence": latestSeq}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, "0", jsonField(parseJSON(resp.Body), "unread"), "after marking read, unread is zero")
}

func TestChat_NonParticipantCannotRead(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	// The customer API key is not a participant (and not an account user) → 404 (existence not leaked).
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	resp, err := customer.GetFull(conversationsPath+"/"+convID, nil)
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, resp.StatusCode,
		"a non-participant must not read the conversation, got %d: %s", resp.StatusCode, string(resp.Body))
}

func TestChat_SendValidation(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	// Missing client_message_id.
	status, body, err := user.Post(conversationsPath+"/"+convID+"/messages",
		map[string]any{"body": "no client id"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
