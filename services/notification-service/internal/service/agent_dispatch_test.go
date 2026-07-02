package service

import (
	"testing"

	"github.com/augno/api/shared/constants"

	"github.com/stretchr/testify/assert"
)

func TestAgentTriggerFires_Always(t *testing.T) {
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyAlways, nil, "anything"))
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyAlways, nil, ""))
}

func TestAgentTriggerFires_Keyword(t *testing.T) {
	kws := []string{"forecast", "Order"}
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, kws, "what is the FORECAST today"), "case-insensitive keyword match")
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, kws, "place an order please"))
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, kws, "hello there"))
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, nil, "forecast"), "no keywords configured → never fires")
}

func TestAgentTriggerFires_Mention(t *testing.T) {
	kws := []string{"planner"}
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyMention, kws, "hey @planner can you help"), "mention requires the @ prefix")
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicyMention, kws, "the planner is out"), "bare keyword without @ does not mention")
}

func TestAgentTriggerFires_UnknownPolicy(t *testing.T) {
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicy("bogus"), []string{"x"}, "x"))
}
