//go:build e2e

package api_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-6c coverage: an @mention pierces mute. A muted participant receives a chat.mention bell when
// they are mentioned, but ordinary messages in the same muted conversation stay silent.

// feedHasBody reports whether the reader's feed for a category contains a notification body.
func feedHasBody(t *testing.T, reader *Client, category, body string) bool {
	t.Helper()
	list, _, err := reader.GetList(notificationsPath, url.Values{"limit": {"100"}, "category": {category}})
	require.NoError(t, err)
	for _, raw := range list.Data {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if jsonField(m, "body") == body {
			return true
		}
	}
	return false
}

func TestMentions_PierceMute(t *testing.T) {
	owner := chatUserClient(t)
	member := chatUser2Client(t)

	group := createGroupConversation(t, owner, uniqueName("mention war room"), SeedAccountUser2ID)
	convID := jsonField(group, "id")

	// The member mutes the conversation: ordinary messages must not raise a bell for them.
	muteResp, err := member.PostFull(conversationsPath+"/"+convID+"/actions/mute", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, muteResp.StatusCode, muteResp.Body)

	// An ordinary (un-mentioned) message stays silent for the muted member.
	silentBody := uniqueName("just chatter")
	sendMessage(t, owner, convID, silentBody, newIdempotencyKey())

	// A message that @mentions the muted member pierces the mute as a chat.mention bell.
	mentionBody := uniqueName("please review this @user2")
	resp, err := owner.PostFull(conversationsPath+"/"+convID+"/messages", map[string]any{
		"body":              mentionBody,
		"client_message_id": uniqueName("cmid"),
		"mentions":          []string{SeedAccountUser2ID},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	// The mention surfaces in the member's chat.mention feed despite the mute.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if feedHasBody(t, member, "chat.mention", mentionBody) {
			return nil
		}
		return errors.New("mention bell not yet delivered to the muted member")
	})

	// The ordinary muted message never raised a chat.message bell for the member.
	assert.False(t, feedHasBody(t, member, "chat.message", silentBody),
		"a muted, un-mentioned message must not raise a bell")
}
