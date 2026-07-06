//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes the e2e coverage gap for the /v1/ai/runs group (agent
// runs), which prior to this file only had four expandable-field include
// tests in crud_agent_runs_test.go and one list-include-isolation row in
// crud_include_isolation_test.go. It reuses agentRunsPath, firstAgentRunID,
// getCustomerPortalClient, and getTenantBClient declared in those sibling
// files (and tenant_isolation_test.go), and the SeedAgentDefinitionID /
// SeedCustomAgentDefinitionID / SeedInactiveAgentDefinitionID /
// SeedAgentRunID / SeedAgentRunFailedID / SeedAgentRunAwaitingInputID
// constants from seed_test.go.
//
// AgentRun is a state machine, not a mutable CRUD resource: there is no
// PATCH and no DELETE. The "update" surface is the three /actions/* verbs
// (cancel/continue/retry), and there is no delete step, so none of the
// tests below call apiClient.Delete for a run.
//
// Trigger is fully async (LLM execution happens out-of-band via an outbox
// consumer) but the DB write that creates the row (status=pending) is
// synchronous, and Cancel/Continue/Retry are synchronous status flips. Every
// assertion here reads state reachable immediately from the gateway's own
// synchronous response or an immediate follow-up call -- none of these
// tests poll for `completed`/`failed` after a live trigger, since that
// depends on real model availability/latency and is out of scope.
//
// SeedAgentRunFailedID and SeedAgentRunAwaitingInputID are dedicated,
// one-shot seed rows (continue/retry permanently flip their status) so each
// is consumed by exactly one happy-path test here, which also folds in that
// verb's idempotency assertion (replaying the same Idempotency-Key) rather
// than spending a second seed row on it.

// ──────────────────────────────────────────────
// Trigger (create) -- all fields, omitted-field defaults
// ──────────────────────────────────────────────

