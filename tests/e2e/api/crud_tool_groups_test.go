//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const toolGroupsPath = "/v1/ai/tool-groups"

// ──────────────────────────────────────────────
// ToolGroup — Include Tests
// ──────────────────────────────────────────────

func TestToolGroups_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(toolGroupsPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "Should have at least 1 tool group")

	for _, item := range list.Data {
		m := parseJSON(item)
		assert.Nil(t, m["tools"], "tools should be null on list items without ?include=tools")
	}
}

func TestToolGroups_IncludeTools(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"include": {"tools"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var list struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &list))
	require.NotEmpty(t, list.Data, "should have at least one tool group")

	first := parseJSON(list.Data[0])
	tools := jsonObject(first, "tools")
	require.NotNil(t, tools, "tools should be present with ?include=tools")
	assert.Equal(t, "list", jsonField(tools, "object"))
}
