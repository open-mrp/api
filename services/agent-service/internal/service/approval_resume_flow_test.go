package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/augno/api/services/agent-service/internal/agents"
	"github.com/augno/api/services/agent-service/internal/domain"
	factorymock "github.com/augno/api/services/agent-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/agent-service/internal/domain/mock/repository"
	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/agent-service/internal/llm"
	"go.uber.org/mock/gomock"
)

// These tests drive the whole approval-resume pipeline end to end — reconstructMessages → resumeApprovedBlockedCalls → runAgentLoop — the way ContinueRun does. They exist to lock in the fix for the "agent said it did X but nothing changed" class of bug: an approved tool must actually execute even when the model does NOT re-issue it, a failed tool must never be recorded as a success, and no path may double-execute an approved write.

// the exact placeholder the guard writes when a review-gated tool is called without approval.
const approvalPlaceholder = "[REQUIRES APPROVAL] This tool requires human approval before it can be executed. The run will pause for review."

// capturedEvent is a run event as it was persisted, decoded enough to assert on step type and is_error.
type capturedEvent struct {
	StepType string
	Title    string
	Metadata map[string]any
	Content  string
}

func (e capturedEvent) isError() bool {
	v, _ := e.Metadata["is_error"].(bool)
	return v
}

// toolBehavior scripts what a spy tool handler returns.
type toolBehavior struct {
	result string
	err    error
}

// spyRegistry builds a registry whose tools record their call count and inputs and return a scripted result/error.
func spyRegistry(behaviors map[string]toolBehavior) (*agents.ToolHandlerRegistry, map[string]int, map[string][]string) {
	calls := map[string]int{}
	inputs := map[string][]string{}
	reg := agents.NewToolHandlerRegistry()
	for name, b := range behaviors {
		reg.Register(name, func(_ context.Context, in json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
			calls[name]++
			inputs[name] = append(inputs[name], string(in))
			return b.result, b.err
		})
	}
	return reg, calls, inputs
}

// newCapturingRunner wires a runnerSvc with the scripted LLM and spy registry, capturing every persisted event so tests can assert the runner's is_error accounting.
func newCapturingRunner(t *testing.T, provider llm.LLMProvider, registry *agents.ToolHandlerRegistry) (*runnerSvc, *[]capturedEvent) {
	t.Helper()
	ctrl := gomock.NewController(t)
	factory := factorymock.NewMockRepoFactory(ctrl)
	eventRepo := repositorymock.NewMockAgentRunEventRepo(ctrl)
	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)

	captured := &[]capturedEvent{}
	factory.EXPECT().NewAgentRunEventRepo().Return(eventRepo).AnyTimes()
	factory.EXPECT().NewAgentRunRepo().Return(runRepo).AnyTimes()
	eventRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p sqlc.InsertAgentRunEventParams) error {
			var meta map[string]any
			if len(p.Metadata) > 0 {
				_ = json.Unmarshal(p.Metadata, &meta)
			}
			content := ""
			if p.Content.Valid {
				content = p.Content.String
			}
			*captured = append(*captured, capturedEvent{StepType: p.StepType, Title: p.Title, Metadata: meta, Content: content})
			return nil
		}).AnyTimes()
	runRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).
		Return(&sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}, nil).AnyTimes()

	return &runnerSvc{
		repos:        factory,
		toolRegistry: registry,
		llmProviders: map[string]llm.LLMProvider{provider.Name(): provider},
	}, captured
}

// blockedCallEvents builds the persisted transcript of a prior turn that paused on a review-gated call: the user's request, the tool_call, and the tool_blocked placeholder — carrying the real tool name and input in metadata, as the runner records them.
func blockedCallEvents(userMsg, toolUseID, name, input string) []sqlc.AgentRunEvent {
	meta, _ := json.Marshal(map[string]any{"tool_use_id": toolUseID, "tool_name": name, "input": json.RawMessage(input)})
	return []sqlc.AgentRunEvent{
		{StepType: "user_message", Content: agentdb.PgText(userMsg)},
		{StepType: "tool_call", Metadata: meta},
		{StepType: "tool_blocked", Metadata: meta, Content: agentdb.PgText(approvalPlaceholder)},
	}
}

