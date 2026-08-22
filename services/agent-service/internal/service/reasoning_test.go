package service

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
)

// isChatRun gates resource-link linking + native reasoning streaming: true only when the run carries
// a non-empty conversation id.
func TestIsChatRun(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		run  *sqlc.AgentRun
		want bool
	}{
		{"has conversation", &sqlc.AgentRun{ConversationID: agentdb.PgText("cv_1")}, true},
		{"no conversation (null)", &sqlc.AgentRun{}, false},
		{"empty conversation id (valid but blank)", &sqlc.AgentRun{ConversationID: pgtype.Text{String: "", Valid: true}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isChatRun(tc.run); got != tc.want {
				t.Errorf("isChatRun = %v, want %v", got, tc.want)
			}
		})
	}
}

// concatThinking joins reasoning blocks into a single trimmed string for the persisted thinking step.
func TestConcatThinking(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		blocks []llm.ThinkingBlock
		want   string
	}{
		{"nil", nil, ""},
		{"single", []llm.ThinkingBlock{{Text: "  hello  "}}, "hello"},
		{"multiple joined", []llm.ThinkingBlock{{Text: "a"}, {Text: "b"}, {Text: "c"}}, "abc"},
		{"signatures ignored in text", []llm.ThinkingBlock{{Text: "x", Signature: "sig"}}, "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := concatThinking(tc.blocks); got != tc.want {
				t.Errorf("concatThinking = %q, want %q", got, tc.want)
			}
		})
	}
}
