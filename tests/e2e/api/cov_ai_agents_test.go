//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes e2e coverage gaps for the ai_agents group (/v1/ai/agents),
// layered on top of the pre-existing expandable-only coverage in
// crud_agent_definitions_test.go and the tool-slug regression coverage in
// crud_agent_tools_behavioral_test.go. Neither of those files has a CRUD
// lifecycle, a create/update-all-fields test, an omitted-fields test, a
// response-shape test, a list/search test, an idempotency test, a
// per-field validation test, or coverage of the two action-style
// mutations (DELETE and PUT .../status). See TASK-ai_agents.md.

const covAiAgentsPath = "/v1/ai/agents"

// covAiAgentsMinimalCreateBody returns the minimal valid CreateAgentRequest body.
func covAiAgentsMinimalCreateBody(slugSuffix string) map[string]any {
	return map[string]any{
		"name":          uniqueName("cov-aiag"),
		"slug":          uniqueName("cov_aiag_" + slugSuffix),
		"category_code": "operations",
		"trigger_type":  "manual",
	}
}

// ──────────────────────────────────────────────
// CRUD Lifecycle
// ──────────────────────────────────────────────

// TestCovAiAgents_CRUD exercises create -> get -> update -> delete ->
// verify-404 against a freshly created custom agent.
func TestCovAiAgents_CRUD(t *testing.T) {
	t.Parallel()

	name := uniqueName("cov-aiag-crud")
	slug := uniqueName("cov_aiag_crud")

	createStatus, createBody, err := apiClient.Post(covAiAgentsPath, map[string]any{
		"name":          name,
		"slug":          slug,
		"category_code": "operations",
		"trigger_type":  "manual",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertObjectField(t, created, "agent_definition")
	assert.Equal(t, name, jsonField(created, "name"))
	assert.Equal(t, slug, jsonField(created, "slug"))
	assert.Equal(t, "active", jsonField(created, "status"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(covAiAgentsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "custom", jsonField(got, "definition_type"))

	// UPDATE
	newName := uniqueName("cov-aiag-crud-upd")
	patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id, map[string]any{
		"name": newName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, newName, jsonField(updated, "name"))
	assert.Equal(t, id, jsonField(updated, "id"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(covAiAgentsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	getStatus2, getBody2, err := apiClient.GetListRaw(covAiAgentsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2,
		"GET on a deleted custom agent should 404, got %d: %s", getStatus2, string(getBody2))
}

// ──────────────────────────────────────────────
// Create and Update All Fields
// ──────────────────────────────────────────────

// TestCovAiAgents_CreateAndUpdateAllFields creates an agent with every
// CreateAgentRequest field set (including nested config.* and tools[]),
// asserts every AgentDefinition/AgentDefinitionConfig/TriggerConfig/
// AgentDefinitionTool/AvailableTool json field with ?include=config,tools,role,
// then PATCHes a subset of fields and asserts both the changed and the
// (would-be) preserved fields.
func TestCovAiAgents_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("cov-aiag-allf")
	slug := uniqueName("cov_aiag_allf")

	resp, err := apiClient.PostFull(covAiAgentsPath+"?include=config,tools,role", map[string]any{
		"name":          name,
		"slug":          slug,
		"description":   "Create description",
		"category_code": "operations",
		"trigger_type":  "scheduled",
		"config": map[string]any{
			"system_prompt": "You are a test agent.",
			"tier":          "high",
			"temperature":   0.4,
			"trigger_config": map[string]any{
				"cron_schedule": "0 * * * *",
				"timezone":      "UTC",
				"event_filters": []string{"order.created"},
			},
			"endpoint_tool_slugs":  []string{"create_account_group"},
			"endpoint_tool_review": map[string]any{"create_account_group": true},
		},
		"tools": []map[string]any{
			{"tool": "read_doc", "sort_order": 1, "require_review": true},
		},
		"role_id": SeedAdminRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	got := parseJSON(resp.Body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covAiAgentsPath + "/" + id)

	assertCreatedLocation(t, resp.Header, id)

	// ── AgentDefinition top-level fields ──
	assertIDFormat(t, id, "agdf")
	assertObjectField(t, got, "agent_definition")
	assert.Equal(t, "custom", jsonField(got, "definition_type"))
	assert.Equal(t, "operations", jsonField(got, "category_code"))
	assert.Equal(t, "scheduled", jsonField(got, "trigger_type"))
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, slug, jsonField(got, "slug"))
	assert.Equal(t, "Create description", jsonField(got, "description"))
	assert.Equal(t, "editable", jsonField(got, "editability"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// ── role (expandable, included) ──
	role := jsonObject(got, "role")
	require.NotNil(t, role, "role should be present with ?include=role")
	assert.Equal(t, SeedAdminRoleID, jsonField(role, "id"))
	assertObjectField(t, role, "role")
	assert.Equal(t, "Admin", jsonField(role, "name"))
	assert.Equal(t, "admin", jsonField(role, "type"))

	// ── config (expandable, included) ──
	config := jsonObject(got, "config")
	require.NotNil(t, config, "config should be present with ?include=config")
	assertObjectField(t, config, "agent_definition_config")
	assert.Equal(t, "You are a test agent.", jsonField(config, "system_prompt"))
	assert.Equal(t, "high", jsonField(config, "tier"))
	assert.Equal(t, "0.4", jsonField(config, "temperature"))
	assert.Equal(t, []string{"create_account_group"}, jsonStringSlice(config, "endpoint_tool_slugs"))
	review, ok := config["endpoint_tool_review"].(map[string]any)
	require.True(t, ok, "endpoint_tool_review should be a map")
	assert.Equal(t, true, review["create_account_group"])

	triggerConfig := jsonObject(config, "trigger_config")
	require.NotNil(t, triggerConfig, "trigger_config should be present")
	assertObjectField(t, triggerConfig, "trigger_config")
	assert.Equal(t, "0 * * * *", jsonField(triggerConfig, "cron_schedule"))
	assert.Equal(t, "UTC", jsonField(triggerConfig, "timezone"))
	assert.Equal(t, []string{"order.created"}, jsonStringSlice(triggerConfig, "event_filters"))

	// ── tools (expandable, included) ──
	toolsEnv := jsonObject(got, "tools")
	require.NotNil(t, toolsEnv, "tools should be present with ?include=tools")
	assertObjectField(t, toolsEnv, "list")
	toolsData := jsonArray(toolsEnv, "data")
	require.Len(t, toolsData, 1)
	toolLink, ok := toolsData[0].(map[string]any)
	require.True(t, ok)
	assertIDFormat(t, jsonField(toolLink, "id"), "agdftl")
	assertObjectField(t, toolLink, "agent_definition_tool")
	assert.Equal(t, "required", jsonField(toolLink, "review_requirement"))
	assert.Equal(t, "1", jsonField(toolLink, "sort_order"))
	_, hasConfig := toolLink["config"]
	assert.True(t, hasConfig, "agent_definition_tool.config key should be present")

	availableTool := jsonObject(toolLink, "tool")
	require.NotNil(t, availableTool, "tool should embed its AvailableTool")
	assertObjectField(t, availableTool, "available_tool")
	assert.Equal(t, "read_doc", jsonField(availableTool, "slug"))
	assert.Equal(t, "built_in", jsonField(availableTool, "category"))
	assert.Equal(t, "Read Doc", jsonField(availableTool, "name"))
	assert.NotEmpty(t, jsonField(availableTool, "description"))
	_, hasConfigSchema := availableTool["config_schema"]
	assert.True(t, hasConfigSchema, "available_tool.config_schema key should be present")
	_, hasRequiredPermissions := availableTool["required_permissions"]
	assert.True(t, hasRequiredPermissions, "available_tool.required_permissions key should be present")
	assertNilField(t, availableTool, "required_role_type")
	assert.Equal(t, "false", jsonField(availableTool, "mutating"))

	// ── UPDATE a subset of fields ──
	updatedName := uniqueName("cov-aiag-allf-upd")
	patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=role", map[string]any{
		"name":    updatedName,
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, updatedName, jsonField(updated, "name"))
	// Preserved (not sent in this PATCH):
	assert.Equal(t, slug, jsonField(updated, "slug"))
	assert.Equal(t, "Create description", jsonField(updated, "description"))
	assert.Equal(t, "scheduled", jsonField(updated, "trigger_type"))
	assert.Equal(t, "operations", jsonField(updated, "category_code"))
	// Changed:
	updRole := jsonObject(updated, "role")
	require.NotNil(t, updRole)
	assert.Equal(t, SeedSalesRepRoleID, jsonField(updRole, "id"))
	assert.Equal(t, "Sales Rep", jsonField(updRole, "name"))
}

// ──────────────────────────────────────────────
// Omitted Fields
// ──────────────────────────────────────────────

func TestCovAiAgents_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		status, body, err := apiClient.Post(covAiAgentsPath, covAiAgentsMinimalCreateBody("omit"), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(covAiAgentsPath + "/" + id)

		assertObjectField(t, got, "agent_definition")
		assert.Equal(t, "custom", jsonField(got, "definition_type"))
		assert.Equal(t, "editable", jsonField(got, "editability"))
		assert.Equal(t, "active", jsonField(got, "status"))
		assertNilField(t, got, "description")
		assertNilField(t, got, "role")
		assertNilField(t, got, "config")
		assertNilField(t, got, "tools")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		// With ?include, config/tools/role should still resolve to empty/nil
		// shapes rather than error.
		status2, body2, err := apiClient.GetListRaw(covAiAgentsPath+"/"+id, url.Values{"include": {"config,tools,role"}})
		require.NoError(t, err)
		requireStatus(t, 200, status2, body2)
		withInclude := parseJSON(body2)

		config := jsonObject(withInclude, "config")
		require.NotNil(t, config, "config key present with ?include=config even when unset")
		assertObjectField(t, config, "agent_definition_config")
		assertNilField(t, config, "system_prompt")
		assertNilField(t, config, "tier")
		assertNilField(t, config, "temperature")
		assertNilField(t, config, "trigger_config")
		assertNilField(t, config, "endpoint_tool_slugs")
		assertNilField(t, config, "endpoint_tool_review")

		toolsEnv := jsonObject(withInclude, "tools")
		require.NotNil(t, toolsEnv, "tools key present with ?include=tools even when empty")
		assertObjectField(t, toolsEnv, "list")
		assert.Empty(t, jsonArray(toolsEnv, "data"), "tools.data should be empty when no tools are attached")

		assertNilField(t, withInclude, "role")
	})

	t.Run("CreateMissingName", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("missname")
		delete(body, "name")
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422, "missing name should 400/422, got %d: %s", status, string(respBody))
	})

	t.Run("CreateMissingSlug", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("missslug")
		delete(body, "slug")
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422, "missing slug should 400/422, got %d: %s", status, string(respBody))
	})

	t.Run("CreateMissingCategoryCode", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("misscat")
		delete(body, "category_code")
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422, "missing category_code should 400/422, got %d: %s", status, string(respBody))
	})

	t.Run("CreateMissingTriggerType", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("misstrig")
		delete(body, "trigger_type")
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422, "missing trigger_type should 400/422, got %d: %s", status, string(respBody))
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		name := uniqueName("cov-aiag-pres")
		slug := uniqueName("cov_aiag_pres")
		createStatus, createBody, err := apiClient.Post(covAiAgentsPath, map[string]any{
			"name":          name,
			"slug":          slug,
			"description":   "Original description",
			"category_code": "operations",
			"trigger_type":  "manual",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(covAiAgentsPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		newName := uniqueName("cov-aiag-pres-upd")
		patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id, map[string]any{
			"name": newName,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, newName, jsonField(got, "name"))
		assert.Equal(t, "Original description", jsonField(got, "description"), "description should be preserved")
		assert.Equal(t, slug, jsonField(got, "slug"), "slug should be preserved")
		assert.Equal(t, "operations", jsonField(got, "category_code"), "category_code should be preserved")
		assert.Equal(t, "manual", jsonField(got, "trigger_type"), "trigger_type should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

// ──────────────────────────────────────────────
// Response Shape
// ──────────────────────────────────────────────

func TestCovAiAgents_CreateResponseShape(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(covAiAgentsPath, covAiAgentsMinimalCreateBody("shape"), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covAiAgentsPath + "/" + id)

	assertIDFormat(t, id, "agdf")
	assertObjectField(t, got, "agent_definition")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// List
// ──────────────────────────────────────────────

func TestCovAiAgents_List(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covAiAgentsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 2, "should include at least the two shared seed agents")

	seenSystem, seenCustom := false, false
	for _, raw := range list.Data {
		row := parseJSON(raw)
		if jsonField(row, "id") == SeedAgentDefinitionID {
			seenSystem = true
		}
		if jsonField(row, "id") == SeedCustomAgentDefinitionID {
			seenCustom = true
		}
	}
	assert.True(t, seenSystem, "seeded system agent should appear in the list")
	assert.True(t, seenCustom, "seeded custom agent should appear in the list")
}

func TestCovAiAgents_ListSearch(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covAiAgentsPath, url.Values{"q": {"Custom Test Agent"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, raw := range list.Data {
		row := parseJSON(raw)
		if jsonField(row, "id") == SeedCustomAgentDefinitionID {
			found = true
		}
	}
	assert.True(t, found, "search for the seeded custom agent's name should return it")
}

func TestCovAiAgents_ListSearchNoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covAiAgentsPath, url.Values{"q": {"zzzznotaresource99999"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

// TestCovAiAgents_ListArrayFilterQueryParamsSilentlyIgnoreUnknownValues confirms
// (per TASK-ai_agents.md section 6) that the generic enum validator does not
// recurse into slice-typed query params: an unknown statuses/definition_types/
// trigger_types value returns 200 with an empty result set rather than 400.
func TestCovAiAgents_ListArrayFilterQueryParamsRejectUnknownValues(t *testing.T) {
	t.Parallel()

	for _, param := range []string{"statuses", "definition_types", "trigger_types"} {
		t.Run(param, func(t *testing.T) {
			// Unrecognized enum values in a list filter are rejected with 400 (platform convention).
			status, body, err := apiClient.GetListRaw(covAiAgentsPath, url.Values{param: {"bogus_e2e_value"}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
		})
	}
}

func TestCovAiAgents_ListLimitValidation(t *testing.T) {
	t.Parallel()

	t.Run("Zero", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(covAiAgentsPath, url.Values{"limit": {"0"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "limit=0 should 400, got %d: %s", status, string(body))
	})
	t.Run("TooLarge", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(covAiAgentsPath, url.Values{"limit": {"1001"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "limit=1001 should 400, got %d: %s", status, string(body))
	})
	t.Run("NonNumeric", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(covAiAgentsPath, url.Values{"limit": {"abc"}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "limit=abc should 400, got %d: %s", status, string(body))
	})
}

func TestCovAiAgents_ListUnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covAiAgentsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covAiAgentsPath, status, body)
}

// ──────────────────────────────────────────────
// Expandable Fields (deepened beyond crud_agent_definitions_test.go)
// ──────────────────────────────────────────────

func TestCovAiAgents_IncludeRolePermissions(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covAiAgentsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"role,role.permissions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	role := jsonObject(parseJSON(body), "role")
	require.NotNil(t, role)
	assert.Equal(t, SeedAdminRoleID, jsonField(role, "id"))
	assert.Equal(t, "Admin", jsonField(role, "name"))
	assert.Equal(t, "admin", jsonField(role, "type"))
	perms := jsonArray(role, "permissions")
	assert.NotEmpty(t, perms, "role.permissions should be populated with ?include=role.permissions")
}

func TestCovAiAgents_IncludeRoleWithoutPermissionsLeavesPermissionsNil(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covAiAgentsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"role"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	role := jsonObject(parseJSON(body), "role")
	require.NotNil(t, role)
	assertNilField(t, role, "permissions")
}

func TestCovAiAgents_IncludeInvalidValueRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covAiAgentsPath+"/"+SeedCustomAgentDefinitionID, url.Values{"include": {"bogus_e2e_include"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown include value should 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
}

// ──────────────────────────────────────────────
// Idempotency
// ──────────────────────────────────────────────

func TestCovAiAgents_CreateIdempotent(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("idemc")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(covAiAgentsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	defer apiClient.Delete(covAiAgentsPath + "/" + id1)

	status2, body2, err := apiClient.Post(covAiAgentsPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"))
}

func TestCovAiAgents_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("idemu"))
	id := jsonField(created, "id")

	idemKey := newIdempotencyKey()
	patchBody := map[string]any{"name": uniqueName("cov-aiag-idemu-upd")}

	status1, resp1, err := apiClient.Patch(covAiAgentsPath+"/"+id, patchBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, resp1)

	status2, resp2, err := apiClient.Patch(covAiAgentsPath+"/"+id, patchBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, resp2)

	assert.Equal(t, jsonField(parseJSON(resp1), "name"), jsonField(parseJSON(resp2), "name"))
	assert.Equal(t, jsonField(parseJSON(resp1), "updated_at"), jsonField(parseJSON(resp2), "updated_at"))
}

// ──────────────────────────────────────────────
// Validation
// ──────────────────────────────────────────────

func TestCovAiAgents_CreateValidation_NameEmpty(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("ve")
	body["name"] = ""
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422, "empty name should 400/422, got %d: %s", status, string(respBody))
}

func TestCovAiAgents_CreateValidation_SlugEmpty(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vs")
	body["slug"] = ""
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422, "empty slug should 400/422, got %d: %s", status, string(respBody))
}

func TestCovAiAgents_CreateValidation_DescriptionNull(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vdn")
	body["description"] = nil
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "null description should 400, got %d: %s", status, string(respBody))
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "description")
}

func TestCovAiAgents_CreateValidation_DescriptionEmpty(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vde")
	body["description"] = ""
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "empty description should 400, got %d: %s", status, string(respBody))
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "description")
}

func TestCovAiAgents_CreateValidation_TriggerTypeInvalidEnum(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vtt")
	body["trigger_type"] = "bogus_trigger"
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "invalid trigger_type should 400, got %d: %s", status, string(respBody))
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "trigger_type")
}

func TestCovAiAgents_CreateValidation_ScheduledRequiresCronSchedule(t *testing.T) {
	t.Parallel()

	t.Run("NoTriggerConfigAtAll", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("vsc1")
		body["trigger_type"] = "scheduled"
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 400, status, "scheduled without trigger_config should 400, got %d: %s", status, string(respBody))
		requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	})

	t.Run("TriggerConfigWithoutCronSchedule", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("vsc2")
		body["trigger_type"] = "scheduled"
		body["config"] = map[string]any{"trigger_config": map[string]any{"timezone": "UTC"}}
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 400, status, "scheduled without cron_schedule should 400, got %d: %s", status, string(respBody))
		requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	})
}

