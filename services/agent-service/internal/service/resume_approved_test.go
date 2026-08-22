package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/agents"
	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
)

// blockedTranscript builds the reconstructed messages for a single tool call that paused for approval: an assistant tool_use answered by the "[REQUIRES APPROVAL]" placeholder the guard writes.
func blockedTranscript(toolUseID, name, input string) []llm.Message {
	return []llm.Message{
		{Role: "assistant", ToolUse: []llm.ToolUseBlock{{ID: toolUseID, Name: name, Input: json.RawMessage(input)}}},
		{Role: "user", ToolResults: []llm.ToolResultBlock{{ToolUseID: toolUseID, Content: "[REQUIRES APPROVAL] ..."}}},
	}
}

func blockedToolEvent(toolUseID, name, input string) sqlc.AgentRunEvent {
	meta, _ := json.Marshal(map[string]any{"tool_use_id": toolUseID, "tool_name": name, "input": json.RawMessage(input)})
	return sqlc.AgentRunEvent{StepType: "tool_blocked", Metadata: meta}
}

// resumeRunCtx builds a run context whose only armed approval is the given slug.
func resumeRunCtx(approvedSlug string) *domain.HandlerRunContext {
	return &domain.HandlerRunContext{
		AccountID:            "acc_test",
		RunID:                "agr_test",
		RequireReviewBySlug:  map[string]bool{approvedSlug: true},
		AlwaysAllowedSlugs:   map[string]bool{},
		OneTimeApprovedSlugs: map[string]bool{approvedSlug: true},
		OneTimeApprovedKeys:  map[string]bool{},
	}
}

// TestResumeApprovedBlockedCalls_ExecutesAndRewrites is the core fix: an approved-but-blocked call runs on resume (rather than depending on the model to re-issue it) and its placeholder result is replaced with the real output.
func TestResumeApprovedBlockedCalls_ExecutesAndRewrites(t *testing.T) {
	ran := 0
	registry := agents.NewToolHandlerRegistry()
	registry.Register(guardGatedTool, func(_ context.Context, _ json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
		ran++
		return `{"id":"ac_1","status":"hold_all"}`, nil
	})
	runner := newGuardTestRunner(t, &scriptedLLM{}, registry)

	messages := blockedTranscript("tu_1", guardGatedTool, `{"status":"hold_all"}`)
	events := []sqlc.AgentRunEvent{blockedToolEvent("tu_1", guardGatedTool, `{"status":"hold_all"}`)}
	runCtx := resumeRunCtx(guardGatedTool)
	seq := 0

	runner.resumeApprovedBlockedCalls(context.Background(), &sqlc.AgentRun{ID: "agr_test"}, "acc_test", &seq, runCtx, messages, events)

	if ran != 1 {
		t.Fatalf("approved blocked tool should run exactly once, ran %d times", ran)
	}
	got := messages[1].ToolResults[0]
	if got.IsError {
		t.Errorf("rewritten result should not be an error: %+v", got)
	}
	if got.Content == "[REQUIRES APPROVAL] ..." {
		t.Error("placeholder result was not rewritten with the real tool output")
	}
	// The consumed approval must not stay armed, or a stray re-issue would double-write.
	if runCtx.OneTimeApprovedSlugs[guardGatedTool] {
		t.Error("approval should be consumed after auto-execution")
	}
}

// TestResumeApprovedBlockedCalls_SkipsUnapproved guards the review gate: a blocked call the resume did not approve must not run.
func TestResumeApprovedBlockedCalls_SkipsUnapproved(t *testing.T) {
	ran := 0
	registry := agents.NewToolHandlerRegistry()
	registry.Register(guardGatedTool, func(_ context.Context, _ json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
		ran++
		return `{}`, nil
	})
	runner := newGuardTestRunner(t, &scriptedLLM{}, registry)

	messages := blockedTranscript("tu_1", guardGatedTool, `{"status":"hold_all"}`)
	events := []sqlc.AgentRunEvent{blockedToolEvent("tu_1", guardGatedTool, `{"status":"hold_all"}`)}
	runCtx := &domain.HandlerRunContext{
		AccountID: "acc_test", RunID: "agr_test",
		RequireReviewBySlug:  map[string]bool{guardGatedTool: true},
		AlwaysAllowedSlugs:   map[string]bool{},
		OneTimeApprovedSlugs: map[string]bool{}, // no approval armed
		OneTimeApprovedKeys:  map[string]bool{},
	}
	seq := 0

	runner.resumeApprovedBlockedCalls(context.Background(), &sqlc.AgentRun{ID: "agr_test"}, "acc_test", &seq, runCtx, messages, events)

	if ran != 0 {
		t.Fatalf("unapproved blocked tool must not run, ran %d times", ran)
	}
	if messages[1].ToolResults[0].Content != "[REQUIRES APPROVAL] ..." {
		t.Error("unapproved placeholder should be left intact for the guard to re-block")
	}
}

