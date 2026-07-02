package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The OpenAI-compatible surface carries native model reasoning in a `reasoning_content` (some
// providers) or `reasoning` (others) delta field. Both must surface as reasoning_delta events,
// kept separate from the answer's content_delta stream.
func TestStreamCompleteWithTools_ReasoningDeltas(t *testing.T) {
	t.Parallel()

	sseBody := strings.Join([]string{
		`data: {"choices":[{"delta":{"reasoning_content":"Let me "},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{"reasoning":"think."},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{"content":"42"},"finish_reason":null}]}`,
		"",
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
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

	provider := &GatewayProvider{httpClient: srv.Client(), stripeAPIKey: "sk_test_fake", baseURL: srv.URL}
	ctx := WithStripeCustomerID(context.Background(), "cus_test_reason")

	var reasoning, content strings.Builder
	resp, err := provider.StreamCompleteWithTools(ctx, &ToolRequest{
		Model:           "gpt-5",
		Messages:        []Message{{Role: "user", Content: "Answer."}},
		ReasoningEffort: "medium",
	}, func(ev StreamEvent) {
		switch ev.Type {
		case "reasoning_delta":
			reasoning.WriteString(ev.ReasoningDelta)
		case "content_delta":
			content.WriteString(ev.ContentDelta)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reasoning.String() != "Let me think." {
		t.Errorf("reasoning = %q, want %q", reasoning.String(), "Let me think.")
	}
	if content.String() != "42" {
		t.Errorf("content = %q, want %q", content.String(), "42")
	}
	// The OpenAI-compat path carries no signed thinking blocks, so nothing is replayable.
	if len(resp.Thinking) != 0 {
		t.Errorf("expected no Thinking blocks on the OpenAI-compat path, got %d", len(resp.Thinking))
	}
	if resp.Content != "42" {
		t.Errorf("resp.Content = %q, want %q", resp.Content, "42")
	}
}

// reasoning_effort must reach the gateway request body only when set, and never on the native path.
func TestGatewayRequest_ReasoningEffort(t *testing.T) {
	t.Parallel()

	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()

	provider := &GatewayProvider{httpClient: srv.Client(), stripeAPIKey: "sk_test_fake", baseURL: srv.URL}
	ctx := WithStripeCustomerID(context.Background(), "cus_test_effort")

	_, err := provider.StreamCompleteWithTools(ctx, &ToolRequest{
		Model:           "gpt-5",
		Messages:        []Message{{Role: "user", Content: "Hi"}},
		ReasoningEffort: "high",
	}, func(StreamEvent) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotBody, `"reasoning_effort":"high"`) {
		t.Errorf("request body missing reasoning_effort: %s", gotBody)
	}
}
