package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/pagination"
)

func strPtr(s string) *string { return &s }

// emptyFilter represents the no-filter case — only target_account_id, cursor,
// ORDER BY, and LIMIT should appear in the generated SQL.
func emptyFilter() *domain.ListRequestLogsFilter {
	return &domain.ListRequestLogsFilter{}
}

func TestBuildListQuery_NoFiltersEmitsOnlyBaselinePredicates(t *testing.T) {
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	mustContain(t, sql, "WHERE rl.target_account_id = ?")
	mustContain(t, sql, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")

	forbidden := []string{
		"rl.identity_type =",
		"rl.method LIKE",
		"rl.status_code =",
		"rl.error_code LIKE",
		"rl.account_id =",
		"rl.actor_id IN",
		"rl.normalized_route IN",
		"rl.host IN",
		"rl.latency_us >=",
		"rl.public_endpoint =",
		"rl.occurred_at >=",
		"rl.occurred_at <=",
		"u.name LIKE",
		"ak.name LIKE",
	}
	for _, f := range forbidden {
		if strings.Contains(sql, f) {
			t.Errorf("no-filter query unexpectedly contains predicate %q; SQL:\n%s", f, sql)
		}
	}
}

func TestBuildListQuery_ActorTypeEmitsSinglePredicateNoSentinelOR(t *testing.T) {
	f := emptyFilter()
	f.ActorType = strPtr("user")

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	if n := strings.Count(sql, "rl.identity_type = ?"); n != 1 {
		t.Fatalf("expected exactly one identity_type predicate, got %d; SQL:\n%s", n, sql)
	}
	if strings.Contains(sql, "OR rl.identity_type") || strings.Contains(sql, "identity_type = ?' OR") {
		t.Errorf("identity_type predicate leaked a sentinel-OR wrapper; SQL:\n%s", sql)
	}
	if !containsArg(args, "user") {
		t.Errorf("expected 'user' in args; got %#v", args)
	}
}

func TestBuildListQuery_ForwardCursorEmitsLessThanComparison(t *testing.T) {
	cur := &pagination.StringCursor{
		OccurredAt: time.Unix(1_700_000_000, 0).UTC(),
		ID:         "rlog_cursor",
		Direction:  pagination.DirectionForward,
	}
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, cur, 101)

	mustContain(t, sql, "rl.occurred_at < ?")
	mustContain(t, sql, "ORDER BY rl.occurred_at DESC, rl.id DESC LIMIT ?")
	if strings.Contains(sql, "rl.occurred_at > ?") {
		t.Errorf("forward cursor produced backward comparison; SQL:\n%s", sql)
	}
}

func TestBuildListQuery_BackwardCursorEmitsGreaterThanComparison(t *testing.T) {
	cur := &pagination.StringCursor{
		OccurredAt: time.Unix(1_700_000_000, 0).UTC(),
		ID:         "rlog_cursor",
		Direction:  pagination.DirectionBackward,
	}
	sql, _ := buildListQuery(queryModeBase, pagination.DirectionBackward, "acc_1", emptyFilter(), false, false, false, cur, 101)

	mustContain(t, sql, "rl.occurred_at > ?")
	mustContain(t, sql, "ORDER BY rl.occurred_at ASC, rl.id ASC LIMIT ?")
	if strings.Contains(sql, "rl.occurred_at < ?") {
		t.Errorf("backward cursor produced forward comparison; SQL:\n%s", sql)
	}
}

func TestBuildListQuery_ActorModeWrapsInDerivedTableWithUserAndApiKeyJoins(t *testing.T) {
	sql, _ := buildListQuery(queryModeActor, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	mustContain(t, sql, " FROM (")
	mustContain(t, sql, ") rl")
	mustContain(t, sql, "LEFT JOIN `user` u ON rl.actor_id = u.id AND rl.identity_type = 'user'")
	mustContain(t, sql, "LEFT JOIN api_key ak ON rl.actor_id = ak.type_id AND rl.identity_type = 'api_key'")

	forbidden := []string{
		"LEFT JOIN account_user au",
		"LEFT JOIN role r_user",
		"LEFT JOIN account a",
	}
	for _, f := range forbidden {
		if strings.Contains(sql, f) {
			t.Errorf("actor mode unexpectedly emitted %q; SQL:\n%s", f, sql)
		}
	}
}

func TestBuildListQuery_FullModeInlinesAllJoinsWithoutDerivedTable(t *testing.T) {
	sql, _ := buildListQuery(queryModeFull, pagination.DirectionForward, "acc_1", emptyFilter(), false, false, false, nil, 101)

	if strings.Contains(sql, " FROM (") {
		t.Errorf("full mode should not wrap in derived table; SQL:\n%s", sql)
	}

	required := []string{
		"LEFT JOIN `user` u",
		"LEFT JOIN api_key ak",
		"LEFT JOIN account_user au",
		"LEFT JOIN role r_user",
		"LEFT JOIN role r_key",
		"LEFT JOIN account a",
		"LEFT JOIN idempotency_key ik",
	}
	for _, r := range required {
		mustContain(t, sql, r)
	}
}

func TestBuildListQuery_ActorNameFilterSkippedOutsideFullMode(t *testing.T) {
	f := emptyFilter()
	f.ActorName = strPtr("alice")

	for _, mode := range []queryMode{queryModeBase, queryModeActor} {
		sql, _ := buildListQuery(mode, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)
		if strings.Contains(sql, "u.name LIKE") || strings.Contains(sql, "ak.name LIKE") {
			t.Errorf("mode %d leaked actor_name predicate into non-full mode; SQL:\n%s", mode, sql)
		}
	}
	sql, _ := buildListQuery(queryModeFull, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)
	mustContain(t, sql, "u.name LIKE ? OR ak.name LIKE ?")
}

func TestBuildListQuery_SliceFiltersExpandToMatchingPlaceholderCounts(t *testing.T) {
	f := emptyFilter()
	f.ActorIDs = []string{"u_1", "u_2", "u_3"}
	f.NormalizedRoutes = []string{"/a", "/b"}
	f.Hosts = []string{"api.example.com"}

	sql, args := buildListQuery(queryModeBase, pagination.DirectionForward, "acc_1", f, false, false, false, nil, 101)

	mustContain(t, sql, "rl.actor_id IN (?, ?, ?)")
	mustContain(t, sql, "rl.normalized_route IN (?, ?)")
	mustContain(t, sql, "rl.host IN (?)")

	// 3 JSON-include booleans (query/request/response) + target_account_id +
	// 3 actor ids + 2 routes + 1 host + limit = 11
	want := 3 + 1 + 3 + 2 + 1 + 1
	if len(args) != want {
		t.Errorf("unexpected arg count: got %d, want %d; args=%#v", len(args), want, args)
	}
}

// Helpers

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected SQL to contain %q, got:\n%s", needle, haystack)
	}
}

func containsArg(args []any, target any) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
