//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Behavioral coverage for the feat/hubspot agent-tooling changes:
//   - GET /v1/ai/tools items are keyed by `slug` (was `id`) and expose `required_role_type`.
//   - An agent's attached tools carry `tool.slug` (was `tool.id`).
//   - Agent config round-trips `endpoint_tool_slugs` (the API-endpoint tool grant set).

const aiToolsPath = "/v1/ai/tools"

func TestAgentTools_ListExposesSlugAndRoleType(t *testing.T) {
	t.Parallel()

	list, code, err := apiClient.GetList(aiToolsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, code, "GET %s should 200", aiToolsPath)
	require.NotEmpty(t, list.Data, "the tool catalog is non-empty")

	sawRoleType := false
	for _, raw := range list.Data {
		tool := parseJSON(raw)
		assertObjectField(t, tool, "available_tool")
		assert.NotEmpty(t, jsonField(tool, "slug"), "each available tool is identified by a slug")
		assert.NotEmpty(t, jsonField(tool, "category"), "each available tool has a category")
		// required_role_type is nullable; just confirm the key is present in the shape.
		if _, present := tool["required_role_type"]; present {
			sawRoleType = true
		}
	}
	assert.True(t, sawRoleType, "the available_tool shape exposes required_role_type")
}

func TestAgentDefinitions_AttachedToolsCarrySlug(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentDefinitionsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"tools"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	toolsList := jsonObject(parseJSON(body), "tools")
	require.NotNil(t, toolsList, "tools present with ?include=tools")
	tools := jsonArray(toolsList, "data")
	require.NotEmpty(t, tools, "the seeded custom agent has at least one attached tool")

	for _, raw := range tools {
		attached, _ := raw.(map[string]any)
		require.NotNil(t, attached)
		assertObjectField(t, attached, "agent_definition_tool")
		tool := jsonObject(attached, "tool")
		require.NotNil(t, tool, "an attached tool embeds its available_tool")
		assert.NotEmpty(t, jsonField(tool, "slug"), "the embedded tool is identified by tool.slug")
		assert.Nil(t, tool["id"], "the embedded tool no longer carries a tool.id")
	}
}

func TestAgentDefinitions_CreateRoundTripsEndpointToolSlugs(t *testing.T) {
	t.Parallel()

	// A manual-trigger agent needs no trigger_config. `*` grants the whole
	// API-endpoint tool catalog; we assert it round-trips through config.
	created := createAndCleanup(t, agentDefinitionsPath, map[string]any{
		"name":          uniqueName("E2E Endpoint Tool Agent"),
		"slug":          uniqueName("e2e_endpoint_tool_agent"),
		"category_code": "inventory",
		"trigger_type":  "manual",
		"role_id":       SeedAdminRoleID,
		"config": map[string]any{
			"system_prompt":       "You are an e2e test agent.",
			"tier":                "high",
			"endpoint_tool_slugs": []string{"*"},
		},
	})
	assertObjectField(t, created, "agent_definition")
	agentID := jsonField(created, "id")
	require.NotEmpty(t, agentID)

	// Re-fetch with the config included and confirm the grant persisted.
	status, body, err := apiClient.GetListRaw(agentDefinitionsPath+"/"+agentID, url.Values{"include": {"config"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	config := jsonObject(parseJSON(body), "config")
	require.NotNil(t, config, "config present with ?include=config")
	assert.Contains(t, jsonStringSlice(config, "endpoint_tool_slugs"), "*",
		"endpoint_tool_slugs round-trips through the agent config")
}
