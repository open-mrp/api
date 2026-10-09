package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicMessagesProvider calls Anthropic's native Messages API through the Stripe AI Gateway passthrough (https://llm.stripe.com/v1/messages). Going native — rather than the OpenAI-compatible /chat/completions path used by GatewayProvider — gives us real thinking blocks: a distinct reasoning channel that streams token-by-token and carries signatures, so reasoning can be replayed across tool-loop turns. Stripe still meters usage per customer via the X-Stripe-Customer-ID header, so billing is unchanged.
type AnthropicMessagesProvider struct {
	stripeAPIKey string
	baseURL      string
}

func NewAnthropicMessagesProvider(stripeAPIKey string) *AnthropicMessagesProvider {
	return &AnthropicMessagesProvider{stripeAPIKey: stripeAPIKey, baseURL: defaultGatewayBaseURL}
}

func (p *AnthropicMessagesProvider) Name() string { return "stripe_gateway_anthropic" }

// client builds an SDK client pinned at the Stripe gateway. The gateway authenticates on the Authorization bearer header and ignores the SDK's x-api-key, so the api key is a placeholder.
func (p *AnthropicMessagesProvider) client(customerID string) anthropic.Client {
	return anthropic.NewClient(
		option.WithBaseURL(p.baseURL),
		option.WithAPIKey("unused"),
		option.WithHeader("Authorization", "Bearer "+p.stripeAPIKey),
		option.WithHeader("X-Stripe-Customer-ID", customerID),
	)
}

func (p *AnthropicMessagesProvider) CompleteWithTools(ctx context.Context, req *ToolRequest) (*ToolResponse, error) {
	customerID := StripeCustomerIDFromContext(ctx)
	if customerID == "" {
		return nil, fmt.Errorf("stripe customer ID not set in context")
	}
	client := p.client(customerID)
	msg, err := client.Messages.New(ctx, p.buildParams(req))
	if err != nil {
		return nil, mapAnthropicError(err)
	}
	return toolResponseFromMessage(msg), nil
}

func (p *AnthropicMessagesProvider) StreamCompleteWithTools(ctx context.Context, req *ToolRequest, callback func(StreamEvent)) (*ToolResponse, error) {
	customerID := StripeCustomerIDFromContext(ctx)
	if customerID == "" {
		return nil, fmt.Errorf("stripe customer ID not set in context")
	}
	client := p.client(customerID)

	stream := client.Messages.NewStreaming(ctx, p.buildParams(req))
	msg := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return nil, fmt.Errorf("accumulate stream event: %w", err)
		}
		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			// A tool_use block has begun: surface its name/id immediately (the arguments stream afterwards as InputJSONDelta, which carry neither). This lets the runner switch chat to the live "calling <tool>" phase before the full input is assembled.
			if ev.ContentBlock.Type == "tool_use" {
				callback(StreamEvent{Type: "tool_call_delta", ToolIndex: int(ev.Index), ToolCallID: ev.ContentBlock.ID, ToolName: ev.ContentBlock.Name})
			}
		case anthropic.ContentBlockDeltaEvent:
			switch d := ev.Delta.AsAny().(type) {
			case anthropic.ThinkingDelta:
				if d.Thinking != "" {
					callback(StreamEvent{Type: "reasoning_delta", ReasoningDelta: d.Thinking})
				}
			case anthropic.TextDelta:
				if d.Text != "" {
					callback(StreamEvent{Type: "content_delta", ContentDelta: d.Text})
				}
			case anthropic.InputJSONDelta:
				if d.PartialJSON != "" {
					callback(StreamEvent{Type: "tool_call_delta", ToolIndex: int(ev.Index), ArgumentsDelta: d.PartialJSON})
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, mapAnthropicError(err)
	}

	resp := toolResponseFromMessage(&msg)
	callback(StreamEvent{Type: "done", FinishReason: resp.StopReason, InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens})
	return resp, nil
}

// buildParams maps a ToolRequest to the native Messages API request. Adaptive thinking is enabled with summarized display so the reasoning stream carries readable text (the default "omitted" would stream empty thinking). Temperature is intentionally omitted: adaptive thinking models reject sampling parameters.
func (p *AnthropicMessagesProvider) buildParams(req *ToolRequest) anthropic.MessageNewParams {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(GatewayModelName(req.Model)),
		MaxTokens: int64(maxTokens),
		Messages:  convertMessagesToAnthropic(req.Messages),
	}
	if req.EnableReasoning {
		params.Thinking = thinkingConfigFor(req.Model, maxTokens)
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	if tools := convertToolsToAnthropic(req.Tools); len(tools) > 0 {
		params.Tools = tools
		if req.ToolChoice == ToolChoiceNone {
			params.ToolChoice = anthropic.ToolChoiceUnionParam{OfNone: &anthropic.ToolChoiceNoneParam{}}
		}
	}
	applyCacheBreakpoints(&params)
	return params
}

