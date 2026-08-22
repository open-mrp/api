package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/agents"
	"github.com/open-mrp/api/services/agent-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
	"go.uber.org/mock/gomock"
)

// These tests drive the REAL runAgentLoop (not a copy of the guard) so they fail if the runner ever stops
// consulting the review gate before executing a tool. A scripted fake LLM emits the tool calls and a spy
// handler records whether the tool actually ran.

const (
	guardGatedTool = "update_customer"
	guardSafeTool  = "list_customers"
)

// scriptedLLM returns canned ToolResponses in order, then ends the turn so the loop can never spin. It
// implements only llm.LLMProvider (not the streaming variant), so completeWithRetry takes the plain
// CompleteWithTools path — no broker required.
type scriptedLLM struct {
	responses []*llm.ToolResponse
	calls     int
}

func (f *scriptedLLM) Name() string { return "anthropic" }

func (f *scriptedLLM) CompleteWithTools(_ context.Context, _ *llm.ToolRequest) (*llm.ToolResponse, error) {
	defer func() { f.calls++ }()
	if f.calls < len(f.responses) {
		return f.responses[f.calls], nil
	}
	return &llm.ToolResponse{StopReason: "end_turn", Content: "done"}, nil
}

var _ llm.LLMProvider = (*scriptedLLM)(nil)

func toolUseResponse(name, input string) *llm.ToolResponse {
	return &llm.ToolResponse{
		StopReason: "tool_use",
		ToolCalls:  []llm.ToolCall{{ID: "tu_" + name, Name: name, Input: json.RawMessage(input)}},
	}
}

func endTurnResponse(content string) *llm.ToolResponse {
	return &llm.ToolResponse{StopReason: "end_turn", Content: content}
}

func toolDefsFor(names ...string) []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(names))
	for _, n := range names {
		defs = append(defs, llm.ToolDefinition{Name: n, Description: "test tool", InputSchema: json.RawMessage(`{"type":"object"}`)})
	}
	return defs
}

func newGuardRunCtx(requireReview, oneTimeApproved map[string]bool) *domain.HandlerRunContext {
	if requireReview == nil {
		requireReview = map[string]bool{}
	}
	if oneTimeApproved == nil {
		oneTimeApproved = map[string]bool{}
	}
	return &domain.HandlerRunContext{
		AccountID:                "acc_test",
		RunID:                    "agr_test",
		RequireReviewBySlug:      requireReview,
		AlwaysAllowedSlugs:       map[string]bool{},
		OneTimeApprovedSlugs:     oneTimeApproved,
		RevealedToolSlugs:        map[string]bool{},
		AllowedEndpointToolSlugs: map[string]bool{},
	}
}

// newGuardTestRunner wires a runnerSvc whose only live dependency is the scripted LLM and the spy tool
// registry. Event persistence and the cancellation probe are no-op'd via mocked repos; there is no broker.
func newGuardTestRunner(t *testing.T, provider llm.LLMProvider, registry *agents.ToolHandlerRegistry) *runnerSvc {
	t.Helper()
	ctrl := gomock.NewController(t)
	factory := factorymock.NewMockRepoFactory(ctrl)
	eventRepo := repositorymock.NewMockAgentRunEventRepo(ctrl)
	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)

	factory.EXPECT().NewAgentRunEventRepo().Return(eventRepo).AnyTimes()
	factory.EXPECT().NewAgentRunRepo().Return(runRepo).AnyTimes()
	eventRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// runCancelled polls the run each iteration/tool — keep it "running" so the loop is never cut short.
	runRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).
		Return(&sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}, nil).AnyTimes()

	return &runnerSvc{
		repos:        factory,
		toolRegistry: registry,
		llmProviders: map[string]llm.LLMProvider{provider.Name(): provider},
	}
}

