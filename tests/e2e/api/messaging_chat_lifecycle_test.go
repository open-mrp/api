//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Further Phase-3 coverage: archive status round-trip, block-on-send in an existing DM, viewer
// post rules, removed-participant send gate, edit/delete edge cases, role validation, and the
// owner-only role-change rule.

func TestChatGroup_ArchiveStatusRoundTrip(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID), "id")

	archived, err := owner.PostFull(conversationsPath+"/"+convID+"/actions/archive", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, archived.StatusCode, archived.Body)
	assert.Equal(t, "archived", jsonField(parseJSON(archived.Body), "status"))

	active, err := owner.PostFull(conversationsPath+"/"+convID+"/actions/unarchive", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, active.StatusCode, active.Body)
	assert.Equal(t, "active", jsonField(parseJSON(active.Body), "status"))
}

func TestChatGroup_BlockPreventsSendInExistingDM(t *testing.T) {
	dane := chatUserClient(t)
	convID := jsonField(createDM(t, dane, SeedAccountUser2ID), "id")

	// Register the unblock cleanup before the block POST so a failed assertion still restores the
	// shared dane↔user2 DM (an orphaned block poisons every other DM test in the run).
	t.Cleanup(func() { _, _ = dane.DeleteFull(blocksPath + "/" + SeedAccountUser2ID) })
	blockResp, err := dane.PostFull(blocksPath, map[string]any{"blocked_account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, blockResp.StatusCode, blockResp.Body)

	// Sending in the pre-existing DM is now forbidden in both directions.
	status, body, err := dane.Post(conversationsPath+"/"+convID+"/messages",
		map[string]any{"body": "after block", "client_message_id": newIdempotencyKey()}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "a block forbids sending in an existing DM: %s", string(body))

	user2 := chatUser2Client(t)
	status2, body2, err := user2.Post(conversationsPath+"/"+convID+"/messages",
		map[string]any{"body": "blocked reply", "client_message_id": newIdempotencyKey()}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status2, "the block is symmetric: %s", string(body2))
}

func TestChatGroup_ViewerCannotPost(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)

	// Owner demotes the member to viewer.
	roleResp, err := owner.PostFull(conversationsPath+"/"+convID+"/participants/"+pid+"/actions/set-role",
		map[string]any{"role": "viewer"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, roleResp.StatusCode, roleResp.Body)

	// A viewer cannot post.
	viewer := chatUser2Client(t)
	status, body, err := viewer.Post(conversationsPath+"/"+convID+"/messages",
		map[string]any{"body": "i am a viewer", "client_message_id": newIdempotencyKey()}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "a viewer cannot post: %s", string(body))
}

func TestChatGroup_RemovedParticipantCannotSend(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)

	resp, err := owner.DeleteFull(conversationsPath + "/" + convID + "/participants/" + pid)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	// The removed member can no longer post.
	removed := chatUser2Client(t)
	status, body, err := removed.Post(conversationsPath+"/"+convID+"/messages",
		map[string]any{"body": "still here?", "client_message_id": newIdempotencyKey()}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status, "a removed participant cannot send: %s", string(body))
}

func TestChatGroup_AddParticipantRoleValidation(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	addPath := conversationsPath + "/" + convID + "/participants"

	// Assigning owner via add is rejected.
	ownerRole, err := owner.PostFull(addPath, map[string]any{"account_user_id": SeedAccountUser2ID, "role": "owner"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, ownerRole.StatusCode, ownerRole.Body)

	// An unknown role is rejected.
	badRole, err := owner.PostFull(addPath, map[string]any{"account_user_id": SeedAccountUser2ID, "role": "wizard"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, badRole.StatusCode, badRole.Body)

	// Remove then re-add as a viewer (reactivation honors the requested role).
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)
	rm, err := owner.DeleteFull(addPath + "/" + pid)
	require.NoError(t, err)
	requireStatus(t, 200, rm.StatusCode, rm.Body)
	readd, err := owner.PostFull(withQuery(addPath, participantIncludeQuery), map[string]any{"account_user_id": SeedAccountUser2ID, "role": "viewer"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, readd.StatusCode, readd.Body)
	_, role, _ := participantInfo(t, parseJSON(readd.Body), SeedAccountUser2ID)
	assert.Equal(t, "viewer", role)
}

func TestChatGroup_OnlyOwnerChangesRoles(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	memberPid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)
	ownerPid, _, _ := participantInfo(t, conv, SeedAccountUserID)

	// Promote the member to admin.
	promote, err := owner.PostFull(conversationsPath+"/"+convID+"/participants/"+memberPid+"/actions/set-role",
		map[string]any{"role": "admin"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, promote.StatusCode, promote.Body)

	// An admin (not owner) cannot change roles.
	admin := chatUser2Client(t)
	denied, err := admin.PostFull(conversationsPath+"/"+convID+"/participants/"+ownerPid+"/actions/set-role",
		map[string]any{"role": "member"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, denied.StatusCode, "only the owner may change roles")
}