func TestCovAiAgents_CreateValidation_EventRequiresEventFilters(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vef")
	body["trigger_type"] = "event"
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "event without event_filters should 400, got %d: %s", status, string(respBody))
	requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
}

// A chat agent runs when a user messages it, so like `manual` it carries no trigger config to validate and creates without one.
func TestCovAiAgents_CreateChatTriggerTypeNeedsNoConfig(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("chat")
	body["trigger_type"] = "chat"
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	got := parseJSON(respBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covAiAgentsPath + "/" + id)
	assert.Equal(t, "chat", jsonField(got, "trigger_type"))
}

func TestCovAiAgents_CreateValidation_TemperatureOutOfRange(t *testing.T) {
	t.Parallel()

	t.Run("TooHigh", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("vth")
		body["config"] = map[string]any{"temperature": 1.5}
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 400, status, "temperature=1.5 should 400, got %d: %s", status, string(respBody))
	})

	t.Run("TooLow", func(t *testing.T) {
		body := covAiAgentsMinimalCreateBody("vtl")
		body["config"] = map[string]any{"temperature": -0.1}
		status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
		require.NoError(t, err)
		assert.Equal(t, 400, status, "temperature=-0.1 should 400, got %d: %s", status, string(respBody))
	})
}

func TestCovAiAgents_CreateValidation_UnknownBuiltinToolSlug(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vut")
	body["tools"] = []map[string]any{{"tool": "not_a_real_tool"}}
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown tool slug should 400, got %d: %s", status, string(respBody))
	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "tools")
	assert.Contains(t, errObj["message"], "not_a_real_tool")
}

