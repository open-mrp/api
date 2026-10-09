package llm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// ToolChoiceNone disables tool calls but keeps the definitions the history's tool_use blocks need.
func TestBuildParams_ToolChoiceNone(t *testing.T) {
	t.Parallel()
	p := NewAnthropicMessagesProvider("sk_test_fake")
	req := &ToolRequest{
		Model:      "claude-sonnet-4.6",
		Messages:   []Message{{Role: "user", Content: "Hi"}},
		Tools:      []ToolDefinition{{Name: "t", Description: "d", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		ToolChoice: ToolChoiceNone,
	}
	body, _ := json.Marshal(p.buildParams(req))
	if !strings.Contains(string(body), `"tool_choice":{"type":"none"}`) {
		t.Errorf("missing tool_choice none\nbody: %s", body)
	}
	if !strings.Contains(string(body), `"name":"t"`) {
		t.Errorf("tools must stay defined\nbody: %s", body)
	}

	req.ToolChoice = ToolChoiceAuto
	body, _ = json.Marshal(p.buildParams(req))
	if strings.Contains(string(body), `"tool_choice"`) {
		t.Errorf("auto must leave tool_choice unset\nbody: %s", body)
	}
}

func TestGatewayToolChoice(t *testing.T) {
	t.Parallel()
	tools := []ToolDefinition{{Name: "t"}}
	if got := gatewayToolChoice(&ToolRequest{Tools: tools, ToolChoice: ToolChoiceNone}); got != "none" {
		t.Errorf("got %q, want none", got)
	}
	if got := gatewayToolChoice(&ToolRequest{ToolChoice: ToolChoiceNone}); got != "" {
		t.Errorf("without tools tool_choice must be omitted, got %q", got)
	}
}

// The tools, system prompt, and newest message each carry a cache breakpoint, within the API's limit of 4.
func TestBuildParams_CacheBreakpoints(t *testing.T) {
	t.Parallel()
	p := NewAnthropicMessagesProvider("sk_test_fake")
	params := p.buildParams(&ToolRequest{
		Model:  "claude-sonnet-4.6",
		System: "You are helpful.",
		Tools: []ToolDefinition{
			{Name: "a", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "b", InputSchema: json.RawMessage(`{"type":"object"}`)},
		},
		Messages: []Message{
			{Role: "user", Content: "go"},
			{Role: "assistant", ToolUse: []ToolUseBlock{{ID: "toolu_1", Name: "a", Input: json.RawMessage(`{}`)}}},
			{Role: "user", ToolResults: []ToolResultBlock{{ToolUseID: "toolu_1", Content: "ok"}}},
		},
	})
	if params.Tools[0].OfTool.CacheControl.Type != "" || params.Tools[1].OfTool.CacheControl.Type == "" {
		t.Error("only the last tool should carry a breakpoint")
	}
	if params.System[0].CacheControl.Type == "" {
		t.Error("system prompt should carry a breakpoint")
	}
	last := params.Messages[len(params.Messages)-1].Content
	if cc := last[len(last)-1].GetCacheControl(); cc == nil || cc.Type == "" {
		t.Error("newest message should carry a breakpoint")
	}
	body, _ := json.Marshal(params)
	if n := strings.Count(string(body), `"cache_control"`); n != 3 {
		t.Errorf("want 3 breakpoints, got %d", n)
	}
}

// InputTokens counts the whole prompt; the cache split is reported alongside it.
func TestToolResponseFromMessage_CacheUsage(t *testing.T) {
	t.Parallel()
	var msg anthropic.Message
	if err := json.Unmarshal([]byte(`{"content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":900,"cache_creation_input_tokens":100}}`), &msg); err != nil {
		t.Fatal(err)
	}
	resp := toolResponseFromMessage(&msg)
	if resp.InputTokens != 1010 || resp.CacheReadInputTokens != 900 || resp.CacheCreationInputTokens != 100 {
		t.Errorf("got input=%d read=%d write=%d", resp.InputTokens, resp.CacheReadInputTokens, resp.CacheCreationInputTokens)
	}
}
