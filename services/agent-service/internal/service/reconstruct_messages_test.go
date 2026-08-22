package service

import (
	"encoding/json"
	"testing"

	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
)

// toolCallEvent builds the event the runner emits when the model requests a tool.
func toolCallEvent(toolUseID, toolName string) sqlc.AgentRunEvent {
	meta, _ := json.Marshal(map[string]any{"tool_use_id": toolUseID, "tool_name": toolName, "input": json.RawMessage(`{}`)})
	return sqlc.AgentRunEvent{StepType: "tool_call", Metadata: meta}
}

// toolTerminalEvent builds a tool-terminating event of the given step type with its message carried
// in event.Content (how tool_denied / doom_loop_detected store it) and the id in metadata.
func toolTerminalEvent(stepType, toolUseID, content string) sqlc.AgentRunEvent {
	meta, _ := json.Marshal(map[string]any{"tool_use_id": toolUseID, "tool_name": "x"})
	return sqlc.AgentRunEvent{StepType: stepType, Metadata: meta, Content: agentdb.PgText(content)}
}

func toolResultEvent(toolUseID, result string) sqlc.AgentRunEvent {
	meta, _ := json.Marshal(map[string]any{"tool_use_id": toolUseID, "is_error": false, "full_result": result})
	return sqlc.AgentRunEvent{StepType: "tool_result", Metadata: meta}
}

// assertNoOrphanToolUse fails if any assistant tool_use lacks a matching tool_result in the very next
// message — the exact invariant the Anthropic API enforces.
func assertNoOrphanToolUse(t *testing.T, messages []llm.Message) {
	t.Helper()
	for i, msg := range messages {
		if msg.Role != "assistant" || len(msg.ToolUse) == 0 {
			continue
		}
		answered := map[string]bool{}
		if i+1 < len(messages) && messages[i+1].Role == "user" {
			for _, tr := range messages[i+1].ToolResults {
				answered[tr.ToolUseID] = true
			}
		}
		for _, tu := range msg.ToolUse {
			if !answered[tu.ID] {
				t.Errorf("message %d: tool_use %q has no tool_result in the next message", i, tu.ID)
			}
		}
	}
}

func TestReconstructMessages_ToolDeniedIsAnswered(t *testing.T) {
	t.Parallel()
	events := []sqlc.AgentRunEvent{
		{StepType: "user_message", Content: agentdb.PgText("do the thing")},
		toolCallEvent("toolu_1", "secret_tool"),
		toolTerminalEvent("tool_denied", "toolu_1", `Tool "secret_tool" is not available to this agent.`),
	}
	messages := reconstructMessages(events)
	assertNoOrphanToolUse(t, messages)
}

func TestReconstructMessages_DoomLoopIsAnswered(t *testing.T) {
	t.Parallel()
	events := []sqlc.AgentRunEvent{
		{StepType: "user_message", Content: agentdb.PgText("go")},
		toolCallEvent("toolu_1", "search"),
		toolTerminalEvent("doom_loop_detected", "toolu_1", "Called search 3 times with identical input."),
	}
	messages := reconstructMessages(events)
	assertNoOrphanToolUse(t, messages)
}

// The reported failure: a turn with two tool calls where the first is denied and the second succeeds.
// The denied call's result was dropped on reconstruction, leaving toolu_1 orphaned -> gateway 400.
func TestReconstructMessages_DeniedThenSucceededBothAnswered(t *testing.T) {
	t.Parallel()
	events := []sqlc.AgentRunEvent{
		{StepType: "user_message", Content: agentdb.PgText("go")},
		toolCallEvent("toolu_1", "secret_tool"),
		toolTerminalEvent("tool_denied", "toolu_1", "not available"),
		toolCallEvent("toolu_2", "search"),
		toolResultEvent("toolu_2", "ok"),
	}
	messages := reconstructMessages(events)
	assertNoOrphanToolUse(t, messages)
}

// Safety net: a tool_call with no terminal event at all (e.g. the run died mid-turn) must still be
// answered with a synthetic placeholder, otherwise it bricks every future continuation.
func TestReconstructMessages_OrphanToolCallBackfilled(t *testing.T) {
	t.Parallel()
	events := []sqlc.AgentRunEvent{
		{StepType: "user_message", Content: agentdb.PgText("go")},
		toolCallEvent("toolu_1", "search"),
		// no result event — run crashed here
		{StepType: "user_message", Content: agentdb.PgText("are you there?")},
	}
	messages := reconstructMessages(events)
	assertNoOrphanToolUse(t, messages)
}

func TestEnsureToolResults_LeavesValidHistoryUnchanged(t *testing.T) {
	t.Parallel()
	in := []llm.Message{
		{Role: "user", Content: "go"},
		{Role: "assistant", ToolUse: []llm.ToolUseBlock{{ID: "toolu_1", Name: "search"}}},
		{Role: "user", ToolResults: []llm.ToolResultBlock{{ToolUseID: "toolu_1", Content: "ok"}}},
		{Role: "assistant", Content: "done"},
	}
	out := ensureToolResults(in)
	if len(out) != len(in) {
		t.Fatalf("expected unchanged length %d, got %d", len(in), len(out))
	}
	assertNoOrphanToolUse(t, out)
}
