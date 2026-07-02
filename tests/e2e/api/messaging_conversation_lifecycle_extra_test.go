//go:build e2e

package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for conversation lifecycle edges: timed mute, and unknown-id /
// invalid-input handling on mute, redact, and legal-hold.

func TestChat_MuteWithExpiry(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("tmute"), SeedAccountUser2ID), "id")

	until := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	resp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/actions/mute", conversationIncludeQuery),
		map[string]any{"muted_until": until}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
	assert.True(t, participantMuted(t, parseJSON(resp.Body), SeedAccountUserID), "a timed mute still marks the participant muted")

	// Unmute restores it.
	unmute, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/actions/unmute", conversationIncludeQuery),
		map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, unmute.StatusCode, unmute.Body)
	assert.False(t, participantMuted(t, parseJSON(unmute.Body), SeedAccountUserID))
}

func TestChat_MuteUnknownConversation(t *testing.T) {
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath+"/cv_doesnotexist0000000000/actions/mute", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "muting an unknown conversation should 404: %s", string(body))
}

func TestChat_RedactUnknownConversation(t *testing.T) {
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath+"/cv_doesnotexist0000000000/redact", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "redacting an unknown conversation should 404: %s", string(body))
}

func TestChat_LegalHoldUnknownConversation(t *testing.T) {
	user := chatUserClient(t)
	status, body, err := user.Post(conversationsPath+"/cv_doesnotexist0000000000/actions/set-legal-hold",
		map[string]any{"legal_hold": "held"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "legal-hold on an unknown conversation should 404: %s", string(body))
}

func TestChat_LegalHoldInvalidStatusRejected(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("lh"), SeedAccountUser2ID), "id")

	status, body, err := owner.Post(conversationsPath+"/"+convID+"/actions/set-legal-hold",
		map[string]any{"legal_hold": "not_a_real_status"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
