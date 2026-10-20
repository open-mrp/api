package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayModelName_OpenRouterSlugs(t *testing.T) {
	t.Parallel()

	for model, want := range map[string]string{
		"claude-sonnet-4.6": "anthropic/claude-sonnet-4.6",
		"gpt-5.5":           "openai/gpt-5.5",
		"gemini-2.5-pro":    "google/gemini-2.5-pro",
		"gemini-3-flash":    "google/gemini-3-flash-preview",
		"grok-4":            "x-ai/grok-4",
	} {
		if got := GatewayModelName(model); got != want {
			t.Errorf("GatewayModelName(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestCompleteWithTools_AuthenticatesWithTheOpenRouterKey(t *testing.T) {
	t.Parallel()

	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body gatewayRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel = body.Model
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	defer srv.Close()

	provider := &GatewayProvider{httpClient: srv.Client(), apiKey: "sk-or-test-fake", baseURL: srv.URL}
	if _, err := provider.CompleteWithTools(context.Background(), &ToolRequest{
		Model:    "gpt-5",
		Messages: []Message{{Role: "user", Content: "Hi."}},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer sk-or-test-fake" {
		t.Errorf("Authorization = %q, want the OpenRouter key as a bearer token", gotAuth)
	}
	if gotModel != "openai/gpt-5" {
		t.Errorf("model = %q, want %q", gotModel, "openai/gpt-5")
	}
}
