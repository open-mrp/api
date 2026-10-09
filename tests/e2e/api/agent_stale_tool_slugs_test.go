//go:build e2e

package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const staleEndpointToolSlug = "endpoint_tool_removed_from_catalog"
const staleBuiltinToolSlug = "builtin_tool_removed_from_catalog"

func createAgentWithStaleEndpointTool(t *testing.T, suffix string) string {
	t.Helper()
	body := covAiAgentsMinimalCreateBody(suffix)
	body["config"] = map[string]any{
		"endpoint_tool_slugs":  []string{"create_account_group"},
		"endpoint_tool_review": map[string]any{"create_account_group": true},
	}
	created := createAndCleanup(t, covAiAgentsPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	stored, err := json.Marshal(map[string]any{
		"endpoint_tool_slugs":  []string{"create_account_group", staleEndpointToolSlug},
		"endpoint_tool_review": map[string]bool{"create_account_group": true, staleEndpointToolSlug: true},
	})
	require.NoError(t, err)
	_, err = agentDB(t).Exec(`UPDATE agent_definition SET config = $1 WHERE id = $2`, stored, id)
	require.NoError(t, err)
	return id
}

func TestAgentStaleSlugs_GetWithStaleEndpointToolSucceeds(t *testing.T) {
	t.Parallel()
	id := createAgentWithStaleEndpointTool(t, "stale-get")

	status, body, err := apiClient.Do("GET", covAiAgentsPath+"/"+id+"?include=config", nil, "")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	config := jsonObject(parseJSON(body), "config")
	require.NotNil(t, config)
	assert.Contains(t, jsonStringSlice(config, "endpoint_tool_slugs"), "create_account_group")
}

func TestAgentStaleSlugs_PatchResendingStoredStaleEndpointToolPrunesIt(t *testing.T) {
	t.Parallel()
	id := createAgentWithStaleEndpointTool(t, "stale-resend")

	status, body, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=config", map[string]any{
		"name": "Stale slug resend",
		"config": map[string]any{
			"endpoint_tool_slugs":  []string{"create_account_group", staleEndpointToolSlug},
			"endpoint_tool_review": map[string]any{"create_account_group": true, staleEndpointToolSlug: true},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	patched := parseJSON(body)
	assert.Equal(t, "Stale slug resend", jsonField(patched, "name"))
	config := jsonObject(patched, "config")
	require.NotNil(t, config)
	assert.Equal(t, []string{"create_account_group"}, jsonStringSlice(config, "endpoint_tool_slugs"))
	assert.Equal(t, map[string]any{"create_account_group": true}, jsonObject(config, "endpoint_tool_review"))

	var stored []byte
	require.NoError(t, agentDB(t).QueryRow(`SELECT config FROM agent_definition WHERE id = $1`, id).Scan(&stored))
	assert.NotContains(t, string(stored), staleEndpointToolSlug, "the stale slug is pruned from the stored config")
}

func TestAgentStaleSlugs_PatchOtherConfigKeyPrunesStoredStaleEndpointTool(t *testing.T) {
	t.Parallel()
	id := createAgentWithStaleEndpointTool(t, "stale-other")

	status, body, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=config", map[string]any{
		"config": map[string]any{"tier": "cheap"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	config := jsonObject(parseJSON(body), "config")
	require.NotNil(t, config)
	assert.Equal(t, "cheap", jsonField(config, "tier"))
	assert.Equal(t, []string{"create_account_group"}, jsonStringSlice(config, "endpoint_tool_slugs"))
}

func TestAgentStaleSlugs_PatchAddingNewUnknownEndpointToolRejected(t *testing.T) {
	t.Parallel()
	id := createAgentWithStaleEndpointTool(t, "stale-newunknown")

	status, body, err := apiClient.Patch(covAiAgentsPath+"/"+id, map[string]any{
		"config": map[string]any{
			"endpoint_tool_slugs": []string{"create_account_group", staleEndpointToolSlug, "not_a_real_endpoint_tool"},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "tools")
	assert.Contains(t, errObj["message"], "not_a_real_endpoint_tool")
}

func TestAgentStaleSlugs_PatchResendingStoredStaleBuiltinToolDropsIt(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("stale-builtin")
	body["tools"] = []map[string]any{{"tool": "read_doc", "sort_order": 1}}
	created := createAndCleanup(t, covAiAgentsPath, body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	_, err := agentDB(t).Exec(`INSERT INTO agent_definition_tool (id, agent_definition_id, tool_slug, config, sort_order, require_review)
		VALUES ($1, $2, $3, '{}', 2, false)`, "adt_stale_"+id, id, staleBuiltinToolSlug)
	require.NoError(t, err)

	status, respBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=tools", map[string]any{
		"tools": []map[string]any{
			{"tool": "read_doc", "sort_order": 1},
			{"tool": staleBuiltinToolSlug, "sort_order": 2},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)

	tools := jsonArray(jsonObject(parseJSON(respBody), "tools"), "data")
	require.Len(t, tools, 1)
	assert.Equal(t, "read_doc", jsonField(jsonObject(tools[0].(map[string]any), "tool"), "slug"))

	var n int
	require.NoError(t, agentDB(t).QueryRow(`SELECT count(*) FROM agent_definition_tool WHERE agent_definition_id = $1 AND tool_slug = $2`, id, staleBuiltinToolSlug).Scan(&n))
	assert.Zero(t, n, "the stale built-in link is not re-created")

	status, respBody, err = apiClient.Patch(covAiAgentsPath+"/"+id, map[string]any{
		"tools": []map[string]any{{"tool": "not_a_real_tool"}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "tools")
}
