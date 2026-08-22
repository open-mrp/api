//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// /v1/ai/tools + /v1/ai/tool-groups — read-only, code-defined tool catalog
// served by services/api-gateway/endpoints/agent-tools. Both routes share one
// gRPC-backed service and one presenter file (pkg/resource/agent_tool_resource.go),
// so they are covered together here per TASK-ai_tools.md. There is no
// create/update/delete/action route in this package (crudLifecycle/actions
// are n/a), and the resource has no timestamps (responseShape's timestamp
// checks are n/a). `aiToolsPath` is declared in crud_agent_tools_behavioral_test.go
// and `toolGroupsPath` in crud_tool_groups_test.go — both already cover the
// expandable-fields-null/populated basics for ToolGroup.tools, so this file
// focuses on the previously-unasserted fields, targeted query-param
// validation, the two 403 auth paths, and two newly-discovered cursor bugs
// (see the prodBugSuspect-tagged tests below).
// ──────────────────────────────────────────────

// covAiToolsAdminGatedSlug is a stable admin-role-gated endpoint tool used to
// pin an exact non-null required_role_type assertion.
const covAiToolsAdminGatedSlug = "create_account_integration"

// ──────────────────────────────────────────────
// AvailableTool — response fields
// ──────────────────────────────────────────────

// TestCovAiTools_ToolsAllFieldsAsserted exercises every json field of
// AvailableTool (object, slug, category, name, description, config_schema,
// required_permissions, required_role_type, mutating) across the full live
// catalog, confirming the documented shapes: category is one of built_in/
// api_endpoint, required_permissions is coerced to [] rather than left null,
// config_schema is always present as a (possibly empty) object, and both a
// non-null and a null required_role_type occur.
func TestCovAiTools_ToolsAllFieldsAsserted(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(list.Data), 100, "the catalog should exceed the default page size of 100")

	var sawAdminRoleType, sawNullRoleType bool
	var sawBuiltIn, sawEndpoint bool
	var sawMutatingTrue, sawMutatingFalse bool
	var sawNonEmptyRequiredPermissions bool

	for _, raw := range list.Data {
		tool := parseJSON(raw)
		assertObjectField(t, tool, "available_tool")
		assert.NotEmpty(t, jsonField(tool, "slug"))

		category := jsonField(tool, "category")
		assert.Contains(t, []string{"built_in", "api_endpoint"}, category, "unexpected category %q", category)
		if category == "built_in" {
			sawBuiltIn = true
		}
		if category == "api_endpoint" {
			sawEndpoint = true
		}

		assert.NotEmpty(t, jsonField(tool, "name"))
		assert.NotEmpty(t, jsonField(tool, "description"), "every catalog tool currently has a non-null, non-empty description")

		configSchema, hasConfigSchema := tool["config_schema"]
		assert.True(t, hasConfigSchema, "config_schema key should be present")
		assert.NotNil(t, configSchema, "config_schema is currently always a (possibly-empty) object, never null")

		perms := jsonArray(tool, "required_permissions")
		assert.NotNil(t, perms, "required_permissions should be coerced to [] rather than left null")
		if len(perms) > 0 {
			sawNonEmptyRequiredPermissions = true
		}

		if roleType, present := tool["required_role_type"]; present && roleType != nil {
			assert.Equal(t, "admin", roleType, "the only non-null required_role_type value in the current catalog is admin")
			sawAdminRoleType = true
		} else {
			sawNullRoleType = true
		}

		mutating, ok := tool["mutating"].(bool)
		require.True(t, ok, "mutating should be a JSON boolean")
		if mutating {
			sawMutatingTrue = true
		} else {
			sawMutatingFalse = true
		}
	}

	assert.True(t, sawBuiltIn, "at least one built_in tool should exist")
	assert.True(t, sawEndpoint, "at least one api_endpoint tool should exist")
	assert.True(t, sawAdminRoleType, "at least one tool should have required_role_type == admin")
	assert.True(t, sawNullRoleType, "at least one tool should have a null required_role_type")
	assert.True(t, sawMutatingTrue, "at least one tool should have mutating == true")
	assert.True(t, sawMutatingFalse, "at least one tool should have mutating == false")
	assert.True(t, sawNonEmptyRequiredPermissions, "at least one tool should carry non-empty required_permissions")
}

