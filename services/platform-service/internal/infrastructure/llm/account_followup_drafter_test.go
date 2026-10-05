package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/platform-service/internal/domain"

	"github.com/stretchr/testify/require"
)

func gatewayResponse(t *testing.T, toolInput string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-test",
		"stop_reason": "tool_use",
		"content":     []map[string]any{{"type": "tool_use", "id": "toolu_1", "name": draftToolName, "input": json.RawMessage(toolInput)}},
		"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
	})
	require.NoError(t, err)
	return string(body)
}

func TestDraftCallsGatewayWithoutCustomer(t *testing.T) {
	t.Parallel()

	var got struct {
		auth, customer string
		body           map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.auth = r.Header.Get("Authorization")
		got.customer = r.Header.Get("X-Stripe-Customer-ID")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got.body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, gatewayResponse(t, `{"engagement":"medium","internal_summary":"Set up locations.","subject":"your openmrp setup","paragraph":"Thanks for trying OpenMRP."}`))
	}))
	defer server.Close()

	drafter, err := NewAccountFollowupDrafter(&AccountFollowupDrafterConfig{StripeSecretKey: "sk_test_x", Model: "claude-test-model", BaseURL: server.URL})
	require.NoError(t, err)

	out, err := drafter.Draft(context.Background(), domain.AccountFollowupDrafterInput{FirstName: "Sam", AccountName: "Example Co"})
	require.NoError(t, err)

	require.Equal(t, "Bearer sk_test_x", got.auth)
	require.Empty(t, got.customer, "no customer header: the platform pays for its own drafting")
	require.Equal(t, "anthropic/claude-test-model", got.body["model"])
	require.Nil(t, got.body["output_config"], "the gateway's Vertex project disallows structured outputs")
	require.Equal(t, map[string]any{"type": "tool", "name": draftToolName}, got.body["tool_choice"])
	require.Equal(t, draftToolName, got.body["tools"].([]any)[0].(map[string]any)["name"])
	require.NotContains(t, got.body["tools"].([]any)[0].(map[string]any), "strict", "strict tool use counts as structured outputs on Vertex")
	require.Contains(t, got.body["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"], "First name: Sam")

	require.Equal(t, &domain.AccountFollowupDrafterOutput{
		Model:           "claude-test-model",
		PromptVersion:   accountFollowupPromptVersion,
		Engagement:      domain.AccountFollowupEngagementMedium,
		InternalSummary: "Set up locations.",
		Subject:         "your openmrp setup",
		Paragraph:       "Thanks for trying OpenMRP.",
	}, out)
}

func TestParseDraftRejectsWhatTheSchemaCannot(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"not json":           `here you go`,
		"unknown engagement": `{"engagement":"huge","internal_summary":"x","subject":"x","paragraph":"x"}`,
		"empty paragraph":    `{"engagement":"low","internal_summary":"x","subject":"x","paragraph":"  "}`,
		"long paragraph":     `{"engagement":"low","internal_summary":"x","subject":"x","paragraph":"` + strings.Repeat("a", maxParagraphChars+1) + `"}`,
	}
	for name, text := range cases {
		_, err := parseDraft(text, "m")
		require.Error(t, err, name)
	}
}