// resumeAndRun replays ContinueRun's resume tail through the SAME method production uses (runResumedLoop): reconstruct the transcript, then auto-execute approved blocked calls and run the loop. Driving the real seam means a regression that drops the auto-execution step fails these tests. Returns the loop result, every captured event, and the (mutated) transcript handed to the loop.
func resumeAndRun(t *testing.T, priorEvents []sqlc.AgentRunEvent, runCtx *domain.HandlerRunContext, registry *agents.ToolHandlerRegistry, toolNames []string, continuation []*llm.ToolResponse) (*domain.RunResult, []capturedEvent, []llm.Message) {
	t.Helper()
	runner, captured := newCapturingRunner(t, &scriptedLLM{responses: continuation}, registry)
	messages := reconstructMessages(priorEvents)
	seq := len(priorEvents)
	run := &sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}

	result, err := runner.runResumedLoop(
		context.Background(), run, "acc_test", nil,
		"you are a test agent", []string{"claude-test"},
		toolDefsFor(toolNames...), 0,
		messages, &seq, runCtx, priorEvents, nil, 0, nil,
	)
	if err != nil {
		t.Fatalf("runResumedLoop returned error: %v", err)
	}
	return result, *captured, messages
}

// approvedRunCtx arms a slug-level (approve-all style) approval for the gated tool.
func approvedRunCtx(gatedTool string) *domain.HandlerRunContext {
	return &domain.HandlerRunContext{
		AccountID:                "acc_test",
		RunID:                    "agr_test",
		RequireReviewBySlug:      map[string]bool{gatedTool: true},
		AlwaysAllowedSlugs:       map[string]bool{},
		OneTimeApprovedSlugs:     map[string]bool{gatedTool: true},
		OneTimeApprovedKeys:      map[string]bool{},
		RejectedSlugs:            map[string]bool{},
		RejectedKeys:             map[string]bool{},
		RevealedToolSlugs:        map[string]bool{},
		AllowedEndpointToolSlugs: map[string]bool{},
	}
}

func toolResultBlockFor(messages []llm.Message, toolUseID string) (llm.ToolResultBlock, bool) {
	for _, m := range messages {
		for _, tr := range m.ToolResults {
			if tr.ToolUseID == toolUseID {
				return tr, true
			}
		}
	}
	return llm.ToolResultBlock{}, false
}

func toolResultEvents(events []capturedEvent, toolUseID string) []capturedEvent {
	var out []capturedEvent
	for _, e := range events {
		if e.StepType != "tool_result" {
			continue
		}
		if id, _ := e.Metadata["tool_use_id"].(string); id == toolUseID {
			out = append(out, e)
		}
	}
	return out
}

// THE headline regression: after approval, the tool executes even though the model does NOT re-issue it — it just narrates "Done." Under the old design (no auto-execution) this write never happened while the agent claimed success.
func TestApprovalResume_ExecutesWhenModelDoesNotReissue(t *testing.T) {
	t.Parallel()
	prior := blockedCallEvents("put this customer on hold", "toolu_1", guardGatedTool, `{"id":"ac_1","status":"hold_all"}`)
	registry, calls, _ := spyRegistry(map[string]toolBehavior{
		guardGatedTool: {result: `{"id":"ac_1","status":"hold_all"}`},
	})
	runCtx := approvedRunCtx(guardGatedTool)

	// The model, seeing a successful result, simply reports done — it does not call the tool again.
	result, events, messages := resumeAndRun(t, prior, runCtx, registry, []string{guardGatedTool},
		[]*llm.ToolResponse{endTurnResponse("Done — the customer is on hold.")})

	if calls[guardGatedTool] != 1 {
		t.Fatalf("approved tool must execute exactly once via auto-execution, ran %d times", calls[guardGatedTool])
	}
	if result.AwaitingApproval {
		t.Fatal("run should complete after the approved call executes, not pause again")
	}
	// The placeholder must be replaced with the real, non-error result in the transcript the model saw.
	block, ok := toolResultBlockFor(messages, "toolu_1")
	if !ok {
		t.Fatal("no tool_result block for the approved call in the transcript")
	}
	if block.Content == approvalPlaceholder || block.IsError {
		t.Errorf("placeholder was not replaced with the real success result: %+v", block)
	}
	// A truthful, non-error tool_result event must be recorded for the timeline / future resumes.
	res := toolResultEvents(events, "toolu_1")
	if len(res) != 1 || res[0].isError() {
		t.Fatalf("expected one non-error tool_result event, got %+v", res)
	}
}