// TestCovAiTools_ToolsKnownAdminGatedEndpointTool pins exact field values for
// a stable admin-role-gated api_endpoint tool, including the required_role_type
// == "admin" value that TestCovAiTools_ToolsAllFieldsAsserted only checks
// presence-wise across the whole catalog.
func TestCovAiTools_ToolsKnownAdminGatedEndpointTool(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"q": {"Create Account Integration"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	tool := parseJSON(list.Data[0])
	assertObjectField(t, tool, "available_tool")
	assert.Equal(t, covAiToolsAdminGatedSlug, jsonField(tool, "slug"))
	assert.Equal(t, "api_endpoint", jsonField(tool, "category"))
	assert.Equal(t, "Create Account Integration", jsonField(tool, "name"))
	assert.NotEmpty(t, jsonField(tool, "description"))
	assert.Equal(t, "admin", jsonField(tool, "required_role_type"))
	assert.Equal(t, true, tool["mutating"])
	assert.Equal(t, []any{}, tool["required_permissions"])
}

// TestCovAiTools_ToolsKnownBuiltInNonMutatingTool pins exact field values for
// a stable non-mutating built-in tool: null required_role_type, empty
// required_permissions, mutating == false.
func TestCovAiTools_ToolsKnownBuiltInNonMutatingTool(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"q": {"Create Artifact"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	tool := parseJSON(list.Data[0])
	assert.Equal(t, "create_artifact", jsonField(tool, "slug"))
	assert.Equal(t, "built_in", jsonField(tool, "category"))
	assert.Equal(t, false, tool["mutating"])
	assertNilField(t, tool, "required_role_type")
	assert.Equal(t, []any{}, tool["required_permissions"])
}

// TestCovAiTools_ToolsBuiltInToolCanBeMutating locks in the actual live
// behavior against the AvailableTool.mutating doc comment's claim of "always
// false for built_in tools": send_email is category built_in and mutating ==
// true (it sends a real external email). This isn't an API defect — the doc
// comment is simply stale — but it's worth pinning explicitly so nobody
// "fixes" the live data to match the comment instead of the other way around.
func TestCovAiTools_ToolsBuiltInToolCanBeMutating(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"q": {"Send Email"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	tool := parseJSON(list.Data[0])
	assert.Equal(t, "send_email", jsonField(tool, "slug"))
	assert.Equal(t, "built_in", jsonField(tool, "category"))
	assert.Equal(t, true, tool["mutating"])
}

// ──────────────────────────────────────────────
// AvailableTool — q search
// ──────────────────────────────────────────────

// TestCovAiTools_ToolsSearchQCaseInsensitiveSubstring confirms q performs a
// case-insensitive substring match against tool DisplayName, pinning the
// exact closed result set for "customer".
func TestCovAiTools_ToolsSearchQCaseInsensitiveSubstring(t *testing.T) {
	t.Parallel()

	expectedSlugs := []string{
		"analyze_customer_pricing", "create_customer", "delete_customer",
		"list_customers", "merge_customers", "retrieve_customer",
		"retrieve_customer_lead_time", "update_customer",
	}

	for _, q := range []string{"customer", "CUSTOMER", "CuStOmEr"} {
		list, status, err := apiClient.GetList(aiToolsPath, url.Values{"q": {q}})
		require.NoError(t, err)
		require.Equal(t, 200, status)
		require.Len(t, list.Data, len(expectedSlugs), "q=%q", q)

		var got []string
		for _, raw := range list.Data {
			got = append(got, jsonField(parseJSON(raw), "slug"))
		}
		assert.ElementsMatch(t, expectedSlugs, got, "q=%q", q)
	}
}

func TestCovAiTools_ToolsSearchNoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"q": {"zzzznomatchresult999"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

func TestCovAiTools_ToolsSearchEmptyIsNoOp(t *testing.T) {
	t.Parallel()

	withQ, status, err := apiClient.GetList(aiToolsPath, url.Values{"q": {""}, "limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	withoutQ, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	assert.Equal(t, len(withoutQ.Data), len(withQ.Data), "an empty q should not filter anything")
}

func TestCovAiTools_ToolsQueryTooLong(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(aiToolsPath, url.Values{"q": {strings.Repeat("a", 501)}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "q")
}

// ──────────────────────────────────────────────
// AvailableTool — limit validation
// ──────────────────────────────────────────────

func TestCovAiTools_ToolsLimitValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		limit string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"tooLarge", "1001"},
		{"wayTooLarge", "999999"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(aiToolsPath, url.Values{"limit": {tc.limit}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
			assertErrorParam(t, errObj, "limit")
		})
	}
}

// TestCovAiTools_ToolsLimitTruncatesPageInfoStaysEmpty pins prodBugSuspect #2
// from TASK-ai_tools.md: the gateway's ListTools handler constructs
// apiresource.NewList with a zero-value apiresource.PageInfo{}, so even
// though limit demonstrably truncates the response body (the catalog has 195
// tools, far more than the requested 5), has_next_page/next_page_url never
// reflect that more data exists. Flagged, not fixed — this locks in the
// current (surprising) behavior as an explicit regression trip-wire.
func TestCovAiTools_ToolsLimitTruncatesPageInfoStaysEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 5)

	assert.False(t, list.PageInfo.HasNextPage, "prodBugSuspect: has_next_page stays false even though limit truncated a 195-row catalog")
	assert.Nil(t, list.PageInfo.NextPageURL)
	assert.False(t, list.PageInfo.HasPrevPage)
	assert.Nil(t, list.PageInfo.PreviousPageURL)
}

// ──────────────────────────────────────────────
// AvailableTool — cursor
// ──────────────────────────────────────────────

// TestCovAiTools_ToolsCursorAdvancesDataButPageInfoStaysEmpty confirms the
// literal-slug cursor documented in TASK-ai_tools.md §1 actually advances the
// data returned (agent_definition_service.go's cursor-scan does slice
// `results` forward), even though (per the previous test) the response's
// page_info metadata never reflects it. The cursor item is discovered
// dynamically from a full first-page fetch rather than hardcoded, since the
// catalog's ordering is code-defined and could shift.
func TestCovAiTools_ToolsCursorAdvancesDataButPageInfoStaysEmpty(t *testing.T) {
	t.Parallel()

	full, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(full.Data), 10, "the catalog should have at least 10 tools")

	cursorSlug := jsonField(parseJSON(full.Data[4]), "slug")
	require.NotEmpty(t, cursorSlug)

	var expectedNext []string
	for _, raw := range full.Data[5:8] {
		expectedNext = append(expectedNext, jsonField(parseJSON(raw), "slug"))
	}

	page, status, err := apiClient.GetList(aiToolsPath, url.Values{"cursor": {cursorSlug}, "limit": {"3"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, page.Data, 3)

	var gotSlugs []string
	for _, raw := range page.Data {
		gotSlugs = append(gotSlugs, jsonField(parseJSON(raw), "slug"))
	}
	assert.Equal(t, expectedNext, gotSlugs, "cursor=<slug> should resume the catalog scan immediately after the matching item")

	assert.False(t, page.PageInfo.HasNextPage)
	assert.Nil(t, page.PageInfo.NextPageURL)
}

// TestCovAiTools_ToolsCursorInvalidRejected asserts a cursor value matching
// neither a tool slug nor a tool-group id is rejected with 400
// validation_failed, per agent_definition_service.go's explicit
// `idx == -1 && gIdx == -1` check.
func TestCovAiTools_ToolsCursorInvalidRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(aiToolsPath, url.Values{"cursor": {"not_a_real_cursor_value"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Equal(t, "Invalid pagination cursor.", errObj["message"])
}

// TestCovAiTools_ToolsCursorAcceptsCrossResourceGroupID documents a
// CONFIRMED BACKEND BUG (see confirmedBugs in the implementer output):
// agent_definition_service.go's ListAvailableTools treats a cursor as valid
// if it matches EITHER a tool slug (idx) OR a tool-group id (gIdx) — but
// /v1/ai/tools only ever paginates tools. Passing a real tool-group id as the
// tools cursor should be rejected the same way TestCovAiTools_ToolsCursorInvalidRejected's
// garbage string is (neither idx nor a *tools* match), yet because gIdx >= 0
// short-circuits the "not found" check, the tools slice-forward branch falls
// into its `else` arm (idx stays -1) and zeroes out `results` entirely — every
// such call silently 200s with an empty page instead of a 400. This asserts
// the CORRECT/desired behavior and will fail until the backend is fixed.
func TestCovAiTools_ToolsCursorAcceptsCrossResourceGroupID(t *testing.T) {
	t.Parallel()

	groups, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, groups.Data, 1)
	groupID := jsonField(parseJSON(groups.Data[0]), "id")
	require.NotEmpty(t, groupID)

	status, body, err := apiClient.GetListRaw(aiToolsPath, url.Values{"cursor": {groupID}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Equal(t, "Invalid pagination cursor.", errObj["message"])
}

// TestCovAiTools_ToolsIncludeParamRejected resolves prodBugSuspect #3 from
// TASK-ai_tools.md: ListToolsEndpoint (unlike ListToolGroupsEndpoint) has no
// IncludeConfig wired, so ?include=tools on the flat route is rejected the
// same way any other undeclared query parameter would be.
func TestCovAiTools_ToolsIncludeParamRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(aiToolsPath, url.Values{"include": {"tools"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "include")
}

func TestCovAiTools_ToolsUnknownQueryParamRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(aiToolsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, aiToolsPath, status, body)
}

// ──────────────────────────────────────────────
// ToolGroup — response fields
// ──────────────────────────────────────────────

// TestCovAiTools_ToolGroupsAllFieldsAsserted exercises id, object, name,
// slug, icon, and sort_order presence/shape across the full live catalog
// (tools is asserted null here since no ?include is set; the populated case
// is already covered by TestToolGroups_IncludeTools in crud_tool_groups_test.go).
func TestCovAiTools_ToolGroupsAllFieldsAsserted(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data)

	for _, raw := range list.Data {
		group := parseJSON(raw)
		assert.NotEmpty(t, jsonField(group, "id"))
		assertIDFormat(t, jsonField(group, "id"), "tgrp")
		assertObjectField(t, group, "tool_group")
		assert.NotEmpty(t, jsonField(group, "name"))
		assert.NotEmpty(t, jsonField(group, "slug"))

		// icon/sort_order carry no validate:"required" tag and may legitimately
		// be zero-valued; just confirm the keys are present with the right
		// JSON types rather than requiring non-zero values.
		_, hasIcon := group["icon"]
		assert.True(t, hasIcon, "icon key should be present even if empty")
		_, hasSortOrder := group["sort_order"]
		assert.True(t, hasSortOrder, "sort_order key should be present even if zero")
		if sortOrder, ok := group["sort_order"].(float64); ok {
			assert.GreaterOrEqual(t, sortOrder, float64(0))
		}

		assertNilField(t, group, "tools")
	}
}

// TestCovAiTools_ToolGroupsDescriptionNullable asserts the nullable
// description field on ToolGroup: the current code-defined catalog (both
// builtinToolCatalogInfos and endpointToolCatalogInfos) never populates
// ToolGroupInfo.Description, so every group should show an explicit JSON
// null (via stringPtrOrNil) rather than an empty string.
func TestCovAiTools_ToolGroupsDescriptionNullable(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NotEmpty(t, list.Data)

	for _, raw := range list.Data {
		group := parseJSON(raw)
		assertNilField(t, group, "description")
	}
}

// TestCovAiTools_ToolGroupsKnownBuiltinGroups pins exact field values (id,
// name, slug, icon, sort_order) for the two stable built-in groups, which are
// defined directly in agent-service code rather than generated from the
// endpoint catalog.
func TestCovAiTools_ToolGroupsKnownBuiltinGroups(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(toolGroupsPath, url.Values{"q": {"General"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	general := parseJSON(list.Data[0])
	assert.Equal(t, "tgrp_builtin_general", jsonField(general, "id"))
	assertObjectField(t, general, "tool_group")
	assert.Equal(t, "General", jsonField(general, "name"))
	assert.Equal(t, "general", jsonField(general, "slug"))
	assert.Equal(t, "settings", jsonField(general, "icon"))
	assert.Equal(t, float64(3), general["sort_order"])
	assertNilField(t, general, "description")

	list, status, err = apiClient.GetList(toolGroupsPath, url.Values{"q": {"Knowledge"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	knowledge := parseJSON(list.Data[0])
	assert.Equal(t, "tgrp_builtin_knowledge", jsonField(knowledge, "id"))
	assert.Equal(t, "Knowledge", jsonField(knowledge, "name"))
	assert.Equal(t, "knowledge", jsonField(knowledge, "slug"))
	assert.Equal(t, "book", jsonField(knowledge, "icon"))
	assert.Equal(t, float64(4), knowledge["sort_order"])
}

// TestCovAiTools_ToolGroupsIncludeToolsCountMatchesFlatTotal fulfills the
// review objective's cross-check: summing every group's ?include=tools nested
// tools list should equal the flat /v1/ai/tools total, with each tool
// belonging to exactly one group (no duplicates, no orphans).
func TestCovAiTools_ToolGroupsIncludeToolsCountMatchesFlatTotal(t *testing.T) {
	t.Parallel()

	flat, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	flatTotal := len(flat.Data)
	require.Greater(t, flatTotal, 100, "the flat tool catalog should exceed the default page size of 100")

	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"include": {"tools"}, "limit": {"1000"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var list struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &list))
	require.NotEmpty(t, list.Data)

	seenSlugs := map[string]bool{}
	for _, raw := range list.Data {
		group := parseJSON(raw)
		for _, rawTool := range jsonListData(group, "tools") {
			tool, ok := rawTool.(map[string]any)
			require.True(t, ok)
			slug := jsonField(tool, "slug")
			require.NotEmpty(t, slug)
			assert.False(t, seenSlugs[slug], "tool %q should belong to exactly one group", slug)
			seenSlugs[slug] = true
		}
	}
	assert.Equal(t, flatTotal, len(seenSlugs),
		"summing every group's ?include=tools tools should equal the flat /v1/ai/tools total, with no duplicates and no orphans")
}

// ──────────────────────────────────────────────
// ToolGroup — q search
// ──────────────────────────────────────────────

func TestCovAiTools_ToolGroupsSearchQMatchesGroupName(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(toolGroupsPath, url.Values{"q": {"customer"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	type groupRow struct{ id, name, slug string }
	want := []groupRow{
		{"tgrp_api_customer_pricing", "Customer Pricing", "api_customer_pricing"},
		{"tgrp_api_customers", "Customers", "api_customers"},
	}
	require.Len(t, list.Data, len(want))

	got := make([]groupRow, 0, len(list.Data))
	for _, raw := range list.Data {
		group := parseJSON(raw)
		got = append(got, groupRow{jsonField(group, "id"), jsonField(group, "name"), jsonField(group, "slug")})
	}
	assert.ElementsMatch(t, want, got)
}

func TestCovAiTools_ToolGroupsSearchNoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(toolGroupsPath, url.Values{"q": {"zzzznomatchresult999"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

func TestCovAiTools_ToolGroupsQueryTooLong(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"q": {strings.Repeat("a", 501)}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "q")
}

// ──────────────────────────────────────────────
// ToolGroup — limit validation
// ──────────────────────────────────────────────

func TestCovAiTools_ToolGroupsLimitValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		limit string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"tooLarge", "1001"},
		{"wayTooLarge", "999999"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"limit": {tc.limit}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
			assertErrorParam(t, errObj, "limit")
		})
	}
}

// TestCovAiTools_ToolGroupsLimitTruncatesPageInfoStaysEmpty mirrors
// TestCovAiTools_ToolsLimitTruncatesPageInfoStaysEmpty for the groups route:
// the catalog has 51 groups, far more than a requested limit of 5, yet
// ListToolGroups also discards PageInfo (see prodBugSuspect #2).
func TestCovAiTools_ToolGroupsLimitTruncatesPageInfoStaysEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 5)

	assert.False(t, list.PageInfo.HasNextPage, "prodBugSuspect: has_next_page stays false even though limit truncated a 51-row catalog")
	assert.Nil(t, list.PageInfo.NextPageURL)
}

// ──────────────────────────────────────────────
// ToolGroup — cursor
// ──────────────────────────────────────────────

func TestCovAiTools_ToolGroupsCursorAdvancesDataButPageInfoStaysEmpty(t *testing.T) {
	t.Parallel()

	full, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(full.Data), 10, "the catalog should have at least 10 groups")

	cursorID := jsonField(parseJSON(full.Data[4]), "id")
	require.NotEmpty(t, cursorID)

	var expectedNext []string
	for _, raw := range full.Data[5:8] {
		expectedNext = append(expectedNext, jsonField(parseJSON(raw), "id"))
	}

	page, status, err := apiClient.GetList(toolGroupsPath, url.Values{"cursor": {cursorID}, "limit": {"3"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, page.Data, 3)

	var gotIDs []string
	for _, raw := range page.Data {
		gotIDs = append(gotIDs, jsonField(parseJSON(raw), "id"))
	}
	assert.Equal(t, expectedNext, gotIDs, "cursor=<id> should resume the catalog scan immediately after the matching group")

	assert.False(t, page.PageInfo.HasNextPage)
	assert.Nil(t, page.PageInfo.NextPageURL)
}

func TestCovAiTools_ToolGroupsCursorInvalidRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"cursor": {"not_a_real_cursor_value"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Equal(t, "Invalid pagination cursor.", errObj["message"])
}

// TestCovAiTools_ToolGroupsIncludeToolsCursorWipesNestedTools documents a
// second facet of the same shared-cursor-parsing bug as
// TestCovAiTools_ToolsCursorAcceptsCrossResourceGroupID (CONFIRMED BACKEND
// BUG — see confirmedBugs): /v1/ai/tool-groups forwards its own `cursor` into
// the exact same agent-service RPC (ListAvailableTools) that both paginates
// groups AND supplies the tools used to build ?include=tools's nested
// per-group tool lists. Because that shared RPC applies the cursor to the
// `results` (tools) slice as well as the groups slice, paginating the *groups*
// list with a legitimate group-id cursor (gIdx>=0, idx==-1 → `results = nil`)
// silently collapses every remaining group's nested `tools` array to empty,
// even though each group still on the page owns the same tools it always did.
// This exercises the CORRECT/desired contract — a groups-pagination cursor
// must slice only the groups page, never the per-group ?include=tools set —
// and will stay red until the RPC scopes its cursor to the resource being
// paginated (see backendPatch). Verified live: with a valid group cursor the
// groups page advances (e.g. 51→48) but the summed nested tools drop to 0.
func TestCovAiTools_ToolGroupsIncludeToolsCursorWipesNestedTools(t *testing.T) {
	t.Parallel()

	full, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.GreaterOrEqual(t, len(full.Data), 10, "the catalog should have at least 10 groups")

	// A real group id (not a tool slug) is the correct cursor for the groups
	// route; picking an early group guarantees several groups remain on the page.
	cursorGroupID := jsonField(parseJSON(full.Data[2]), "id")
	require.NotEmpty(t, cursorGroupID)
	assertIDFormat(t, cursorGroupID, "tgrp")

	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"cursor": {cursorGroupID}, "include": {"tools"}, "limit": {"1000"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var list struct {
		Data []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &list))
	require.NotEmpty(t, list.Data, "groups after the cursor should still be returned")

	totalNestedTools := 0
	for _, raw := range list.Data {
		group := parseJSON(raw)
		totalNestedTools += len(jsonListData(group, "tools"))
	}
	assert.Greater(t, totalNestedTools, 0, "a groups-pagination cursor must not wipe out each remaining group's ?include=tools set")
}

func TestCovAiTools_ToolGroupsIncludeInvalidValueRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{"include": {"bogus_include_value"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "include[]")
	msg, _ := errObj["message"].(string)
	assert.Contains(t, msg, "tools")
}

func TestCovAiTools_ToolGroupsUnknownQueryParamRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(toolGroupsPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, toolGroupsPath, status, body)
}

// ──────────────────────────────────────────────
// Cross-cutting: auth (both routes)
// ──────────────────────────────────────────────

// TestCovAiTools_RequiresAuthBothRoutes asserts both routes require
// authentication: a request with an empty bearer token (but valid
// OpenMRP-Version/OpenMRP-Account headers, so the request reaches auth
// middleware rather than failing an earlier header check) is rejected with
// 401 invalid_credentials.
func TestCovAiTools_RequiresAuthBothRoutes(t *testing.T) {
	t.Parallel()

	unauth := apiClient.WithBearerToken("", SeedAccountID)
	for _, path := range []string{aiToolsPath, toolGroupsPath} {
		status, body, err := unauth.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 401, status, body)
		requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
	}
}

// TestCovAiTools_ForbiddenNonInternalActorBothRoutes covers the first of the
// two distinct 403 paths from TASK-ai_tools.md §9: both endpoints call
// identity.CheckIsInternalActor() before the permission check, so a
// non-internal actor (the customer portal API key) is rejected even though
// it's otherwise a valid, authenticated caller for this account.
func TestCovAiTools_ForbiddenNonInternalActorBothRoutes(t *testing.T) {
	t.Parallel()

	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	for _, path := range []string{aiToolsPath, toolGroupsPath} {
		status, body, err := customer.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
		errObj := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
		msg, _ := errObj["message"].(string)
		assert.Contains(t, msg, "internal user")
	}
}

// TestCovAiTools_ForbiddenMissingAgentsPermissionBothRoutes covers the second
// 403 path from TASK-ai_tools.md §9: an internal actor whose role lacks the
// `agents` permission domain (SeedScannerRoleID grants only batches/scanners/
// inventory/inventory_change_logs/inventory_logs/self) is rejected by
// identity.CheckHasPermission(Agents, Read) on both routes.
func TestCovAiTools_ForbiddenMissingAgentsPermissionBothRoutes(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-covAiTools-scanner"),
		"role_id": SeedScannerRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	info := jsonObject(created, "api_key_info")
	require.NotNil(t, info)
	keyID := jsonField(info, "id")
	require.NotEmpty(t, keyID)
	secret := jsonField(created, "api_key_secret")
	require.NotEmpty(t, secret)
	defer apiClient.Delete(apiKeysPath + "/" + keyID)

	scoped := apiClient.WithBearerToken(secret, SeedAccountID)
	for _, path := range []string{aiToolsPath, toolGroupsPath} {
		status, body, err := scoped.GetListRaw(path, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
		errObj := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
		msg, _ := errObj["message"].(string)
		assert.Contains(t, msg, "agents:read")
	}
}

// TestCovAiTools_NoAccountScopingBothRoutes confirms the code-defined catalog
// is not account-scoped: tenant B (a different account) sees the same total
// row counts as the primary test account on both routes.
func TestCovAiTools_NoAccountScopingBothRoutes(t *testing.T) {
	t.Parallel()

	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	toolsA, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	toolsB, status, err := tenantB.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, len(toolsA.Data), len(toolsB.Data), "the code-defined tool catalog is not account-scoped")

	groupsA, status, err := apiClient.GetList(toolGroupsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	groupsB, status, err := tenantB.GetList(toolGroupsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, len(groupsA.Data), len(groupsB.Data), "the code-defined tool-group catalog is not account-scoped")
}