// applyCacheBreakpoints marks the tools, system prompt, and newest message as cache prefixes (3 of the API's 4).
// Each loop turn resends the previous prompt plus one exchange, so the message breakpoint serves it from cache;
// the tool and system breakpoints let runs of the same agent share the stable head.
func applyCacheBreakpoints(params *anthropic.MessageNewParams) {
	if n := len(params.Tools); n > 0 && params.Tools[n-1].OfTool != nil {
		params.Tools[n-1].OfTool.CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
	if n := len(params.System); n > 0 {
		params.System[n-1].CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
	if n := len(params.Messages); n > 0 {
		content := params.Messages[n-1].Content
		if m := len(content); m > 0 {
			if cc := content[m-1].GetCacheControl(); cc != nil {
				*cc = anthropic.NewCacheControlEphemeralParam()
			}
		}
	}
}

// thinkingConfigFor selects the thinking API by model generation. The Stripe gateway serves Claude via Vertex, where 4.6+ models take adaptive thinking (and reject budget_tokens) while 4.0–4.5 models take the legacy enabled+budget_tokens form. Both stream thinking blocks with signatures, so reasoning and cross-turn replay work either way.
func thinkingConfigFor(model string, maxTokens int) anthropic.ThinkingConfigParamUnion {
	if supportsAdaptiveThinking(model) {
		return anthropic.ThinkingConfigParamUnion{OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{
			Display: anthropic.ThinkingConfigAdaptiveDisplaySummarized,
		}}
	}
	budget := maxTokens / 2
	if budget < 1024 {
		budget = 1024
	}
	if budget >= maxTokens {
		budget = maxTokens - 1
	}
	return anthropic.ThinkingConfigParamOfEnabled(int64(budget))
}

// reasoningMaxOutputTokens leaves room for both reasoning and a long answer: thinking counts against max_tokens, so at the 4096 default a long reasoning pass truncates the reply mid-sentence.
const reasoningMaxOutputTokens = 16000

// MaxOutputTokens is the max_tokens for one turn. Only Claude 4+ with reasoning gets the larger cap; older fallbacks reject anything above their (4k–8k) output limit.
func MaxOutputTokens(model string, reasoning bool) int {
	if !reasoning || !strings.HasPrefix(model, "claude-") {
		return 4096
	}
	idx := strings.LastIndex(model, "-")
	if idx < 0 || idx+1 >= len(model) {
		return 4096
	}
	if major, _ := parseModelVersion(model[idx+1:]); major < 4 {
		return 4096
	}
	return reasoningMaxOutputTokens
}

// supportsAdaptiveThinking reports whether a model id (e.g. "claude-sonnet-4.6") is generation 4.6 or newer, where adaptive thinking is the only accepted thinking mode.
func supportsAdaptiveThinking(model string) bool {
	idx := strings.LastIndex(model, "-")
	if idx < 0 || idx+1 >= len(model) {
		return false
	}
	major, minor := parseModelVersion(model[idx+1:])
	return major > 4 || (major == 4 && minor >= 6)
}

func parseModelVersion(v string) (major, minor int) {
	parts := strings.SplitN(v, ".", 2)
	major, _ = strconv.Atoi(parts[0])
	if len(parts) == 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}

func convertToolsToAnthropic(tools []ToolDefinition) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		out = append(out, anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropicInputSchema(t.InputSchema),
		}})
	}
	return out
}

// anthropicInputSchema converts a raw JSON Schema into the SDK's input_schema param. The Anthropic API requires input_schema on every tool, but the SDK omits a Go-zero ToolInputSchemaParam (json:"...,omitzero") — which happens for a no-arg tool whose schema has no "properties". So we always set Properties (defaulting to an empty object) to keep the struct non-zero, and preserve every other top-level schema key (required, additionalProperties, $defs, …) so nothing is lost vs the raw schema.
func anthropicInputSchema(raw json.RawMessage) anthropic.ToolInputSchemaParam {
	schema := anthropic.ToolInputSchemaParam{Properties: map[string]any{}}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed == nil {
		return schema
	}
	if props, ok := parsed["properties"]; ok && props != nil {
		schema.Properties = props
	}
	if reqAny, ok := parsed["required"].([]any); ok {
		req := make([]string, 0, len(reqAny))
		for _, r := range reqAny {
			if s, ok := r.(string); ok {
				req = append(req, s)
			}
		}
		schema.Required = req
	}
	for k, v := range parsed {
		switch k {
		case "properties", "required", "type":
			// handled above / fixed to "object"
		default:
			if schema.ExtraFields == nil {
				schema.ExtraFields = map[string]any{}
			}
			schema.ExtraFields[k] = v
		}
	}
	return schema
}

