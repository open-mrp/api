package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// assertValidAnthropicTranscript converts messages exactly as a request would and checks the invariants the
// Messages API enforces: roles alternate starting with user, tool_result blocks lead their user turn and answer
// a tool_use in the assistant turn right before it, and every tool_use except the newest is answered.
func assertValidAnthropicTranscript(t *testing.T, messages []Message) {
	t.Helper()
	params := convertMessagesToAnthropic(messages)
	if len(params) == 0 {
		t.Fatal("transcript converted to no messages")
	}
	if params[0].Role != anthropic.MessageParamRoleUser {
		t.Errorf("first message role = %s, want user", params[0].Role)
	}
	for i, p := range params {
		if i > 0 && p.Role == params[i-1].Role {
			t.Errorf("messages %d and %d are both %s; roles must alternate", i-1, i, p.Role)
		}
		if p.Role != anthropic.MessageParamRoleUser {
			continue
		}
		asked := map[string]bool{}
		if i > 0 {
			for _, b := range params[i-1].Content {
				if b.OfToolUse != nil {
					asked[b.OfToolUse.ID] = true
				}
			}
		}
		seenOther := false
		for _, b := range p.Content {
			if b.OfToolResult == nil {
				seenOther = true
				continue
			}
			if seenOther {
				t.Errorf("message %d: tool_result follows other content; tool_result blocks must come first", i)
			}
			if !asked[b.OfToolResult.ToolUseID] {
				t.Errorf("message %d: tool_result %q has no matching tool_use in the preceding assistant turn", i, b.OfToolResult.ToolUseID)
			}
			delete(asked, b.OfToolResult.ToolUseID)
		}
		for id := range asked {
			t.Errorf("tool_use %q in message %d is never answered", id, i-1)
		}
	}
}

// toolLoopTranscript is a task followed by n tool exchanges whose results are resultChars long.
func toolLoopTranscript(n, resultChars int) []Message {
	msgs := []Message{{Role: "user", Content: "find the order"}}
	for i := range n {
		id := fmt.Sprintf("toolu_%d", i)
		msgs = append(msgs,
			Message{Role: "assistant", Content: "checking", ToolUse: []ToolUseBlock{{ID: id, Name: "list_orders", Input: json.RawMessage(fmt.Sprintf(`{"page":%d}`, i))}}},
			Message{Role: "user", ToolResults: []ToolResultBlock{{ToolUseID: id, Content: strings.Repeat("r", resultChars)}}},
		)
	}
	return msgs
}

func TestCompactedHistory_KeepsLastToolExchange(t *testing.T) {
	t.Parallel()
	msgs := toolLoopTranscript(5, 100)
	got := CompactedHistory(Message{Role: "user", Content: "[summary]"}, msgs)

	if len(got) != 3 || got[0].Content != "[summary]" || got[1].ToolUse[0].ID != "toolu_4" || got[2].ToolResults[0].ToolUseID != "toolu_4" {
		t.Fatalf("want summary + last assistant tool_use + its results, got %+v", got)
	}
	assertValidAnthropicTranscript(t, got)
}

func TestCompactedHistory_TrailingUserText(t *testing.T) {
	t.Parallel()
	msgs := append(toolLoopTranscript(2, 100),
		Message{Role: "assistant", Content: "Found it."},
		Message{Role: "user", Content: "Now cancel it."},
	)
	got := CompactedHistory(Message{Role: "user", Content: "[summary]"}, msgs)
	if len(got) != 2 || got[1].Content != "Now cancel it." {
		t.Fatalf("want summary + the latest user message, got %+v", got)
	}
	assertValidAnthropicTranscript(t, got)
}

// Dropping old exchanges must remove a tool_use together with its results, at any budget.
func TestDropOldNonUserMessages_NeverSplitsToolExchanges(t *testing.T) {
	t.Parallel()
	msgs := toolLoopTranscript(12, 4000)
	total := EstimateAllMessages(msgs)
	for budget := total / 10; budget < total; budget += total / 37 {
		got := dropOldNonUserMessages(CopyMessages(msgs), budget)
		t.Run(fmt.Sprintf("budget_%d", budget), func(t *testing.T) {
			assertValidAnthropicTranscript(t, got)
		})
	}
}

// The last-resort pass keeps a recent tail; a tail that starts mid-exchange must not keep the orphaned results.
func TestDropOldMessages_NeverOrphansToolResults(t *testing.T) {
	t.Parallel()
	msgs := toolLoopTranscript(12, 4000)
	total := EstimateAllMessages(msgs)
	for budget := total / 10; budget < total; budget += total / 41 {
		got := dropOldMessages(CopyMessages(msgs), budget)
		t.Run(fmt.Sprintf("budget_%d", budget), func(t *testing.T) {
			assertValidAnthropicTranscript(t, got)
		})
	}
}

// End to end through TruncateMessages with a prompt that forces the drop passes.
func TestTruncateMessages_ResultIsValidTranscript(t *testing.T) {
	t.Parallel()
	msgs := toolLoopTranscript(1200, 1000)
	got := TruncateMessages("sys", msgs, nil, "claude-sonnet-4")
	if len(got) >= len(msgs) {
		t.Fatal("expected truncation to drop messages")
	}
	assertValidAnthropicTranscript(t, got)
}

func TestRepairToolPairs(t *testing.T) {
	t.Parallel()
	msgs := []Message{
		{Role: "user", Content: "go"},
		{Role: "user", ToolResults: []ToolResultBlock{{ToolUseID: "orphan", Content: "x"}}},
		{Role: "assistant", ToolUse: []ToolUseBlock{{ID: "a", Name: "t"}, {ID: "unanswered", Name: "t"}}},
		{Role: "user", ToolResults: []ToolResultBlock{{ToolUseID: "a", Content: "ok"}}},
		{Role: "assistant", ToolUse: []ToolUseBlock{{ID: "pending", Name: "t"}}},
	}
	got := RepairToolPairs(msgs)
	if len(got) != 4 {
		t.Fatalf("the emptied orphan message should be removed, got %d messages: %+v", len(got), got)
	}
	if len(got[1].ToolUse) != 1 || got[1].ToolUse[0].ID != "a" {
		t.Errorf("unanswered tool_use should be stripped, got %+v", got[1].ToolUse)
	}
	if len(got[3].ToolUse) != 1 {
		t.Error("the trailing tool_use is still awaiting results and must be kept")
	}
	if len(msgs[2].ToolUse) != 2 {
		t.Error("RepairToolPairs must not mutate its input")
	}
}

// A note appended after tool results merges into the same user turn, behind the results.
func TestConvertMessages_NoteAfterToolResultsMerges(t *testing.T) {
	t.Parallel()
	msgs := append(toolLoopTranscript(1, 10), Message{Role: "user", Content: "You have 2 steps left."})
	msgs[2].Content = "reminder on the results message"
	params := convertMessagesToAnthropic(msgs)
	if len(params) != 3 {
		t.Fatalf("want user/assistant/user, got %d messages", len(params))
	}
	assertValidAnthropicTranscript(t, msgs)
}

func TestCapToolInputs_KeepsValidJSON(t *testing.T) {
	t.Parallel()
	msgs := []Message{{Role: "assistant", ToolUse: []ToolUseBlock{{ID: "a", Name: "t", Input: json.RawMessage(`{"body":"` + strings.Repeat("x", maxToolInputChars*2) + `"}`)}}}}
	capToolInputs(msgs)
	if !json.Valid(msgs[0].ToolUse[0].Input) {
		t.Errorf("capped input must stay valid JSON, got %.80s", msgs[0].ToolUse[0].Input)
	}
}
