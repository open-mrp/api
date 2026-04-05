package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"a", 1},
		{"abcd", 1},
		{"abcde", 2},
		{strings.Repeat("x", 100), 25},
	}
	for _, tt := range tests {
		got := EstimateTokens(tt.input)
		if got != tt.expected {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestTruncateMessages_FitsWithinBudget(t *testing.T) {
	t.Parallel()
	messages := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	result := TruncateMessages("short system", messages, nil, "claude-sonnet-4")
	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}
	if result[1].Content != "hi there" {
		t.Errorf("expected unmodified assistant content, got %q", result[1].Content)
	}
}

func TestTruncateMessages_TruncatesAssistantContentFirst(t *testing.T) {
	t.Parallel(
	// Each assistant message is 7000 chars = ~1750 tokens.
	// With 5 messages, total message tokens ~3600.
	// Set system prompt so total just barely exceeds budget.
	)

	longThinking := strings.Repeat("Let me think about this carefully. ", 200) // ~7000 chars
	messages := []Message{
		{Role: "user", Content: "What should I do?"},
		{Role: "assistant", Content: longThinking},
		{Role: "user", Content: "Thanks, now do the next step"},
		{Role: "assistant", Content: longThinking},
		{Role: "user", Content: "And one more thing"},
	}

	// Total message chars ~14200 => ~3550 tokens.
	// gpt-4o-mini budget = 115000. System needs to eat most of that.
	// Budget = 115000 - system_tokens - 4096.
	// We want budget ~ 2500 tokens so assistant content must be truncated.
	// system_tokens = 115000 - 4096 - 2500 = 108404 tokens => ~433616 chars
	systemPrompt := strings.Repeat("x", 433616)

	result := TruncateMessages(systemPrompt, messages, nil, "gpt-4o-mini")

	// User messages should be intact.
	for _, m := range result {
		if m.Role == "user" && strings.Contains(m.Content, "[") {
			t.Errorf("user content should not be truncated, got: %s", m.Content)
		}
		if m.Role == "assistant" && !strings.Contains(m.Content, "[assistant reasoning truncated") {
			if len(m.Content) > maxAssistantContentChars+200 {
				t.Errorf("assistant content should have been truncated, len=%d", len(m.Content))
			}
		}
	}

	// All 5 original messages should still be present (content truncated, not removed).
	nonPlaceholder := 0
	for _, m := range result {
		if !strings.Contains(m.Content, "[") || m.Role != "user" {
			nonPlaceholder++
		}
	}
	if nonPlaceholder < 5 {
		t.Errorf("expected all 5 messages preserved (with truncated content), got %d", nonPlaceholder)
	}
}

func TestTruncateMessages_CapsToolResults(t *testing.T) {
	t.Parallel()
	longResult := strings.Repeat("a", maxToolResultCharsRecent+5000) // 25000 chars
	messages := []Message{
		{Role: "user", Content: "do something"},
		{
			Role: "assistant",
			ToolUse: []ToolUseBlock{
				{ID: "1", Name: "search", Input: json.RawMessage(`{"q":"test"}`)},
			},
		},
		{
			Role: "user",
			ToolResults: []ToolResultBlock{
				{ToolUseID: "1", Content: longResult},
			},
		},
		{Role: "user", Content: "what did you find?"},
	}

	// Budget needs to be tight enough that the long tool result causes overflow.
	// Messages total ~25200 chars => ~6300 tokens.
	// Set system so budget is ~5500 tokens (fits after capping but not before).
	// system_tokens = 180000 - 4096 - 5500 = 170404 => ~681616 chars
	systemPrompt := strings.Repeat("x", 681616)

	result := TruncateMessages(systemPrompt, messages, nil, "claude-sonnet-4")

	for _, m := range result {
		for _, tr := range m.ToolResults {
			if len(tr.Content) > maxToolResultCharsRecent+200 {
				t.Errorf("tool result should have been capped, len=%d", len(tr.Content))
			}
		}
	}
}

func TestTruncateMessages_PreservesUserMessages(t *testing.T) {
	t.Parallel()
	var messages []Message
	messages = append(messages, Message{Role: "user", Content: "initial request"})

	for range 30 {
		messages = append(messages, Message{
			Role:    "assistant",
			Content: strings.Repeat("thinking ", 200),
			ToolUse: []ToolUseBlock{
				{ID: "tc", Name: "search", Input: json.RawMessage(`{"q":"test"}`)},
			},
		})
		messages = append(messages, Message{
			Role: "user",
			ToolResults: []ToolResultBlock{
				{ToolUseID: "tc", Content: strings.Repeat("result ", 500)},
			},
		})
	}
	messages = append(messages, Message{Role: "user", Content: "user followup question"})
	messages = append(messages, Message{Role: "assistant", Content: strings.Repeat("more thinking ", 200)})
	messages = append(messages, Message{Role: "user", Content: "final user message"})

	result := TruncateMessages("system", messages, nil, "gpt-4o-mini")

	// First and last user messages must be present.
	foundFirst := false
	foundLast := false
	for _, m := range result {
		if m.Content == "initial request" {
			foundFirst = true
		}
		if m.Content == "final user message" {
			foundLast = true
		}
	}
	if !foundFirst {
		t.Error("first user message was not preserved")
	}
	if !foundLast {
		t.Error("last user message was not preserved")
	}
}

func TestTruncateMessages_DropOldNonUserMessagesBeforeUserMessages(t *testing.T) {
	t.Parallel(
	// Create a conversation where dropping assistant messages should suffice.
	)

	messages := []Message{
		{Role: "user", Content: "start"},
		{Role: "assistant", Content: strings.Repeat("old reasoning ", 2000)},
		{Role: "user", Content: "important user context"},
		{Role: "assistant", Content: strings.Repeat("more old reasoning ", 2000)},
		{Role: "user", Content: "another important point"},
		{Role: "assistant", Content: "recent answer"},
		{Role: "user", Content: "recent question"},
	}

	// After assistant truncation (pass 1), content is capped at 2000 chars each.
	// If still over budget, pass 4 drops non-user messages.
	// Budget = 180000 - system_tokens - 4096
	// Total after pass 1: ~4200 chars user + ~4400 chars assistant = ~8600 chars => ~2150 tokens
	// System = 180000 - 4096 - 2000 = 173904 tokens => ~695616 chars
	systemPrompt := strings.Repeat("x", 695616)

	result := TruncateMessages(systemPrompt, messages, nil, "claude-sonnet-4")

	// Count user messages vs assistant messages (excluding placeholders).
	userCount := 0
	assistantCount := 0
	for _, m := range result {
		switch m.Role {
		case "user":
			if !strings.Contains(m.Content, "[") {
				userCount++
			}
		case "assistant":
			assistantCount++
		}
	}

	if userCount < assistantCount {
		t.Errorf("user messages (%d) should be prioritized over assistant messages (%d)", userCount, assistantCount)
	}
}

func TestTruncateMessages_NoMutationOfOriginal(t *testing.T) {
	t.Parallel()
	original := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: strings.Repeat("thinking ", 2000)},
	}
	origContent := original[1].Content

	// Use system prompt large enough to trigger truncation.
	systemPrompt := strings.Repeat("x", 700000)
	TruncateMessages(systemPrompt, original, nil, "claude-sonnet-4")

	if original[1].Content != origContent {
		t.Error("original messages were mutated")
	}
}

func TestTruncateMessages_HandlesEmptyMessages(t *testing.T) {
	t.Parallel()
	result := TruncateMessages("system", nil, nil, "claude-sonnet-4")
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestTruncateMessages_HandlesSingleMessage(t *testing.T) {
	t.Parallel()
	messages := []Message{
		{Role: "user", Content: "just one message"},
	}
	result := TruncateMessages("system", messages, nil, "claude-sonnet-4")
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}
