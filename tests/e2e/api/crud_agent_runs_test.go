//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentRunsPath = "/v1/ai/runs"

// firstAgentRunID returns the id of the first agent run in seed data. Fails
// loudly if the list endpoint errors or seed is empty.
func firstAgentRunID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(agentRunsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "agent runs list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one agent run must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// AgentRun — Include Tests
// ──────────────────────────────────────────────

func TestAgentRuns_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstAgentRunID(t)

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["actions"], "actions should be null without ?include=actions")
	assert.Nil(t, got["definition"], "definition should be null without ?include=definition")
	assert.Nil(t, got["steps"], "steps should be null without ?include=steps")
}

func TestAgentRuns_IncludeActions(t *testing.T) {
	t.Parallel()
	id := firstAgentRunID(t)

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+id, url.Values{"include": {"actions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["actions"]
	assert.True(t, ok, "actions key should be present with ?include=actions")
	if actions := jsonObject(got, "actions"); actions != nil {
		assert.Equal(t, "list", jsonField(actions, "object"))
	}
}

func TestAgentRuns_IncludeDefinition(t *testing.T) {
	t.Parallel()
	id := firstAgentRunID(t)

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+id, url.Values{"include": {"definition"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	def := jsonObject(parseJSON(body), "definition")
	require.NotNil(t, def, "definition should be present with ?include=definition")
	assert.Equal(t, "agent_definition", jsonField(def, "object"))
}

func TestAgentRuns_IncludeSteps(t *testing.T) {
	t.Parallel()
	id := firstAgentRunID(t)

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+id, url.Values{"include": {"steps"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["steps"]
	assert.True(t, ok, "steps key should be present with ?include=steps")
	if steps := jsonObject(got, "steps"); steps != nil {
		assert.Equal(t, "list", jsonField(steps, "object"))
	}
}
