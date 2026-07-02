//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for reusable rosters (messaging groups): CRUD, member management, and seeding many
// conversations from one roster with snapshot (not live) membership.

const messagingGroupsPath = "/v1/messaging/groups"

// createMessagingGroup creates a roster and returns the parsed resource.
func createMessagingGroup(t *testing.T, c *Client, name string, userIDs []string, agentIDs []string) map[string]any {
	t.Helper()
	body := map[string]any{"name": name}
	if len(userIDs) > 0 {
		body["member_account_user_ids"] = userIDs
	}
	if len(agentIDs) > 0 {
		body["member_agent_config_ids"] = agentIDs
	}
	resp, err := c.PostFull(messagingGroupsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

// groupMemberActorIDs returns the set of member actor ids (account_user/agent ids) on a roster.
func groupMemberActorIDs(t *testing.T, group map[string]any) map[string]string {
	t.Helper()
	out := map[string]string{} // actor id -> actor type
	members, _ := listData(group, "members")
	for _, raw := range members {
		m, _ := raw.(map[string]any)
		actor, _ := m["actor"].(map[string]any)
		if actor != nil {
			out[jsonField(actor, "id")] = jsonField(actor, "type")
		}
	}
	return out
}

// groupMemberID returns the membership id for a given actor id within a roster.
func groupMemberID(t *testing.T, group map[string]any, actorID string) string {
	t.Helper()
	members, _ := listData(group, "members")
	for _, raw := range members {
		m, _ := raw.(map[string]any)
		actor, _ := m["actor"].(map[string]any)
		if actor != nil && jsonField(actor, "id") == actorID {
			return jsonField(m, "id")
		}
	}
	return ""
}

func TestMessagingGroup_CRUD(t *testing.T) {
	owner := chatUserClient(t)

	// Create a roster with a user and an agent.
	group := createMessagingGroup(t, owner, uniqueName("ops"), []string{SeedAccountUser2ID}, []string{SeedAgentConfigID})
	groupID := jsonField(group, "id")
	require.NotEmpty(t, groupID)
	assert.Equal(t, "messaging_group", jsonField(group, "object"))

	members := groupMemberActorIDs(t, group)
	assert.Equal(t, "user", members[SeedAccountUser2ID], "the user member is present")
	assert.Equal(t, "agent", members[SeedAgentConfigID], "the agent member is present")

	// Get returns the same roster with members.
	getResp, err := owner.GetFull(messagingGroupsPath+"/"+groupID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	assert.Len(t, groupMemberActorIDs(t, parseJSON(getResp.Body)), 2)

	// List includes the roster.
	list, _, err := owner.GetList(messagingGroupsPath, nil)
	require.NoError(t, err)
	var found bool
	for _, raw := range list.Data {
		if jsonField(parseJSON(raw), "id") == groupID {
			found = true
		}
	}
	assert.True(t, found, "the roster is listed")

	// Rename.
	newName := uniqueName("renamed-ops")
	patch, err := owner.PatchFull(messagingGroupsPath+"/"+groupID, map[string]any{"name": newName}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patch.StatusCode, patch.Body)
	assert.Equal(t, newName, jsonField(parseJSON(patch.Body), "name"))

	// Remove the agent member.
	agentMemberID := groupMemberID(t, group, SeedAgentConfigID)
	require.NotEmpty(t, agentMemberID)
	delMember, err := owner.DeleteFull(messagingGroupsPath + "/" + groupID + "/members/" + agentMemberID)
	require.NoError(t, err)
	requireStatus(t, 200, delMember.StatusCode, delMember.Body)
	_, agentStillThere := groupMemberActorIDs(t, parseJSON(delMember.Body))[SeedAgentConfigID]
	assert.False(t, agentStillThere, "the agent member was removed")

	// Add the agent back.
	addMember, err := owner.PostFull(messagingGroupsPath+"/"+groupID+"/members",
		map[string]any{"member_type": "agent", "agent_config_id": SeedAgentConfigID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, addMember.StatusCode, addMember.Body)
	_, agentBack := groupMemberActorIDs(t, parseJSON(addMember.Body))[SeedAgentConfigID]
	assert.True(t, agentBack, "the agent member was re-added")

	// Delete the roster.
	del, err := owner.DeleteFull(messagingGroupsPath + "/" + groupID)
	require.NoError(t, err)
	requireStatus(t, 200, del.StatusCode, del.Body)
	getGone, err := owner.GetFull(messagingGroupsPath+"/"+groupID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getGone.StatusCode, "a deleted roster is gone")
}

// startConversationFromGroup creates a group conversation seeded from a roster, with its own title.
func startConversationFromGroup(t *testing.T, c *Client, groupID, title string) map[string]any {
	t.Helper()
	resp, err := c.PostFull(withQuery(conversationsPath, conversationIncludeQuery), map[string]any{
		"type":     "group",
		"title":    title,
		"group_id": groupID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return parseJSON(resp.Body)
}

func TestMessagingGroup_SeedsConversationsWithSnapshot(t *testing.T) {
	owner := chatUserClient(t)
	group := createMessagingGroup(t, owner, uniqueName("ops"), []string{SeedAccountUser2ID}, []string{SeedAgentConfigID})
	groupID := jsonField(group, "id")

	// Start two distinct conversations from the same roster, each with its own title.
	conv1 := startConversationFromGroup(t, owner, groupID, uniqueName("Q3 Planning"))
	conv2 := startConversationFromGroup(t, owner, groupID, uniqueName("Incident 42"))

	id1, id2 := jsonField(conv1, "id"), jsonField(conv2, "id")
	assert.NotEqual(t, id1, id2, "each create yields a distinct conversation (no dedup)")
	assert.NotEqual(t, jsonField(conv1, "title"), jsonField(conv2, "title"), "each conversation keeps its own title")

	// Each conversation references the roster it was seeded from.
	for _, conv := range []map[string]any{conv1, conv2} {
		groupRef, _ := conv["group"].(map[string]any)
		require.NotNil(t, groupRef, "conversation carries its source group")
		assert.Equal(t, groupID, jsonField(groupRef, "id"))
	}

	// Both conversations seeded the roster's members: the creator (owner), the user, and the agent
	// (defaulting to the mention trigger). Membership came from a snapshot.
	for _, conv := range []map[string]any{conv1, conv2} {
		_, ownerRole, _ := participantInfo(t, conv, SeedAccountUserID)
		assert.Equal(t, "owner", ownerRole, "the creator is the owner")
		_, memberRole, memberState := participantInfo(t, conv, SeedAccountUser2ID)
		assert.Equal(t, "member", memberRole, "the roster user is a member")
		assert.Equal(t, "active", memberState)
		_, _, agentState := participantInfoByAgent(t, conv, SeedAgentConfigID)
		assert.Equal(t, "active", agentState, "the roster agent was seated")
	}

	// Snapshot semantics: removing a member from the roster does NOT change conversations already
	// created from it.
	user2MemberID := groupMemberID(t, group, SeedAccountUser2ID)
	require.NotEmpty(t, user2MemberID)
	del, err := owner.DeleteFull(messagingGroupsPath + "/" + groupID + "/members/" + user2MemberID)
	require.NoError(t, err)
	requireStatus(t, 200, del.StatusCode, del.Body)

	getConv, err := owner.GetFull(withQuery(conversationsPath+"/"+id1, conversationIncludeQuery), nil)
	require.NoError(t, err)
	requireStatus(t, 200, getConv.StatusCode, getConv.Body)
	_, _, stillState := participantInfo(t, parseJSON(getConv.Body), SeedAccountUser2ID)
	assert.Equal(t, "active", stillState, "an existing conversation is unaffected by later roster edits (snapshot)")
}

func TestMessagingGroup_DeleteDetachesConversations(t *testing.T) {
	owner := chatUserClient(t)
	group := createMessagingGroup(t, owner, uniqueName("ops"), []string{SeedAccountUser2ID}, nil)
	groupID := jsonField(group, "id")

	convID := jsonField(startConversationFromGroup(t, owner, groupID, uniqueName("planning")), "id")

	del, err := owner.DeleteFull(messagingGroupsPath + "/" + groupID)
	require.NoError(t, err)
	requireStatus(t, 200, del.StatusCode, del.Body)

	// The conversation survives the roster's deletion; only the provenance link is detached.
	getConv, err := owner.GetFull(withQuery(conversationsPath+"/"+convID, conversationIncludeQuery), nil)
	require.NoError(t, err)
	requireStatus(t, 200, getConv.StatusCode, getConv.Body)
	conv := parseJSON(getConv.Body)
	assert.Nil(t, conv["group"], "the deleted roster's link is nulled on the conversation")
	_, _, memberState := participantInfo(t, conv, SeedAccountUser2ID)
	assert.Equal(t, "active", memberState, "the conversation's members are untouched")
}

func TestMessagingGroup_AdHocGroupStillWorks(t *testing.T) {
	owner := chatUserClient(t)
	// A group conversation created without a roster (ad-hoc) keeps working and has no group link.
	conv := createGroupConversation(t, owner, uniqueName("adhoc"), SeedAccountUser2ID)
	assert.Equal(t, "group", jsonField(conv, "type"))
	assert.Nil(t, conv["group"], "an ad-hoc group conversation has no roster link")
}