// The double-write guard: even if the model re-issues the identical call after approval (a new tool_use_id), it must NOT run a second time — the consumed approval re-blocks it.
func TestApprovalResume_ModelReissueDoesNotDoubleExecute(t *testing.T) {
	t.Parallel()
	prior := blockedCallEvents("put this customer on hold", "toolu_1", guardGatedTool, `{"id":"ac_1","status":"hold_all"}`)
	registry, calls, _ := spyRegistry(map[string]toolBehavior{
		guardGatedTool: {result: `{"id":"ac_1","status":"hold_all"}`},
	})
	runCtx := approvedRunCtx(guardGatedTool)

	// Worst case: the model re-issues the same tool call anyway.
	result, _, _ := resumeAndRun(t, prior, runCtx, registry, []string{guardGatedTool},
		[]*llm.ToolResponse{toolUseResponse(guardGatedTool, `{"id":"ac_1","status":"hold_all"}`)})

	if calls[guardGatedTool] != 1 {
		t.Fatalf("re-issued approved call must not double-execute: handler ran %d times", calls[guardGatedTool])
	}
	// The re-issue hits the (now-consumed) approval guard and re-pauses rather than silently writing again.
	if !result.AwaitingApproval {
		t.Fatal("a re-issued gated call with the approval consumed should re-block, not run")
	}
}

// A failed approved call must surface as an error, never as a silent success — the other half of the "said it did it, didn't" bug.
func TestApprovalResume_FailedApprovedCallSurfacesError(t *testing.T) {
	t.Parallel()
	prior := blockedCallEvents("update the customer", "toolu_1", guardGatedTool, `{"id":"ac_1","status":"bogus"}`)
	registry, calls, _ := spyRegistry(map[string]toolBehavior{
		guardGatedTool: {err: errors.New("gateway returned HTTP 422: validation_failed")},
	})
	runCtx := approvedRunCtx(guardGatedTool)

	_, events, messages := resumeAndRun(t, prior, runCtx, registry, []string{guardGatedTool},
		[]*llm.ToolResponse{endTurnResponse("acknowledged")})

	if calls[guardGatedTool] != 1 {
		t.Fatalf("approved call should have been attempted once, ran %d", calls[guardGatedTool])
	}
	block, ok := toolResultBlockFor(messages, "toolu_1")
	if !ok || !block.IsError {
		t.Errorf("failed approved call must be an is_error result in the transcript: %+v", block)
	}
	res := toolResultEvents(events, "toolu_1")
	if len(res) != 1 || !res[0].isError() {
		t.Fatalf("expected one is_error tool_result event for the failed call, got %+v", res)
	}
}

// Approve-all resumes multiple pending calls at once: every approved blocked call executes.
func TestApprovalResume_MultipleApprovedAllExecute(t *testing.T) {
	t.Parallel()
	prior := append(
		blockedCallEvents("hold A", "toolu_1", guardGatedTool, `{"id":"ac_1","status":"hold_all"}`),
		blockedCallEvents("email B", "toolu_2", "send_email", `{"to":"x@y.z"}`)...,
	)
	registry, calls, _ := spyRegistry(map[string]toolBehavior{
		guardGatedTool: {result: `{"ok":true}`},
		"send_email":   {result: `{"ok":true}`},
	})
	runCtx := approvedRunCtx(guardGatedTool)
	// approve-all also armed the second slug.
	runCtx.RequireReviewBySlug["send_email"] = true
	runCtx.OneTimeApprovedSlugs["send_email"] = true

	_, _, messages := resumeAndRun(t, prior, runCtx, registry, []string{guardGatedTool, "send_email"},
		[]*llm.ToolResponse{endTurnResponse("both done")})

	if calls[guardGatedTool] != 1 || calls["send_email"] != 1 {
		t.Fatalf("both approved calls should execute once each: %v", calls)
	}
	for _, id := range []string{"toolu_1", "toolu_2"} {
		if block, _ := toolResultBlockFor(messages, id); block.Content == approvalPlaceholder {
			t.Errorf("call %s placeholder was not replaced", id)
		}
	}
}

