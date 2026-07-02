package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

// buildParams must enable adaptive thinking with summarized display, omit sampling params, and faithfully
// translate our messages — replaying signed thinking blocks (for interleaved thinking) while dropping
// unsigned ones, and carrying tool_use / tool_result blocks.
func TestBuildParams_ThinkingToolsAndReplay(t *testing.T) {
	t.Parallel()

	p := NewAnthropicMessagesProvider("sk_test_fake")
	req := &ToolRequest{
		Model:           "claude-sonnet-4.6", // 4.6+ → adaptive thinking
		System:          "You are helpful.",
		EnableReasoning: true,
		Temperature:     0.7, // must NOT appear — adaptive thinking models reject sampling params
		Tools: []ToolDefinition{{
			Name:        "get_weather",
			Description: "Get weather",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
		}},
		Messages: []Message{
			{Role: "user", Content: "Weather in Paris?"},
			{
				Role:    "assistant",
				Content: "Checking.",
				Thinking: []ThinkingBlock{
					{Text: "signed reasoning", Signature: "sig_abc"},
					{Text: "unsigned reasoning"}, // dropped: no signature
				},
				ToolUse: []ToolUseBlock{{ID: "toolu_1", Name: "get_weather", Input: json.RawMessage(`{"city":"Paris"}`)}},
			},
			{Role: "user", ToolResults: []ToolResultBlock{{ToolUseID: "toolu_1", Content: "sunny"}}},
		},
	}

	body, err := json.Marshal(p.buildParams(req))
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	got := string(body)

	mustContain := []string{
		`"type":"adaptive"`,
		`"display":"summarized"`,
		`"signature":"sig_abc"`, // signed thinking block replayed
		`"id":"toolu_1"`,        // tool_use carried
		`tool_result`,           // tool_result carried
		`"model":"anthropic/claude-sonnet-4.6"`,
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("params missing %q\nbody: %s", want, got)
		}
	}
	if strings.Contains(got, "unsigned reasoning") {
		t.Errorf("unsigned thinking block must be dropped (the API rejects it)\nbody: %s", got)
	}
	if strings.Contains(got, `"temperature"`) {
		t.Errorf("temperature must be omitted on the native adaptive-thinking path\nbody: %s", got)
	}
}

// 4.0–4.5 models must use the legacy enabled+budget_tokens thinking form (adaptive is rejected there).
func TestBuildParams_LegacyThinkingForOlderModels(t *testing.T) {
	t.Parallel()
	p := NewAnthropicMessagesProvider("sk_test_fake")
	body, err := json.Marshal(p.buildParams(&ToolRequest{
		Model:           "claude-sonnet-4",
		MaxTokens:       4096,
		EnableReasoning: true,
		Messages:        []Message{{Role: "user", Content: "Hi"}},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, `"type":"enabled"`) || !strings.Contains(got, `"budget_tokens":2048`) {
		t.Errorf("expected legacy enabled+budget_tokens for claude-sonnet-4\nbody: %s", got)
	}
	if strings.Contains(got, "adaptive") {
		t.Errorf("4.0-gen model must not use adaptive thinking\nbody: %s", got)
	}
}

func TestSupportsAdaptiveThinking(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"claude-sonnet-4.6": true,
		"claude-opus-4.8":   true,
		"claude-opus-4.7":   true,
		"claude-sonnet-4":   false,
		"claude-opus-4.5":   false,
		"claude-haiku-4.5":  false,
	}
	for model, want := range cases {
		if got := supportsAdaptiveThinking(model); got != want {
			t.Errorf("supportsAdaptiveThinking(%q) = %v, want %v", model, got, want)
		}
	}
}

// Regression: a no-arg tool (schema with no "properties") must still emit input_schema. The SDK omits
// a Go-zero ToolInputSchemaParam, which previously produced "tools.N.custom.input_schema: Field required".
func TestBuildParams_ToolWithoutPropertiesStillEmitsInputSchema(t *testing.T) {
	t.Parallel()
	p := NewAnthropicMessagesProvider("sk_test_fake")
	cases := map[string]json.RawMessage{
		"type-only schema": json.RawMessage(`{"type":"object"}`),
		"empty object":     json.RawMessage(`{}`),
		"null/missing":     nil,
	}
	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(p.buildParams(&ToolRequest{
				Model:    "claude-sonnet-4.6",
				Messages: []Message{{Role: "user", Content: "hi"}},
				Tools:    []ToolDefinition{{Name: "list_open_orders", Description: "List open orders", InputSchema: schema}},
			}))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(body), `"input_schema"`) {
				t.Errorf("input_schema must be present even for a no-arg tool\nbody: %s", body)
			}
		})
	}
}

// The full JSON schema must survive translation — properties, required, and other keys like
// additionalProperties / $defs (carried via ExtraFields).
func TestBuildParams_PreservesFullToolSchema(t *testing.T) {
	t.Parallel()
	p := NewAnthropicMessagesProvider("sk_test_fake")
	body, err := json.Marshal(p.buildParams(&ToolRequest{
		Model:    "claude-sonnet-4.6",
		Messages: []Message{{Role: "user", Content: "hi"}},
		Tools: []ToolDefinition{{
			Name:        "create_customer",
			Description: "Create a customer",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`),
		}},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	for _, want := range []string{`"name"`, `"required":["name"]`, `"additionalProperties":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("schema lost %q\nbody: %s", want, got)
		}
	}
}

func TestMapAnthropicStopReason(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"tool_use":   "tool_use",
		"max_tokens": "max_tokens",
		"end_turn":   "end_turn",
		"refusal":    "end_turn",
		"":           "end_turn",
	}
	for in, want := range cases {
		if got := mapAnthropicStopReason(in); got != want {
			t.Errorf("mapAnthropicStopReason(%q) = %q, want %q", in, got, want)
		}
	}
}
