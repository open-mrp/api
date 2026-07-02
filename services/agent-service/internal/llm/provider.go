package llm

import (
	"context"
	"encoding/json"
)

// Context helpers for Stripe customer ID propagation.
type ctxKey string

const stripeCustomerIDKey ctxKey = "stripe_customer_id"

func WithStripeCustomerID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, stripeCustomerIDKey, id)
}

func StripeCustomerIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(stripeCustomerIDKey).(string); ok {
		return v
	}
	return ""
}

// LLMProvider abstracts multi-turn LLM interactions with tool use.
type LLMProvider interface {
	CompleteWithTools(ctx context.Context, req *ToolRequest) (*ToolResponse, error)
	Name() string
}

// Message represents a single conversation turn.
type Message struct {
	Role        string            `json:"role"`
	Content     string            `json:"content,omitempty"`
	Thinking    []ThinkingBlock   `json:"thinking,omitempty"`
	ToolUse     []ToolUseBlock    `json:"tool_use,omitempty"`
	ToolResults []ToolResultBlock `json:"tool_results,omitempty"`
}

// ThinkingBlock carries a provider reasoning block together with its signature so it can be replayed to the model on later tool-loop turns. Anthropic requires preserved, unmodified thinking blocks when continuing a turn after tool_use under interleaved thinking; providers that don't return signatures (the OpenAI-compatible path) leave Signature empty, and such blocks are not replayed.
type ThinkingBlock struct {
	Text      string `json:"text"`
	Signature string `json:"signature,omitempty"`
}

// ToolUseBlock represents an assistant's request to use a tool.
type ToolUseBlock struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ToolDefinition describes a tool available to the LLM.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolRequest is the input to a provider's CompleteWithTools call.
type ToolRequest struct {
	Model       string
	System      string
	Messages    []Message
	Tools       []ToolDefinition
	MaxTokens   int
	Temperature float64
	// EnableReasoning turns on the provider's reasoning output — adaptive thinking on the native Anthropic path. Chat runs set it so the live thinking panel has content; other runs leave it off to avoid the extra thinking-token cost.
	EnableReasoning bool
	// ReasoningEffort, when set ("low"/"medium"/"high"), asks reasoning-capable models on the OpenAI-compatible path to emit reasoning. Ignored by the native Anthropic provider.
	ReasoningEffort string
}

// ToolResponse is the normalized output from a provider.
type ToolResponse struct {
	Content      string
	Thinking     []ThinkingBlock // native reasoning blocks (Anthropic); empty for the OpenAI-compat path
	ToolCalls    []ToolCall
	InputTokens  int
	OutputTokens int
	StopReason   string // "end_turn", "tool_use", "max_tokens"
}

// ToolCall represents a single tool invocation from the LLM.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// StreamEvent represents a single chunk from a streaming LLM response.
// "reasoning_delta" carries native provider reasoning (Anthropic thinking blocks, or the OpenAI-compatible reasoning_content field) in ReasoningDelta — a channel distinct from "content_delta", which now carries only the user-facing answer text.
// "tool_call_delta" is emitted both at a tool block's start — carrying ToolName/ToolCallID so a live indicator can name the tool — and for each subsequent argument chunk in ArgumentsDelta (where
// ToolName is empty, as the arguments stream after the name on both providers).
type StreamEvent struct {
	Type           string // "reasoning_delta", "content_delta", "tool_call_delta", "done"
	ContentDelta   string
	ReasoningDelta string
	ToolIndex      int
	ToolCallID     string
	ToolName       string
	ArgumentsDelta string
	FinishReason   string
	InputTokens    int
	OutputTokens   int
}

// StreamingLLMProvider extends LLMProvider with streaming support.
type StreamingLLMProvider interface {
	LLMProvider
	StreamCompleteWithTools(ctx context.Context, req *ToolRequest, callback func(StreamEvent)) (*ToolResponse, error)
}