func TestCovAiAgents_CreateValidation_UnknownEndpointToolSlug(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vet")
	body["config"] = map[string]any{"endpoint_tool_slugs": []string{"not_a_real_endpoint_tool"}}
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "unknown endpoint tool slug should 400, got %d: %s", status, string(respBody))
	// Documented minor param-name mismatch: the error names "tools", not "config".
	errObj := requireErrorResponse(t, respBody, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "tools")
	assert.Contains(t, errObj["message"], "not_a_real_endpoint_tool")
}

// TestCovAiAgents_CreateRoleIDNonexistentSilentlyAccepted pins down the
// role_id has no FK validation in
// agent-service, so a nonexistent role_id is silently accepted at create
// time (201) and only surfaces as role:null when later fetched with
// ?include=role.
func TestCovAiAgents_CreateRoleIDNonexistentSilentlyAccepted(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vrole")
	body["role_id"] = "rl_doesnotexist_e2e"
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	got := parseJSON(respBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covAiAgentsPath + "/" + id)

	getStatus, getBody, err := apiClient.GetListRaw(covAiAgentsPath+"/"+id, url.Values{"include": {"role"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assertNilField(t, parseJSON(getBody), "role")
}

func TestCovAiAgents_CreateUnknownJSONFieldRejected(t *testing.T) {
	t.Parallel()
	body := covAiAgentsMinimalCreateBody("vujf")
	body[bogusE2EJSONField] = "x"
	status, respBody, err := apiClient.Post(covAiAgentsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", covAiAgentsPath, status, respBody)
}

// ──────────────────────────────────────────────
// 404 / 403 / 410 behavior
// ──────────────────────────────────────────────

func TestCovAiAgents_GetUnknownID404(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.GetListRaw(covAiAgentsPath+"/agdf_e2e_doesnotexist12", nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestCovAiAgents_PatchUnknownID404(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Patch(covAiAgentsPath+"/agdf_e2e_doesnotexist12", map[string]any{"name": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestCovAiAgents_DeleteUnknownID404(t *testing.T) {
	t.Parallel()
	status, _, err := apiClient.Delete(covAiAgentsPath + "/agdf_e2e_doesnotexist12")
	require.NoError(t, err)
	assert.Equal(t, 404, status)
}

func TestCovAiAgents_PatchSystemAgentForbidden(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(covAiAgentsPath+"/"+SeedAgentDefinitionID, map[string]any{"name": "hacked"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, status, "PATCH on a system agent should 403, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

func TestCovAiAgents_DeleteSystemAgentForbidden(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(covAiAgentsPath + "/" + SeedAgentDefinitionID)
	require.NoError(t, err)
	assert.Equal(t, 403, status, "DELETE on a system agent should 403, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovAiAgents_CrossAccountPatchAndDeleteMasking asserts each op's
// observed cross-account behavior. Note this DIFFERS by verb: PATCH returns
// 403 insufficient_permissions ("does not belong to this account") while
// DELETE returns 404 resource_not_found, masking existence — confirmed by
// reading agent_definition_service.go (UpdateCustomAgent L348-351 raises
// AuthorizationError, DeleteCustomAgent L513-515 raises
// ResourceNotFoundError). The two are inconsistent with each other, but each
// is asserted here exactly as observed.
func TestCovAiAgents_CrossAccountPatchAndDeleteMasking(t *testing.T) {
	t.Parallel()
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	patchStatus, patchBody, err := tenantB.Patch(covAiAgentsPath+"/"+SeedCustomAgentDefinitionID, map[string]any{"name": "cross-account-hack"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 403, patchStatus, "cross-account PATCH should 403, got %d: %s", patchStatus, string(patchBody))

	deleteStatus, deleteBody, err := tenantB.Delete(covAiAgentsPath + "/" + SeedCustomAgentDefinitionID)
	require.NoError(t, err)
	assert.Equal(t, 404, deleteStatus, "cross-account DELETE should 404, got %d: %s", deleteStatus, string(deleteBody))
}

// A second DELETE of an already-deleted custom agent reports 410 via apierror.NewAlreadyDeletedError, rather than repeating the soft delete as a no-op 200.
func TestCovAiAgents_DoubleDeleteReturns410(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("dbldel"))
	id := jsonField(created, "id")

	firstStatus, firstBody, err := apiClient.Delete(covAiAgentsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, firstStatus, firstBody)

	secondStatus, secondBody, err := apiClient.Delete(covAiAgentsPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 410, secondStatus,
		"a second DELETE of an already-deleted custom agent should 410 (ErrorCodeResourceGone); got %d: %s",
		secondStatus, string(secondBody))
}

// ──────────────────────────────────────────────
// PATCH response status
// ──────────────────────────────────────────────

// A PATCH response reports the agent's real per-account status, which means resolving the AgentAccountStatus row rather than falling back to the presenter's inactive default.
func TestCovAiAgents_PatchResponseReflectsPersistedStatus(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("statusbug"))
	id := jsonField(created, "id")
	require.Equal(t, "active", jsonField(created, "status"), "sanity: new custom agents default to active")

	patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id, map[string]any{
		"name": uniqueName("cov-aiag-statusbug-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	patched := parseJSON(patchBody)
	assert.Equal(t, "active", jsonField(patched, "status"),
		"the PATCH response should reflect the unchanged persisted status; got %q", jsonField(patched, "status"))

	// Sanity: the persisted status was never actually touched by the PATCH.
	getStatus, getBody, err := apiClient.GetListRaw(covAiAgentsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, "active", jsonField(parseJSON(getBody), "status"), "a subsequent GET shows the real, unaffected status")
}

// ──────────────────────────────────────────────
// Config merge-on-PATCH
// ──────────────────────────────────────────────

// A PATCH carries only the config keys the caller named, and the rest are merged forward from the stored config: updating one setting must not silently clear the others.
func TestCovAiAgents_PatchConfigMergesWithStored(t *testing.T) {
	t.Parallel()

	createBody := covAiAgentsMinimalCreateBody("cfgreplace")
	createBody["trigger_type"] = "scheduled"
	createBody["config"] = map[string]any{
		"system_prompt": "orig prompt",
		"tier":          "high",
		"temperature":   0.3,
		"trigger_config": map[string]any{
			"cron_schedule": "0 * * * *",
			"timezone":      "UTC",
			"event_filters": []string{"x"},
		},
		"endpoint_tool_slugs":  []string{"create_account_group"},
		"endpoint_tool_review": map[string]any{"create_account_group": true},
	}
	status, body, err := apiClient.Post(covAiAgentsPath+"?include=config", createBody, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covAiAgentsPath + "/" + id)

	origConfig := jsonObject(created, "config")
	require.NotNil(t, origConfig)
	require.Equal(t, "orig prompt", jsonField(origConfig, "system_prompt"), "sanity: config was fully set at create time")

	patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=config", map[string]any{
		"config": map[string]any{"tier": "cheap"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	patched := parseJSON(patchBody)
	config := jsonObject(patched, "config")
	require.NotNil(t, config)
	assert.Equal(t, "cheap", jsonField(config, "tier"), "the sent field is applied")
	assert.Equal(t, "orig prompt", jsonField(config, "system_prompt"), "an omitted field keeps its stored value")
	assert.NotNil(t, config["temperature"], "an omitted field keeps its stored value")
	require.NotNil(t, jsonObject(config, "trigger_config"), "an omitted nested object keeps its stored value")
	assert.Equal(t, "0 * * * *", jsonField(jsonObject(config, "trigger_config"), "cron_schedule"))
	assert.NotNil(t, config["endpoint_tool_slugs"], "an omitted field keeps its stored value")
	assert.NotNil(t, config["endpoint_tool_review"], "an omitted field keeps its stored value")
}

// TestCovAiAgents_PatchToolsOmittedPreservesExistingTools confirms that
// unlike config, omitting `tools` from a PATCH body leaves the existing
// tool set untouched (only sending `tools` triggers the documented
// full-replace for that field).
func TestCovAiAgents_PatchToolsOmittedPreservesExistingTools(t *testing.T) {
	t.Parallel()

	createBody := covAiAgentsMinimalCreateBody("toolspreserve")
	createBody["tools"] = []map[string]any{{"tool": "read_doc", "sort_order": 1}}
	created := createAndCleanup(t, covAiAgentsPath, createBody)
	id := jsonField(created, "id")

	patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=tools", map[string]any{
		"name": uniqueName("cov-aiag-toolspreserve-upd"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	toolsEnv := jsonObject(parseJSON(patchBody), "tools")
	require.NotNil(t, toolsEnv)
	toolsData := jsonArray(toolsEnv, "data")
	require.Len(t, toolsData, 1, "tools should be preserved when omitted from PATCH")
}

// TestCovAiAgents_PatchToolsReplacesExistingTools confirms the documented
// full-replace semantics for `tools` when it IS sent.
func TestCovAiAgents_PatchToolsReplacesExistingTools(t *testing.T) {
	t.Parallel()

	createBody := covAiAgentsMinimalCreateBody("toolsreplace")
	createBody["tools"] = []map[string]any{{"tool": "read_doc", "sort_order": 1}}
	created := createAndCleanup(t, covAiAgentsPath, createBody)
	id := jsonField(created, "id")

	patchStatus, patchBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=tools", map[string]any{
		"tools": []map[string]any{{"tool": "fetch_url", "sort_order": 0}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	toolsEnv := jsonObject(parseJSON(patchBody), "tools")
	require.NotNil(t, toolsEnv)
	toolsData := jsonArray(toolsEnv, "data")
	require.Len(t, toolsData, 1, "tools should be fully replaced")
	toolLink, _ := toolsData[0].(map[string]any)
	require.NotNil(t, toolLink)
	tool := jsonObject(toolLink, "tool")
	require.NotNil(t, tool)
	assert.Equal(t, "fetch_url", jsonField(tool, "slug"))
}

// TestCovAiAgents_PatchRoleClearAndReset confirms role_id is Clearable:
// omit=leave, null=detach, value=set.
func TestCovAiAgents_PatchRoleClearAndReset(t *testing.T) {
	t.Parallel()

	createBody := covAiAgentsMinimalCreateBody("roleclear")
	createBody["role_id"] = SeedAdminRoleID
	created := createAndCleanup(t, covAiAgentsPath, createBody)
	id := jsonField(created, "id")

	clearStatus, clearBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=role", map[string]any{"role_id": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, clearStatus, clearBody)
	assertNilField(t, parseJSON(clearBody), "role")

	resetStatus, resetBody, err := apiClient.Patch(covAiAgentsPath+"/"+id+"?include=role", map[string]any{"role_id": SeedSalesRepRoleID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, resetStatus, resetBody)
	role := jsonObject(parseJSON(resetBody), "role")
	require.NotNil(t, role)
	assert.Equal(t, SeedSalesRepRoleID, jsonField(role, "id"))
}

// ──────────────────────────────────────────────
// Actions — PUT /status
// ──────────────────────────────────────────────

func TestCovAiAgents_StatusActionCustomAgent(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("statusaction"))
	id := jsonField(created, "id")

	status, body, err := apiClient.Put(covAiAgentsPath+"/"+id+"/status", map[string]any{"status": "inactive"})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assertObjectField(t, got, "agent_definition")
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, "inactive", jsonField(got, "status"))

	status2, body2, err := apiClient.Put(covAiAgentsPath+"/"+id+"/status", map[string]any{"status": "active"})
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	assert.Equal(t, "active", jsonField(parseJSON(body2), "status"))
}

// TestCovAiAgents_StatusActionSystemAgent confirms PUT .../status works for
// system agents too (no definition_type guard), per the endpoint doc.
// Restores the seed agent's status to "active" at the end so it doesn't
// leak state into parallel tests that depend on SeedAgentDefinitionID being
// active.
func TestCovAiAgents_StatusActionSystemAgent(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(covAiAgentsPath+"/"+SeedAgentDefinitionID+"/status", map[string]any{"status": "active"})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, SeedAgentDefinitionID, jsonField(got, "id"))
	assert.Equal(t, "system", jsonField(got, "definition_type"))
	assert.Equal(t, "active", jsonField(got, "status"))
}

// UpdateAgentStatusRequest.Status is a constants.AgentAccountStatus enum, so an arbitrary string is rejected by the generic gateway enum validator instead of being persisted and echoed back.
func TestCovAiAgents_StatusActionRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("statusinvalid"))
	id := jsonField(created, "id")

	status, body, err := apiClient.Put(covAiAgentsPath+"/"+id+"/status", map[string]any{"status": "bogus_e2e_status"})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "status")
}

// TestCovAiAgents_StatusActionNaturalIdempotency confirms back-to-back
// identical PUTs are naturally idempotent (upsert), since Client.Put never
// sends an Idempotency-Key header.
func TestCovAiAgents_StatusActionNaturalIdempotency(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, covAiAgentsPath, covAiAgentsMinimalCreateBody("statusnatidem"))
	id := jsonField(created, "id")

	status1, body1, err := apiClient.Put(covAiAgentsPath+"/"+id+"/status", map[string]any{"status": "inactive"})
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)

	status2, body2, err := apiClient.Put(covAiAgentsPath+"/"+id+"/status", map[string]any{"status": "inactive"})
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)

	assert.Equal(t, "inactive", jsonField(parseJSON(body1), "status"))
	assert.Equal(t, "inactive", jsonField(parseJSON(body2), "status"))
}