// convertMessagesToAnthropic maps our transcript onto alternating Messages API turns. Consecutive same-role
// messages (an injected reminder after tool results, a truncation placeholder) merge into one turn, and the
// blocks the API requires first — tool_result in a user turn, thinking in an assistant turn — lead it.
func convertMessagesToAnthropic(messages []Message) []anthropic.MessageParam {
	type turn struct {
		role    string
		leading []anthropic.ContentBlockParamUnion
		rest    []anthropic.ContentBlockParamUnion
	}
	var turns []*turn
	for _, m := range messages {
		var leading, rest []anthropic.ContentBlockParamUnion
		switch m.Role {
		case "user":
			for _, tr := range m.ToolResults {
				leading = append(leading, anthropic.NewToolResultBlock(tr.ToolUseID, tr.Content, tr.IsError))
			}
			if m.Content != "" {
				rest = append(rest, anthropic.NewTextBlock(m.Content))
			}
		case "assistant":
			// The API rejects unsigned thinking blocks (e.g. from the OpenAI-compat path), so only signed ones replay.
			for _, tb := range m.Thinking {
				if tb.Signature != "" {
					leading = append(leading, anthropic.NewThinkingBlock(tb.Signature, tb.Text))
				}
			}
			if m.Content != "" {
				rest = append(rest, anthropic.NewTextBlock(m.Content))
			}
			for _, tu := range m.ToolUse {
				rest = append(rest, anthropic.ContentBlockParamUnion{OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    tu.ID,
					Name:  tu.Name,
					Input: json.RawMessage(normalizeToolInput(tu.Input)),
				}})
			}
		default:
			continue
		}
		if len(leading) == 0 && len(rest) == 0 {
			continue
		}
		if n := len(turns); n > 0 && turns[n-1].role == m.Role {
			turns[n-1].leading = append(turns[n-1].leading, leading...)
			turns[n-1].rest = append(turns[n-1].rest, rest...)
			continue
		}
		turns = append(turns, &turn{role: m.Role, leading: leading, rest: rest})
	}

	out := make([]anthropic.MessageParam, 0, len(turns))
	for _, t := range turns {
		blocks := append(t.leading, t.rest...)
		if t.role == "user" {
			out = append(out, anthropic.NewUserMessage(blocks...))
		} else {
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		}
	}
	return out
}

// toolResponseFromMessage normalizes a completed native Message into the provider-agnostic ToolResponse, splitting native thinking blocks (reasoning, with signatures for replay) from the user-facing text and tool calls.
func toolResponseFromMessage(msg *anthropic.Message) *ToolResponse {
	cacheRead := int(msg.Usage.CacheReadInputTokens)
	cacheWrite := int(msg.Usage.CacheCreationInputTokens)
	resp := &ToolResponse{
		InputTokens:              int(msg.Usage.InputTokens) + cacheRead + cacheWrite,
		OutputTokens:             int(msg.Usage.OutputTokens),
		CacheReadInputTokens:     cacheRead,
		CacheCreationInputTokens: cacheWrite,
		StopReason:               mapAnthropicStopReason(string(msg.StopReason)),
	}
	var content strings.Builder
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			content.WriteString(b.Text)
		case anthropic.ThinkingBlock:
			resp.Thinking = append(resp.Thinking, ThinkingBlock{Text: b.Thinking, Signature: b.Signature})
		case anthropic.ToolUseBlock:
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:    b.ID,
				Name:  b.Name,
				Input: normalizeToolInput(json.RawMessage(b.Input)),
			})
		}
	}
	resp.Content = content.String()
	return resp
}

func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "tool_use":
		return "tool_use"
	case "max_tokens":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

// mapAnthropicError converts an SDK API error into a GatewayError so the runner's existing retry/failover logic (which inspects *GatewayError) treats native-path failures uniformly.
func mapAnthropicError(err error) error {
	var apierr *anthropic.Error
	if errors.As(err, &apierr) {
		return NewGatewayError(apierr.StatusCode, apierr.RawJSON(), nil)
	}
	return err
}
