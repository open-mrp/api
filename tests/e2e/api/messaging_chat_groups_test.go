//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-3 coverage: group create + roles, participant add/remove (incl. reactivate), role
// changes, leave/hide, mute, DM management guards, and blocks.

const blocksPath = "/v1/messaging/blocks"

func createGroupConversation(t *testing.T, c *Client, title string, memberAccountUserIDs ...string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(withQuery(conversationsPath, conversationIncludeQuery), map[string]any{
		"type":                         "group",
		"title":                        title,
		"participant_account_user_ids": memberAccountUserIDs,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// participantInfo returns the (participant id, role, state) for an account user in a conversation.
// The participant's user is surfaced via the polymorphic `actor` field (type=user).
func participantInfo(t *testing.T, conv map[string]any, accountUserID string) (string, string, string) {
	t.Helper()
	parts, ok := listData(conv, "participants")
	require.True(t, ok, "conversation has participants")
	for _, raw := range parts {
		p, _ := raw.(map[string]any)
		actor, _ := p["actor"].(map[string]any)
		if actor != nil && jsonField(actor, "id") == accountUserID {
			return jsonField(p, "id"), jsonField(p, "role"), jsonField(p, "membership")
		}
	}
	return "", "", ""
}

func participantMuted(t *testing.T, conv map[string]any, accountUserID string) bool {
	t.Helper()
	parts, _ := listData(conv, "participants")
	for _, raw := range parts {
		p, _ := raw.(map[string]any)
		actor, _ := p["actor"].(map[string]any)
		if actor != nil && jsonField(actor, "id") == accountUserID {
			return jsonField(p, "notifications") == "muted"
		}
	}
	return false
}

func TestChatGroup_CreateWithRoles(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	assert.Equal(t, "group", jsonField(conv, "type"))

	_, ownerRole, _ := participantInfo(t, conv, SeedAccountUserID)
	_, memberRole, _ := participantInfo(t, conv, SeedAccountUser2ID)
	assert.Equal(t, "owner", ownerRole, "the creator is the owner")
	assert.Equal(t, "member", memberRole, "added participants are members")
}

func TestChatGroup_RemoveAndReaddParticipant(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)
	require.NotEmpty(t, pid)

	// Owner removes the member.
	resp, err := owner.DeleteFull(conversationsPath + "/" + convID + "/participants/" + pid)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	// The removed user can no longer read the conversation.
	other := chatUser2Client(t)
	getResp, err := other.GetFull(conversationsPath+"/"+convID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getResp.StatusCode, "a removed participant loses access")

	// Owner re-adds the member (reactivation).
	addResp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/participants", participantIncludeQuery),
		map[string]any{"account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addResp.StatusCode, addResp.Body)
	_, _, state := participantInfo(t, parseJSON(addResp.Body), SeedAccountUser2ID)
	assert.Equal(t, "active", state, "re-adding reactivates the participant")
}

func TestChatGroup_MemberCannotManage(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	// A member (user2) cannot add participants or rename the group.
	member := chatUser2Client(t)
	addResp, err := member.PostFull(conversationsPath+"/"+convID+"/participants",
		map[string]any{"account_user_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, addResp.StatusCode, "a member cannot add participants")

	patch, err := member.PatchFull(conversationsPath+"/"+convID, map[string]any{"title": "hijacked"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, patch.StatusCode, "a member cannot rename the group")
}

func TestChatGroup_UpdateRoleAndRename(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	pid, _, _ := participantInfo(t, conv, SeedAccountUser2ID)

	// Owner promotes the member to admin.
	roleResp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/participants/"+pid+"/actions/set-role", participantIncludeQuery),
		map[string]any{"role": "admin"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, roleResp.StatusCode, roleResp.Body)
	_, role, _ := participantInfo(t, parseJSON(roleResp.Body), SeedAccountUser2ID)
	assert.Equal(t, "admin", role)

	// Owner renames the group.
	newTitle := uniqueName("renamed")
	patch, err := owner.PatchFull(conversationsPath+"/"+convID, map[string]any{"title": newTitle}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	assert.Equal(t, newTitle, jsonField(parseJSON(patch.Body), "title"))
}

func TestChatGroup_LeaveHidesConversation(t *testing.T) {
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	member := chatUser2Client(t)
	leaveResp, err := member.PostFull(conversationsPath+"/"+convID+"/actions/leave", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, leaveResp.StatusCode, leaveResp.Body)

	// After leaving, it no longer appears in the member's conversation list.
	assert.False(t, listContainsConversation(t, member, convID, nil), "a left conversation is hidden")
}

func TestChatGroup_MuteUnmute(t *testing.T) {
	owner := chatUserClient(t)
	convID := jsonField(createGroupConversation(t, owner, uniqueName("team"), SeedAccountUser2ID), "id")

	muteResp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/actions/mute", conversationIncludeQuery), map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, muteResp.StatusCode, muteResp.Body)
	assert.True(t, participantMuted(t, parseJSON(muteResp.Body), SeedAccountUserID), "muting sets notifications to muted")

	unmuteResp, err := owner.PostFull(withQuery(conversationsPath+"/"+convID+"/actions/unmute", conversationIncludeQuery), map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, unmuteResp.StatusCode, unmuteResp.Body)
	assert.False(t, participantMuted(t, parseJSON(unmuteResp.Body), SeedAccountUserID), "unmuting clears notifications mute")
}

func TestChatGroup_DMCannotBeManaged(t *testing.T) {
	user := chatUserClient(t)
	convID := jsonField(createDM(t, user, SeedAccountUser2ID), "id")

	patch, err := user.PatchFull(conversationsPath+"/"+convID, map[string]any{"title": "nope"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, patch.StatusCode, patch.Body)

	add, err := user.PostFull(conversationsPath+"/"+convID+"/participants",
		map[string]any{"account_user_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, add.StatusCode, add.Body)
}

func TestChatGroup_BlockPreventsDM(t *testing.T) {
	user := chatUserClient(t) // dane

	// dane blocks user2. Register the unblock cleanup first so a failed assertion still restores the
	// shared dane↔user2 DM (an orphaned block poisons every other DM test in the run).
	t.Cleanup(func() { _, _ = user.DeleteFull(blocksPath + "/" + SeedAccountUser2ID) })
	blockResp, err := user.PostFull(blocksPath, map[string]any{"blocked_account_user_id": SeedAccountUser2ID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, blockResp.StatusCode, blockResp.Body)
	assert.Equal(t, "messaging_block", jsonField(parseJSON(blockResp.Body), "object"), "block returns a messaging_block: %s", string(blockResp.Body))

	// Opening a DM with the blocked user is forbidden.
	status, body, err := user.Post(conversationsPath,
		map[string]any{"type": "direct_message", "participant_account_user_ids": []string{SeedAccountUser2ID}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "blocked users cannot DM: %s", string(body))

	// The block appears in the list.
	list, _, err := user.GetList(blocksPath, nil)
	require.NoError(t, err)
	var found bool
	for _, raw := range list.Data {
		if jsonField(parseJSON(raw), "object") == "messaging_block" {
			found = true
		}
	}
	assert.True(t, found, "the block is listed")

	// Unblocking restores DM creation.
	unblock, err := user.DeleteFull(blocksPath + "/" + SeedAccountUser2ID)
	require.NoError(t, err)
	requireStatus(t, 200, unblock.StatusCode, unblock.Body)
	status2, body2, err := user.Post(conversationsPath,
		map[string]any{"type": "direct_message", "participant_account_user_ids": []string{SeedAccountUser2ID}}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Contains(t, []int{200, 201}, status2, "after unblock, DM creation works: %s", string(body2))
}
