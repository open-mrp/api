//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for the per-caller conversation visibility actions (hide/unhide) and
// the ephemeral typing indicator — none of which had dedicated e2e tests.
// Hiding is a per-participant flag: a hidden conversation drops out of the
// owner's default (status=active) list and surfaces only under status=hidden,
// without affecting the other participant's view.

var hiddenListQuery = url.Values{"status": {"hidden"}}

func TestChat_HideAndUnhide(t *testing.T) {
	// The owner cannot hide a group they own, so hiding is exercised by a member.
	owner := chatUserClient(t)
	member := chatUser2Client(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("hideme"), SeedAccountUser2ID), "id")
	// A message gives it a last_message_at so it sorts into the active feed.
	sendMessage(t, owner, convID, uniqueName("hi"), newIdempotencyKey())

	require.True(t, listContainsConversation(t, member, convID, nil), "the conversation starts in the member's active list")

	// Hide it for the member.
	hideResp, err := member.PostFull(conversationsPath+"/"+convID+"/actions/hide", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, hideResp.StatusCode, hideResp.Body)
	hidden := parseJSON(hideResp.Body)
	assert.Equal(t, "conversation", jsonField(hidden, "object"), "hide returns the refreshed conversation")
	assert.Equal(t, convID, jsonField(hidden, "id"))

	assert.False(t, listContainsConversation(t, member, convID, nil),
		"a hidden conversation drops out of the member's default (active) list")
	assert.True(t, listContainsConversation(t, member, convID, hiddenListQuery),
		"a hidden conversation appears under status=hidden")
	assert.True(t, listContainsConversation(t, owner, convID, nil),
		"hiding is per-caller; the owner still sees it")

	// Hiding again is idempotent.
	reHide, err := member.PostFull(conversationsPath+"/"+convID+"/actions/hide", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, reHide.StatusCode, reHide.Body)
	assert.True(t, listContainsConversation(t, member, convID, hiddenListQuery), "re-hiding keeps it hidden")

	// Unhide restores it to the member's active list.
	unhideResp, err := member.PostFull(conversationsPath+"/"+convID+"/actions/unhide", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, unhideResp.StatusCode, unhideResp.Body)
	assert.True(t, listContainsConversation(t, member, convID, nil),
		"an unhidden conversation returns to the active list")
	assert.False(t, listContainsConversation(t, member, convID, hiddenListQuery),
		"an unhidden conversation no longer appears under status=hidden")
}

func TestChat_UnhideNonHiddenIsNoop(t *testing.T) {
	owner := chatUserClient(t)
	member := chatUser2Client(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("nothidden"), SeedAccountUser2ID), "id")

	// Unhiding a conversation that was never hidden succeeds and leaves it active.
	resp, err := member.PostFull(conversationsPath+"/"+convID+"/actions/unhide", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.True(t, listContainsConversation(t, member, convID, nil), "the conversation stays active")
}

func TestChat_HideUnhideUnknownConversation(t *testing.T) {
	user := chatUserClient(t)
	missing := "cv_doesnotexist0000000000"

	for _, action := range []string{"hide", "unhide"} {
		status, body, err := user.Post(conversationsPath+"/"+missing+"/actions/"+action, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 404, status, "%s on an unknown conversation should 404: %s", action, string(body))
	}
}

func TestChat_HideNonParticipantRejected(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("private"), SeedAccountUser2ID), "id")

	// The customer API key is not a participant → must not hide the conversation (existence not leaked).
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.Post(conversationsPath+"/"+convID+"/actions/hide", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status,
		"a non-participant must not hide the conversation, got %d: %s", status, string(body))
}

func TestChat_TypingIndicator(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("typing"), SeedAccountUser2ID), "id")

	// Typing is ephemeral (not persisted) and acknowledged with 202 Accepted.
	resp, err := owner.PostFull(conversationsPath+"/"+convID+"/actions/typing", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 202, resp.StatusCode, resp.Body)
}

func TestChat_TypingUnknownConversation(t *testing.T) {
	user := chatUserClient(t)
	missing := "cv_doesnotexist0000000000"
	status, body, err := user.Post(conversationsPath+"/"+missing+"/actions/typing", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "typing on an unknown conversation should 404: %s", string(body))
}

func TestChat_TypingNonParticipantRejected(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("typingpriv"), SeedAccountUser2ID), "id")

	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.Post(conversationsPath+"/"+convID+"/actions/typing", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{403, 404}, status,
		"a non-participant must not send typing to the conversation, got %d: %s", status, string(body))
}
