//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §12.12 self-DM ("note to self"): a DM where the pair is the caller themselves is allowed and has a
// single participant.
func TestSelfDM_Allowed(t *testing.T) {
	user := chatUserClient(t)

	dm := createDM(t, user, SeedAccountUserID) // target == caller
	assert.Equal(t, "direct_message", jsonField(dm, "type"))
	convID := jsonField(dm, "id")
	assertIDFormat(t, convID, "cv")

	parts, ok := listData(dm, "participants")
	require.True(t, ok)
	assert.Len(t, parts, 1, "a self-DM has a single participant")

	// Re-creating the self-DM is deduped to the same conversation.
	again := createDM(t, user, SeedAccountUserID)
	assert.Equal(t, convID, jsonField(again, "id"), "self-DM create is deduped")

	// The user can post a note to themselves.
	body := uniqueName("remember to ship")
	sendMessage(t, user, convID, body, newIdempotencyKey())
}

// §12.12 being added to a conversation writes a chat.added bell for the added user.
func TestAddedToConversation_WritesBell(t *testing.T) {
	owner := chatUserClient(t)
	member := chatUser2Client(t)

	// Creating a group with user2 as an initial member writes them a chat.added bell. The bell's
	// body is the (unique) group title, so we match on it.
	title := uniqueName("project room")
	createGroupConversation(t, owner, title, SeedAccountUser2ID)

	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		if feedHasBody(t, member, "chat.added", title) {
			return nil
		}
		return errAddedBellMissing
	})
}

var (
	errAddedBellMissing = &simpleErr{"chat.added bell not yet delivered"}
	errBellMissing      = &simpleErr{"chat.message bell not yet delivered"}
)

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