// Per-call (by tool_use_id → slug+input key) approval: with two same-slug calls, only the one the human approved runs; the other stays blocked for the guard to re-prompt.
func TestApprovalResume_PerCallApprovalRunsOnlyApprovedCall(t *testing.T) {
	t.Parallel()
	approvedInput := `{"id":"ac_1","status":"hold_all"}`
	otherInput := `{"id":"ac_2","status":"hold_all"}`
	prior := append(
		blockedCallEvents("hold ac_1", "toolu_1", guardGatedTool, approvedInput),
		blockedCallEvents("hold ac_2", "toolu_2", guardGatedTool, otherInput)...,
	)
	registry, calls, inputs := spyRegistry(map[string]toolBehavior{
		guardGatedTool: {result: `{"ok":true}`},
	})
	// Only the first call's (slug+input) key is approved — NOT the slug, so the second call is not covered.
	runCtx := &domain.HandlerRunContext{
		AccountID: "acc_test", RunID: "agr_test",
		RequireReviewBySlug:      map[string]bool{guardGatedTool: true},
		AlwaysAllowedSlugs:       map[string]bool{},
		OneTimeApprovedSlugs:     map[string]bool{},
		OneTimeApprovedKeys:      map[string]bool{toolCallApprovalKey(guardGatedTool, json.RawMessage(approvedInput)): true},
		RejectedSlugs:            map[string]bool{},
		RejectedKeys:             map[string]bool{},
		RevealedToolSlugs:        map[string]bool{},
		AllowedEndpointToolSlugs: map[string]bool{},
	}

	_, _, messages := resumeAndRun(t, prior, runCtx, registry, []string{guardGatedTool},
		[]*llm.ToolResponse{endTurnResponse("done")})

	if calls[guardGatedTool] != 1 {
		t.Fatalf("only the per-call-approved call should run, ran %d times (inputs=%v)", calls[guardGatedTool], inputs[guardGatedTool])
	}
	if len(inputs[guardGatedTool]) != 1 || inputs[guardGatedTool][0] != approvedInput {
		t.Fatalf("the wrong call executed: %v", inputs[guardGatedTool])
	}
	if block, _ := toolResultBlockFor(messages, "toolu_1"); block.Content == approvalPlaceholder {
		t.Error("approved call toolu_1 was not executed")
	}
	if block, _ := toolResultBlockFor(messages, "toolu_2"); block.Content != approvalPlaceholder {
		t.Error("unapproved call toolu_2 should stay blocked, not execute")
	}
}

// Runner-level accounting (the downstream half of the swallowed-error bug): a tool that returns a Go error during the normal loop is recorded is_error and is NOT captured as a successful action. If the gateway client ever regresses to swallowing HTTP errors into (body, nil), this is what would fail.
func TestRunAgentLoop_ErroringToolIsNotRecordedAsSuccess(t *testing.T) {
	t.Parallel()
	registry, calls, _ := spyRegistry(map[string]toolBehavior{
		guardSafeTool: {err: errors.New("gateway returned HTTP 400: validation_failed")},
	})
	runner, captured := newCapturingRunner(t, &scriptedLLM{responses: []*llm.ToolResponse{
		toolUseResponse(guardSafeTool, `{"q":"x"}`),
		endTurnResponse("ok"),
	}}, registry)

	runCtx := newGuardRunCtx(nil, nil) // not gated — exercises the normal execution path
	seq := 0
	run := &sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}
	result, err := runner.runAgentLoop(context.Background(), run, "acc_test", nil,
		"sys", []string{"claude-test"}, toolDefsFor(guardSafeTool), 0,
		[]llm.Message{{Role: "user", Content: "go"}}, &seq, runCtx, nil, 0, nil)
	if err != nil {
		t.Fatalf("runAgentLoop: %v", err)
	}

	if calls[guardSafeTool] != 1 {
		t.Fatalf("tool should have been attempted once, ran %d", calls[guardSafeTool])
	}
	res := toolResultEvents(*captured, "tu_"+guardSafeTool)
	if len(res) != 1 || !res[0].isError() {
		t.Fatalf("an erroring tool must record an is_error tool_result, got %+v", res)
	}
	// The failed call must not appear as a successful (non-review) action.
	for _, a := range result.Actions {
		if a.ToolSlug == guardSafeTool && !a.RequiresReview {
			t.Errorf("a failed tool call was recorded as a successful action: %+v", a)
		}
	}
}