// TestResumeApprovedBlockedCalls_Idempotent ensures a call an earlier resume already executed (it has a tool_result event) is not run a second time.
func TestResumeApprovedBlockedCalls_Idempotent(t *testing.T) {
	ran := 0
	registry := agents.NewToolHandlerRegistry()
	registry.Register(guardGatedTool, func(_ context.Context, _ json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
		ran++
		return `{}`, nil
	})
	runner := newGuardTestRunner(t, &scriptedLLM{}, registry)

	messages := blockedTranscript("tu_1", guardGatedTool, `{"status":"hold_all"}`)
	events := []sqlc.AgentRunEvent{
		blockedToolEvent("tu_1", guardGatedTool, `{"status":"hold_all"}`),
		toolResultEvent("tu_1", `{"ok":true}`), // already executed on a prior turn
	}
	runCtx := resumeRunCtx(guardGatedTool)
	seq := 0

	runner.resumeApprovedBlockedCalls(context.Background(), &sqlc.AgentRun{ID: "agr_test"}, "acc_test", &seq, runCtx, messages, events)

	if ran != 0 {
		t.Fatalf("already-executed call must not run again, ran %d times", ran)
	}
}

// TestResumeApprovedBlockedCalls_ErrorSurfacesAsIsError verifies a failing approved call is rewritten as an error result, so the model can't report a failed write as success.
func TestResumeApprovedBlockedCalls_ErrorSurfacesAsIsError(t *testing.T) {
	registry := agents.NewToolHandlerRegistry()
	registry.Register(guardGatedTool, func(_ context.Context, _ json.RawMessage, _ *domain.HandlerRunContext) (string, error) {
		return "", errors.New("gateway returned HTTP 422: validation_failed")
	})
	runner := newGuardTestRunner(t, &scriptedLLM{}, registry)

	messages := blockedTranscript("tu_1", guardGatedTool, `{"status":"bogus"}`)
	events := []sqlc.AgentRunEvent{blockedToolEvent("tu_1", guardGatedTool, `{"status":"bogus"}`)}
	runCtx := resumeRunCtx(guardGatedTool)
	seq := 0

	runner.resumeApprovedBlockedCalls(context.Background(), &sqlc.AgentRun{ID: "agr_test"}, "acc_test", &seq, runCtx, messages, events)

	got := messages[1].ToolResults[0]
	if !got.IsError {
		t.Errorf("failed approved call must be rewritten as an error result: %+v", got)
	}
}

// TestReconstructMessages_BlockedThenResultOverwrites proves a later tool_result overwrites an earlier tool_blocked placeholder for the same tool_use_id, leaving a single result block — otherwise the duplicate id would 400 on the next resume.
func TestReconstructMessages_BlockedThenResultOverwrites(t *testing.T) {
	events := []sqlc.AgentRunEvent{
		toolCallEvent("tu_1", guardGatedTool),
		toolTerminalEvent("tool_blocked", "tu_1", "[REQUIRES APPROVAL] ..."),
		toolResultEvent("tu_1", `{"status":"hold_all"}`),
	}

	messages := reconstructMessages(events)
	assertNoOrphanToolUse(t, messages)

	var results []llm.ToolResultBlock
	for _, m := range messages {
		results = append(results, m.ToolResults...)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly one tool_result block for tu_1, got %d: %+v", len(results), results)
	}
	if results[0].Content != `{"status":"hold_all"}` {
		t.Errorf("blocked placeholder should be overwritten by the real result, got %q", results[0].Content)
	}
}
