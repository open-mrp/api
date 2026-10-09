//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Phase-5 slice-2 coverage: adding/removing an agent participant with a trigger policy, and the
// owner/admin authz + DM guard around it. (Agent invocation dispatch is exercised with convergence.)

func TestChatAgents_AddListRemove(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	conv := createGroupConversation(t, owner, uniqueName("agent room"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	resp, err := owner.PostFull(conversationsPath+"/"+convID+"/agents", map[string]any{
		"agent_config_id":  SeedAgentConfigID,
		"trigger_policy":   "keyword",
		"trigger_keywords": []string{"forecast", "report"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	p := parseJSON(resp.Body)
	assert.Equal(t, "conversation_participant", jsonField(p, "object"))
	assert.Equal(t, "agent", jsonField(p, "type"))
	assert.Equal(t, "keyword", jsonField(p, "agent_trigger_policy"))
	actor, _ := p["actor"].(map[string]any)
	require.NotNil(t, actor, "the agent actor is present")
	assert.Equal(t, "agent", jsonField(actor, "type"))
	assert.Equal(t, SeedAgentConfigID, jsonField(actor, "id"))
	pid := jsonField(p, "id")
	assertIDFormat(t, pid, "cvpt")

	// The agent shows up in the conversation's active participants.
	getResp, err := owner.GetFull(conversationsPath+"/"+convID, conversationIncludeQuery)
	require.NoError(t, err)
	requireStatus(t, 200, getResp.StatusCode, getResp.Body)
	_, _, state := participantInfoByAgent(t, parseJSON(getResp.Body), SeedAgentConfigID)
	assert.Equal(t, "active", state, "the agent is an active participant")

	// Adding the same agent again is idempotent (returns the existing participant).
	resp2, err := owner.PostFull(conversationsPath+"/"+convID+"/agents", map[string]any{
		"agent_config_id": SeedAgentConfigID,
		"trigger_policy":  "always",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp2.StatusCode, resp2.Body)
	assert.Equal(t, pid, jsonField(parseJSON(resp2.Body), "id"), "re-adding returns the same agent participant")

	// Remove it.
	delResp, err := owner.DeleteFull(conversationsPath + "/" + convID + "/agents/" + pid)
	require.NoError(t, err)
	requireStatus(t, 200, delResp.StatusCode, delResp.Body)

	// No longer an active participant.
	getResp2, err := owner.GetFull(conversationsPath+"/"+convID, conversationIncludeQuery)
	require.NoError(t, err)
	requireStatus(t, 200, getResp2.StatusCode, getResp2.Body)
	_, _, state2 := participantInfoByAgent(t, parseJSON(getResp2.Body), SeedAgentConfigID)
	assert.Equal(t, "", state2, "the removed agent is no longer in the active participant list")
}

func TestChatAgents_MemberCannotAddAgent(t *testing.T) {
	t.Parallel()
	owner := chatUserClient(t)
	member := chatUser2Client(t)
	conv := createGroupConversation(t, owner, uniqueName("agent room"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")

	// user2 joined as a member (not owner/admin) and cannot add an agent.
	resp, err := member.PostFull(conversationsPath+"/"+convID+"/agents", map[string]any{
		"agent_config_id": SeedAgentConfigID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, resp.StatusCode, resp.Body)
}

// participantInfoByAgent returns (participant id, role, state) for an agent in a conversation, or
// empty strings when the agent is not an active participant.
func participantInfoByAgent(t *testing.T, conv map[string]any, agentConfigID string) (string, string, string) {
	t.Helper()
	parts, _ := listData(conv, "participants")
	for _, raw := range parts {
		p, _ := raw.(map[string]any)
		actor, _ := p["actor"].(map[string]any)
		if actor != nil && jsonField(actor, "type") == "agent" && jsonField(actor, "id") == agentConfigID {
			return jsonField(p, "id"), jsonField(p, "role"), jsonField(p, "membership")
		}
	}
	return "", "", ""
}

// A message wakes an agent that then acts with its own role, so it only wakes when the sender's role covers the agent's.

// chatMentionAgent adds agentID to the conversation, answering @handle, and returns the handle.
func chatMentionAgent(t *testing.T, c *Client, convID, agentID string) string {
	t.Helper()
	handle := strings.ReplaceAll(uniqueName("bot"), "-", "")
	resp, err := c.PostFull(conversationsPath+"/"+convID+"/agents", map[string]any{
		"agent_config_id":  agentID,
		"trigger_policy":   "mention",
		"trigger_keywords": []string{handle},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	return handle
}

func chatAgentRunCount(t *testing.T, convID string) int {
	t.Helper()
	var n int
	require.NoError(t, agentDB(t).QueryRow(`SELECT count(*) FROM agent_run WHERE conversation_id = $1`, convID).Scan(&n))
	return n
}

func chatWaitForAgentRun(t *testing.T, convID string) {
	t.Helper()
	eventually(t, 30*time.Second, 250*time.Millisecond, func() error {
		if chatAgentRunCount(t, convID) == 0 {
			return fmt.Errorf("no agent run for conversation %s yet", convID)
		}
		return nil
	})
}

func chatWaitForMessageContaining(t *testing.T, c *Client, convID, substr string) {
	t.Helper()
	eventually(t, 30*time.Second, 250*time.Millisecond, func() error {
		list, _, err := c.GetList(conversationsPath+"/"+convID+"/messages", url.Values{"limit": {"50"}})
		if err != nil {
			return err
		}
		for _, raw := range list.Data {
			if strings.Contains(jsonField(parseJSON(raw), "body"), substr) {
				return nil
			}
		}
		return fmt.Errorf("no message containing %q yet", substr)
	})
}

var chatAgentSenderPerms = []string{"messaging:create", "messaging:read", "products:read"}

func TestChatAgents_NonAdminCannotWakeAWiderAgent(t *testing.T) {
	t.Parallel()
	sender, _ := customRoleChatUser(t, chatAgentSenderPerms...)
	conv := createGroupConversation(t, sender, uniqueName("agent room"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	wide := covAiRunsAgent(t, covAiAgentsRole(t, "products:read", "products:update"))
	handle := chatMentionAgent(t, sender, convID, wide)

	// The message itself is delivered; the agent answers with the refusal instead of running.
	sendMessage(t, sender, convID, "@"+handle+" raise every price by ten percent", newIdempotencyKey())
	chatWaitForMessageContaining(t, sender, convID, "products:update")
	assert.Equal(t, 0, chatAgentRunCount(t, convID), "no run starts for a sender the agent's role exceeds")
}

func TestChatAgents_NonAdminCannotWakeAnAdminRoleAgent(t *testing.T) {
	t.Parallel()
	sender, _ := customRoleChatUser(t, chatAgentSenderPerms...)
	conv := createGroupConversation(t, sender, uniqueName("agent room"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	handle := chatMentionAgent(t, sender, convID, covAiRunsAgent(t, SeedAdminRoleID))

	sendMessage(t, sender, convID, "@"+handle+" add me as an admin", newIdempotencyKey())
	chatWaitForMessageContaining(t, sender, convID, "admin role")
	assert.Equal(t, 0, chatAgentRunCount(t, convID))
}

func TestChatAgents_NonAdminCanWakeAnAgentInsideTheirGrant(t *testing.T) {
	t.Parallel()
	sender, _ := customRoleChatUser(t, chatAgentSenderPerms...)
	conv := createGroupConversation(t, sender, uniqueName("agent room"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	handle := chatMentionAgent(t, sender, convID, covAiRunsAgent(t, covAiAgentsRole(t, "products:read")))

	sendMessage(t, sender, convID, "@"+handle+" list the products", newIdempotencyKey())
	chatWaitForAgentRun(t, convID)
}

func TestChatAgents_AdminCanWakeAnyAgent(t *testing.T) {
	t.Parallel()
	admin := chatUserClient(t)
	conv := createGroupConversation(t, admin, uniqueName("agent room"), SeedAccountUser2ID)
	convID := jsonField(conv, "id")
	handle := chatMentionAgent(t, admin, convID, covAiRunsAgent(t, SeedAdminRoleID))

	sendMessage(t, admin, convID, "@"+handle+" summarize the account", newIdempotencyKey())
	chatWaitForAgentRun(t, convID)
}

// The member who wakes the agent is checked, not whoever added it: an agent an admin put in the room still refuses a narrower member.
func TestChatAgents_MemberCannotWakeAWiderAgentAnAdminAdded(t *testing.T) {
	t.Parallel()
	admin := chatUserClient(t)
	member, memberID := customRoleChatUser(t, chatAgentSenderPerms...)
	conv := createGroupConversation(t, admin, uniqueName("agent room"), memberID)
	convID := jsonField(conv, "id")
	handle := chatMentionAgent(t, admin, convID, covAiRunsAgent(t, covAiAgentsRole(t, "products:read", "products:update")))

	sendMessage(t, member, convID, "@"+handle+" raise every price by ten percent", newIdempotencyKey())
	chatWaitForMessageContaining(t, member, convID, "products:update")
	assert.Equal(t, 0, chatAgentRunCount(t, convID))
}