// A successful tool in the normal loop IS recorded as a non-error result and a successful action — the positive control for the test above.
func TestRunAgentLoop_SucceedingToolRecordedAsSuccess(t *testing.T) {
	t.Parallel()
	registry, _, _ := spyRegistry(map[string]toolBehavior{
		guardSafeTool: {result: `{"data":[]}`},
	})
	runner, captured := newCapturingRunner(t, &scriptedLLM{responses: []*llm.ToolResponse{
		toolUseResponse(guardSafeTool, `{"q":"x"}`),
		endTurnResponse("ok"),
	}}, registry)

	runCtx := newGuardRunCtx(nil, nil)
	seq := 0
	run := &sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}
	result, err := runner.runAgentLoop(context.Background(), run, "acc_test", nil,
		"sys", []string{"claude-test"}, toolDefsFor(guardSafeTool), 0,
		[]llm.Message{{Role: "user", Content: "go"}}, &seq, runCtx, nil, 0, nil)
	if err != nil {
		t.Fatalf("runAgentLoop: %v", err)
	}

	res := toolResultEvents(*captured, "tu_"+guardSafeTool)
	if len(res) != 1 || res[0].isError() {
		t.Fatalf("a successful tool must record a non-error tool_result, got %+v", res)
	}
	found := false
	for _, a := range result.Actions {
		if a.ToolSlug == guardSafeTool {
			found = true
		}
	}
	if !found {
		t.Error("a successful tool call should be recorded as an action")
	}
}

// A resume whose transcript (blocked → executed) is persisted must reconstruct on the NEXT resume without a duplicate tool_result for the same id — otherwise the Anthropic API rejects the whole call and the run is permanently bricked. This simulates the second resume by feeding the events a first resume would have written.
func TestReconstructMessages_ResumeAfterExecutionHasNoDuplicateResult(t *testing.T) {
	t.Parallel()
	// Events as they exist after a first approval-resume executed the call: original block, the approval turn's user message (higher sequence than the block), then the executed result.
	blockMeta, _ := json.Marshal(map[string]any{"tool_use_id": "toolu_1", "tool_name": guardGatedTool, "input": json.RawMessage(`{"status":"hold_all"}`)})
	resultMeta, _ := json.Marshal(map[string]any{"tool_use_id": "toolu_1", "is_error": false, "full_result": `{"status":"hold_all"}`})
	events := []sqlc.AgentRunEvent{
		{StepType: "user_message", Content: agentdb.PgText("hold the customer")},
		{StepType: "tool_call", Metadata: blockMeta},
		{StepType: "tool_blocked", Metadata: blockMeta, Content: agentdb.PgText(approvalPlaceholder)},
		{StepType: "user_message", Content: agentdb.PgText("Approved all pending tool actions")},
		{StepType: "tool_result", Metadata: resultMeta},
		{StepType: "assistant_message", Content: agentdb.PgText("Done.")},
	}

	messages := reconstructMessages(events)
	assertNoOrphanToolUse(t, messages)

	count := 0
	var content string
	for _, m := range messages {
		for _, tr := range m.ToolResults {
			if tr.ToolUseID == "toolu_1" {
				count++
				content = tr.Content
			}
		}
	}
	if count != 1 {
		t.Fatalf("tool_use_id toolu_1 must have exactly one result block after reconstruction, got %d", count)
	}
	if content != `{"status":"hold_all"}` {
		t.Errorf("the executed result must win over the blocked placeholder, got %q", content)
	}
}
