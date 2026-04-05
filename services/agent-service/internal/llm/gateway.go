package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const defaultGatewayBaseURL = "https://llm.stripe.com"

// GatewayProvider routes all LLM calls through the Stripe AI Gateway,
// which uses the OpenAI-compatible /chat/completions endpoint for all providers.
type GatewayProvider struct {
	httpClient   *http.Client
	stripeAPIKey string
	baseURL      string
}

func NewGatewayProvider(stripeAPIKey string) *GatewayProvider {
	return &GatewayProvider{
		httpClient:   &http.Client{},
		stripeAPIKey: stripeAPIKey,
		baseURL:      defaultGatewayBaseURL,
	}
}

func (p *GatewayProvider) Name() string { return "stripe_gateway" }

func (p *GatewayProvider) CompleteWithTools(ctx context.Context, req *ToolRequest) (*ToolResponse, error) {
	customerID := StripeCustomerIDFromContext(ctx)
	if customerID == "" {
		return nil, fmt.Errorf("stripe customer ID not set in context")
	}

	gatewayModel := GatewayModelName(req.Model)

	// Build request body
	body := gatewayRequest{
		Model:    gatewayModel,
		Messages: convertMessagesToGateway(req.System, req.Messages),
		Tools:    convertToolsToGateway(req.Tools),
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	body.MaxTokens = &maxTokens
	if req.Temperature > 0 {
		body.Temperature = &req.Temperature
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gateway request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.stripeAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Stripe-Customer-ID", customerID)

	httpResp, err := p.httpClient.Do(httpReq) // #nosec G704 -- URL from server-configured LLM gateway endpoint
	if err != nil {
		return nil, fmt.Errorf("gateway request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gateway response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, NewGatewayError(httpResp.StatusCode, string(respBody), httpResp.Header)
	}

	var completion gatewayResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway response: %w", err)
	}

	return convertGatewayResponse(&completion), nil
}

// GatewayModelName prefixes a model name with its provider for the Stripe AI Gateway.
// Model names already match Stripe's naming convention (e.g. "claude-sonnet-4", "gpt-4o").
func GatewayModelName(model string) string {
	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic/" + model
	case strings.HasPrefix(model, "gpt-"), strings.HasPrefix(model, "o1"), strings.HasPrefix(model, "o3"):
		return "openai/" + model
	default:
		return "anthropic/" + model
	}
}

// --- Request types ---

type gatewayRequest struct {
	Model         string                `json:"model"`
	Messages      []gatewayMessage      `json:"messages"`
	Tools         []gatewayTool         `json:"tools,omitempty"`
	MaxTokens     *int                  `json:"max_tokens,omitempty"`
	Temperature   *float64              `json:"temperature,omitempty"`
	Stream        bool                  `json:"stream,omitempty"`
	StreamOptions *gatewayStreamOptions `json:"stream_options,omitempty"`
}

type gatewayMessage struct {
	Role       string            `json:"role"`
	Content    any               `json:"content,omitempty"`
	ToolCalls  []gatewayToolCall `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

type gatewayToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function gatewayFunctionCall `json:"function"`
}

type gatewayFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type gatewayTool struct {
	Type     string          `json:"type"`
	Function gatewayFunction `json:"function"`
}

type gatewayFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// --- Response types ---

type gatewayResponse struct {
	Choices []gatewayChoice `json:"choices"`
	Usage   gatewayUsage    `json:"usage"`
}

type gatewayChoice struct {
	Message      gatewayChoiceMessage `json:"message"`
	FinishReason string               `json:"finish_reason"`
}

type gatewayChoiceMessage struct {
	Content   string            `json:"content"`
	ToolCalls []gatewayToolCall `json:"tool_calls"`
}

type gatewayUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// --- Streaming types ---

type gatewayStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type gatewayStreamChunk struct {
	Choices []gatewayStreamChoice `json:"choices"`
	Usage   *gatewayUsage         `json:"usage,omitempty"`
}

type gatewayStreamChoice struct {
	Delta        gatewayStreamDelta `json:"delta"`
	FinishReason *string            `json:"finish_reason"`
}

type gatewayStreamDelta struct {
	Content   string                 `json:"content,omitempty"`
	ToolCalls []gatewayToolCallDelta `json:"tool_calls,omitempty"`
}

type gatewayToolCallDelta struct {
	Index    int                      `json:"index"`
	ID       string                   `json:"id,omitempty"`
	Type     string                   `json:"type,omitempty"`
	Function gatewayFunctionCallDelta `json:"function,omitempty"`
}

type gatewayFunctionCallDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// --- Streaming method ---

func (p *GatewayProvider) StreamCompleteWithTools(ctx context.Context, req *ToolRequest, callback func(StreamEvent)) (*ToolResponse, error) {
	customerID := StripeCustomerIDFromContext(ctx)
	if customerID == "" {
		return nil, fmt.Errorf("stripe customer ID not set in context")
	}

	gatewayModel := GatewayModelName(req.Model)

	body := gatewayRequest{
		Model:         gatewayModel,
		Messages:      convertMessagesToGateway(req.System, req.Messages),
		Tools:         convertToolsToGateway(req.Tools),
		Stream:        true,
		StreamOptions: &gatewayStreamOptions{IncludeUsage: true},
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	body.MaxTokens = &maxTokens
	if req.Temperature > 0 {
		body.Temperature = &req.Temperature
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gateway request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.stripeAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Stripe-Customer-ID", customerID)

	httpResp, err := p.httpClient.Do(httpReq) // #nosec G704 -- URL from server-configured LLM gateway endpoint
	if err != nil {
		return nil, fmt.Errorf("gateway request failed: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		return nil, NewGatewayError(httpResp.StatusCode, string(respBody), httpResp.Header)
	}

	var contentBuilder strings.Builder
	toolCallMap := make(map[int]*ToolCall)
	var finishReason string
	var inputTokens, outputTokens int

	scanner := bufio.NewScanner(httpResp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk gatewayStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			if delta.Content != "" {
				callback(StreamEvent{Type: "content_delta", ContentDelta: delta.Content})
				contentBuilder.WriteString(delta.Content)
			}

			for _, tcd := range delta.ToolCalls {
				callback(StreamEvent{
					Type:           "tool_call_delta",
					ToolIndex:      tcd.Index,
					ToolCallID:     tcd.ID,
					ToolName:       tcd.Function.Name,
					ArgumentsDelta: tcd.Function.Arguments,
				})

				if tcd.ID != "" {
					toolCallMap[tcd.Index] = &ToolCall{
						ID:   tcd.ID,
						Name: tcd.Function.Name,
					}
				}
				if tc, ok := toolCallMap[tcd.Index]; ok {
					tc.Input = append(tc.Input, []byte(tcd.Function.Arguments)...)
					if tcd.Function.Name != "" && tc.Name == "" {
						tc.Name = tcd.Function.Name
					}
				}
			}

			if choice.FinishReason != nil {
				finishReason = *choice.FinishReason
			}
		}

		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}
	}
	_ = httpResp.Body.Close()

	callback(StreamEvent{
		Type:         "done",
		FinishReason: finishReason,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	})

	// Assemble tool calls sorted by index.
	var indices []int
	for idx := range toolCallMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	var toolCalls []ToolCall
	for _, idx := range indices {
		tc := toolCallMap[idx]
		toolCalls = append(toolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Name,
			Input: json.RawMessage(tc.Input),
		})
	}

	stopReason := "end_turn"
	switch finishReason {
	case "tool_calls":
		stopReason = "tool_use"
	case "length":
		stopReason = "max_tokens"
	}

	return &ToolResponse{
		Content:      contentBuilder.String(),
		ToolCalls:    toolCalls,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		StopReason:   stopReason,
	}, nil
}

// --- Conversion helpers ---

func convertMessagesToGateway(systemPrompt string, messages []Message) []gatewayMessage {
	var out []gatewayMessage

	if systemPrompt != "" {
		out = append(out, gatewayMessage{Role: "system", Content: systemPrompt})
	}

	for _, m := range messages {
		out = append(out, convertSingleMessageToGateway(m)...)
	}

	return out
}

func convertSingleMessageToGateway(m Message) []gatewayMessage {
	var msgs []gatewayMessage

	switch m.Role {
	case "user":
		if m.Content != "" {
			msgs = append(msgs, gatewayMessage{Role: "user", Content: m.Content})
		}
		for _, tr := range m.ToolResults {
			msgs = append(msgs, gatewayMessage{
				Role:       "tool",
				Content:    tr.Content,
				ToolCallID: tr.ToolUseID,
			})
		}
	case "assistant":
		if m.Content != "" && len(m.ToolUse) == 0 {
			msgs = append(msgs, gatewayMessage{Role: "assistant", Content: m.Content})
		} else if len(m.ToolUse) > 0 {
			toolCalls := make([]gatewayToolCall, len(m.ToolUse))
			for i, tu := range m.ToolUse {
				toolCalls[i] = gatewayToolCall{
					ID:   tu.ID,
					Type: "function",
					Function: gatewayFunctionCall{
						Name:      tu.Name,
						Arguments: string(tu.Input),
					},
				}
			}
			msg := gatewayMessage{
				Role:      "assistant",
				ToolCalls: toolCalls,
			}
			if m.Content != "" {
				msg.Content = m.Content
			}
			msgs = append(msgs, msg)
		}
	case "system":
		msgs = append(msgs, gatewayMessage{Role: "system", Content: m.Content})
	}

	return msgs
}

func convertToolsToGateway(tools []ToolDefinition) []gatewayTool {
	out := make([]gatewayTool, len(tools))
	for i, t := range tools {
		var params map[string]any
		_ = json.Unmarshal(t.InputSchema, &params)
		out[i] = gatewayTool{
			Type: "function",
			Function: gatewayFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return out
}

func convertGatewayResponse(resp *gatewayResponse) *ToolResponse {
	result := &ToolResponse{
		StopReason:   "end_turn",
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}

	if len(resp.Choices) == 0 {
		return result
	}

	choice := resp.Choices[0]
	result.Content = choice.Message.Content

	switch choice.FinishReason {
	case "tool_calls":
		result.StopReason = "tool_use"
	case "length":
		result.StopReason = "max_tokens"
	}

	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	return result
}
