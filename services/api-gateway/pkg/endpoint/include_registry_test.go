package apiendpoint

import (
	"testing"

	"github.com/augno/api/shared/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncludesFor_AgentDefinition(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAgentDefinition,
		Fields:     []string{"config", "tools", "role"},
	})
	require.Len(t, cfg.Fields, 3)
	assert.Equal(t, IncludeField{Key: "config", ObjectType: constants.ObjectTypeAgentDefinition, JSONPaths: []string{"config"}}, cfg.Fields[0])
	assert.Equal(t, IncludeField{Key: "tools", ObjectType: constants.ObjectTypeAgentDefinitionTool, JSONPaths: []string{"tools"}}, cfg.Fields[1])
	assert.Equal(t, IncludeField{Key: "role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"role"}}, cfg.Fields[2])
}

func TestIncludesFor_AgentAlert(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAgentAlert,
		Fields:     []string{"run", "action"},
	})
	require.Len(t, cfg.Fields, 2)
	assert.Equal(t, IncludeField{Key: "run", ObjectType: constants.ObjectTypeAgentRun, JSONPaths: []string{"run"}}, cfg.Fields[0])
	assert.Equal(t, IncludeField{Key: "action", ObjectType: constants.ObjectTypeAgentAction, JSONPaths: []string{"action"}}, cfg.Fields[1])
}

func TestIncludesFor_AgentRun_Subset(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAgentRun,
		Fields:     []string{"definition", "actions"},
	})
	require.Len(t, cfg.Fields, 2)
	assert.Equal(t, "definition", cfg.Fields[0].Key)
	assert.Equal(t, "actions", cfg.Fields[1].Key)
}

func TestIncludesFor_AgentRun_Full(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAgentRun,
		Fields:     []string{"actions", "definition", "steps"},
	})
	require.Len(t, cfg.Fields, 3)
	assert.Equal(t, "actions", cfg.Fields[0].Key)
	assert.Equal(t, constants.ObjectTypeAgentAction, cfg.Fields[0].ObjectType)
	assert.Equal(t, "definition", cfg.Fields[1].Key)
	assert.Equal(t, "steps", cfg.Fields[2].Key)
}

func TestIncludesFor_APIKey(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAPIKey,
		Fields:     []string{"role"},
	})
	require.Len(t, cfg.Fields, 1)
	assert.Equal(t, IncludeField{Key: "role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"role"}}, cfg.Fields[0])
}

func TestIncludesFor_APIKey_WithPathPrefix(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAPIKey,
		Fields:     []string{"role"},
		PathPrefix: "api_key_info",
	})
	require.Len(t, cfg.Fields, 1)
	assert.Equal(t, IncludeField{Key: "role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"api_key_info.role"}}, cfg.Fields[0])
}

func TestIncludesFor_RequestLog(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeRequestLog,
		Fields:     []string{"account", "actor", "actor.role", "query_json", "request_body_json", "response_body_json"},
	})
	require.Len(t, cfg.Fields, 6)
	assert.Equal(t, IncludeField{Key: "account", ObjectType: constants.ObjectTypeAccount, JSONPaths: []string{"account"}}, cfg.Fields[0])
	assert.Equal(t, IncludeField{Key: "actor", ObjectType: constants.ObjectTypeUser, JSONPaths: []string{"actor"}}, cfg.Fields[1])
	assert.Equal(t, IncludeField{Key: "actor.role", ObjectType: constants.ObjectTypeRole, JSONPaths: []string{"actor.role"}}, cfg.Fields[2])
	assert.Equal(t, IncludeField{Key: "query_json", ObjectType: constants.ObjectTypeRequestLog, JSONPaths: []string{"query_json"}}, cfg.Fields[3])
	assert.Equal(t, IncludeField{Key: "request_body_json", ObjectType: constants.ObjectTypeRequestLog, JSONPaths: []string{"request_body_json"}}, cfg.Fields[4])
	assert.Equal(t, IncludeField{Key: "response_body_json", ObjectType: constants.ObjectTypeRequestLog, JSONPaths: []string{"response_body_json"}}, cfg.Fields[5])
}

func TestIncludesFor_Sandbox(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeSandbox,
		Fields:     []string{"owner_account"},
	})
	require.Len(t, cfg.Fields, 1)
	assert.Equal(t, IncludeField{Key: "owner_account", ObjectType: constants.ObjectTypeAccount, JSONPaths: []string{"owner_account"}}, cfg.Fields[0])
}

func TestIncludesFor_PanicsOnUnknownField(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		IncludesFor(IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{"nonexistent"},
		})
	})
}

func TestIncludesFor_PanicsOnEmptyFields(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		IncludesFor(IncludesParams{
			ObjectType: constants.ObjectTypeAPIKey,
			Fields:     []string{},
		})
	})
}

func TestIncludesFor_PanicsOnUnregisteredType(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		IncludesFor(IncludesParams{
			ObjectType: constants.ObjectType("nonexistent"),
			Fields:     []string{"anything"},
		})
	})
}

func TestIncludesFor_DotNotation_Composable(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAgentRun,
		Fields:     []string{"definition", "definition.role"},
	})
	require.Len(t, cfg.Fields, 2)
	assert.Equal(t, "definition", cfg.Fields[0].Key)
	assert.Equal(t, []string{"definition"}, cfg.Fields[0].JSONPaths)
	assert.Equal(t, "definition.role", cfg.Fields[1].Key)
	assert.Equal(t, []string{"definition.role"}, cfg.Fields[1].JSONPaths)
}

func TestIncludesFor_AgentRun_NestedDefinitionFields(t *testing.T) {
	t.Parallel()
	cfg := IncludesFor(IncludesParams{
		ObjectType: constants.ObjectTypeAgentRun,
		Fields:     []string{"actions", "definition", "definition.config", "definition.tools", "definition.role"},
	})
	require.Len(t, cfg.Fields, 5)

	assert.Equal(t, "actions", cfg.Fields[0].Key)
	assert.Equal(t, constants.ObjectTypeAgentAction, cfg.Fields[0].ObjectType)

	assert.Equal(t, "definition", cfg.Fields[1].Key)
	assert.Equal(t, constants.ObjectTypeAgentDefinition, cfg.Fields[1].ObjectType)
	assert.Equal(t, []string{"definition"}, cfg.Fields[1].JSONPaths)

	assert.Equal(t, "definition.config", cfg.Fields[2].Key)
	assert.Equal(t, constants.ObjectTypeAgentDefinition, cfg.Fields[2].ObjectType)
	assert.Equal(t, []string{"definition.config"}, cfg.Fields[2].JSONPaths)

	assert.Equal(t, "definition.tools", cfg.Fields[3].Key)
	assert.Equal(t, constants.ObjectTypeAgentDefinitionTool, cfg.Fields[3].ObjectType)
	assert.Equal(t, []string{"definition.tools"}, cfg.Fields[3].JSONPaths)

	assert.Equal(t, "definition.role", cfg.Fields[4].Key)
	assert.Equal(t, constants.ObjectTypeRole, cfg.Fields[4].ObjectType)
	assert.Equal(t, []string{"definition.role"}, cfg.Fields[4].JSONPaths)
}
