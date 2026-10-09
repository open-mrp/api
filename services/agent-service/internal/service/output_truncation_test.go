package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
)

type truncatingLLM struct{ stopReason string }

func (f *truncatingLLM) Name() string { return "anthropic" }

func (f *truncatingLLM) CompleteWithTools(context.Context, *llm.ToolRequest) (*llm.ToolResponse, error) {
	return &llm.ToolResponse{StopReason: f.stopReason, Content: "| SKU | lbs |\n| 00013-Z | 0.0"}, nil
}

func runTruncationLoop(t *testing.T, stopReason string) (string, []capturedEvent) {
	t.Helper()
	runner, captured := newCapturingRunner(t, &truncatingLLM{stopReason: stopReason}, nil)
	seq := 0
	run := &sqlc.AgentRun{ID: "agr_test", StatusCode: domain.RunStatusRunning}
	result, err := runner.runAgentLoop(context.Background(), run, "acc_test", nil,
		"sys", []string{"claude-test"}, nil, 0,
		[]llm.Message{{Role: "user", Content: "go"}}, &seq, newGuardRunCtx(nil, nil), nil, 0, nil)
	if err != nil {
		t.Fatalf("runAgentLoop: %v", err)
	}
	var out map[string]string
	_ = json.Unmarshal(result.Output, &out)
	return out["response"], *captured
}

func hasEvent(events []capturedEvent, stepType string) bool {
	for _, e := range events {
		if e.StepType == stepType {
			return true
		}
	}
	return false
}

func TestRunAgentLoop_FlagsAnswerCutOffAtMaxTokens(t *testing.T) {
	t.Parallel()
	got, events := runTruncationLoop(t, "max_tokens")
	if !strings.HasSuffix(got, truncatedAnswerNotice) {
		t.Errorf("response = %q, want it to end with the truncation notice", got)
	}
	if !hasEvent(events, "output_truncated") {
		t.Error("expected an output_truncated event")
	}
}

func TestRunAgentLoop_CompleteAnswerIsUnchanged(t *testing.T) {
	t.Parallel()
	got, events := runTruncationLoop(t, "end_turn")
	if strings.Contains(got, truncatedAnswerNotice) {
		t.Errorf("response = %q, should not carry the truncation notice", got)
	}
	if hasEvent(events, "output_truncated") {
		t.Error("unexpected output_truncated event")
	}
}
