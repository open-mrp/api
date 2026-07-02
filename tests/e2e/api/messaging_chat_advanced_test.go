//go:build e2e

package api_test

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These extend the Phase-2 chat coverage: DM dedup symmetry, conversation listing for both
// participants, bidirectional replies, message pagination + reconnect catch-up, forward-only
// read cursor, and not-found handling.

// listContainsConversation reports whether the caller's conversation list contains the id.
func listContainsConversation(t *testing.T, c *Client, conversationID string, params url.Values) bool {
	t.Helper()
	merged := url.Values{"limit": {"100"}}
	for k, v := range params {
		merged[k] = v
	}
	list, _, err := c.GetList(conversationsPath, merged)
	require.NoError(t, err)
	for _, raw := range list.Data {
		if jsonField(parseJSON(raw), "id") == conversationID {
			return true
		}
	}
	return false
}

func TestChat_DMDedupIsOrderIndependent(t *testing.T) {
	user := chatUserClient(t)   // dane (SeedAccountUserID)
	other := chatUser2Client(t) // user2 (SeedAccountUser2ID)

	fromUser := jsonField(createDM(t, user, SeedAccountUser2ID), "id")
	fromOther := jsonField(createDM(t, other, SeedAccountUserID), "id")

	assert.Equal(t, fromUser, fromOther,
		"a DM is the same conversation regardless of which participant creates it (sorted dm_key)")
}

func TestChat_ListConversationsForBothParticipantsAndTypeFilter(t *testing.T) {
	user := chatUserClient(t)
	other := chatUser2Client(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")
	// A message gives the conversation a last_message_at so it sorts into the feed.
	sendMessage(t, user, convID, uniqueName("listme"), newIdempotencyKey())

	assert.True(t, listContainsConversation(t, user, convID, nil), "the creator sees the conversation")
	assert.True(t, listContainsConversation(t, other, convID, nil), "the recipient sees the conversation")

	assert.True(t, listContainsConversation(t, user, convID, url.Values{"type": {"direct_message"}}),
		"type=direct_message includes the DM")
	assert.False(t, listContainsConversation(t, user, convID, url.Values{"type": {"group"}}),
		"type=group excludes the DM")
}

func TestChat_BidirectionalReply(t *testing.T) {
	user := chatUserClient(t)
	other := chatUser2Client(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	sendMessage(t, user, convID, "ping", newIdempotencyKey())
	reply := sendMessage(t, other, convID, "pong", newIdempotencyKey())
	replySender, ok := reply["sender"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, SeedAccountUser2ID, jsonField(replySender, "id"), "the reply is attributed to the other user")

	// The original sender sees both messages and now has unread (the reply).
	list, _, err := user.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
	require.NoError(t, err)
	bodies := map[string]bool{}
	for _, raw := range list.Data {
		bodies[jsonField(parseJSON(raw), "body")] = true
	}
	assert.True(t, bodies["ping"] && bodies["pong"], "both directions appear in the thread")

	conv, err := user.GetFull(conversationsPath+"/"+convID, nil)
	require.NoError(t, err)
	unread, parseErr := strconv.Atoi(jsonField(parseJSON(conv.Body), "unread"))
	require.NoError(t, parseErr)
	assert.GreaterOrEqual(t, unread, 1, "the reply is unread for the original sender")
}

func TestChat_MessagePaginationAndCatchup(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	m1 := sendMessage(t, user, convID, "m1", newIdempotencyKey())
	sendMessage(t, user, convID, "m2", newIdempotencyKey())
	sendMessage(t, user, convID, "m3", newIdempotencyKey())
	seq1 := jsonField(m1, "sequence")

	// Page newest-first, one at a time, and confirm pages don't overlap.
	page1, _, err := user.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"2"}})
	require.NoError(t, err)
	require.Len(t, page1.Data, 2)
	require.True(t, page1.PageInfo.HasNextPage)
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page2, _, err := user.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	require.NotEmpty(t, page2.Data)
	assert.NotEqual(t, DataItemField(page1.Data[0], "id"), DataItemField(page2.Data[0], "id"),
		"older page returns different messages")

	// Catch-up: after_sequence returns only messages newer than seq1 (m2, m3).
	after, _, err := user.GetList(conversationsPath+"/"+convID+"/messages",
		url.Values{"limit": {"50"}, "after_sequence": {seq1}})
	require.NoError(t, err)
	for _, raw := range after.Data {
		s, convErr := strconv.ParseInt(jsonField(parseJSON(raw), "sequence"), 10, 64)
		require.NoError(t, convErr)
		s1, _ := strconv.ParseInt(seq1, 10, 64)
		assert.Greater(t, s, s1, "after_sequence must exclude messages at or before the bound")
	}
}

func TestChat_ReadCursorIsForwardOnly(t *testing.T) {
	user := chatUserClient(t)
	other := chatUser2Client(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	sendMessage(t, user, convID, "a", newIdempotencyKey())
	sendMessage(t, user, convID, "b", newIdempotencyKey())
	latest := sendMessage(t, user, convID, "c", newIdempotencyKey())
	latestSeq, _ := strconv.ParseInt(jsonField(latest, "sequence"), 10, 64)

	// other reads to the latest → unread 0.
	resp, err := other.PostFull(conversationsPath+"/"+convID+"/actions/read",
		map[string]any{"up_to_sequence": latestSeq}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.Equal(t, "0", jsonField(parseJSON(resp.Body), "unread"))

	// Marking an earlier sequence must NOT rewind the cursor.
	resp2, err := other.PostFull(conversationsPath+"/"+convID+"/actions/read",
		map[string]any{"up_to_sequence": 1}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp2.StatusCode, resp2.Body)
	assert.Equal(t, "0", jsonField(parseJSON(resp2.Body), "unread"),
		"a lower up_to_sequence must not rewind the read cursor")
}

func TestChat_NotFoundOnUnknownConversation(t *testing.T) {
	user := chatUserClient(t)
	missing := "cv_doesnotexist0000000000"

	status, body, err := user.Post(conversationsPath+"/"+missing+"/messages",
		map[string]any{"body": "x", "client_message_id": newIdempotencyKey()}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "sending to an unknown conversation should 404: %s", string(body))

	status, body, err = user.Post(conversationsPath+"/"+missing+"/actions/read",
		map[string]any{"up_to_sequence": 1}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "marking an unknown conversation read should 404: %s", string(body))

	resp, err := user.GetFull(conversationsPath+"/"+missing, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode, "getting an unknown conversation should 404: %s", string(resp.Body))
}