// TestCovAiRuns_TriggerCancelLifecycle walks the full lifecycle for a
// freshly triggered run: trigger (201, pending, every AgentRun field
// asserted) -> get (200, pending) -> cancel (200, cancelled) -> get (200,
// cancelled, completed_at/duration_ms still null since cancelling isn't a
// completion).
func TestCovAiRuns_TriggerCancelLifecycle(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(agentRunsPath, map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
		"input":               "Process the latest incoming orders.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	created := parseJSON(resp.Body)
	id := jsonField(created, "id")
	assertIDFormat(t, id, "agrn")
	assertCreatedLocation(t, resp.Header, id)
	defer func() { _, _, _ = apiClient.Post(agentRunsPath+"/"+id+"/actions/cancel", nil, newIdempotencyKey()) }()

	assertObjectField(t, created, "agent_run")
	assert.Equal(t, "manual", jsonField(created, "trigger_type"))
	assert.Equal(t, "pending", jsonField(created, "status"))
	assertNilField(t, created, "definition")
	inputObj := jsonObject(created, "input")
	require.NotNil(t, inputObj, "input should wrap the trigger input as {\"message\": ...}")
	assert.Equal(t, "Process the latest incoming orders.", inputObj["message"])
	outputObj := jsonObject(created, "output")
	require.NotNil(t, outputObj)
	assert.Empty(t, outputObj, "output should be an empty object before completion")
	assertNilField(t, created, "error_message")
	assertNilField(t, created, "triggered_by")
	assertNilField(t, created, "started_at")
	assertNilField(t, created, "completed_at")
	assertNilField(t, created, "duration_ms")
	assertNilField(t, created, "actions")
	assertNilField(t, created, "steps")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	// The runner may already have claimed the run by the time this GET lands (pending -> running), a race that surfaces under the parallel suite. Both are valid pre-cancel states; the lifecycle assertion that matters is the cancelled terminal state below.
	assert.Contains(t, []string{"pending", "running"}, jsonField(parseJSON(body), "status"))

	status, body, err = apiClient.Post(agentRunsPath+"/"+id+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	cancelled := parseJSON(body)
	assert.Equal(t, id, jsonField(cancelled, "id"))
	assert.Equal(t, "cancelled", jsonField(cancelled, "status"))

	status, body, err = apiClient.GetListRaw(agentRunsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	final := parseJSON(body)
	assert.Equal(t, "cancelled", jsonField(final, "status"))
	assertNilField(t, final, "completed_at")
	assertNilField(t, final, "duration_ms")
}

// TestCovAiRuns_TriggerOmittedInputDefaultsEmptyObject asserts that omitting
// the optional `input` field entirely (as opposed to sending it null or
// blank) succeeds and stores AgentRun.Input as an empty JSON object, not
// null and not a raw echo.
func TestCovAiRuns_TriggerOmittedInputDefaultsEmptyObject(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	id := jsonField(created, "id")
	defer func() { _, _, _ = apiClient.Post(agentRunsPath+"/"+id+"/actions/cancel", nil, newIdempotencyKey()) }()

	assert.Equal(t, "pending", jsonField(created, "status"))
	inputObj := jsonObject(created, "input")
	require.NotNil(t, inputObj, "input should still be an object (not null) when omitted")
	assert.Empty(t, inputObj)
}

// TestCovAiRuns_TriggerIdempotent asserts that replaying the same
// Idempotency-Key on two Trigger calls returns the same run id both times
// (both 201s), per the standard gateway idempotency contract.
func TestCovAiRuns_TriggerIdempotent(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	body := map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
		"input":               "idempotent trigger test",
	}

	status1, body1, err := apiClient.Post(agentRunsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	defer func() { _, _, _ = apiClient.Post(agentRunsPath+"/"+id1+"/actions/cancel", nil, newIdempotencyKey()) }()

	status2, body2, err := apiClient.Post(agentRunsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	id2 := jsonField(parseJSON(body2), "id")

	assert.Equal(t, id1, id2, "replaying the same Idempotency-Key should return the same run id")
}

// ──────────────────────────────────────────────
// Trigger -- validation
// ──────────────────────────────────────────────

func TestCovAiRuns_TriggerMissingAgentDefinitionID(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"input": "x",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "agent_definition_id")
}

func TestCovAiRuns_TriggerEmptyAgentDefinitionID(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "agent_definition_id")
}

func TestCovAiRuns_TriggerNonexistentAgentDefinitionID(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": "agdf_doesnotexist000000",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovAiRuns_TriggerInactiveAgent asserts that triggering an agent
// definition with no active agent_account_status row for the caller's
// account (SeedInactiveAgentDefinitionID) returns a gateway-level 400
// validation error distinct from the 404 for a nonexistent definition id.
func TestCovAiRuns_TriggerInactiveAgent(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": SeedInactiveAgentDefinitionID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "inactive")
}

// TestCovAiRuns_TriggerNullInput asserts an explicit JSON null for the
// optional `input` field is rejected 400 ("cannot be null"), per
// field.Optional semantics.
func TestCovAiRuns_TriggerNullInput(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
		"input":               nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "input")
	assert.Contains(t, errObj["message"], "cannot be null")
}

// TestCovAiRuns_TriggerBlankInput asserts a present-but-empty string for
// `input` is rejected 400 ("must not be blank"), distinct from the omitted
// case (TestCovAiRuns_TriggerOmittedInputDefaultsEmptyObject) which
// succeeds.
func TestCovAiRuns_TriggerBlankInput(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
		"input":               "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "input")
	assert.Contains(t, errObj["message"], "must not be blank")
}

// ──────────────────────────────────────────────
// Cancel
// ──────────────────────────────────────────────

// TestCovAiRuns_CancelWrongState asserts that cancelling a run already in a
// terminal state (SeedAgentRunID is `completed`) is rejected 400, not
// silently accepted.
func TestCovAiRuns_CancelWrongState(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "cannot be cancelled")
}

func TestCovAiRuns_CancelNotFound(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/agrn_doesnotexist00000/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovAiRuns_CancelIdempotent triggers a fresh run and cancels it twice
// with the same Idempotency-Key, asserting both calls return 200 with the
// same cancelled run.
func TestCovAiRuns_CancelIdempotent(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath, map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")

	key := newIdempotencyKey()
	status1, body1, err := apiClient.Post(agentRunsPath+"/"+id+"/actions/cancel", nil, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	first := parseJSON(body1)
	assert.Equal(t, "cancelled", jsonField(first, "status"))

	status2, body2, err := apiClient.Post(agentRunsPath+"/"+id+"/actions/cancel", nil, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	second := parseJSON(body2)
	assert.Equal(t, "cancelled", jsonField(second, "status"))
	assert.Equal(t, jsonField(first, "updated_at"), jsonField(second, "updated_at"),
		"replaying the same key should return the exact cached response, not re-execute the transition")
}

// ──────────────────────────────────────────────
// Continue
// ──────────────────────────────────────────────

// TestCovAiRuns_ContinueHappyPathAndIdempotent consumes the dedicated
// awaiting_input seed row (SeedAgentRunAwaitingInputID) exactly once: it
// resumes the run with a message plus approved/rejected tool slugs and
// asserts the status flip to `running`, then replays the identical request
// with the same Idempotency-Key and asserts the cached response is
// returned unchanged (folding the idempotency assertion into this same
// seed-row consumption rather than spending a second dedicated row on it).
func TestCovAiRuns_ContinueHappyPathAndIdempotent(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	body := map[string]any{
		"message":             "Yes, proceed.",
		"approved_tool_slugs": []string{"create_alert"},
		"rejected_tool_slugs": []string{"save_memory"},
	}

	status1, respBody1, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunAwaitingInputID+"/actions/continue", body, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, respBody1)
	first := parseJSON(respBody1)
	assert.Equal(t, SeedAgentRunAwaitingInputID, jsonField(first, "id"))
	assertObjectField(t, first, "agent_run")
	assert.Equal(t, "running", jsonField(first, "status"))

	status2, respBody2, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunAwaitingInputID+"/actions/continue", body, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, respBody2)
	second := parseJSON(respBody2)
	assert.Equal(t, "running", jsonField(second, "status"))
	assert.Equal(t, jsonField(first, "updated_at"), jsonField(second, "updated_at"),
		"replaying the same key should return the exact cached response")
}

// TestCovAiRuns_ContinueMissingMessage asserts the required `message` field
// is validated before the run's current state is even considered: an empty
// body (and a present-but-blank message) against a stable completed seed
// run (SeedAgentRunID) both return 400 missing_field, never touching run
// state, so this test is safe to run alongside other tests touching that
// same row.
func TestCovAiRuns_ContinueMissingMessage(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/continue", map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "message")

	status, body, err = apiClient.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/continue", map[string]any{"message": ""}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj = requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "message")
}

// TestCovAiRuns_ContinueWrongState asserts continuing a run that is not
// awaiting_input/awaiting_approval (SeedAgentRunID is `completed`) is
// rejected 400.
func TestCovAiRuns_ContinueWrongState(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/continue", map[string]any{"message": "hi"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "not awaiting input or approval")
}

func TestCovAiRuns_ContinueNotFound(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/agrn_doesnotexist00000/actions/continue", map[string]any{"message": "hi"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Retry
// ──────────────────────────────────────────────

// TestCovAiRuns_RetryHappyPathAndIdempotent consumes the dedicated failed
// seed row (SeedAgentRunFailedID) exactly once: retries it and asserts the
// status flip to `running` with error_message reset to null, then replays
// the same Idempotency-Key and asserts the cached response is returned
// unchanged.
func TestCovAiRuns_RetryHappyPathAndIdempotent(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()

	status1, body1, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunFailedID+"/actions/retry", nil, key)
	require.NoError(t, err)
	requireStatus(t, 200, status1, body1)
	first := parseJSON(body1)
	assert.Equal(t, SeedAgentRunFailedID, jsonField(first, "id"))
	assert.Equal(t, "running", jsonField(first, "status"))
	assertNilField(t, first, "error_message")

	status2, body2, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunFailedID+"/actions/retry", nil, key)
	require.NoError(t, err)
	requireStatus(t, 200, status2, body2)
	second := parseJSON(body2)
	assert.Equal(t, "running", jsonField(second, "status"))
	assert.Equal(t, jsonField(first, "updated_at"), jsonField(second, "updated_at"),
		"replaying the same key should return the exact cached response")
}

// TestCovAiRuns_RetryWrongState asserts retrying a non-failed run
// (SeedAgentRunID is `completed`) is rejected 400.
func TestCovAiRuns_RetryWrongState(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/retry", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "Only failed runs can be retried")
}

func TestCovAiRuns_RetryNotFound(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(agentRunsPath+"/agrn_doesnotexist00000/actions/retry", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// ──────────────────────────────────────────────
// List
// ──────────────────────────────────────────────

func TestCovAiRuns_List_Basic(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(agentRunsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	require.GreaterOrEqual(t, len(list.Data), 1, "at least one seeded agent run must be present")

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "agrn")
	assertObjectField(t, row, "agent_run")
}

// TestCovAiRuns_List_SearchHit asserts `q` substring-matches a fragment of
// the seeded run's id and returns exactly that row.
func TestCovAiRuns_List_SearchHit(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(agentRunsPath, url.Values{"q": {"run00001"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	row := parseJSON(list.Data[0])
	assert.Equal(t, SeedAgentRunID, jsonField(row, "id"))
}

func TestCovAiRuns_List_SearchMiss(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(agentRunsPath, url.Values{"q": {"zzzz_no_such_run_qqqq"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

// TestCovAiRuns_List_StatusFilter asserts the `status` query param filters
// the list, with the seeded completed run reachable somewhere in the
// filtered pages.
func TestCovAiRuns_List_StatusFilter(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(agentRunsPath, url.Values{"status": {"completed"}, "limit": {"2"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	for _, raw := range list.Data {
		row := parseJSON(raw)
		assert.Equal(t, "completed", jsonField(row, "status"))
	}
	assertListContainsID(t, agentRunsPath, url.Values{"status": {"completed"}}, SeedAgentRunID)
}

// TestCovAiRuns_List_StatusFilterUnrecognizedIsEmpty documents (rather than
// 400s) the observed behaviour that an unrecognized `status` value is a
// bare equality filter with no enum validation at the gateway -- it
// silently returns an empty list instead of erroring. This locks the
// current behaviour in place so a future change to a 400 (or a regression
// to a 5xx) is caught explicitly.
func TestCovAiRuns_List_StatusFilterUnrecognizedIsEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(agentRunsPath, url.Values{"status": {"bogus_status_xyz"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

func TestCovAiRuns_List_AgentDefinitionIDFilter(t *testing.T) {
	t.Parallel()

	assertListContainsID(t, agentRunsPath, url.Values{"agent_definition_id": {SeedAgentDefinitionID}}, SeedAgentRunID)
}

func TestCovAiRuns_List_AgentDefinitionIDNonexistentIsEmpty(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(agentRunsPath, url.Values{"agent_definition_id": {"agdf_doesnotexist000000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

func TestCovAiRuns_List_PaginationAdvances(t *testing.T) {
	t.Parallel()

	assertCursorPaginationAdvances(t, agentRunsPath, nil)
}

func TestCovAiRuns_List_LimitOutOfRange(t *testing.T) {
	t.Parallel()

	for _, limit := range []string{"0", "-1", "1001"} {
		status, body, err := apiClient.GetListRaw(agentRunsPath, url.Values{"limit": {limit}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	}
}

func TestCovAiRuns_List_CursorInvalid(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentRunsPath, url.Values{"cursor": {"not_a_real_cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovAiRuns_List_IncludeStepsRejected asserts that `?include=steps`,
// valid on GET-by-id, is rejected 400 on the list endpoint -- the two
// endpoints have deliberately different allowed include sets (fact
// documented in presenter.go / endpoint_list_runs.go IncludeConfig).
func TestCovAiRuns_List_IncludeStepsRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentRunsPath, url.Values{"include": {"steps"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Expandable -- retrieve-by-id include set (triggered_by, nested
// definition.config/tools/role, and full AgentAction/AgentRunStep field
// coverage via a run that actually has action/step rows).
// ──────────────────────────────────────────────

// TestCovAiRuns_IncludeTriggeredBy asserts `?include=triggered_by` (valid
// on GET-by-id, absent from the mutating endpoints' allowed set) populates
// an Actor stub.
func TestCovAiRuns_IncludeTriggeredBy(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+SeedAgentRunID, url.Values{"include": {"triggered_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	triggeredBy := jsonObject(got, "triggered_by")
	require.NotNil(t, triggeredBy, "triggered_by should populate with ?include=triggered_by")
	assertObjectField(t, triggeredBy, "actor")
	assert.NotEmpty(t, jsonField(triggeredBy, "id"))
	assert.NotEmpty(t, jsonField(triggeredBy, "type"))
	assert.NotEmpty(t, jsonField(triggeredBy, "name"))
}

// TestCovAiRuns_ActionEndpoints_TriggeredByRejected asserts the trigger
// endpoint's IncludeConfig omits `triggered_by` (present on retrieve, but
// not on the four mutating/action endpoints) -- an unsupported include
// value 400s rather than silently no-op-ing.
func TestCovAiRuns_ActionEndpoints_TriggeredByRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Do("POST", agentRunsPath+"?include=triggered_by", map[string]any{
		"agent_definition_id": SeedAgentDefinitionID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovAiRuns_IncludeDefinitionNested asserts
// ?include=definition,definition.config,definition.tools,definition.role
// populates the nested AgentDefinition sub-objects.
func TestCovAiRuns_IncludeDefinitionNested(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+SeedAgentRunID,
		url.Values{"include": {"definition,definition.config,definition.tools,definition.role"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	def := jsonObject(got, "definition")
	require.NotNil(t, def)
	assertIDFormat(t, jsonField(def, "id"), "agdf")
	assertObjectField(t, def, "agent_definition")

	role := jsonObject(def, "role")
	require.NotNil(t, role, "definition.role should populate")
	assertObjectField(t, role, "role")

	config := jsonObject(def, "config")
	require.NotNil(t, config, "definition.config should populate")
	assertObjectField(t, config, "agent_definition_config")

	tools := jsonObject(def, "tools")
	require.NotNil(t, tools, "definition.tools should populate")
	assertObjectField(t, tools, "list")
}

// TestCovAiRuns_IncludeActions_AllFields asserts every AgentAction json
// field is populated (or explicitly null) via ?include=actions on
// SeedAgentRunID, which has a real seeded agent_action row. In particular
// it hard-asserts action.run.status/trigger_type are real, non-empty
// values matching the parent run -- the "always-populated reference stub"
// pattern documented on AgentAction.Run, which a future call site
// forgetting to pass runStatusCode/runTriggerType to sqlcActionToProto
// would silently blank out.
func TestCovAiRuns_IncludeActions_AllFields(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+SeedAgentRunID, url.Values{"include": {"actions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	actions := jsonListData(got, "actions")
	require.NotEmpty(t, actions, "SeedAgentRunID should have at least one seeded agent_action")

	action, ok := actions[0].(map[string]any)
	require.True(t, ok)

	assertIDFormat(t, jsonField(action, "id"), "agac")
	assertObjectField(t, action, "agent_action")
	assert.NotEmpty(t, jsonField(action, "tool"))
	assert.NotEmpty(t, jsonField(action, "status"))
	assert.NotEmpty(t, jsonField(action, "label"))
	assert.NotEmpty(t, jsonField(action, "description"))

	run := jsonObject(action, "run")
	require.NotNil(t, run, "action.run should never be nil (always-populated stub)")
	assert.Equal(t, SeedAgentRunID, jsonField(run, "id"))
	assertObjectField(t, run, "agent_run")
	assert.NotEmpty(t, jsonField(run, "status"), "action.run.status must be a real value, not blanked out")
	assert.NotEmpty(t, jsonField(run, "trigger_type"), "action.run.trigger_type must be a real value, not blanked out")
	assertNilField(t, run, "definition")
	assertNilField(t, run, "actions")
	assertNilField(t, run, "steps")

	inputObj := jsonObject(action, "input")
	require.NotNil(t, inputObj)
	assert.NotEmpty(t, inputObj)
	outputObj := jsonObject(action, "output")
	require.NotNil(t, outputObj)
	assert.NotEmpty(t, outputObj)
	assertNilField(t, action, "error_message")
	assertNilField(t, action, "entity")
	assert.Equal(t, "not_required", jsonField(action, "review_requirement"))
	assertNilField(t, action, "reviewed_at")
	assertNilField(t, action, "reviewed_by")
	assertNilField(t, action, "executed_at")
	assertValidTimestamp(t, jsonField(action, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(action, "updated_at"), "updated_at")
}

// TestCovAiRuns_IncludeSteps_AllFields asserts every AgentRunStep json
// field is populated (or explicitly null) via ?include=steps on
// SeedAgentRunID, which has real seeded agent_run_event rows.
//
// Note: the ids observed here ("agev_...") are hand-authored literals in
// the seed SQL fixture and do not match the id.AgentRunEventIDPrefix
// constant ("agrnev_") that the real id generator would produce for a
// step created by an actual run -- this is a seed-fixture naming quirk,
// not something reachable via any live API call (steps are never created
// through a POST endpoint), so it is asserted against the literal observed
// prefix rather than the source constant.
func TestCovAiRuns_IncludeSteps_AllFields(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(agentRunsPath+"/"+SeedAgentRunID, url.Values{"include": {"steps"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	steps := jsonListData(got, "steps")
	require.GreaterOrEqual(t, len(steps), 2, "SeedAgentRunID should have at least two seeded agent_run_event rows")

	for _, raw := range steps {
		step, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.True(t, len(jsonField(step, "id")) > 0 && jsonField(step, "id")[:5] == "agev_",
			"step id %q should start with the seeded agev_ prefix", jsonField(step, "id"))
		assertObjectField(t, step, "agent_run_step")
		assert.NotEmpty(t, jsonField(step, "step_type"))
		assert.NotEmpty(t, jsonField(step, "title"))
		assert.NotEmpty(t, jsonField(step, "content"))
		assertValidTimestamp(t, jsonField(step, "created_at"), "created_at")
	}

	first, ok := steps[0].(map[string]any)
	require.True(t, ok)
	seq, ok := first["sequence"].(float64)
	require.True(t, ok, "sequence should be a JSON number")
	assert.Equal(t, float64(1), seq)
	dur, ok := first["duration_ms"].(float64)
	require.True(t, ok, "duration_ms should be populated for the seeded event")
	assert.Greater(t, dur, float64(0))
	// actor is documented as "not expandable -- always populated when known",
	// but the seed rows carry no resolvable actor identity, so it is null.
	assertNilField(t, first, "actor")
	// metadata stored as the literal empty-object string "{}" is deliberately
	// treated as absent by the presenter (presenter.go: `MetadataJson != "{}"`),
	// so it renders as null rather than {}.
	assertNilField(t, first, "metadata")
}

// ──────────────────────────────────────────────
// Auth / permission boundaries
// ──────────────────────────────────────────────

func TestCovAiRuns_RequiresAuth(t *testing.T) {
	t.Parallel()

	unauth := apiClient.WithBearerToken("", SeedAccountID)
	status, body, err := unauth.GetListRaw(agentRunsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}

// TestCovAiRuns_CustomerPortalForbidden asserts a customer-portal actor
// (not an internal user for the account) is rejected 403
// insufficient_permissions on every one of the six routes in this group --
// CheckIsInternalActor is enforced per-endpoint, not by shared middleware,
// so each verb gets its own assertion.
func TestCovAiRuns_CustomerPortalForbidden(t *testing.T) {
	t.Parallel()
	customer := getCustomerPortalClient()

	status, body, err := customer.GetListRaw(agentRunsPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = customer.GetListRaw(agentRunsPath+"/"+SeedAgentRunID, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = customer.Post(agentRunsPath, map[string]any{"agent_definition_id": SeedAgentDefinitionID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = customer.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = customer.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/continue", map[string]any{"message": "hi"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	status, body, err = customer.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/retry", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Tenant isolation -- one assertion per mutating verb, not just GetByID,
// since AccountID scoping is enforced per service-layer branch rather than
// by shared middleware.
// ──────────────────────────────────────────────

func TestCovAiRuns_TenantIsolation_Get(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	status, body, err := tenantB.GetListRaw(agentRunsPath+"/"+SeedAgentRunID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovAiRuns_TenantIsolation_Cancel(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	status, body, err := tenantB.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/cancel", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovAiRuns_TenantIsolation_Continue(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	status, body, err := tenantB.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/continue", map[string]any{"message": "hi"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovAiRuns_TenantIsolation_Retry(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	status, body, err := tenantB.Post(agentRunsPath+"/"+SeedAgentRunID+"/actions/retry", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovAiRuns_TenantIsolation_TriggerOtherTenantsAgentDefinition asserts
// that triggering with tenant A's agent_definition_id from tenant B does
// not leak tenant A's run/definition: the shared agent_definition catalog
// row is visible cross-tenant (GetAgentDefinition itself is not
// account-scoped), but tenant B has no agent_account_status activation row
// for it, so the gateway's own inactive-agent check rejects it with the
// same 400 validation_failed used for any inactive agent -- not a 404 and
// not a successful trigger.
func TestCovAiRuns_TenantIsolation_TriggerOtherTenantsAgentDefinition(t *testing.T) {
	t.Parallel()
	tenantB := getTenantBClient()

	status, body, err := tenantB.Post(agentRunsPath, map[string]any{"agent_definition_id": SeedAgentDefinitionID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Contains(t, errObj["message"], "inactive")
}
