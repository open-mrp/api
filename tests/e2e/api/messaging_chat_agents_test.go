//go:build e2e

package api_test

import (
	"testing"

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
