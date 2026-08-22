package service

import (
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/llm"
)

func TestNeedsCompaction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		inputTokens int
		model       string
		want        bool
	}{
		{"well under limit", 50_000, "claude-sonnet-4", false},
		{"approaching limit", 165_000, "claude-sonnet-4", true},
		{"at limit", 180_000, "claude-sonnet-4", true},
		{"unknown model uses default", 165_000, "unknown-model", true},
		{"gpt-4o under limit", 80_000, "gpt-4o", false},
		{"gpt-4o approaching limit", 100_000, "gpt-4o", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsCompaction(tt.inputTokens, tt.model)
			if got != tt.want {
				t.Errorf("needsCompaction(%d, %q) = %v, want %v", tt.inputTokens, tt.model, got, tt.want)
			}
		})
	}
}

func TestPruneOldToolResults(t *testing.T) {
	t.Parallel()
	longResult := strings.Repeat("x", 4000)

	messages := []llm.Message{
		{Role: "user", Content: "initial task"},
		// Turn 1 - old
		{Role: "assistant", Content: "thinking", ToolUse: []llm.ToolUseBlock{{ID: "1", Name: "search"}}},
		{Role: "user", ToolResults: []llm.ToolResultBlock{{ToolUseID: "1", Content: longResult}}},
		// Turn 2 - old
		{Role: "assistant", Content: "more thinking", ToolUse: []llm.ToolUseBlock{{ID: "2", Name: "search"}}},
		{Role: "user", ToolResults: []llm.ToolResultBlock{{ToolUseID: "2", Content: longResult}}},
		// Turn 3 - recent (second-to-last user turn)
		{Role: "user", Content: "follow up"},
		{Role: "assistant", Content: "response", ToolUse: []llm.ToolUseBlock{{ID: "3", Name: "search"}}},
		{Role: "user", ToolResults: []llm.ToolResultBlock{{ToolUseID: "3", Content: longResult}}},
		// Turn 4 - recent (last user turn)
		{Role: "user", Content: "another question"},
	}

	freed := pruneOldToolResults(messages)

	if freed <= 0 {
		t.Error("expected some tokens to be freed")
	}

	// Old tool results should be pruned.
	if messages[2].ToolResults[0].Content != prunedPlaceholder {
		t.Error("expected first old tool result to be pruned")
	}
	if messages[4].ToolResults[0].Content != prunedPlaceholder {
		t.Error("expected second old tool result to be pruned")
	}

	// Recent tool results should be preserved.
	if messages[7].ToolResults[0].Content == prunedPlaceholder {
		t.Error("expected recent tool result to be preserved")
	}
}

func TestPruneOldToolResults_TooFewMessages(t *testing.T) {
	t.Parallel()
	messages := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	freed := pruneOldToolResults(messages)
	if freed != 0 {
		t.Errorf("expected 0 tokens freed for short conversation, got %d", freed)
	}
}

func TestPruneOldToolResults_SkipsAlreadyPruned(t *testing.T) {
	t.Parallel()
	messages := []llm.Message{
		{Role: "user", Content: "task"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", ToolResults: []llm.ToolResultBlock{{ToolUseID: "1", Content: prunedPlaceholder}}},
		{Role: "user", Content: "recent 1"},
		{Role: "user", Content: "recent 2"},
	}
	freed := pruneOldToolResults(messages)
	if freed != 0 {
		t.Errorf("expected 0 tokens freed when already pruned, got %d", freed)
	}
}