// runGuardLoop runs the agent loop once with the given LLM script, review map, and approvals, returning the
// result and how many times the gated tool's handler actually executed.
func runGuardLoop(t *testing.T, responses []*llm.ToolResponse, toolNames []string, runCtx *domain.HandlerRunContext) (*domain.RunResult, map[string]int) {
	t.Helper()
	calls := map[string]int{}
	registry := agents.NewToolHandlerRegistry()
	for _, name := range toolNames {
		registry.Register(name, func(_ context.Context, _ json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
			calls[name]++
			return `{"ok":true}`, nil
		})
	}

	runner := newGuardTestRunner(t, &scriptedLLM{responses: responses}, registry)
	seq := 0
	run := &sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}
	result, err := runner.runAgentLoop(
		context.Background(),
		run,
		"acc_test",
		nil, // identity is unused by the loop
		"you are a test agent",
		[]string{"claude-test"},
		toolDefsFor(toolNames...),
		0,
		[]llm.Message{{Role: "user", Content: "please act"}},
		&seq,
		runCtx,
		nil, // no spending cap
		0,
		nil, // no token rates
	)
	if err != nil {
		t.Fatalf("runAgentLoop returned error: %v", err)
	}
	return result, calls
}

// The core safety property: a review-gated tool the agent calls without an approval must NOT execute — the
// run pauses awaiting approval and records the call for review instead.
func TestRunAgentLoop_ReviewGatedToolBlockedWithoutApproval(t *testing.T) {
	t.Parallel()
	runCtx := newGuardRunCtx(map[string]bool{guardGatedTool: true}, nil) // required, NOT approved
	result, calls := runGuardLoop(t,
		[]*llm.ToolResponse{toolUseResponse(guardGatedTool, `{"id":"cus_1","note":"x"}`)},
		[]string{guardGatedTool},
		runCtx,
	)

	if calls[guardGatedTool] != 0 {
		t.Fatalf("review-gated tool executed without approval (handler ran %d times)", calls[guardGatedTool])
	}
	if !result.AwaitingApproval {
		t.Fatal("expected the run to pause awaiting approval")
	}
	if len(result.Actions) != 1 || !result.Actions[0].RequiresReview || result.Actions[0].ToolSlug != guardGatedTool {
		t.Fatalf("expected one pending review action for %q, got %+v", guardGatedTool, result.Actions)
	}
}

// With an explicit one-time approval the gated tool runs exactly once, and the approval is consumed so the
// next call would re-prompt.
func TestRunAgentLoop_ReviewGatedToolRunsOnlyAfterApproval(t *testing.T) {
	t.Parallel()
	runCtx := newGuardRunCtx(
		map[string]bool{guardGatedTool: true},
		map[string]bool{guardGatedTool: true}, // one-time approved
	)
	result, calls := runGuardLoop(t,
		[]*llm.ToolResponse{
			toolUseResponse(guardGatedTool, `{"id":"cus_1","note":"x"}`),
			endTurnResponse("done"),
		},
		[]string{guardGatedTool},
		runCtx,
	)

	if calls[guardGatedTool] != 1 {
		t.Fatalf("expected approved tool to execute exactly once, ran %d times", calls[guardGatedTool])
	}
	if result.AwaitingApproval {
		t.Fatal("expected the run to complete, not pause for approval")
	}
	if runCtx.OneTimeApprovedSlugs[guardGatedTool] {
		t.Fatal("one-time approval should be consumed after the tool executes, so the next call re-prompts")
	}
}

// A tool not marked for review runs without any approval — the gate must not over-block.
func TestRunAgentLoop_NonGatedToolRunsWithoutApproval(t *testing.T) {
	t.Parallel()
	runCtx := newGuardRunCtx(nil, nil) // nothing requires review
	result, calls := runGuardLoop(t,
		[]*llm.ToolResponse{
			toolUseResponse(guardSafeTool, `{"limit":10}`),
			endTurnResponse("done"),
		},
		[]string{guardSafeTool},
		runCtx,
	)

	if calls[guardSafeTool] != 1 {
		t.Fatalf("expected non-gated tool to execute once, ran %d times", calls[guardSafeTool])
	}
	if result.AwaitingApproval {
		t.Fatal("a non-gated tool must not pause the run for approval")
	}
}
