//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const agentAlertsPath = "/v1/ai/alerts"

func firstAgentAlertID(t *testing.T) string {
	t.Helper()
	list, status, err := apiClient.GetList(agentAlertsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "agent alerts list should return 200")
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one agent alert must be seeded")
	id := DataItemField(list.Data[0], "id")
	require.NotEmpty(t, id)
	return id
}

// ──────────────────────────────────────────────
// AgentAlert — Include Tests
// ──────────────────────────────────────────────

func TestAgentAlerts_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	id := firstAgentAlertID(t)

	status, body, err := apiClient.GetListRaw(agentAlertsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["run"], "run should be null without ?include=run")
	assert.Nil(t, got["action"], "action should be null without ?include=action")
}

func TestAgentAlerts_IncludeRun(t *testing.T) {
	t.Parallel()
	id := firstAgentAlertID(t)

	status, body, err := apiClient.GetListRaw(agentAlertsPath+"/"+id, url.Values{"include": {"run"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["run"]
	assert.True(t, ok, "run key should be present with ?include=run")
	if run := jsonObject(got, "run"); run != nil {
		assert.Equal(t, "agent_run", jsonField(run, "object"))
	}
}

func TestAgentAlerts_IncludeAction(t *testing.T) {
	t.Parallel()
	id := firstAgentAlertID(t)

	status, body, err := apiClient.GetListRaw(agentAlertsPath+"/"+id, url.Values{"include": {"action"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["action"]
	assert.True(t, ok, "action key should be present with ?include=action")
	if action := jsonObject(got, "action"); action != nil {
		assert.Equal(t, "agent_action", jsonField(action, "object"))
	}
}
