package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
)

// loopingLLM calls a tool with a fresh input on every request until tools are disabled, then answers in text.
// It records each request so tests can inspect what the runner sent.
type loopingLLM struct {
	tool     string
	answer   string
	requests []*llm.ToolRequest
}

func (f *loopingLLM) Name() string { return "anthropic" }

func (f *loopingLLM) CompleteWithTools(_ context.Context, req *llm.ToolRequest) (*llm.ToolResponse, error) {
	f.requests = append(f.requests, req)
	if req.ToolChoice == llm.ToolChoiceNone {
		return &llm.ToolResponse{StopReason: "end_turn", Content: f.answer}, nil
	}
	n := len(f.requests)
	return &llm.ToolResponse{
		StopReason: "tool_use",
		ToolCalls:  []llm.ToolCall{{ID: fmt.Sprintf("tu_%d", n), Name: f.tool, Input: json.RawMessage(fmt.Sprintf(`{"page":%d}`, n))}},
	}, nil
}

func lastMessageText(req *llm.ToolRequest) string {
	var sb strings.Builder
	for _, m := range req.Messages {
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

func runLoopWithBudget(t *testing.T, provider *loopingLLM, toolErr error, maxSteps int) (*domain.RunResult, []capturedEvent) {
	t.Helper()
	registry, _, _ := spyRegistry(map[string]toolBehavior{provider.tool: {result: `{"data":[]}`, err: toolErr}})
	runner, captured := newCapturingRunner(t, provider, registry)
	runCtx := newGuardRunCtx(nil, nil)
	runCtx.MaxSteps = maxSteps
	seq := 0
	run := &sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}
	result, err := runner.runAgentLoop(context.Background(), run, "acc_test", nil,
		"sys", []string{"claude-test"}, toolDefsFor(provider.tool), 0,
		[]llm.Message{{Role: "user", Content: "go"}}, &seq, runCtx, nil, 0, nil)
	if err != nil {
		t.Fatalf("runAgentLoop: %v", err)
	}
	return result, *captured
}

func completionEvent(t *testing.T, events []capturedEvent) capturedEvent {
	t.Helper()
	for _, e := range events {
		if e.StepType == "completion" {
			return e
		}
	}
	t.Fatal("no completion event")
	return capturedEvent{}
}

// Exhausting the budget ends with one tools-disabled call whose answer becomes the run output.
func TestRunAgentLoop_WrapsUpAtStepBudget(t *testing.T) {
	t.Parallel()
	provider := &loopingLLM{tool: guardSafeTool, answer: "Listed customers; could not find credit limits."}
	result, events := runLoopWithBudget(t, provider, nil, 3)

	if got := len(provider.requests); got != 4 {
		t.Fatalf("expected 3 steps plus a wrap-up call, got %d calls", got)
	}
	wrap := provider.requests[3]
	if wrap.ToolChoice != llm.ToolChoiceNone {
		t.Error("wrap-up call must disable tools")
	}
	if len(wrap.Tools) == 0 {
		t.Error("wrap-up call must keep tool definitions, since the history has tool_use blocks")
	}
	if !strings.Contains(lastMessageText(wrap), "maximum number of steps") {
		t.Error("wrap-up call must carry the max-steps instruction")
	}

	var out map[string]string
	_ = json.Unmarshal(result.Output, &out)
	if out["response"] != provider.answer {
		t.Errorf("output = %q, want the wrap-up answer", out["response"])
	}
	if hit, _ := completionEvent(t, events).Metadata["maxIterationsHit"].(bool); !hit {
		t.Error("completion metadata should keep maxIterationsHit=true")
	}
}

// Three quarters through the budget the agent is told how many steps remain.
func TestRunAgentLoop_RemindsNearBudget(t *testing.T) {
	t.Parallel()
	provider := &loopingLLM{tool: guardSafeTool, answer: "done"}
	runLoopWithBudget(t, provider, nil, 4)

	// stepsReminderAt(4) == 3, so the 4th request is the first to see the reminder.
	if strings.Contains(lastMessageText(provider.requests[2]), "steps left") {
		t.Error("reminder arrived too early")
	}
	if !strings.Contains(lastMessageText(provider.requests[3]), "You have 1 steps left") {
		t.Errorf("4th request should carry the remaining-steps reminder, got %q", lastMessageText(provider.requests[3]))
	}
}

// An agent whose calls keep failing is steered once, then wrapped up early instead of burning the budget.
func TestRunAgentLoop_NoProgressWrapsUpEarly(t *testing.T) {
	t.Parallel()
	provider := &loopingLLM{tool: guardSafeTool, answer: "I couldn't reach the data."}
	result, events := runLoopWithBudget(t, provider, errors.New("HTTP 404"), 30)

	if got := len(provider.requests); got != noProgressWrapUpAfter+1 {
		t.Fatalf("expected %d failing steps plus a wrap-up, got %d calls", noProgressWrapUpAfter, got)
	}
	if !strings.Contains(lastMessageText(provider.requests[noProgressSteerAfter]), noProgressSteeringPrompt) {
		t.Error("the request after the steer threshold should carry the steering message")
	}
	if !strings.Contains(lastMessageText(provider.requests[noProgressWrapUpAfter]), "stopped making progress") {
		t.Error("early wrap-up should explain why tools were disabled")
	}
	meta := completionEvent(t, events).Metadata
	if stop, _ := meta["noProgressStop"].(bool); !stop {
		t.Error("completion metadata should mark the no-progress stop")
	}
	var out map[string]string
	_ = json.Unmarshal(result.Output, &out)
	if out["response"] != provider.answer {
		t.Errorf("output = %q, want the wrap-up answer", out["response"])
	}
}

func TestResolveMaxSteps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		configured int
		trigger    string
		want       int
	}{
		{0, "chat", defaultInteractiveMaxSteps},
		{0, "manual", defaultInteractiveMaxSteps},
		{0, "scheduled", defaultBackgroundMaxSteps},
		{0, "event", defaultBackgroundMaxSteps},
		{12, "event", 12},
		{500, "chat", maxStepsCeiling},
	}
	for _, tc := range cases {
		if got := resolveMaxSteps(tc.configured, tc.trigger); got != tc.want {
			t.Errorf("resolveMaxSteps(%d, %q) = %d, want %d", tc.configured, tc.trigger, got, tc.want)
		}
	}
}

func TestValidateAgentConfig_MaxSteps(t *testing.T) {
	t.Parallel()
	for _, cfg := range []string{`{}`, `{"max_steps":1}`, `{"max_steps":60}`, `{"max_steps":null}`} {
		if err := validateAgentConfig(cfg); err != nil {
			t.Errorf("%s should be valid, got %v", cfg, err)
		}
	}
	for _, cfg := range []string{`{"max_steps":0}`, `{"max_steps":61}`, `{"max_steps":-3}`, `{"max_steps":"ten"}`} {
		if err := validateAgentConfig(cfg); err == nil {
			t.Errorf("%s should be rejected", cfg)
		}
	}
}
