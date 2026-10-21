package repository

import (
	"strings"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

func TestBuildAgentRunListQuery_NoFilters(t *testing.T) {
	query, args := buildAgentRunListQuery(domain.AgentRunPageQuery{AccountID: "ac_1", Limit: 26})

	if strings.Contains(query, "LIKE") || strings.Contains(query, "cr.") || strings.Contains(query, "status_code =") {
		t.Fatalf("absent filters must emit no predicate: %s", query)
	}
	if want := []any{"ac_1", int32(26)}; !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestBuildAgentRunListQuery_AllFilters(t *testing.T) {
	query, args := buildAgentRunListQuery(domain.AgentRunPageQuery{
		AccountID: "ac_1", StatusCode: "failed", AgentDefinitionID: "agdf_1", Query: "50%", CursorID: "agrn_9", Limit: 11,
	})

	if !strings.HasPrefix(query, "SELECT ar.id,") || !strings.Contains(query, "JOIN agent_run cr ON cr.id = ?") {
		t.Fatalf("unexpected query: %s", query)
	}
	want := []any{"agrn_9", "ac_1", "failed", "agdf_1", `%50\%%`, `%50\%%`, `%50\%%`, int32(11)}
	if !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	if got := strings.Count(query, "?"); got != len(args) {
		t.Fatalf("query has %d placeholders for %d args", got, len(args))
	}
}

func TestBuildAgentDefinitionListQuery_Statuses(t *testing.T) {
	cases := []struct {
		name     string
		statuses []string
		want     string
		join     bool
	}{
		{"none", nil, "", false},
		{"active", []string{"active"}, "aas.status_code = 'active'", true},
		{"inactive", []string{"inactive"}, "(aas.status_code IS NULL OR aas.status_code != 'active')", true},
		{"both", []string{"active", "inactive"}, "", false},
		{"unknown", []string{"paused"}, "FALSE", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query, args := buildAgentDefinitionListQuery(domain.AgentDefinitionPageQuery{
				AccountID: "ac_1", Statuses: tc.statuses, DefinitionTypes: []string{"custom", "system"}, Limit: 5,
			})
			if tc.want != "" && !strings.Contains(query, tc.want) {
				t.Fatalf("query missing %q: %s", tc.want, query)
			}
			if strings.Contains(query, "agent_account_status") != tc.join {
				t.Fatalf("status join present = %v, want %v: %s", !tc.join, tc.join, query)
			}
			if got := strings.Count(query, "?"); got != len(args) {
				t.Fatalf("query has %d placeholders for %d args", got, len(args))
			}
		})
	}
}

func TestBuildAgentMemoryListQuery_Cursor(t *testing.T) {
	query, args := buildAgentMemoryListQuery(domain.AgentMemoryPageQuery{AccountID: "ac_1", Category: "pref", CursorID: "agmm_1", Limit: 3})

	if !strings.Contains(query, "am.created_at < cr.created_at") {
		t.Fatalf("cursor predicate missing: %s", query)
	}
	if want := []any{"agmm_1", "ac_1", "pref", int32(3)}; !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func equalArgs(got, want []any) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
