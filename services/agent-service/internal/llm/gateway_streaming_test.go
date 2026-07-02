package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStreamCompleteWithTools_ContentOnly(t *testing.T) {
	t.Parallel(
	// Mock SSE server that returns content-only deltas.
	)

	sseBody := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{"content":" world"},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{"content":"!"},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":15,"completion_tokens":3}}`,
		"",
		`data: [DONE]`,
		"",
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	provider := &GatewayProvider{
		httpClient:   srv.Client(),
		stripeAPIKey: "sk_test_fake",
		baseURL:      srv.URL,
	}

	ctx := WithStripeCustomerID(context.Background(), "cus_test123")

	var contentDeltas []string
	var doneEvent *StreamEvent

	resp, err := provider.StreamCompleteWithTools(ctx, &ToolRequest{
		Model:    "claude-sonnet-4",
		System:   "You are helpful.",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}, func(ev StreamEvent) {
		switch ev.Type {
		case "content_delta":
			contentDeltas = append(contentDeltas, ev.ContentDelta)
		case "done":
			doneEvent = &ev
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify content deltas
	if len(contentDeltas) != 3 {
		t.Fatalf("expected 3 content deltas, got %d", len(contentDeltas))
	}
	if contentDeltas[0] != "Hello" || contentDeltas[1] != " world" || contentDeltas[2] != "!" {
		t.Errorf("unexpected content deltas: %v", contentDeltas)
	}

	// Verify done event
	if doneEvent == nil {
		t.Fatal("expected done event")
	}
	if doneEvent.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", doneEvent.FinishReason)
	}
	if doneEvent.InputTokens != 15 {
		t.Errorf("expected input_tokens 15, got %d", doneEvent.InputTokens)
	}
	if doneEvent.OutputTokens != 3 {
		t.Errorf("expected output_tokens 3, got %d", doneEvent.OutputTokens)
	}

	// Verify returned ToolResponse
	if resp.Content != "Hello world!" {
		t.Errorf("expected accumulated content 'Hello world!', got %q", resp.Content)
	}
	if resp.InputTokens != 15 {
		t.Errorf("expected InputTokens 15, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 3 {
		t.Errorf("expected OutputTokens 3, got %d", resp.OutputTokens)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("expected StopReason 'end_turn', got %q", resp.StopReason)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("expected no tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestStreamCompleteWithTools_ToolCalls(t *testing.T) {
	t.Parallel(
	// Mock SSE server that returns tool call deltas spread across chunks.
	)

	sseBody := strings.Join([]string{
		// First chunk: tool call ID and function name
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc123","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
		"",
		// Second chunk: partial arguments
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"finish_reason":null}]}`,
		"",
		// Third chunk: rest of arguments
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":\"NYC\"}"}}]},"finish_reason":null}]}`,
		"",
		// Final chunk: finish_reason and usage
		`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":8}}`,
		"",
		`data: [DONE]`,
		"",
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	provider := &GatewayProvider{
		httpClient:   srv.Client(),
		stripeAPIKey: "sk_test_fake",
		baseURL:      srv.URL,
	}

	ctx := WithStripeCustomerID(context.Background(), "cus_test456")

	var toolCallDeltas []StreamEvent

	resp, err := provider.StreamCompleteWithTools(ctx, &ToolRequest{
		Model:    "claude-sonnet-4",
		System:   "You are helpful.",
		Messages: []Message{{Role: "user", Content: "What is the weather?"}},
	}, func(ev StreamEvent) {
		if ev.Type == "tool_call_delta" {
			toolCallDeltas = append(toolCallDeltas, ev)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool_call_delta callbacks were received
	if len(toolCallDeltas) != 3 {
		t.Fatalf("expected 3 tool_call_delta events, got %d", len(toolCallDeltas))
	}
	if toolCallDeltas[0].ToolCallID != "call_abc123" {
		t.Errorf("expected first delta ToolCallID 'call_abc123', got %q", toolCallDeltas[0].ToolCallID)
	}
	if toolCallDeltas[0].ToolName != "get_weather" {
		t.Errorf("expected first delta ToolName 'get_weather', got %q", toolCallDeltas[0].ToolName)
	}

	// Verify returned ToolResponse
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_abc123" {
		t.Errorf("expected tool call ID 'call_abc123', got %q", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("expected tool call name 'get_weather', got %q", tc.Name)
	}
	expectedArgs := `{"location":"NYC"}`
	if string(tc.Input) != expectedArgs {
		t.Errorf("expected tool call arguments %q, got %q", expectedArgs, string(tc.Input))
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("expected StopReason 'tool_use', got %q", resp.StopReason)
	}
	if resp.InputTokens != 20 || resp.OutputTokens != 8 {
		t.Errorf("expected usage (20, 8), got (%d, %d)", resp.InputTokens, resp.OutputTokens)
	}
}

func TestStreamCompleteWithTools_UsageFromFinalChunk(t *testing.T) {
	t.Parallel(
	// Mock SSE server where usage only appears in the final chunk
	// (separate from the finish_reason chunk), as with stream_options.include_usage.
	)

	sseBody := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"OK"},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		"",
		// Usage-only chunk (no choices) as sent by some providers with include_usage
		`data: {"choices":[],"usage":{"prompt_tokens":42,"completion_tokens":17}}`,
		"",
		`data: [DONE]`,
		"",
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	provider := &GatewayProvider{
		httpClient:   srv.Client(),
		stripeAPIKey: "sk_test_fake",
		baseURL:      srv.URL,
	}

	ctx := WithStripeCustomerID(context.Background(), "cus_test789")

	var doneEvent *StreamEvent

	resp, err := provider.StreamCompleteWithTools(ctx, &ToolRequest{
		Model:    "claude-sonnet-4",
		System:   "You are helpful.",
		Messages: []Message{{Role: "user", Content: "Say OK"}},
	}, func(ev StreamEvent) {
		if ev.Type == "done" {
			doneEvent = &ev
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify usage is correctly extracted from the final usage-only chunk
	if resp.InputTokens != 42 {
		t.Errorf("expected InputTokens 42, got %d", resp.InputTokens)
	}
	if resp.OutputTokens != 17 {
		t.Errorf("expected OutputTokens 17, got %d", resp.OutputTokens)
	}
	if resp.Content != "OK" {
		t.Errorf("expected content 'OK', got %q", resp.Content)
	}

	// Verify done event carries the usage
	if doneEvent == nil {
		t.Fatal("expected done event")
	}
	if doneEvent.InputTokens != 42 {
		t.Errorf("expected done InputTokens 42, got %d", doneEvent.InputTokens)
	}
	if doneEvent.OutputTokens != 17 {
		t.Errorf("expected done OutputTokens 17, got %d", doneEvent.OutputTokens)
	}
}

// TestStreamCompleteWithTools_IdleStallAborts verifies the idle watchdog aborts a
// stream that goes silent mid-flight (as when an egress NAT/firewall silently drops
// a long-lived connection) instead of blocking forever, and surfaces it as a
// retryable gateway error so the run fails over / retries rather than hanging in "running".
func TestStreamCompleteWithTools_IdleStallAborts(t *testing.T) {
	t.Parallel()

	// Sends one delta, flushes, then holds the connection open with no further bytes
	// and never sends [DONE] — a stalled/dead stream.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"Hi"},"finish_reason":null}]}`+"\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done() // release the handler once the client aborts the read
	}))
	defer srv.Close()

	provider := &GatewayProvider{
		httpClient:   srv.Client(),
		stripeAPIKey: "sk_test_fake",
		baseURL:      srv.URL,
		idleTimeout:  50 * time.Millisecond,
	}

	ctx := WithStripeCustomerID(context.Background(), "cus_stall")

	start := time.Now()
	_, err := provider.StreamCompleteWithTools(ctx, &ToolRequest{
		Model:    "claude-sonnet-4",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	}, func(ev StreamEvent) {})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a stalled stream, got nil")
	}
	var ge *GatewayError
	if !errors.As(err, &ge) {
		t.Fatalf("expected *GatewayError, got %T: %v", err, err)
	}
	if !ge.Retryable {
		t.Errorf("expected stall error to be retryable, got %+v", ge)
	}
	if ge.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", ge.StatusCode)
	}
	// Should abort shortly after idleTimeout, not hang.
	if elapsed > 2*time.Second {
		t.Errorf("watchdog took too long to abort: %s", elapsed)
	}
}
