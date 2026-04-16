//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentDefinitionsPath = "/v1/ai/agents"

// ──────────────────────────────────────────────
// AgentDefinition — Include Tests
// ──────────────────────────────────────────────

func TestAgentDefinitions_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentDefinitionsPath+"/"+SeedCustomAgentDefinitionID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["config"], "config should be null without ?include=config")
	assert.Nil(t, got["tools"], "tools should be null without ?include=tools")
	assert.Nil(t, got["role"], "role should be null without ?include=role")
}

func TestAgentDefinitions_IncludeConfig(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(agentDefinitionsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"config"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["config"]
	assert.True(t, ok, "config key should be present with ?include=config")
}

func TestAgentDefinitions_IncludeTools(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(agentDefinitionsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"tools"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	tools := jsonObject(parseJSON(body), "tools")
	require.NotNil(t, tools, "tools should be present with ?include=tools")
	assert.Equal(t, "list", jsonField(tools, "object"))
}

func TestAgentDefinitions_IncludeRole(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(agentDefinitionsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"role"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["role"]
	assert.True(t, ok, "role key should be present with ?include=role")
	if role := jsonObject(got, "role"); role != nil {
		assert.Equal(t, "role", jsonField(role, "object"))
	}
}
